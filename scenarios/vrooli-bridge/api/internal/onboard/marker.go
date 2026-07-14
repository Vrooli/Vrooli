package onboard

import (
	"regexp"
	"strings"
)

// uuidPattern matches a canonical UUID (the shape a registry node id takes). The
// bootstrap emits the node id inside the free-text `detail` of the pair-redeem /
// pin-verify / run-ok markers ("paired as <id>", "node <id> paired and online");
// extracting the UUID token is robust to the surrounding phrasing.
var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// extractNodeID returns the first UUID-shaped token in a marker detail, or "" if
// none is present.
func extractNodeID(detail string) string {
	return uuidPattern.FindString(detail)
}

// markerPrefix is the leading token every bootstrap progress line carries on
// stdout. The bootstrap contract (bootstrap/README.md) guarantees stdout is
// marker-only, so a line that does not begin with this prefix is ignored.
const markerPrefix = "VBOOTSTRAP "

// Bootstrap marker events (bootstrap/README.md "Marker grammar").
const (
	eventRunStart  = "run-start"
	eventRunOK     = "run-ok"
	eventRunFail   = "run-fail"
	eventStepStart = "step-start"
	eventStepOK    = "step-ok"
	eventStepSkip  = "step-skip"
	eventStepFail  = "step-fail"
)

// Marker is one parsed VBOOTSTRAP stdout line. Detail is unescaped from the
// double-quoted, newline-free `detail="…"` field the grammar guarantees is
// always last.
type Marker struct {
	Event  string
	Step   string
	Detail string
}

// parseMarker parses a single stdout line into a Marker. ok is false for a line
// that is not a VBOOTSTRAP marker (the caller ignores it) or one missing the
// required event field.
//
// Grammar (bootstrap/README.md): `VBOOTSTRAP v=1 event=<event> [step=<step-id>]
// [detail="<single-line text>"]`. Fields are space-separated key=value; detail
// is always last, double-quoted, newline-free. Parsing detail specially (rather
// than naive space-splitting) lets its value contain spaces.
func parseMarker(line string) (Marker, bool) {
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, markerPrefix) {
		return Marker{}, false
	}
	rest := line[len(markerPrefix):]

	var (
		m         Marker
		detailSet bool
	)
	for rest != "" {
		// detail is always last and its value may contain spaces, so peel it off
		// whole rather than space-splitting.
		if strings.HasPrefix(rest, "detail=") {
			m.Detail = unquoteDetail(strings.TrimPrefix(rest, "detail="))
			detailSet = true
			break
		}
		field := rest
		if i := strings.IndexByte(rest, ' '); i >= 0 {
			field = rest[:i]
			rest = rest[i+1:]
		} else {
			rest = ""
		}
		key, val, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch key {
		case "event":
			m.Event = val
		case "step":
			m.Step = val
		}
	}
	_ = detailSet
	if m.Event == "" {
		return Marker{}, false
	}
	return m, true
}

// unquoteDetail strips the surrounding double quotes from a detail value. The
// grammar guarantees the value is newline-free and double-quoted; a value that
// is not quoted (defensive) is returned as-is.
func unquoteDetail(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return v
}

// stepStatusForEvent maps a step-level marker event to its StepStatus. It
// returns StepStatusUnspecified for a non-step event.
func stepStatusForEvent(event string) StepStatus {
	switch event {
	case eventStepStart:
		return StepStatusStarted
	case eventStepOK:
		return StepStatusOK
	case eventStepSkip:
		return StepStatusSkipped
	case eventStepFail:
		return StepStatusFailed
	default:
		return StepStatusUnspecified
	}
}
