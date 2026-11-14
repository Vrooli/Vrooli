package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"device-control/strategy"
)

// Screen transitions can be reported shortly after the key is accepted. Retry
// only the observation, never the input effect, within a small bounded window.
func (s *Service) waitForScreenPostcondition(ctx context.Context, strat strategy.Strategy, condition Condition, previousFrame, lastFrame []byte, lastErr error) error {
	waitCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	for {
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-waitCtx.Done():
			timer.Stop()
			return lastErr
		case <-timer.C:
		}
		if err := s.evaluateFlowCondition(waitCtx, strat, condition, previousFrame, lastFrame); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
}

func (s *Service) evaluateFlowCondition(ctx context.Context, strat strategy.Strategy, condition Condition, previousFrame, lastFrame []byte) error {
	switch strings.ToLower(strings.TrimSpace(condition.Kind)) {
	case "device-unlocked":
		reader, ok := strat.(strategy.StateReader)
		if !ok {
			return fmt.Errorf("strategy does not expose state")
		}
		state, err := reader.ReadState(ctx)
		if err != nil {
			return err
		}
		if state.LockState != "unlocked" {
			return fmt.Errorf("lock state is %q", state.LockState)
		}
		return nil
	case "screen-on", "screen-off":
		reader, ok := strat.(strategy.StateReader)
		if !ok {
			return fmt.Errorf("strategy does not expose state")
		}
		state, err := reader.ReadState(ctx)
		if err != nil {
			return err
		}
		want := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(condition.Kind)), "screen-")
		if state.ScreenState != want {
			return fmt.Errorf("screen state is %q; %s was not verified", state.ScreenState, want)
		}
		return nil
	case "capability":
		declaration, err := strat.Describe(ctx)
		if err != nil {
			return err
		}
		capability, ok := declaration.Capabilities[condition.Target]
		if !ok || capability.Status != strategy.StatusAvailable {
			return fmt.Errorf("capability %q is unavailable", condition.Target)
		}
		return nil
	case "property-equals":
		property, ok := strat.(strategy.PropertyActuator)
		if !ok {
			return fmt.Errorf("strategy does not expose properties")
		}
		actual, err := property.GetProperty(ctx, condition.Target)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(actual, condition.Expected) {
			return fmt.Errorf("property %q is %v, expected %v", condition.Target, actual, condition.Expected)
		}
		return nil
	case "frame-different":
		if len(previousFrame) == 0 || len(lastFrame) == 0 {
			return fmt.Errorf("frame-difference requires two observations")
		}
		if bytes.Equal(previousFrame, lastFrame) {
			return fmt.Errorf("observations are identical")
		}
		return nil
	default:
		return fmt.Errorf("unsupported condition kind %q", condition.Kind)
	}
}

func retryableFlowStep(kind string) bool {
	switch kind {
	case "wait", "property-get", "property-assert", "sensor-read", "semantic-assert", "assert-foreground", "wireless-debugging-assert":
		// These operations are observational or locally repeatable. Native
		// effects remain one-shot unless their strategy supplies a stronger
		// idempotency contract.
		return true
	default:
		return false
	}
}

func classifyFlowFailure(err error) string {
	if err == nil {
		return ""
	}
	var availability *strategy.AvailabilityError
	if errors.As(err, &availability) {
		return "transport"
	}
	var unsupported *strategy.UnsupportedCapabilityError
	if errors.As(err, &unsupported) {
		return "capability"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "precondition"), strings.Contains(message, "postcondition"), strings.Contains(message, "lock state"), strings.Contains(message, "screen"):
		return "app-state"
	case strings.Contains(message, "coordinate"), strings.Contains(message, "semantic"), strings.Contains(message, "target"):
		return "target"
	case strings.Contains(message, "lease"), strings.Contains(message, "session"), strings.Contains(message, "admission"):
		return "host-session"
	default:
		return "execution"
	}
}

func requiresVisibleSurface(kind string) bool {
	switch kind {
	case "observe", "screenshot", "semantic-target", "semantic-assert", "assert-foreground", "recording-start", "screenrecord":
		return true
	default:
		return false
	}
}
