package scopecatalog

import "strings"

// ParsedScope is the normalized namespace:effect grammar used by Bridge grants.
type ParsedScope struct {
	Namespace string
	Effect    string
}

// ParseScope validates and normalizes one Bridge grant.
func ParseScope(raw string) (ParsedScope, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != 2 {
		return ParsedScope{}, false
	}
	namespace := strings.ToLower(strings.TrimSpace(parts[0]))
	effect := strings.ToLower(strings.TrimSpace(parts[1]))
	if namespace == "" || effect == "" {
		return ParsedScope{}, false
	}
	switch effect {
	case "read", "write", "destructive", "*":
		return ParsedScope{Namespace: namespace, Effect: effect}, true
	default:
		return ParsedScope{}, false
	}
}

// ClassifyScopes summarizes a grant for operator-facing surfaces.
func ClassifyScopes(scopes []string) (effects []string, appCount int, coversAllApps bool) {
	seenEffects := map[string]bool{}
	apps := map[string]struct{}{}
	for _, raw := range scopes {
		scope, ok := ParseScope(raw)
		if !ok {
			continue
		}
		if scope.Namespace == "*" {
			coversAllApps = true
		} else {
			apps[scope.Namespace] = struct{}{}
		}
		if scope.Effect == "*" {
			seenEffects["read"], seenEffects["write"], seenEffects["destructive"] = true, true, true
		} else {
			seenEffects[scope.Effect] = true
		}
	}
	for _, effect := range []string{"read", "write", "destructive"} {
		if seenEffects[effect] {
			effects = append(effects, effect)
		}
	}
	return effects, len(apps), coversAllApps
}

// SummarizeScopes returns stable operator wording for a grant.
func SummarizeScopes(scopes []string) string {
	effects, _, _ := ClassifyScopes(scopes)
	for i := len(effects) - 1; i >= 0; i-- {
		switch effects[i] {
		case "destructive":
			return "Full control, including destructive actions"
		case "write":
			return "Read and operate; destructive actions withheld"
		case "read":
			return "Read only; changes are not permitted"
		}
	}
	return "No remote actions granted"
}
