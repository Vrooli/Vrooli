package execution

import (
	"context"
	"testing"
	"time"
)

func TestDiscoveryFindingsFromReview(t *testing.T) {
	if got := discoveryFindingsFromReview(nil); got != nil {
		t.Errorf("nil result should yield no findings, got %v", got)
	}
	result := &ReviewResult{
		Dimensions: []ReviewDimension{
			{Name: "tests", Status: "red", Details: "3 failing"},
			{Name: "structure", Status: "green"},
			{Name: "rules", Status: "yellow", Details: "2 warnings"},
			{Name: "visuals", Status: "skipped"},
		},
	}
	findings := discoveryFindingsFromReview(result)
	if len(findings) != 2 {
		t.Fatalf("expected 2 actionable findings (red+yellow), got %d: %v", len(findings), findings)
	}
	if findings[0].Dimension != "tests" || findings[0].Status != "red" {
		t.Errorf("unexpected first finding: %+v", findings[0])
	}
	if findings[1].Dimension != "rules" || findings[1].Status != "yellow" {
		t.Errorf("unexpected second finding: %+v", findings[1])
	}
}

func TestDiscoveryMarkerFreshness(t *testing.T) {
	svc := &Service{dataRoot: t.TempDir()}
	const scenario = "demo"

	if svc.discoveryMarkerFresh(scenario) {
		t.Errorf("missing marker should be stale")
	}
	if err := svc.writeDiscoveryMarker(scenario); err != nil {
		t.Fatalf("writeDiscoveryMarker: %v", err)
	}
	if !svc.discoveryMarkerFresh(scenario) {
		t.Errorf("just-written marker should be fresh")
	}
}

// stubFiler records FileRemediation invocations.
type stubFiler struct {
	calls chan filerCall
}

type filerCall struct {
	scenario string
	findings []DiscoveryFinding
}

func (f *stubFiler) FileRemediation(_ context.Context, scenario string, findings []DiscoveryFinding) (int, error) {
	f.calls <- filerCall{scenario: scenario, findings: findings}
	return len(findings), nil
}

func TestMaybeTriggerDiscovery(t *testing.T) {
	filer := &stubFiler{calls: make(chan filerCall, 1)}
	svc := &Service{
		dataRoot: t.TempDir(),
		reviewClient: &stubReviewClient{
			triggerJobID: "job-1",
			pollDone:     true,
			pollResult: &ReviewResult{
				Dimensions: []ReviewDimension{{Name: "tests", Status: "red", Details: "boom"}},
			},
		},
	}
	svc.SetRemediationFiler(filer)

	// "alpha" has no open items → discovery should fire. "beta" has an open
	// remediation item → discovery must skip it.
	svc.maybeTriggerDiscovery(
		[]string{"alpha", "beta"},
		[]openRemediationItem{{kind: "fix", name: "beta-bug", scenarios: []string{"beta"}}},
	)

	select {
	case call := <-filer.calls:
		if call.scenario != "alpha" {
			t.Errorf("discovery fired for %q, want alpha", call.scenario)
		}
		if len(call.findings) != 1 || call.findings[0].Dimension != "tests" {
			t.Errorf("unexpected findings: %+v", call.findings)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected FileRemediation to be called for alpha")
	}

	// A second call is suppressed by the fresh marker for alpha.
	svc.maybeTriggerDiscovery([]string{"alpha"}, nil)
	select {
	case call := <-filer.calls:
		t.Fatalf("second discovery should be suppressed by fresh marker, got call for %q", call.scenario)
	case <-time.After(500 * time.Millisecond):
		// expected: no call
	}
}

func TestMaybeTriggerDiscoveryNoFilerNoop(t *testing.T) {
	svc := &Service{dataRoot: t.TempDir(), reviewClient: &stubReviewClient{}}
	// No filer set → must not panic and must not write a marker.
	svc.maybeTriggerDiscovery([]string{"alpha"}, nil)
	if svc.discoveryMarkerFresh("alpha") {
		t.Errorf("no-op discovery should not write a marker")
	}
}
