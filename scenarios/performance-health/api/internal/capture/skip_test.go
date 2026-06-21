package capture

import (
	"context"
	"testing"

	"performance-health/internal/readiness"
)

// [REQ:PH-CAPTURE-003] Capture cleanly SKIPS (never errors) when the scenario
// has no UI surface, mirroring the Lighthouse silent-skip.
func TestOrchestrateSkipsNoUI(t *testing.T) {
	svc := NewService(fakeBAS{}, &fakeBuild{})
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.TierNone)
	if err != nil {
		t.Fatalf("Orchestrate should not error on skip: %v", err)
	}
	if res.Outcome != OutcomeSkipped {
		t.Fatalf("expected SKIPPED, got %#v", res)
	}
}

// [REQ:PH-CAPTURE-003] Capture SKIPS when BAS returns no trace (e.g. no browser
// available) rather than failing.
func TestOrchestrateSkipsNoBrowser(t *testing.T) {
	svc := NewService(fakeBAS{artifacts: Artifacts{}}, &fakeBuild{uiURL: "http://localhost:3000"})
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier0)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeSkipped {
		t.Fatalf("expected SKIPPED when BAS returns no trace, got %#v", res)
	}
}
