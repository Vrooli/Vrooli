// Package resilience provides circuit breaker and retry patterns for external service calls.
package resilience

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker/v2"
)

// Common errors returned by circuit breaker operations.
var (
	// ErrCircuitOpen is returned when the circuit breaker is open and rejecting requests.
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrServiceUnavailable is returned when the underlying service is unavailable.
	ErrServiceUnavailable = errors.New("service unavailable")
)

// BreakerState represents the current state of a circuit breaker.
type BreakerState string

const (
	StateClosed   BreakerState = "closed"
	StateOpen     BreakerState = "open"
	StateHalfOpen BreakerState = "half-open"
)

// BreakerConfig configures a circuit breaker instance.
type BreakerConfig struct {
	// Name identifies this breaker in logs and metrics.
	Name string

	// MaxRequests is the maximum number of requests allowed in half-open state.
	// Default: 1
	MaxRequests uint32

	// Interval is the cyclic period of the closed state for clearing internal counts.
	// If 0, internal counts are never cleared while in closed state.
	// Default: 0 (never clear)
	Interval time.Duration

	// Timeout is how long to wait before transitioning from open to half-open.
	// Default: 30 seconds
	Timeout time.Duration

	// FailureThreshold is the minimum number of requests needed before the breaker can trip.
	// Default: 5
	FailureThreshold uint32

	// FailureRatio is the failure ratio threshold (0.0-1.0) to trip the breaker.
	// Default: 0.6 (60% failure rate)
	FailureRatio float64

	// Logger for state change events.
	Logger *logrus.Logger

	// OnStateChange is called when the breaker state changes.
	OnStateChange func(name string, from, to BreakerState)
}

// DefaultBreakerConfig returns sensible defaults for a circuit breaker.
func DefaultBreakerConfig(name string) BreakerConfig {
	return BreakerConfig{
		Name:             name,
		MaxRequests:      1,
		Interval:         0,
		Timeout:          30 * time.Second,
		FailureThreshold: 5,
		FailureRatio:     0.6,
	}
}

// ConfigFromEnv loads circuit breaker configuration from environment variables.
// Prefix should be like "PLAYWRIGHT" or "MINIO" - will look for:
// - {PREFIX}_CB_TIMEOUT (duration, e.g., "30s")
// - {PREFIX}_CB_FAILURE_THRESHOLD (int)
// - {PREFIX}_CB_FAILURE_RATIO (float, 0.0-1.0)
// - {PREFIX}_CB_MAX_REQUESTS (int, for half-open state)
func ConfigFromEnv(prefix, name string) BreakerConfig {
	cfg := DefaultBreakerConfig(name)
	prefix = strings.ToUpper(prefix)

	if v := os.Getenv(prefix + "_CB_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.Timeout = d
		}
	}
	if v := os.Getenv(prefix + "_CB_FAILURE_THRESHOLD"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil && n > 0 {
			cfg.FailureThreshold = uint32(n)
		}
	}
	if v := os.Getenv(prefix + "_CB_FAILURE_RATIO"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 1.0 {
			cfg.FailureRatio = f
		}
	}
	if v := os.Getenv(prefix + "_CB_MAX_REQUESTS"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil && n > 0 {
			cfg.MaxRequests = uint32(n)
		}
	}

	return cfg
}

// Breaker wraps a gobreaker circuit breaker with additional functionality.
type Breaker struct {
	cb     *gobreaker.CircuitBreaker[any]
	name   string
	log    *logrus.Logger
	config BreakerConfig
}

// NewBreaker creates a new circuit breaker with the given configuration.
func NewBreaker(cfg BreakerConfig) *Breaker {
	log := cfg.Logger
	if log == nil {
		log = logrus.StandardLogger()
	}

	readyToTrip := func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= cfg.FailureThreshold && failureRatio >= cfg.FailureRatio
	}

	onStateChange := func(name string, from, to gobreaker.State) {
		fromState := gobreakerStateToState(from)
		toState := gobreakerStateToState(to)

		log.WithFields(logrus.Fields{
			"breaker": name,
			"from":    fromState,
			"to":      toState,
		}).Info("Circuit breaker state changed")

		if cfg.OnStateChange != nil {
			cfg.OnStateChange(name, fromState, toState)
		}
	}

	settings := gobreaker.Settings{
		Name:          cfg.Name,
		MaxRequests:   cfg.MaxRequests,
		Interval:      cfg.Interval,
		Timeout:       cfg.Timeout,
		ReadyToTrip:   readyToTrip,
		OnStateChange: onStateChange,
		IsSuccessful: func(err error) bool {
			// Consider context cancellation as success (user cancelled, not service failure)
			if errors.Is(err, context.Canceled) {
				return true
			}
			// Consider deadline exceeded as failure (timeout = service slow/unavailable)
			if errors.Is(err, context.DeadlineExceeded) {
				return false
			}
			return err == nil
		},
	}

	return &Breaker{
		cb:     gobreaker.NewCircuitBreaker[any](settings),
		name:   cfg.Name,
		log:    log,
		config: cfg,
	}
}

// Execute runs the given function through the circuit breaker.
// Returns ErrCircuitOpen if the breaker is open.
func (b *Breaker) Execute(fn func() (any, error)) (any, error) {
	result, err := b.cb.Execute(fn)
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: %s circuit breaker is open, service may be unavailable", ErrCircuitOpen, b.name)
		}
		return nil, err
	}
	return result, nil
}

// ExecuteContext runs the given function with context through the circuit breaker.
func (b *Breaker) ExecuteContext(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	return b.Execute(func() (any, error) {
		return fn(ctx)
	})
}

// State returns the current state of the circuit breaker.
func (b *Breaker) State() BreakerState {
	return gobreakerStateToState(b.cb.State())
}

// Counts returns the current counts for monitoring.
func (b *Breaker) Counts() gobreaker.Counts {
	return b.cb.Counts()
}

// Name returns the breaker name.
func (b *Breaker) Name() string {
	return b.name
}

// IsOpen returns true if the circuit breaker is in open state.
func (b *Breaker) IsOpen() bool {
	return b.cb.State() == gobreaker.StateOpen
}

func gobreakerStateToState(s gobreaker.State) BreakerState {
	switch s {
	case gobreaker.StateClosed:
		return StateClosed
	case gobreaker.StateOpen:
		return StateOpen
	case gobreaker.StateHalfOpen:
		return StateHalfOpen
	default:
		return StateClosed
	}
}

// Registry manages multiple circuit breakers by name.
type Registry struct {
	mu       sync.RWMutex
	breakers map[string]*Breaker
	log      *logrus.Logger
}

// NewRegistry creates a new circuit breaker registry.
func NewRegistry(log *logrus.Logger) *Registry {
	if log == nil {
		log = logrus.StandardLogger()
	}
	return &Registry{
		breakers: make(map[string]*Breaker),
		log:      log,
	}
}

// Get returns a breaker by name, creating it with defaults if it doesn't exist.
func (r *Registry) Get(name string) *Breaker {
	r.mu.RLock()
	if b, ok := r.breakers[name]; ok {
		r.mu.RUnlock()
		return b
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if b, ok := r.breakers[name]; ok {
		return b
	}

	cfg := DefaultBreakerConfig(name)
	cfg.Logger = r.log
	b := NewBreaker(cfg)
	r.breakers[name] = b
	return b
}

// GetOrCreate returns a breaker by name, creating it with the given config if it doesn't exist.
func (r *Registry) GetOrCreate(name string, cfg BreakerConfig) *Breaker {
	r.mu.RLock()
	if b, ok := r.breakers[name]; ok {
		r.mu.RUnlock()
		return b
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if b, ok := r.breakers[name]; ok {
		return b
	}

	if cfg.Logger == nil {
		cfg.Logger = r.log
	}
	b := NewBreaker(cfg)
	r.breakers[name] = b
	return b
}

// States returns the current state of all registered breakers.
func (r *Registry) States() map[string]BreakerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	states := make(map[string]BreakerState, len(r.breakers))
	for name, b := range r.breakers {
		states[name] = b.State()
	}
	return states
}

// DefaultRegistry is a global circuit breaker registry.
var DefaultRegistry = NewRegistry(nil)

// SetDefaultLogger sets the logger for the default registry.
func SetDefaultLogger(log *logrus.Logger) {
	DefaultRegistry.log = log
}
