// Package scenarioroot resolves the landing-page-business-suite scenario
// directory for code that reads checked-in data files (the fallback landing
// config, the variant space).
//
// The control plane exports VROOLI_SCENARIO_DIR and VROOLI_ROOT to every
// scenario process, so those come first. runtime.Caller is only a development
// fallback: a -trimpath build reports a relative source path, and joining
// ".." onto it silently points at the working directory instead of the
// scenario, which is how the on-disk fallback config went unread.
package scenarioroot

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const scenarioName = "landing-page-business-suite"

// Resolve returns the absolute scenario directory, or "" when nothing reliable
// is available; callers then fall back to working-directory candidates.
func Resolve() string {
	if dir := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO_DIR")); dir != "" && filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" && filepath.IsAbs(root) {
		candidate := filepath.Join(root, "scenarios", scenarioName)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return fromSource()
}

// fromSource climbs from this file to the scenario directory; empty under -trimpath.
func fromSource() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(file) {
		return ""
	}
	// <scenario>/api/internal/scenarioroot/scenarioroot.go
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
