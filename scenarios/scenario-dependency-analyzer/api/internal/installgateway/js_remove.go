package installgateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResolveRemoval plans the removal of direct JavaScript dependencies from one
// surface. It reuses the staged install machinery: the manager removes the
// packages from a staging copy of the inputs, admission measures what that
// leaves to link, and apply publishes the staged manifest and lockfile and
// relinks node_modules from the frozen lock. The same store, environment and
// rollback guarantees therefore hold for removal as for installation.
func ResolveRemoval(repoRoot, scenario, surface, ecosystem string, packages []string) (Resolution, error) {
	if err := validateScenarioName(scenario); err != nil {
		return Resolution{}, err
	}
	if CanonicalEcosystem(ecosystem) != "npm" {
		return Resolution{}, fmt.Errorf("deps remove supports npm packages on pnpm and npm surfaces, not %q (the Go root-module requirement verb is `deps remove --scenario vrooli --surface control-plane --ecosystem go --module <path>`)", ecosystem)
	}
	surface, err := normalizedSurface(surface)
	if err != nil {
		return Resolution{}, err
	}
	if surface == "control-plane" || strings.HasPrefix(surface, "template/") {
		return Resolution{}, fmt.Errorf("deps remove does not apply to the %s surface", surface)
	}
	names, err := lockedNames(packages, "remove")
	if err != nil {
		return Resolution{}, err
	}
	surfaceRoot := resolveSurfaceRoot(repoRoot, scenario, surface)
	if surfaceRoot == "" {
		return Resolution{}, surfaceNotFoundError(repoRoot, scenario, surface)
	}
	manager := jsManager(surfaceRoot)
	switch manager {
	case "pnpm", "npm":
	case "":
		return Resolution{}, fmt.Errorf("javascript package manager evidence is missing in %s; provide exactly one supported lockfile", surfaceRoot)
	default:
		return Resolution{}, fmt.Errorf("deps remove supports pnpm and npm surfaces; %s uses %s", surfaceRoot, manager)
	}
	manifest := filepath.Join(surfaceRoot, "package.json")
	if err := requireDeclared(manifest, names); err != nil {
		return Resolution{}, err
	}
	argv := []string{manager, "remove"}
	switch manager {
	case "npm":
		argv = []string{"npm", "uninstall", "--ignore-scripts"}
	case "pnpm":
		if isSharedPackageRoot(repoRoot, surfaceRoot) && !fileExists(filepath.Join(surfaceRoot, "pnpm-workspace.yaml")) {
			argv = append(argv, "--ignore-workspace")
		}
	}
	argv = append(argv, names...)
	return Resolution{
		RepositoryRoot:  repoRoot,
		SurfaceRoot:     surfaceRoot,
		PackageManager:  manager,
		ManifestPath:    manifest,
		PackageName:     strings.Join(names, ", "),
		RemovedPackages: names,
		Argv:            argv,
		Profile:         SafeProfileFor(manager, argv),
	}, nil
}

// lockedNames validates and de-duplicates the package names a lockfile
// operation was asked to act on. verb is "remove" or "update", for messages.
func lockedNames(packages []string, verb string) ([]string, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("name at least one package to %s, for example npm/three", verb)
	}
	seen := map[string]bool{}
	var names []string
	for _, name := range packages {
		name = strings.TrimSpace(name)
		if err := validatePackageSpec(name, ""); err != nil {
			return nil, err
		}
		if strings.Contains(name, "@") && !strings.HasPrefix(name, "@") || strings.Count(name, "@") > 1 {
			return nil, fmt.Errorf("%q carries a version; name the package only (the manifest and lockfile decide which version is used)", name)
		}
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names, nil
}

// requireDeclared refuses a removal that names a package the manifest does not
// declare directly, so a typo or a transitive dependency is explained before
// any package manager runs.
func requireDeclared(manifestPath string, names []string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read package manifest: %w", err)
	}
	var manifest struct {
		Dependencies         map[string]json.RawMessage `json:"dependencies"`
		DevDependencies      map[string]json.RawMessage `json:"devDependencies"`
		OptionalDependencies map[string]json.RawMessage `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse package manifest: %w", err)
	}
	var missing []string
	for _, name := range names {
		_, runtime := manifest.Dependencies[name]
		_, dev := manifest.DevDependencies[name]
		_, optional := manifest.OptionalDependencies[name]
		if !runtime && !dev && !optional {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("%s is not a direct dependency in %s (dependencies, devDependencies or optionalDependencies); a package that is only a transitive dependency goes away when the package that needs it does", strings.Join(missing, ", "), manifestPath)
}
