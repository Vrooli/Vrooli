// Package infra provides infrastructure health checks
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// RDPType identifies which RDP implementation is in use
type RDPType string

const (
	RDPTypeXrdp        RDPType = "xrdp"
	RDPTypeGnome       RDPType = "gnome-remote-desktop"
	RDPTypeTermService RDPType = "TermService"
	RDPTypeUnknown     RDPType = "unknown"
)

// RDPServiceInfo describes which RDP service to check on a given platform
type RDPServiceInfo struct {
	ServiceName string
	Type        RDPType
	Checkable   bool
	// IsUserSession indicates if the RDP runs as a user session daemon (not systemd)
	IsUserSession bool
}

// SelectRDPService is a static helper that determines which RDP service to check
// based on platform capabilities WITHOUT runtime detection.
// This is used for tests and static analysis. For actual runtime detection
// (which checks for running processes), use RDPCheck.detectRDPService().
//
// Decision logic:
//   - Linux with systemd → xrdp service (static fallback)
//   - Linux without systemd → not checkable
//   - Windows → TermService (built-in RDP)
//   - Other platforms → not checkable
func SelectRDPService(caps *platform.Capabilities) RDPServiceInfo {
	switch caps.Platform {
	case platform.Linux:
		if caps.SupportsSystemd {
			// Note: Runtime detection in detectRDPService prefers GNOME RDP if running
			return RDPServiceInfo{
				ServiceName:   "xrdp",
				Type:          RDPTypeXrdp,
				Checkable:     true,
				IsUserSession: false,
			}
		}
		return RDPServiceInfo{
			ServiceName: "xrdp",
			Type:        RDPTypeXrdp,
			Checkable:   false,
		}
	case platform.Windows:
		return RDPServiceInfo{
			ServiceName:   "TermService",
			Type:          RDPTypeTermService,
			Checkable:     true,
			IsUserSession: false,
		}
	default:
		return RDPServiceInfo{Checkable: false}
	}
}

// RDPCheck verifies RDP service (xrdp, GNOME Remote Desktop, or Windows TermService).
// Platform capabilities are injected to avoid hidden dependencies and enable testing.
type RDPCheck struct {
	caps     *platform.Capabilities
	executor checks.CommandExecutor
	// cachedServiceInfo stores detected RDP service info to avoid repeated detection
	cachedServiceInfo *RDPServiceInfo
}

// RDPCheckOption configures an RDPCheck.
type RDPCheckOption func(*RDPCheck)

// WithRDPExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithRDPExecutor(executor checks.CommandExecutor) RDPCheckOption {
	return func(c *RDPCheck) {
		c.executor = executor
	}
}

// NewRDPCheck creates an RDP health check with injected platform capabilities.
func NewRDPCheck(caps *platform.Capabilities, opts ...RDPCheckOption) *RDPCheck {
	c := &RDPCheck{
		caps:     caps,
		executor: checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// detectRDPService determines which RDP implementation is available on this system.
// Detection order on Linux:
//  1. Check if GNOME Remote Desktop is CONFIGURED (via grdctl) - catches crashed daemons
//  2. Check for running gnome-remote-desktop-daemon process (GNOME 42+ native RDP)
//  3. Check for xrdp systemd service
//  4. Fall back to unknown if neither is found
func (c *RDPCheck) detectRDPService(ctx context.Context) RDPServiceInfo {
	switch c.caps.Platform {
	case platform.Linux:
		// First check if GNOME Remote Desktop is CONFIGURED (not just running)
		// This catches the case where RDP is enabled but daemon has crashed
		if c.isGnomeRDPConfigured(ctx) {
			return RDPServiceInfo{
				ServiceName:   "gnome-remote-desktop",
				Type:          RDPTypeGnome,
				Checkable:     true,
				IsUserSession: true,
			}
		}

		// Then check for xrdp (traditional systemd service)
		if c.caps.SupportsSystemd {
			// Check if xrdp is installed/available
			output, err := c.executor.Output(ctx, "systemctl", "list-unit-files", "xrdp.service")
			if err == nil && strings.Contains(string(output), "xrdp.service") {
				return RDPServiceInfo{
					ServiceName:   "xrdp",
					Type:          RDPTypeXrdp,
					Checkable:     true,
					IsUserSession: false,
				}
			}
		}

		// No RDP service detected - this is OK, not all systems need RDP
		return RDPServiceInfo{
			Type:      RDPTypeUnknown,
			Checkable: false,
		}

	case platform.Windows:
		return RDPServiceInfo{
			ServiceName:   "TermService",
			Type:          RDPTypeTermService,
			Checkable:     true,
			IsUserSession: false,
		}

	default:
		return RDPServiceInfo{Checkable: false}
	}
}

// isGnomeRDPRunning checks if gnome-remote-desktop-daemon is running
func (c *RDPCheck) isGnomeRDPRunning(ctx context.Context) bool {
	// Check for the process using pgrep
	output, err := c.executor.Output(ctx, "pgrep", "-f", "gnome-remote-desktop-daemon")
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		return true
	}
	return false
}

// isGnomeRDPConfigured checks if GNOME Remote Desktop is enabled in settings
// using grdctl status. This detects configuration even when the daemon isn't running.
func (c *RDPCheck) isGnomeRDPConfigured(ctx context.Context) bool {
	output, err := c.executor.Output(ctx, "grdctl", "status")
	if err != nil {
		return false
	}
	// grdctl status shows "Status: enabled" when RDP is configured
	return strings.Contains(string(output), "Status: enabled")
}

func (c *RDPCheck) ID() string    { return "infra-rdp" }
func (c *RDPCheck) Title() string { return "Remote Desktop" }
func (c *RDPCheck) Description() string {
	return "Checks RDP service (GNOME Remote Desktop, xrdp, or Windows TermService)"
}
func (c *RDPCheck) Importance() string {
	return "Required for remote desktop access to this machine"
}
func (c *RDPCheck) Category() checks.Category { return checks.CategoryInfrastructure }
func (c *RDPCheck) IntervalSeconds() int      { return 60 }
func (c *RDPCheck) Platforms() []platform.Type {
	return []platform.Type{platform.Linux, platform.Windows}
}

func (c *RDPCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Detect which RDP service is available on this system
	serviceInfo := c.detectRDPService(ctx)
	result.Details["service"] = serviceInfo.ServiceName
	result.Details["type"] = string(serviceInfo.Type)
	result.Details["isUserSession"] = serviceInfo.IsUserSession

	if !serviceInfo.Checkable {
		// No RDP service detected - this is informational, not a failure
		result.Status = checks.StatusOK
		result.Message = "No RDP service installed (remote desktop not configured)"
		result.Details["note"] = "RDP is optional; install xrdp or enable GNOME Remote Desktop if needed"
		return result
	}

	// Cache the service info for recovery actions
	c.cachedServiceInfo = &serviceInfo

	// Execute type-specific service check
	switch serviceInfo.Type {
	case RDPTypeGnome:
		return c.checkGnomeRDP(ctx, result)
	case RDPTypeXrdp:
		return c.checkLinuxXRDP(ctx, result)
	case RDPTypeTermService:
		return c.checkWindowsTermService(ctx, result)
	default:
		result.Status = checks.StatusOK
		result.Message = "No RDP service installed"
		return result
	}
}

// checkGnomeRDP verifies GNOME Remote Desktop daemon is running
func (c *RDPCheck) checkGnomeRDP(ctx context.Context, result checks.Result) checks.Result {
	isRunning := c.isGnomeRDPRunning(ctx)
	result.Details["configured"] = true
	result.Details["running"] = isRunning

	if isRunning {
		result.Status = checks.StatusOK
		result.Message = "GNOME Remote Desktop is running"
		result.Details["status"] = "active"

		// Get additional info about the daemon
		output, _ := c.executor.Output(ctx, "pgrep", "-a", "-f", "gnome-remote-desktop-daemon")
		if len(output) > 0 {
			result.Details["processInfo"] = strings.TrimSpace(string(output))
		}

		return result
	}

	// Configured but NOT running - this is a problem that needs attention
	result.Status = checks.StatusWarning
	result.Message = "GNOME Remote Desktop is configured but not running"
	result.Details["status"] = "inactive"
	result.Details["note"] = "Service may have crashed or been stopped; auto-heal can restart it"
	return result
}

// checkLinuxXRDP checks xrdp service status on Linux systems with systemd
func (c *RDPCheck) checkLinuxXRDP(ctx context.Context, result checks.Result) checks.Result {
	output, err := c.executor.Output(ctx, "systemctl", "is-active", "xrdp")
	status := strings.TrimSpace(string(output))
	result.Details["status"] = status

	if err != nil || status != "active" {
		result.Status = checks.StatusWarning
		result.Message = "xrdp service not active"
		return result
	}
	result.Status = checks.StatusOK
	result.Message = "xrdp is running"
	return result
}

// checkWindowsTermService checks TermService status on Windows
func (c *RDPCheck) checkWindowsTermService(ctx context.Context, result checks.Result) checks.Result {
	output, err := c.executor.Output(ctx, "sc", "query", "TermService")

	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Unable to check RDP service"
		return result
	}

	// Decision: Windows sc query returns "STATE : X RUNNING" when service is running
	if strings.Contains(string(output), "RUNNING") {
		result.Status = checks.StatusOK
		result.Message = "RDP service is running"
	} else {
		result.Status = checks.StatusWarning
		result.Message = "RDP service not running"
	}
	return result
}

// RecoveryActions returns available recovery actions for RDP service issues
// [REQ:HEAL-ACTION-001]
func (c *RDPCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	// Use cached service info if available, otherwise detect
	var serviceInfo RDPServiceInfo
	if c.cachedServiceInfo != nil {
		serviceInfo = *c.cachedServiceInfo
	} else {
		serviceInfo = c.detectRDPService(context.Background())
	}

	isRunning := false
	if lastResult != nil {
		if status, ok := lastResult.Details["status"].(string); ok {
			isRunning = status == "active" || strings.Contains(status, "RUNNING")
		}
	}

	// Actions differ based on RDP type
	switch serviceInfo.Type {
	case RDPTypeGnome:
		// GNOME Remote Desktop is a user session daemon - we can still restart it via systemctl --user
		return []checks.RecoveryAction{
			{
				ID:          "start",
				Name:        "Start Service",
				Description: "Start the GNOME Remote Desktop user session service",
				Dangerous:   false,
				Available:   !isRunning,
			},
			{
				ID:          "restart",
				Name:        "Restart Service",
				Description: "Restart the GNOME Remote Desktop user session service",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "status",
				Name:        "Check Status",
				Description: "Get detailed GNOME Remote Desktop status",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "diagnose",
				Name:        "Diagnose",
				Description: "Gather diagnostic information about GNOME Remote Desktop",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "logs",
				Name:        "View Logs",
				Description: "View recent GNOME Remote Desktop logs",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "open-settings",
				Name:        "Open Settings",
				Description: "Show command to open GNOME Remote Desktop settings",
				Dangerous:   false,
				Available:   true,
			},
		}

	case RDPTypeXrdp:
		return []checks.RecoveryAction{
			{
				ID:          "start",
				Name:        "Start Service",
				Description: "Start the xrdp service",
				Dangerous:   false,
				Available:   !isRunning,
			},
			{
				ID:          "restart",
				Name:        "Restart Service",
				Description: "Restart the xrdp service",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "status",
				Name:        "Service Status",
				Description: "Get detailed xrdp service status",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "logs",
				Name:        "View Logs",
				Description: "View recent xrdp logs",
				Dangerous:   false,
				Available:   true,
			},
		}

	case RDPTypeTermService:
		return []checks.RecoveryAction{
			{
				ID:          "start",
				Name:        "Start Service",
				Description: "Start the Windows RDP service",
				Dangerous:   false,
				Available:   !isRunning,
			},
			{
				ID:          "restart",
				Name:        "Restart Service",
				Description: "Restart the Windows RDP service",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "status",
				Name:        "Service Status",
				Description: "Get detailed RDP service status",
				Dangerous:   false,
				Available:   true,
			},
		}

	default:
		// No RDP detected - provide informational action
		return []checks.RecoveryAction{
			{
				ID:          "install-info",
				Name:        "Installation Info",
				Description: "Show how to install RDP on this system",
				Dangerous:   false,
				Available:   true,
			},
		}
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *RDPCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Use cached service info if available, otherwise detect
	var serviceInfo RDPServiceInfo
	if c.cachedServiceInfo != nil {
		serviceInfo = *c.cachedServiceInfo
	} else {
		serviceInfo = c.detectRDPService(ctx)
	}

	// Handle actions based on RDP type
	switch actionID {
	case "status":
		return c.executeStatus(ctx, result, serviceInfo, start)

	case "diagnose":
		return c.executeDiagnose(ctx, result, serviceInfo, start)

	case "open-settings":
		return c.executeOpenSettings(ctx, result, start)

	case "install-info":
		return c.executeInstallInfo(ctx, result, start)

	case "start":
		if serviceInfo.Type == RDPTypeGnome {
			return c.executeGnomeRDPAction(ctx, result, "start", start)
		}
		return c.executeServiceAction(ctx, result, "start", serviceInfo, start)

	case "restart":
		if serviceInfo.Type == RDPTypeGnome {
			return c.executeGnomeRDPAction(ctx, result, "restart", start)
		}
		return c.executeServiceAction(ctx, result, "restart", serviceInfo, start)

	case "logs":
		return c.executeLogs(ctx, result, serviceInfo, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeServiceAction starts or restarts the RDP service
func (c *RDPCheck) executeServiceAction(ctx context.Context, result checks.ActionResult, action string, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var output []byte
	var err error

	if c.caps.Platform == platform.Linux {
		output, err = c.executor.CombinedOutput(ctx, "sudo", "systemctl", action, serviceInfo.ServiceName)
	} else if c.caps.Platform == platform.Windows {
		// Windows: use sc command
		if action == "restart" {
			// Windows doesn't have restart, need to stop then start
			c.executor.CombinedOutput(ctx, "sc", "stop", serviceInfo.ServiceName)
			time.Sleep(2 * time.Second)
			output, err = c.executor.CombinedOutput(ctx, "sc", "start", serviceInfo.ServiceName)
		} else {
			output, err = c.executor.CombinedOutput(ctx, "sc", action, serviceInfo.ServiceName)
		}
	}

	result.Output = string(output)

	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to " + action + " " + serviceInfo.ServiceName
		return result
	}

	// Verify service is running
	return c.verifyRecovery(ctx, result, action, start)
}

// executeGnomeRDPAction starts or restarts GNOME Remote Desktop user session service.
// GNOME Remote Desktop runs as a user session service (systemctl --user), not a system service.
// This function handles the complexity of running user-session commands with proper context.
func (c *RDPCheck) executeGnomeRDPAction(ctx context.Context, result checks.ActionResult, action string, start time.Time) checks.ActionResult {
	var output []byte
	var err error
	var outputBuilder strings.Builder

	outputBuilder.WriteString(fmt.Sprintf("=== %s GNOME Remote Desktop ===\n", strings.Title(action)))

	// GNOME Remote Desktop is a user session service.
	// We need to determine who owns the graphical session and run as that user.

	// First, try to find the active graphical session user
	sessionUser := c.findGraphicalSessionUser(ctx)
	if sessionUser == "" {
		// Fallback to current user or SUDO_USER
		sessionUser = os.Getenv("SUDO_USER")
		if sessionUser == "" {
			sessionUser = os.Getenv("USER")
		}
	}

	outputBuilder.WriteString(fmt.Sprintf("Target user: %s\n", sessionUser))

	// Get the user's UID for XDG_RUNTIME_DIR
	uidOutput, uidErr := c.executor.Output(ctx, "id", "-u", sessionUser)
	uid := strings.TrimSpace(string(uidOutput))
	if uidErr != nil || uid == "" {
		uid = "1000" // Common default for first user
	}
	xdgRuntimeDir := fmt.Sprintf("/run/user/%s", uid)

	outputBuilder.WriteString(fmt.Sprintf("XDG_RUNTIME_DIR: %s\n", xdgRuntimeDir))

	// Try different approaches to restart the service

	// Approach 1: Direct systemctl --user (works if we're already the correct user)
	output, err = c.executor.CombinedOutput(ctx, "systemctl", "--user", action, "gnome-remote-desktop")
	if err == nil {
		outputBuilder.WriteString("Method: direct systemctl --user\n")
		outputBuilder.WriteString(string(output))
		result.Output = outputBuilder.String()
		return c.verifyRecovery(ctx, result, action, start)
	}

	// Approach 2: Use sudo -u with XDG_RUNTIME_DIR and DBUS_SESSION_BUS_ADDRESS
	dbusAddr := fmt.Sprintf("unix:path=%s/bus", xdgRuntimeDir)
	output, err = c.executor.CombinedOutput(ctx,
		"sudo", "-u", sessionUser,
		fmt.Sprintf("XDG_RUNTIME_DIR=%s", xdgRuntimeDir),
		fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=%s", dbusAddr),
		"systemctl", "--user", action, "gnome-remote-desktop")

	if err == nil {
		outputBuilder.WriteString("Method: sudo -u with environment\n")
		outputBuilder.WriteString(string(output))
		result.Output = outputBuilder.String()
		return c.verifyRecovery(ctx, result, action, start)
	}

	// Approach 3: Use machinectl to enter user session (more robust for some setups)
	output, err = c.executor.CombinedOutput(ctx,
		"sudo", "machinectl", "shell", fmt.Sprintf("%s@.host", sessionUser),
		"/bin/systemctl", "--user", action, "gnome-remote-desktop")

	if err == nil {
		outputBuilder.WriteString("Method: machinectl shell\n")
		outputBuilder.WriteString(string(output))
		result.Output = outputBuilder.String()
		return c.verifyRecovery(ctx, result, action, start)
	}

	// All approaches failed
	outputBuilder.WriteString(fmt.Sprintf("\nAll restart methods failed. Last error: %v\n", err))
	outputBuilder.WriteString(string(output))

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = false
	result.Error = err.Error()
	result.Message = fmt.Sprintf("Failed to %s GNOME Remote Desktop - may need manual intervention", action)
	return result
}

// findGraphicalSessionUser finds the user who owns the active graphical session
func (c *RDPCheck) findGraphicalSessionUser(ctx context.Context) string {
	// Try to find the active session on seat0 (the main display)
	output, err := c.executor.Output(ctx, "loginctl", "show-seat", "seat0", "-p", "ActiveSession", "--value")
	if err != nil {
		return ""
	}

	sessionID := strings.TrimSpace(string(output))
	if sessionID == "" {
		return ""
	}

	// Get the user for this session
	output, err = c.executor.Output(ctx, "loginctl", "show-session", sessionID, "-p", "Name", "--value")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// executeStatus gets detailed service status
func (c *RDPCheck) executeStatus(ctx context.Context, result checks.ActionResult, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	switch serviceInfo.Type {
	case RDPTypeGnome:
		outputBuilder.WriteString("=== GNOME Remote Desktop Status ===\n")

		// Check process status
		output, err := c.executor.CombinedOutput(ctx, "pgrep", "-a", "-f", "gnome-remote-desktop-daemon")
		if err == nil && len(output) > 0 {
			outputBuilder.WriteString("Process: RUNNING\n")
			outputBuilder.WriteString(string(output))
		} else {
			outputBuilder.WriteString("Process: NOT RUNNING\n")
		}

		// Check port 3389
		outputBuilder.WriteString("\n=== Port 3389 Status ===\n")
		portOutput, _ := c.executor.CombinedOutput(ctx, "ss", "-tlnp", "sport = :3389")
		if len(portOutput) > 0 {
			outputBuilder.Write(portOutput)
		} else {
			outputBuilder.WriteString("No listener on port 3389\n")
		}

	case RDPTypeXrdp:
		output, _ := c.executor.CombinedOutput(ctx, "systemctl", "status", "xrdp")
		outputBuilder.Write(output)

	case RDPTypeTermService:
		output, _ := c.executor.CombinedOutput(ctx, "sc", "query", "TermService")
		outputBuilder.Write(output)

	default:
		outputBuilder.WriteString("No RDP service detected on this system.\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Service status retrieved"
	return result
}

// executeDiagnose gathers diagnostic information about RDP
func (c *RDPCheck) executeDiagnose(ctx context.Context, result checks.ActionResult, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== RDP Diagnostics ===\n")
	outputBuilder.WriteString(fmt.Sprintf("Detected RDP Type: %s\n", serviceInfo.Type))
	outputBuilder.WriteString(fmt.Sprintf("Service Name: %s\n", serviceInfo.ServiceName))
	outputBuilder.WriteString(fmt.Sprintf("User Session: %v\n\n", serviceInfo.IsUserSession))

	// Check port 3389 listener
	outputBuilder.WriteString("=== Port 3389 Status ===\n")
	portOutput, _ := c.executor.CombinedOutput(ctx, "ss", "-tlnp", "sport = :3389")
	if len(portOutput) > 0 {
		outputBuilder.Write(portOutput)
	} else {
		outputBuilder.WriteString("No listener on port 3389\n")
	}

	// Network interfaces
	outputBuilder.WriteString("\n=== Network Interfaces ===\n")
	ifOutput, _ := c.executor.CombinedOutput(ctx, "ip", "addr", "show")
	if len(ifOutput) > 0 {
		// Just show the first few interfaces
		lines := strings.Split(string(ifOutput), "\n")
		for i, line := range lines {
			if i >= 20 {
				outputBuilder.WriteString("...(truncated)\n")
				break
			}
			outputBuilder.WriteString(line + "\n")
		}
	}

	// Firewall status
	outputBuilder.WriteString("\n=== Firewall Port 3389 ===\n")
	if c.caps.Platform == platform.Linux {
		fwOutput, _ := c.executor.CombinedOutput(ctx, "sudo", "iptables", "-L", "-n", "--line-numbers")
		if strings.Contains(string(fwOutput), "3389") {
			outputBuilder.WriteString("Port 3389 found in iptables rules\n")
		} else {
			// Check ufw if iptables doesn't show it
			ufwOutput, _ := c.executor.CombinedOutput(ctx, "sudo", "ufw", "status")
			if strings.Contains(string(ufwOutput), "3389") {
				outputBuilder.WriteString("Port 3389 found in UFW rules\n")
			} else {
				outputBuilder.WriteString("Port 3389 not explicitly allowed (may use default policy)\n")
			}
		}
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Diagnostic information gathered"
	return result
}

// executeOpenSettings provides information about opening RDP settings
func (c *RDPCheck) executeOpenSettings(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== GNOME Remote Desktop Settings ===\n\n")
	outputBuilder.WriteString("To configure GNOME Remote Desktop:\n\n")
	outputBuilder.WriteString("1. Open GNOME Settings:\n")
	outputBuilder.WriteString("   gnome-control-center sharing\n\n")
	outputBuilder.WriteString("2. Or navigate to:\n")
	outputBuilder.WriteString("   Settings → Sharing → Remote Desktop\n\n")
	outputBuilder.WriteString("3. Enable 'Remote Desktop' toggle\n")
	outputBuilder.WriteString("4. Configure username and password\n")
	outputBuilder.WriteString("5. Optionally enable 'Remote Control' for input\n")

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Settings information provided"
	return result
}

// executeInstallInfo provides information about installing RDP
func (c *RDPCheck) executeInstallInfo(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== RDP Installation Options ===\n\n")

	if c.caps.Platform == platform.Linux {
		outputBuilder.WriteString("Option 1: GNOME Remote Desktop (recommended for GNOME desktop)\n")
		outputBuilder.WriteString("  • Built into GNOME 42+\n")
		outputBuilder.WriteString("  • Enable in Settings → Sharing → Remote Desktop\n\n")

		outputBuilder.WriteString("Option 2: xrdp (traditional RDP server)\n")
		outputBuilder.WriteString("  • Install: sudo apt install xrdp\n")
		outputBuilder.WriteString("  • Start: sudo systemctl enable --now xrdp\n")
		outputBuilder.WriteString("  • Note: May conflict with GNOME Remote Desktop\n")
	} else if c.caps.Platform == platform.Windows {
		outputBuilder.WriteString("Windows Remote Desktop:\n")
		outputBuilder.WriteString("  • Enable in Settings → System → Remote Desktop\n")
		outputBuilder.WriteString("  • Or: sysdm.cpl → Remote tab\n")
	} else {
		outputBuilder.WriteString("RDP is not typically available on this platform.\n")
		outputBuilder.WriteString("Consider using VNC or SSH instead.\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Installation information provided"
	return result
}

// executeLogs gets recent service logs
func (c *RDPCheck) executeLogs(ctx context.Context, result checks.ActionResult, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var output []byte
	var err error

	switch serviceInfo.Type {
	case RDPTypeGnome:
		// GNOME Remote Desktop logs to user journal
		output, err = c.executor.CombinedOutput(ctx, "journalctl", "--user", "-u", "gnome-remote-desktop", "-n", "100", "--no-pager")
		if err != nil || len(output) == 0 {
			// Fallback to grep in syslog
			output, err = c.executor.CombinedOutput(ctx, "grep", "-i", "gnome-remote-desktop", "/var/log/syslog")
		}

	case RDPTypeXrdp:
		output, err = c.executor.CombinedOutput(ctx, "journalctl", "-u", "xrdp", "-n", "100", "--no-pager")

	case RDPTypeTermService:
		// Windows event logs would need different approach
		output = []byte("Windows event logs require Event Viewer. Run: eventvwr.msc")
		err = nil

	default:
		output = []byte("No RDP service detected to retrieve logs from")
		err = nil
	}

	result.Duration = time.Since(start)
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to retrieve logs"
		return result
	}

	result.Success = true
	result.Message = "Service logs retrieved"
	return result
}

// verifyRecovery checks that RDP is working after a recovery action using polling
func (c *RDPCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, action string, start time.Time) checks.ActionResult {
	// Use polling to verify recovery instead of fixed sleep
	pollConfig := checks.PollConfig{
		Timeout:      30 * time.Second,
		Interval:     2 * time.Second,
		InitialDelay: 3 * time.Second, // Service needs time to initialize
	}

	pollResult := checks.PollForSuccess(ctx, c, pollConfig)
	result.Duration = time.Since(start)

	if pollResult.Success {
		result.Success = true
		result.Message = "RDP service " + action + " successful and verified running"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(verified after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	} else {
		result.Success = false
		result.Error = "RDP not running after " + action
		result.Message = "RDP service " + action + " completed but verification failed"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification Failed ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(failed after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	}

	return result
}

// Ensure RDPCheck implements HealableCheck
var _ checks.HealableCheck = (*RDPCheck)(nil)
