// Package vrooli provides Vrooli-specific health checks
// [REQ:WATCH-DETECT-001] OS Watchdog health monitoring
package vrooli

import (
	"context"
	"fmt"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
	"vrooli-autoheal/internal/watchdog"
)

// WatchdogCheck monitors the OS-level watchdog service status.
// This check ensures boot recovery protection is properly configured.
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
	return "Monitors OS-level watchdog service for boot recovery protection"
}
func (c *WatchdogCheck) Importance() string {
	return "Ensures vrooli-autoheal automatically restarts after system reboots or crashes"
}
func (c *WatchdogCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *WatchdogCheck) IntervalSeconds() int       { return 300 } // Check every 5 minutes
func (c *WatchdogCheck) Platforms() []platform.Type { return nil } // All platforms

func (c *WatchdogCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{},
	}

	// Get current watchdog status
	status := c.detector.Detect()

	// Populate details
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

	// Build health metrics
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
			Passed: status.WatchdogRunning || status.LoopRunning,
			Detail: fmt.Sprintf("Service running: %v, Loop running: %v", status.WatchdogRunning, status.LoopRunning),
		},
	}

	// Add lingering check for Linux user services
	if status.IsUserService {
		subChecks = append(subChecks, checks.SubCheck{
			Name:   "lingering-enabled",
			Passed: status.LingeringEnabled,
			Detail: fmt.Sprintf("Systemd lingering enabled: %v (required for headless boot)", status.LingeringEnabled),
		})
	}

	// Calculate score based on protection level
	score := 0
	switch status.ProtectionLevel {
	case watchdog.ProtectionFull:
		score = 100
	case watchdog.ProtectionPartial:
		score = 50
	case watchdog.ProtectionNone:
		score = 0
	}

	result.Metrics = &checks.HealthMetrics{
		Score:     &score,
		SubChecks: subChecks,
	}

	// Determine status and message based on protection level
	switch status.ProtectionLevel {
	case watchdog.ProtectionFull:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Full boot protection active (%s)", status.WatchdogType)

		// Check lingering for user services - if missing, downgrade to warning
		if status.IsUserService && !status.LingeringEnabled {
			result.Status = checks.StatusWarning
			result.Message = "Watchdog running but lingering not enabled - won't start on headless boot"
		}

	case watchdog.ProtectionPartial:
		result.Status = checks.StatusWarning
		if status.LoopRunning && !status.WatchdogInstalled {
			result.Message = "Loop is running but OS watchdog not installed - no crash/reboot recovery"
		} else if status.WatchdogInstalled && !status.WatchdogEnabled {
			result.Message = "Watchdog installed but not enabled - run 'vrooli-autoheal install'"
		} else {
			result.Message = "Partial protection - loop running but watchdog not configured properly"
		}

	case watchdog.ProtectionNone:
		if status.CanInstall {
			result.Status = checks.StatusCritical
			result.Message = "No boot protection - run 'vrooli-autoheal install' to enable"
		} else if status.LastError != "" {
			result.Status = checks.StatusWarning
			result.Message = fmt.Sprintf("Watchdog not available: %s", status.LastError)
		} else {
			result.Status = checks.StatusWarning
			result.Message = "Watchdog not available on this platform"
		}
	}

	return result
}

// RecoveryActions returns available recovery actions for the watchdog check
// [REQ:HEAL-ACTION-001]
func (c *WatchdogCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	status := c.detector.GetCached()

	actions := []checks.RecoveryAction{
		{
			ID:          "install",
			Name:        "Install Watchdog",
			Description: "Install OS-level watchdog service for boot recovery",
			Dangerous:   false,
			Available:   status.CanInstall && !status.WatchdogInstalled,
		},
		{
			ID:          "enable",
			Name:        "Enable Watchdog",
			Description: "Enable the installed watchdog service",
			Dangerous:   false,
			Available:   status.WatchdogInstalled && !status.WatchdogEnabled,
		},
		{
			ID:          "enable-linger",
			Name:        "Enable Lingering",
			Description: "Enable systemd lingering for headless boot support (Linux only)",
			Dangerous:   false,
			Available:   status.IsUserService && !status.LingeringEnabled,
		},
		{
			ID:          "uninstall",
			Name:        "Uninstall Watchdog",
			Description: "Remove the OS-level watchdog service",
			Dangerous:   true,
			Available:   status.WatchdogInstalled,
		},
		{
			ID:          "diagnose",
			Name:        "Diagnose",
			Description: "Get diagnostic information about watchdog status",
			Dangerous:   false,
			Available:   true,
		},
	}

	return actions
}

// ExecuteAction runs the specified recovery action for the watchdog
// [REQ:HEAL-ACTION-001]
func (c *WatchdogCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	result := checks.ActionResult{
		ActionID: actionID,
		CheckID:  c.ID(),
	}

	switch actionID {
	case "install":
		installResult := c.detector.Install(ctx, watchdog.InstallOptions{
			UseSystemService: false, // Default to user service (safer)
			EnableLingering:  true,  // Try to enable lingering automatically
		})
		result.Success = installResult.Success
		result.Message = installResult.Message
		if installResult.Error != "" {
			result.Error = installResult.Error
		}
		if installResult.ServicePath != "" {
			result.Output = fmt.Sprintf("Service path: %s", installResult.ServicePath)
		}

	case "enable":
		// For enable, we reinstall which will enable it
		installResult := c.detector.Install(ctx, watchdog.InstallOptions{
			UseSystemService: false,
			EnableLingering:  true,
		})
		result.Success = installResult.Success
		result.Message = installResult.Message
		if installResult.Error != "" {
			result.Error = installResult.Error
		}

	case "enable-linger":
		lingerResult := c.detector.EnableLingering(ctx)
		result.Success = lingerResult.Success
		result.Message = lingerResult.Message
		if lingerResult.Error != "" {
			result.Error = lingerResult.Error
		}
		if lingerResult.LingerCommand != "" {
			result.Output = fmt.Sprintf("Manual command: %s", lingerResult.LingerCommand)
		}

	case "uninstall":
		uninstallResult := c.detector.Uninstall(ctx)
		result.Success = uninstallResult.Success
		result.Message = uninstallResult.Message
		if uninstallResult.Error != "" {
			result.Error = uninstallResult.Error
		}

	case "diagnose":
		status := c.detector.Detect()
		result.Success = true
		result.Message = "Diagnostic information gathered"
		result.Output = fmt.Sprintf(
			"Platform: %s\n"+
				"Watchdog Type: %s\n"+
				"Installed: %v\n"+
				"Enabled: %v\n"+
				"Running: %v\n"+
				"Loop Running: %v\n"+
				"Boot Protection: %v\n"+
				"Protection Level: %s\n"+
				"Service Path: %s\n"+
				"Is User Service: %v\n"+
				"Lingering Enabled: %v\n"+
				"Can Install: %v\n"+
				"Last Error: %s",
			status.WatchdogType,
			status.WatchdogType,
			status.WatchdogInstalled,
			status.WatchdogEnabled,
			status.WatchdogRunning,
			status.LoopRunning,
			status.BootProtectionActive,
			status.ProtectionLevel,
			status.ServicePath,
			status.IsUserService,
			status.LingeringEnabled,
			status.CanInstall,
			status.LastError,
		)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
	}

	return result
}

// Ensure WatchdogCheck implements HealableCheck
var _ checks.HealableCheck = (*WatchdogCheck)(nil)
