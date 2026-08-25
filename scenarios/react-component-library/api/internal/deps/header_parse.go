package deps

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var sourceDepsHeaderRE = regexp.MustCompile(`(?m)^\s*(?:\*|//)\s*@deps\s+(.+?)\s*$`)

// ParseSourceDeclarations extracts every @deps declaration in a versioned
// source file. Preview bundles follow relative imports across a version's
// immutable file set, so the runtime dependency surface is the union of the
// declarations carried by that closure—not only the entry file.
func ParseSourceDeclarations(source string) ([]DeclarationFields, error) {
	var out []DeclarationFields
	for _, match := range sourceDepsHeaderRE.FindAllStringSubmatch(source, -1) {
		declarations, err := ParseHeaderField(strings.TrimSpace(match[1]))
		if err != nil {
			return nil, err
		}
		out = append(out, declarations...)
	}
	return out, nil
}

// ParseHeaderField interprets the raw `@deps` header field value the
// components indexer captures into Headers["deps"]. Accepted shapes:
//
//	{}                                         → no declarations
//	{"react": "^18.0.0"}                       → map form
//	{"react": "^18.0.0", "lodash": "*"}
//	[{"name":"react","range":"^18.0.0","kind":"peer"}, ...]  → array form
//
// Missing or empty → nil, nil (perfectly valid; component has no deps).
// Malformed JSON → error so the operator sees a fix-the-header signal.
func ParseHeaderField(raw string) ([]DeclarationFields, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if strings.HasPrefix(raw, "{") {
		var simple map[string]string
		if err := json.Unmarshal([]byte(raw), &simple); err == nil {
			out := make([]DeclarationFields, 0, len(simple))
			for k, v := range simple {
				out = append(out, DeclarationFields{DepName: k, VersionRange: v, Kind: DepKindRuntime})
			}
			return out, nil
		}
		var detailed map[string]struct {
			Range string `json:"range"`
			Kind  string `json:"kind"`
		}
		if err := json.Unmarshal([]byte(raw), &detailed); err != nil {
			return nil, fmt.Errorf("invalid @deps object: %w", err)
		}
		out := make([]DeclarationFields, 0, len(detailed))
		for k, v := range detailed {
			kind, err := parseKind(v.Kind)
			if err != nil {
				return nil, err
			}
			out = append(out, DeclarationFields{DepName: k, VersionRange: v.Range, Kind: kind})
		}
		return out, nil
	}
	if strings.HasPrefix(raw, "[") {
		var arr []struct {
			Name  string `json:"name"`
			Range string `json:"range"`
			Kind  string `json:"kind"`
		}
		if err := json.Unmarshal([]byte(raw), &arr); err != nil {
			return nil, fmt.Errorf("invalid @deps array: %w", err)
		}
		out := make([]DeclarationFields, 0, len(arr))
		for _, e := range arr {
			kind, err := parseKind(e.Kind)
			if err != nil {
				return nil, err
			}
			out = append(out, DeclarationFields{DepName: e.Name, VersionRange: e.Range, Kind: kind})
		}
		return out, nil
	}
	return nil, fmt.Errorf("@deps must be a JSON object or array, got %q", raw[:1])
}

func parseKind(raw string) (DepKind, error) {
	kind := DepKind(strings.ToLower(strings.TrimSpace(raw)))
	if kind == "" {
		return DepKindRuntime, nil
	}
	switch kind {
	case DepKindRuntime, DepKindPeer, DepKindDev:
		return kind, nil
	default:
		return "", fmt.Errorf("invalid @deps kind %q", raw)
	}
}
