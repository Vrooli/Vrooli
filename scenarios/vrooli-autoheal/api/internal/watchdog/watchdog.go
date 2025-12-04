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
func (d *Detector) isLoopRunning() bool {
	// The API itself being up indicates the loop is running
	// In production, this would check for the specific loop process
	// For now, if we're being called, the API is up which means the scenario is running
	return true
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
			break
		}
	}

	if !status.WatchdogInstalled {
		return
	}

	// Check if loaded (enabled and potentially running)
	cmd := exec.Command("launchctl", "list", "com.vrooli.autoheal")
	if err := cmd.Run(); err == nil {
		status.WatchdogEnabled = true
		status.WatchdogRunning = true
	}
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
	// Note: This is a template for documentation/reference.
	// The CLI install command generates the actual service file with proper paths.
	// For system-wide installs, the path should be updated by the installer.
	return `[Unit]
Description=Vrooli Autoheal - Self-healing infrastructure supervisor
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
# Update ExecStart path to match your VROOLI_ROOT installation
ExecStart=${VROOLI_ROOT}/scenarios/vrooli-autoheal/cli/vrooli-autoheal loop
Restart=always
RestartSec=10
User=root
Environment=VROOLI_LIFECYCLE_MANAGED=true
WorkingDirectory=${VROOLI_ROOT}/scenarios/vrooli-autoheal

[Install]
WantedBy=multi-user.target
`
}

func (d *Detector) getLaunchdTemplate() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.vrooli.autoheal</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/vrooli</string>
        <string>autoheal</string>
        <string>loop</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/vrooli-autoheal.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/vrooli-autoheal.error.log</string>
</dict>
</plist>
`
}

func (d *Detector) getWindowsTaskTemplate() string {
	return `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Vrooli Autoheal - Self-healing infrastructure supervisor</Description>
  </RegistrationInfo>
  <Triggers>
    <BootTrigger>
      <Enabled>true</Enabled>
    </BootTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>999</Count>
    </RestartOnFailure>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
  </Settings>
  <Actions>
    <Exec>
      <Command>vrooli</Command>
      <Arguments>autoheal loop</Arguments>
    </Exec>
  </Actions>
</Task>
`
}
