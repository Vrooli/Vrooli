package registry

import (
	"context"
	"testing"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"

	"portal/internal/testutil/mocks"
)

func TestServiceStatusProbesAndEvaluatesMode(t *testing.T) {
	t.Parallel()

	clk := mocks.NewFakeClock(time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC))
	service := NewService(Config{
		Clock:      clk,
		WindowSize: 4,
		Probes: map[IntegrationID]Probe{
			IntegrationSearchHub:    ProbeFunc(func(context.Context) ProbeResult { return ProbeResult{OK: true} }),
			IntegrationOpenRouter:   ProbeFunc(func(context.Context) ProbeResult { return ProbeResult{OK: false, Reason: "missing key"} }),
			IntegrationAgentManager: ProbeFunc(func(context.Context) ProbeResult { return ProbeResult{OK: true} }),
			IntegrationPromptMgr:    ProbeFunc(func(context.Context) ProbeResult { return ProbeResult{OK: true} }),
		},
	})

	first, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if first.ActiveMode != sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF {
		t.Fatalf("first sample should wait for recovery window, got %v", first.ActiveMode)
	}

	second, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if second.ActiveMode != sharedv1.BehaviorMode_BEHAVIOR_MODE_PASSIVE {
		t.Fatalf("expected passive after recovery window, got %v", second.ActiveMode)
	}
	openrouter := findStatus(second.Integrations, IntegrationOpenRouter)
	if openrouter.State != sharedv1.IntegrationState_INTEGRATION_STATE_UNAVAILABLE {
		t.Fatalf("expected openrouter unavailable, got %v", openrouter.State)
	}
}

func TestServiceSetOverrideForcesMode(t *testing.T) {
	t.Parallel()

	service := NewService(Config{
		Probes: map[IntegrationID]Probe{
			IntegrationSearchHub:    ProbeFunc(func(context.Context) ProbeResult { return ProbeResult{OK: false, Reason: "down"} }),
			IntegrationOpenRouter:   ProbeFunc(func(context.Context) ProbeResult { return ProbeResult{OK: true} }),
			IntegrationAgentManager: ProbeFunc(func(context.Context) ProbeResult { return ProbeResult{OK: true} }),
			IntegrationPromptMgr:    ProbeFunc(func(context.Context) ProbeResult { return ProbeResult{OK: true} }),
		},
	})

	status, err := service.SetOverride(context.Background(), OverrideForcePassive)
	if err != nil {
		t.Fatalf("set override: %v", err)
	}
	if status.ActiveMode != sharedv1.BehaviorMode_BEHAVIOR_MODE_PASSIVE {
		t.Fatalf("expected forced passive, got %v", status.ActiveMode)
	}
	if status.Override != OverrideForcePassive {
		t.Fatalf("expected force-passive override, got %q", status.Override)
	}
}
