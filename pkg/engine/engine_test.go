package engine

import (
	"testing"
	"time"

	"github.com/chandanpasunoori/billshare/pkg/domain"
)

func TestCalculateBalancesAndSettleDebts(t *testing.T) {
	// Scenario:
	// Alice, Bob, Charlie are in a group.
	// 1. Alice pays $90.00 for dinner, split equally between Alice, Bob, Charlie ($30 each).
	// 2. Bob pays $30.00 for drinks, split equally between Bob and Charlie ($15 each).
	members := []string{"alice", "bob", "charlie"}
	expenses := []domain.Expense{
		{
			ID:          "1",
			Description: "Dinner",
			Amount:      9000,
			PaidBy:      "alice",
			Splits: map[string]int64{
				"alice":   3000,
				"bob":     3000,
				"charlie": 3000,
			},
			Date: time.Now(),
		},
		{
			ID:          "2",
			Description: "Drinks",
			Amount:      3000,
			PaidBy:      "bob",
			Splits: map[string]int64{
				"bob":     1500,
				"charlie": 1500,
			},
			Date: time.Now(),
		},
	}

	balances := CalculateBalances(members, expenses)

	// Alice: +90.00 - 30.00 = +60.00 (6000 cents)
	if balances["alice"] != 6000 {
		t.Errorf("expected alice balance to be 6000, got %d", balances["alice"])
	}
	// Bob: +30.00 - 30.00 (dinner) - 15.00 (drinks) = -15.00 (-1500 cents)
	if balances["bob"] != -1500 {
		t.Errorf("expected bob balance to be -1500, got %d", balances["bob"])
	}
	// Charlie: -30.00 (dinner) - 15.00 (drinks) = -45.00 (-4500 cents)
	if balances["charlie"] != -4500 {
		t.Errorf("expected charlie balance to be -4500, got %d", balances["charlie"])
	}

	transfers := SettleDebts(balances)

	// Expected settlements:
	// Charlie owes Alice $45.00
	// Bob owes Alice $15.00
	if len(transfers) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(transfers))
	}

	// Find and verify the transfers
	var charlieToAlice, bobToAlice int64
	for _, tr := range transfers {
		if tr.From == "charlie" && tr.To == "alice" {
			charlieToAlice = tr.Amount
		} else if tr.From == "bob" && tr.To == "alice" {
			bobToAlice = tr.Amount
		} else {
			t.Errorf("unexpected transfer: %+v", tr)
		}
	}

	if charlieToAlice != 4500 {
		t.Errorf("expected charlie to owe alice 4500, got %d", charlieToAlice)
	}
	if bobToAlice != 1500 {
		t.Errorf("expected bob to owe alice 1500, got %d", bobToAlice)
	}
}
