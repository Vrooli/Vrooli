package conflicts

import (
	"sort"
	"strings"
)

// SortedUniqueDomainNames returns non-empty, trimmed domain names in stable
// order for conflict envelopes shared by multiple detector families.
func SortedUniqueDomainNames(parts ...string) []string {
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	sort.Strings(out)
	return out
}
