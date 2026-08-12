// Package infra provides infrastructure health checks
// [REQ:INFRA-DISPLAY-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// DisplayManagerCheck monitors the display manager (GDM, LightDM, SDDM, etc.)
// and X11/Wayland responsiveness on Linux systems.
type DisplayManagerCheck struct {
	caps                  *platform.Capabilities
	executor              checks.CommandExecutor
	autoLoginUserProvider func() string
}

// DisplayManagerOption configures a DisplayManagerCheck.
type DisplayManagerOption func(*DisplayManagerCheck)

// WithDisplayExecutor sets the command executor for testing.
// [REQ:TEST-SEAM-001]
func WithDisplayExecutor(exec checks.CommandExecutor) DisplayManagerOption {
	return func(c *DisplayManagerCheck) {
		c.executor = exec
	}
}

// WithDisplayAutoLoginUserProvider sets the auto-login discovery seam for tests.
// [REQ:TEST-SEAM-001]
func WithDisplayAutoLoginUserProvider(provider func() string) DisplayManagerOption {
	return func(c *DisplayManagerCheck) {
		c.autoLoginUserProvider = provider
	}
}

// NewDisplayManagerCheck creates a display manager health check.
func NewDisplayManagerCheck(caps *platform.Capabilities, opts ...DisplayManagerOption) *DisplayManagerCheck {
	c := &DisplayManagerCheck{
		caps:                  caps,
		executor:              checks.DefaultExecutor,
		autoLoginUserProvider: autoLoginUser,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *DisplayManagerCheck) ID() string    { return "infra-display" }
func (c *DisplayManagerCheck) Title() string { return "Display Manager" }
func (c *DisplayManagerCheck) Description() string {
	return "Monitors display manager and X11/Wayland status"
}

func (c *DisplayManagerCheck) Importance() string {
	return "Required for GUI sessions - failures prevent graphical login and desktop access"
}
func (c *DisplayManagerCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *DisplayManagerCheck) IntervalSeconds() int       { return 300 } // Check every 5 minutes
func (c *DisplayManagerCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

// supportedDisplayManagers lists the display managers we can check
var supportedDisplayManagers = hostinventory.DisplayManagerNames

func (c *DisplayManagerCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Only supported on Linux
	if runtime.GOOS != "linux" {
		result.Status = checks.StatusNotApplicable
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

	// Check for auto-login configuration and gnome-shell status
	autoLoginUser := c.getAutoLoginUser()
	result.Details["autoLoginUser"] = autoLoginUser

	sessionUser, gnomeShellRunning := graphicalSessionAvailable(ctx, c.executor, autoLoginUser)
	result.Details["gnomeShellRunning"] = gnomeShellRunning
	if sessionUser != "" {
		result.Details["gnomeShellUser"] = sessionUser
	}

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

	// GNOME shell check (when auto-login is configured)
	if autoLoginUser != "" {
		subChecks = append(subChecks, checks.SubCheck{
			Name:   "gnome-shell-session",
			Passed: gnomeShellRunning,
			Detail: func() string {
				if gnomeShellRunning {
					return "gnome-shell running for " + autoLoginUser
				}
				return "gnome-shell NOT running for auto-login user " + autoLoginUser
			}(),
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

	// Determine final status - check from most critical to least
	if serviceStatus != "active" {
		result.Status = checks.StatusCritical
		result.Message = activeManager + " display manager is not running"
		return result
	}

	// Auto-login configured but gnome-shell not running. Consumers that depend on
	// a graphical session (RDP, browser automation, screenshots) report their own
	// degradation; this check only reports the missing session.
	if autoLoginUser != "" && !gnomeShellRunning {
		result.Status = checks.StatusWarning
		result.Message = "Auto-login user " + autoLoginUser + " has no active desktop session"
		result.Details["healSuggestion"] = "Restart display manager to create desktop session"
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

// getAutoLoginUser reads the GDM auto-login configuration from /etc/gdm3/custom.conf
// Returns the configured auto-login user, or empty string if not configured
func (c *DisplayManagerCheck) getAutoLoginUser() string {
	if c.autoLoginUserProvider != nil {
		return c.autoLoginUserProvider()
	}
	return autoLoginUser()
}

// isGnomeShellRunning checks if gnome-shell is running for a specific user
// If user is empty, checks for any gnome-shell process
func (c *DisplayManagerCheck) isGnomeShellRunning(ctx context.Context, user string) bool {
	return gnomeShellRunningFor(ctx, c.executor, user)
}

// detectActiveDisplayManager finds which display manager is currently active
func (c *DisplayManagerCheck) detectActiveDisplayManager(ctx context.Context) (string, error) {
	// First check systemctl for the default display-manager target
	output, _ := c.executor.Output(ctx, "systemctl", "get-default")
	if strings.Contains(string(output), "graphical.target") {
		// Graphical target is default, look for active display manager
		for _, dm := range supportedDisplayManagers {
			if output, err := c.executor.Output(ctx, "systemctl", "is-active", dm); err == nil {
				if strings.TrimSpace(string(output)) == "active" {
					return dm, nil
				}
			}
		}
	}

	// Fallback: check which display manager service exists and is enabled
	for _, dm := range supportedDisplayManagers {
		if output, err := c.executor.Output(ctx, "systemctl", "is-enabled", dm); err == nil {
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
	output, _ := c.executor.Output(ctx, "systemctl", "is-active", service)
	return strings.TrimSpace(string(output))
}

// checkX11 tests X11 display responsiveness
func (c *DisplayManagerCheck) checkX11(ctx context.Context) (string, map[string]interface{}) {
	details := make(map[string]interface{})

	// Check if DISPLAY is set
	output, err := c.executor.Output(ctx, "printenv", "DISPLAY")
	display := strings.TrimSpace(string(output))

	if err != nil || display == "" {
		details["available"] = false
		details["reason"] = "DISPLAY environment variable not set"
		return "unavailable", details
	}

	details["available"] = true
	details["display"] = display

	// Try xdpyinfo to check X11 responsiveness
	_, err = c.executor.Output(ctx, "xdpyinfo")
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

	// Determine if gnome-shell is not running (for contextual action availability)
	gnomeShellNotRunning := false
	if lastResult != nil {
		if running, ok := lastResult.Details["gnomeShellRunning"].(bool); ok && !running {
			gnomeShellNotRunning = true
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "restart",
			Name:        "Restart Display Manager",
			Description: "Restart " + dmName + " to recreate desktop session (triggers auto-login if configured)",
			Dangerous:   true, // Restarting DM disconnects users
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "recover-session",
			Name:        "Recover Desktop Session",
			Description: "Restart display manager to restore the desktop session",
			Dangerous:   true,
			Available:   isLinux && hasSystemd && gnomeShellNotRunning,
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
		{
			ID:          "diagnose",
			Name:        "Diagnose Session",
			Description: "Check loginctl sessions and gnome-shell process status",
			Dangerous:   false,
			Available:   isLinux,
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
	case "restart", "recover-session":
		// Both actions restart the display manager to recover desktop session
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", dmName)
		result.Output = string(output)

		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart " + dmName
			return result
		}

		// Wait for display manager to come back and auto-login to create session
		// This is important because GDM restart + auto-login + gnome-shell startup takes time
		autoLoginUser := c.getAutoLoginUser()
		if autoLoginUser != "" {
			result.Output += "\nWaiting for auto-login session to initialize..."

			// Wait up to 30 seconds for gnome-shell to start
			maxWait := 30
			for i := 0; i < maxWait; i++ {
				time.Sleep(time.Second)
				if c.isGnomeShellRunning(ctx, autoLoginUser) {
					result.Output += "\ngnome-shell started for " + autoLoginUser
					break
				}
				if i == maxWait-1 {
					result.Output += fmt.Sprintf("\nWARNING: gnome-shell did not start within %d seconds", maxWait)
				}
			}
		}

		result.Duration = time.Since(start)

		// Final verification
		gnomeShellRunning := false
		if autoLoginUser != "" {
			gnomeShellRunning = c.isGnomeShellRunning(ctx, autoLoginUser)
		} else {
			gnomeShellRunning = c.isGnomeShellRunning(ctx, "")
		}

		if gnomeShellRunning {
			result.Success = true
			result.Message = dmName + " restarted - desktop session restored"
		} else {
			result.Success = false
			result.Message = dmName + " restarted but desktop session not fully recovered"
		}
		return result

	case "status":
		output, _ := c.executor.CombinedOutput(ctx, "systemctl", "status", dmName)
		result.Duration = time.Since(start)
		result.Output = string(output)
		result.Success = true
		result.Message = "Retrieved " + dmName + " status"
		return result

	case "logs":
		output, _ := journal.NewReader(c.executor).Tail(ctx, journal.QueryOpts{
			Unit: []string{dmName},
			Tail: 100,
		})
		result.Duration = time.Since(start)
		result.Output = string(output)
		result.Success = true
		result.Message = "Retrieved " + dmName + " logs"
		return result

	case "diagnose":
		var diag strings.Builder
		diag.WriteString("=== Display Manager Status ===\n")
		if output, err := c.executor.CombinedOutput(ctx, "systemctl", "is-active", dmName); err == nil {
			diag.WriteString(dmName + " service: " + strings.TrimSpace(string(output)) + "\n")
		}

		diag.WriteString("\n=== Auto-Login Configuration ===\n")
		autoLoginUser := c.getAutoLoginUser()
		if autoLoginUser != "" {
			diag.WriteString("Auto-login user: " + autoLoginUser + "\n")
		} else {
			diag.WriteString("Auto-login: not configured\n")
		}

		diag.WriteString("\n=== GNOME Shell Status ===\n")
		if output, err := c.executor.CombinedOutput(ctx, "pgrep", "-a", "gnome-shell"); err == nil {
			diag.WriteString(string(output))
		} else {
			diag.WriteString("gnome-shell: NOT RUNNING\n")
		}

		diag.WriteString("\n=== Login Sessions ===\n")
		if output, err := c.executor.CombinedOutput(ctx, "loginctl", "list-sessions", "--no-legend"); err == nil {
			diag.WriteString(string(output))
		}

		diag.WriteString("\n=== Seat Assignment ===\n")
		if user := seat0SessionUser(ctx, c.executor); user != "" {
			diag.WriteString("seat0 active session user: " + user + "\n")
		} else {
			diag.WriteString("seat0: no active session\n")
		}

		result.Duration = time.Since(start)
		result.Output = diag.String()
		result.Success = true
		result.Message = "Diagnostic information collected"
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
