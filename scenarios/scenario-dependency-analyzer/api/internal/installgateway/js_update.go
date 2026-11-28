package installgateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResolveLockUpdate plans moving packages that are already locked on one surface
// to the newest version their declared ranges allow. It exists for the case a
// range already admits a fix (a security release inside ~1.20.3, say) while the
// lockfile still pins an older resolution. It reuses the staged install
// machinery, so the store, admission, rollback and closure preview are the same
// as for an install or a removal. It never edits a manifest range and adds no
// direct dependency, which is why it needs no governance verdict.
func ResolveLockUpdate(repoRoot, scenario, surface, ecosystem string, packages []string) (Resolution, error) {
	if err := validateScenarioName(scenario); err != nil {
		return Resolution{}, err
	}
	if CanonicalEcosystem(ecosystem) != "npm" {
		return Resolution{}, fmt.Errorf("deps update supports npm packages on pnpm and npm surfaces, not %q", ecosystem)
	}
	surface, err := normalizedSurface(surface)
	if err != nil {
		return Resolution{}, err
	}
	if surface == "control-plane" || strings.HasPrefix(surface, "template/") {
		return Resolution{}, fmt.Errorf("deps update does not apply to the %s surface", surface)
	}
	names, err := lockedNames(packages, "update")
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
		return Resolution{}, fmt.Errorf("deps update supports pnpm and npm surfaces; %s uses %s", surfaceRoot, manager)
	}
	if err := requireLocked(surfaceRoot, manager, names); err != nil {
		return Resolution{}, err
	}
	argv := []string{manager, "update"}
	switch manager {
	case "npm":
		argv = append(argv, "--ignore-scripts")
	case "pnpm":
		if isSharedPackageRoot(repoRoot, surfaceRoot) && !fileExists(filepath.Join(surfaceRoot, "pnpm-workspace.yaml")) {
			argv = append(argv, "--ignore-workspace")
		}
	}
	// pnpm limits one `update` to the direct dependencies as soon as any name
	// it is given is direct, leaving the transitive names untouched (reported
	// "unchanged"). So the direct names move first and the transitive ones in a
	// second command of the same staged resolution.
	first, then := names, [][]string(nil)
	if manager == "pnpm" {
		direct, transitive, splitErr := splitDirectNames(filepath.Join(surfaceRoot, "package.json"), names)
		if splitErr != nil {
			return Resolution{}, splitErr
		}
		if len(direct) > 0 && len(transitive) > 0 {
			first = direct
			then = [][]string{append(append([]string(nil), argv...), transitive...)}
		}
	}
	argv = append(argv, first...)
	return Resolution{
		RepositoryRoot:  repoRoot,
		SurfaceRoot:     surfaceRoot,
		PackageManager:  manager,
		ManifestPath:    filepath.Join(surfaceRoot, "package.json"),
		PackageName:     strings.Join(names, ", "),
		UpdatedPackages: names,
		Argv:            argv,
		ThenArgv:        then,
		Profile:         SafeProfileFor(manager, argv),
	}, nil
}

// requireLocked refuses a name the lockfile does not contain, so a typo is
// explained before any package manager runs.
func requireLocked(surfaceRoot, manager string, names []string) error {
	locked, err := lockedPackageNames(surfaceRoot, manager)
	if err != nil {
		return err
	}
	var missing []string
	for _, name := range names {
		if !locked[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("%s is not in the lockfile of %s; deps update moves packages that are already locked (use deps install to add one)", strings.Join(missing, ", "), surfaceRoot)
}

// lockedPackageNames is the set of package names present in the surface lockfile.
func lockedPackageNames(surfaceRoot, manager string) (map[string]bool, error) {
	names := map[string]bool{}
	switch manager {
	case "pnpm":
		data, err := os.ReadFile(filepath.Join(surfaceRoot, "pnpm-lock.yaml"))
		if err != nil {
			return nil, fmt.Errorf("read pnpm-lock.yaml: %w", err)
		}
		packages, err := readPnpmLockPackages(data)
		if err != nil {
			return nil, err
		}
		for key := range packages {
			if name, _, ok := splitPnpmPackageKey(key); ok {
				names[name] = true
			}
		}
	case "npm":
		data, err := os.ReadFile(filepath.Join(surfaceRoot, "package-lock.json"))
		if err != nil {
			return nil, fmt.Errorf("read package-lock.json: %w", err)
		}
		var document struct {
			Packages map[string]npmLockEntry `json:"packages"`
		}
		if err := json.Unmarshal(data, &document); err != nil {
			return nil, fmt.Errorf("decode package-lock.json: %w", err)
		}
		for key, entry := range document.Packages {
			name := entry.Name
			if name == "" {
				if idx := strings.LastIndex(key, "node_modules/"); idx >= 0 {
					name = key[idx+len("node_modules/"):]
				}
			}
			if name != "" {
				names[name] = true
			}
		}
	}
	return names, nil
}

// splitDirectNames separates the names the manifest declares directly from the
// ones only the lockfile holds.
func splitDirectNames(manifestPath string, names []string) (direct, transitive []string, err error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read package manifest: %w", err)
	}
	var manifest struct {
		Dependencies         map[string]json.RawMessage `json:"dependencies"`
		DevDependencies      map[string]json.RawMessage `json:"devDependencies"`
		OptionalDependencies map[string]json.RawMessage `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, nil, fmt.Errorf("parse package manifest: %w", err)
	}
	for _, name := range names {
		_, runtime := manifest.Dependencies[name]
		_, dev := manifest.DevDependencies[name]
		_, optional := manifest.OptionalDependencies[name]
		if runtime || dev || optional {
			direct = append(direct, name)
		} else {
			transitive = append(transitive, name)
		}
	}
	return direct, transitive, nil
}
