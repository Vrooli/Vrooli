package services

import "strings"

func truncateForLog(val string, max int) string {
	trimmed := strings.TrimSpace(val)
	if len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max] + "…"
}
