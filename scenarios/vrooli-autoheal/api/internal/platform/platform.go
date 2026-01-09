// Package platform provides cross-platform detection and capability checking
// [REQ:PLAT-DETECT-001] [REQ:PLAT-DETECT-002] [REQ:PLAT-DETECT-003] [REQ:PLAT-DETECT-004]
package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// Type represents the detected operating system platform
type Type string

const (
	Linux   Type = "linux"
	Windows Type = "windows"
	MacOS   Type = "macos"
	Other   Type = "other"
)

// Capabilities describes what features are available on the current platform
type Capabilities struct {
	Platform            Type `json:"platform"`
	SupportsRDP         bool `json:"supportsRdp"`
	SupportsSystemd     bool `json:"supportsSystemd"`
	SupportsLaunchd     bool `json:"supportsLaunchd"`
	SupportsWindowsSvc  bool `json:"supportsWindowsServices"`
	IsHeadlessServer    bool `json:"isHeadlessServer"`
	HasDocker           bool `json:"hasDocker"`
	IsWSL               bool `json:"isWsl"`
	SupportsCloudflared bool `json:"supportsCloudflared"`
}

var (
	once       sync.Once
	cachedCaps *Capabilities
)

// Detect returns the current platform capabilities (cached after first call)
func Detect() *Capabilities {
	once.Do(func() {
		cachedCaps = detect()
	})
	return cachedCaps
}

// detect performs the actual platform detection
func detect() *Capabilities {
	caps := &Capabilities{
		Platform: detectPlatform(),
	}

	caps.IsWSL = detectWSL()
	caps.HasDocker = detectDocker()
	caps.SupportsSystemd = detectSystemd()
	caps.SupportsLaunchd = detectLaunchd()
	caps.SupportsWindowsSvc = detectWindowsServices()
	caps.SupportsRDP = detectRDP(caps)
	caps.IsHeadlessServer = detectHeadless()
	caps.SupportsCloudflared = detectCloudflared()

	return caps
}

// detectPlatform returns the OS platform type
func detectPlatform() Type {
	switch runtime.GOOS {
	case "linux":
		return Linux
	case "darwin":
		return MacOS
	case "windows":
		return Windows
	default:
		return Other
	}
}

// detectWSL checks if running inside Windows Subsystem for Linux
func detectWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check for WSL-specific indicators
	// 1. Check /proc/version for Microsoft/WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
			return true
		}
	}

	// 2. Check for WSL interop
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
		return true
	}

	// 3. Check WSL_DISTRO_NAME env var
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}

	return false
}

// detectDocker checks if Docker is available
func detectDocker() bool {
	_, err := exec.LookPath("docker")
	if err != nil {
		return false
	}

	// Try to run docker info to verify daemon is responsive
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// detectSystemd checks if systemd is the init system
func detectSystemd() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check if systemctl is available and systemd is PID 1
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}

	// Check if /run/systemd/system exists (indicates systemd is running)
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}

	// Fallback: check if PID 1 is systemd
	if data, err := os.ReadFile("/proc/1/comm"); err == nil {
		return strings.TrimSpace(string(data)) == "systemd"
	}

	return false
}

// detectLaunchd checks if launchd is available (macOS)
func detectLaunchd() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("launchctl")
	return err == nil
}

// detectWindowsServices checks if Windows service management is available
func detectWindowsServices() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	_, err := exec.LookPath("sc.exe")
	return err == nil
}

// detectRDP checks for RDP/remote desktop support
func detectRDP(caps *Capabilities) bool {
	switch caps.Platform {
	case Linux:
		// Check for xrdp service
		if caps.SupportsSystemd {
			cmd := exec.Command("systemctl", "list-unit-files", "xrdp.service")
			output, err := cmd.Output()
			if err == nil && strings.Contains(string(output), "xrdp") {
				return true
			}
		}
		// Fallback: check if xrdp binary exists
		_, err := exec.LookPath("xrdp")
		return err == nil

	case Windows:
		// Windows has native RDP (TermService)
		return true

	case MacOS:
		// macOS has Screen Sharing but not traditional RDP
		return false

	default:
		return false
	}
}

// detectHeadless checks if running on a headless server (no display capability)
// Note: This is about whether the SYSTEM has display capabilities, not whether
// this process has access to them. A system is not headless if it has a display
// manager running or configured.
func detectHeadless() bool {
	if runtime.GOOS == "linux" {
		// First check if DISPLAY or WAYLAND_DISPLAY is set (process has access)
		if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
			return false
		}

		// Even if this process doesn't have display access, check if a display
		// manager is running on the system (gdm, lightdm, sddm, etc.)
		displayManagers := []string{"gdm", "gdm3", "lightdm", "sddm", "lxdm", "xdm"}
		for _, dm := range displayManagers {
			cmd := exec.Command("systemctl", "is-active", dm)
			if output, err := cmd.Output(); err == nil {
				if strings.TrimSpace(string(output)) == "active" {
					return false // Display manager is running, not headless
				}
			}
		}

		// Also check for graphical.target being the default
		cmd := exec.Command("systemctl", "get-default")
		if output, err := cmd.Output(); err == nil {
			if strings.Contains(string(output), "graphical.target") {
				return false // Graphical target is default, not headless
			}
		}

		return true // No display indicators found
	}

	if runtime.GOOS == "darwin" {
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return true
		}
	}

	// On Windows, assume desktop unless specific headless indicators
	if runtime.GOOS == "windows" {
		// Windows Server Core typically lacks explorer.exe
		if _, err := exec.LookPath("explorer.exe"); err != nil {
			return true
		}
	}

	return false
}

// detectCloudflared checks if cloudflared is available
func detectCloudflared() bool {
	_, err := exec.LookPath("cloudflared")
	return err == nil
}
