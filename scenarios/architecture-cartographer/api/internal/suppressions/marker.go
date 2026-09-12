// Package suppressions implements the durable, in-repo suppression marker:
// an inline source comment that sanctions a specific architecture finding
// as intentional. Markers are version-controlled and discoverable next to
// the code they excuse — the deliberate alternative to a DB-only suppression
// that is invisible to code readers.
//
// Grammar (modeled on the documentation-health `// DOC:` / `// seam:`
// conventions):
//
//	// arch:allow <id> reason="…" [expires="<condition>"]
//
// where <id> matches a detector name (e.g. "cycle"), a conflict type
// (e.g. "coupling_smell"), or a smell/finding subtype (e.g. "god_domain").
// The marker suppresses findings whose location or domain the marker's file
// belongs to. A suppressed finding is reported as suppressed-with-reason,
// never silently dropped.
package suppressions

import (
	"strings"
	"time"
)

// MarkerDirective is the literal that introduces a suppression marker inside a
// comment. The scanner matches it anywhere in a line so it works across
// comment styles (// , /* */, # ).
const MarkerDirective = "arch:allow"

// Marker is one parsed suppression directive.
type Marker struct {
	// ID is the detector name, conflict type, or finding subtype this
	// marker sanctions.
	ID string
	// Reason is the mandatory human rationale.
	Reason string
	// Expires is an optional condition; empty means no automatic expiry.
	// The only machine-evaluated form is "until:YYYY-MM-DD" (the marker is
	// inactive after that date). Other free-form conditions are advisory.
	Expires string
	// File is the repo-relative file the marker lives in (set by the scanner).
	File string
	// Line is the 1-based line number of the marker.
	Line int
}

// ParseMarker extracts a Marker from a single line of source. ok is false
// when the line carries no marker token. A marker token with no id or no
// reason still parses (returns ok=true) but is flagged invalid by Validate
// — the scanner surfaces malformed markers rather than ignoring them.
func ParseMarker(line string) (Marker, bool) {
	idx := strings.Index(line, MarkerDirective)
	if idx < 0 {
		return Marker{}, false
	}
	rest := strings.TrimSpace(line[idx+len(MarkerDirective):])
	// Trim a trailing comment closer like "*/".
	rest = strings.TrimSpace(strings.TrimSuffix(rest, "*/"))

	m := Marker{}
	// The id is the first whitespace-delimited token before any key=value.
	if sp := firstFieldEnd(rest); sp > 0 {
		m.ID = rest[:sp]
		rest = strings.TrimSpace(rest[sp:])
	} else {
		m.ID = rest
		rest = ""
	}
	m.Reason = extractQuoted(rest, "reason")
	m.Expires = extractQuoted(rest, "expires")
	return m, true
}

// Validate reports whether the marker is well-formed (has an id and a
// non-empty reason). Malformed markers are surfaced as diagnostics, not
// honored as suppressions.
func (m Marker) Validate() bool {
	return strings.TrimSpace(m.ID) != "" && strings.TrimSpace(m.Reason) != ""
}

// IsActive reports whether the marker suppresses at time now. A marker with
// no expiry, or a non-"until:" expiry, is always active. An "until:DATE"
// marker is inactive once now is past DATE (end of that day).
func (m Marker) IsActive(now time.Time) bool {
	exp := strings.TrimSpace(m.Expires)
	const untilPrefix = "until:"
	if !strings.HasPrefix(exp, untilPrefix) {
		return true
	}
	dateStr := strings.TrimSpace(strings.TrimPrefix(exp, untilPrefix))
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return true // unparseable date → advisory, stays active
	}
	// Active through the end of the named day.
	return !now.After(d.Add(24 * time.Hour))
}

// firstFieldEnd returns the index just past the first whitespace-delimited
// token, or -1 if the whole string is one token.
func firstFieldEnd(s string) int {
	for i, r := range s {
		if r == ' ' || r == '\t' {
			return i
		}
	}
	return -1
}

// extractQuoted returns the value of key="value" (double quotes) from s, or
// "" when the key is absent.
func extractQuoted(s, key string) string {
	needle := key + "=\""
	idx := strings.Index(s, needle)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(needle):]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	return rest[:end]
}
