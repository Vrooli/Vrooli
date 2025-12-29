package supervisor

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(io.Discard)
	return log
}

func testConfig() Config {
	return Config{
		Enabled:           true,
		MaxRestarts:       3,
		RestartWindow:     1 * time.Minute,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		StartupTimeout:    500 * time.Millisecond,
		GracefulStop:      100 * time.Millisecond,
	}
}

func TestProcessSupervisor_Start(t *testing.T) {
	t.Run("starts process and becomes running", func(t *testing.T) {
		mock := NewMockProcess()
		healthCalls := 0
		healthCheck := func(ctx context.Context) error {
			healthCalls++
			return nil
		}

		sup := NewProcessSupervisor(testConfig(), mock, healthCheck, testLogger())

		err := sup.Start(context.Background())
		require.NoError(t, err)

		assert.Equal(t, StateRunning, sup.State())
		assert.Equal(t, 1, mock.StartCalled)
		assert.True(t, healthCalls > 0, "health check should have been called")

		// Cleanup
		_ = sup.Stop(context.Background())
	})

	t.Run("fails if process start fails", func(t *testing.T) {
		mock := NewMockProcess()
		mock.StartErr = errors.New("failed to start")
		healthCheck := func(ctx context.Context) error { return nil }

		sup := NewProcessSupervisor(testConfig(), mock, healthCheck, testLogger())

		err := sup.Start(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start")
		assert.Equal(t, StateStopped, sup.State())
	})

	t.Run("fails if health check times out", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error {
			return errors.New("unhealthy")
		}

		cfg := testConfig()
		cfg.StartupTimeout = 100 * time.Millisecond

		sup := NewProcessSupervisor(cfg, mock, healthCheck, testLogger())

		err := sup.Start(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "health check")
		assert.Equal(t, StateStopped, sup.State())
	})
}

func TestProcessSupervisor_Stop(t *testing.T) {
	t.Run("stops running process", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		sup := NewProcessSupervisor(testConfig(), mock, healthCheck, testLogger())
		_ = sup.Start(context.Background())
		assert.Equal(t, StateRunning, sup.State())

		err := sup.Stop(context.Background())
		require.NoError(t, err)

		assert.Equal(t, StateStopped, sup.State())
		assert.Equal(t, 1, mock.StopCalled)
	})

	t.Run("idempotent - can call multiple times", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		sup := NewProcessSupervisor(testConfig(), mock, healthCheck, testLogger())
		_ = sup.Start(context.Background())

		err1 := sup.Stop(context.Background())
		err2 := sup.Stop(context.Background())

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, StateStopped, sup.State())
	})
}

func TestProcessSupervisor_AutoRestart(t *testing.T) {
	t.Run("restarts after crash", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		sup := NewProcessSupervisor(testConfig(), mock, healthCheck, testLogger())
		_ = sup.Start(context.Background())
		assert.Equal(t, StateRunning, sup.State())
		assert.Equal(t, 1, mock.StartCalled)

		// Simulate crash
		mock.SimulateCrash <- struct{}{}

		// Wait for restart
		eventually(t, 500*time.Millisecond, func() bool {
			return sup.State() == StateRunning && mock.StartCalled == 2
		})

		assert.Equal(t, 1, sup.RestartCount())

		_ = sup.Stop(context.Background())
	})

	t.Run("enters unrecoverable after max restarts", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		cfg := testConfig()
		cfg.MaxRestarts = 2
		cfg.InitialBackoff = 5 * time.Millisecond

		sup := NewProcessSupervisor(cfg, mock, healthCheck, testLogger())
		_ = sup.Start(context.Background())

		// Simulate multiple crashes
		for i := 0; i < 3; i++ {
			mock.SimulateCrash = make(chan struct{})
			close(mock.SimulateCrash)
			time.Sleep(50 * time.Millisecond)
		}

		eventually(t, 500*time.Millisecond, func() bool {
			return sup.State() == StateUnrecoverable
		})

		_ = sup.Stop(context.Background())
	})
}

func TestProcessSupervisor_Restart(t *testing.T) {
	t.Run("manual restart works", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		sup := NewProcessSupervisor(testConfig(), mock, healthCheck, testLogger())
		_ = sup.Start(context.Background())

		err := sup.Restart(context.Background())
		require.NoError(t, err)

		assert.Equal(t, StateRunning, sup.State())
		assert.Equal(t, 2, mock.StartCalled)
		assert.Equal(t, 1, mock.StopCalled)

		_ = sup.Stop(context.Background())
	})

	t.Run("manual restart resets unrecoverable state", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		cfg := testConfig()
		cfg.MaxRestarts = 1
		cfg.InitialBackoff = 5 * time.Millisecond

		sup := NewProcessSupervisor(cfg, mock, healthCheck, testLogger())
		_ = sup.Start(context.Background())

		// Force unrecoverable state
		mock.SimulateCrash = make(chan struct{})
		close(mock.SimulateCrash)
		time.Sleep(30 * time.Millisecond)
		mock.SimulateCrash = make(chan struct{})
		close(mock.SimulateCrash)

		eventually(t, 500*time.Millisecond, func() bool {
			return sup.State() == StateUnrecoverable
		})

		// Manual restart should work
		err := sup.Restart(context.Background())
		require.NoError(t, err)

		assert.Equal(t, StateRunning, sup.State())
		assert.Equal(t, 0, sup.RestartCount()) // Reset

		_ = sup.Stop(context.Background())
	})
}

func TestProcessSupervisor_Subscribe(t *testing.T) {
	t.Run("receives state change events", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		sup := NewProcessSupervisor(testConfig(), mock, healthCheck, testLogger())

		events := sup.Subscribe()

		_ = sup.Start(context.Background())

		// Should receive starting and running events
		var received []State
		timeout := time.After(500 * time.Millisecond)
	collectLoop:
		for {
			select {
			case evt, ok := <-events:
				if !ok {
					break collectLoop
				}
				received = append(received, evt.Current)
				if evt.Current == StateRunning {
					break collectLoop
				}
			case <-timeout:
				break collectLoop
			}
		}

		assert.Contains(t, received, StateStarting)
		assert.Contains(t, received, StateRunning)

		_ = sup.Stop(context.Background())
	})
}

func TestProcessSupervisor_BackoffCalculation(t *testing.T) {
	mock := NewMockProcess()
	healthCheck := func(ctx context.Context) error { return nil }

	cfg := testConfig()
	cfg.InitialBackoff = 100 * time.Millisecond
	cfg.MaxBackoff = 1 * time.Second
	cfg.BackoffMultiplier = 2.0

	sup := NewProcessSupervisor(cfg, mock, healthCheck, testLogger())

	// Test backoff calculation
	assert.Equal(t, 100*time.Millisecond, sup.calculateBackoff(1))
	assert.Equal(t, 200*time.Millisecond, sup.calculateBackoff(2))
	assert.Equal(t, 400*time.Millisecond, sup.calculateBackoff(3))
	assert.Equal(t, 800*time.Millisecond, sup.calculateBackoff(4))
	assert.Equal(t, 1*time.Second, sup.calculateBackoff(5)) // Clamped to max
	assert.Equal(t, 1*time.Second, sup.calculateBackoff(10))
}

func TestState_Helpers(t *testing.T) {
	t.Run("IsTerminal", func(t *testing.T) {
		assert.True(t, StateStopped.IsTerminal())
		assert.True(t, StateUnrecoverable.IsTerminal())
		assert.False(t, StateRunning.IsTerminal())
		assert.False(t, StateStarting.IsTerminal())
		assert.False(t, StateRestarting.IsTerminal())
	})

	t.Run("IsHealthy", func(t *testing.T) {
		assert.True(t, StateRunning.IsHealthy())
		assert.False(t, StateStopped.IsHealthy())
		assert.False(t, StateStarting.IsHealthy())
		assert.False(t, StateRestarting.IsHealthy())
		assert.False(t, StateUnrecoverable.IsHealthy())
	})
}

// eventually polls a condition until it returns true or times out.
func eventually(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
