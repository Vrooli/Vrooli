// Package captureprofile defines the capture-depth dial: a request-level lever
// controlling how deeply the smoke phase captures UI visuals. It is orthogonal
// to the phase set (which phases run) — it tunes only the depth of the smoke
// phase's visual capture.
//
// Lever summary (control-surface-tunable-levers-design):
//
//	name:    capture profile
//	field:   StartRunRequest.capture_profile / SuiteExecutionRequest.CaptureProfile
//	default: "" (default depth) — single-page smoke on the workflow engine,
//	         unchanged in cost. This is the FLOOR: routine `comprehensive` runs
//	         pay nothing extra.
//	values:  "baseline" — all-pages visual capture (one CaptureService.Capture
//	         per discovered page) + video + full diagnostics. Used by GCT's
//	         baseline snapshot. This is the ceiling.
//	effect:  gates the smoke phase's all-pages mode and whether video is captured.
package captureprofile

import "strings"

// Profile is a parsed capture-depth profile.
type Profile struct {
	// AllPages requests one visual capture per discovered page (in addition to
	// the single-page handshake smoke, which always runs on the workflow engine).
	AllPages bool
	// Video requests a VIDEO artifact for each captured page.
	Video bool
	// Name is the canonical profile name ("" for default depth).
	Name string
}

const (
	// NameDefault is the empty default profile: single-page smoke, no extra cost.
	NameDefault = ""
	// NameBaseline is the full-diagnostics, all-pages, video profile used by GCT
	// baseline snapshots.
	NameBaseline = "baseline"
)

// DiagnosticsPreset returns the playbooks diagnostics preset that pairs with a
// capture profile, or "" to leave the request's own diagnostics preset in
// control. The baseline profile pairs with "full" diagnostics.
func (p Profile) DiagnosticsPreset() string {
	if p.Name == NameBaseline {
		return "full"
	}
	return ""
}

// Resolve maps a profile name to a Profile. An empty/unknown name yields the
// default (cheapest) profile; ok reports whether the name was recognized so
// callers can warn on a typo without changing behavior.
func Resolve(name string) (Profile, bool) {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case NameDefault:
		return Profile{Name: NameDefault}, true
	case NameBaseline:
		return Profile{Name: NameBaseline, AllPages: true, Video: true}, true
	default:
		return Profile{Name: NameDefault}, false
	}
}
