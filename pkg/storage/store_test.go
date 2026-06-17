package storage

import (
	"os"
	"path/filepath"
	"testing"

	"billshare/pkg/domain"
)

func TestJSONStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "billshare_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.json")

	store, err := NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// 1. Test Users
	u1, err := store.CreateUser("Alice")
	if err != nil {
		t.Fatalf("failed to create user Alice: %v", err)
	}
	if u1.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", u1.Name)
	}

	u2, err := store.CreateUser("Bob")
	if err != nil {
		t.Fatalf("failed to create user Bob: %v", err)
	}

	users, err := store.ListUsers()
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	// 2. Test Groups
	g, err := store.CreateGroup("Trip", []string{u1.ID, u2.ID})
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if g.Name != "Trip" {
		t.Errorf("expected group name Trip, got %s", g.Name)
	}
	if len(g.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(g.Members))
	}

	groups, err := store.ListGroups()
	if err != nil {
		t.Fatalf("failed to list groups: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}

	// 3. Test Expenses
	exp, err := store.AddExpense(g.ID, domain.Expense{
		Description: "Gas",
		Amount:      4000,
		PaidBy:      u1.ID,
		Splits: map[string]int64{
			u1.ID: 2000,
			u2.ID: 2000,
		},
	})
	if err != nil {
		t.Fatalf("failed to add expense: %v", err)
	}

	gLoaded, err := store.GetGroup(g.ID)
	if err != nil {
		t.Fatalf("failed to get group: %v", err)
	}
	if len(gLoaded.Expenses) != 1 {
		t.Errorf("expected 1 expense, got %d", len(gLoaded.Expenses))
	}
	if gLoaded.Expenses[0].Description != "Gas" {
		t.Errorf("expected expense description Gas, got %s", gLoaded.Expenses[0].Description)
	}

	// 4. Test delete expense
	err = store.DeleteExpense(g.ID, exp.ID)
	if err != nil {
		t.Fatalf("failed to delete expense: %v", err)
	}

	gLoaded2, err := store.GetGroup(g.ID)
	if err != nil {
		t.Fatalf("failed to get group: %v", err)
	}
	if len(gLoaded2.Expenses) != 0 {
		t.Errorf("expected 0 expenses after deletion, got %d", len(gLoaded2.Expenses))
	}

	// 5. Test AddUserToGroup
	u3, err := store.CreateUser("Charlie")
	if err != nil {
		t.Fatalf("failed to create user Charlie: %v", err)
	}

	err = store.AddUserToGroup(g.ID, u3.ID)
	if err != nil {
		t.Fatalf("failed to add user to group: %v", err)
	}

	gLoaded3, err := store.GetGroup(g.ID)
	if err != nil {
		t.Fatalf("failed to get group: %v", err)
	}

	foundCharlie := false
	for _, m := range gLoaded3.Members {
		if m == u3.ID {
			foundCharlie = true
			break
		}
	}
	if !foundCharlie {
		t.Errorf("expected Charlie (ID: %s) to be in group, but wasn't found", u3.ID)
	}

	// 6. Test UpdateExpenseSplits
	exp2, err := store.AddExpense(g.ID, domain.Expense{
		Description: "Taxi",
		Amount:      3000,
		PaidBy:      u1.ID,
		Splits: map[string]int64{
			u1.ID: 1500,
			u2.ID: 1500,
		},
	})
	if err != nil {
		t.Fatalf("failed to add expense: %v", err)
	}

	newSplits := map[string]int64{
		u1.ID: 1000,
		u2.ID: 1000,
		u3.ID: 1000,
	}
	err = store.UpdateExpenseSplits(g.ID, exp2.ID, newSplits)
	if err != nil {
		t.Fatalf("failed to update splits: %v", err)
	}

	gLoaded4, err := store.GetGroup(g.ID)
	if err != nil {
		t.Fatalf("failed to get group: %v", err)
	}

	var loadedExp2 domain.Expense
	for _, e := range gLoaded4.Expenses {
		if e.ID == exp2.ID {
			loadedExp2 = e
			break
		}
	}

	if len(loadedExp2.Splits) != 3 {
		t.Errorf("expected 3 splits, got %d", len(loadedExp2.Splits))
	}
	if loadedExp2.Splits[u3.ID] != 1000 {
		t.Errorf("expected u3 split to be 1000, got %d", loadedExp2.Splits[u3.ID])
	}
}
