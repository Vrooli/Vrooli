package packagegov

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type packageJSON struct {
	Scripts              map[string]string `json:"scripts"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

type DiscoveryReport struct {
	Dependents []Dependent       `json:"dependents"`
	Issues     []ValidationIssue `json:"issues"`
}

type consumerScope string

const (
	scopeScenario consumerScope = "scenario"
	scopeTemplate consumerScope = "template"
	scopeResource consumerScope = "resource"
)

func DiscoverDependents(root string, pkg Package) (DiscoveryReport, error) {
	inv, err := buildDependencyInventory(root, []Package{pkg})
	if err != nil {
		return DiscoveryReport{}, err
	}
	return inv.reportFor(pkg), nil
}

func walkPackageJSONs(root string, fn func(path string) error) error {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if shouldSkipDiscoveryDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if d.Name() != "package.json" {
			return nil
		}
		return fn(path)
	})
}

func walkGoMods(root string, fn func(path string) error) error {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}
		if d.IsDir() {
			if shouldSkipDiscoveryDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		return fn(path)
	})
}

func readPackageJSON(path string) (packageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return packageJSON{}, fmt.Errorf("read %s: %w", path, err)
	}
	var manifest packageJSON
	if err := json.Unmarshal(data, &manifest); err != nil {
		return packageJSON{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return manifest, nil
}

func readGoMod(path string) (parsedGoMod, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return parsedGoMod{}, fmt.Errorf("read %s: %w", path, err)
	}
	return parseGoMod(string(data)), nil
}

func packageModuleIdentifiers(pkg Package) []string {
	identifiers := make([]string, 0, len(pkg.Manifest.Package.ModuleIdentifiers))
	seen := make(map[string]struct{}, len(pkg.Manifest.Package.ModuleIdentifiers))
	appendUnique := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		identifiers = append(identifiers, id)
	}
	for _, id := range pkg.Manifest.Package.ModuleIdentifiers {
		appendUnique(id)
	}
	for _, output := range pkg.Manifest.Package.GeneratedOutputs {
		for _, id := range output.Identifiers {
			appendUnique(id)
		}
	}
	return identifiers
}

func classifyPackageJSONAdoption(pkg Package, target, version string) AdoptionMode {
	version = strings.TrimSpace(version)
	for _, output := range pkg.Manifest.Package.GeneratedOutputs {
		for _, id := range output.Identifiers {
			if strings.TrimSpace(id) == strings.TrimSpace(target) {
				return ModeGeneratedArtifact
			}
		}
	}
	switch {
	case strings.HasPrefix(version, "file:"):
		return ModeFileDependency
	case version == "workspace:*":
		return ModePublishedSemver
	default:
		return ModePublishedSemver
	}
}

func postinstallTouchesSharedPackages(script string) bool {
	script = strings.TrimSpace(script)
	if script == "" {
		return false
	}
	return strings.Contains(script, "node_modules/@vrooli") || strings.Contains(script, "packages/")
}

func classifyConsumer(root, path string, scope consumerScope) (string, ConsumerClass) {
	rel, _ := filepath.Rel(root, path)
	slash := filepath.ToSlash(rel)
	parts := strings.Split(slash, "/")
	if scope == scopeScenario && len(parts) >= 3 {
		name := parts[1]
		component := parts[2]
		switch {
		case strings.HasPrefix(component, "ui"):
			return name, ConsumerScenarioUI
		case component == "playwright-driver":
			return name, ConsumerScenarioTest
		default:
			return name, ConsumerScenarioTest
		}
	}
	if scope == scopeTemplate && len(parts) >= 3 {
		name := parts[2]
		switch {
		case strings.Contains(slash, "/ui/"):
			return name, ConsumerTemplateUI
		case strings.Contains(slash, "/api/"):
			return name, ConsumerTemplateAPI
		case strings.Contains(slash, "/cli/"):
			return name, ConsumerTemplateCLI
		default:
			return name, ConsumerTemplateUI
		}
	}
	if scope == scopeResource && len(parts) >= 2 {
		return parts[1], ConsumerResourceRuntime
	}
	return filepath.Base(filepath.Dir(path)), ConsumerInternalPlatform
}

func classifyGoConsumer(root, path string, scope consumerScope) (string, ConsumerClass) {
	rel, _ := filepath.Rel(root, path)
	slash := filepath.ToSlash(rel)
	parts := strings.Split(slash, "/")
	if scope == scopeScenario && len(parts) >= 3 {
		name := parts[1]
		component := parts[2]
		switch {
		case component == "api" || component == "runtime":
			return name, ConsumerScenarioAPI
		case component == "cli":
			return name, ConsumerScenarioCLI
		default:
			return name, ConsumerScenarioAPI
		}
	}
	if scope == scopeTemplate && len(parts) >= 3 {
		name := parts[2]
		switch {
		case strings.Contains(slash, "/api/"):
			return name, ConsumerTemplateAPI
		case strings.Contains(slash, "/cli/"):
			return name, ConsumerTemplateCLI
		default:
			return name, ConsumerInternalPlatform
		}
	}
	if scope == scopeResource && len(parts) >= 2 {
		return parts[1], ConsumerResourceRuntime
	}
	return filepath.Base(filepath.Dir(path)), ConsumerInternalPlatform
}

func consumerRootFromFile(root, path string, scope consumerScope) string {
	rel, _ := filepath.Rel(root, path)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if scope == scopeScenario && len(parts) >= 2 {
		return filepath.Join(root, parts[0], parts[1])
	}
	if scope == scopeTemplate && len(parts) >= 3 {
		return filepath.Join(root, parts[0], parts[1], parts[2])
	}
	if scope == scopeResource && len(parts) >= 2 {
		return filepath.Join(root, parts[0], parts[1])
	}
	return filepath.Dir(path)
}

func shouldSkipDiscoveryDir(name string) bool {
	name = strings.TrimSpace(name)
	switch name {
	case "node_modules", "dist", "build", "bundle", "bin", "coverage", ".turbo", ".vite", "generated", "testdata", "logs", "artifacts":
		return true
	}
	if strings.HasPrefix(name, ".git") {
		return true
	}
	return strings.Contains(strings.ToLower(name), "backup")
}

type parsedGoMod struct {
	requires map[string]string
	replaces map[string]string
}

type goModDependency struct {
	Target             string
	Version            string
	Present            bool
	HasGovernedReplace bool
}

var (
	goModRequireLinePattern = regexp.MustCompile(`^\s*([A-Za-z0-9._/\-]+)\s+([^\s]+)\s*(?://.*)?$`)
	goModReplaceLinePattern = regexp.MustCompile(`^\s*([A-Za-z0-9._/\-]+)(?:\s+[^\s]+)?\s*=>\s*([^\s]+)(?:\s+[^\s]+)?\s*(?://.*)?$`)
)

func parseGoMod(content string) parsedGoMod {
	mod := parsedGoMod{
		requires: make(map[string]string),
		replaces: make(map[string]string),
	}

	var inRequireBlock bool
	var inReplaceBlock bool

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		switch line {
		case "require (":
			inRequireBlock = true
			inReplaceBlock = false
			continue
		case "replace (":
			inReplaceBlock = true
			inRequireBlock = false
			continue
		case ")":
			inRequireBlock = false
			inReplaceBlock = false
			continue
		}

		switch {
		case strings.HasPrefix(line, "require "):
			parseGoRequireLine(mod.requires, strings.TrimSpace(strings.TrimPrefix(line, "require ")))
		case strings.HasPrefix(line, "replace "):
			parseGoReplaceLine(mod.replaces, strings.TrimSpace(strings.TrimPrefix(line, "replace ")))
		case inRequireBlock:
			parseGoRequireLine(mod.requires, line)
		case inReplaceBlock:
			parseGoReplaceLine(mod.replaces, line)
		}
	}

	return mod
}

func parseGoRequireLine(bucket map[string]string, line string) {
	matches := goModRequireLinePattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return
	}
	bucket[matches[1]] = matches[2]
}

func parseGoReplaceLine(bucket map[string]string, line string) {
	matches := goModReplaceLinePattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return
	}
	bucket[matches[1]] = matches[2]
}

func (m parsedGoMod) dependencyFor(goModPath, repoRoot, module string) goModDependency {
	version, hasRequire := m.requires[module]
	replaceTarget, hasReplace := m.replaces[module]
	if !hasRequire && !hasReplace {
		return goModDependency{}
	}
	if hasReplace {
		if isGovernedReplaceTarget(goModPath, repoRoot, replaceTarget) {
			return goModDependency{Target: module, Version: version, Present: true, HasGovernedReplace: true}
		}
		return goModDependency{Target: module, Version: version, Present: true}
	}
	return goModDependency{Target: module, Version: version, Present: true}
}

func isGovernedReplaceTarget(goModPath, repoRoot, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	if !(strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/")) {
		return false
	}
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(goModPath), resolved)
	}
	resolved = filepath.Clean(resolved)
	packagesRoot := filepath.Clean(filepath.Join(repoRoot, "packages"))
	rel, err := filepath.Rel(packagesRoot, resolved)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(filepath.ToSlash(rel), "..")
}
