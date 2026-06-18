// Package platform holds the node-agent's cross-platform abstractions so the
// single agent codebase runs on Linux, macOS, and Windows without Linux-only
// assumptions (OT-P0-007). Phase 0 ships the portable pieces every later phase
// builds on: the per-OS state directory and a typed identifier for the native
// service manager. The privileged-helper install adapters (systemd / launchd /
// Windows Service) land in Phase 4 behind ServiceManagerKind.
//
// Everything here uses the cross-platform stdlib (os.UserConfigDir, runtime)
// rather than hardcoded paths or build-tagged files, so the package compiles
// and behaves identically across the CGO_ENABLED=0 cross-compile matrix.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ServiceManagerKind names the platform-native background-service mechanism the
// agent installs itself under. The bootstrap (Phase 2) and the privileged
// provisioning helper (Phase 4) switch on this rather than scattering GOOS
// checks through the codebase.
type ServiceManagerKind string

const (
	ServiceManagerUnknown ServiceManagerKind = "unknown"
	ServiceManagerSystemd ServiceManagerKind = "systemd"         // Linux
	ServiceManagerLaunchd ServiceManagerKind = "launchd"         // macOS
	ServiceManagerWindows ServiceManagerKind = "windows-service" // Windows
)

// NativeServiceManager reports the service manager for the OS the agent is
// running on. It is deliberately a pure function of runtime.GOOS so tests are
// deterministic and cross-builds need no special casing.
func NativeServiceManager() ServiceManagerKind {
	switch runtime.GOOS {
	case "linux":
		return ServiceManagerSystemd
	case "darwin":
		return ServiceManagerLaunchd
	case "windows":
		return ServiceManagerWindows
	default:
		return ServiceManagerUnknown
	}
}

// StateDir returns the directory the agent persists its runtime state in
// (the Ed25519 credential, the pinned control-plane key, a small amount of
// run bookkeeping). It is rooted at the OS-appropriate per-user config
// location via os.UserConfigDir:
//
//	Linux:   $XDG_CONFIG_HOME/vrooli-bridge-agent  (~/.config/...)
//	macOS:   ~/Library/Application Support/vrooli-bridge-agent
//	Windows: %AppData%\vrooli-bridge-agent
//
// A BRIDGE_AGENT_STATE_DIR override takes precedence so a service install
// (which runs as a dedicated non-privileged user, DECISIONS.md) and tests can
// pin an explicit location. The directory is created (0o700) if missing —
// it holds secret key material.
func StateDir() (string, error) {
	if override := os.Getenv("BRIDGE_AGENT_STATE_DIR"); override != "" {
		return ensureDir(override)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return ensureDir(filepath.Join(base, "vrooli-bridge-agent"))
}

func ensureDir(path string) (string, error) {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", fmt.Errorf("create agent state dir %q: %w", path, err)
	}
	return path, nil
}
