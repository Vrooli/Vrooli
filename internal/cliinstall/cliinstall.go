package cliinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/discovery"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

type Kind string

const (
	KindScenario Kind = "scenario"
	KindResource Kind = "resource"
)

type InstallableCLI struct {
	Kind         Kind
	Name         string
	BinaryName   string
	ModulePath   string
	ManifestPath string
	ScenarioPath string
	ResourcePath string
	ServicePath  string
	CLI          *scenario.CLIConfig
}

type Installer interface {
	Install(context.Context, InstallableCLI, string) error
}

type Manager struct {
	Root      string
	Home      string
	Installer Installer
}

type InstallMetadata struct {
	Kind        Kind   `json:"kind"`
	Name        string `json:"name"`
	BinaryName  string `json:"binary_name"`
	ModulePath  string `json:"module_path"`
	Fingerprint string `json:"fingerprint"`
	InstalledAt string `json:"installed_at,omitempty"`
}

type InstallLocationStatus struct {
	Command           string `json:"command"`
	CanonicalPath     string `json:"canonical_path"`
	ResolvedPath      string `json:"resolved_path,omitempty"`
	CanonicalExists   bool   `json:"canonical_exists"`
	Resolved          bool   `json:"resolved"`
	ResolvedCanonical bool   `json:"resolved_canonical"`
}

type DiscoveryReport struct {
	Items    []InstallableCLI    `json:"items"`
	Failures []discovery.Failure `json:"failures,omitempty"`
}

func (s InstallLocationStatus) PathMismatch() bool {
	return s.Resolved && !s.ResolvedCanonical
}

func NewManager(root, home string) *Manager {
	return &Manager{
		Root:      filepath.Clean(root),
		Home:      filepath.Clean(home),
		Installer: GoInstaller{},
	}
}

func ScenarioBinaryName(name string) string {
	return strings.TrimSpace(name)
}

func ResourceBinaryName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return "resource-" + name
}

func (m *Manager) InstallScenarioCLI(name string) error {
	item, err := m.DiscoverScenarioCLI(name)
	if err != nil {
		return err
	}
	return m.install(context.Background(), item)
}

func (m *Manager) InstallResourceCLI(name string) error {
	item, err := m.DiscoverResourceCLI(name)
	if err != nil {
		return err
	}
	return m.install(context.Background(), item)
}

func (m *Manager) InstallAllScenarioCLIs() error {
	items, err := m.DiscoverScenarioCLIs()
	if err != nil {
		return err
	}
	return m.installAll(context.Background(), items)
}

func (m *Manager) InstallEnabledResourceCLIs() error {
	items, err := m.DiscoverEnabledResourceCLIs()
	if err != nil {
		return err
	}
	return m.installAll(context.Background(), items)
}

func (m *Manager) EnsureScenarioCLI(name string) error {
	item, err := m.DiscoverScenarioCLI(name)
	if err != nil {
		return err
	}
	return m.ensure(context.Background(), item)
}

func (m *Manager) ResolveScenarioCLIExecutable(name string) (string, error) {
	item, err := m.DiscoverScenarioCLI(name)
	if err != nil {
		return "", err
	}
	if err := m.ensure(context.Background(), item); err != nil {
		return "", err
	}
	return m.InstalledBinaryPath(item), nil
}

func (m *Manager) InspectScenarioCLIInstallLocation(name string, lookPath func(string) (string, error)) (InstallLocationStatus, error) {
	item, err := m.DiscoverScenarioCLI(name)
	if err != nil {
		return InstallLocationStatus{}, err
	}
	return m.inspectInstallLocation(item, lookPath)
}

func (m *Manager) InspectResourceCLIInstallLocation(name string, lookPath func(string) (string, error)) (InstallLocationStatus, error) {
	item, err := m.DiscoverResourceCLI(name)
	if err != nil {
		return InstallLocationStatus{}, err
	}
	return m.inspectInstallLocation(item, lookPath)
}

func (m *Manager) EnsureResourceCLI(name string) error {
	item, err := m.DiscoverResourceCLI(name)
	if err != nil {
		return err
	}
	return m.ensure(context.Background(), item)
}

func (m *Manager) DiscoverScenarioCLI(name string) (InstallableCLI, error) {
	contract, err := repocontract.LoadDefault(m.Root)
	if err != nil {
		return InstallableCLI{}, err
	}
	scenarioRoot, err := contract.ScenarioRoot(m.Root, name)
	if err != nil {
		return InstallableCLI{}, err
	}
	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")
	if err := requireFile(servicePath); err != nil {
		return InstallableCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
	}
	manifest, err := scenario.ReadService(servicePath)
	if err != nil {
		return InstallableCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
	}
	if !manifest.CLIEnabled() {
		return InstallableCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, fs.ErrNotExist)
	}
	item := InstallableCLI{
		Kind:         KindScenario,
		Name:         name,
		BinaryName:   ScenarioBinaryName(manifest.CLI.Command),
		ManifestPath: servicePath,
		ScenarioPath: scenarioRoot,
		ServicePath:  servicePath,
		CLI:          manifest.CLI,
	}
	switch manifest.CLI.Adapter.Kind {
	case "go_module":
		item.ModulePath = filepath.Join(scenarioRoot, filepath.FromSlash(manifest.CLI.Adapter.ModuleDir))
		if err := requireFile(filepath.Join(item.ModulePath, "go.mod")); err != nil {
			return InstallableCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
		}
	case "shell_script":
		if err := requireFile(filepath.Join(scenarioRoot, filepath.FromSlash(manifest.CLI.Adapter.ScriptPath))); err != nil {
			return InstallableCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
		}
		if err := requireFile(filepath.Join(scenarioRoot, filepath.FromSlash(manifest.CLI.Adapter.InstallScript))); err != nil {
			return InstallableCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
		}
	default:
		return InstallableCLI{}, fmt.Errorf("discover scenario CLI %q: unsupported adapter kind %q", name, manifest.CLI.Adapter.Kind)
	}
	return item, nil
}

func (m *Manager) DiscoverResourceCLI(name string) (InstallableCLI, error) {
	contract, err := repocontract.LoadDefault(m.Root)
	if err != nil {
		return InstallableCLI{}, err
	}
	resourceRoot, err := contract.ResourceRoot(m.Root, name)
	if err != nil {
		return InstallableCLI{}, err
	}
	manifestPath, err := contract.ResourceFile(m.Root, name, "manifest")
	if err != nil {
		return InstallableCLI{}, err
	}
	if err := requireFile(manifestPath); err != nil {
		return InstallableCLI{}, fmt.Errorf("discover resource CLI %q: %w", name, err)
	}
	manifest, err := manifestpkg.Load(manifestPath)
	if err != nil {
		return InstallableCLI{}, fmt.Errorf("discover resource CLI %q: %w", name, err)
	}
	if manifest.CLI == nil || !manifest.CLI.Enabled {
		return InstallableCLI{}, fmt.Errorf("discover resource CLI %q: %w", name, fs.ErrNotExist)
	}
	item := InstallableCLI{
		Kind:         KindResource,
		Name:         name,
		BinaryName:   ScenarioBinaryName(manifest.CLI.Command),
		ManifestPath: manifestPath,
		ResourcePath: resourceRoot,
		ServicePath:  manifestPath,
		CLI:          manifest.CLI,
	}
	switch manifest.CLI.Adapter.Kind {
	case "go_module":
		item.ModulePath = filepath.Join(resourceRoot, filepath.FromSlash(manifest.CLI.Adapter.ModuleDir))
		if err := requireFile(filepath.Join(item.ModulePath, "go.mod")); err != nil {
			return InstallableCLI{}, fmt.Errorf("discover resource CLI %q: %w", name, err)
		}
	case "shell_script":
		if err := requireFile(filepath.Join(resourceRoot, filepath.FromSlash(manifest.CLI.Adapter.ScriptPath))); err != nil {
			return InstallableCLI{}, fmt.Errorf("discover resource CLI %q: %w", name, err)
		}
		if err := requireFile(filepath.Join(resourceRoot, filepath.FromSlash(manifest.CLI.Adapter.InstallScript))); err != nil {
			return InstallableCLI{}, fmt.Errorf("discover resource CLI %q: %w", name, err)
		}
	default:
		return InstallableCLI{}, fmt.Errorf("discover resource CLI %q: unsupported adapter kind %q", name, manifest.CLI.Adapter.Kind)
	}
	return item, nil
}

func (m *Manager) DiscoverScenarioCLIs() ([]InstallableCLI, error) {
	report, err := m.DiscoverScenarioCLIReport()
	if err != nil {
		return nil, err
	}
	if len(report.Failures) > 0 {
		failure := report.Failures[0]
		return nil, fmt.Errorf("discover %s CLI %q: %s", failure.Kind, failure.Name, failure.Error)
	}
	return report.Items, nil
}

func (m *Manager) DiscoverScenarioCLIReport() (DiscoveryReport, error) {
	contract, err := repocontract.LoadDefault(m.Root)
	if err != nil {
		return DiscoveryReport{}, err
	}
	scenarioDir, err := contract.TopLevelDir(m.Root, "scenarios")
	if err != nil {
		return DiscoveryReport{}, err
	}
	names, err := childDirNames(scenarioDir)
	if err != nil {
		return DiscoveryReport{}, err
	}
	return m.discoverCLIs(names, KindScenario, m.DiscoverScenarioCLI)
}

func (m *Manager) DiscoverResourceCLIs() ([]InstallableCLI, error) {
	report, err := m.DiscoverResourceCLIReport()
	if err != nil {
		return nil, err
	}
	if len(report.Failures) > 0 {
		failure := report.Failures[0]
		return nil, fmt.Errorf("discover %s CLI %q: %s", failure.Kind, failure.Name, failure.Error)
	}
	return report.Items, nil
}

func (m *Manager) DiscoverResourceCLIReport() (DiscoveryReport, error) {
	contract, err := repocontract.LoadDefault(m.Root)
	if err != nil {
		return DiscoveryReport{}, err
	}
	resourceDir, err := contract.TopLevelDir(m.Root, "resources")
	if err != nil {
		return DiscoveryReport{}, err
	}
	names, err := childDirNames(resourceDir)
	if err != nil {
		return DiscoveryReport{}, err
	}
	return m.discoverCLIs(names, KindResource, m.DiscoverResourceCLI)
}

func (m *Manager) DiscoverEnabledResourceCLIs() ([]InstallableCLI, error) {
	report, err := m.DiscoverEnabledResourceCLIReport()
	if err != nil {
		return nil, err
	}
	if len(report.Failures) > 0 {
		failure := report.Failures[0]
		return nil, fmt.Errorf("discover %s CLI %q: %s", failure.Kind, failure.Name, failure.Error)
	}
	return report.Items, nil
}

func (m *Manager) DiscoverEnabledResourceCLIReport() (DiscoveryReport, error) {
	enabled, err := m.enabledResourceNames()
	if err != nil {
		return DiscoveryReport{}, err
	}
	return m.discoverCLIs(enabled, KindResource, m.DiscoverResourceCLI)
}

func (m *Manager) InstallDir() string {
	if strings.TrimSpace(m.Home) != "" {
		return filepath.Join(m.Home, ".vrooli", "bin")
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".vrooli", "bin")
	}
	return filepath.Join(".", ".vrooli", "bin")
}

func (m *Manager) InstalledBinaryPath(item InstallableCLI) string {
	return filepath.Join(m.InstallDir(), item.BinaryName)
}

func (m *Manager) InstallMetadataPath(item InstallableCLI) string {
	return installedBuildMetadataPath(m.InstalledBinaryPath(item))
}

func (m *Manager) InstalledManifestPath(item InstallableCLI) string {
	return installedManifestPath(m.InstalledBinaryPath(item))
}

func (m *Manager) inspectInstallLocation(item InstallableCLI, lookPath func(string) (string, error)) (InstallLocationStatus, error) {
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	canonicalPath := m.InstalledBinaryPath(item)
	status := InstallLocationStatus{
		Command:       item.BinaryName,
		CanonicalPath: canonicalPath,
	}
	if _, err := os.Stat(canonicalPath); err == nil {
		status.CanonicalExists = true
	} else if err != nil && !os.IsNotExist(err) {
		return InstallLocationStatus{}, err
	}

	resolvedPath, err := lookPath(item.BinaryName)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return status, nil
		}
		return InstallLocationStatus{}, err
	}
	status.Resolved = true
	status.ResolvedPath = normalizeInstallPath(resolvedPath)
	status.ResolvedCanonical = sameInstallPath(status.CanonicalPath, resolvedPath)
	return status, nil
}

func (m *Manager) ensure(ctx context.Context, item InstallableCLI) error {
	if _, err := os.Stat(m.InstalledBinaryPath(item)); err == nil {
		current, fingerprintErr := m.installedBinaryCurrent(item)
		if fingerprintErr != nil {
			return fingerprintErr
		}
		if current {
			return nil
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return m.install(ctx, item)
}

func (m *Manager) installedBinaryCurrent(item InstallableCLI) (bool, error) {
	meta, ok, err := m.readInstallMetadata(item)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	fingerprint, err := m.computeInstallFingerprint(item)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(meta.Fingerprint) == fingerprint, nil
}

func (m *Manager) readInstallMetadata(item InstallableCLI) (InstallMetadata, bool, error) {
	data, err := os.ReadFile(m.InstallMetadataPath(item))
	if err != nil {
		if os.IsNotExist(err) {
			return InstallMetadata{}, false, nil
		}
		return InstallMetadata{}, false, err
	}
	var meta InstallMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return InstallMetadata{}, false, fmt.Errorf("parse install metadata %s: %w", m.InstallMetadataPath(item), err)
	}
	return meta, true, nil
}

func (m *Manager) installAll(ctx context.Context, items []InstallableCLI) error {
	for _, item := range items {
		if err := m.install(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) install(ctx context.Context, item InstallableCLI) error {
	installer := m.Installer
	if installer == nil {
		installer = GoInstaller{}
	}
	if err := installer.Install(ctx, item, m.InstallDir()); err != nil {
		return err
	}
	fingerprint, err := m.computeInstallFingerprint(item)
	if err != nil {
		return err
	}
	meta := InstallMetadata{
		Kind:        item.Kind,
		Name:        item.Name,
		BinaryName:  item.BinaryName,
		ModulePath:  m.installMetadataSource(item),
		Fingerprint: fingerprint,
	}
	return m.writeInstallMetadata(item, meta)
}

func (m *Manager) installMetadataSource(item InstallableCLI) string {
	if strings.TrimSpace(item.ModulePath) != "" {
		return filepath.ToSlash(item.ModulePath)
	}
	if strings.TrimSpace(item.ServicePath) != "" {
		return filepath.ToSlash(item.ServicePath)
	}
	return filepath.ToSlash(item.ScenarioPath)
}

func (m *Manager) writeInstallMetadata(item InstallableCLI, meta InstallMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(m.InstallMetadataPath(item)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.InstallMetadataPath(item), data, 0o644)
}

func (m *Manager) computeInstallFingerprint(item InstallableCLI) (string, error) {
	spec, err := item.FreshnessSpec()
	if err != nil {
		return "", err
	}
	return cliutil.ComputeFreshnessFingerprint(spec)
}

// FreshnessSpec returns the canonical freshness contract for this installable.
// Both the Go-native installer and the baked-in runtime StaleChecker must
// evaluate the same spec to avoid perpetual reinstalls. See
// cliutil.CanonicalScenarioGoModuleFreshnessSpec et al. for the mapping used
// by cli-core's NewStandardScenarioApp / NewResourceApp.
func (item InstallableCLI) FreshnessSpec() (cliutil.FreshnessSpec, error) {
	if item.CLI == nil {
		if item.ModulePath == "" {
			return cliutil.FreshnessSpec{}, fmt.Errorf("cannot compute freshness spec without CLI manifest or module path")
		}
		return cliutil.FreshnessSpec{
			SourceRoot: item.ModulePath,
			SkipFiles:  []string{item.BinaryName},
		}, nil
	}
	var customInputs []string
	if item.CLI.Freshness != nil {
		customInputs = item.CLI.Freshness.Inputs
	}
	switch item.Kind {
	case KindScenario:
		switch item.CLI.Adapter.Kind {
		case "go_module":
			return cliutil.CanonicalScenarioGoModuleFreshnessSpec(item.ScenarioPath, item.ModulePath, item.BinaryName, customInputs), nil
		case "shell_script":
			manifestRel, err := filepath.Rel(item.ScenarioPath, item.ServicePath)
			if err != nil {
				return cliutil.FreshnessSpec{}, err
			}
			return cliutil.CanonicalShellScriptFreshnessSpec(item.ScenarioPath, item.CLI.Adapter.ScriptPath, item.CLI.Adapter.InstallScript, manifestRel, item.BinaryName, customInputs), nil
		default:
			return cliutil.FreshnessSpec{}, fmt.Errorf("unsupported scenario CLI adapter kind %q", item.CLI.Adapter.Kind)
		}
	case KindResource:
		switch item.CLI.Adapter.Kind {
		case "go_module":
			return cliutil.CanonicalResourceGoModuleFreshnessSpec(item.ResourcePath, item.ModulePath, item.BinaryName, customInputs), nil
		case "shell_script":
			manifestRel, err := filepath.Rel(item.ResourcePath, item.ServicePath)
			if err != nil {
				return cliutil.FreshnessSpec{}, err
			}
			return cliutil.CanonicalShellScriptFreshnessSpec(item.ResourcePath, item.CLI.Adapter.ScriptPath, item.CLI.Adapter.InstallScript, manifestRel, item.BinaryName, customInputs), nil
		default:
			return cliutil.FreshnessSpec{}, fmt.Errorf("unsupported resource CLI adapter kind %q", item.CLI.Adapter.Kind)
		}
	default:
		return cliutil.FreshnessSpec{}, fmt.Errorf("unsupported installable CLI kind %q", item.Kind)
	}
}

func (m *Manager) enabledResourceNames() ([]string, error) {
	contract, err := repocontract.LoadDefault(m.Root)
	if err != nil {
		return nil, err
	}
	configDir, err := contract.TopLevelDir(m.Root, "project_config")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(configDir, "service.json"))
	if err != nil {
		return nil, err
	}

	var payload struct {
		Dependencies struct {
			Resources map[string]struct {
				Enabled bool `json:"enabled"`
			} `json:"resources"`
		} `json:"dependencies"`
	}
	if err := jsonUnmarshal(data, &payload); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(payload.Dependencies.Resources))
	for name, entry := range payload.Dependencies.Resources {
		if entry.Enabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

var jsonUnmarshal = func(data []byte, value any) error {
	return json.Unmarshal(data, value)
}

type GoInstaller struct{}

func (GoInstaller) Install(ctx context.Context, item InstallableCLI, installDir string) error {
	switch item.Kind {
	case KindScenario:
		if item.CLI == nil {
			return errors.New("scenario CLI manifest is required")
		}
		switch item.CLI.Adapter.Kind {
		case "go_module":
			return runGoModuleInstaller(ctx, item, installDir)
		case "shell_script":
			installScript := filepath.Join(item.ScenarioPath, filepath.FromSlash(item.CLI.Adapter.InstallScript))
			cmd := exec.CommandContext(ctx, "bash", installScript)
			cmd.Dir = item.ScenarioPath
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		default:
			return fmt.Errorf("unsupported scenario CLI adapter kind %q", item.CLI.Adapter.Kind)
		}
	case KindResource:
		if item.CLI == nil {
			return errors.New("resource CLI manifest is required")
		}
		switch item.CLI.Adapter.Kind {
		case "go_module":
			return runGoModuleInstaller(ctx, item, installDir)
		case "shell_script":
			installScript := filepath.Join(item.ResourcePath, filepath.FromSlash(item.CLI.Adapter.InstallScript))
			cmd := exec.CommandContext(ctx, "bash", installScript)
			cmd.Dir = item.ResourcePath
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		default:
			return fmt.Errorf("unsupported resource CLI adapter kind %q", item.CLI.Adapter.Kind)
		}
	default:
		return fmt.Errorf("unsupported installable CLI kind %q", item.Kind)
	}
}

func runGoModuleInstaller(ctx context.Context, item InstallableCLI, installDir string) error {
	repoRoot, ok := findInstallerRepoRoot(item.ModulePath)
	if !ok {
		return fmt.Errorf("locate repo root for %s CLI %q", item.Kind, item.Name)
	}
	installerDir := filepath.Join(repoRoot, "packages", "cli-core")

	spec, err := item.FreshnessSpec()
	if err != nil {
		return err
	}

	args := []string{
		"run", "./cmd/cli-installer",
		"--module", item.ModulePath,
		"--manifest", item.ManifestPath,
		"--name", item.BinaryName,
		"--install-dir", installDir,
	}
	if strings.TrimSpace(spec.ContextRoot) != "" && filepath.Clean(spec.ContextRoot) != filepath.Clean(item.ModulePath) {
		args = append(args, "--context-root", spec.ContextRoot)
	}
	for _, input := range spec.Inputs {
		args = append(args, "--freshness-input", input)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = installerDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func findInstallerRepoRoot(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		candidate := filepath.Join(dir, "packages", "cli-core")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func childDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func normalizeInstallPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil && strings.TrimSpace(resolved) != "" {
		path = resolved
	}
	if abs, err := filepath.Abs(path); err == nil && strings.TrimSpace(abs) != "" {
		path = abs
	}
	return filepath.Clean(path)
}

func sameInstallPath(a, b string) bool {
	a = normalizeInstallPath(a)
	b = normalizeInstallPath(b)
	if a == "" || b == "" {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}

func installedManifestPath(binaryPath string) string {
	return binaryPath + ".manifest.json"
}

func installedBuildMetadataPath(binaryPath string) string {
	return binaryPath + ".build.meta"
}

func isSkippableDiscoveryError(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err)
}

func (m *Manager) discoverCLIs(names []string, kind Kind, discoverOne func(string) (InstallableCLI, error)) (DiscoveryReport, error) {
	report := DiscoveryReport{
		Items:    make([]InstallableCLI, 0, len(names)),
		Failures: make([]discovery.Failure, 0),
	}
	for _, name := range names {
		item, err := discoverOne(name)
		if err == nil {
			report.Items = append(report.Items, item)
			continue
		}
		if isSkippableDiscoveryError(err) {
			continue
		}
		report.Failures = append(report.Failures, discovery.Failure{
			Kind:  string(kind),
			Name:  name,
			Stage: "discover_cli",
			Error: err.Error(),
		})
	}
	sort.Slice(report.Items, func(i, j int) bool { return report.Items[i].Name < report.Items[j].Name })
	return report, nil
}
