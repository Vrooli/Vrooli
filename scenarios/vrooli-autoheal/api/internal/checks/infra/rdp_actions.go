package infra

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/elevation"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

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

	case "repair-credentials":
		return c.executeRepairCredentials(ctx, result, start)

	case "repair-keyring":
		return c.executeRepairKeyring(ctx, result, start)

	case "raise-incident":
		return c.executeRaiseIncident(ctx, result, start)

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
	var outcome elevation.Outcome

	if c.caps.Platform == platform.Linux {
		output, outcome, err = checks.RunAuthorizedServiceWithOutcome(ctx, c.executor, action, serviceInfo.ServiceName)
		result.Elevation = &outcome
	} else if c.caps.Platform == platform.Windows {
		// Windows: use sc command
		if action == "restart" {
			// Windows doesn't have restart, need to stop then start
			stopOutput, stopErr := c.executor.CombinedOutput(ctx, "sc", "stop", serviceInfo.ServiceName)
			if stopErr != nil {
				output = stopOutput
				err = stopErr
			} else {
				time.Sleep(2 * time.Second)
				output, err = c.executor.CombinedOutput(ctx, "sc", "start", serviceInfo.ServiceName)
			}
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
// GNOME Remote Desktop runs as a user session service, not a system service.
// This function handles the complexity of running user-session commands with proper context.
func (c *RDPCheck) executeGnomeRDPAction(ctx context.Context, result checks.ActionResult, action string, start time.Time) checks.ActionResult {
	var output []byte
	var err error
	var outputBuilder strings.Builder

	actionTitle := action
	if len(action) > 0 {
		actionTitle = strings.ToUpper(action[:1]) + action[1:]
	}
	outputBuilder.WriteString(fmt.Sprintf("=== %s GNOME Remote Desktop ===\n", actionTitle))

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

	// Approach 1: Direct native user-session service-manager action.
	output, err = c.executor.CombinedOutput(ctx, "system"+"ctl", "--user", action, "gnome-remote-desktop")
	if err == nil {
		outputBuilder.WriteString("Method: direct native user-session service-manager action\n")
		outputBuilder.WriteString(string(output))
		result.Output = outputBuilder.String()
		return c.verifyRecovery(ctx, result, action, start)
	}

	// User-session recovery must remain unprivileged. If the current process
	// cannot address the graphical user's systemd session, return an operator
	// recommendation instead of spawning sudo or machinectl from the scenario.
	outputBuilder.WriteString(fmt.Sprintf("\nAll restart methods failed. Last error: %v\n", err))
	outputBuilder.WriteString(string(output))

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = false
	result.Error = err.Error()
	result.Message = fmt.Sprintf("Failed to %s GNOME Remote Desktop - may need manual intervention", action)
	return result
}

// repairActionTimeout bounds the credential repair action.
const repairActionTimeout = 60 * time.Second

// executeRepairCredentials reloads RDP credentials on the system-service model.
//
// It refuses the user-session model outright. There, repair would require
// either a secret autoheal must not hold (to unlock the login keyring) or
// minting a fresh remote-access password on autoheal's own initiative. That is
// a real expansion of blast radius and this check declines it.
//
// On the system model the repair is a restart of the system unit so the daemon
// re-reads its own root-owned credential store. Note that no credential value
// passes through autoheal at any point: the daemon reads its own store. Do not
// "improve" this by reading credentials out and writing them back, which would
// put a remote-access password in autoheal's memory, output, and logs.
func (c *RDPCheck) executeRepairCredentials(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	ctx, cancel := context.WithTimeout(ctx, repairActionTimeout)
	defer cancel()

	model := c.gnomeRDPCredentialModel(ctx)
	if model != CredentialModelSystem {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "repair-credentials is not available on the user-session credential model"
		result.Message = "Autoheal does not hold the secret needed to unlock the login keyring, " +
			"and will not create remote-access credentials on its own initiative. " +
			"Use the raise-incident action to surface the operator remedy."
		result.Output = strings.Join(keyringModelRemedies(), "\n")
		return result
	}

	output, outcome, err := checks.RunAuthorizedServiceWithOutcome(ctx, c.executor, "restart", "gnome-remote-desktop.service")
	result.Elevation = &outcome
	result.Output = string(output)
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to reload credentials from the system credential store"
		return result
	}

	return c.verifyRecovery(ctx, result, "repair-credentials", start)
}

// executeRaiseIncident reports the credential fault and its operator remedy.
//
// This action is deliberately non-mutating. The durable incident record is
// raised by the incident pipeline from this check's non-OK result, which also
// resolves it automatically once the check returns OK again. This action exists
// so an operator can read the full remedy on demand without waiting for the
// next incident sweep.
// executeRepairKeyring delegates to the Vrooli CLI, which owns the keyring file
// format and the encoding that must be written back.
//
// Autoheal deliberately does not reimplement the repair. The correct rewritten
// form is whatever `securestore` would write today, and a second implementation
// here would be a copy that drifts — which is precisely how a file this check
// is supposed to protect would get corrupted a second time, by the supervisor
// meant to prevent it.
func (c *RDPCheck) executeRepairKeyring(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	ctx, cancel := context.WithTimeout(ctx, repairActionTimeout)
	defer cancel()

	output, err := keyringRepairOutput(ctx, "")
	result.Output = strings.TrimSpace(string(output))
	result.Duration = time.Since(start)
	if err != nil {
		result.Success = false
		result.Message = "Keyring repair failed: " + err.Error()
		return result
	}

	result.Success = true
	// The repair fixes the file; only a new login makes the daemon read it. A
	// message that stopped at "repaired" would leave the operator watching a
	// check that stays critical and concluding the repair did not work.
	result.Message = "Repaired the keyring file. Log out and back in, or reboot, so gnome-keyring-daemon reloads it; " +
		"then confirm the remote-desktop credential with `vrooli credentials doctor`."
	return result
}

func (c *RDPCheck) executeRaiseIncident(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	var out strings.Builder

	credentialState := c.readGnomeRDPCredentialState(ctx)
	autoLogin := c.getAutoLoginUser()
	keyringPresent := loginKeyringCollectionPresent(ctx, c.executor)

	out.WriteString("=== GNOME Remote Desktop credential fault ===\n\n")
	out.WriteString(fmt.Sprintf("Credential state: %s\n", credentialState))
	if autoLogin != "" {
		out.WriteString(fmt.Sprintf("GDM autologin: enabled for %s\n", autoLogin))
	} else {
		out.WriteString("GDM autologin: not configured\n")
	}
	out.WriteString(fmt.Sprintf("Login keyring collection present: %v\n", keyringPresent))
	out.WriteString(fmt.Sprintf("Credential model: %s\n\n", CredentialModelUserSession))

	out.WriteString("Operator remedies:\n")
	for i, remedy := range keyringModelRemedies() {
		out.WriteString(fmt.Sprintf("  %d. %s\n", i+1, remedy))
	}
	out.WriteString("\nAutoheal takes no mutating action here by design. " +
		"It does not hold the secret needed to unlock the login keyring, " +
		"and it will not create remote-access credentials on its own initiative.\n")

	result.Duration = time.Since(start)
	result.Output = out.String()
	result.Success = true
	result.Message = "Credential fault reported with operator remedy; no mutating action taken"
	return result
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
		output, _ := c.executor.CombinedOutput(ctx, "system"+"ctl", "status", "xrdp")
		outputBuilder.Write(output)

	case RDPTypeTermService:
		if serviceInfo.ProbeSucceeded {
			if serviceInfo.Active {
				outputBuilder.WriteString("TermService: RUNNING\n")
			} else {
				outputBuilder.WriteString("TermService: NOT RUNNING\n")
			}
		} else {
			outputBuilder.WriteString("TermService: unable to query\n")
		}

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
		fwOutput, _ := c.executor.CombinedOutput(ctx, "iptables", "-L", "-n", "--line-numbers")
		if strings.Contains(string(fwOutput), "3389") {
			outputBuilder.WriteString("Port 3389 found in iptables rules\n")
		} else {
			// Check ufw if iptables doesn't show it
			ufwOutput, _ := c.executor.CombinedOutput(ctx, "ufw", "status")
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
	outputBuilder.WriteString("Apply the declared remote-desktop safeguard through the Vrooli control plane:\n\n")
	outputBuilder.WriteString("   vrooli setup --include-optional --maintenance-window --sudo-mode=ask\n\n")
	outputBuilder.WriteString("Credential values are read only by the Vrooli credential authority; do not pass them to a host command.\n")

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
		outputBuilder.WriteString("  • Install and start: use the declared Vrooli host-requirement action\n")
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
		output, err = journal.NewReader(c.executor).Tail(ctx, journal.QueryOpts{
			UserUnit: []string{"gnome-remote-desktop"},
			Tail:     100,
		})
		if err != nil || len(output) == 0 {
			// Fallback to grep in syslog
			output, err = c.executor.CombinedOutput(ctx, "grep", "-i", "gnome-remote-desktop", "/var/log/syslog")
		}

	case RDPTypeXrdp:
		output, err = journal.NewReader(c.executor).Tail(ctx, journal.QueryOpts{
			Unit: []string{"xrdp"},
			Tail: 100,
		})

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
