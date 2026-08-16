package androidadb

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

type unlockRunner struct {
	lockedAfterAttempt bool
	lockProbes         int
	calls              []string
}

type restorationRunner struct {
	locked          bool
	lockProbeCount  int
	lockAfterProbes int
	calls           []string
}

type failingUnlockRunner struct {
	err error
}

func (r failingUnlockRunner) Run(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return nil, ctx.Err()
}

func (r *restorationRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, call)
	if strings.Contains(call, "KEYCODE_POWER") {
		r.locked = true
	}
	if strings.Contains(call, "dumpsys window") {
		r.lockProbeCount++
		locked := r.locked && r.lockProbeCount > r.lockAfterProbes
		return []byte("mShowingLockscreen=" + strconv.FormatBool(locked)), nil
	}
	return nil, nil
}

func (r *unlockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, call)
	if strings.HasSuffix(call, "shell dumpsys window") {
		r.lockProbes++
		if r.lockProbes == 1 || r.lockedAfterAttempt {
			return []byte("mShowingLockscreen=true"), nil
		}
		return []byte("mShowingLockscreen=false"), nil
	}
	return nil, nil
}

func TestUnlockUsesNonSecretKeyEventsAndFreshVerification(t *testing.T) {
	runner := &unlockRunner{}
	adapter := NewWithRunner(runner, "serial-1")
	secret := []byte{'1', '2', '3'}
	original := string(secret)
	result, err := adapter.Unlock(context.Background(), strategy.UnlockRequest{Method: "pin", Secret: secret, MaxAttempts: 1, AttemptLimit: time.Second, Settle: time.Millisecond})
	require.NoError(t, err)
	require.Equal(t, "unlocked", result.Outcome)
	require.Equal(t, 1, result.Attempts)
	for _, call := range runner.calls {
		require.NotContains(t, call, original)
	}
	require.Contains(t, strings.Join(runner.calls, "\n"), "KEYCODE_1")
	require.Contains(t, strings.Join(runner.calls, "\n"), "KEYCODE_ENTER")
	require.Equal(t, []byte{0, 0, 0}, secret)
}

func TestUnlockStopsAfterOneWrongCredentialAttempt(t *testing.T) {
	runner := &unlockRunner{lockedAfterAttempt: true}
	adapter := NewWithRunner(runner, "serial-1")
	result, err := adapter.Unlock(context.Background(), strategy.UnlockRequest{Method: "numeric_passcode", Secret: []byte{'1', '2'}, MaxAttempts: 3, AttemptLimit: time.Second, Settle: time.Millisecond})
	require.NoError(t, err)
	require.Equal(t, "wrong_credential", result.Outcome)
	require.Equal(t, 1, result.Attempts)
}

func TestUnlockBoundsSettleTimeoutWithoutRetry(t *testing.T) {
	runner := &unlockRunner{}
	adapter := NewWithRunner(runner, "serial-1")
	result, err := adapter.Unlock(context.Background(), strategy.UnlockRequest{
		Method: "pin", Secret: []byte{'1', '2'}, MaxAttempts: 3,
		AttemptLimit: time.Millisecond, Settle: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.Equal(t, "timeout", result.Outcome)
	require.Equal(t, 1, result.Attempts)
}

func TestUnlockMapsCancellationAndTransportFailures(t *testing.T) {
	t.Run("cancelled before probe", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		secret := []byte{'1'}
		result, err := NewWithRunner(failingUnlockRunner{}, "serial-1").Unlock(ctx, strategy.UnlockRequest{Method: "pin", Secret: secret})
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, "cancelled", result.Outcome)
		require.Equal(t, []byte{0}, secret)
	})
	t.Run("transport", func(t *testing.T) {
		result, err := NewWithRunner(failingUnlockRunner{err: errors.New("transport unavailable")}, "serial-1").Unlock(context.Background(), strategy.UnlockRequest{Method: "pin", Secret: []byte{'1'}})
		require.Error(t, err)
		require.Equal(t, "transport_error", result.Outcome)
	})
}

func TestUnlockReportsHumanAndUnsupportedMethodsWithoutActuation(t *testing.T) {
	runner := &unlockRunner{}
	adapter := NewWithRunner(runner, "serial-1")
	for _, method := range []string{"biometric", "pattern", "password"} {
		result, err := adapter.Unlock(context.Background(), strategy.UnlockRequest{Method: method, Secret: []byte("not-used")})
		require.NoError(t, err)
		if method == "biometric" {
			require.Equal(t, "human_required", result.Outcome)
		} else {
			require.Equal(t, "unsupported_method", result.Outcome)
		}
	}
	require.Empty(t, runner.calls)
}

func TestRestoreStateReLocksDeviceThatStartedLocked(t *testing.T) {
	runner := &restorationRunner{}
	adapter := NewWithRunner(runner, "serial-1")
	err := adapter.RestoreState(context.Background(), strategy.DeviceState{LockState: "locked", Orientation: "portrait"})
	require.NoError(t, err)
	require.True(t, runner.locked)
	require.Contains(t, strings.Join(runner.calls, "\n"), "KEYCODE_POWER")
}

func TestRestoreStateWaitsForAsynchronousKeyguardTransition(t *testing.T) {
	runner := &restorationRunner{lockAfterProbes: 2}
	adapter := NewWithRunner(runner, "serial-1")

	err := adapter.RestoreState(context.Background(), strategy.DeviceState{LockState: "locked", Orientation: "portrait"})

	require.NoError(t, err)
	require.GreaterOrEqual(t, runner.lockProbeCount, 3)
	require.Contains(t, strings.Join(runner.calls, "\n"), "KEYCODE_POWER")
}
