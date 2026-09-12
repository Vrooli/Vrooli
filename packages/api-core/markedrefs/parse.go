package markedrefs

import (
	"regexp"
	"strings"
)

var tokenPattern = regexp.MustCompile(`^([a-z][a-z0-9-]*)(?:\[([a-z][a-z0-9-]*(?:,[a-z][a-z0-9-]*)*)\])?:(.+)$`)

// Reference is one parsed marked inline reference.
//
// Marker and Qualifiers are metadata. Value is the literal value after the
// marker colon and does not include the marker or qualifier text.
type Reference struct {
	Marker     string
	Qualifiers []string
	Value      string
	Raw        string
	Line       int
	Column     int
}

// ParseToken parses one inline-code token without surrounding backticks.
//
// It returns ok=false when token is not marker-shaped. Unknown markers and
// unknown qualifiers still parse; callers can inspect UnknownMarker and
// UnknownQualifiers to decide how to report them.
func ParseToken(token string) (Reference, bool) {
	raw := token
	token = strings.TrimSpace(token)
	matches := tokenPattern.FindStringSubmatch(token)
	if matches == nil {
		return Reference{}, false
	}

	qualifiers := splitQualifiers(matches[2])
	ref := Reference{
		Marker:     matches[1],
		Qualifiers: qualifiers,
		Value:      strings.TrimSpace(matches[3]),
		Raw:        raw,
	}
	if ref.Value == "" {
		return Reference{}, false
	}
	return ref, true
}

// ParseInlineCode parses every marked reference found in markdown inline code
// spans on a single line. The returned Column is one-based and points at the
// opening backtick.
func ParseInlineCode(line string, lineNumber int) []Reference {
	var refs []Reference
	for i := 0; i < len(line); i++ {
		if line[i] != '`' {
			continue
		}
		start := i
		end := strings.IndexByte(line[start+1:], '`')
		if end < 0 {
			break
		}
		end = start + 1 + end
		token := line[start+1 : end]
		if ref, ok := ParseToken(token); ok {
			ref.Line = lineNumber
			ref.Column = start + 1
			ref.Raw = line[start : end+1]
			refs = append(refs, ref)
		}
		i = end
	}
	return refs
}

func splitQualifiers(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
