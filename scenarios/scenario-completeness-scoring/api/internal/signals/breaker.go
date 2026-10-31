package signals

import (
	"sync"
	"time"
)

// breakerState is the circuit breaker state machine: closed lets calls
// through, open fast-fails until retryInterval elapses, halfOpen lets one
// probe through and re-opens on failure.
type breakerState int

const (
	stateClosed breakerState = iota
	stateOpen
	stateHalfOpen
)

const (
	// failureThreshold consecutive failures trip a closed breaker open.
	failureThreshold = 3
	// retryInterval is how long an open breaker fast-fails before allowing
	// a half-open probe.
	retryInterval = 30 * time.Second
)

// breaker guards one collector. Concurrency-safe; time is injected so tests
// can drive the open -> half-open transition without sleeping.
type breaker struct {
	mu              sync.Mutex
	state           breakerState
	failureCount    int
	lastStateChange time.Time

	threshold int
	retry     time.Duration
	now       func() time.Time
}

func newBreaker(now func() time.Time) *breaker {
	if now == nil {
		now = time.Now
	}
	return &breaker{
		state:           stateClosed,
		threshold:       failureThreshold,
		retry:           retryInterval,
		now:             now,
		lastStateChange: now(),
	}
}

// allow reports whether the next call may proceed. An open breaker
// transitions to half-open (and allows the call) once retry has elapsed.
func (b *breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case stateClosed, stateHalfOpen:
		return true
	case stateOpen:
		if b.now().Sub(b.lastStateChange) >= b.retry {
			b.state = stateHalfOpen
			b.lastStateChange = b.now()
			return true
		}
		return false
	default:
		return false
	}
}

func (b *breaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failureCount = 0
	if b.state == stateHalfOpen {
		b.state = stateClosed
		b.lastStateChange = b.now()
	}
}

func (b *breaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failureCount++
	switch b.state {
	case stateClosed:
		if b.failureCount >= b.threshold {
			b.state = stateOpen
			b.lastStateChange = b.now()
		}
	case stateHalfOpen:
		// Failed probe: re-open without resetting the failure history.
		b.state = stateOpen
		b.lastStateChange = b.now()
	case stateOpen:
		// Already open; nothing to transition.
	}
}

// breakerSet tracks one breaker per collector id.
type breakerSet struct {
	mu   sync.Mutex
	byID map[string]*breaker
	now  func() time.Time
}

func newBreakerSet(now func() time.Time) *breakerSet {
	return &breakerSet{byID: make(map[string]*breaker), now: now}
}

func (s *breakerSet) get(id string) *breaker {
	s.mu.Lock()
	defer s.mu.Unlock()

	if b, ok := s.byID[id]; ok {
		return b
	}
	b := newBreaker(s.now)
	s.byID[id] = b
	return b
}
