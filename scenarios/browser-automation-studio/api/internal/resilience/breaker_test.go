package resilience

import (
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBreaker(t *testing.T) {
	cfg := DefaultBreakerConfig("test-breaker")
	b := NewBreaker(cfg)

	assert.NotNil(t, b)
	assert.Equal(t, "test-breaker", b.Name())
	assert.Equal(t, StateClosed, b.State())
	assert.False(t, b.IsOpen())
}

func TestBreakerExecuteSuccess(t *testing.T) {
	cfg := DefaultBreakerConfig("test-success")
	b := NewBreaker(cfg)

	result, err := b.Execute(func() (any, error) {
		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, StateClosed, b.State())
}

func TestBreakerTripsAfterFailures(t *testing.T) {
	cfg := DefaultBreakerConfig("test-trip")
	cfg.FailureThreshold = 3
	cfg.FailureRatio = 0.5
	cfg.Timeout = 100 * time.Millisecond
	cfg.Logger = logrus.New()

	stateChanges := make([]BreakerState, 0)
	cfg.OnStateChange = func(name string, from, to BreakerState) {
		stateChanges = append(stateChanges, to)
	}

	b := NewBreaker(cfg)
	testErr := errors.New("test failure")

	// Generate enough failures to trip the breaker
	for i := 0; i < 5; i++ {
		_, err := b.Execute(func() (any, error) {
			return nil, testErr
		})
		assert.Error(t, err)
	}

	// Breaker should now be open
	assert.Equal(t, StateOpen, b.State())
	assert.True(t, b.IsOpen())

	// Subsequent calls should fail fast with ErrCircuitOpen
	_, err := b.Execute(func() (any, error) {
		return "should not execute", nil
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCircuitOpen))

	// State change callback should have been called
	assert.Contains(t, stateChanges, StateOpen)
}

func TestBreakerRecovery(t *testing.T) {
	cfg := DefaultBreakerConfig("test-recovery")
	cfg.FailureThreshold = 2
	cfg.FailureRatio = 0.5
	cfg.Timeout = 50 * time.Millisecond // Short timeout for testing
	cfg.MaxRequests = 1
	cfg.Logger = logrus.New()

	b := NewBreaker(cfg)
	testErr := errors.New("test failure")

	// Trip the breaker
	for i := 0; i < 3; i++ {
		b.Execute(func() (any, error) {
			return nil, testErr
		})
	}
	assert.Equal(t, StateOpen, b.State())

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)

	// Next successful call should close the breaker
	result, err := b.Execute(func() (any, error) {
		return "recovered", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "recovered", result)
	assert.Equal(t, StateClosed, b.State())
}

func TestDefaultBreakerConfig(t *testing.T) {
	cfg := DefaultBreakerConfig("test")

	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, uint32(1), cfg.MaxRequests)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, uint32(5), cfg.FailureThreshold)
	assert.Equal(t, 0.6, cfg.FailureRatio)
}

func TestConfigFromEnv(t *testing.T) {
	// Set test environment variables
	t.Setenv("TEST_CB_TIMEOUT", "45s")
	t.Setenv("TEST_CB_FAILURE_THRESHOLD", "10")
	t.Setenv("TEST_CB_FAILURE_RATIO", "0.8")
	t.Setenv("TEST_CB_MAX_REQUESTS", "3")

	cfg := ConfigFromEnv("TEST", "test-env")

	assert.Equal(t, "test-env", cfg.Name)
	assert.Equal(t, 45*time.Second, cfg.Timeout)
	assert.Equal(t, uint32(10), cfg.FailureThreshold)
	assert.Equal(t, 0.8, cfg.FailureRatio)
	assert.Equal(t, uint32(3), cfg.MaxRequests)
}

func TestConfigFromEnvDefaults(t *testing.T) {
	// No env vars set - should use defaults
	cfg := ConfigFromEnv("NONEXISTENT", "default-test")

	assert.Equal(t, "default-test", cfg.Name)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, uint32(5), cfg.FailureThreshold)
	assert.Equal(t, 0.6, cfg.FailureRatio)
	assert.Equal(t, uint32(1), cfg.MaxRequests)
}

func TestRegistry(t *testing.T) {
	log := logrus.New()
	r := NewRegistry(log)

	// Get creates a new breaker if not exists
	b1 := r.Get("breaker-1")
	assert.NotNil(t, b1)
	assert.Equal(t, "breaker-1", b1.Name())

	// Get returns the same breaker for same name
	b1Again := r.Get("breaker-1")
	assert.Same(t, b1, b1Again)

	// Different name creates different breaker
	b2 := r.Get("breaker-2")
	assert.NotSame(t, b1, b2)

	// States returns all breaker states
	states := r.States()
	assert.Len(t, states, 2)
	assert.Equal(t, StateClosed, states["breaker-1"])
	assert.Equal(t, StateClosed, states["breaker-2"])
}

func TestRegistryGetOrCreate(t *testing.T) {
	log := logrus.New()
	r := NewRegistry(log)

	cfg := DefaultBreakerConfig("custom-breaker")
	cfg.FailureThreshold = 10

	b1 := r.GetOrCreate("custom-breaker", cfg)
	assert.NotNil(t, b1)

	// GetOrCreate with different config returns existing breaker
	cfg2 := DefaultBreakerConfig("custom-breaker")
	cfg2.FailureThreshold = 20
	b1Again := r.GetOrCreate("custom-breaker", cfg2)
	assert.Same(t, b1, b1Again)
}

func TestBreakerCounts(t *testing.T) {
	cfg := DefaultBreakerConfig("test-counts")
	b := NewBreaker(cfg)

	// Initial counts should be zero
	counts := b.Counts()
	assert.Equal(t, uint32(0), counts.Requests)
	assert.Equal(t, uint32(0), counts.TotalSuccesses)
	assert.Equal(t, uint32(0), counts.TotalFailures)

	// Execute some operations
	b.Execute(func() (any, error) { return nil, nil })
	b.Execute(func() (any, error) { return nil, nil })
	b.Execute(func() (any, error) { return nil, errors.New("fail") })

	counts = b.Counts()
	assert.Equal(t, uint32(3), counts.Requests)
	assert.Equal(t, uint32(2), counts.TotalSuccesses)
	assert.Equal(t, uint32(1), counts.TotalFailures)
}
