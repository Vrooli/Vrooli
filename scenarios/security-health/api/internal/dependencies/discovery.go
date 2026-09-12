package dependencies

import (
	"encoding/json"
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
		case "package-lock.json", "npm-shrinkwrap.json":
			records = append(records, parsePackageLock(path, rel, scenarioID, EcosystemNPM)...)
		case "yarn.lock":
			records = append(records, parseYarnLock(path, rel, scenarioID)...)
		case "bun.lock", "bun.lockb":
			records = append(records, parseBunLock(path, rel, scenarioID)...)
		case "requirements.txt", "Pipfile.lock", "poetry.lock", "pyproject.toml":
			records = append(records, parsePythonManifest(path, rel, scenarioID)...)
		case "Cargo.lock":
			records = append(records, parseCargoLock(path, rel, scenarioID)...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

type packageLockJSON struct {
	Packages map[string]struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"packages"`
	Dependencies map[string]packageLockDependency `json:"dependencies"`
}

type packageLockDependency struct {
	Version      string                           `json:"version"`
	Dependencies map[string]packageLockDependency `json:"dependencies"`
}

func parsePackageLock(absPath, relPath, scenario string, ecosystem Ecosystem) []DependencyRecord {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	var lock packageLockJSON
	if json.Unmarshal(raw, &lock) != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []DependencyRecord
	for key, pkg := range lock.Packages {
		if key == "" || pkg.Version == "" || pkg.Name == "" {
			continue
		}
		addDependencyRecord(&out, seen, scenario, ecosystem, pkg.Name, pkg.Version, relPath)
	}
	var walk func(map[string]packageLockDependency)
	walk = func(deps map[string]packageLockDependency) {
		for name, dep := range deps {
			if dep.Version != "" {
				addDependencyRecord(&out, seen, scenario, ecosystem, name, dep.Version, relPath)
			}
			walk(dep.Dependencies)
		}
	}
	walk(lock.Dependencies)
	return out
}

func parseBunLock(absPath, relPath, scenario string) []DependencyRecord {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	var document map[string]any
	if json.Unmarshal(raw, &document) != nil {
		// bun.lockb is binary and intentionally produces no false-clean
		// dependency records. The ecosystem adapter still reports that Bun
		// was discovered; a future binary decoder can add evidence here.
		return nil
	}
	var out []DependencyRecord
	seen := map[string]struct{}{}
	var walk func(any)
	walk = func(value any) {
		switch node := value.(type) {
		case map[string]any:
			for key, child := range node {
				if key == "dependencies" || key == "devDependencies" || key == "optionalDependencies" {
					if deps, ok := child.(map[string]any); ok {
						for name, spec := range deps {
							if version := bunVersion(spec); version != "" {
								addDependencyRecord(&out, seen, scenario, EcosystemBun, name, version, relPath)
							}
							walk(spec)
						}
						continue
					}
				}
				walk(child)
			}
		case []any:
			for _, child := range node {
				walk(child)
			}
		}
	}
	walk(document)
	return out
}

func bunVersion(value any) string {
	switch spec := value.(type) {
	case string:
		return strings.TrimSpace(spec)
	case []any:
		for _, item := range spec {
			if version, ok := item.(string); ok && strings.TrimSpace(version) != "" {
				return strings.TrimSpace(version)
			}
		}
	case map[string]any:
		for _, key := range []string{"version", "resolved", "reference"} {
			if version, ok := spec[key].(string); ok && strings.TrimSpace(version) != "" {
				return strings.TrimSpace(version)
			}
		}
	}
	return ""
}

func parseYarnLock(absPath, relPath, scenario string) []DependencyRecord {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	var out []DependencyRecord
	seen := map[string]struct{}{}
	var names []string
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "version") {
			for _, selector := range strings.Split(strings.TrimSuffix(trimmed, ":"), ",") {
				selector = strings.Trim(strings.TrimSpace(selector), "\"")
				if at := strings.LastIndex(selector, "@"); at > 0 {
					names = append(names, selector[:at])
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "version ") && len(names) > 0 {
			version := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "version")), "\"")
			for _, name := range names {
				addDependencyRecord(&out, seen, scenario, EcosystemYarn, name, version, relPath)
			}
			names = nil
		}
	}
	return out
}

func parsePythonManifest(absPath, relPath, scenario string) []DependencyRecord {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	var out []DependencyRecord
	seen := map[string]struct{}{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "{") {
			continue
		}
		name, version := parsePythonRequirement(line)
		if name != "" && version != "" {
			addDependencyRecord(&out, seen, scenario, EcosystemPython, name, version, relPath)
		}
	}
	return out
}

func parsePythonRequirement(line string) (string, string) {
	line = strings.Trim(strings.TrimSpace(line), "\"',")
	for _, marker := range []string{"==", ">=", "<=", "~=", "!=", ">", "<"} {
		if i := strings.Index(line, marker); i > 0 {
			name := strings.TrimSpace(line[:i])
			version := strings.TrimSpace(strings.Fields(line[i+len(marker):])[0])
			name = strings.SplitN(name, "[", 2)[0]
			return name, version
		}
	}
	return "", ""
}

func parseCargoLock(absPath, relPath, scenario string) []DependencyRecord {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	var out []DependencyRecord
	seen := map[string]struct{}{}
	var name, version string
	flush := func() {
		if name != "" && version != "" {
			addDependencyRecord(&out, seen, scenario, EcosystemRust, name, version, relPath)
		}
		name, version = "", ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[[package]]" {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "name =") {
			name = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "name =")), "\"")
		}
		if strings.HasPrefix(trimmed, "version =") {
			version = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "version =")), "\"")
		}
	}
	flush()
	return out
}

func addDependencyRecord(out *[]DependencyRecord, seen map[string]struct{}, scenario string, ecosystem Ecosystem, name, version, source string) {
	key := string(ecosystem) + "|" + name + "|" + version
	if name == "" || version == "" {
		return
	}
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*out = append(*out, DependencyRecord{Scenario: scenario, Ecosystem: ecosystem, Name: name, Version: version, SourceFile: source})
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
