package tui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/chandanpasunoori/billshare/pkg/domain"
	"github.com/chandanpasunoori/billshare/pkg/engine"
	"github.com/chandanpasunoori/billshare/pkg/report"
	"github.com/chandanpasunoori/billshare/pkg/storage"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateHome state = iota
	stateAddUser
	stateCreateGroupName
	stateCreateGroupMembers
	stateViewGroup
	stateAddExpenseDesc
	stateAddExpenseAmount
	stateAddExpensePayer
	stateAddExpenseParticipants
	stateSettleUpDebtor
	stateSettleUpCreditor
	stateSettleUpAmount
	stateAddUserToGroup
	stateEditExpenseParticipants
	stateSettleUpSelectDebt
)

type model struct {
	store storage.Store
	state state

	// Navigation data
	groups           []domain.Group
	users            []domain.User
	selectedGroupIdx int

	// Active Group view state
	activeGroup          domain.Group
	activeGroupBalances  map[string]int64
	activeGroupTransfers []domain.Transfer
	selectedExpenseIdx   int

	// Form inputs & helper inputs
	textInput textinput.Model
	err       error
	infoMsg   string

	// Create Group Wizard state
	newGroupName       string
	groupMemberCursor  int
	groupMemberChecked map[string]bool // userID -> checked

	// Add Expense Wizard state
	expDesc        string
	expAmount      int64
	expPayerID     string
	expPayerCursor int
	expPartCursor  int
	expPartChecked map[string]bool // userID -> checked

	// Settle Up Wizard state
	settleDebtorID       string
	settleDebtorCursor   int
	settleCreditorID     string
	settleCreditorCursor int
	settleAmount         int64
	addUserToGroupCursor int
	settleDebtCursor     int
}

func NewModel(store storage.Store) model {
	ti := textinput.New()
	ti.Focus()
	ti.Width = 30

	return model{
		store:              store,
		state:              stateHome,
		textInput:          ti,
		groupMemberChecked: make(map[string]bool),
		expPartChecked:     make(map[string]bool),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.loadDataCmd(),
	)
}

// Msg types
type loadedDataMsg struct {
	groups []domain.Group
	users  []domain.User
}

type errorMsg error

func (m model) loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		groups, err := m.store.ListGroups()
		if err != nil {
			return errorMsg(err)
		}
		users, err := m.store.ListUsers()
		if err != nil {
			return errorMsg(err)
		}
		return loadedDataMsg{groups: groups, users: users}
	}
}

func (m model) reloadActiveGroup() model {
	if m.activeGroup.ID == "" {
		return m
	}
	g, err := m.store.GetGroup(m.activeGroup.ID)
	if err != nil {
		m.err = err
		return m
	}
	m.activeGroup = g
	m.activeGroupBalances = engine.CalculateBalances(g.Members, g.Expenses)
	m.activeGroupTransfers = engine.SettleDebts(m.activeGroupBalances)
	return m
}

func (m model) getUserName(id string) string {
	for _, u := range m.users {
		if u.ID == id {
			return u.Name
		}
	}
	return id
}

func parseAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("amount cannot be empty")
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid decimal format")
	}

	var dollars, cents int64
	var err error

	dollars, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid dollar amount: %s", parts[0])
	}
	if dollars < 0 {
		return 0, fmt.Errorf("amount cannot be negative")
	}

	if len(parts) == 2 {
		centsStr := parts[1]
		if len(centsStr) > 2 {
			centsStr = centsStr[:2] // truncate to 2 digits
		}
		if len(centsStr) == 1 {
			centsStr += "0"
		}
		if len(centsStr) == 0 {
			centsStr = "0"
		}
		cents, err = strconv.ParseInt(centsStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid cents amount: %s", parts[1])
		}
	}

	return dollars*100 + cents, nil
}

func formatAmount(cents int64) string {
	dollars := cents / 100
	remCents := cents % 100
	if remCents < 0 {
		remCents = -remCents
	}
	sign := ""
	if cents < 0 {
		sign = "-"
		if dollars < 0 {
			dollars = -dollars
		}
	}
	return fmt.Sprintf("%s$%d.%02d", sign, dollars, remCents)
}

func splitEqually(amount int64, payerID string, participants []string) map[string]int64 {
	splits := make(map[string]int64)
	n := int64(len(participants))
	if n == 0 {
		return splits
	}
	base := amount / n
	remainder := amount % n

	for _, p := range participants {
		splits[p] = base
	}

	if remainder > 0 {
		hasPayer := false
		for _, p := range participants {
			if p == payerID {
				splits[p] += remainder
				hasPayer = true
				break
			}
		}
		if !hasPayer {
			splits[participants[0]] += remainder
		}
	}
	return splits
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case loadedDataMsg:
		m.groups = msg.groups
		m.users = msg.users
		m.err = nil
		return m, nil

	case errorMsg:
		m.err = msg
		return m, nil
	}

	// State-specific Updates
	switch m.state {
	case stateHome:
		m, cmd = m.updateHome(msg)
	case stateAddUser:
		m, cmd = m.updateAddUser(msg)
	case stateCreateGroupName:
		m, cmd = m.updateCreateGroupName(msg)
	case stateCreateGroupMembers:
		m, cmd = m.updateCreateGroupMembers(msg)
	case stateViewGroup:
		m, cmd = m.updateViewGroup(msg)
	case stateAddExpenseDesc:
		m, cmd = m.updateAddExpenseDesc(msg)
	case stateAddExpenseAmount:
		m, cmd = m.updateAddExpenseAmount(msg)
	case stateAddExpensePayer:
		m, cmd = m.updateAddExpensePayer(msg)
	case stateAddExpenseParticipants:
		m, cmd = m.updateAddExpenseParticipants(msg)
	case stateSettleUpDebtor:
		m, cmd = m.updateSettleUpDebtor(msg)
	case stateSettleUpCreditor:
		m, cmd = m.updateSettleUpCreditor(msg)
	case stateSettleUpAmount:
		m, cmd = m.updateSettleUpAmount(msg)
	case stateAddUserToGroup:
		m, cmd = m.updateAddUserToGroup(msg)
	case stateEditExpenseParticipants:
		m, cmd = m.updateEditExpenseParticipants(msg)
	case stateSettleUpSelectDebt:
		m, cmd = m.updateSettleUpSelectDebt(msg)
	}

	return m, cmd
}

func (m model) updateHome(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedGroupIdx > 0 {
				m.selectedGroupIdx--
			}
		case "down", "j":
			if m.selectedGroupIdx < len(m.groups)-1 {
				m.selectedGroupIdx++
			}
		case "u": // Add User
			m.state = stateAddUser
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter user name"
			m.textInput.Focus()
			m.err = nil
			m.infoMsg = ""
		case "c": // Create Group
			if len(m.users) == 0 {
				m.err = fmt.Errorf("please create at least one user first ('u')")
				return m, nil
			}
			m.state = stateCreateGroupName
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter group name"
			m.textInput.Focus()
			m.err = nil
			m.infoMsg = ""
		case "enter":
			if len(m.groups) > 0 {
				m.activeGroup = m.groups[m.selectedGroupIdx]
				m = m.reloadActiveGroup()
				m.state = stateViewGroup
				m.selectedExpenseIdx = 0
				m.err = nil
				m.infoMsg = ""
			}
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) updateAddUser(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.textInput.Value())
			if name == "" {
				m.err = fmt.Errorf("name cannot be empty")
				return m, nil
			}
			_, err := m.store.CreateUser(name)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.state = stateHome
			m.infoMsg = fmt.Sprintf("User '%s' created successfully!", name)
			return m, m.loadDataCmd()
		case "esc":
			m.state = stateHome
			m.err = nil
			m.infoMsg = ""
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) updateCreateGroupName(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.textInput.Value())
			if name == "" {
				m.err = fmt.Errorf("group name cannot be empty")
				return m, nil
			}
			m.newGroupName = name
			m.state = stateCreateGroupMembers
			m.groupMemberCursor = 0
			m.groupMemberChecked = make(map[string]bool)
			// check all by default
			for _, u := range m.users {
				m.groupMemberChecked[u.ID] = false
			}
			m.err = nil
			return m, nil
		case "esc":
			m.state = stateHome
			m.err = nil
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) updateCreateGroupMembers(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.groupMemberCursor > 0 {
				m.groupMemberCursor--
			}
		case "down", "j":
			if m.groupMemberCursor < len(m.users)-1 {
				m.groupMemberCursor++
			}
		case " ": // Toggle selection
			uID := m.users[m.groupMemberCursor].ID
			m.groupMemberChecked[uID] = !m.groupMemberChecked[uID]
		case "enter":
			// Collect checked user IDs
			var memberIDs []string
			for uID, checked := range m.groupMemberChecked {
				if checked {
					memberIDs = append(memberIDs, uID)
				}
			}
			if len(memberIDs) == 0 {
				m.err = fmt.Errorf("select at least one group member")
				return m, nil
			}
			_, err := m.store.CreateGroup(m.newGroupName, memberIDs)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.state = stateHome
			m.infoMsg = fmt.Sprintf("Group '%s' created successfully!", m.newGroupName)
			return m, m.loadDataCmd()
		case "esc":
			m.state = stateHome
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateViewGroup(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "b":
			m.state = stateHome
			m.err = nil
			m.infoMsg = ""
			return m, m.loadDataCmd()
		case "up", "k":
			if m.selectedExpenseIdx > 0 {
				m.selectedExpenseIdx--
			}
		case "down", "j":
			if m.selectedExpenseIdx < len(m.activeGroup.Expenses)-1 {
				m.selectedExpenseIdx++
			}
		case "e": // Add Expense
			m.state = stateAddExpenseDesc
			m.textInput.SetValue("")
			m.textInput.Placeholder = "What was this for? (e.g. Pizza)"
			m.textInput.Focus()
			m.err = nil
			m.infoMsg = ""
		case "d": // Delete Expense
			if len(m.activeGroup.Expenses) > 0 {
				targetExp := m.activeGroup.Expenses[m.selectedExpenseIdx]
				err := m.store.DeleteExpense(m.activeGroup.ID, targetExp.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m = m.reloadActiveGroup()
				m.selectedExpenseIdx = 0
				m.infoMsg = fmt.Sprintf("Deleted expense '%s'", targetExp.Description)
			}
		case "s": // Settle Up
			m.state = stateSettleUpSelectDebt
			m.settleDebtCursor = 0
			m.err = nil
			m.infoMsg = ""
		case "a": // Add User to Group
			m.state = stateAddUserToGroup
			m.addUserToGroupCursor = 0
			m.err = nil
			m.infoMsg = ""
		case "o": // Edit Who Owes (Splits)
			if len(m.activeGroup.Expenses) > 0 {
				targetExp := m.activeGroup.Expenses[m.selectedExpenseIdx]
				m.state = stateEditExpenseParticipants
				m.expPartCursor = 0
				m.expPartChecked = make(map[string]bool)
				for _, mID := range m.activeGroup.Members {
					amt, exists := targetExp.Splits[mID]
					m.expPartChecked[mID] = exists && amt > 0
				}
				m.err = nil
				m.infoMsg = ""
			}
		case "r": // Recalculate
			m = m.reloadActiveGroup()
			m.infoMsg = "Balances and simplified debts recalculated successfully!"
			m.err = nil
		case "p": // Export PNG Report
			filename := fmt.Sprintf("%s_report.png", strings.ReplaceAll(strings.ToLower(m.activeGroup.Name), " ", "_"))
			err := report.GenerateGroupReportImage(m.activeGroup, m.users, filename)
			if err != nil {
				m.err = fmt.Errorf("failed to save report: %w", err)
			} else {
				m.infoMsg = fmt.Sprintf("Report image exported successfully as %s", filename)
				m.err = nil
			}
		case "w": // Share to WhatsApp
			reportText := GenerateWhatsAppText(m.activeGroup, m.users)
			encodedText := url.QueryEscape(reportText)
			shareURL := fmt.Sprintf("https://wa.me/?text=%s", encodedText)
			err := clipboard.WriteAll(shareURL)
			if err != nil {
				m.err = fmt.Errorf("failed to copy to clipboard: %w", err)
			} else {
				m.infoMsg = "WhatsApp share link copied to clipboard!"
				m.err = nil
			}
		}
	}
	return m, nil
}

func (m model) updateAddExpenseDesc(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			desc := strings.TrimSpace(m.textInput.Value())
			if desc == "" {
				m.err = fmt.Errorf("description cannot be empty")
				return m, nil
			}
			m.expDesc = desc
			m.state = stateAddExpenseAmount
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter total amount (e.g. 45.50)"
			m.textInput.Focus()
			m.err = nil
			return m, nil
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) updateAddExpenseAmount(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			amount, err := parseAmount(m.textInput.Value())
			if err != nil {
				m.err = err
				return m, nil
			}
			if amount == 0 {
				m.err = fmt.Errorf("amount must be greater than zero")
				return m, nil
			}
			m.expAmount = amount
			m.state = stateAddExpensePayer
			m.expPayerCursor = 0
			m.err = nil
			return m, nil
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) updateAddExpensePayer(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.expPayerCursor > 0 {
				m.expPayerCursor--
			}
		case "down", "j":
			if m.expPayerCursor < len(m.activeGroup.Members)-1 {
				m.expPayerCursor++
			}
		case "enter":
			m.expPayerID = m.activeGroup.Members[m.expPayerCursor]
			m.state = stateAddExpenseParticipants
			m.expPartCursor = 0
			m.expPartChecked = make(map[string]bool)
			for _, mID := range m.activeGroup.Members {
				m.expPartChecked[mID] = true // checked by default
			}
			m.err = nil
			return m, nil
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateAddExpenseParticipants(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.expPartCursor > 0 {
				m.expPartCursor--
			}
		case "down", "j":
			if m.expPartCursor < len(m.activeGroup.Members)-1 {
				m.expPartCursor++
			}
		case " ": // Toggle participant
			mID := m.activeGroup.Members[m.expPartCursor]
			m.expPartChecked[mID] = !m.expPartChecked[mID]
		case "enter":
			var participants []string
			for mID, checked := range m.expPartChecked {
				if checked {
					participants = append(participants, mID)
				}
			}
			if len(participants) == 0 {
				m.err = fmt.Errorf("select at least one participant")
				return m, nil
			}

			splits := splitEqually(m.expAmount, m.expPayerID, participants)

			_, err := m.store.AddExpense(m.activeGroup.ID, domain.Expense{
				Description: m.expDesc,
				Amount:      m.expAmount,
				PaidBy:      m.expPayerID,
				Splits:      splits,
			})
			if err != nil {
				m.err = err
				return m, nil
			}

			m = m.reloadActiveGroup()
			m.state = stateViewGroup
			m.infoMsg = fmt.Sprintf("Added expense '%s' of %s", m.expDesc, formatAmount(m.expAmount))
			return m, nil
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateSettleUpDebtor(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.settleDebtorCursor > 0 {
				m.settleDebtorCursor--
			}
		case "down", "j":
			if m.settleDebtorCursor < len(m.activeGroup.Members)-1 {
				m.settleDebtorCursor++
			}
		case "enter":
			m.settleDebtorID = m.activeGroup.Members[m.settleDebtorCursor]
			m.state = stateSettleUpCreditor
			m.settleCreditorCursor = 0
			m.err = nil
			return m, nil
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateSettleUpCreditor(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.settleCreditorCursor > 0 {
				m.settleCreditorCursor--
			}
		case "down", "j":
			if m.settleCreditorCursor < len(m.activeGroup.Members)-1 {
				m.settleCreditorCursor++
			}
		case "enter":
			creditorID := m.activeGroup.Members[m.settleCreditorCursor]
			if creditorID == m.settleDebtorID {
				m.err = fmt.Errorf("debtor and creditor must be different users")
				return m, nil
			}
			m.settleCreditorID = creditorID
			m.state = stateSettleUpAmount
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter amount to settle (e.g. 15.00)"
			m.textInput.Focus()
			m.err = nil
			return m, nil
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateSettleUpAmount(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			amount, err := parseAmount(m.textInput.Value())
			if err != nil {
				m.err = err
				return m, nil
			}
			if amount == 0 {
				m.err = fmt.Errorf("settlement amount must be greater than zero")
				return m, nil
			}
			m.settleAmount = amount

			debtorName := m.getUserName(m.settleDebtorID)
			creditorName := m.getUserName(m.settleCreditorID)
			desc := fmt.Sprintf("Settle: %s -> %s", debtorName, creditorName)

			splits := map[string]int64{
				m.settleCreditorID: m.settleAmount,
			}

			_, err = m.store.AddExpense(m.activeGroup.ID, domain.Expense{
				Description: desc,
				Amount:      m.settleAmount,
				PaidBy:      m.settleDebtorID,
				Splits:      splits,
			})
			if err != nil {
				m.err = err
				return m, nil
			}

			m = m.reloadActiveGroup()
			m.state = stateViewGroup
			m.infoMsg = fmt.Sprintf("Recorded settlement: %s paid %s to %s", debtorName, formatAmount(m.settleAmount), creditorName)
			return m, nil
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) getEligibleUsersForGroup() []domain.User {
	var eligible []domain.User
	memberMap := make(map[string]bool)
	for _, mID := range m.activeGroup.Members {
		memberMap[mID] = true
	}
	for _, u := range m.users {
		if !memberMap[u.ID] {
			eligible = append(eligible, u)
		}
	}
	return eligible
}

func (m model) updateAddUserToGroup(msg tea.Msg) (model, tea.Cmd) {
	eligible := m.getEligibleUsersForGroup()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.addUserToGroupCursor > 0 {
				m.addUserToGroupCursor--
			}
		case "down", "j":
			if m.addUserToGroupCursor < len(eligible)-1 {
				m.addUserToGroupCursor++
			}
		case "enter":
			if len(eligible) > 0 {
				targetUser := eligible[m.addUserToGroupCursor]
				err := m.store.AddUserToGroup(m.activeGroup.ID, targetUser.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m = m.reloadActiveGroup()
				m.state = stateViewGroup
				m.infoMsg = fmt.Sprintf("Added %s to the group!", targetUser.Name)
			}
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateSettleUpSelectDebt(msg tea.Msg) (model, tea.Cmd) {
	debts := m.activeGroupTransfers

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.settleDebtCursor > 0 {
				m.settleDebtCursor--
			}
		case "down", "j":
			if m.settleDebtCursor < len(debts) {
				m.settleDebtCursor++
			}
		case "enter":
			if m.settleDebtCursor == len(debts) {
				m.state = stateSettleUpDebtor
				m.settleDebtorCursor = 0
				m.err = nil
				return m, nil
			}

			debt := debts[m.settleDebtCursor]
			m.settleDebtorID = debt.From
			m.settleCreditorID = debt.To

			amountStr := fmt.Sprintf("%d.%02d", debt.Amount/100, debt.Amount%100)
			m.textInput.SetValue(amountStr)
			m.textInput.Placeholder = "Enter amount to settle"
			m.textInput.Focus()

			m.state = stateSettleUpAmount
			m.err = nil
			return m, nil

		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateEditExpenseParticipants(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.expPartCursor > 0 {
				m.expPartCursor--
			}
		case "down", "j":
			if m.expPartCursor < len(m.activeGroup.Members)-1 {
				m.expPartCursor++
			}
		case " ": // Toggle participant
			mID := m.activeGroup.Members[m.expPartCursor]
			m.expPartChecked[mID] = !m.expPartChecked[mID]
		case "enter":
			var participants []string
			for mID, checked := range m.expPartChecked {
				if checked {
					participants = append(participants, mID)
				}
			}
			if len(participants) == 0 {
				m.err = fmt.Errorf("select at least one participant")
				return m, nil
			}

			targetExp := m.activeGroup.Expenses[m.selectedExpenseIdx]
			splits := splitEqually(targetExp.Amount, targetExp.PaidBy, participants)

			err := m.store.UpdateExpenseSplits(m.activeGroup.ID, targetExp.ID, splits)
			if err != nil {
				m.err = err
				return m, nil
			}

			m = m.reloadActiveGroup()
			m.state = stateViewGroup
			m.infoMsg = fmt.Sprintf("Updated who owes for expense '%s'", targetExp.Description)
			return m, nil
		case "esc":
			m.state = stateViewGroup
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() string {
	var s strings.Builder

	// Header Banner
	s.WriteString(titleStyle.Render(" BILLSHARE - Splitwise CLI ") + "\n\n")

	// Status messages / Errors
	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n")
	} else if m.infoMsg != "" {
		s.WriteString(successStyle.Render(m.infoMsg) + "\n\n")
	}

	switch m.state {
	case stateHome:
		s.WriteString(m.viewHome())
	case stateAddUser:
		s.WriteString(m.viewAddUser())
	case stateCreateGroupName:
		s.WriteString(m.viewCreateGroupName())
	case stateCreateGroupMembers:
		s.WriteString(m.viewCreateGroupMembers())
	case stateViewGroup:
		s.WriteString(m.viewViewGroup())
	case stateAddExpenseDesc:
		s.WriteString(m.viewAddExpenseDesc())
	case stateAddExpenseAmount:
		s.WriteString(m.viewAddExpenseAmount())
	case stateAddExpensePayer:
		s.WriteString(m.viewAddExpensePayer())
	case stateAddExpenseParticipants:
		s.WriteString(m.viewAddExpenseParticipants())
	case stateSettleUpDebtor:
		s.WriteString(m.viewSettleUpDebtor())
	case stateSettleUpCreditor:
		s.WriteString(m.viewSettleUpCreditor())
	case stateSettleUpAmount:
		s.WriteString(m.viewSettleUpAmount())
	case stateAddUserToGroup:
		s.WriteString(m.viewAddUserToGroup())
	case stateEditExpenseParticipants:
		s.WriteString(m.viewEditExpenseParticipants())
	case stateSettleUpSelectDebt:
		s.WriteString(m.viewSettleUpSelectDebt())
	}

	return s.String()
}

func (m model) viewHome() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Groups") + "\n")

	if len(m.groups) == 0 {
		s.WriteString(normalItemStyle.Render("No groups found. Press 'c' to create a group.") + "\n\n")
	} else {
		for idx, g := range m.groups {
			var groupStr string
			if idx == m.selectedGroupIdx {
				groupStr = selectedItemStyle.Render(fmt.Sprintf("➔ %s (%d members)", g.Name, len(g.Members)))
			} else {
				groupStr = normalItemStyle.Render(fmt.Sprintf("  %s (%d members)", g.Name, len(g.Members)))
			}
			s.WriteString(groupStr + "\n")
		}
		s.WriteString("\n")
	}

	s.WriteString(headerStyle.Render("Registered Users") + "\n")
	if len(m.users) == 0 {
		s.WriteString(normalItemStyle.Render("No users registered yet. Press 'u' to register a user.") + "\n\n")
	} else {
		var userNames []string
		for _, u := range m.users {
			userNames = append(userNames, u.Name)
		}
		s.WriteString(normalItemStyle.Render("  "+strings.Join(userNames, ", ")) + "\n\n")
	}

	s.WriteString(helpStyle.Render("Commands: [u] Add User • [c] Create Group • [enter] View Group • [q] Quit") + "\n")
	return s.String()
}

func (m model) viewAddUser() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		headerStyle.Render("Add New User"),
		m.textInput.View(),
		helpStyle.Render("[enter] Submit • [esc] Cancel"),
	)
}

func (m model) viewCreateGroupName() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		headerStyle.Render("Create New Group (Step 1 of 2)"),
		m.textInput.View(),
		helpStyle.Render("[enter] Next • [esc] Cancel"),
	)
}

func (m model) viewCreateGroupMembers() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render(fmt.Sprintf("Select Members for Group '%s' (Step 2 of 2)", m.newGroupName)) + "\n")

	for idx, u := range m.users {
		checked := "[ ]"
		if m.groupMemberChecked[u.ID] {
			checked = "[x]"
		}

		var line string
		if idx == m.groupMemberCursor {
			line = selectedItemStyle.Render(fmt.Sprintf("➔ %s %s", checked, u.Name))
		} else {
			line = normalItemStyle.Render(fmt.Sprintf("  %s %s", checked, u.Name))
		}
		s.WriteString(line + "\n")
	}
	s.WriteString("\n" + helpStyle.Render("[space] Toggle selection • [enter] Create Group • [esc] Cancel") + "\n")
	return s.String()
}

func (m model) viewViewGroup() string {
	var s strings.Builder

	// Top Group Name card
	var members []string
	for _, mID := range m.activeGroup.Members {
		members = append(members, m.getUserName(mID))
	}
	cardContent := fmt.Sprintf(
		"Group: %s\nMembers: %s",
		lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(m.activeGroup.Name),
		strings.Join(members, ", "),
	)
	s.WriteString(cardStyle.Render(cardContent) + "\n\n")

	// Split view: Left: Expenses, Right: Balances & Simplified Debts
	var leftSide, rightSide strings.Builder

	// Left: Expenses
	leftSide.WriteString(headerStyle.Render("Expenses") + "\n")
	if len(m.activeGroup.Expenses) == 0 {
		leftSide.WriteString(normalItemStyle.Render("No expenses yet.") + "\n")
	} else {
		for idx, exp := range m.activeGroup.Expenses {
			payer := m.getUserName(exp.PaidBy)
			var line string
			expDetails := fmt.Sprintf("%s - %s (Paid by %s)", exp.Description, formatAmount(exp.Amount), payer)
			if idx == m.selectedExpenseIdx {
				line = selectedItemStyle.Render(fmt.Sprintf("➔ %s", expDetails))
			} else {
				line = normalItemStyle.Render(fmt.Sprintf("  %s", expDetails))
			}
			leftSide.WriteString(line + "\n")
		}
	}

	// Right: Balances
	rightSide.WriteString(headerStyle.Render("Net Balances") + "\n")
	for _, mID := range m.activeGroup.Members {
		bal := m.activeGroupBalances[mID]
		name := m.getUserName(mID)
		var balStr string
		if bal > 0 {
			balStr = positiveAmountStyle.Render(fmt.Sprintf("+%s", formatAmount(bal)))
		} else if bal < 0 {
			balStr = negativeAmountStyle.Render(formatAmount(bal))
		} else {
			balStr = normalItemStyle.Render(formatAmount(0))
		}
		fmt.Fprintf(&rightSide, "  %s: %s\n", name, balStr)
	}
	rightSide.WriteString("\n")

	// Right: Transfers
	rightSide.WriteString(headerStyle.Render("Simplified Debts") + "\n")
	if len(m.activeGroupTransfers) == 0 {
		rightSide.WriteString(normalItemStyle.Render("  All settled!") + "\n")
	} else {
		for _, tr := range m.activeGroupTransfers {
			fromName := m.getUserName(tr.From)
			toName := m.getUserName(tr.To)
			transferStr := fmt.Sprintf("  %s owes %s %s", fromName, toName, formatAmount(tr.Amount))
			rightSide.WriteString(normalItemStyle.Render(transferStr) + "\n")
		}
	}

	// Join side-by-side using Lip Gloss
	leftCol := lipgloss.NewStyle().Width(50).Render(leftSide.String())
	rightCol := lipgloss.NewStyle().Width(40).Render(rightSide.String())

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol) + "\n\n")

	s.WriteString(helpStyle.Render("Commands: [e] Add Expense • [d] Delete Selected Expense • [s] Settle Up • [a] Add User • [o] Edit Who Owes • [r] Recalculate • [p] Export Image • [w] WhatsApp Share • [b/esc] Back") + "\n")
	return s.String()
}

func (m model) viewAddExpenseDesc() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		headerStyle.Render("Add Expense - Step 1 of 4: Description"),
		m.textInput.View(),
		helpStyle.Render("[enter] Next • [esc] Cancel"),
	)
}

func (m model) viewAddExpenseAmount() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		headerStyle.Render("Add Expense - Step 2 of 4: Amount"),
		m.textInput.View(),
		helpStyle.Render("[enter] Next • [esc] Cancel"),
	)
}

func (m model) viewAddExpensePayer() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Add Expense - Step 3 of 4: Who paid?") + "\n")

	for idx, mID := range m.activeGroup.Members {
		name := m.getUserName(mID)
		var line string
		if idx == m.expPayerCursor {
			line = selectedItemStyle.Render(fmt.Sprintf("➔ %s", name))
		} else {
			line = normalItemStyle.Render(fmt.Sprintf("  %s", name))
		}
		s.WriteString(line + "\n")
	}
	s.WriteString("\n" + helpStyle.Render("[enter] Next • [esc] Cancel") + "\n")
	return s.String()
}

func (m model) viewAddExpenseParticipants() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Add Expense - Step 4 of 4: Who participates?") + "\n")

	for idx, mID := range m.activeGroup.Members {
		name := m.getUserName(mID)
		checked := "[ ]"
		if m.expPartChecked[mID] {
			checked = "[x]"
		}
		var line string
		if idx == m.expPartCursor {
			line = selectedItemStyle.Render(fmt.Sprintf("➔ %s %s", checked, name))
		} else {
			line = normalItemStyle.Render(fmt.Sprintf("  %s %s", checked, name))
		}
		s.WriteString(line + "\n")
	}
	s.WriteString("\n" + helpStyle.Render("[space] Toggle • [enter] Save Expense • [esc] Cancel") + "\n")
	return s.String()
}

func (m model) viewSettleUpDebtor() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Settle Up - Step 1 of 3: Who paid?") + "\n")

	for idx, mID := range m.activeGroup.Members {
		name := m.getUserName(mID)
		var line string
		if idx == m.settleDebtorCursor {
			line = selectedItemStyle.Render(fmt.Sprintf("➔ %s", name))
		} else {
			line = normalItemStyle.Render(fmt.Sprintf("  %s", name))
		}
		s.WriteString(line + "\n")
	}
	s.WriteString("\n" + helpStyle.Render("[enter] Next • [esc] Cancel") + "\n")
	return s.String()
}

func (m model) viewSettleUpCreditor() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Settle Up - Step 2 of 3: Who received payment?") + "\n")

	for idx, mID := range m.activeGroup.Members {
		name := m.getUserName(mID)
		var line string
		if idx == m.settleCreditorCursor {
			line = selectedItemStyle.Render(fmt.Sprintf("➔ %s", name))
		} else {
			line = normalItemStyle.Render(fmt.Sprintf("  %s", name))
		}
		s.WriteString(line + "\n")
	}
	s.WriteString("\n" + helpStyle.Render("[enter] Next • [esc] Cancel") + "\n")
	return s.String()
}

func (m model) viewSettleUpAmount() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		headerStyle.Render("Settle Up - Step 3 of 3: How much?"),
		m.textInput.View(),
		helpStyle.Render("[enter] Record Settlement • [esc] Cancel"),
	)
}

func (m model) viewAddUserToGroup() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Add User to Group") + "\n")

	eligible := m.getEligibleUsersForGroup()
	if len(eligible) == 0 {
		s.WriteString(normalItemStyle.Render("All registered users are already members of this group.") + "\n\n")
		s.WriteString(helpStyle.Render("[esc] Go Back") + "\n")
		return s.String()
	}

	for idx, u := range eligible {
		var line string
		if idx == m.addUserToGroupCursor {
			line = selectedItemStyle.Render(fmt.Sprintf("➔ %s", u.Name))
		} else {
			line = normalItemStyle.Render(fmt.Sprintf("  %s", u.Name))
		}
		s.WriteString(line + "\n")
	}
	s.WriteString("\n" + helpStyle.Render("[enter] Add User • [esc] Cancel") + "\n")
	return s.String()
}

func (m model) viewEditExpenseParticipants() string {
	var s strings.Builder

	if len(m.activeGroup.Expenses) == 0 {
		return "No expense selected."
	}

	targetExp := m.activeGroup.Expenses[m.selectedExpenseIdx]
	s.WriteString(headerStyle.Render(fmt.Sprintf("Edit Who Owes: %s (%s)", targetExp.Description, formatAmount(targetExp.Amount))) + "\n")

	for idx, mID := range m.activeGroup.Members {
		name := m.getUserName(mID)
		checked := "[ ]"
		if m.expPartChecked[mID] {
			checked = "[x]"
		}
		var line string
		if idx == m.expPartCursor {
			line = selectedItemStyle.Render(fmt.Sprintf("➔ %s %s", checked, name))
		} else {
			line = normalItemStyle.Render(fmt.Sprintf("  %s %s", checked, name))
		}
		s.WriteString(line + "\n")
	}
	s.WriteString("\n" + helpStyle.Render("[space] Toggle • [enter] Save Splits • [esc] Cancel") + "\n")
	return s.String()
}

func (m model) viewSettleUpSelectDebt() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Select a Simplified Debt to Settle") + "\n")

	debts := m.activeGroupTransfers

	for idx, tr := range debts {
		fromName := m.getUserName(tr.From)
		toName := m.getUserName(tr.To)
		debtStr := fmt.Sprintf("%s owes %s %s", fromName, toName, formatAmount(tr.Amount))

		var line string
		if idx == m.settleDebtCursor {
			line = selectedItemStyle.Render(fmt.Sprintf("➔ %s", debtStr))
		} else {
			line = normalItemStyle.Render(fmt.Sprintf("  %s", debtStr))
		}
		s.WriteString(line + "\n")
	}

	var otherLine string
	if m.settleDebtCursor == len(debts) {
		otherLine = selectedItemStyle.Render("➔ [ Record custom payment... ]")
	} else {
		otherLine = normalItemStyle.Render("  [ Record custom payment... ]")
	}
	s.WriteString(otherLine + "\n")

	s.WriteString("\n" + helpStyle.Render("[enter] Select • [esc] Cancel") + "\n")
	return s.String()
}

func GenerateWhatsAppText(g domain.Group, allUsers []domain.User) string {
	balances := engine.CalculateBalances(g.Members, g.Expenses)
	transfers := engine.SettleDebts(balances)

	userMap := make(map[string]string)
	for _, u := range allUsers {
		userMap[u.ID] = u.Name
	}
	getUserName := func(id string) string {
		if name, ok := userMap[id]; ok {
			return name
		}
		return id
	}

	formatCents := func(cents int64) string {
		dollars := cents / 100
		remCents := cents % 100
		if remCents < 0 {
			remCents = -remCents
		}
		sign := ""
		if cents < 0 {
			sign = "-"
			if dollars < 0 {
				dollars = -dollars
			}
		}
		return fmt.Sprintf("%s$%d.%02d", sign, dollars, remCents)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "*BILLSHARE REPORT - %s*\n\n", strings.ToUpper(g.Name))

	sb.WriteString("*Net Balances:*\n")
	for _, mID := range g.Members {
		bal := balances[mID]
		name := getUserName(mID)
		balStr := formatCents(bal)
		if bal > 0 {
			balStr = "+" + balStr
		}
		fmt.Fprintf(&sb, "• %s: %s\n", name, balStr)
	}

	sb.WriteString("\n*Simplified Debts:*\n")
	if len(transfers) == 0 {
		sb.WriteString("All settled up! 🎉\n")
	} else {
		for _, tr := range transfers {
			fromName := getUserName(tr.From)
			toName := getUserName(tr.To)
			fmt.Fprintf(&sb, "• %s owes %s %s\n", fromName, toName, formatCents(tr.Amount))
		}
	}

	return sb.String()
}
