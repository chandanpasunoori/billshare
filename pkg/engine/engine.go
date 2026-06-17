package engine

import (
	"sort"

	"billshare/pkg/domain"
)

// UserBalance holds the net balance of a user (positive means creditor, negative means debtor).
type UserBalance struct {
	UserID  string
	Balance int64
}

// CalculateBalances computes the net balance of each user in a group or from a list of expenses.
// If members is provided, it ensures every member has a balance entry (even if 0).
func CalculateBalances(members []string, expenses []domain.Expense) map[string]int64 {
	balances := make(map[string]int64)

	// Initialize all members with 0 balance
	for _, memberID := range members {
		balances[memberID] = 0
	}

	for _, exp := range expenses {
		// The person who paid gets credited the full amount
		balances[exp.PaidBy] += exp.Amount

		// Everyone who splits gets debited their share
		for userID, oweAmount := range exp.Splits {
			balances[userID] -= oweAmount
		}
	}

	return balances
}

// SettleDebts takes the net balances of users and simplifies them into a minimal set of transfers.
func SettleDebts(balances map[string]int64) []domain.Transfer {
	var debtors []UserBalance
	var creditors []UserBalance

	// Split into debtors and creditors, ignoring those with zero balance
	for userID, bal := range balances {
		if bal < 0 {
			debtors = append(debtors, UserBalance{UserID: userID, Balance: bal})
		} else if bal > 0 {
			creditors = append(creditors, UserBalance{UserID: userID, Balance: bal})
		}
	}

	var transfers []domain.Transfer

	// Greedily match debtors and creditors
	for len(debtors) > 0 && len(creditors) > 0 {
		// Sort debtors so that the largest debtor (most negative balance) is at the end
		sort.Slice(debtors, func(i, j int) bool {
			return debtors[i].Balance > debtors[j].Balance // e.g. -10 > -50, so -50 is at the end
		})

		// Sort creditors so that the largest creditor (most positive balance) is at the end
		sort.Slice(creditors, func(i, j int) bool {
			return creditors[i].Balance < creditors[j].Balance // e.g. 10 < 50, so 50 is at the end
		})

		debtorIdx := len(debtors) - 1
		creditorIdx := len(creditors) - 1

		d := &debtors[debtorIdx]
		c := &creditors[creditorIdx]

		debtAmount := -d.Balance
		creditAmount := c.Balance

		var settleAmount int64
		if debtAmount < creditAmount {
			settleAmount = debtAmount
		} else {
			settleAmount = creditAmount
		}

		// Create the transfer
		transfers = append(transfers, domain.Transfer{
			From:   d.UserID,
			To:     c.UserID,
			Amount: settleAmount,
		})

		// Update balances
		d.Balance += settleAmount
		c.Balance -= settleAmount

		// Remove settled users
		if d.Balance == 0 {
			debtors = debtors[:debtorIdx]
		}
		if c.Balance == 0 {
			creditors = creditors[:creditorIdx]
		}
	}

	return transfers
}
