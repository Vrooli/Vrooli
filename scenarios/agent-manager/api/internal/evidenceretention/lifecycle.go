package evidenceretention

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Effort is the lifecycle view of one Agent Manager effort enrollment that
// evidence retention needs. Keys are every name the effort is known by:
// display name, workspace, delivery commission and delivery home basename.
type Effort struct {
	Ref  string
	Keys []string
	// Terminal is true for a withdrawn effort or a stopped and finalized
	// delivery commission.
	Terminal bool
	// TerminalAt is when the effort became terminal (finalization, or the
	// enrollment update that withdrew it).
	TerminalAt time.Time
}

// Lifecycle states of an evidence directory.
const (
	StateActive     = "active"
	StateTerminal   = "terminal"
	StateUnenrolled = "unenrolled"
)

// Lifecycle explains whether automatic retention may touch an effort's
// evidence.
type Lifecycle struct {
	State      string     `json:"state"`
	EffortRefs []string   `json:"effortRefs,omitempty"`
	TerminalAt *time.Time `json:"terminalAt,omitempty"`
	// AutoEligible is true only for a terminal effort past the grace window.
	AutoEligible bool   `json:"autoEligible"`
	Reason       string `json:"reason"`
}

var (
	isoDateSuffix   = regexp.MustCompile(`[-_]?20\d\d-?\d\d-?\d\d$`)
	nonAlnum        = regexp.MustCompile(`[^a-z0-9]+`)
	deliverySuffix  = regexp.MustCompile(`-(delivery|campaign)$`)
	surroundingDash = "-"
)

// normalizeKey makes "Native-feel desktop packaging", "desktop-packaging"
// and ".vrooli/work/delivery/desktop-packaging-delivery" comparable: lower
// case, dashes, no trailing date or -delivery suffix.
func normalizeKey(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = path.Base(strings.TrimRight(s, "/"))
	s = isoDateSuffix.ReplaceAllString(s, "")
	s = nonAlnum.ReplaceAllString(s, surroundingDash)
	s = strings.Trim(s, surroundingDash)
	s = deliverySuffix.ReplaceAllString(s, "")
	return s
}

// ResolveLifecycle matches an evidence directory to enrolled efforts. Every
// matching enrollment must be terminal; one live match keeps the evidence
// active. No match leaves it unenrolled, which automatic retention never
// touches.
func ResolveLifecycle(dir string, efforts []Effort, grace time.Duration, now time.Time) Lifecycle {
	want := normalizeKey(dir)
	var matched []Effort
	if want != "" {
		for _, e := range efforts {
			for _, k := range e.Keys {
				if k != "" && normalizeKey(k) == want {
					matched = append(matched, e)
					break
				}
			}
		}
	}
	if len(matched) == 0 {
		return Lifecycle{State: StateUnenrolled, Reason: "no Agent Manager effort enrollment matches this directory; only an operator-named prune may remove its intermediates"}
	}
	lc := Lifecycle{State: StateTerminal}
	var latest time.Time
	for _, e := range matched {
		lc.EffortRefs = append(lc.EffortRefs, e.Ref)
		if !e.Terminal {
			lc.State = StateActive
		}
		if e.TerminalAt.After(latest) {
			latest = e.TerminalAt
		}
	}
	sort.Strings(lc.EffortRefs)
	if lc.State == StateActive {
		lc.Reason = "a matching effort is still live; its intermediates may still be in use"
		return lc
	}
	lc.TerminalAt = &latest
	if now.Sub(latest) < grace {
		lc.Reason = "effort is terminal but inside the " + grace.String() + " review grace window"
		return lc
	}
	lc.AutoEligible = true
	lc.Reason = "every matching effort is terminal past the grace window"
	return lc
}
