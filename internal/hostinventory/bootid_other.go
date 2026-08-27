//go:build !linux

package hostinventory

import "github.com/vrooli/vrooli/internal/scenarioruntime"

func bootID() string {
	return scenarioruntime.HealthStatusUnknown
}
