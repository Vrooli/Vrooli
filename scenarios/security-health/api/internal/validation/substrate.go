package validation

import (
	"context"
	"fmt"
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
	// GoPackagePatterns optionally narrows the package patterns run within a
	// module. Keys are GoModDirs entries. The default is ["./..."]. This lets
	// repository targets bound a root module to owned source trees without
	// pretending each package directory is a separate module.
	GoPackagePatterns map[string][]string
	// PnpmLockDirs are the directories (relative to the scenario root) that
	// contain a pnpm-lock.yaml.
	PnpmLockDirs []string
	// Unsupported lists substrates we observed but do not scan in v1 (e.g.
	// an unregistered ecosystem). Surfaced as an INFO observation, never a
	// failure.
	Unsupported []string
	// Targets are the adapter-driven construction facts used by scanners and
	// policy. They are independent of directory names such as api/ or ui/.
	Targets []EcosystemTarget
}

// FactDiscovery is the neutral seam for the Code Facts scenario. Production
// wiring may provide Code Facts' bounded target/parse-unit projection; direct
// CLI validation keeps the filesystem detector as an explicit degraded
// fallback when that provider is unavailable.
type FactDiscovery func(context.Context, string) ([]FactTarget, error)

// DetectSubstrateFromFacts converts Code Facts evidence into the scanner
// substrate shape without importing Code Facts transport or generated types.
func DetectSubstrateFromFacts(facts []FactTarget) Substrate {
	registry := DefaultAdapterRegistry()
	targets := registry.DiscoverFromFacts(facts)
	s := Substrate{Targets: targets}
	unsupported := map[string]struct{}{}
	for _, target := range targets {
		if target.Coverage == CoverageUnsupported || target.Coverage == CoverageUnknown {
			unsupported[string(target.Ecosystem)] = struct{}{}
		}
		for _, manifest := range target.Manifests {
			base := filepath.Base(manifest)
			dir := filepath.ToSlash(filepath.Dir(manifest))
			if dir == "." {
				dir = ""
			}
			switch base {
			case "go.mod":
				s.Go = true
				s.GoModDirs = append(s.GoModDirs, dir)
			case "pnpm-lock.yaml":
				s.PnpmUI = true
				s.PnpmLockDirs = append(s.PnpmLockDirs, dir)
			}
		}
	}
	s.GoModDirs = dedupeStrings(s.GoModDirs)
	s.PnpmLockDirs = dedupeStrings(s.PnpmLockDirs)
	s.Unsupported = sortedKeys(unsupported)
	return s
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
		}
		return nil
	})
	if err != nil {
		return Substrate{}, err
	}

	s.GoModDirs = sortedKeys(goDirs)
	s.PnpmLockDirs = sortedKeys(pnpmDirs)
	registry := DefaultAdapterRegistry()
	targets, err := registry.Discover(scenarioDir)
	if err != nil {
		return Substrate{}, fmt.Errorf("discover ecosystem adapters: %w", err)
	}
	s.Targets = targets
	for _, target := range targets {
		if target.Coverage == CoverageUnsupported || target.Coverage == CoverageUnknown {
			unsupported[string(target.Ecosystem)] = struct{}{}
		}
	}
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
