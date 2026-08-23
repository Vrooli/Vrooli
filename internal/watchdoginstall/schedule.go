// Package watchdoginstall defines the portable scheduling contract for the
// emergency watchdog. The safeguard handler owns file installation and the
// native command calls; this package owns the cross-platform schedule policy
// shared by inspection, rendering, and tests.
package watchdoginstall

import (
	"fmt"
	"strings"
	"time"
)

const DefaultInterval = 5 * time.Minute

type Schedule struct {
	Platform    string
	Backend     string
	Interval    time.Duration
	Supported   bool
	Remediation string
}

// For returns the native scheduler contract for an operating-system token.
// macos is accepted alongside Go's darwin token because safeguard manifests
// use the product-level platform vocabulary.
func For(goos string, interval time.Duration) Schedule {
	if interval <= 0 {
		interval = DefaultInterval
	} else if interval < time.Minute {
		interval = time.Minute
	}
	normalized := strings.ToLower(strings.TrimSpace(goos))
	switch normalized {
	case "macos":
		normalized = "darwin"
	}
	backend := map[string]string{
		"linux":   "systemd-user-timer",
		"darwin":  "launchd-user-agent",
		"windows": "windows-task-scheduler",
	}[normalized]
	if backend == "" {
		return Schedule{
			Platform:    normalized,
			Interval:    interval,
			Remediation: fmt.Sprintf("install a supported native scheduler for %q or run the watchdog manually", normalized),
		}
	}
	return Schedule{Platform: normalized, Backend: backend, Interval: interval, Supported: true}
}
