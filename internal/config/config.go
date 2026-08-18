package config

import (
	"os"
	osuser "os/user"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
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
	// Service managers and controlled SSH sessions can intentionally omit HOME.
	// os.UserHomeDir follows that environment variable on Unix, so use the
	// authenticated process identity as the portable fallback before returning
	// its less-actionable "$HOME is not defined" error.
	if current, err := osuser.Current(); err == nil {
		if home := strings.TrimSpace(current.HomeDir); home != "" {
			return home, nil
		}
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

// VrooliHome returns the absolute operator runtime-home root (the sudo-aware
// $HOME joined with the contract's runtime_home dir name). This is the single
// internal resolution entrypoint for the runtime home: sudo-awareness comes
// from HomeDir(); the structural name comes from the repo contract. Contract
// load failure is a hard error (no fallback).
func VrooliHome() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return repocontract.VrooliUserRoot(home)
}

// VrooliPath resolves a well-known runtime-home entry (by repocontract.HomeKey*)
// and optionally joins consumer-specific sub-components beneath it. The entry
// name always comes from the contract; sub-components are caller structure under
// that contract-defined root.
func VrooliPath(key string, sub ...string) (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	base, err := repocontract.RuntimeHomeEntryPath(home, key)
	if err != nil {
		return "", err
	}
	if len(sub) == 0 {
		return base, nil
	}
	return filepath.Join(append([]string{base}, sub...)...), nil
}

// VrooliScopedPath expands a parameterized runtime-home template (by
// repocontract.Scoped*) against the sudo-aware home.
func VrooliScopedPath(key string, params map[string]string) (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return repocontract.RuntimeHomeScopedPath(home, key, params)
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
