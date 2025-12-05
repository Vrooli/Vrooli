// Package watchdog detects and manages OS-level service installation
// for vrooli-autoheal persistence across reboots.
// [REQ:WATCH-DETECT-001] [REQ:WATCH-LINUX-001] [REQ:WATCH-MAC-001] [REQ:WATCH-WIN-001]
package watchdog

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"vrooli-autoheal/internal/platform"
)

// WatchdogType represents the type of OS-level service manager
type WatchdogType string

const (
	WatchdogTypeNone    WatchdogType = ""
	WatchdogTypeSystemd WatchdogType = "systemd"
	WatchdogTypeLaunchd WatchdogType = "launchd"
	WatchdogTypeWindows WatchdogType = "windows-task"
)

// Status represents the current state of the OS watchdog
type Status struct {
	// LoopRunning indicates if the autoheal loop process is currently active
	LoopRunning bool `json:"loopRunning"`

	// WatchdogType is the type of OS service manager (systemd, launchd, windows-task, or empty)
	WatchdogType WatchdogType `json:"watchdogType"`

	// WatchdogInstalled indicates if the OS-level service is installed
	WatchdogInstalled bool `json:"watchdogInstalled"`

	// WatchdogEnabled indicates if the service is enabled to start on boot
	WatchdogEnabled bool `json:"watchdogEnabled"`

	// WatchdogRunning indicates if the OS service manager reports the service as active
	WatchdogRunning bool `json:"watchdogRunning"`

	// BootProtectionActive is true when the service is both installed and enabled
	BootProtectionActive bool `json:"bootProtectionActive"`

	// CanInstall indicates whether the current platform supports watchdog installation
	CanInstall bool `json:"canInstall"`

	// ServicePath is the path to the service configuration file (if applicable)
	ServicePath string `json:"servicePath,omitempty"`

	// LastError contains any error encountered during detection
	LastError string `json:"lastError,omitempty"`

	// ProtectionLevel summarizes the protection state
	ProtectionLevel ProtectionLevel `json:"protectionLevel"`

	// LingeringEnabled indicates if systemd lingering is enabled for the user (Linux only)
	// When false on a user service, the service won't start at boot without a login session
	LingeringEnabled bool `json:"lingeringEnabled"`

	// Username is the current user, used for displaying fix commands in the UI
	Username string `json:"username,omitempty"`

	// IsUserService indicates if this is a user-level service (vs system-level)
	IsUserService bool `json:"isUserService,omitempty"`
}

// ProtectionLevel represents the overall protection state
type ProtectionLevel string

const (
	// ProtectionFull means watchdog is installed, enabled, and running
	ProtectionFull ProtectionLevel = "full"
	// ProtectionPartial means the loop is running but no OS watchdog
	ProtectionPartial ProtectionLevel = "partial"
	// ProtectionNone means no protection is active
	ProtectionNone ProtectionLevel = "none"
)

// Detector detects and manages OS watchdog status
type Detector struct {
	platform *platform.Capabilities
	mu       sync.RWMutex
	cached   *Status
}

// NewDetector creates a new watchdog detector
func NewDetector(plat *platform.Capabilities) *Detector {
	return &Detector{
		platform: plat,
	}
}

// Detect checks the current watchdog status
// [REQ:WATCH-DETECT-001]
func (d *Detector) Detect() *Status {
	d.mu.Lock()
	defer d.mu.Unlock()

	status := &Status{
		LoopRunning: d.isLoopRunning(),
		CanInstall:  d.canInstall(),
	}

	// Detect based on platform
	switch d.platform.Platform {
	case "linux":
		d.detectLinux(status)
	case "macos":
		d.detectMacOS(status)
	case "windows":
		d.detectWindows(status)
	default:
		status.LastError = "unsupported platform for watchdog"
	}

	// Calculate protection level
	status.ProtectionLevel = d.calculateProtectionLevel(status)
	status.BootProtectionActive = status.WatchdogInstalled && status.WatchdogEnabled

	d.cached = status
	return status
}

// GetCached returns the last detected status without re-detecting
func (d *Detector) GetCached() *Status {
	d.mu.RLock()
	cached := d.cached
	d.mu.RUnlock()

	if cached == nil {
		return d.Detect()
	}
	return cached
}

// isLoopRunning checks if the autoheal loop is currently running
// by looking for the vrooli-autoheal CLI process running with the "loop" argument
func (d *Detector) isLoopRunning() bool {
	// Look for vrooli-autoheal loop process
	// This checks if the CLI loop is actually running (not just the API)

	switch runtime.GOOS {
	case "windows":
		// On Windows, use tasklist
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq vrooli-autoheal*")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), "vrooli-autoheal")

	default:
		// On Unix-like systems, use pgrep
		// Look for processes matching "vrooli-autoheal" with "loop" argument
		cmd := exec.Command("pgrep", "-f", "vrooli-autoheal.*loop")
		if err := cmd.Run(); err == nil {
			return true
		}

		// Fallback: check /proc directly on Linux
		if runtime.GOOS == "linux" {
			entries, err := os.ReadDir("/proc")
			if err != nil {
				return false
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				// Skip non-numeric directories
				pid := entry.Name()
				if len(pid) == 0 || pid[0] < '0' || pid[0] > '9' {
					continue
				}
				cmdline, err := os.ReadFile(filepath.Join("/proc", pid, "cmdline"))
				if err != nil {
					continue
				}
				cmdStr := string(cmdline)
				if strings.Contains(cmdStr, "vrooli-autoheal") && strings.Contains(cmdStr, "loop") {
					return true
				}
			}
		}
		return false
	}
}

// canInstall checks if the platform supports watchdog installation
func (d *Detector) canInstall() bool {
	switch d.platform.Platform {
	case "linux":
		return d.platform.SupportsSystemd
	case "macos":
		return d.platform.SupportsLaunchd
	case "windows":
		return d.platform.SupportsWindowsSvc
	default:
		return false
	}
}

// detectLinux checks systemd service status
// [REQ:WATCH-LINUX-001] [REQ:WATCH-LINUX-002]
func (d *Detector) detectLinux(status *Status) {
	if !d.platform.SupportsSystemd {
		status.LastError = "systemd not available on this Linux system"
		return
	}

	status.WatchdogType = WatchdogTypeSystemd

	// Get current user for lingering check and UI display
	currentUser, err := user.Current()
	if err == nil {
		status.Username = currentUser.Username
	}

	// Check common service file locations
	servicePaths := []string{
		"/etc/systemd/system/vrooli-autoheal.service",
		"/usr/lib/systemd/system/vrooli-autoheal.service",
		filepath.Join(os.Getenv("HOME"), ".config/systemd/user/vrooli-autoheal.service"),
	}

	for _, path := range servicePaths {
		if _, err := os.Stat(path); err == nil {
			status.WatchdogInstalled = true
			status.ServicePath = path
			break
		}
	}

	if !status.WatchdogInstalled {
		return
	}

	// Determine if this is a user service or system service
	isUserService := strings.Contains(status.ServicePath, ".config/systemd/user")
	status.IsUserService = isUserService

	// Check if service is enabled
	var cmd *exec.Cmd
	if isUserService {
		cmd = exec.Command("systemctl", "--user", "is-enabled", "vrooli-autoheal")
	} else {
		cmd = exec.Command("systemctl", "is-enabled", "vrooli-autoheal")
	}
	output, _ := cmd.Output()
	status.WatchdogEnabled = strings.TrimSpace(string(output)) == "enabled"

	// Check if service is running
	if isUserService {
		cmd = exec.Command("systemctl", "--user", "is-active", "vrooli-autoheal")
	} else {
		cmd = exec.Command("systemctl", "is-active", "vrooli-autoheal")
	}
	output, _ = cmd.Output()
	status.WatchdogRunning = strings.TrimSpace(string(output)) == "active"

	// For user services, check if lingering is enabled
	// Lingering allows user services to run at boot without a login session
	if isUserService && status.Username != "" {
		status.LingeringEnabled = d.isLingeringEnabled(status.Username)
	} else if !isUserService {
		// System services don't need lingering - they always start at boot
		status.LingeringEnabled = true
	}
}

// isLingeringEnabled checks if systemd lingering is enabled for a user
// Lingering allows user services to start at boot without requiring a login session
func (d *Detector) isLingeringEnabled(username string) bool {
	// Method 1: Check for linger file (most reliable)
	lingerPath := filepath.Join("/var/lib/systemd/linger", username)
	if _, err := os.Stat(lingerPath); err == nil {
		return true
	}

	// Method 2: Use loginctl as fallback (in case linger directory differs)
	cmd := exec.Command("loginctl", "show-user", username, "--property=Linger")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)) == "Linger=yes"
	}

	return false
}

// detectMacOS checks launchd plist status
// [REQ:WATCH-MAC-001]
func (d *Detector) detectMacOS(status *Status) {
	if !d.platform.SupportsLaunchd {
		status.LastError = "launchd not available"
		return
	}

	status.WatchdogType = WatchdogTypeLaunchd

	// Check plist locations
	homeDir, _ := os.UserHomeDir()
	plistPaths := []string{
		filepath.Join(homeDir, "Library/LaunchAgents/com.vrooli.autoheal.plist"),
		"/Library/LaunchDaemons/com.vrooli.autoheal.plist",
	}

	for _, path := range plistPaths {
		if _, err := os.Stat(path); err == nil {
			status.WatchdogInstalled = true
			status.ServicePath = path
			status.IsUserService = strings.Contains(path, "LaunchAgents")
			break
		}
	}

	if !status.WatchdogInstalled {
		return
	}

	// Check if loaded and running via launchctl list
	// Output format: "PID\tStatus\tLabel" where PID is "-" if not running
	cmd := exec.Command("launchctl", "list", "com.vrooli.autoheal")
	output, err := cmd.Output()
	if err == nil {
		// Service is loaded (enabled)
		status.WatchdogEnabled = true

		// Parse output to determine if actually running
		// Format: "PID\tStatus\tLabel" - if PID is a number, service is running
		outputStr := strings.TrimSpace(string(output))
		fields := strings.Fields(outputStr)
		if len(fields) >= 1 {
			// First field is PID - if it's a number (not "-"), service is running
			pid := fields[0]
			if pid != "-" && pid != "" {
				// Try to parse as number to confirm it's a valid PID
				if _, parseErr := fmt.Sscanf(pid, "%d", new(int)); parseErr == nil {
					status.WatchdogRunning = true
				}
			}
		}
	}
	// If launchctl list fails, service is not loaded (not enabled)
}

// detectWindows checks Windows Task Scheduler
// [REQ:WATCH-WIN-001]
func (d *Detector) detectWindows(status *Status) {
	if runtime.GOOS != "windows" {
		// Can't detect Windows services from non-Windows
		status.LastError = "Windows detection not available on this platform"
		return
	}

	status.WatchdogType = WatchdogTypeWindows

	// Check for scheduled task
	cmd := exec.Command("schtasks", "/Query", "/TN", "VrooliAutoheal")
	output, err := cmd.Output()
	if err == nil {
		status.WatchdogInstalled = true
		status.ServicePath = "Task Scheduler: VrooliAutoheal"

		// Check if task is enabled
		outputStr := string(output)
		status.WatchdogEnabled = strings.Contains(outputStr, "Ready") ||
			strings.Contains(outputStr, "Running")
		status.WatchdogRunning = strings.Contains(outputStr, "Running")
	}
}

// calculateProtectionLevel determines the overall protection level
func (d *Detector) calculateProtectionLevel(status *Status) ProtectionLevel {
	if status.WatchdogInstalled && status.WatchdogEnabled && status.WatchdogRunning {
		return ProtectionFull
	}
	if status.LoopRunning {
		return ProtectionPartial
	}
	return ProtectionNone
}

// GetServiceTemplate returns the service configuration template for the current platform
func (d *Detector) GetServiceTemplate() (string, error) {
	switch d.platform.Platform {
	case "linux":
		return d.getSystemdTemplate(), nil
	case "macos":
		return d.getLaunchdTemplate(), nil
	case "windows":
		return d.getWindowsTaskTemplate(), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", d.platform.Platform)
	}
}

func (d *Detector) getSystemdTemplate() string {
	// Resolve VROOLI_ROOT at runtime for a working template
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir, _ := os.UserHomeDir()
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}

	// Use the Go loop binary for cross-platform consistency
	loopBinaryPath := filepath.Join(vrooliRoot, "scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop")
	workDir := filepath.Join(vrooliRoot, "scenarios/vrooli-autoheal")
	homeDir, _ := os.UserHomeDir()
	currentUser, _ := user.Current()
	username := "root"
	if currentUser != nil {
		username = currentUser.Username
	}

	return fmt.Sprintf(`[Unit]
Description=Vrooli Autoheal - Self-healing infrastructure supervisor
# Wait for network and Docker to be ready before starting
# This prevents race conditions with docker-dependent scenarios
After=network-online.target docker.service docker.socket
Wants=network-online.target
# Optional dependency on Docker - don't fail if Docker isn't installed
Wants=docker.service
# Rate limiting to prevent crash loops
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
ExecStart=%s
Restart=always
# Wait a bit before restarting to allow dependencies to recover
RestartSec=15
User=%s
Environment=VROOLI_LIFECYCLE_MANAGED=true
Environment=HOME=%s
Environment=VROOLI_ROOT=%s
Environment=PATH=/usr/local/bin:/usr/bin:/bin:%s/.vrooli/bin
WorkingDirectory=%s
# Graceful shutdown timeout
TimeoutStopSec=30

[Install]
WantedBy=default.target
`, loopBinaryPath, username, homeDir, vrooliRoot, homeDir, workDir)
}

func (d *Detector) getLaunchdTemplate() string {
	// Resolve paths at runtime for a working template
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir, _ := os.UserHomeDir()
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}
	homeDir, _ := os.UserHomeDir()

	// Use the Go loop binary for cross-platform consistency
	loopBinaryPath := filepath.Join(vrooliRoot, "scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop")
	logPath := filepath.Join(homeDir, "Library/Logs/vrooli-autoheal.log")
	errPath := filepath.Join(homeDir, "Library/Logs/vrooli-autoheal.error.log")

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.vrooli.autoheal</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>EnvironmentVariables</key>
    <dict>
        <key>VROOLI_LIFECYCLE_MANAGED</key>
        <string>true</string>
        <key>VROOLI_ROOT</key>
        <string>%s</string>
        <key>HOME</key>
        <string>%s</string>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:%s/.vrooli/bin</string>
    </dict>
    <key>WorkingDirectory</key>
    <string>%s/scenarios/vrooli-autoheal</string>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>ThrottleInterval</key>
    <integer>15</integer>
</dict>
</plist>
`, loopBinaryPath, vrooliRoot, homeDir, homeDir, vrooliRoot, logPath, errPath)
}

func (d *Detector) getWindowsTaskTemplate() string {
	// Resolve paths at runtime for a working template
	// On Windows, VROOLI_ROOT would typically be in user's home directory
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir, _ := os.UserHomeDir()
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}

	// Use the Go loop binary - must have .exe extension on Windows
	cliPath := filepath.Join(vrooliRoot, "scenarios", "vrooli-autoheal", "cli", "vrooli-autoheal-loop.exe")
	workDir := filepath.Join(vrooliRoot, "scenarios", "vrooli-autoheal")

	// Get current user for the task principal
	currentUser, _ := user.Current()
	username := ""
	if currentUser != nil {
		username = currentUser.Username
	}

	// Use S4U (Service for User) logon type which allows running at boot without
	// an active user session and without storing credentials. This is the correct
	// approach for a watchdog that needs to start at system boot.
	// Note: S4U requires "Log on as a batch job" privilege for the user.
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Vrooli Autoheal - Self-healing infrastructure supervisor</Description>
    <Author>Vrooli</Author>
  </RegistrationInfo>
  <Triggers>
    <BootTrigger>
      <Enabled>true</Enabled>
      <Delay>PT30S</Delay>
    </BootTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>%s</UserId>
      <LogonType>S4U</LogonType>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>7</Priority>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>999</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>%s</Command>
      <WorkingDirectory>%s</WorkingDirectory>
    </Exec>
  </Actions>
</Task>
`, username, cliPath, workDir)
}
