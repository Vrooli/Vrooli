package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const DefaultPrivilegeBrokerSocketPath = "/run/vrooli/privilege-broker.sock"

// PrivilegeBrokerSocketPath is the single cross-module socket policy. The
// Linux broker is a system service, so its endpoint must not depend on the
// caller's user-session environment (in particular XDG_RUNTIME_DIR).
func PrivilegeBrokerSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\vrooli-privilege-broker`
	}
	if configured := strings.TrimSpace(os.Getenv("VROOLI_PRIVILEGE_BROKER_SOCKET")); configured != "" {
		return configured
	}
	if runtime.GOOS == "linux" {
		return DefaultPrivilegeBrokerSocketPath
	}
	if runtimeRoot := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeRoot != "" {
		return filepath.Join(runtimeRoot, "vrooli", "privilege-broker.sock")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.TempDir(), "vrooli", "privilege-broker.sock")
	}
	return DefaultPrivilegeBrokerSocketPath
}
