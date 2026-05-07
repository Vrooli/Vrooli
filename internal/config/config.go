package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const (
	TemplateBaseDirEnvVar = "SCENARIO_TEMPLATE_BASE_DIR"
	DesignBaseDirEnvVar   = "SCENARIO_DESIGN_BASE_DIR"
)

// HomeDir returns the home directory whose ~/.vrooli/, ~/.config/vrooli/,
// and ~/.vrooli/bin/ vrooli should read and write.
//
// Sudo handling: when vrooli is invoked as `sudo vrooli ...`, the kernel
// process is real-user-id 0 and `$HOME` has been overwritten to `/root`
// (sudo's default). Naively returning `$HOME` would point every per-user
// install path (binaries, ports.json, scenario state) at /root and break
// the operator's normal-user shell after they drop sudo. We detect this
// case and resolve the *invoking* user's home from /etc/passwd via
// hostreqkit.InvokingUserHomeDir — the same primitive sudo-aware
// handlers (go install, npm install) use to drop privileges.
//
// Behavior:
//   - Non-root process: read $HOME, fall back to os.UserHomeDir.
//   - Root process *not* via sudo (real root login): no $SUDO_USER, falls
//     through to $HOME=/root, which is correct.
//   - Root process via sudo: returns $SUDO_USER's home from /etc/passwd.
func HomeDir() (string, error) {
	if home, ok := invokingUserHomeWhenSudoed(); ok {
		return home, nil
	}
	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		return home, nil
	}
	return os.UserHomeDir()
}

func invokingUserHomeWhenSudoed() (string, bool) {
	if !hostreqkit.RunningAsRootFn() {
		return "", false
	}
	if strings.TrimSpace(os.Getenv("SUDO_USER")) == "" {
		return "", false
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", false
	}
	return home, true
}

func VrooliDir(home string) string {
	return filepath.Join(home, ".vrooli")
}

func RepoConfigDir(root string) string {
	return filepath.Join(root, ".vrooli")
}

func TemplateBaseDir(root string) string {
	if override := strings.TrimSpace(os.Getenv(TemplateBaseDirEnvVar)); override != "" {
		if filepath.IsAbs(override) {
			return filepath.Clean(override)
		}
		return filepath.Clean(filepath.Join(root, filepath.FromSlash(override)))
	}
	return filepath.Join(root, "templates", "scenarios")
}

func DesignBaseDir(root string) string {
	if override := strings.TrimSpace(os.Getenv(DesignBaseDirEnvVar)); override != "" {
		if filepath.IsAbs(override) {
			return filepath.Clean(override)
		}
		return filepath.Clean(filepath.Join(root, filepath.FromSlash(override)))
	}
	return filepath.Join(root, "templates", "design")
}
