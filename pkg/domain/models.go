package domain

import (
	"time"
)

// User represents a participant in expenses.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Expense represents an expense paid by one user and split among others.
type Expense struct {
	ID          string           `json:"id"`
	Description string           `json:"description"`
	Amount      int64            `json:"amount"` // represented in cents (e.g. $10.00 is 1000)
	PaidBy      string           `json:"paid_by"` // User ID of the person who paid
	Splits      map[string]int64 `json:"splits"`  // User ID -> amount in cents they owe
	Date        time.Time        `json:"date"`
}

// Group represents a group of users sharing expenses.
type Group struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Members  []string  `json:"members"`  // list of User IDs
	Expenses []Expense `json:"expenses"` // list of expenses in this group
}

// Transfer represents a simplified debt settlement step (who should pay whom how much).
type Transfer struct {
	From   string `json:"from"`   // User ID of debtor
	To     string `json:"to"`     // User ID of creditor
	Amount int64  `json:"amount"` // amount in cents
}
