package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/chandanpasunoori/billshare/pkg/domain"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrGroupNotFound   = errors.New("group not found")
	ErrExpenseNotFound = errors.New("expense not found")
)

// Store defines the storage operations for the application.
type Store interface {
	CreateUser(name string) (domain.User, error)
	GetUser(id string) (domain.User, error)
	ListUsers() ([]domain.User, error)

	CreateGroup(name string, memberIDs []string) (domain.Group, error)
	GetGroup(id string) (domain.Group, error)
	ListGroups() ([]domain.Group, error)
	AddUserToGroup(groupID string, userID string) error

	AddExpense(groupID string, expense domain.Expense) (domain.Expense, error)
	DeleteExpense(groupID string, expenseID string) error
	UpdateExpenseSplits(groupID string, expenseID string, splits map[string]int64) error

	Save() error
}

// JSONStore is a file-backed JSON storage.
type JSONStore struct {
	filePath string
	mu       sync.RWMutex
	Users    map[string]domain.User  `json:"users"`
	Groups   map[string]domain.Group `json:"groups"`
}

// NewJSONStore creates a new JSONStore. If the file exists, it loads it.
func NewJSONStore(filePath string) (*JSONStore, error) {
	s := &JSONStore{
		filePath: filePath,
		Users:    make(map[string]domain.User),
		Groups:   make(map[string]domain.Group),
	}

	if err := s.load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Save default empty structures
			if err := s.Save(); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("failed to load database file: %w", err)
		}
	}

	return s, nil
}

func (s *JSONStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, s)
}

func (s *JSONStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// CreateUser creates a new user.
func (s *JSONStore) CreateUser(name string) (domain.User, error) {
	s.mu.Lock()
	// Generate a simple ID based on timestamp
	id := fmt.Sprintf("u_%d", time.Now().UnixNano())
	u := domain.User{
		ID:   id,
		Name: name,
	}
	s.Users[id] = u
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// GetUser retrieves a user by ID.
func (s *JSONStore) GetUser(id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, exists := s.Users[id]
	if !exists {
		return domain.User{}, ErrUserNotFound
	}
	return u, nil
}

// ListUsers lists all users.
func (s *JSONStore) ListUsers() ([]domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []domain.User
	for _, u := range s.Users {
		users = append(users, u)
	}
	return users, nil
}

// CreateGroup creates a new group with the given members.
func (s *JSONStore) CreateGroup(name string, memberIDs []string) (domain.Group, error) {
	s.mu.Lock()
	id := fmt.Sprintf("g_%d", time.Now().UnixNano())
	g := domain.Group{
		ID:       id,
		Name:     name,
		Members:  memberIDs,
		Expenses: []domain.Expense{},
	}
	s.Groups[id] = g
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return domain.Group{}, err
	}
	return g, nil
}

// GetGroup retrieves a group by ID.
func (s *JSONStore) GetGroup(id string) (domain.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, exists := s.Groups[id]
	if !exists {
		return domain.Group{}, ErrGroupNotFound
	}
	return g, nil
}

// ListGroups lists all groups.
func (s *JSONStore) ListGroups() ([]domain.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var groups []domain.Group
	for _, g := range s.Groups {
		groups = append(groups, g)
	}
	return groups, nil
}

// AddExpense adds an expense to a group.
func (s *JSONStore) AddExpense(groupID string, expense domain.Expense) (domain.Expense, error) {
	s.mu.Lock()
	g, exists := s.Groups[groupID]
	if !exists {
		s.mu.Unlock()
		return domain.Expense{}, ErrGroupNotFound
	}

	expense.ID = fmt.Sprintf("e_%d", time.Now().UnixNano())
	expense.Date = time.Now()

	g.Expenses = append(g.Expenses, expense)
	s.Groups[groupID] = g
	s.mu.Unlock()

	if err := s.Save(); err != nil {
		return domain.Expense{}, err
	}
	return expense, nil
}

// DeleteExpense removes an expense from a group.
func (s *JSONStore) DeleteExpense(groupID string, expenseID string) error {
	s.mu.Lock()
	g, exists := s.Groups[groupID]
	if !exists {
		s.mu.Unlock()
		return ErrGroupNotFound
	}

	foundIdx := -1
	for idx, exp := range g.Expenses {
		if exp.ID == expenseID {
			foundIdx = idx
			break
		}
	}

	if foundIdx == -1 {
		s.mu.Unlock()
		return ErrExpenseNotFound
	}

	// Remove the expense
	g.Expenses = append(g.Expenses[:foundIdx], g.Expenses[foundIdx+1:]...)
	s.Groups[groupID] = g
	s.mu.Unlock()

	return s.Save()
}

// AddUserToGroup adds a user to an existing group.
func (s *JSONStore) AddUserToGroup(groupID string, userID string) error {
	s.mu.Lock()
	g, exists := s.Groups[groupID]
	if !exists {
		s.mu.Unlock()
		return ErrGroupNotFound
	}

	_, exists = s.Users[userID]
	if !exists {
		s.mu.Unlock()
		return ErrUserNotFound
	}

	// Check if already a member
	for _, m := range g.Members {
		if m == userID {
			s.mu.Unlock()
			return nil // already a member, no drama
		}
	}

	g.Members = append(g.Members, userID)
	s.Groups[groupID] = g
	s.mu.Unlock()

	return s.Save()
}

// UpdateExpenseSplits updates the split amounts for a specific expense.
func (s *JSONStore) UpdateExpenseSplits(groupID string, expenseID string, splits map[string]int64) error {
	s.mu.Lock()
	g, exists := s.Groups[groupID]
	if !exists {
		s.mu.Unlock()
		return ErrGroupNotFound
	}

	foundIdx := -1
	for idx, exp := range g.Expenses {
		if exp.ID == expenseID {
			foundIdx = idx
			break
		}
	}

	if foundIdx == -1 {
		s.mu.Unlock()
		return ErrExpenseNotFound
	}

	// Update splits
	g.Expenses[foundIdx].Splits = splits
	s.Groups[groupID] = g
	s.mu.Unlock()

	return s.Save()
}
