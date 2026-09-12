package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const DefaultPrivilegeBrokerSocketPath = "/run/vrooli/privilege-broker.sock"

// PrivilegeBrokerSocketPath is the single cross-module socket policy. User
// session platforms use their runtime directory; Linux keeps /run as fallback.
func PrivilegeBrokerSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\vrooli-privilege-broker`
	}
	if runtimeRoot := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeRoot != "" {
		return filepath.Join(runtimeRoot, "vrooli", "privilege-broker.sock")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.TempDir(), "vrooli", "privilege-broker.sock")
	}
	return DefaultPrivilegeBrokerSocketPath
}
