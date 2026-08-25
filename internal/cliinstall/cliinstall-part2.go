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
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/discovery"
)

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
	if err := m.writeInstallMetadata(item, meta); err != nil {
		return err
	}
	return m.recordInstalledCLI(item)
}

func (m *Manager) recordInstalledCLI(item InstallableCLI) error {
	path := m.InstalledBinaryPath(item)
	if err := RecordInstallEntries(m.Home, InstallEntry{
		Scope: ScopeRuntime, Kind: EntryBinary, Path: path, Prefix: m.InstallDir(),
	}, InstallEntry{
		Scope: ScopeRuntime, Kind: EntryFile, Path: m.InstallMetadataPath(item), Prefix: m.InstallDir(),
	}, InstallEntry{
		Scope: ScopeRuntime, Kind: EntryFile, Path: m.InstalledManifestPath(item), Prefix: m.InstallDir(),
	}); err != nil {
		return fmt.Errorf("record installed CLI %q: %w", item.Name, err)
	}
	return nil
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
	return config.WriteOwnedFile(m.InstallMetadataPath(item), data, 0o644)
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
		default:
			return cliutil.FreshnessSpec{}, fmt.Errorf("unsupported scenario CLI adapter kind %q", item.CLI.Adapter.Kind)
		}
	case KindResource:
		switch item.CLI.Adapter.Kind {
		case "go_module":
			return cliutil.CanonicalResourceGoModuleFreshnessSpec(item.ResourcePath, item.ModulePath, item.BinaryName, customInputs), nil
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

	args := cliutil.GoModuleInstallerArgs(item.ModulePath, item.ManifestPath, item.BinaryName, installDir, spec)

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
