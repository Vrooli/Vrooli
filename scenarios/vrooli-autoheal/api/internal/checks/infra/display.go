// Package infra provides infrastructure health checks
// [REQ:INFRA-DISPLAY-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// DisplayManagerCheck monitors the display manager (GDM, LightDM, SDDM, etc.)
// and X11/Wayland responsiveness on Linux systems.
type DisplayManagerCheck struct {
	caps *platform.Capabilities
}

// NewDisplayManagerCheck creates a display manager health check.
func NewDisplayManagerCheck(caps *platform.Capabilities) *DisplayManagerCheck {
	return &DisplayManagerCheck{caps: caps}
}

func (c *DisplayManagerCheck) ID() string          { return "infra-display" }
func (c *DisplayManagerCheck) Title() string       { return "Display Manager" }
func (c *DisplayManagerCheck) Description() string { return "Monitors display manager and X11/Wayland status" }
func (c *DisplayManagerCheck) Importance() string {
	return "Required for GUI sessions - failures prevent graphical login and desktop access"
}
func (c *DisplayManagerCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *DisplayManagerCheck) IntervalSeconds() int       { return 300 } // Check every 5 minutes
func (c *DisplayManagerCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

// supportedDisplayManagers lists the display managers we can check
var supportedDisplayManagers = []string{
	"gdm",     // GNOME Display Manager
	"gdm3",    // GNOME Display Manager (Debian/Ubuntu)
	"lightdm", // Light Display Manager
	"sddm",    // Simple Desktop Display Manager (KDE)
	"lxdm",    // LXDE Display Manager
	"xdm",     // X Display Manager
}

func (c *DisplayManagerCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Only supported on Linux
	if runtime.GOOS != "linux" {
		result.Status = checks.StatusOK
		result.Message = "Display manager check not applicable on this platform"
		result.Details["platform"] = runtime.GOOS
		return result
	}

	// Check if this is a headless server
	if c.caps != nil && c.caps.IsHeadlessServer {
		result.Status = checks.StatusOK
		result.Message = "Headless server - no display manager expected"
		result.Details["headless"] = true
		return result
	}

	// Detect which display manager is in use
	activeManager, err := c.detectActiveDisplayManager(ctx)
	if err != nil || activeManager == "" {
		result.Status = checks.StatusOK
		result.Message = "No display manager detected (headless system or custom setup)"
		result.Details["detected"] = false
		return result
	}

	result.Details["displayManager"] = activeManager
	result.Details["detected"] = true

	// Check display manager service status
	serviceStatus := c.getServiceStatus(ctx, activeManager)
	result.Details["serviceStatus"] = serviceStatus

	// Check X11 display responsiveness (if DISPLAY is set)
	x11Status, x11Details := c.checkX11(ctx)
	result.Details["x11"] = x11Details

	// Calculate overall status
	var subChecks []checks.SubCheck

	// Display manager service check
	dmPassed := serviceStatus == "active"
	subChecks = append(subChecks, checks.SubCheck{
		Name:   "display-manager-service",
		Passed: dmPassed,
		Detail: activeManager + " service is " + serviceStatus,
	})

	// X11 check (only if DISPLAY is available)
	if x11Details["available"] == true {
		x11Passed := x11Status == "responsive"
		subChecks = append(subChecks, checks.SubCheck{
			Name:   "x11-responsiveness",
			Passed: x11Passed,
			Detail: "X11 display is " + x11Status,
		})
	}

	// Calculate score
	passed := 0
	for _, sc := range subChecks {
		if sc.Passed {
			passed++
		}
	}
	score := 100
	if len(subChecks) > 0 {
		score = (passed * 100) / len(subChecks)
	}
	result.Metrics = &checks.HealthMetrics{
		Score:     &score,
		SubChecks: subChecks,
	}

	// Determine final status
	if serviceStatus != "active" {
		result.Status = checks.StatusCritical
		result.Message = activeManager + " display manager is not running"
		return result
	}

	if x11Details["available"] == true && x11Status != "responsive" {
		result.Status = checks.StatusWarning
		result.Message = "Display manager running but X11 not responsive"
		return result
	}

	result.Status = checks.StatusOK
	result.Message = activeManager + " display manager is healthy"
	return result
}

// detectActiveDisplayManager finds which display manager is currently active
func (c *DisplayManagerCheck) detectActiveDisplayManager(ctx context.Context) (string, error) {
	// First check systemctl for the default display-manager target
	cmd := exec.CommandContext(ctx, "systemctl", "get-default")
	output, _ := cmd.Output()
	if strings.Contains(string(output), "graphical.target") {
		// Graphical target is default, look for active display manager
		for _, dm := range supportedDisplayManagers {
			cmd := exec.CommandContext(ctx, "systemctl", "is-active", dm)
			if output, err := cmd.Output(); err == nil {
				if strings.TrimSpace(string(output)) == "active" {
					return dm, nil
				}
			}
		}
	}

	// Fallback: check which display manager service exists and is enabled
	for _, dm := range supportedDisplayManagers {
		cmd := exec.CommandContext(ctx, "systemctl", "is-enabled", dm)
		if output, err := cmd.Output(); err == nil {
			status := strings.TrimSpace(string(output))
			if status == "enabled" || status == "static" {
				return dm, nil
			}
		}
	}

	return "", nil
}

// getServiceStatus checks the systemd service status
func (c *DisplayManagerCheck) getServiceStatus(ctx context.Context, service string) string {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", service)
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

// checkX11 tests X11 display responsiveness
func (c *DisplayManagerCheck) checkX11(ctx context.Context) (string, map[string]interface{}) {
	details := make(map[string]interface{})

	// Check if DISPLAY is set
	cmd := exec.CommandContext(ctx, "printenv", "DISPLAY")
	output, err := cmd.Output()
	display := strings.TrimSpace(string(output))

	if err != nil || display == "" {
		details["available"] = false
		details["reason"] = "DISPLAY environment variable not set"
		return "unavailable", details
	}

	details["available"] = true
	details["display"] = display

	// Try xdpyinfo to check X11 responsiveness
	cmd = exec.CommandContext(ctx, "xdpyinfo")
	_, err = cmd.Output()
	if err != nil {
		details["responsive"] = false
		details["error"] = err.Error()
		return "unresponsive", details
	}

	details["responsive"] = true
	return "responsive", details
}

// RecoveryActions returns available recovery actions for display manager
// [REQ:HEAL-ACTION-001]
func (c *DisplayManagerCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	hasSystemd := c.caps != nil && c.caps.SupportsSystemd
	isLinux := runtime.GOOS == "linux"

	dmName := "display-manager"
	if lastResult != nil {
		if name, ok := lastResult.Details["displayManager"].(string); ok && name != "" {
			dmName = name
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "restart",
			Name:        "Restart Display Manager",
			Description: "Restart the " + dmName + " service (WARNING: may disconnect active sessions)",
			Dangerous:   true, // Restarting DM disconnects users
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "status",
			Name:        "Check Status",
			Description: "Get detailed status of the display manager service",
			Dangerous:   false,
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent display manager logs",
			Dangerous:   false,
			Available:   isLinux && hasSystemd,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *DisplayManagerCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Detect active display manager
	dmName, _ := c.detectActiveDisplayManager(ctx)
	if dmName == "" {
		dmName = "display-manager" // Fallback service name
	}

	switch actionID {
	case "restart":
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "restart", dmName)
		output, err := cmd.CombinedOutput()
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart " + dmName
			return result
		}

		result.Success = true
		result.Message = dmName + " restarted successfully"
		return result

	case "status":
		cmd := exec.CommandContext(ctx, "systemctl", "status", dmName)
		output, _ := cmd.CombinedOutput()
		result.Duration = time.Since(start)
		result.Output = string(output)
		result.Success = true
		result.Message = "Retrieved " + dmName + " status"
		return result

	case "logs":
		cmd := exec.CommandContext(ctx, "journalctl", "-u", dmName, "-n", "100", "--no-pager")
		output, _ := cmd.CombinedOutput()
		result.Duration = time.Since(start)
		result.Output = string(output)
		result.Success = true
		result.Message = "Retrieved " + dmName + " logs"
		return result

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// Ensure DisplayManagerCheck implements HealableCheck
var _ checks.HealableCheck = (*DisplayManagerCheck)(nil)
