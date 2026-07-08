package deps

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
			out = append(out, DeclarationFields{DepName: k, VersionRange: v.Range, Kind: normalizeKind(DepKind(v.Kind))})
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
			out = append(out, DeclarationFields{DepName: e.Name, VersionRange: e.Range, Kind: normalizeKind(DepKind(e.Kind))})
		}
		return out, nil
	}
	return nil, fmt.Errorf("@deps must be a JSON object or array, got %q", raw[:1])
}
