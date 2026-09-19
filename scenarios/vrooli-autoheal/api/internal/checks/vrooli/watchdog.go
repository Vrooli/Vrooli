// Package vrooli provides Vrooli-specific health checks
// [REQ:WATCH-DETECT-001] OS Watchdog health monitoring
package vrooli

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/watchdog"
)

// setupRemediation is the one repair for every boot-protection finding. The
// control plane's autoheal_watchdog safeguard installs, enables, lingers and
// restarts the unit; the scenario only observes it. Before 2026-09-02 this
// check offered install/enable/enable-linger/uninstall actions whose
// executors had been replaced with "run vrooli setup" stubs, so the UI showed
// repairs that could never happen.
const setupRemediation = "vrooli setup"

// WatchdogCheck observes the OS-level watchdog unit that restarts the autoheal
// loop after a crash or reboot. It is observation-only.
type WatchdogCheck struct {
	detector *watchdog.Detector
}

// WatchdogCheckOption configures a WatchdogCheck.
type WatchdogCheckOption func(*WatchdogCheck)

// WithWatchdogDetector sets the watchdog detector (for testing).
func WithWatchdogDetector(detector *watchdog.Detector) WatchdogCheckOption {
	return func(c *WatchdogCheck) {
		c.detector = detector
	}
}

// NewWatchdogCheck creates an OS watchdog health check.
func NewWatchdogCheck(caps *platform.Capabilities, opts ...WatchdogCheckOption) *WatchdogCheck {
	c := &WatchdogCheck{
		detector: watchdog.NewDetector(caps),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *WatchdogCheck) ID() string    { return "os-watchdog" }
func (c *WatchdogCheck) Title() string { return "OS Watchdog" }
func (c *WatchdogCheck) Description() string {
	return "Observes the OS-level watchdog unit that gives the autoheal loop boot and crash recovery"
}

func (c *WatchdogCheck) Importance() string {
	return "Without the unit, nothing restarts vrooli-autoheal after a reboot or crash; the repair is `vrooli setup`, never this API"
}
func (c *WatchdogCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *WatchdogCheck) IntervalSeconds() int       { return 300 } // Check every 5 minutes
func (c *WatchdogCheck) Platforms() []platform.Type { return nil } // All platforms

func (c *WatchdogCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{"remediation": setupRemediation},
	}

	// Every run re-probes: a cached status from five minutes ago is exactly
	// the stale answer a boot-protection check must not give.
	c.detector.Invalidate()
	status := c.detector.Detect()

	result.Details["watchdogType"] = string(status.WatchdogType)
	result.Details["installed"] = status.WatchdogInstalled
	result.Details["enabled"] = status.WatchdogEnabled
	result.Details["running"] = status.WatchdogRunning
	result.Details["bootProtectionActive"] = status.BootProtectionActive
	result.Details["protectionLevel"] = string(status.ProtectionLevel)
	result.Details["canInstall"] = status.CanInstall
	result.Details["loopRunning"] = status.LoopRunning

	if status.ServicePath != "" {
		result.Details["servicePath"] = status.ServicePath
	}
	if status.IsUserService {
		result.Details["isUserService"] = status.IsUserService
		result.Details["lingeringEnabled"] = status.LingeringEnabled
	}
	if status.LastError != "" {
		result.Details["lastError"] = status.LastError
	}

	subChecks := []checks.SubCheck{
		{
			Name:   "watchdog-installed",
			Passed: status.WatchdogInstalled,
			Detail: fmt.Sprintf("Service installed: %v", status.WatchdogInstalled),
		},
		{
			Name:   "watchdog-enabled",
			Passed: status.WatchdogEnabled,
			Detail: fmt.Sprintf("Service enabled: %v", status.WatchdogEnabled),
		},
		{
			Name:   "watchdog-running",
			Passed: status.WatchdogRunning,
			Detail: fmt.Sprintf("Service running: %v", status.WatchdogRunning),
		},
		{
			Name:   "loop-running",
			Passed: status.LoopRunning,
			Detail: fmt.Sprintf("Loop main process present: %v", status.LoopRunning),
		},
	}
	if status.IsUserService {
		subChecks = append(subChecks, checks.SubCheck{
			Name:   "lingering-enabled",
			Passed: status.LingeringEnabled,
			Detail: fmt.Sprintf("Systemd lingering enabled: %v (required for headless boot)", status.LingeringEnabled),
		})
	}

	score := 0
	switch status.ProtectionLevel {
	case watchdog.ProtectionFull:
		score = 100
	case watchdog.ProtectionPartial:
		score = 50
	case watchdog.ProtectionNone:
		score = 0
	}
	result.Metrics = &checks.HealthMetrics{Score: &score, SubChecks: subChecks}

	switch status.ProtectionLevel {
	case watchdog.ProtectionFull:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Full boot protection active (%s)", status.WatchdogType)
		if status.IsUserService && !status.LingeringEnabled {
			result.Status = checks.StatusWarning
			result.Message = "Watchdog running but lingering not enabled - won't start on headless boot; run `vrooli setup`"
		}

	case watchdog.ProtectionPartial:
		result.Status = checks.StatusWarning
		switch {
		case status.LoopRunning && !status.WatchdogInstalled:
			result.Message = "Loop is running but the OS watchdog unit is not installed - no crash/reboot recovery; run `vrooli setup`"
		case status.WatchdogInstalled && !status.WatchdogEnabled:
			result.Message = "Watchdog unit installed but not enabled; run `vrooli setup`"
		case status.WatchdogInstalled && status.WatchdogEnabled && !status.LoopRunning:
			result.Status = checks.StatusCritical
			result.Message = "Watchdog unit is enabled but the loop has no main process; run `vrooli setup`"
		default:
			result.Message = "Partial protection - loop running but the watchdog unit is not configured properly; run `vrooli setup`"
		}

	case watchdog.ProtectionNone:
		switch {
		case status.CanInstall:
			result.Status = checks.StatusCritical
			result.Message = "No boot protection - run `vrooli setup` to install the autoheal watchdog unit"
		case status.LastError != "":
			result.Status = checks.StatusWarning
			result.Message = fmt.Sprintf("Watchdog not available: %s", status.LastError)
		default:
			result.Status = checks.StatusWarning
			result.Message = "Watchdog not available on this platform"
		}
	}

	return result
}

// The check is deliberately not a HealableCheck: it offers no recovery
// actions because it can perform none.
var _ checks.Check = (*WatchdogCheck)(nil)
