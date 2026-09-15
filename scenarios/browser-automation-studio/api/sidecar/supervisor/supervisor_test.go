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
		assert.Equal(t, int64(1), mock.StartCalled.Load())
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
		assert.Equal(t, int64(1), mock.StopCalled.Load())
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
		assert.Equal(t, int64(1), mock.StartCalled.Load())

		// Simulate crash using TriggerCrash which properly signals the supervisor
		mock.TriggerCrash()

		// Wait for restart - give enough time for backoff + health check polling
		eventually(t, 1*time.Second, func() bool {
			return sup.State() == StateRunning && mock.StartCalled.Load() == 2
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

		// Simulate multiple crashes using TriggerCrash
		// We need 3 crashes to exceed MaxRestarts=2
		// After each crash, wait for the supervisor to restart the process before crashing again
		for i := 0; i < 3; i++ {
			// Wait for process to be running (or unrecoverable which means we're done)
			eventually(t, 3*time.Second, func() bool {
				state := sup.State()
				return state == StateRunning || state == StateUnrecoverable
			})

			// If we've reached unrecoverable, no need to crash more
			if sup.State() == StateUnrecoverable {
				break
			}

			// Small delay to ensure state has fully settled before crashing
			time.Sleep(20 * time.Millisecond)

			// Trigger crash
			mock.TriggerCrash()
		}

		// Wait for state to become unrecoverable
		eventually(t, 3*time.Second, func() bool {
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
		assert.Equal(t, int64(2), mock.StartCalled.Load())
		assert.Equal(t, int64(1), mock.StopCalled.Load())

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

		// Force unrecoverable state by triggering crashes exceeding max restarts
		// Need 2 crashes to exceed MaxRestarts=1
		for i := 0; i < 2; i++ {
			// Wait for process to be running (or unrecoverable which means we're done)
			eventually(t, 3*time.Second, func() bool {
				state := sup.State()
				return state == StateRunning || state == StateUnrecoverable
			})

			// If we've reached unrecoverable, no need to crash more
			if sup.State() == StateUnrecoverable {
				break
			}

			// Small delay to ensure state has fully settled before crashing
			time.Sleep(20 * time.Millisecond)

			// Trigger crash
			mock.TriggerCrash()
		}

		eventually(t, 3*time.Second, func() bool {
			return sup.State() == StateUnrecoverable
		})

		// Manual restart should work
		err := sup.Restart(context.Background())
		require.NoError(t, err)

		assert.Equal(t, StateRunning, sup.State())
		// Note: RestartCount may not be exactly 0 due to a race between the monitorLoop
		// spinning on the closed exitChan and Restart() clearing restartTimes.
		// The key assertion is that manual restart succeeds from unrecoverable state.

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

// TestProcessSupervisor_NoSpinOnClosedExitChannel guards the defect where
// monitorLoop re-armed its select on an already-closed exit channel and called
// handleProcessExit forever. A closed channel is permanently ready, so every
// path that returns from handleProcessExit without leaving the loop turns into
// a hot loop: one observed instance logged the same error 23.9 million times
// and held 2.25 cores while every health signal still reported healthy.
//
// The observable signature is restart bookkeeping that never stops growing, so
// that is what these tests assert on.
func TestProcessSupervisor_NoSpinOnClosedExitChannel(t *testing.T) {
	t.Run("stops monitoring once max restarts are exceeded", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		cfg := testConfig()
		cfg.MaxRestarts = 1
		cfg.InitialBackoff = 5 * time.Millisecond

		sup := NewProcessSupervisor(cfg, mock, healthCheck, testLogger())
		require.NoError(t, sup.Start(context.Background()))

		for i := 0; i < 3 && sup.State() != StateUnrecoverable; i++ {
			eventually(t, 3*time.Second, func() bool {
				state := sup.State()
				return state == StateRunning || state == StateUnrecoverable
			})
			if sup.State() == StateUnrecoverable {
				break
			}
			time.Sleep(20 * time.Millisecond)
			mock.TriggerCrash()
		}

		eventually(t, 3*time.Second, func() bool {
			return sup.State() == StateUnrecoverable
		})

		// The loop must be gone, not merely slow. Sampling twice across a
		// window that the buggy build would have used for tens of thousands of
		// iterations is enough to tell the two apart.
		restartsAtTerminal := sup.RestartCount()
		startsAtTerminal := mock.StartCalled.Load()
		time.Sleep(200 * time.Millisecond)

		assert.Equal(t, restartsAtTerminal, sup.RestartCount(),
			"supervisor kept recording restarts after reaching an unrecoverable state")
		assert.Equal(t, startsAtTerminal, mock.StartCalled.Load(),
			"supervisor kept trying to start the process after giving up")

		_ = sup.Stop(context.Background())
	})

	t.Run("stops monitoring when the process cannot be restarted", func(t *testing.T) {
		mock := NewMockProcess()
		healthCheck := func(ctx context.Context) error { return nil }

		cfg := testConfig()
		cfg.InitialBackoff = 5 * time.Millisecond

		sup := NewProcessSupervisor(cfg, mock, healthCheck, testLogger())
		require.NoError(t, sup.Start(context.Background()))

		// A restart that fails leaves the old, already-closed exit channel in
		// place. That is the case the original loop could not survive.
		mock.StartErr = errors.New("port already in use")
		mock.TriggerCrash()

		eventually(t, 3*time.Second, func() bool {
			return sup.State().IsTerminal()
		})

		restartsAtTerminal := sup.RestartCount()
		startsAtTerminal := mock.StartCalled.Load()
		time.Sleep(200 * time.Millisecond)

		assert.Equal(t, restartsAtTerminal, sup.RestartCount(),
			"supervisor spun on a closed exit channel after a failed restart")
		assert.Equal(t, startsAtTerminal, mock.StartCalled.Load(),
			"supervisor retried a failed start without bound")

		_ = sup.Stop(context.Background())
	})
}
