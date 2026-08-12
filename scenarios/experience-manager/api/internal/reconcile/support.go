package reconcile

import (
	"sort"

	"experience-manager/internal/claimtypes"
)

// This file is the reconciler's self-description: what it can actually drive and
// actually observe. The experience capability registry derives every capability's
// status from these three functions rather than recording status in its own files,
// which is what stops a capability from claiming to be provable when nothing can
// prove it.
//
// When the reconciler gains an axis or an evidence channel, the corresponding list
// here must grow with it. The tests in this package pin each list against the code
// that would have to change, so a new capture dimension cannot be added silently.

// AxisSupport describes how completely the reconciler drives one capture axis.
type AxisSupport struct {
	// Axis is the axis id as declared in the capability registry's axes.json.
	Axis string
	// Values are the axis values the reconciler can actually capture. An axis
	// that is driven but only at a subset of its declared values reports that
	// subset, so partial wiring is visible rather than rounding up to "works".
	Values []string
}

// WiredAxes returns the capture axes represented in the default capture matrix.
// Each returned value is transmitted to BAS as part of the capture profile;
// interaction-state is applied by the Playwright driver before evidence is read.
func WiredAxes() []AxisSupport { return WiredAxesFromProfiles(DefaultCaptureProfiles) }

// WiredAxesFromProfiles derives support only from values that the capture
// request can transmit. Empty fields are deliberately omitted: adding a
// field to CaptureTarget without putting it in the BAS request must not make
// capability status claim that axis is wired.
func WiredAxesFromProfiles(profiles []CaptureProfile) []AxisSupport {
	values := map[string]map[string]struct{}{}
	for _, profile := range profiles {
		addAxisValue(values, "viewport", profile.ID)
		addAxisValue(values, "color-scheme", profile.ColorScheme)
		addAxisValue(values, "locale", profile.Locale)
		addAxisValue(values, "motion-preference", profile.MotionPreference)
		addAxisValue(values, "interaction-state", profile.InteractionState)
	}
	var out []AxisSupport
	for axis, set := range values {
		if len(set) == 0 {
			continue
		}
		axisValues := make([]string, 0, len(set))
		for value := range set {
			axisValues = append(axisValues, value)
		}
		sort.Strings(axisValues)
		out = append(out, AxisSupport{Axis: axis, Values: axisValues})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Axis < out[j].Axis })
	return out
}

// AvailableEvidence returns the evidence kinds a capture actually yields.
//
// ax-tree, layout-box, computed-style, and screenshot all come from the BAS
// capture response
// snapshot. The latter is a declared property map, so evaluators can distinguish
// absent CSS evidence from a failed arithmetic check. runtime-log and timing
// remain outside the experience evidence join until their durable contracts are
// defined.
func AvailableEvidence() []string {
	return []string{"ax-tree", "computed-style", "layout-box", "screenshot"}
}

func addAxisValue(values map[string]map[string]struct{}, axis, value string) {
	if value == "" {
		return
	}
	if values[axis] == nil {
		values[axis] = map[string]struct{}{}
	}
	values[axis][value] = struct{}{}
}

// ImplementedClaimTypes returns the claim types that have a deterministic
// evaluator, sorted. A claim type absent from this list parses and is accepted by
// the spec, but can only ever record an unverifiable verdict.
func ImplementedClaimTypes() []string {
	// Keep the public self-description sourced from the evaluator map. The
	// shared vocabulary package lets the parser use the same registry without
	// creating a spec -> reconcile import cycle.
	out := make([]string, 0, len(claimEvaluators))
	for claimType := range claimEvaluators {
		out = append(out, claimType)
	}
	sort.Strings(out)
	if want := claimtypes.ImplementedClaimTypes(); !sameStrings(out, want) {
		panic("claim evaluator vocabulary drifted from shared claimtypes registry")
	}
	return out
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
