// Package watchdog provides OS-level service installation for boot recovery
// [REQ:WATCH-INSTALL-001] [REQ:WATCH-LINUX-002] [REQ:WATCH-MAC-002] [REQ:WATCH-WIN-002]
package watchdog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InstallResult represents the outcome of an installation attempt
type InstallResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	ServicePath   string `json:"servicePath,omitempty"`
	NeedsLinger   bool   `json:"needsLinger,omitempty"`
	LingerCommand string `json:"lingerCommand,omitempty"`
	Error         string `json:"error,omitempty"`
}

// UninstallResult represents the outcome of an uninstallation attempt
type UninstallResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// InstallOptions configures the installation behavior
type InstallOptions struct {
	// UseSystemService installs as a system-wide service (requires root/admin)
	// If false, installs as a user service
	UseSystemService bool `json:"useSystemService"`
	// EnableLingering automatically enables lingering for user services on Linux
	// Requires sudo access
	EnableLingering bool `json:"enableLingering"`
}

// Install installs the watchdog service on the current platform
// [REQ:WATCH-INSTALL-001]
func (d *Detector) Install(ctx context.Context, opts InstallOptions) *InstallResult {
	// Verify the loop binary exists before installing
	if err := d.verifyLoopBinaryExists(); err != nil {
		return &InstallResult{
			Success: false,
			Message: "Loop binary not found - please build it first",
			Error:   err.Error(),
		}
	}

	switch d.platform.Platform {
	case "linux":
		return d.installLinux(ctx, opts)
	case "macos":
		return d.installMacOS(ctx, opts)
	case "windows":
		return d.installWindows(ctx, opts)
	default:
		return &InstallResult{
			Success: false,
			Message: "Unsupported platform for watchdog installation",
			Error:   fmt.Sprintf("platform %s is not supported", d.platform.Platform),
		}
	}
}

// verifyLoopBinaryExists checks if the Go loop binary has been built
func (d *Detector) verifyLoopBinaryExists() error {
	vrooliRoot := d.probe.getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir, _ := d.probe.userHomeDir()
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}

	var binaryPath string
	switch d.probe.goos() {
	case "windows":
		binaryPath = filepath.Join(vrooliRoot, "scenarios", "vrooli-autoheal", "cli", "vrooli-autoheal-loop.exe")
	default:
		binaryPath = filepath.Join(vrooliRoot, "scenarios", "vrooli-autoheal", "cli", "vrooli-autoheal-loop")
	}

	if err := d.probe.stat(binaryPath); os.IsNotExist(err) {
		buildCmd := "go build -o vrooli-autoheal-loop ./cli/loop"
		if d.probe.goos() == "windows" {
			buildCmd = "go build -o vrooli-autoheal-loop.exe ./cli/loop"
		}
		return fmt.Errorf("loop binary not found at %s. Build it with: cd %s/scenarios/vrooli-autoheal && %s",
			binaryPath, vrooliRoot, buildCmd)
	} else if err != nil {
		return fmt.Errorf("failed to verify loop binary at %s: %w", binaryPath, err)
	}

	return nil
}

// Uninstall removes the watchdog service from the current platform
func (d *Detector) Uninstall(ctx context.Context) *UninstallResult {
	switch d.platform.Platform {
	case "linux":
		return d.uninstallLinux(ctx)
	case "macos":
		return d.uninstallMacOS(ctx)
	case "windows":
		return d.uninstallWindows(ctx)
	default:
		return &UninstallResult{
			Success: false,
			Message: "Unsupported platform for watchdog uninstallation",
			Error:   fmt.Sprintf("platform %s is not supported", d.platform.Platform),
		}
	}
}

// EnableLingering enables systemd lingering for the current user (Linux only)
// This allows user services to start at boot without a login session
func (d *Detector) EnableLingering(ctx context.Context) *InstallResult {
	if d.platform.Platform != "linux" {
		return &InstallResult{
			Success: false,
			Message: "Lingering is only applicable to Linux systems",
			Error:   "not linux",
		}
	}

	currentUser, err := d.probe.currentUser()
	if err != nil {
		return &InstallResult{
			Success: false,
			Message: "Failed to get current user",
			Error:   err.Error(),
		}
	}

	// Try to enable lingering (requires sudo)
	output, err := d.probe.commandOutput("sudo", "loginctl", "enable-linger", currentUser.Username)
	if err != nil {
		return &InstallResult{
			Success:       false,
			Message:       "Failed to enable lingering - sudo access may be required",
			Error:         fmt.Sprintf("%v: %s", err, string(output)),
			LingerCommand: fmt.Sprintf("sudo loginctl enable-linger %s", currentUser.Username),
		}
	}

	return &InstallResult{
		Success: true,
		Message: fmt.Sprintf("Lingering enabled for user %s", currentUser.Username),
	}
}

// installLinux installs the systemd service on Linux
// [REQ:WATCH-LINUX-002]
func (d *Detector) installLinux(ctx context.Context, opts InstallOptions) *InstallResult {
	if !d.platform.SupportsSystemd {
		return &InstallResult{
			Success: false,
			Message: "Systemd is not available on this Linux system",
			Error:   "systemd not supported",
		}
	}

	template := d.getSystemdTemplate()
	var servicePath string
	var isUserService bool

	if opts.UseSystemService {
		// System-wide service (requires root)
		servicePath = "/etc/systemd/system/vrooli-autoheal.service"
		isUserService = false
	} else {
		// User service
		homeDir, err := d.probe.userHomeDir()
		if err != nil {
			return &InstallResult{
				Success: false,
				Message: "Failed to determine home directory",
				Error:   err.Error(),
			}
		}
		serviceDir := filepath.Join(homeDir, ".config", "systemd", "user")
		if err := d.probe.mkdirAll(serviceDir, 0o755); err != nil {
			return &InstallResult{
				Success: false,
				Message: "Failed to create systemd user directory",
				Error:   err.Error(),
			}
		}
		servicePath = filepath.Join(serviceDir, "vrooli-autoheal.service")
		isUserService = true
	}

	// Write service file
	var writeErr error
	if opts.UseSystemService {
		// System service needs sudo to write
		output, err := d.probe.commandOutputInput("sudo", template, "tee", servicePath)
		if err != nil {
			writeErr = fmt.Errorf("%v: %s", err, string(output))
		}
	} else {
		// User service can be written directly
		writeErr = d.probe.writeFile(servicePath, []byte(template), 0o644)
	}

	if writeErr != nil {
		return &InstallResult{
			Success: false,
			Message: "Failed to write service file",
			Error:   writeErr.Error(),
		}
	}

	// Reload systemd daemon
	reloadArgs := []string{"daemon-reload"}
	enableArgs := []string{"enable", "vrooli-autoheal"}
	startArgs := []string{"start", "vrooli-autoheal"}
	reloadCmdName := "systemctl"
	enableCmdName := "systemctl"
	startCmdName := "systemctl"
	if opts.UseSystemService {
		reloadCmdName = "sudo"
		enableCmdName = "sudo"
		startCmdName = "sudo"
		reloadArgs = []string{"systemctl", "daemon-reload"}
		enableArgs = []string{"systemctl", "enable", "vrooli-autoheal"}
		startArgs = []string{"systemctl", "start", "vrooli-autoheal"}
	} else {
		reloadArgs = []string{"--user", "daemon-reload"}
		enableArgs = []string{"--user", "enable", "vrooli-autoheal"}
		startArgs = []string{"--user", "start", "vrooli-autoheal"}
	}

	if output, err := d.probe.commandOutput(reloadCmdName, reloadArgs...); err != nil {
		return &InstallResult{
			Success:     false,
			Message:     "Service file written but daemon-reload failed",
			ServicePath: servicePath,
			Error:       fmt.Sprintf("%v: %s", err, string(output)),
		}
	}

	if output, err := d.probe.commandOutput(enableCmdName, enableArgs...); err != nil {
		return &InstallResult{
			Success:     false,
			Message:     "Service file written but enable failed",
			ServicePath: servicePath,
			Error:       fmt.Sprintf("%v: %s", err, string(output)),
		}
	}

	if output, err := d.probe.commandOutput(startCmdName, startArgs...); err != nil {
		// Start may fail if already running, which is OK
		errStr := string(output)
		if !strings.Contains(errStr, "already") {
			return &InstallResult{
				Success:     false,
				Message:     "Service enabled but start failed",
				ServicePath: servicePath,
				Error:       fmt.Sprintf("%v: %s", err, errStr),
			}
		}
	}

	result := &InstallResult{
		Success:     true,
		ServicePath: servicePath,
	}

	// Check lingering for user services - this is critical for boot protection
	if isUserService {
		currentUser, _ := d.probe.currentUser()
		lingerEnabled := currentUser != nil && d.isLingeringEnabled(currentUser.Username)

		if !lingerEnabled && opts.EnableLingering {
			// Try to enable lingering if requested
			lingerResult := d.EnableLingering(ctx)
			lingerEnabled = lingerResult.Success
		}

		if lingerEnabled {
			result.Message = "Watchdog installed with FULL BOOT PROTECTION - service will start automatically on boot"
			result.NeedsLinger = false
			result.LingerCommand = ""
		} else {
			// Lingering not enabled - boot protection is incomplete
			result.NeedsLinger = true
			if currentUser != nil {
				result.LingerCommand = fmt.Sprintf("sudo loginctl enable-linger %s", currentUser.Username)
			}
			result.Message = "Watchdog installed but BOOT PROTECTION INCOMPLETE - service will only start on user login, not on headless boot. Run: " + result.LingerCommand
		}
	} else {
		// System service - always has full boot protection
		result.Message = "Watchdog installed with FULL BOOT PROTECTION - service will start automatically on boot"
	}

	// Invalidate cache to reflect new state
	d.mu.Lock()
	d.cached = nil
	d.mu.Unlock()

	return result
}

// installMacOS installs the launchd plist on macOS
// [REQ:WATCH-MAC-002]
func (d *Detector) installMacOS(ctx context.Context, opts InstallOptions) *InstallResult {
	if !d.platform.SupportsLaunchd {
		return &InstallResult{
			Success: false,
			Message: "Launchd is not available",
			Error:   "launchd not supported",
		}
	}

	template := d.getLaunchdTemplate()
	var plistPath string

	if opts.UseSystemService {
		// System-wide daemon (requires root)
		plistPath = "/Library/LaunchDaemons/com.vrooli.autoheal.plist"
	} else {
		// User agent
		homeDir, err := d.probe.userHomeDir()
		if err != nil {
			return &InstallResult{
				Success: false,
				Message: "Failed to determine home directory",
				Error:   err.Error(),
			}
		}
		launchDir := filepath.Join(homeDir, "Library", "LaunchAgents")
		if err := d.probe.mkdirAll(launchDir, 0o755); err != nil {
			return &InstallResult{
				Success: false,
				Message: "Failed to create LaunchAgents directory",
				Error:   err.Error(),
			}
		}
		plistPath = filepath.Join(launchDir, "com.vrooli.autoheal.plist")
	}

	// Write plist file
	var writeErr error
	if opts.UseSystemService {
		output, err := d.probe.commandOutputInput("sudo", template, "tee", plistPath)
		if err != nil {
			writeErr = fmt.Errorf("%v: %s", err, string(output))
		}
	} else {
		writeErr = d.probe.writeFile(plistPath, []byte(template), 0o644)
	}

	if writeErr != nil {
		return &InstallResult{
			Success: false,
			Message: "Failed to write plist file",
			Error:   writeErr.Error(),
		}
	}

	// Load the service
	loadCmdName := "launchctl"
	loadArgs := []string{"load", plistPath}
	if opts.UseSystemService {
		loadCmdName = "sudo"
		loadArgs = []string{"launchctl", "load", plistPath}
	}
	if output, err := d.probe.commandOutput(loadCmdName, loadArgs...); err != nil {
		errStr := string(output)
		// May already be loaded
		if !strings.Contains(errStr, "already loaded") && !strings.Contains(errStr, "service already loaded") {
			return &InstallResult{
				Success:     false,
				Message:     "Plist written but load failed",
				ServicePath: plistPath,
				Error:       fmt.Sprintf("%v: %s", err, errStr),
			}
		}
	}

	// Invalidate cache
	d.mu.Lock()
	d.cached = nil
	d.mu.Unlock()

	return &InstallResult{
		Success:     true,
		Message:     "Watchdog service installed and loaded successfully",
		ServicePath: plistPath,
	}
}

// installWindows installs the Windows scheduled task
// [REQ:WATCH-WIN-002]
func (d *Detector) installWindows(ctx context.Context, opts InstallOptions) *InstallResult {
	if d.probe.goos() != "windows" {
		return &InstallResult{
			Success: false,
			Message: "Windows installation must be run on Windows",
			Error:   "not windows",
		}
	}

	template := d.getWindowsTaskTemplate()

	// Write XML to temp file
	tmpPath, err := d.probe.writeTempFile("vrooli-autoheal-*.xml", []byte(template))
	if err != nil {
		return &InstallResult{
			Success: false,
			Message: "Failed to create temporary file",
			Error:   err.Error(),
		}
	}
	defer func() { _ = d.probe.remove(tmpPath) }()

	// Create the scheduled task
	if output, err := d.probe.commandOutput("schtasks", "/Create", "/TN", "VrooliAutoheal", "/XML", tmpPath, "/F"); err != nil {
		return &InstallResult{
			Success: false,
			Message: "Failed to create scheduled task",
			Error:   fmt.Sprintf("%v: %s", err, string(output)),
		}
	}

	// Invalidate cache
	d.mu.Lock()
	d.cached = nil
	d.mu.Unlock()

	return &InstallResult{
		Success:     true,
		Message:     "Watchdog scheduled task created successfully",
		ServicePath: "Task Scheduler: VrooliAutoheal",
	}
}

// uninstallLinux removes the systemd service
func (d *Detector) uninstallLinux(ctx context.Context) *UninstallResult {
	// Find and remove the service from all possible locations
	servicePaths := []string{
		"/etc/systemd/system/vrooli-autoheal.service",
		"/usr/lib/systemd/system/vrooli-autoheal.service",
	}

	homeDir, _ := d.probe.userHomeDir()
	if homeDir != "" {
		servicePaths = append(servicePaths, filepath.Join(homeDir, ".config/systemd/user/vrooli-autoheal.service"))
	}

	var removed bool
	var isUserService bool
	var removePath string

	for _, path := range servicePaths {
		if err := d.probe.stat(path); err == nil {
			removePath = path
			isUserService = strings.Contains(path, ".config/systemd/user")
			break
		}
	}

	if removePath == "" {
		return &UninstallResult{
			Success: true,
			Message: "No watchdog service found to uninstall",
		}
	}

	// Stop and disable the service
	if isUserService {
		_ = d.probe.commandRun("systemctl", "--user", "stop", "vrooli-autoheal")
		_ = d.probe.commandRun("systemctl", "--user", "disable", "vrooli-autoheal")
	} else {
		_ = d.probe.commandRun("sudo", "systemctl", "stop", "vrooli-autoheal")
		_ = d.probe.commandRun("sudo", "systemctl", "disable", "vrooli-autoheal")
	}

	// Remove the service file
	if isUserService {
		if err := d.probe.remove(removePath); err != nil {
			return &UninstallResult{
				Success: false,
				Message: "Failed to remove service file",
				Error:   err.Error(),
			}
		}
		removed = true
	} else {
		if output, err := d.probe.commandOutput("sudo", "rm", "-f", removePath); err != nil {
			return &UninstallResult{
				Success: false,
				Message: "Failed to remove service file",
				Error:   fmt.Sprintf("%v: %s", err, string(output)),
			}
		}
		removed = true
	}

	// Reload daemon
	if isUserService {
		_ = d.probe.commandRun("systemctl", "--user", "daemon-reload")
	} else {
		_ = d.probe.commandRun("sudo", "systemctl", "daemon-reload")
	}

	// Invalidate cache
	d.mu.Lock()
	d.cached = nil
	d.mu.Unlock()

	if removed {
		return &UninstallResult{
			Success: true,
			Message: "Watchdog service uninstalled successfully",
		}
	}

	return &UninstallResult{
		Success: false,
		Message: "Failed to uninstall service",
		Error:   "unknown error",
	}
}

// uninstallMacOS removes the launchd plist
func (d *Detector) uninstallMacOS(ctx context.Context) *UninstallResult {
	homeDir, _ := d.probe.userHomeDir()
	plistPaths := []string{
		"/Library/LaunchDaemons/com.vrooli.autoheal.plist",
	}
	if homeDir != "" {
		plistPaths = append(plistPaths, filepath.Join(homeDir, "Library/LaunchAgents/com.vrooli.autoheal.plist"))
	}

	var removed bool
	for _, path := range plistPaths {
		if err := d.probe.stat(path); err == nil {
			// Unload first
			isSystem := strings.HasPrefix(path, "/Library")
			if isSystem {
				_ = d.probe.commandRun("sudo", "launchctl", "unload", path)
				_ = d.probe.commandRun("sudo", "rm", "-f", path)
			} else {
				_ = d.probe.commandRun("launchctl", "unload", path)
				_ = d.probe.remove(path)
			}
			removed = true
		}
	}

	// Invalidate cache
	d.mu.Lock()
	d.cached = nil
	d.mu.Unlock()

	if removed {
		return &UninstallResult{
			Success: true,
			Message: "Watchdog service uninstalled successfully",
		}
	}

	return &UninstallResult{
		Success: true,
		Message: "No watchdog service found to uninstall",
	}
}

// uninstallWindows removes the scheduled task
func (d *Detector) uninstallWindows(ctx context.Context) *UninstallResult {
	if d.probe.goos() != "windows" {
		return &UninstallResult{
			Success: false,
			Message: "Windows uninstallation must be run on Windows",
			Error:   "not windows",
		}
	}

	if output, err := d.probe.commandOutput("schtasks", "/Delete", "/TN", "VrooliAutoheal", "/F"); err != nil {
		errStr := string(output)
		if strings.Contains(errStr, "does not exist") {
			return &UninstallResult{
				Success: true,
				Message: "No watchdog task found to uninstall",
			}
		}
		return &UninstallResult{
			Success: false,
			Message: "Failed to delete scheduled task",
			Error:   fmt.Sprintf("%v: %s", err, errStr),
		}
	}

	// Invalidate cache
	d.mu.Lock()
	d.cached = nil
	d.mu.Unlock()

	return &UninstallResult{
		Success: true,
		Message: "Watchdog scheduled task removed successfully",
	}
}

// GetInstallStatus returns a detailed installation status
func (d *Detector) GetInstallStatus() *InstallStatus {
	status := d.Detect()

	return &InstallStatus{
		Installed:        status.WatchdogInstalled,
		Enabled:          status.WatchdogEnabled,
		Running:          status.WatchdogRunning,
		BootProtected:    status.BootProtectionActive,
		ServicePath:      status.ServicePath,
		WatchdogType:     string(status.WatchdogType),
		CanInstall:       status.CanInstall,
		NeedsLinger:      status.IsUserService && !status.LingeringEnabled,
		LingerCommand:    fmt.Sprintf("sudo loginctl enable-linger %s", status.Username),
		ProtectionLevel:  string(status.ProtectionLevel),
		LastChecked:      time.Now().UTC().Format(time.RFC3339),
		RecommendedSetup: d.getRecommendedSetup(),
	}
}

// InstallStatus provides detailed installation information
type InstallStatus struct {
	Installed        bool   `json:"installed"`
	Enabled          bool   `json:"enabled"`
	Running          bool   `json:"running"`
	BootProtected    bool   `json:"bootProtected"`
	ServicePath      string `json:"servicePath,omitempty"`
	WatchdogType     string `json:"watchdogType"`
	CanInstall       bool   `json:"canInstall"`
	NeedsLinger      bool   `json:"needsLinger"`
	LingerCommand    string `json:"lingerCommand,omitempty"`
	ProtectionLevel  string `json:"protectionLevel"`
	LastChecked      string `json:"lastChecked"`
	RecommendedSetup string `json:"recommendedSetup"`
}

// getRecommendedSetup returns platform-specific setup recommendations
func (d *Detector) getRecommendedSetup() string {
	switch d.platform.Platform {
	case "linux":
		return "user" // User service is recommended for non-root installations
	case "macos":
		return "user"
	case "windows":
		return "system" // Windows typically runs as system
	default:
		return ""
	}
}
