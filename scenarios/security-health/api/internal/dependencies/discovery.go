package dependencies

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
	"gopkg.in/yaml.v3"
)

// skipDirs are trees that never hold first-party dependency manifests worth
// indexing (vendored copies, build output, VCS metadata).
var skipDirs = map[string]struct{}{
	"node_modules": {},
	"vendor":       {},
	".git":         {},
	"dist":         {},
	"build":        {},
	".cache":       {},
}

// DiscoverFleet walks every scenario under repoRoot/scenarios and returns the
// union of their declared dependencies (Go modules + pnpm packages),
// unannotated. Vuln status is added later by the annotator.
func DiscoverFleet(repoRoot string) ([]DependencyRecord, error) {
	scenariosDir := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("read scenarios dir: %w", err)
	}
	var all []DependencyRecord
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		recs, err := DiscoverScenario(filepath.Join(scenariosDir, e.Name()), e.Name())
		if err != nil {
			// One unparseable scenario must not sink the whole fleet scan.
			continue
		}
		all = append(all, recs...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Key() < all[j].Key() })
	return all, nil
}

// DiscoverScenario walks one scenario tree and parses every go.mod and
// pnpm-lock.yaml it finds (skipping vendored/build trees).
func DiscoverScenario(scenarioDir, scenarioID string) ([]DependencyRecord, error) {
	var records []DependencyRecord
	err := filepath.WalkDir(scenarioDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			if path != scenarioDir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(scenarioDir, path)
		rel = filepath.ToSlash(rel)
		switch d.Name() {
		case "go.mod":
			records = append(records, parseGoMod(path, rel, scenarioID)...)
		case "pnpm-lock.yaml":
			records = append(records, parsePnpmLock(path, rel, scenarioID)...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

// parseGoMod extracts every required module (direct + indirect) from a go.mod.
func parseGoMod(absPath, relPath, scenario string) []DependencyRecord {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	f, err := modfile.Parse(absPath, raw, nil)
	if err != nil {
		return nil
	}
	out := make([]DependencyRecord, 0, len(f.Require))
	for _, r := range f.Require {
		if r == nil {
			continue
		}
		out = append(out, DependencyRecord{
			Scenario:   scenario,
			Ecosystem:  EcosystemGo,
			Name:       r.Mod.Path,
			Version:    r.Mod.Version,
			SourceFile: relPath,
		})
	}
	return out
}

// pnpmLockFile is the minimal shape we read from pnpm-lock.yaml: the package
// keys under `packages:` encode name@version (v9 lockfile format).
type pnpmLockFile struct {
	Packages map[string]any `yaml:"packages"`
}

// parsePnpmLock extracts package name@version pairs from a pnpm-lock.yaml.
func parsePnpmLock(absPath, relPath, scenario string) []DependencyRecord {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	var lock pnpmLockFile
	if err := yaml.Unmarshal(raw, &lock); err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]DependencyRecord, 0, len(lock.Packages))
	for key := range lock.Packages {
		name, version := splitPnpmKey(key)
		if name == "" || version == "" {
			continue
		}
		dedup := name + "@" + version
		if _, ok := seen[dedup]; ok {
			continue
		}
		seen[dedup] = struct{}{}
		out = append(out, DependencyRecord{
			Scenario:   scenario,
			Ecosystem:  EcosystemNPM,
			Name:       name,
			Version:    version,
			SourceFile: relPath,
		})
	}
	return out
}

// splitPnpmKey parses a pnpm v9 package key into (name, version). Keys look
// like "esbuild@0.21.5", "@scope/pkg@1.2.3", or "vite@5.0.0(peer@1.0.0)" — the
// peer-dependency suffix in parentheses is stripped.
func splitPnpmKey(key string) (name, version string) {
	key = strings.TrimPrefix(key, "/")
	if i := strings.IndexByte(key, '('); i >= 0 {
		key = key[:i]
	}
	at := strings.LastIndexByte(key, '@')
	if at <= 0 { // not found, or leading '@' of an unversioned scoped name
		return "", ""
	}
	return key[:at], key[at+1:]
}
