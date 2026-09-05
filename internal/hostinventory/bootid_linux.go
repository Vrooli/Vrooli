//go:build linux

package hostinventory

import (
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func bootID() string {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return scenarioruntime.HealthStatusUnknown
	}
	return strings.TrimSpace(string(b))
}
