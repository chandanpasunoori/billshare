package tui

import (
	"os"
	"path/filepath"
	"testing"

	"billshare/pkg/domain"
	"billshare/pkg/storage"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUISettleUpFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "billshare_tui_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.json")
	store, err := storage.NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Setup users and group
	alice, _ := store.CreateUser("Alice")
	bob, _ := store.CreateUser("Bob")
	g, _ := store.CreateGroup("Trip", []string{alice.ID, bob.ID})

	// Add an expense: Alice paid $30.00, split between Alice and Bob ($15.00 each)
	_, _ = store.AddExpense(g.ID, domain.Expense{
		Description: "Dinner",
		Amount:      3000,
		PaidBy:      alice.ID,
		Splits: map[string]int64{
			alice.ID: 1500,
			bob.ID:   1500,
		},
	})

	// Initialize TUI model
	m := NewModel(store)

	// Simulate data load msg
	groups, _ := store.ListGroups()
	users, _ := store.ListUsers()
	modelInterface, _ := m.Update(loadedDataMsg{groups: groups, users: users})
	m = modelInterface.(model)

	// Select group and Enter
	m.selectedGroupIdx = 0
	modelInterface, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})
	m = modelInterface.(model)

	if m.state != stateViewGroup {
		t.Fatalf("expected state to be stateViewGroup, got %v", m.state)
	}

	// Press 's' to Settle Up
	modelInterface, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = modelInterface.(model)

	if m.state != stateSettleUpSelectDebt {
		t.Fatalf("expected state to be stateSettleUpSelectDebt, got %v", m.state)
	}

	// Ensure there is exactly 1 debt in the list
	if len(m.activeGroupTransfers) != 1 {
		t.Fatalf("expected 1 debt transfer, got %d", len(m.activeGroupTransfers))
	}

	// Select the simplified debt (Bob owes Alice $15.00) and press Enter
	m.settleDebtCursor = 0
	modelInterface, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})
	m = modelInterface.(model)

	if m.state != stateSettleUpAmount {
		t.Fatalf("expected state to be stateSettleUpAmount, got %v", m.state)
	}

	if m.settleDebtorID != bob.ID {
		t.Errorf("expected debtor to be Bob (%s), got %s", bob.ID, m.settleDebtorID)
	}
	if m.settleCreditorID != alice.ID {
		t.Errorf("expected creditor to be Alice (%s), got %s", alice.ID, m.settleCreditorID)
	}
	if m.textInput.Value() != "15.00" {
		t.Errorf("expected pre-filled value '15.00', got '%s'", m.textInput.Value())
	}

	// Press Enter to confirm full settlement
	modelInterface, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})
	m = modelInterface.(model)

	if m.state != stateViewGroup {
		t.Fatalf("expected state to return to stateViewGroup, got %v", m.state)
	}

	// Reload group and verify Bob owes Alice $0 now (balances are settled)
	gLoaded, _ := store.GetGroup(g.ID)
	if len(gLoaded.Expenses) != 2 {
		t.Errorf("expected 2 expenses, got %d", len(gLoaded.Expenses))
	}

	settlementExp := gLoaded.Expenses[1]
	if settlementExp.PaidBy != bob.ID {
		t.Errorf("expected settlement paid by Bob, got %s", settlementExp.PaidBy)
	}
	if settlementExp.Amount != 1500 {
		t.Errorf("expected settlement amount 1500, got %d", settlementExp.Amount)
	}
	if settlementExp.Splits[alice.ID] != 1500 {
		t.Errorf("expected settlement split to Alice of 1500, got %d", settlementExp.Splits[alice.ID])
	}
}
