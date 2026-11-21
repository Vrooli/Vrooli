package androidadb

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"device-control/strategy"
)

// Unlock performs the narrow Android PIN/numeric-passcode procedure. It does
// not use `input text`: every digit is translated to a non-secret Android key
// code before it reaches adb. The only success proof is a fresh keyguard probe
// after submission.
func (a *Adapter) Unlock(ctx context.Context, request strategy.UnlockRequest) (result strategy.UnlockResult, err error) {
	defer func() {
		for i := range request.Secret {
			request.Secret[i] = 0
		}
	}()

	if request.Method != "pin" && request.Method != "numeric_passcode" {
		if request.Method == "biometric" || request.Method == "human_gated" {
			return strategy.UnlockResult{Outcome: "human_required", Detail: "this authentication method requires an operator"}, nil
		}
		return strategy.UnlockResult{Outcome: "unsupported_method", Detail: "android adb supports only numeric PIN methods"}, nil
	}
	if len(request.Secret) == 0 {
		return strategy.UnlockResult{Outcome: "credential_unconfigured", Detail: "the resolved credential was empty"}, nil
	}
	maxAttempts := request.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if maxAttempts > 1 {
		// The service may permit an explicit policy, but an adapter never loops
		// without a fresh live state check between attempts.
		maxAttempts = 1
	}
	if request.AttemptLimit <= 0 {
		request.AttemptLimit = 15 * time.Second
	}
	if request.Settle <= 0 {
		request.Settle = 750 * time.Millisecond
	}

	state, probeErr := a.lockState(ctx)
	if probeErr != nil {
		return strategy.UnlockResult{Outcome: unlockErrorOutcome(probeErr, "transport_error"), Detail: "unable to read Android keyguard state"}, probeErr
	}
	if state == "unlocked" {
		return strategy.UnlockResult{Outcome: "already_unlocked"}, nil
	}
	if state != "locked" {
		return strategy.UnlockResult{Outcome: "unknown_device_state", Detail: "Android keyguard state was not classifiable"}, nil
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, request.AttemptLimit)
		if _, runErr := a.runner.Run(attemptCtx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_WAKEUP")...); runErr != nil {
			cancel()
			return strategy.UnlockResult{Outcome: unlockErrorOutcome(runErr, "transport_error"), Attempts: attempt, Detail: "unable to wake the Android display"}, runErr
		}
		for _, digit := range request.Secret {
			keyCode, ok := digitKeyCode(digit)
			if !ok {
				cancel()
				return strategy.UnlockResult{Outcome: "invalid_credential", Attempts: attempt, Detail: "numeric PIN contains a non-digit"}, nil
			}
			if _, runErr := a.runner.Run(attemptCtx, "adb", a.args("shell", "input", "keyevent", keyCode)...); runErr != nil {
				cancel()
				return strategy.UnlockResult{Outcome: unlockErrorOutcome(runErr, "transport_error"), Attempts: attempt, Detail: "Android rejected a key event"}, runErr
			}
		}
		if _, runErr := a.runner.Run(attemptCtx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_ENTER")...); runErr != nil {
			cancel()
			return strategy.UnlockResult{Outcome: unlockErrorOutcome(runErr, "transport_error"), Attempts: attempt, Detail: "Android rejected PIN submission"}, runErr
		}
		if settleErr := waitForUnlockSettle(attemptCtx, request.Settle); settleErr != nil {
			cancel()
			return strategy.UnlockResult{Outcome: unlockErrorOutcome(settleErr, "timeout"), Attempts: attempt, Detail: "unlock settle window expired"}, nil
		}
		state, probeErr = a.lockState(attemptCtx)
		cancel()
		if probeErr != nil {
			return strategy.UnlockResult{Outcome: unlockErrorOutcome(probeErr, "transport_error"), Attempts: attempt, Detail: "unable to verify Android keyguard state"}, probeErr
		}
		if state == "unlocked" {
			return strategy.UnlockResult{Outcome: "unlocked", Attempts: attempt}, nil
		}
		if state != "locked" {
			return strategy.UnlockResult{Outcome: "unknown_device_state", Attempts: attempt, Detail: "Android keyguard state became unclassifiable"}, nil
		}
	}
	return strategy.UnlockResult{Outcome: "wrong_credential", Attempts: maxAttempts, Detail: "the postcondition remained locked; no automatic retry was attempted"}, nil
}

func (a *Adapter) lockState(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "unknown", err
	}
	state := strategy.DeviceState{Unavailable: map[string]string{}}
	lock := a.readLockState(ctx, &state)
	if lock == "" {
		if err := ctx.Err(); err != nil {
			return "unknown", err
		}
		return "unknown", errors.New("Android keyguard state unavailable")
	}
	return lock, nil
}

func unlockErrorOutcome(err error, fallback string) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	default:
		return fallback
	}
}

// readLockState prefers the broad window dump but falls back to the smaller
// policy dump. Samsung builds can truncate or reject the broad dump over a
// wireless ADB transport while still exposing the keyguard policy state.
func (a *Adapter) readLockState(ctx context.Context, state *strategy.DeviceState) string {
	for _, command := range [][]string{
		{"shell", "dumpsys", "window"},
		{"shell", "dumpsys", "window", "policy"},
		{"shell", "dumpsys", "keyguard"},
		{"shell", "dumpsys", "window", "displays"},
		{"shell", "cmd", "statusbar", "is-keyguard-showing"},
	} {
		out, err := a.runner.Run(ctx, "adb", a.args(command[:]...)...)
		if err == nil {
			if state != nil {
				if focused := focusedPackage(string(out)); focused != "" && state.ForegroundPackage == "" {
					state.ForegroundPackage = focused
				}
			}
			if lock := lockState(string(out)); lock != "" {
				return lock
			}
			if lock := exactBooleanLockState(string(out)); lock != "" {
				return lock
			}
		}
	}
	if state != nil {
		state.Unavailable["lock_state"] = "Android keyguard probe failed"
	}
	return ""
}

func digitKeyCode(digit byte) (string, bool) {
	if digit < '0' || digit > '9' {
		return "", false
	}
	return "KEYCODE_" + string(digit), true
}

func waitForUnlockSettle(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *Adapter) RestoreState(ctx context.Context, state strategy.DeviceState) error {
	if state.Orientation != "" {
		if len(state.RotationSettings) != 2 {
			return fmt.Errorf("cannot restore rotation without exact raw settings")
		}
		for _, name := range []string{"accelerometer_rotation", "user_rotation"} {
			value, known := state.RotationSettings[name]
			if !known || (value != "null" && value != "0" && value != "1" && value != "2" && value != "3") {
				return fmt.Errorf("cannot restore unknown rotation setting %s", name)
			}
			args := []string{"shell", "settings", "put", "system", name, value}
			if value == "null" {
				args = []string{"shell", "settings", "delete", "system", name}
			}
			if _, err := a.runner.Run(ctx, "adb", a.args(args...)...); err != nil {
				return fmt.Errorf("restore %s: %w", name, err)
			}
		}
	}
	if state.LockState == "locked" {
		current, err := a.lockState(ctx)
		if err != nil {
			return fmt.Errorf("verify keyguard before restore: %w", err)
		}
		if current == "unlocked" {
			if _, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_POWER")...); err != nil {
				return fmt.Errorf("restore locked keyguard: %w", err)
			}
			current, err = a.waitForLockState(ctx, "locked", 5*time.Second)
			if err != nil {
				return fmt.Errorf("verify restored keyguard: %w", err)
			}
		}
		if current != "locked" {
			return fmt.Errorf("restored keyguard state is %q, want locked", current)
		}
	}
	return nil
}

// waitForLockState accounts for the asynchronous transition after a power
// key. Android can report the old keyguard state for several probes while the
// display and policy services settle. The deadline keeps teardown fail-closed
// without turning a transient vendor delay into a false run failure.
func (a *Adapter) waitForLockState(ctx context.Context, want string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		return "unknown", errors.New("keyguard confirmation timeout must be positive")
	}
	deadline := time.Now().Add(timeout)
	var last string
	var lastErr error
	for {
		current, err := a.lockState(ctx)
		if err == nil {
			last = current
			lastErr = nil
			if current == want {
				return current, nil
			}
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			if lastErr != nil {
				return last, fmt.Errorf("keyguard remained unconfirmed: %w", lastErr)
			}
			return last, fmt.Errorf("restored keyguard state is %q, want %q", last, want)
		}
		timer := time.NewTimer(150 * time.Millisecond)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return last, ctx.Err()
		}
	}
}

func lockState(output string) string {
	// Android's window/keyguard dump format is not stable across vendors and
	// releases. Samsung Android 13 commonly reports the delegate state as
	// `showing=true` or `mShowingState=SHOWING`, while older builds expose one
	// of the explicit m* fields. Keep this parser deliberately scoped to
	// keyguard/window terms so unrelated boolean fields cannot unlock a flow.
	if regexp.MustCompile(`(?i)(mShowingLockscreen|isStatusBarKeyguard|mKeyguardShowing|isKeyguardShowing)\s*[:=]\s*true`).MatchString(output) {
		return "locked"
	}
	if regexp.MustCompile(`(?i)(mShowingLockscreen|isStatusBarKeyguard|mKeyguardShowing|isKeyguardShowing)\s*[:=]\s*false`).MatchString(output) {
		return "unlocked"
	}
	if regexp.MustCompile(`(?i)\bmShowingState\s*[:=]\s*(?:SHOWING|LOCKED)\b`).MatchString(output) {
		return "locked"
	}
	if regexp.MustCompile(`(?i)\bmShowingState\s*[:=]\s*(?:NOT_SHOWING|UNLOCKED)\b`).MatchString(output) {
		return "unlocked"
	}
	if regexp.MustCompile(`(?i)\b(?:keyguard\s+)?showing\s*[:=]\s*true`).MatchString(output) {
		return "locked"
	}
	if regexp.MustCompile(`(?i)\b(?:keyguard\s+)?showing\s*[:=]\s*false`).MatchString(output) {
		return "unlocked"
	}
	return ""
}

func exactBooleanLockState(output string) string {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "true":
		return "locked"
	case "false":
		return "unlocked"
	default:
		return ""
	}
}
