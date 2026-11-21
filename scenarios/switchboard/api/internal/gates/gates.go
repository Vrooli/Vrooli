// Package gates owns the short-lived, owner-only approval requests raised by
// a conversation when a turn needs a capability outside its grant.
package gates

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	Pending Status = "pending"
	Granted Status = "granted"
	Denied  Status = "denied"
	Expired Status = "expired"
)

type Gate struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	OwnerID   string    `json:"owner_id"`
	Scope     string    `json:"scope"`
	Withheld  string    `json:"withheld"`
	Unblock   string    `json:"unblock"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    Status    `json:"status"`
	GrantOnce bool      `json:"grant_once"`
}

var (
	ErrNotOwner          = errors.New("only the thread owner may answer a capability gate")
	ErrNotPending        = errors.New("capability gate is no longer pending")
	ErrInvalidTransition = errors.New("invalid capability gate status transition")
)

func (s Status) valid() bool {
	switch s {
	case Pending, Granted, Denied, Expired:
		return true
	default:
		return false
	}
}

func (s Status) CanTransitionTo(next Status) bool {
	if s == next {
		return s.valid()
	}
	return s == Pending && (next == Granted || next == Denied || next == Expired)
}

func (g *Gate) TransitionTo(next Status) error {
	if g == nil {
		return fmt.Errorf("%w: nil gate", ErrInvalidTransition)
	}
	if !g.Status.valid() || !next.valid() || !g.Status.CanTransitionTo(next) {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidTransition, g.Status, next)
	}
	g.Status = next
	return nil
}

type Service struct {
	mu    sync.Mutex
	now   func() time.Time
	gates map[string]Gate
}

func New(now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{now: now, gates: make(map[string]Gate)}
}

func (s *Service) Raise(threadID, ownerID, scope, withheld, unblock string, ttl time.Duration) (Gate, error) {
	if threadID == "" || ownerID == "" || scope == "" {
		return Gate{}, fmt.Errorf("thread, owner, and scope are required")
	}
	if ttl <= 0 {
		return Gate{}, fmt.Errorf("gate ttl must be positive")
	}
	now := s.now().UTC()
	g := Gate{ID: uuid.NewString(), ThreadID: threadID, OwnerID: ownerID, Scope: scope, Withheld: withheld, Unblock: unblock, CreatedAt: now, ExpiresAt: now.Add(ttl), Status: Pending, GrantOnce: true}
	s.mu.Lock()
	s.gates[g.ID] = g
	s.mu.Unlock()
	return g, nil
}

func (s *Service) Get(id string) (Gate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.gates[id]
	if ok {
		g = s.expireLocked(g)
	}
	return g, ok
}

func (s *Service) Answer(id, actorID string, grant bool) (Gate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.gates[id]
	if !ok {
		return Gate{}, fmt.Errorf("gate %q not found", id)
	}
	g = s.expireLocked(g)
	if actorID != g.OwnerID {
		return Gate{}, ErrNotOwner
	}
	if g.Status != Pending {
		return Gate{}, ErrNotPending
	}
	if grant {
		if err := g.TransitionTo(Granted); err != nil {
			return Gate{}, err
		}
	} else {
		if err := g.TransitionTo(Denied); err != nil {
			return Gate{}, err
		}
	}
	s.gates[id] = g
	return g, nil
}

func (s *Service) Expire() []Gate {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := make([]Gate, 0)
	for id, g := range s.gates {
		if g.Status == Pending && !s.now().Before(g.ExpiresAt) {
			if err := g.TransitionTo(Expired); err != nil {
				continue
			}
			s.gates[id] = g
			changed = append(changed, g)
		}
	}
	return changed
}

func (s *Service) expireLocked(g Gate) Gate {
	if g.Status == Pending && !s.now().Before(g.ExpiresAt) {
		if g.TransitionTo(Expired) == nil {
			s.gates[g.ID] = g
		}
	}
	return g
}
