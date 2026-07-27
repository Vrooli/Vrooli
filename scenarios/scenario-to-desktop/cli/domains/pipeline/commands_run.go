package pipeline

import "strings"

// splitValues normalizes comma-separated declarative command values before a
// primitive builds its typed request.
func splitValues(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.ToLower(strings.TrimSpace(item)); item != "" {
			result = append(result, item)
		}
	}
	return result
}
