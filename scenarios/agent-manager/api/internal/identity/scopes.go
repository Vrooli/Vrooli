package identity

import "strings"

// IntersectScopes computes the one-way attenuation of account, profile and
// request scopes. A nil input means that layer did not narrow the account;
// an explicit empty slice means that layer grants nothing.
func IntersectScopes(account, profile, requested []string) []string {
	candidates := requested
	if candidates == nil {
		candidates = profile
	}
	if candidates == nil {
		candidates = account
	}
	if candidates == nil {
		return []string{}
	}
	result := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, scope := range candidates {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if account != nil && !scopeCovered(account, scope) {
			continue
		}
		if profile != nil && !scopeCovered(profile, scope) {
			continue
		}
		if requested != nil && !scopeCovered(requested, scope) {
			continue
		}
		if _, ok := seen[scope]; !ok {
			seen[scope] = struct{}{}
			result = append(result, scope)
		}
	}
	return result
}

func scopeCovered(grants []string, requested string) bool {
	for _, grant := range grants {
		grant = strings.TrimSpace(grant)
		if grant == "*" || grant == requested {
			return true
		}
		if strings.HasSuffix(grant, "*") && strings.HasPrefix(requested, strings.TrimSuffix(grant, "*")) {
			return true
		}
	}
	return false
}
