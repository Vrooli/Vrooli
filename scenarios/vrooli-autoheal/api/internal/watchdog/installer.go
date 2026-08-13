// Package watchdog adapts the scenario watchdog contract to the shared native
// service backend. Service definitions, lifecycle verbs, and state parsing are
// owned by platform-go; this package only supplies scenario identity and
// translates results for the API.
package watchdog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	platformgo "github.com/vrooli/platform-go"
)

type InstallResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	ServicePath   string `json:"servicePath,omitempty"`
	NeedsLinger   bool   `json:"needsLinger,omitempty"`
	LingerCommand string `json:"lingerCommand,omitempty"`
	Error         string `json:"error,omitempty"`
}

type UninstallResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type InstallOptions struct {
	UseSystemService bool `json:"useSystemService"`
	EnableLingering  bool `json:"enableLingering"`
}

func (d *Detector) Install(ctx context.Context, opts InstallOptions) *InstallResult {
	if err := d.verifyLoopBinaryExists(); err != nil {
		return &InstallResult{Message: "Loop binary not found - please build it first", Error: err.Error()}
	}
	switch d.platform.Platform {
	case "linux":
		return d.installLinux(ctx, opts)
	case "macos":
		return d.installMacOS(ctx, opts)
	case "windows":
		return d.installWindows(ctx, opts)
	default:
		return &InstallResult{Message: "Unsupported platform for watchdog installation", Error: fmt.Sprintf("platform %s is not supported", d.platform.Platform)}
	}
}

func (d *Detector) verifyLoopBinaryExists() error {
	root := d.resolveVrooliRoot()
	name := "vrooli-autoheal-loop"
	if d.probe.goos() == "windows" {
		name += ".exe"
	}
	path := filepath.Join(root, "scenarios", "vrooli-autoheal", "cli", name)
	if err := d.probe.stat(path); err != nil {
		build := "go build -o vrooli-autoheal-loop ./cli/loop"
		if d.probe.goos() == "windows" {
			build = "go build -o vrooli-autoheal-loop.exe ./cli/loop"
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("loop binary not found at %s. Build it with: cd %s/scenarios/vrooli-autoheal && %s", path, root, build)
		}
		return fmt.Errorf("failed to verify loop binary at %s: %w", path, err)
	}
	return nil
}

func (d *Detector) nativeDefinition(opts InstallOptions) (platformgo.NativeServiceOptions, error) {
	root := d.resolveVrooliRoot()
	home, err := d.probe.userHomeDir()
	if err != nil {
		return platformgo.NativeServiceOptions{}, err
	}
	loop := filepath.Join(root, "scenarios", "vrooli-autoheal", "cli", "vrooli-autoheal-loop")
	if d.probe.goos() == "windows" {
		loop += ".exe"
	}

	name, path := "vrooli-autoheal.service", "/etc/systemd/system/vrooli-autoheal.service"
	userService := !opts.UseSystemService
	switch string(d.platform.Platform) {
	case "linux":
		if userService {
			path = filepath.Join(home, ".config", "systemd", "user", name)
		}
	case "macos":
		name, path = "com.vrooli.autoheal", "/Library/LaunchDaemons/com.vrooli.autoheal.plist"
		if userService {
			path = filepath.Join(home, "Library", "LaunchAgents", name+".plist")
		}
	case "windows":
		name, path, userService = "VrooliAutoheal", filepath.Join(os.TempDir(), "vrooli-autoheal.xml"), false
	default:
		return platformgo.NativeServiceOptions{}, fmt.Errorf("unsupported platform %q", d.platform.Platform)
	}
	content, err := platformgo.RenderWatchdogDefinition(string(d.platform.Platform), platformgo.WatchdogDefinitionOptions{
		Root: root, Home: home, LoopBinary: loop, VrooliBinary: d.resolveVrooliBinary(), SystemService: opts.UseSystemService,
	})
	if err != nil {
		return platformgo.NativeServiceOptions{}, err
	}
	return platformgo.NativeServiceOptions{Name: name, Path: path, Content: content, User: userService}, nil
}

func (d *Detector) installLinux(ctx context.Context, opts InstallOptions) *InstallResult {
	if !d.platform.SupportsSystemd {
		return &InstallResult{Message: "Systemd is not available on this Linux system", Error: "systemd not supported"}
	}
	return d.installNative(ctx, opts, "Watchdog installed with native service backend")
}

func (d *Detector) installMacOS(ctx context.Context, opts InstallOptions) *InstallResult {
	if !d.platform.SupportsLaunchd {
		return &InstallResult{Message: "Launchd is not available", Error: "launchd not supported"}
	}
	return d.installNative(ctx, opts, "Watchdog installed with native service backend")
}

func (d *Detector) installWindows(ctx context.Context, opts InstallOptions) *InstallResult {
	if d.probe.goos() != "windows" {
		return &InstallResult{Message: "Windows installation must be run on Windows", Error: "not windows"}
	}
	if !d.platform.SupportsWindowsSvc {
		return &InstallResult{Message: "Windows Task Scheduler is not available", Error: "windows service backend not supported"}
	}
	return d.installNative(ctx, opts, "Watchdog scheduled task created successfully")
}

func (d *Detector) installNative(_ context.Context, opts InstallOptions, message string) *InstallResult {
	options, err := d.nativeDefinition(opts)
	if err != nil {
		return &InstallResult{Message: "Failed to prepare native watchdog service", Error: err.Error()}
	}
	result, err := d.service.Install(options)
	if err != nil {
		return &InstallResult{Message: "Failed to install native watchdog service", ServicePath: options.Path, Error: err.Error()}
	}
	d.invalidate()
	out := &InstallResult{Success: true, Message: message, ServicePath: result.Path}
	if string(d.platform.Platform) == "linux" && options.User {
		current, _ := d.probe.currentUser()
		if current != nil && d.isLingeringEnabled(current.Username) {
			out.Message = "Watchdog installed with full boot protection"
		} else {
			out.NeedsLinger = true
			if current != nil {
				out.LingerCommand = fmt.Sprintf("sudo loginctl enable-linger %s", current.Username)
			}
			out.Message = "Watchdog installed but BOOT PROTECTION INCOMPLETE"
		}
	}
	return out
}

func (d *Detector) Uninstall(ctx context.Context) *UninstallResult {
	switch d.platform.Platform {
	case "linux":
		return d.uninstallLinux(ctx)
	case "macos":
		return d.uninstallMacOS(ctx)
	case "windows":
		return d.uninstallWindows(ctx)
	default:
		return &UninstallResult{Message: "Unsupported platform for watchdog uninstallation", Error: fmt.Sprintf("platform %s is not supported", d.platform.Platform)}
	}
}

func (d *Detector) uninstallLinux(_ context.Context) *UninstallResult {
	home, _ := d.probe.userHomeDir()
	items := []platformgo.NativeServiceOptions{
		{Name: "vrooli-autoheal.service", Path: "/etc/systemd/system/vrooli-autoheal.service"},
		{Name: "vrooli-autoheal.service", Path: filepath.Join(home, ".config", "systemd", "user", "vrooli-autoheal.service"), User: true},
	}
	return d.uninstallNative(items)
}

func (d *Detector) uninstallMacOS(_ context.Context) *UninstallResult {
	home, _ := d.probe.userHomeDir()
	items := []platformgo.NativeServiceOptions{
		{Name: "com.vrooli.autoheal", Path: "/Library/LaunchDaemons/com.vrooli.autoheal.plist"},
		{Name: "com.vrooli.autoheal", Path: filepath.Join(home, "Library", "LaunchAgents", "com.vrooli.autoheal.plist"), User: true},
	}
	return d.uninstallNative(items)
}

func (d *Detector) uninstallWindows(_ context.Context) *UninstallResult {
	if d.probe.goos() != "windows" {
		return &UninstallResult{Message: "Windows uninstallation must be run on Windows", Error: "not windows"}
	}
	return d.uninstallNative([]platformgo.NativeServiceOptions{{Name: "VrooliAutoheal"}})
}

func (d *Detector) uninstallNative(items []platformgo.NativeServiceOptions) *UninstallResult {
	for _, item := range items {
		if item.Path != "" {
			if err := d.probe.stat(item.Path); err != nil {
				continue
			}
		}
		if _, err := d.service.Uninstall(item); err != nil {
			return &UninstallResult{Message: "Failed to remove native watchdog service", Error: err.Error()}
		}
	}
	d.invalidate()
	return &UninstallResult{Success: true, Message: "Watchdog service uninstalled successfully"}
}

func (d *Detector) EnableLingering(_ context.Context) *InstallResult {
	if d.platform.Platform != "linux" {
		return &InstallResult{Message: "Lingering is only applicable to Linux systems", Error: "not linux"}
	}
	current, err := d.probe.currentUser()
	if err != nil || current == nil {
		return &InstallResult{Message: "Failed to get current user", Error: "current user unavailable"}
	}
	return &InstallResult{
		Message: "Lingering requires operator setup", Error: "needs setup",
		LingerCommand: fmt.Sprintf("sudo loginctl enable-linger %s", current.Username),
	}
}

func (d *Detector) GetInstallStatus() *InstallStatus {
	status := d.Detect()
	linger := status.IsUserService && !status.LingeringEnabled
	command := ""
	if linger && status.Username != "" && d.platform.Platform == "linux" {
		command = fmt.Sprintf("sudo loginctl enable-linger %s", status.Username)
	}
	return &InstallStatus{Installed: status.WatchdogInstalled, Enabled: status.WatchdogEnabled, Running: status.WatchdogRunning, BootProtected: status.BootProtectionActive, ServicePath: status.ServicePath, WatchdogType: string(status.WatchdogType), CanInstall: status.CanInstall, NeedsLinger: linger, LingerCommand: command, ProtectionLevel: string(status.ProtectionLevel), LastChecked: time.Now().UTC().Format(time.RFC3339), RecommendedSetup: d.getRecommendedSetup()}
}

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

func (d *Detector) getRecommendedSetup() string {
	switch d.platform.Platform {
	case "linux", "macos":
		return "user"
	case "windows":
		return "system"
	default:
		return ""
	}
}
