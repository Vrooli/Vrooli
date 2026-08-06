package reconcile

import (
	"path/filepath"
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

func TestElementAbsentHasEvaluator(t *testing.T) {
	if claimEvaluator("element-absent") == nil {
		t.Fatal("element-absent must have a deterministic evaluator")
	}
}

// TestWiredAxesReflectsCaptureTarget pins the axis list against the only
// dimension CaptureTarget can actually vary. If a new axis field is added to
// CaptureTarget without extending WiredAxes, capability status would silently
// under-report; if WiredAxes grows without the field, it would over-report.
func TestWiredAxesReflectsCaptureTarget(t *testing.T) {
	axes := WiredAxes()
	if len(axes) != 5 {
		t.Fatalf("expected viewport plus four capture axes, got %+v", axes)
	}
	var viewport AxisSupport
	for _, axis := range axes {
		if axis.Axis == "viewport" {
			viewport = axis
			break
		}
	}
	uniqueViewports := map[string]bool{}
	for _, profile := range DefaultCaptureProfiles {
		uniqueViewports[profile.ID] = true
	}
	if len(viewport.Values) != len(uniqueViewports) {
		t.Fatalf("wired viewport values (%d) must match unique default viewports (%d)",
			len(viewport.Values), len(uniqueViewports))
	}
	for _, profile := range DefaultCaptureProfiles {
		if !slices.Contains(viewport.Values, profile.ID) {
			t.Errorf("capture profile %q missing from wired viewport values", profile.ID)
		}
	}
	for _, wanted := range []string{"color-scheme", "interaction-state", "locale", "motion-preference", "viewport"} {
		found := false
		for _, axis := range axes {
			if axis.Axis == wanted {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing wired axis %q", wanted)
		}
	}
}

func TestWiredAxesOnlyReportsTransmittedCaptureFields(t *testing.T) {
	axes := WiredAxesFromProfiles([]CaptureProfile{{ID: "desktop", Width: 1280, Height: 720}})
	if len(axes) != 1 || axes[0].Axis != "viewport" || !slices.Contains(axes[0].Values, "desktop") {
		t.Fatalf("axes = %+v, want only the transmitted viewport axis", axes)
	}
}

func TestCaptureProfilesFromAxesIncludesDesktopDarkWithinBudget(t *testing.T) {
	profiles, err := CaptureProfilesFromAxes(filepath.Join(repoRootForSupportTest(t), "scenarios", "experience-manager", "capabilities", "axes.json"), 12)
	if err != nil {
		t.Fatalf("CaptureProfilesFromAxes: %v", err)
	}
	if len(profiles) > 12 {
		t.Fatalf("profiles = %d, exceeds capture budget", len(profiles))
	}
	foundDesktopDark := false
	for _, profile := range profiles {
		if profile.ID == "desktop" && profile.ColorScheme == "dark" {
			foundDesktopDark = true
		}
	}
	if !foundDesktopDark {
		t.Fatalf("profiles = %+v, want a desktop-dark capture", profiles)
	}
}

func repoRootForSupportTest(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join("..", "..", "..", "..", ".."))
}

func TestAvailableEvidenceIncludesComputedStyle(t *testing.T) {
	evidence := AvailableEvidence()
	if !slices.Contains(evidence, "computed-style") {
		t.Fatal("computed-style must be reported once BAS captures the declared property map")
	}
	if !slices.Contains(evidence, "ax-tree") {
		t.Fatal("ax-tree must be available; it is the load-bearing evidence channel")
	}
}
