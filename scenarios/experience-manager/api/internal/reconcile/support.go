package reconcile

import "sort"

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

// WiredAxes returns the capture axes the reconciler drives today.
//
// Only viewport is wired: CaptureTarget carries ViewportID, ViewportWidth, and
// ViewportHeight and nothing else, so no other dimension can vary between two
// capture targets. Six further axes (color-scheme, locale, interaction-state,
// orientation, connectivity, and the browser half of pointer) are already
// available in the underlying engine and are unwired here, which is a plumbing
// gap rather than an engineering one.
func WiredAxes() []AxisSupport {
	out := make([]AxisSupport, 0, 1)
	values := make([]string, 0, len(DefaultCaptureProfiles))
	for _, profile := range DefaultCaptureProfiles {
		values = append(values, profile.ID)
	}
	sort.Strings(values)
	out = append(out, AxisSupport{Axis: "viewport", Values: values})
	return out
}

// AvailableEvidence returns the evidence kinds a capture actually yields.
//
// ax-tree and layout-box both come from the accessibility snapshot; layout-box is
// listed because bounds are populated often enough to be usable, though not
// universally — the live spacing and size-parity claims report unverifiable for
// exactly that reason. computed-style does not exist at all, which is why every
// contrast, typography, ramp, and motion-duration capability is blocked. The
// screenshot, runtime-log, and timing channels exist elsewhere in the fleet but
// are not joined to experience evidence, so they are deliberately absent here.
func AvailableEvidence() []string {
	return []string{"ax-tree", "layout-box"}
}

// ImplementedClaimTypes returns the claim types that have a deterministic
// evaluator, sorted. A claim type absent from this list parses and is accepted by
// the spec, but can only ever record an unverifiable verdict.
func ImplementedClaimTypes() []string {
	out := make([]string, 0, len(claimEvaluators))
	for claimType := range claimEvaluators {
		out = append(out, claimType)
	}
	sort.Strings(out)
	return out
}
