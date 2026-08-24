// Package watchdog owns the scenario-facing watchdog contract. Native
// lifecycle operations and definition rendering live in platform-go.
package watchdog

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reporoot"
)

type WatchdogType string

const (
	WatchdogTypeNone    WatchdogType = ""
	WatchdogTypeSystemd WatchdogType = "systemd"
	WatchdogTypeLaunchd WatchdogType = "launchd"
	WatchdogTypeWindows WatchdogType = "windows-task"
)

type Status struct {
	LoopRunning          bool            `json:"loopRunning"`
	WatchdogType         WatchdogType    `json:"watchdogType"`
	WatchdogInstalled    bool            `json:"watchdogInstalled"`
	WatchdogEnabled      bool            `json:"watchdogEnabled"`
	WatchdogRunning      bool            `json:"watchdogRunning"`
	BootProtectionActive bool            `json:"bootProtectionActive"`
	CanInstall           bool            `json:"canInstall"`
	ServicePath          string          `json:"servicePath,omitempty"`
	LastError            string          `json:"lastError,omitempty"`
	ProtectionLevel      ProtectionLevel `json:"protectionLevel"`
	LingeringEnabled     bool            `json:"lingeringEnabled"`
	Username             string          `json:"username,omitempty"`
	IsUserService        bool            `json:"isUserService,omitempty"`
}

type ProtectionLevel string

const (
	ProtectionFull    ProtectionLevel = "full"
	ProtectionPartial ProtectionLevel = "partial"
	ProtectionNone    ProtectionLevel = "none"
)

type Detector struct {
	platform *platform.Capabilities
	probe    detectorProbe
	service  serviceBackend
	mu       sync.RWMutex
	cached   *Status
}

type detectorProbe interface {
	goos() string
	commandOutput(name string, args ...string) ([]byte, error)
	commandRun(name string, args ...string) error
	readDir(path string) ([]os.DirEntry, error)
	readFile(path string) ([]byte, error)
	stat(path string) error
	currentUser() (*user.User, error)
	userHomeDir() (string, error)
	getenv(key string) string
}

type realDetectorProbe struct{}

func (realDetectorProbe) goos() string { return runtime.GOOS }
func (realDetectorProbe) commandOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (realDetectorProbe) commandRun(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}
func (realDetectorProbe) readDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }
func (realDetectorProbe) readFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (realDetectorProbe) stat(path string) error                     { _, err := os.Stat(path); return err }
func (realDetectorProbe) currentUser() (*user.User, error)           { return user.Current() }
func (realDetectorProbe) userHomeDir() (string, error)               { return os.UserHomeDir() }
func (realDetectorProbe) getenv(key string) string                   { return os.Getenv(key) }

func NewDetector(plat *platform.Capabilities) *Detector {
	if plat == nil {
		plat = platform.Detect()
	}
	return &Detector{platform: plat, probe: realDetectorProbe{}, service: nativeServiceBackend{}}
}

func (d *Detector) Detect() *Status {
	d.mu.Lock()
	defer d.mu.Unlock()
	status := &Status{LoopRunning: d.isLoopRunning(), CanInstall: d.canInstall()}
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
	status.ProtectionLevel = d.calculateProtectionLevel(status)
	status.BootProtectionActive = status.WatchdogInstalled && status.WatchdogEnabled
	d.cached = status
	return status
}

func (d *Detector) GetCached() *Status {
	d.mu.RLock()
	cached := d.cached
	d.mu.RUnlock()
	if cached == nil {
		return d.Detect()
	}
	return cached
}

func (d *Detector) invalidate() {
	d.mu.Lock()
	d.cached = nil
	d.mu.Unlock()
}

func (d *Detector) isLoopRunning() bool {
	if d.probe.goos() == "windows" {
		// tasklist can identify an image but cannot inspect its arguments, so it
		// cannot distinguish the API process from the loop. Query the native
		// process command line first; retain the tasklist fallback for hosts
		// where PowerShell process inspection is unavailable.
		output, err := d.probe.commandOutput("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-CimInstance Win32_Process | Select-Object -ExpandProperty CommandLine")
		if err == nil {
			return strings.Contains(strings.ToLower(string(output)), "vrooli-autoheal") && strings.Contains(strings.ToLower(string(output)), "loop")
		}
		output, err = d.probe.commandOutput("tasklist", "/FI", "IMAGENAME eq vrooli-autoheal*")
		return err == nil && strings.Contains(strings.ToLower(string(output)), "vrooli-autoheal") && strings.Contains(strings.ToLower(string(output)), "loop")
	}
	if d.probe.commandRun("pgrep", "-f", "vrooli-autoheal.*loop") == nil {
		return true
	}
	if d.probe.goos() != "linux" {
		return false
	}
	entries, err := d.probe.readDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "" || entry.Name()[0] < '0' || entry.Name()[0] > '9' {
			continue
		}
		cmdline, err := d.probe.readFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err == nil && strings.Contains(string(cmdline), "vrooli-autoheal") && strings.Contains(string(cmdline), "loop") {
			return true
		}
	}
	return false
}

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

func (d *Detector) detectLinux(status *Status) {
	if !d.platform.SupportsSystemd {
		status.LastError = "systemd not available on this Linux system"
		return
	}
	status.WatchdogType = WatchdogTypeSystemd
	if current, err := d.probe.currentUser(); err == nil && current != nil {
		status.Username = current.Username
	}
	home, _ := d.probe.userHomeDir()
	if home == "" {
		home = d.probe.getenv("HOME")
	}
	paths := []string{"/etc/systemd/system/vrooli-autoheal.service", "/usr/lib/systemd/system/vrooli-autoheal.service", filepath.Join(home, ".config", "systemd", "user", "vrooli-autoheal.service")}
	for _, path := range paths {
		if d.probe.stat(path) == nil {
			status.WatchdogInstalled = true
			status.ServicePath = path
			break
		}
	}
	if !status.WatchdogInstalled {
		return
	}
	status.IsUserService = strings.Contains(status.ServicePath, ".config/systemd/user")
	evidence, err := d.service.Status(platformgo.NativeServiceOptions{
		Name: "vrooli-autoheal.service", Path: status.ServicePath, User: status.IsUserService,
	})
	if err != nil {
		status.LastError = err.Error()
	}
	status.WatchdogRunning, status.WatchdogEnabled = evidence.State == platformgo.ServiceStateRunning, evidence.Enabled
	if status.IsUserService {
		status.LingeringEnabled = d.isLingeringEnabled(status.Username)
	} else {
		status.LingeringEnabled = true
	}
}

func (d *Detector) isLingeringEnabled(username string) bool {
	if username == "" {
		return false
	}
	if d.probe.stat(filepath.Join("/var/lib/systemd/linger", username)) == nil {
		return true
	}
	output, err := d.probe.commandOutput("loginctl", "show-user", username, "--property=Linger")
	return err == nil && strings.TrimSpace(string(output)) == "Linger=yes"
}

func (d *Detector) detectMacOS(status *Status) {
	if !d.platform.SupportsLaunchd {
		status.LastError = "launchd not available"
		return
	}
	status.WatchdogType = WatchdogTypeLaunchd
	home, _ := d.probe.userHomeDir()
	paths := []string{filepath.Join(home, "Library", "LaunchAgents", "com.vrooli.autoheal.plist"), "/Library/LaunchDaemons/com.vrooli.autoheal.plist"}
	for _, path := range paths {
		if d.probe.stat(path) == nil {
			status.WatchdogInstalled = true
			status.ServicePath = path
			status.IsUserService = strings.Contains(path, "LaunchAgents")
			break
		}
	}
	if !status.WatchdogInstalled {
		return
	}
	evidence, err := d.service.Status(platformgo.NativeServiceOptions{
		Name: "com.vrooli.autoheal", Path: status.ServicePath, User: status.IsUserService,
	})
	if err != nil {
		status.LastError = err.Error()
	}
	status.WatchdogEnabled, status.WatchdogRunning = evidence.Enabled, evidence.State == platformgo.ServiceStateRunning
}

func (d *Detector) detectWindows(status *Status) {
	if d.probe.goos() != "windows" {
		status.LastError = "Windows detection not available on this platform"
		return
	}
	status.WatchdogType = WatchdogTypeWindows
	evidence, err := d.service.Status(platformgo.NativeServiceOptions{Name: "VrooliAutoheal"})
	if err != nil {
		status.LastError = err.Error()
		return
	}
	status.WatchdogInstalled, status.ServicePath = evidence.Enabled, "Task Scheduler: VrooliAutoheal"
	status.WatchdogEnabled, status.WatchdogRunning = evidence.Enabled, evidence.State == platformgo.ServiceStateRunning
}

func (d *Detector) calculateProtectionLevel(status *Status) ProtectionLevel {
	if status.WatchdogInstalled && status.WatchdogEnabled && status.WatchdogRunning {
		return ProtectionFull
	}
	if status.LoopRunning {
		return ProtectionPartial
	}
	return ProtectionNone
}

func (d *Detector) GetServiceTemplate() (string, error) {
	home, _ := d.probe.userHomeDir()
	root := d.resolveVrooliRoot()
	loop := filepath.Join(root, "scenarios", "vrooli-autoheal", "cli", "vrooli-autoheal-loop")
	if d.probe.goos() == "windows" {
		loop += ".exe"
	}
	username := ""
	if current, err := d.probe.currentUser(); err == nil && current != nil {
		username = current.Username
	}
	return platformgo.RenderWatchdogDefinition(string(d.platform.Platform), platformgo.WatchdogDefinitionOptions{Root: root, Home: home, LoopBinary: loop, VrooliBinary: d.resolveVrooliBinary(), Username: username})
}

// The following compatibility helpers are deliberately thin delegates. They
// keep older callers compiling while the service definition authority remains
// in platform-go.
func (d *Detector) getSystemdTemplateForService(system bool) string {
	home, _ := d.probe.userHomeDir()
	root := d.resolveVrooliRoot()
	loop := filepath.Join(root, "scenarios", "vrooli-autoheal", "cli", "vrooli-autoheal-loop")
	value, _ := platformgo.RenderWatchdogDefinition("linux", platformgo.WatchdogDefinitionOptions{Root: root, Home: home, LoopBinary: loop, VrooliBinary: d.resolveVrooliBinary(), SystemService: system})
	return value
}

func (d *Detector) resolveVrooliRoot() string { return reporoot.Resolve(d.probe.getenv) }

func (d *Detector) resolveVrooliBinary() string {
	root := d.resolveVrooliRoot()
	candidates := []string{filepath.Join(root, ".vrooli", "build", "vrooli"), filepath.Join(root, "vrooli")}
	if d.probe.goos() == "windows" {
		candidates = []string{filepath.Join(root, ".vrooli", "build", "vrooli.exe"), filepath.Join(root, "vrooli.exe")}
	}
	for _, candidate := range candidates {
		if d.probe.stat(candidate) == nil {
			return candidate
		}
	}
	if path, err := exec.LookPath("vrooli"); err == nil {
		return path
	}
	return "vrooli"
}
