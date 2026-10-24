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
	"regexp"
	"strings"
	"unicode"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/internal/tools/hyperframes"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"
)

// Resolution is the resolved install plan for a request: where it runs, which
// manager, the manifest it mutates, and the exact argv.
type Resolution struct {
	HostTool              bool
	HostToolRefresh       bool
	HostToolTargetVersion string
	RepositoryRoot        string
	SurfaceRoot           string
	PackageManager        string
	ManifestPath          string
	PackageName           string
	TemplateVersion       string
	ExactVersion          bool
	GeneratedManifestPath string
	PythonLockfiles       []string
	PythonIndexSets       map[string]pythonIndexSet
	PythonBootstrap       *PythonBootstrap
	Argv                  []string
	Profile               InstallProfile
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
	if r.PythonBootstrap != nil {
		return "uv pip compile --only-binary :all: --generate-hashes " + filepath.Base(r.PythonBootstrap.InputPath) + " into " + strings.Join(baseNames(r.PythonLockfiles), ", ")
	}
	if r.TemplateVersion != "" {
		qualifier := ""
		if r.ExactVersion {
			qualifier = " exactly"
		}
		return "declare " + r.PackageName + "@" + r.TemplateVersion + qualifier + " and regenerate the template lockfile in a temporary scenario surface"
	}
	if len(r.PythonLockfiles) > 0 {
		return "uv pip compile --only-binary :all: --generate-hashes " + r.PackageName + " into " + strings.Join(baseNames(r.PythonLockfiles), ", ")
	}
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
	if r.HostToolRefresh {
		return refreshHostToolSource(ctx, r)
	}
	if r.HostTool {
		return provisionHostTool(ctx, r)
	}
	if r.TemplateVersion != "" {
		return installTemplate(ctx, r, ExecInstaller{})
	}
	if len(r.PythonLockfiles) > 0 {
		return updatePythonLocks(ctx, r)
	}

	if r.PackageManager == "pnpm" {
		if err := ensureReleaseAge(r.SurfaceRoot); err != nil {
			return "", err
		}
	}
	if r.PackageManager == "go" && r.RepositoryRoot != "" && r.PackageName != "" {
		topo, err := gomodreconcile.LoadTopology(r.RepositoryRoot)
		if err != nil {
			return "", fmt.Errorf("load repository-local Go modules: %w", err)
		}
		if _, err := gomodreconcile.PrepareLocalInstall(ctx, r.ManifestPath, r.PackageName, topo); err != nil {
			return "", err
		}
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
		repair.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
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
var allowedSurfaces = map[string]struct{}{"ui": {}, "api": {}, "cli": {}, "agent": {}, "playwright-driver": {}, "resource": {}}

// Resolve maps a request to a Resolution. repoRoot is the Vrooli repo root;
// surface is ui/api/cli/agent/playwright-driver, resource, tools/<package>, platforms/<package>, or template/<id>.
// It validates the surface exists and that the ecosystem
// matches the surface's detected package manager, and builds the install argv.
func Resolve(repoRoot, scenario, surface, ecosystem, packageName, version string) (Resolution, error) {
	if surface == "host-tool" {
		return resolveHostTool(repoRoot, scenario, ecosystem, packageName, version)
	}
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
		return Resolution{}, fmt.Errorf("surface directory not found: scenarios/%s/%s or resources/%s/%s", scenario, surface, scenario, surface)
	}

	ecosystem = strings.ToLower(strings.TrimSpace(ecosystem))
	manager, manifest, argv, err := planForEcosystem(surfaceRoot, ecosystem, packageName, version)
	if err != nil {
		return Resolution{}, err
	}
	if manager == "pnpm" && isSharedPackageRoot(repoRoot, surfaceRoot) && !fileExists(filepath.Join(surfaceRoot, "pnpm-workspace.yaml")) {
		argv = append(argv, "--ignore-workspace")
	}
	templateVersion := ""
	if strings.HasPrefix(surface, "template/") {
		if ecosystem != "npm" || version == "" {
			return Resolution{}, fmt.Errorf("template declarations require an explicit npm version")
		}
		templateVersion = version
	}
	generatedManifestPath := ""
	if templateVersion != "" && scenario == "scenario-to-desktop" && surface == "template/vanilla" && (packageName == "electron" || packageName == "electron-mksnapshot") {
		generatedManifestPath = filepath.Join(surfaceRoot, "package.json.template")
	}
	resolution := Resolution{
		TemplateVersion:       templateVersion,
		GeneratedManifestPath: generatedManifestPath,
		PythonLockfiles:       pythonLockfilesFor(surfaceRoot, manager),
		PythonIndexSets:       pythonIndexSetsFor(surfaceRoot, manager),
		RepositoryRoot:        repoRoot,
		SurfaceRoot:           surfaceRoot,
		PackageManager:        manager,
		ManifestPath:          manifest,
		PackageName:           packageName,
		Argv:                  argv,
		Profile:               SafeProfileFor(manager, argv),
	}
	if manager == "uv-lock" && !fileExists(filepath.Join(surfaceRoot, "requirements.lock")) {
		bootstrap, err := resolvePythonBootstrap(surfaceRoot, packageName, version)
		if err != nil {
			return Resolution{}, err
		}
		resolution.PythonBootstrap = bootstrap
		resolution.PythonIndexSets = map[string]pythonIndexSet{"requirements.lock": bootstrap.Indexes}
		resolution.PythonLockfiles = []string{filepath.Join(surfaceRoot, "requirements.lock")}
		resolution.Profile.FrozenLockfile = false // Resolves a new lock; does not reproduce an existing frozen closure.
		resolution.Profile.ScriptsDisabled = true
		resolution.Profile.LifecycleMode = "wheel-only-lock-resolution"
	}
	return resolution, nil
}

// ApplyExactVersion adds the package manager's exact-save option to a resolved
// install. Template installs also project the same version into the generated
// Electron app manifest through installTemplate.
func ApplyExactVersion(r *Resolution, version string) error {
	if r == nil {
		return fmt.Errorf("install resolution is required")
	}
	version = strings.TrimSpace(version)
	if !exactNpmVersion.MatchString(version) {
		return fmt.Errorf("exact version requires a concrete semantic version")
	}
	if r.HostToolRefresh {
		if version != r.HostToolTargetVersion {
			return fmt.Errorf("host-tool source refresh target changed")
		}
		r.ExactVersion = true
		return nil
	}
	if r.HostTool {
		if version != hyperframes.Version {
			return fmt.Errorf("host tool version must match the committed closure")
		}
		r.ExactVersion = true
		return nil
	}
	var flag string
	switch r.PackageManager {
	case "npm":
		flag = "--save-exact"
	case "pnpm":
		flag = "--save-exact"
	case "yarn":
		flag = "--exact"
	default:
		return fmt.Errorf("exact package pinning is not supported for %s installs", r.PackageManager)
	}
	r.Argv = append(r.Argv, flag)
	r.ExactVersion = true
	return nil
}

// ApplyDevelopmentDependency expresses an explicit JS development placement.
// Omitting it preserves Resolve's existing dependency classification. Flags
// remain typed argv and never become a direct package.json edit.
func ApplyDevelopmentDependency(r *Resolution) error {
	if r == nil || r.HostTool || r.HostToolRefresh || len(r.Argv) < 2 {
		return fmt.Errorf("development placement requires a mutating JavaScript surface")
	}
	var flag string
	switch r.PackageManager {
	case "npm":
		flag = "--save-dev"
	case "pnpm":
		flag = "-D"
	case "yarn":
		flag = "--dev"
	default:
		return fmt.Errorf("development placement is unsupported for %s", r.PackageManager)
	}
	for _, arg := range r.Argv {
		if arg == flag {
			return nil
		}
	}
	r.Argv = append(r.Argv, flag)
	r.Profile = SafeProfileFor(r.PackageManager, r.Argv)
	return nil
}

var exactNpmVersion = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

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
	if strings.Contains(joined, "go mod download") || strings.Contains(joined, "cargo fetch --locked") || strings.Contains(joined, "poetry install --sync") || strings.Contains(joined, "pip install --require-hashes") || manager == "uv-lock" {
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
		return Resolution{}, fmt.Errorf("surface directory not found: scenarios/%s/%s or resources/%s/%s", scenario, surface, scenario, surface)
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

// resolveSurfaceRoot supports scenario surfaces, resource surfaces, and
// package targets. Resource surfaces deliberately use the same request shape
// as scenarios so the dependency analyzer remains the single governed install
// path for both kinds of repository-owned code.
// Package installs use the existing tools/<package> surface spelling so the
// CLI remains backward-compatible while governed dependency changes can reach
// shared packages without pretending they are scenarios.
func resolveSurfaceRoot(repoRoot, scenario, surface string) string {
	if strings.HasPrefix(surface, "template/") {
		// A scenario-owned canonical template (for example
		// scenario-to-desktop/templates/vanilla) is an install surface in its
		// own right. Resolve it before the legacy template-manager registry so
		// dependency governance edits the source template that generates the
		// scenario's desktop shell.
		canonical := filepath.Join(repoRoot, "scenarios", scenario, "templates", strings.TrimPrefix(surface, "template/"))
		if info, err := os.Stat(canonical); err == nil && info.IsDir() {
			return canonical
		}
		root := filepath.Join(repoRoot, "templates", "scenarios", strings.TrimPrefix(surface, "template/"), "ui")
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			return root
		}
		return ""
	}

	scenarioRoot := filepath.Join(repoRoot, "scenarios", scenario, surface)
	if surface != "resource" {
		if info, err := os.Stat(scenarioRoot); err == nil && info.IsDir() {
			return scenarioRoot
		}
	}
	if surface == "resource" {
		resourceRoot := filepath.Join(repoRoot, "resources", scenario)
		if info, err := os.Stat(resourceRoot); err == nil && info.IsDir() {
			return resourceRoot
		}
	}
	if info, err := os.Stat(scenarioRoot); err == nil && info.IsDir() {
		return scenarioRoot
	}
	resourceRoot := filepath.Join(repoRoot, "resources", scenario, surface)
	if info, err := os.Stat(resourceRoot); err == nil && info.IsDir() {
		return resourceRoot
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

// pnpm distinguishes parent selectors from comparison ranges with this delimiter.
var pnpmParentSelector = regexp.MustCompile(`[^ |@]>`)

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
	// A package-wide replacement must retire the older version-specific policy.
	// Keep parent-qualified choices, which apply to a separate dependency edge.
	for selector := range overrides {
		if strings.HasPrefix(selector, packageName+"@") && !pnpmParentSelector.MatchString(selector) {
			delete(overrides, selector)
		}
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
	if len(parts) == 2 && (parts[0] == "tools" || parts[0] == "platforms" || parts[0] == "template") && parts[1] != "" && parts[1] != "." && parts[1] != ".." && !strings.ContainsAny(parts[1], `/\\`) {
		return surface, nil
	}
	return "", fmt.Errorf("surface %q is not ui/api/cli/agent/playwright-driver/resource, tools/<package>, platforms/<package>, or template/<id>", surface)
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
		if fileExists(filepath.Join(surfaceRoot, "uv.lock")) {
			// A committed uv.lock is authoritative package-manager evidence.
			// Keep uv projects on uv instead of routing pyproject.toml through
			// Poetry and leaving the declared lockfile stale.
			manager = "uv"
			manifest = filepath.Join(surfaceRoot, "pyproject.toml")
			argv = []string{"uv", "add", "--dev", spec}
		} else if fileExists(filepath.Join(surfaceRoot, "poetry.lock")) || fileExists(filepath.Join(surfaceRoot, "pyproject.toml")) {
			manager = "poetry"
			manifest = filepath.Join(surfaceRoot, "pyproject.toml")
			argv = []string{"poetry", "add", spec}
		} else if fileExists(filepath.Join(surfaceRoot, "requirements.txt")) {
			manager = "pip"
			manifest = filepath.Join(surfaceRoot, "requirements.txt")
			argv = []string{"pip", "install", spec}
		} else if fileExists(filepath.Join(surfaceRoot, "requirements.lock")) || fileExists(filepath.Join(surfaceRoot, "requirements.in")) {
			manager = "uv-lock"
			manifest = filepath.Join(surfaceRoot, "requirements.lock")
			if strings.TrimSpace(version) == "" {
				return "", "", nil, fmt.Errorf("resource Python lock updates require an explicit package version")
			}
			argv = []string{"uv", "pip", "compile", "--generate-hashes", "--only-binary", ":all:", packageName + "==" + strings.TrimSpace(version)}
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
