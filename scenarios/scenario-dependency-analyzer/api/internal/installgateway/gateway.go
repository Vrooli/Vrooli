// Package installgateway resolves and executes governed dependency installs: it
// maps a (scenario, surface, ecosystem, package) request to the right package
// manager, manifest, and install argv, and runs it behind a PackageInstaller
// seam so the execution is unit-testable without invoking a real pnpm/go/pip.
// Governance enforcement and the security re-scan live in the caller
// (dependencygovernance); this package owns only resolution + execution.
package installgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Resolution is the resolved install plan for a request: where it runs, which
// manager, the manifest it mutates, and the exact argv.
type Resolution struct {
	SurfaceRoot    string
	PackageManager string
	ManifestPath   string
	Argv           []string
}

// Command renders the argv as a copy-pasteable shell line.
func (r Resolution) Command() string {
	return strings.Join(r.Argv, " ")
}

// PackageInstaller executes a resolved install plan. The real implementation
// runs the command in the surface root; tests inject a fake that records the
// plan and returns canned output.
type PackageInstaller interface {
	Install(ctx context.Context, r Resolution) (output string, err error)
}

// ExecInstaller runs the install command in the surface root directory.
type ExecInstaller struct{}

func (ExecInstaller) Install(ctx context.Context, r Resolution) (string, error) {
	if len(r.Argv) == 0 {
		return "", fmt.Errorf("no install command resolved")
	}
	cmd := exec.CommandContext(ctx, r.Argv[0], r.Argv[1:]...)
	cmd.Dir = r.SurfaceRoot
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// allowedSurfaces is the closed set of installable top-level scenario surfaces.
// playwright-driver is a lifecycle-managed production sidecar, while tools and
// platforms packages are addressed as tools/<package> and platforms/<package>
// to keep auxiliary and distribution installs scoped.
var allowedSurfaces = map[string]struct{}{"ui": {}, "api": {}, "cli": {}, "playwright-driver": {}}

// Resolve maps a request to a Resolution. repoRoot is the Vrooli repo root;
// surface is ui/api/cli/playwright-driver, tools/<package>, or platforms/<package>.
// It validates the surface exists and that the ecosystem
// matches the surface's detected package manager, and builds the install argv.
func Resolve(repoRoot, scenario, surface, ecosystem, packageName, version string) (Resolution, error) {
	surface, err := normalizedSurface(surface)
	if err != nil {
		return Resolution{}, err
	}
	if strings.TrimSpace(packageName) == "" {
		return Resolution{}, fmt.Errorf("package name is required")
	}
	surfaceRoot := filepath.Join(repoRoot, "scenarios", scenario, surface)
	if info, err := os.Stat(surfaceRoot); err != nil || !info.IsDir() {
		return Resolution{}, fmt.Errorf("surface directory not found: scenarios/%s/%s", scenario, surface)
	}

	ecosystem = strings.ToLower(strings.TrimSpace(ecosystem))
	manager, manifest, argv, err := planForEcosystem(surfaceRoot, ecosystem, packageName, version)
	if err != nil {
		return Resolution{}, err
	}
	return Resolution{
		SurfaceRoot:    surfaceRoot,
		PackageManager: manager,
		ManifestPath:   manifest,
		Argv:           argv,
	}, nil
}

// ResolveNpmOverride builds a lockfile-refresh plan for a governed pnpm
// resolver override. The caller persists the override only after governance
// accepts it, then this command regenerates the lock without hand-editing it.
func ResolveNpmOverride(repoRoot, scenario, surface string) (Resolution, error) {
	surface, err := normalizedSurface(surface)
	if err != nil {
		return Resolution{}, err
	}
	surfaceRoot := filepath.Join(repoRoot, "scenarios", scenario, surface)
	if info, err := os.Stat(surfaceRoot); err != nil || !info.IsDir() {
		return Resolution{}, fmt.Errorf("surface directory not found: scenarios/%s/%s", scenario, surface)
	}
	if manager := jsManager(surfaceRoot); manager != "pnpm" {
		return Resolution{}, fmt.Errorf("npm overrides require a pnpm surface, got %s", manager)
	}
	argv := []string{"pnpm", "install"}
	if fileExists(filepath.Join(surfaceRoot, "pnpm-workspace.yaml")) {
		argv = append(argv, "--workspace-root")
	}
	return Resolution{SurfaceRoot: surfaceRoot, PackageManager: "pnpm", ManifestPath: filepath.Join(surfaceRoot, "package.json"), Argv: argv}, nil
}

// SetNpmOverride records a pnpm override in package.json. It intentionally
// owns only manifest mutation; ResolveNpmOverride regenerates the lockfile.
func SetNpmOverride(manifestPath, packageName, version string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read package manifest: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse package manifest: %w", err)
	}
	pnpm, _ := manifest["pnpm"].(map[string]any)
	if pnpm == nil {
		pnpm = map[string]any{}
		manifest["pnpm"] = pnpm
	}
	overrides, _ := pnpm["overrides"].(map[string]any)
	if overrides == nil {
		overrides = map[string]any{}
		pnpm["overrides"] = overrides
	}
	overrides[packageName] = version
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode package manifest: %w", err)
	}
	return os.WriteFile(manifestPath, append(encoded, '\n'), 0o644)
}

func normalizedSurface(surface string) (string, error) {
	surface = strings.ToLower(strings.TrimSpace(surface))
	if _, ok := allowedSurfaces[surface]; ok {
		return surface, nil
	}
	parts := strings.Split(surface, "/")
	if len(parts) == 2 && (parts[0] == "tools" || parts[0] == "platforms") && parts[1] != "" && parts[1] != "." && parts[1] != ".." && !strings.Contains(parts[1], `\\`) {
		return surface, nil
	}
	return "", fmt.Errorf("surface %q is not ui/api/cli/playwright-driver, tools/<package>, or platforms/<package>", surface)
}

// planForEcosystem builds the package manager, manifest path, and install argv
// for one ecosystem, detecting the JS manager from the surface lockfiles.
func planForEcosystem(surfaceRoot, ecosystem, packageName, version string) (manager, manifest string, argv []string, err error) {
	switch ecosystem {
	case "npm", "node", "js", "ts", "typescript":
		manager = jsManager(surfaceRoot)
		manifest = filepath.Join(surfaceRoot, "package.json")
		devDependency, err := existingJSDevDependency(manifest, packageName)
		if err != nil {
			return "", "", nil, err
		}
		spec := packageName
		if v := strings.TrimSpace(version); v != "" {
			spec = packageName + "@" + v
		}
		switch manager {
		case "npm":
			argv = []string{"npm", "install"}
			if devDependency {
				argv = append(argv, "--save-dev")
			}
			argv = append(argv, spec)
		case "yarn":
			argv = []string{"yarn", "add"}
			if devDependency {
				argv = append(argv, "--dev")
			}
			argv = append(argv, spec)
		default:
			manager = "pnpm"
			argv = []string{"pnpm", "add"}
			if devDependency {
				argv = append(argv, "-D")
			}
			// Many scenario surfaces are intentionally standalone pnpm
			// workspaces. ExecInstaller runs from the surface root, so this flag
			// confirms the surface workspace root rather than the repo root.
			if fileExists(filepath.Join(surfaceRoot, "pnpm-workspace.yaml")) {
				argv = append(argv, "--workspace-root")
			}
			argv = append(argv, spec)
		}
	case "go", "golang":
		manager = "go"
		manifest = filepath.Join(surfaceRoot, "go.mod")
		spec := packageName
		if v := strings.TrimSpace(version); v != "" {
			spec = packageName + "@" + v
		}
		argv = []string{"go", "get", spec}
	case "pip", "python", "pypi":
		spec := packageName
		if v := strings.TrimSpace(version); v != "" {
			spec = packageName + "==" + v
		}
		if fileExists(filepath.Join(surfaceRoot, "poetry.lock")) || fileExists(filepath.Join(surfaceRoot, "pyproject.toml")) {
			manager = "poetry"
			manifest = filepath.Join(surfaceRoot, "pyproject.toml")
			argv = []string{"poetry", "add", spec}
		} else {
			manager = "pip"
			manifest = filepath.Join(surfaceRoot, "requirements.txt")
			argv = []string{"pip", "install", spec}
		}
	default:
		return "", "", nil, fmt.Errorf("unsupported ecosystem %q (want npm, go, or pip)", ecosystem)
	}
	return manager, manifest, argv, nil
}

// existingJSDevDependency preserves the manifest's dependency classification
// when upgrading an already declared package. Tooling such as Vite and Vitest
// must not silently become runtime dependencies merely because their upgrade
// passed through the governed installer.
func existingJSDevDependency(manifestPath, packageName string) (bool, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read package manifest: %w", err)
	}
	var manifest struct {
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return false, fmt.Errorf("parse package manifest: %w", err)
	}
	_, isDevDependency := manifest.DevDependencies[packageName]
	return isDevDependency, nil
}

// jsManager picks the JS package manager from the surface lockfiles, defaulting
// to pnpm (the Vrooli convention).
func jsManager(root string) string {
	switch {
	case fileExists(filepath.Join(root, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(root, "package-lock.json")):
		return "npm"
	case fileExists(filepath.Join(root, "yarn.lock")):
		return "yarn"
	default:
		return "pnpm"
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
