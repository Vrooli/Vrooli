package validation

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Substrate describes which scannable technologies a scenario actually
// contains. Scanners declare which substrates they apply to via Applies; the
// Service only runs a scanner when its substrate is present, so a Go-only
// scenario never invokes pnpm-audit and a UI-only scenario never invokes gosec.
type Substrate struct {
	// Go is true when a go.mod is present anywhere in the scenario tree.
	Go bool
	// PnpmUI is true when a pnpm-lock.yaml is present (the UI dependency graph
	// pnpm audit / osv-scanner can read).
	PnpmUI bool
	// GoModDirs are the directories (relative to the scenario root) that
	// contain a go.mod — gosec/govulncheck run per-module.
	GoModDirs []string
	// PnpmLockDirs are the directories (relative to the scenario root) that
	// contain a pnpm-lock.yaml.
	PnpmLockDirs []string
	// Unsupported lists substrates we observed but do not scan in v1 (e.g.
	// Python). Surfaced as an INFO observation, never a failure.
	Unsupported []string
}

// skipDirs are directories whose contents never represent first-party source
// or first-party dependency manifests worth scanning.
var skipDirs = map[string]struct{}{
	"node_modules": {},
	"vendor":       {},
	".git":         {},
	"dist":         {},
	"build":        {},
	".cache":       {},
}

// DetectSubstrate walks a scenario directory and classifies the substrates it
// contains by the presence of canonical lockfiles/manifests. The walk skips
// vendored and build-output trees so a vendored Python dep doesn't make a Go
// scenario look polyglot.
func DetectSubstrate(scenarioDir string) (Substrate, error) {
	var s Substrate
	goDirs := map[string]struct{}{}
	pnpmDirs := map[string]struct{}{}
	unsupported := map[string]struct{}{}

	err := filepath.WalkDir(scenarioDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entries are skipped, not fatal
		}
		if d.IsDir() {
			name := d.Name()
			if _, skip := skipDirs[name]; skip {
				return filepath.SkipDir
			}
			// Skip hidden directories except the scenario root itself.
			if path != scenarioDir && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(scenarioDir, filepath.Dir(path))
		switch d.Name() {
		case "go.mod":
			s.Go = true
			goDirs[rel] = struct{}{}
		case "pnpm-lock.yaml":
			s.PnpmUI = true
			pnpmDirs[rel] = struct{}{}
		case "requirements.txt", "pyproject.toml", "Pipfile":
			unsupported["python"] = struct{}{}
		case "Cargo.toml":
			unsupported["rust"] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return Substrate{}, err
	}

	s.GoModDirs = sortedKeys(goDirs)
	s.PnpmLockDirs = sortedKeys(pnpmDirs)
	s.Unsupported = sortedKeys(unsupported)
	return s, nil
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// resolveScenarioDir joins a repo root and scenario id into the absolute
// scenario directory, verifying it exists and is a directory. Returns the
// path and ok=false when the scenario does not exist.
func resolveScenarioDir(repoRoot, scenario string) (string, bool) {
	dir := filepath.Join(repoRoot, "scenarios", scenario)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return dir, true
}
