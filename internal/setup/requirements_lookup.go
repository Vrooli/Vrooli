package setup

import (
	"strings"

	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

func findItemByName(report vrooliruntime.Report, name string) (vrooliruntime.ItemStatus, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return vrooliruntime.ItemStatus{}, false
	}
	for _, item := range report.Tools {
		if strings.ToLower(item.Name) == lower {
			return item, true
		}
	}
	for _, item := range report.Safeguards {
		if strings.ToLower(item.Name) == lower {
			return item, true
		}
	}
	return vrooliruntime.ItemStatus{}, false
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
