package reconcile

import (
	"slices"
	"testing"
)

// TestImplementedClaimTypesMatchesEvaluatorMap is the guard that keeps the
// capability registry honest: the exposed list must be exactly the evaluator map,
// so a claim type can never be reported as checkable without a checker.
func TestImplementedClaimTypesMatchesEvaluatorMap(t *testing.T) {
	got := ImplementedClaimTypes()
	if len(got) != len(claimEvaluators) {
		t.Fatalf("ImplementedClaimTypes returned %d entries, evaluator map has %d", len(got), len(claimEvaluators))
	}
	for _, claimType := range got {
		if claimEvaluator(claimType) == nil {
			t.Errorf("claim type %q is reported as implemented but has no evaluator", claimType)
		}
	}
	if !slices.IsSorted(got) {
		t.Error("ImplementedClaimTypes must be sorted so registry output is stable")
	}
}

// TestElementAbsentHasNoEvaluator documents a known defect rather than asserting
// desired behaviour. `element-absent` is accepted by spec.knownClaimType and has
// no evaluator, so a machine-tier claim using it can never pass. When an evaluator
// is added this test should be deleted, not amended.
func TestElementAbsentHasNoEvaluator(t *testing.T) {
	if claimEvaluator("element-absent") != nil {
		t.Fatal("element-absent now has an evaluator; delete this test and drop the known-defect note in support.go")
	}
}

// TestWiredAxesReflectsCaptureTarget pins the axis list against the only
// dimension CaptureTarget can actually vary. If a new axis field is added to
// CaptureTarget without extending WiredAxes, capability status would silently
// under-report; if WiredAxes grows without the field, it would over-report.
func TestWiredAxesReflectsCaptureTarget(t *testing.T) {
	axes := WiredAxes()
	if len(axes) != 1 || axes[0].Axis != "viewport" {
		t.Fatalf("expected viewport to be the only wired axis, got %+v", axes)
	}
	if len(axes[0].Values) != len(DefaultCaptureProfiles) {
		t.Fatalf("wired viewport values (%d) must match DefaultCaptureProfiles (%d)",
			len(axes[0].Values), len(DefaultCaptureProfiles))
	}
	for _, profile := range DefaultCaptureProfiles {
		if !slices.Contains(axes[0].Values, profile.ID) {
			t.Errorf("capture profile %q missing from wired viewport values", profile.ID)
		}
	}
}

func TestAvailableEvidenceExcludesComputedStyle(t *testing.T) {
	evidence := AvailableEvidence()
	if slices.Contains(evidence, "computed-style") {
		t.Fatal("computed-style is reported as available; if the channel now exists, update support.go's doc comment too")
	}
	if !slices.Contains(evidence, "ax-tree") {
		t.Fatal("ax-tree must be available; it is the load-bearing evidence channel")
	}
}
