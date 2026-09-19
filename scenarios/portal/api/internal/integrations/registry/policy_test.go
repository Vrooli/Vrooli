package registry

import (
	"testing"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

func TestDecideModeOverridePrecedence(t *testing.T) {
	t.Parallel()

	search := IntegrationStatus{
		State: sharedv1.IntegrationState_INTEGRATION_STATE_UNAVAILABLE,
		Stats: RollingStats{SampleCount: 4, ConsecutiveOK: 0},
	}
	decision := DecideMode(PolicyInput{Override: OverrideForcePassive, Search: search})
	if decision.Mode != sharedv1.BehaviorMode_BEHAVIOR_MODE_PASSIVE {
		t.Fatalf("expected forced passive, got %v", decision.Mode)
	}

	decision = DecideMode(PolicyInput{Override: OverrideForceOff, Search: search})
	if decision.Mode != sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF {
		t.Fatalf("expected forced off, got %v", decision.Mode)
	}
}

func TestDecideModeHysteresisRecovery(t *testing.T) {
	t.Parallel()

	search := IntegrationStatus{
		State: sharedv1.IntegrationState_INTEGRATION_STATE_AVAILABLE,
		Stats: RollingStats{
			SampleCount:   1,
			ConsecutiveOK: 1,
			LatencyP95:    time.Second,
		},
	}
	decision := DecideMode(PolicyInput{
		PreviousMode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF,
		Override:     OverrideAuto,
		Search:       search,
	})
	if decision.Mode != sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF {
		t.Fatalf("expected recovery window to hold off, got %v", decision.Mode)
	}

	search.Stats.SampleCount = 2
	search.Stats.ConsecutiveOK = 2
	decision = DecideMode(PolicyInput{
		PreviousMode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF,
		Override:     OverrideAuto,
		Search:       search,
	})
	if decision.Mode != sharedv1.BehaviorMode_BEHAVIOR_MODE_PASSIVE {
		t.Fatalf("expected recovered passive, got %v", decision.Mode)
	}
}

func TestDecideModeDropsToOffOnLatencyAndNeverFull(t *testing.T) {
	t.Parallel()

	search := IntegrationStatus{
		State: sharedv1.IntegrationState_INTEGRATION_STATE_AVAILABLE,
		Stats: RollingStats{
			SampleCount:   8,
			ConsecutiveOK: 8,
			LatencyP95:    20 * time.Second,
		},
	}
	decision := DecideMode(PolicyInput{
		PreviousMode: sharedv1.BehaviorMode_BEHAVIOR_MODE_PASSIVE,
		Override:     OverrideAuto,
		Search:       search,
	})
	if decision.Mode != sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF {
		t.Fatalf("expected latency off, got %v", decision.Mode)
	}
	if decision.Mode == sharedv1.BehaviorMode_BEHAVIOR_MODE_FULL {
		t.Fatal("v0 policy must never emit FULL")
	}
}
