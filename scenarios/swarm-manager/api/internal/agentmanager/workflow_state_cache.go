package agentmanager

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultWorkflowStateCacheTTL bounds how stale a cached workflow lifecycle
// projection may be. It is shorter than the reconciler tick so a terminal
// transition is still observed on the next pass after it lands.
const DefaultWorkflowStateCacheTTL = 3 * time.Second

// workflowStates is the process-wide cache behind readWorkflowExecutionState.
var workflowStates = newWorkflowStateCache(DefaultWorkflowStateCacheTTL)

// SetWorkflowStateCacheTTL changes the cache window (zero disables caching)
// and drops every cached entry. Tests use it; production keeps the default.
func SetWorkflowStateCacheTTL(ttl time.Duration) {
	workflowStates.mu.Lock()
	defer workflowStates.mu.Unlock()
	workflowStates.ttl = ttl
	workflowStates.entries = make(map[string]workflowStateEntry)
}

// InvalidateWorkflowStateCache drops every cached workflow state so the next
// read observes agent-manager directly.
func InvalidateWorkflowStateCache() { SetWorkflowStateCacheTTL(workflowStates.currentTTL()) }

type workflowStateEntry struct {
	state   WorkflowExecutionState
	expires time.Time
}

type workflowStateCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	entries map[string]workflowStateEntry
	flight  singleflight.Group
}

func newWorkflowStateCache(ttl time.Duration) *workflowStateCache {
	return &workflowStateCache{ttl: ttl, now: time.Now, entries: make(map[string]workflowStateEntry)}
}

func (c *workflowStateCache) currentTTL() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ttl
}

func (c *workflowStateCache) lookup(key string) (WorkflowExecutionState, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || !c.now().Before(entry.expires) {
		return WorkflowExecutionState{}, false
	}
	return entry.state, true
}

// get returns the cached state for key or runs fetch once for all concurrent
// callers. Only successful reads are cached: an error is returned to every
// waiting caller but not remembered, so the next tick observes recovery.
func (c *workflowStateCache) get(key string, fetch func() (WorkflowExecutionState, error)) (WorkflowExecutionState, error) {
	if state, ok := c.lookup(key); ok {
		return state, nil
	}
	result, err, _ := c.flight.Do(key, func() (any, error) {
		if state, ok := c.lookup(key); ok {
			return state, nil
		}
		state, fetchErr := fetch()
		if fetchErr != nil {
			return WorkflowExecutionState{}, fetchErr
		}
		c.mu.Lock()
		if c.ttl > 0 {
			c.entries[key] = workflowStateEntry{state: state, expires: c.now().Add(c.ttl)}
			c.pruneLocked()
		}
		c.mu.Unlock()
		return state, nil
	})
	if err != nil {
		return WorkflowExecutionState{}, err
	}
	return result.(WorkflowExecutionState), nil
}

// pruneLocked drops expired entries once the map grows past a small bound so
// a long-lived process does not accumulate one entry per workflow ever seen.
func (c *workflowStateCache) pruneLocked() {
	if len(c.entries) < 256 {
		return
	}
	now := c.now()
	for key, entry := range c.entries {
		if !now.Before(entry.expires) {
			delete(c.entries, key)
		}
	}
}
