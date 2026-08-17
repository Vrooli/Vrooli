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
	"unicode"
)

// Resolution is the resolved install plan for a request: where it runs, which
// manager, the manifest it mutates, and the exact argv.
type Resolution struct {
	SurfaceRoot    string
	PackageManager string
	ManifestPath   string
	Argv           []string
	Profile        InstallProfile
}

// InstallProfile records the security properties of a governed mutation. It
// is data, not a shell wrapper, so the same policy is inspectable on Linux,
// macOS, and Windows.
type InstallProfile struct {
	FrozenLockfile  bool   `json:"frozen_lockfile"`
	ScriptsDisabled bool   `json:"scripts_disabled"`
	LifecycleMode   string `json:"lifecycle_mode"`
	Governance      string `json:"governance"`
}

type ProtectedBuildException struct {
	Owner      string `json:"owner"`
	Reason     string `json:"reason"`
	Command    string `json:"command"`
	PolicyMode string `json:"policy_mode"`
}

func ValidateProtectedBuildException(exception ProtectedBuildException) error {
	if strings.TrimSpace(exception.Owner) == "" || strings.TrimSpace(exception.Reason) == "" || strings.TrimSpace(exception.Command) == "" {
		return fmt.Errorf("protected build exception requires owner, reason, and command")
	}
	if exception.PolicyMode != "guided" && exception.PolicyMode != "guarded" && exception.PolicyMode != "enforcing" {
		return fmt.Errorf("protected build exception policy_mode must be guided, guarded, or enforcing")
	}
	return nil
}

// Command is display-only text for the existing response contract. Execution
// always uses Argv with exec.CommandContext; callers must never feed this
// string to a shell.
func (r Resolution) Command() string {
	parts := make([]string, 0, len(r.Argv))
	for _, arg := range r.Argv {
		parts = append(parts, displayArg(arg))
	}
	return strings.Join(parts, " ")
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
	if err := validateResolution(r); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, r.Argv[0], r.Argv[1:]...)
	cmd.Dir = r.SurfaceRoot
	out, err := cmd.CombinedOutput()
	if err != nil && r.PackageManager == "pnpm" && strings.Contains(string(out), "ERR_PNPM_UNEXPECTED_STORE") {
		// A governed install must be able to repair a package workspace whose
		// node_modules was linked by a different pnpm store configuration. The
		// frozen, script-disabled relink is intentionally read-only with respect
		// to dependency versions; it only restores the workspace invariant before
		// retrying the requested governed mutation.
		repair := exec.CommandContext(ctx, "pnpm", "install", "--frozen-lockfile", "--ignore-scripts", "--ignore-workspace")
		repair.Dir = r.SurfaceRoot
		repairOut, repairErr := repair.CombinedOutput()
		out = append(out, repairOut...)
		if repairErr != nil {
			return string(out), repairErr
		}
		retry := exec.CommandContext(ctx, r.Argv[0], r.Argv[1:]...)
		retry.Dir = r.SurfaceRoot
		retryOut, retryErr := retry.CombinedOutput()
		out = append(out, retryOut...)
		return string(out), retryErr
	}
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
	if err := validateScenarioName(scenario); err != nil {
		return Resolution{}, err
	}
	if err := validatePackageSpec(packageName, version); err != nil {
		return Resolution{}, err
	}
	surface, err := normalizedSurface(surface)
	if err != nil {
		return Resolution{}, err
	}
	if strings.TrimSpace(packageName) == "" {
		return Resolution{}, fmt.Errorf("package name is required")
	}
	surfaceRoot := resolveSurfaceRoot(repoRoot, scenario, surface)
	if surfaceRoot == "" {
		return Resolution{}, fmt.Errorf("surface directory not found: scenarios/%s/%s", scenario, surface)
	}

	ecosystem = strings.ToLower(strings.TrimSpace(ecosystem))
	manager, manifest, argv, err := planForEcosystem(surfaceRoot, ecosystem, packageName, version)
	if err != nil {
		return Resolution{}, err
	}
	if manager == "pnpm" && isSharedPackageRoot(repoRoot, surfaceRoot) && !fileExists(filepath.Join(surfaceRoot, "pnpm-workspace.yaml")) {
		argv = append(argv, "--ignore-workspace")
	}
	return Resolution{
		SurfaceRoot:    surfaceRoot,
		PackageManager: manager,
		ManifestPath:   manifest,
		Argv:           argv,
		Profile:        SafeProfileFor(manager, argv),
	}, nil
}

func SafeProfileFor(manager string, argv []string) InstallProfile {
	profile := InstallProfile{Governance: "scenario-dependency-analyzer", LifecycleMode: "native-governed"}
	for _, arg := range argv {
		if arg == "--ignore-scripts" || arg == "--ignore-script" {
			profile.ScriptsDisabled = true
		}
		if arg == "--frozen-lockfile" || arg == "--frozen" || arg == "--immutable" || arg == "--locked" || arg == "ci" {
			profile.FrozenLockfile = true
		}
	}
	joined := strings.Join(argv, " ")
	if strings.Contains(joined, "go mod download") || strings.Contains(joined, "cargo fetch --locked") || strings.Contains(joined, "poetry install --sync") || strings.Contains(joined, "pip install --require-hashes") {
		profile.FrozenLockfile = true
	}
	if manager == "pnpm" || manager == "npm" || manager == "yarn" || manager == "bun" {
		profile.LifecycleMode = "scripts-disabled-by-default"
	}
	return profile
}

// FrozenReproductionArgs returns the read-only, lockfile-reproducing command
// for a package manager. Mutating installs and reproduction are intentionally
// separate operations: additions go through Resolve, while CI/validation uses
// this function so a lockfile cannot be silently rewritten and lifecycle
// scripts stay disabled.
func FrozenReproductionArgs(manager string) ([]string, error) {
	manager = strings.ToLower(strings.TrimSpace(manager))
	var argv []string
	switch manager {
	case "npm":
		argv = []string{"npm", "ci", "--ignore-scripts"}
	case "pnpm":
		argv = []string{"pnpm", "install", "--frozen-lockfile", "--ignore-scripts"}
	case "yarn":
		argv = []string{"yarn", "install", "--immutable", "--ignore-scripts"}
	case "bun":
		argv = []string{"bun", "install", "--frozen-lockfile", "--ignore-scripts"}
	case "go":
		argv = []string{"go", "mod", "download"}
	case "pip":
		argv = []string{"pip", "install", "--require-hashes", "-r", "requirements.txt", "--no-build-isolation"}
	case "poetry":
		argv = []string{"poetry", "install", "--sync", "--no-root"}
	case "cargo":
		argv = []string{"cargo", "fetch", "--locked"}
	default:
		return nil, fmt.Errorf("unsupported package manager %q", manager)
	}
	return argv, nil
}

// ResolveNpmOverride builds a lockfile-refresh plan for a governed pnpm
// resolver override. The caller persists the override only after governance
// accepts it, then this command regenerates the lock without hand-editing it.
func ResolveNpmOverride(repoRoot, scenario, surface string) (Resolution, error) {
	if err := validateScenarioName(scenario); err != nil {
		return Resolution{}, err
	}
	surface, err := normalizedSurface(surface)
	if err != nil {
		return Resolution{}, err
	}
	surfaceRoot := resolveSurfaceRoot(repoRoot, scenario, surface)
	if surfaceRoot == "" {
		return Resolution{}, fmt.Errorf("surface directory not found: scenarios/%s/%s", scenario, surface)
	}
	if manager := jsManager(surfaceRoot); manager != "pnpm" {
		return Resolution{}, fmt.Errorf("npm overrides require a pnpm surface, got %s", manager)
	}
	argv := []string{"pnpm", "install", "--ignore-scripts"}
	if fileExists(filepath.Join(surfaceRoot, "pnpm-workspace.yaml")) {
		argv = append(argv, "--workspace-root")
	}
	return Resolution{SurfaceRoot: surfaceRoot, PackageManager: "pnpm", ManifestPath: filepath.Join(surfaceRoot, "package.json"), Argv: argv, Profile: SafeProfileFor("pnpm", argv)}, nil
}

// resolveSurfaceRoot supports both scenario surfaces and package targets.
// Package installs use the existing tools/<package> surface spelling so the
// CLI remains backward-compatible while governed dependency changes can reach
// shared packages without pretending they are scenarios.
func resolveSurfaceRoot(repoRoot, scenario, surface string) string {
	scenarioRoot := filepath.Join(repoRoot, "scenarios", scenario, surface)
	if info, err := os.Stat(scenarioRoot); err == nil && info.IsDir() {
		return scenarioRoot
	}
	if strings.HasPrefix(surface, "tools/") {
		packageName := strings.TrimPrefix(surface, "tools/")
		packageRoot := filepath.Join(repoRoot, "packages", packageName)
		if info, err := os.Stat(packageRoot); err == nil && info.IsDir() {
			return packageRoot
		}
	}
	return ""
}

func isSharedPackageRoot(repoRoot, surfaceRoot string) bool {
	packagesRoot := filepath.Join(repoRoot, "packages")
	rel, err := filepath.Rel(packagesRoot, surfaceRoot)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// SetNpmOverride records a pnpm override in package.json. It intentionally
// owns only manifest mutation; ResolveNpmOverride regenerates the lockfile.
func SetNpmOverride(manifestPath, packageName, version string) error {
	if err := validatePackageSpec(packageName, version); err != nil {
		return err
	}
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
	if len(parts) == 2 && (parts[0] == "tools" || parts[0] == "platforms") && parts[1] != "" && parts[1] != "." && parts[1] != ".." && !strings.ContainsAny(parts[1], `/\\`) {
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
		if manager == "" {
			return "", "", nil, fmt.Errorf("javascript package manager evidence is missing; provide exactly one supported lockfile")
		}
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
			argv = []string{"npm", "install", "--ignore-scripts"}
			if devDependency {
				argv = append(argv, "--save-dev")
			}
			argv = append(argv, spec)
		case "yarn":
			argv = []string{"yarn", "add", "--ignore-scripts"}
			if devDependency {
				argv = append(argv, "--dev")
			}
			argv = append(argv, spec)
		default:
			manager = "pnpm"
			argv = []string{"pnpm", "add", "--ignore-scripts"}
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
		if !fileExists(filepath.Join(surfaceRoot, "go.mod")) {
			return "", "", nil, fmt.Errorf("go module evidence is missing: go.mod")
		}
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
		} else if fileExists(filepath.Join(surfaceRoot, "requirements.txt")) {
			manager = "pip"
			manifest = filepath.Join(surfaceRoot, "requirements.txt")
			argv = []string{"pip", "install", spec}
		} else {
			return "", "", nil, fmt.Errorf("python package-manager evidence is missing: requirements.txt, pyproject.toml, or poetry.lock")
		}
	case "rust", "cargo":
		if !fileExists(filepath.Join(surfaceRoot, "Cargo.toml")) {
			return "", "", nil, fmt.Errorf("rust package-manager evidence is missing: Cargo.toml")
		}
		manager = "cargo"
		manifest = filepath.Join(surfaceRoot, "Cargo.toml")
		spec := packageName
		if v := strings.TrimSpace(version); v != "" {
			spec += "@" + v
		}
		argv = []string{"cargo", "add", spec}
	default:
		return "", "", nil, fmt.Errorf("unsupported ecosystem %q (want npm, go, pip, or cargo)", ecosystem)
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

// jsManager picks the JS package manager from lockfile evidence. An absent or
// ambiguous lockfile is unknown; callers must not infer pnpm from convention.
func jsManager(root string) string {
	switch {
	case fileExists(filepath.Join(root, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(root, "package-lock.json")):
		return "npm"
	case fileExists(filepath.Join(root, "yarn.lock")):
		return "yarn"
	default:
		return ""
	}
}

func validateScenarioName(scenario string) error {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" || scenario == "." || scenario == ".." || strings.ContainsAny(scenario, `/\\`) {
		return fmt.Errorf("scenario must be a single safe scenario name")
	}
	for _, r := range scenario {
		if unicode.IsControl(r) {
			return fmt.Errorf("scenario contains a control character")
		}
	}
	return nil
}

func validatePackageSpec(packageName, version string) error {
	packageName = strings.TrimSpace(packageName)
	version = strings.TrimSpace(version)
	if packageName == "" {
		return fmt.Errorf("package name is required")
	}
	if strings.HasPrefix(packageName, "-") || strings.ContainsAny(packageName, "\x00\r\n;|&`$()<>\t ") {
		return fmt.Errorf("package name must be one package spec and cannot contain shell syntax or flags")
	}
	if strings.Contains(packageName, "../") || strings.Contains(packageName, `/..`) {
		return fmt.Errorf("package name cannot contain path traversal")
	}
	if strings.HasPrefix(version, "-") || strings.ContainsAny(version, "\x00\r\n;|&`$()<>\t ") {
		return fmt.Errorf("version must be one package spec and cannot contain shell syntax or flags")
	}
	return nil
}

func validateResolution(resolution Resolution) error {
	if len(resolution.Argv) == 0 {
		return fmt.Errorf("no install command resolved")
	}
	if strings.TrimSpace(resolution.SurfaceRoot) == "" {
		return fmt.Errorf("install surface root is required")
	}
	info, err := os.Stat(resolution.SurfaceRoot)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("install surface root is not a directory")
	}
	for i, arg := range resolution.Argv {
		if strings.TrimSpace(arg) == "" || strings.ContainsAny(arg, "\x00\r\n") {
			return fmt.Errorf("argv[%d] is invalid", i)
		}
		if i > 0 && arg == "--" {
			return fmt.Errorf("argv contains an unexpected terminator")
		}
	}
	return nil
}

func displayArg(arg string) string {
	if arg != "" {
		safe := true
		for _, r := range arg {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_./:@+^~=,-", r)) {
				safe = false
				break
			}
		}
		if safe {
			return arg
		}
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
