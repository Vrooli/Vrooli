package execution

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"swarm-manager/internal/storage"
)

// CircuitBreakerState tracks per-item failure counts and cooldown state.
type CircuitBreakerState struct {
	Items map[string]*CircuitBreakerEntry `json:"items"`
}

// CircuitBreakerEntry tracks consecutive failures for a single backlog item.
type CircuitBreakerEntry struct {
	ConsecutiveFailures int    `json:"consecutive_failures"`
	LastFailure         string `json:"last_failure"`
	BrokenAt            string `json:"broken_at,omitempty"`
}

// CircuitBreaker manages per-item failure tracking with file-backed persistence.
type CircuitBreaker struct {
	path string
	mu   sync.Mutex
}

// NewCircuitBreaker creates a circuit breaker backed by the given file path.
func NewCircuitBreaker(path string) *CircuitBreaker {
	return &CircuitBreaker{path: path}
}

// RecordFailure increments the failure counter for the given item key.
// If the counter reaches or exceeds the threshold, the breaker trips.
func (cb *CircuitBreaker) RecordFailure(itemKey string, threshold int) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, err := cb.loadLocked()
	if err != nil {
		return err
	}

	entry, ok := state.Items[itemKey]
	if !ok {
		entry = &CircuitBreakerEntry{}
		state.Items[itemKey] = entry
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry.ConsecutiveFailures++
	entry.LastFailure = now
	if entry.ConsecutiveFailures >= threshold && entry.BrokenAt == "" {
		entry.BrokenAt = now
	}

	return cb.saveLocked(state)
}

// RecordSuccess resets the failure counter for the given item key.
func (cb *CircuitBreaker) RecordSuccess(itemKey string) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, err := cb.loadLocked()
	if err != nil {
		return err
	}

	delete(state.Items, itemKey)
	return cb.saveLocked(state)
}

// IsBroken returns true if the item's breaker is tripped and the cooldown has not expired.
func (cb *CircuitBreaker) IsBroken(itemKey string, cooldownMinutes int) (bool, time.Duration, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, err := cb.loadLocked()
	if err != nil {
		return false, 0, err
	}

	entry, ok := state.Items[itemKey]
	if !ok || entry.BrokenAt == "" {
		return false, 0, nil
	}

	brokenAt, err := time.Parse(time.RFC3339, entry.BrokenAt)
	if err != nil {
		return false, 0, nil
	}

	cooldown := time.Duration(cooldownMinutes) * time.Minute
	remaining := time.Until(brokenAt.Add(cooldown))
	if remaining <= 0 {
		// Cooldown expired — auto-reset.
		delete(state.Items, itemKey)
		_ = cb.saveLocked(state)
		return false, 0, nil
	}

	return true, remaining, nil
}

// Reset explicitly clears the breaker for a specific item.
func (cb *CircuitBreaker) Reset(itemKey string) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, err := cb.loadLocked()
	if err != nil {
		return err
	}

	if _, ok := state.Items[itemKey]; !ok {
		return errNotFound
	}

	delete(state.Items, itemKey)
	return cb.saveLocked(state)
}

// BrokenItems returns a list of item keys that are currently circuit-broken.
func (cb *CircuitBreaker) BrokenItems(cooldownMinutes int) ([]string, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, err := cb.loadLocked()
	if err != nil {
		return nil, err
	}

	var broken []string
	for key, entry := range state.Items {
		if entry.BrokenAt == "" {
			continue
		}
		brokenAt, parseErr := time.Parse(time.RFC3339, entry.BrokenAt)
		if parseErr != nil {
			continue
		}
		cooldown := time.Duration(cooldownMinutes) * time.Minute
		if time.Until(brokenAt.Add(cooldown)) > 0 {
			broken = append(broken, key)
		}
	}
	return broken, nil
}

func (cb *CircuitBreaker) loadLocked() (CircuitBreakerState, error) {
	state := CircuitBreakerState{Items: map[string]*CircuitBreakerEntry{}}
	data, err := os.ReadFile(cb.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return state, err
	}
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return CircuitBreakerState{Items: map[string]*CircuitBreakerEntry{}}, nil
	}
	if state.Items == nil {
		state.Items = map[string]*CircuitBreakerEntry{}
	}
	return state, nil
}

func (cb *CircuitBreaker) saveLocked(state CircuitBreakerState) error {
	return storage.WriteJSONAtomic(cb.path, state)
}
