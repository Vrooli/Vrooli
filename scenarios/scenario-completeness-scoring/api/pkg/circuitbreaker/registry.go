// Package circuitbreaker provides circuit breaker registry for managing multiple breakers
// [REQ:SCS-CB-001] Manage circuit breakers for all collectors
// [REQ:SCS-CB-004] Reset all breakers via API
package circuitbreaker

import (
	"sync"
)

// Registry manages a collection of circuit breakers
type Registry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   Config
}

// NewRegistry creates a new circuit breaker registry with the given default config
func NewRegistry(cfg Config) *Registry {
	return &Registry{
		breakers: make(map[string]*CircuitBreaker),
		config:   cfg,
	}
}

// Get returns the circuit breaker for the given name, creating one if it doesn't exist
func (r *Registry) Get(name string) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cb, ok := r.breakers[name]; ok {
		return cb
	}

	cb := New(name, r.config)
	r.breakers[name] = cb
	return cb
}

// GetIfExists returns the circuit breaker for the given name, or nil if it doesn't exist
func (r *Registry) GetIfExists(name string) *CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.breakers[name]
}

// List returns all registered circuit breakers
func (r *Registry) List() []*CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	breakers := make([]*CircuitBreaker, 0, len(r.breakers))
	for _, cb := range r.breakers {
		breakers = append(breakers, cb)
	}
	return breakers
}

// ListStatus returns the status of all circuit breakers
func (r *Registry) ListStatus() []Status {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make([]Status, 0, len(r.breakers))
	for _, cb := range r.breakers {
		statuses = append(statuses, cb.Status())
	}
	return statuses
}

// Reset resets the circuit breaker with the given name
// [REQ:SCS-CB-004] Reset specific breaker
func (r *Registry) Reset(name string) bool {
	r.mu.RLock()
	cb, ok := r.breakers[name]
	r.mu.RUnlock()

	if !ok {
		return false
	}

	cb.Reset()
	return true
}

// ResetAll resets all circuit breakers
// [REQ:SCS-CB-004] Reset all breakers via API
func (r *Registry) ResetAll() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, cb := range r.breakers {
		cb.Reset()
		count++
	}
	return count
}

// Stats returns aggregate statistics about all circuit breakers
type Stats struct {
	Total    int `json:"total"`
	Closed   int `json:"closed"`
	Open     int `json:"open"`
	HalfOpen int `json:"half_open"`
}

// Stats returns aggregate statistics
func (r *Registry) Stats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := Stats{
		Total: len(r.breakers),
	}

	for _, cb := range r.breakers {
		switch cb.State() {
		case Closed:
			stats.Closed++
		case Open:
			stats.Open++
		case HalfOpen:
			stats.HalfOpen++
		}
	}

	return stats
}

// OpenBreakers returns a list of open circuit breakers
func (r *Registry) OpenBreakers() []Status {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var open []Status
	for _, cb := range r.breakers {
		if cb.State() == Open {
			open = append(open, cb.Status())
		}
	}
	return open
}
