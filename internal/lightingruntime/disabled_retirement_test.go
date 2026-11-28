package lightingruntime

import (
	"context"
	"errors"
	"testing"
)

func TestExplicitDisableRetiresRetainedBackendWhileDisabled(t *testing.T) {
	_, c, owners := newLightingFixture(t)
	failure := errors.New("retained native cleanup incomplete")
	owners.stopErr = failure
	owners.quiesce = func(context.Context) error {
		intent, err := c.Current()
		if err != nil || intent.Phase != Disabling {
			t.Fatalf("explicit cleanup lacks durable start fence: %+v %v", intent, err)
		}
		return nil
	}
	// A normal owner command must reach retained backend cleanup even if a
	// prior cleanup already recorded lighting disabled before that debt arose.
	if err := c.Disable(t.Context()); !errors.Is(err, failure) {
		t.Fatalf("explicit disabled cleanup was skipped: %v", err)
	}
	expectLightingCalls(t, owners, "quiesce", "stop")
	intent, err := c.Current()
	if err != nil || intent.Phase != Disabling {
		t.Fatalf("failed cleanup was claimed disabled: %+v %v", intent, err)
	}
	owners.stopErr = nil
	if err := c.Disable(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := c.Disable(t.Context()); err != nil {
		t.Fatal(err)
	}
	expectLightingCalls(t, owners, "quiesce", "stop", "quiesce", "stop", "quiesce", "stop")
	intent, err = c.Current()
	if err != nil || intent.Phase != Disabled {
		t.Fatalf("completed explicit cleanup lacks disabled intent: %+v %v", intent, err)
	}
	// Status/reconciliation must not turn this explicit-maintenance behavior
	// into an autonomous effect on an opted-out machine.
	owners.calls = nil
	if err := c.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	expectLightingCalls(t, owners)
}
