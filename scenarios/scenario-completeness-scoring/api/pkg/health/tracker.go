// Package health provides health monitoring for collectors
// [REQ:SCS-HEALTH-001] Health status API
// [REQ:SCS-HEALTH-002] Collector status tracking
// [REQ:SCS-HEALTH-003] Collector test endpoint
package health

import (
	"sync"
	"time"

	"scenario-completeness-scoring/pkg/circuitbreaker"
)

// CollectorStatus represents the health status of a collector
type CollectorStatus string

const (
	// StatusOK means the collector is working normally
	StatusOK CollectorStatus = "ok"
	// StatusDegraded means the collector is experiencing issues but still working
	StatusDegraded CollectorStatus = "degraded"
	// StatusFailed means the collector has failed and is disabled
	StatusFailed CollectorStatus = "failed"
)

// CollectorHealth holds health information for a single collector
// [REQ:SCS-HEALTH-002] Track collector status
type CollectorHealth struct {
	Name          string          `json:"name"`
	Status        CollectorStatus `json:"status"`
	LastCheck     *time.Time      `json:"last_check,omitempty"`
	LastSuccess   *time.Time      `json:"last_success,omitempty"`
	LastFailure   *time.Time      `json:"last_failure,omitempty"`
	FailureCount  int             `json:"failure_count"`
	Message       string          `json:"message,omitempty"`
	CircuitState  string          `json:"circuit_state,omitempty"`
}

// OverallHealth represents the overall system health
// [REQ:SCS-HEALTH-001] Overall health status
type OverallHealth struct {
	Status      CollectorStatus            `json:"status"`
	Collectors  map[string]CollectorHealth `json:"collectors"`
	Healthy     int                        `json:"healthy"`
	Degraded    int                        `json:"degraded"`
	Failed      int                        `json:"failed"`
	Total       int                        `json:"total"`
	CheckedAt   time.Time                  `json:"checked_at"`
}

// Tracker monitors health of collectors
// [REQ:SCS-HEALTH-002] Track OK, Degraded, Failed states
type Tracker struct {
	mu         sync.RWMutex
	collectors map[string]*collectorState
	registry   *circuitbreaker.Registry
}

// collectorState holds internal state for a collector
type collectorState struct {
	name           string
	lastCheck      time.Time
	lastSuccess    time.Time
	lastFailure    time.Time
	consecutiveFail int
	totalFail      int
	totalSuccess   int
	lastError      string
}

// NewTracker creates a new health tracker with circuit breaker integration
func NewTracker(registry *circuitbreaker.Registry) *Tracker {
	return &Tracker{
		collectors: make(map[string]*collectorState),
		registry:   registry,
	}
}

// RecordCheck records the result of a collector check
// [REQ:SCS-HEALTH-002] Track collector status
func (t *Tracker) RecordCheck(name string, success bool, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, ok := t.collectors[name]
	if !ok {
		state = &collectorState{name: name}
		t.collectors[name] = state
	}

	state.lastCheck = time.Now()

	if success {
		state.lastSuccess = time.Now()
		state.consecutiveFail = 0
		state.totalSuccess++
		state.lastError = ""
	} else {
		state.lastFailure = time.Now()
		state.consecutiveFail++
		state.totalFail++
		state.lastError = errMsg
	}
}

// GetCollectorHealth returns the health of a specific collector
// [REQ:SCS-HEALTH-002] Get collector status
func (t *Tracker) GetCollectorHealth(name string) CollectorHealth {
	t.mu.RLock()
	state := t.collectors[name]
	t.mu.RUnlock()

	health := CollectorHealth{
		Name: name,
	}

	// Check circuit breaker state
	if t.registry != nil {
		if cb := t.registry.GetIfExists(name); cb != nil {
			health.CircuitState = cb.State().String()
			health.FailureCount = cb.FailureCount()

			// Failed if circuit is open
			if cb.State() == circuitbreaker.Open {
				health.Status = StatusFailed
				health.Message = "Circuit breaker tripped due to consecutive failures"
				return health
			}
		}
	}

	if state == nil {
		health.Status = StatusOK // No state means not yet checked
		health.Message = "Not yet checked"
		return health
	}

	// Set timestamps
	if !state.lastCheck.IsZero() {
		t := state.lastCheck
		health.LastCheck = &t
	}
	if !state.lastSuccess.IsZero() {
		t := state.lastSuccess
		health.LastSuccess = &t
	}
	if !state.lastFailure.IsZero() {
		t := state.lastFailure
		health.LastFailure = &t
	}

	// Determine status based on failure pattern
	if state.consecutiveFail >= 3 {
		health.Status = StatusFailed
		health.Message = state.lastError
	} else if state.consecutiveFail > 0 {
		health.Status = StatusDegraded
		health.Message = state.lastError
	} else {
		health.Status = StatusOK
	}

	return health
}

// GetOverallHealth returns the overall system health
// [REQ:SCS-HEALTH-001] Health status API
func (t *Tracker) GetOverallHealth() OverallHealth {
	t.mu.RLock()
	defer t.mu.RUnlock()

	health := OverallHealth{
		Collectors: make(map[string]CollectorHealth),
		CheckedAt:  time.Now(),
	}

	// Default collector names
	defaultCollectors := []string{"requirements", "tests", "ui", "service"}

	for _, name := range defaultCollectors {
		ch := t.GetCollectorHealth(name)
		health.Collectors[name] = ch
		health.Total++

		switch ch.Status {
		case StatusOK:
			health.Healthy++
		case StatusDegraded:
			health.Degraded++
		case StatusFailed:
			health.Failed++
		}
	}

	// Add any additional collectors we've tracked
	for name := range t.collectors {
		found := false
		for _, def := range defaultCollectors {
			if name == def {
				found = true
				break
			}
		}
		if !found {
			ch := t.GetCollectorHealth(name)
			health.Collectors[name] = ch
			health.Total++

			switch ch.Status {
			case StatusOK:
				health.Healthy++
			case StatusDegraded:
				health.Degraded++
			case StatusFailed:
				health.Failed++
			}
		}
	}

	// Determine overall status
	if health.Failed > 0 {
		health.Status = StatusFailed
	} else if health.Degraded > 0 {
		health.Status = StatusDegraded
	} else {
		health.Status = StatusOK
	}

	return health
}

// TestResult holds the result of testing a collector
type TestResult struct {
	Name      string          `json:"name"`
	Success   bool            `json:"success"`
	Duration  time.Duration   `json:"duration_ms"`
	Error     string          `json:"error,omitempty"`
	Status    CollectorStatus `json:"status"`
}

// CollectorTestFunc is a function that tests a collector
type CollectorTestFunc func() error

// TestCollector tests a specific collector and records the result
// [REQ:SCS-HEALTH-003] Test specific collector on demand
func (t *Tracker) TestCollector(name string, testFn CollectorTestFunc) TestResult {
	start := time.Now()

	result := TestResult{
		Name: name,
	}

	err := testFn()
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Status = StatusFailed
		t.RecordCheck(name, false, err.Error())
	} else {
		result.Success = true
		result.Status = StatusOK
		t.RecordCheck(name, true, "")
	}

	return result
}
