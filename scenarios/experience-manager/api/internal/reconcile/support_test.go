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

// TestWiredAxesHaveDistinctCaptureRequests is the regression guard for phantom
// axes. It deliberately derives the axes from WiredAxes rather than repeating
// the registry list: every advertised value must change the capture request
// that BAS receives. Native browser evidence is covered by the Playwright
// driver tests and scenario capture gates.
func TestWiredAxesHaveDistinctCaptureRequests(t *testing.T) {
	profiles := DefaultCaptureProfiles
	for _, axis := range WiredAxes() {
		baseline, ok := profileForAxisValue(profiles, axis.Axis, axis.Values[0])
		if !ok {
			t.Fatalf("axis %q has no baseline profile for %q", axis.Axis, axis.Values[0])
		}
		for _, value := range axis.Values[1:] {
			variant, ok := profileForAxisValue(profiles, axis.Axis, value)
			if !ok {
				t.Errorf("axis %q value %q has no capture profile", axis.Axis, value)
				continue
			}
			if captureRequestFingerprint(baseline) == captureRequestFingerprint(variant) {
				t.Errorf("axis %q value %q produced the same capture request as %q", axis.Axis, value, axis.Values[0])
			}
			t.Logf("axis %s: %s -> %s changes the BAS capture request", axis.Axis, axis.Values[0], value)
		}
	}
}

func profileForAxisValue(profiles []CaptureProfile, axis, value string) (CaptureProfile, bool) {
	for _, profile := range profiles {
		matches := map[string]string{
			"viewport":          profile.ID,
			"color-scheme":      profile.ColorScheme,
			"locale":            profile.Locale,
			"motion-preference": profile.MotionPreference,
			"interaction-state": profile.InteractionState,
		}[axis]
		if matches == value {
			return profile, true
		}
	}
	return CaptureProfile{}, false
}

// This models the request identity BAS must preserve: two requests that
// differ in a capture axis cannot collapse to one evidence row. The Playwright
// driver tests exercise the native pseudo-state application; this package test
// guards the reconciler's matrix/evidence join.
func captureRequestFingerprint(profile CaptureProfile) string {
	return profile.ID + "|" + profile.ColorScheme + "|" + profile.Locale + "|" + profile.MotionPreference + "|" + profile.InteractionState
}

func TestCaptureProfilesFromAxesIncludesDesktopDarkWithinBudget(t *testing.T) {
	profiles, err := CaptureProfilesFromAxes(filepath.Join(repoRootForSupportTest(t), "scenarios", "experience-manager", "capabilities", "axes.json"), 16)
	if err != nil {
		t.Fatalf("CaptureProfilesFromAxes: %v", err)
	}
	if len(profiles) > 16 {
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

func TestCaptureProfilesFromAxesCoversEveryTransmittedAxisValue(t *testing.T) {
	profiles, err := CaptureProfilesFromAxes(filepath.Join(repoRootForSupportTest(t), "scenarios", "experience-manager", "capabilities", "axes.json"), 16)
	if err != nil {
		t.Fatalf("CaptureProfilesFromAxes: %v", err)
	}
	seen := map[string]map[string]bool{}
	for _, profile := range profiles {
		seen["viewport"] = addSeen(seen["viewport"], profile.ID)
		seen["color-scheme"] = addSeen(seen["color-scheme"], profile.ColorScheme)
		seen["locale"] = addSeen(seen["locale"], profile.Locale)
		seen["motion-preference"] = addSeen(seen["motion-preference"], profile.MotionPreference)
		seen["interaction-state"] = addSeen(seen["interaction-state"], profile.InteractionState)
	}
	for axis, values := range map[string][]string{
		"viewport":          {"mobile", "tablet", "desktop", "wide"},
		"color-scheme":      {"light", "dark"},
		"locale":            {"en", "ar", "ja", "de"},
		"motion-preference": {"no-preference", "reduce"},
		"interaction-state": {"rest", "hover", "focus-visible", "pressed", "disabled"},
	} {
		for _, value := range values {
			if !seen[axis][value] {
				t.Errorf("axis %s value %q is not represented in profiles", axis, value)
			}
		}
	}
}

func addSeen(values map[string]bool, value string) map[string]bool {
	if values == nil {
		values = map[string]bool{}
	}
	values[value] = true
	return values
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

func TestAvailableEvidenceIncludesRetrievableScreenshot(t *testing.T) {
	if !slices.Contains(AvailableEvidence(), "screenshot") {
		t.Fatal("screenshot must be advertised because BAS captures and persists ScreenshotRef")
	}
}
