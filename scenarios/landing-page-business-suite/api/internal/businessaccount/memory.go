package businessaccount

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryRepository is used by focused domain and HTTP tests.
type MemoryRepository struct {
	mu       sync.Mutex
	accounts map[string]Account
	members  map[string]map[string]bool
	nextID   int
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{accounts: make(map[string]Account), members: make(map[string]map[string]bool)}
}

func (r *MemoryRepository) ListForUser(_ context.Context, userID, userEmail string) ([]Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ensureIdentity(userID, userEmail); err != nil {
		return nil, err
	}
	r.ensureDefaultLocked(userID, userEmail)
	var result []Account
	for id, members := range r.members {
		if !members[userID] {
			continue
		}
		result = append(result, r.accounts[id])
	}
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].ID < result[i].ID {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

func (r *MemoryRepository) ResolveForUser(_ context.Context, userID, userEmail, requestedID string) (Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ensureIdentity(userID, userEmail); err != nil {
		return Account{}, err
	}
	r.ensureDefaultLocked(userID, userEmail)
	requestedID = strings.TrimSpace(requestedID)
	if requestedID == "" {
		if members := r.members[defaultID(userID)]; members[userID] {
			return r.accounts[defaultID(userID)], nil
		}
		for id, members := range r.members {
			if members[userID] {
				return r.accounts[id], nil
			}
		}
		return Account{}, ErrNotFound
	}
	if !r.members[requestedID][userID] {
		return Account{}, ErrNotMember
	}
	return r.accounts[requestedID], nil
}

func (r *MemoryRepository) CreateForUser(_ context.Context, userID, userEmail, displayName string) (Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ensureIdentity(userID, userEmail); err != nil {
		return Account{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(displayName) > 200 {
		return Account{}, ErrInvalid
	}
	r.nextID++
	now := time.Now().UTC()
	account := Account{ID: fmt.Sprintf("acct:test-%d", r.nextID), DisplayName: displayName, BillingEmail: strings.ToLower(strings.TrimSpace(userEmail)), Role: "owner", CreatedAt: now}
	r.accounts[account.ID] = account
	r.members[account.ID] = map[string]bool{userID: true}
	return account, nil
}

func (r *MemoryRepository) ensureDefaultLocked(userID, userEmail string) {
	id := defaultID(userID)
	if _, ok := r.accounts[id]; !ok {
		r.accounts[id] = Account{ID: id, DisplayName: "Personal account", BillingEmail: strings.ToLower(strings.TrimSpace(userEmail)), Role: "owner", CreatedAt: time.Now().UTC()}
	}
	if r.members[id] == nil {
		r.members[id] = make(map[string]bool)
	}
	r.members[id][userID] = true
}

var _ Repository = (*MemoryRepository)(nil)
