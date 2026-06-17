// Package hostbin resolves external command binaries cross-platform.
//
// Unit Health runs third-party tools (package managers, bats) that may live on
// PATH or in a per-user bin directory a sudo'd PATH cannot see. This package
// centralizes that resolution so planners and the executor agree on what is
// available, and so platform behavior (PATHEXT/.exe on Windows, ~/.local/bin on
// POSIX) lives in one place. The repo-root internal/hostreqkit does the same
// for the platform binary, but Go's internal-package rule makes it unimportable
// from this scenario's module — this is the scenario-local equivalent, not a
// fork of it.
package hostbin

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Seams for deterministic tests.
var (
	lookPath    = exec.LookPath
	userHomeDir = os.UserHomeDir
)

// Resolve returns the first candidate command available to the invoking user
// and true, or "", false when none resolve. It probes PATH first (LookPath
// honors PATHEXT/.exe on Windows), then the user's standard per-user bin dirs
// which a sudo'd PATH commonly misses. The bare candidate name is returned; the
// caller invokes it via PATH-based exec.
func Resolve(candidates []string) (string, bool) {
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, err := lookPath(c); err == nil {
			return c, true
		}
	}
	home, err := userHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	// ~/.local/bin is the canonical XDG location (and where Vrooli symlinks
	// live); ~/go/bin is Go's default install target; ~/bin is the older
	// convention. On Windows these are simply absent and skipped.
	userDirs := []string{filepath.Join(".local", "bin"), filepath.Join("go", "bin"), "bin"}
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		for _, dir := range userDirs {
			full := filepath.Join(home, dir, c)
			if info, statErr := os.Stat(full); statErr == nil && !info.IsDir() {
				return c, true
			}
			if info, statErr := os.Stat(full + ".exe"); statErr == nil && !info.IsDir() {
				return c, true
			}
		}
	}
	return "", false
}

// Available reports whether name resolves on this host.
func Available(name string) bool {
	_, ok := Resolve([]string{name})
	return ok
}
