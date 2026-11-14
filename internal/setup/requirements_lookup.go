package setup

import (
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func findItemByName(report hostreqkit.Report, name string) (hostreqkit.ItemStatus, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return hostreqkit.ItemStatus{}, false
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
	return hostreqkit.ItemStatus{}, false
}
