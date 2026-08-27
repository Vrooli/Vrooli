package cliinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/cliresolve"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlease"
	"github.com/vrooli/vrooli/internal/artifactledger"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/discovery"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

const (
	managerGoModule = "go_module"
)

const (
	managerParameterA = 4
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
	Root       string
	Home       string
	installDir string // resolved once from the runtime_home authority (bin entry)
	Installer  Installer
	removeFile func(string) error
	// ledger makes every removal this manager performs attributable. It is
	// never nil in a manager built by NewManager; a nil ledger means a caller
	// constructed a Manager literal, and removal refuses rather than deleting
	// something nothing will have recorded.
	ledger *artifactledger.Ledger
}

// ScenarioCLIRemovalReport describes the installed artifacts removed during
// scenario deletion. A locked artifact is reported as skipped so source
// deletion can still complete and the operator has an explicit residue list.
type ScenarioCLIRemovalReport struct {
	Removed int
	Skipped []string
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

// AtomicInstall copies src to dst without ever truncating the live executable.
// The temporary-file, fsync, rename sequence preserves the last runnable
// binary if power is lost during installation. This retains the 2026-05-07
// incident history: a direct install left a zero-filled executable in PATH.
func AtomicInstall(src, dst string) error {
	release, err := buildinfo.AcquireBinaryInstallLock(dst)
	if err != nil {
		return fmt.Errorf("lock binary install %s: %w", dst, err)
	}
	defer release()
	_ = buildinfo.PreserveRootBinaryFallback(dst)
	return config.InstallExecutableAtomic(src, dst)
}

func (s InstallLocationStatus) PathMismatch() bool {
	return s.Resolved && !s.ResolvedCanonical
}

func NewManager(root, home string) (*Manager, error) {
	if strings.TrimSpace(home) == "" {
		resolved, err := config.HomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
		home = resolved
	}
	cleanHome := filepath.Clean(home)
	installDir, err := repocontract.RuntimeHomeEntryPath(cleanHome, repocontract.HomeKeyBin)
	if err != nil {
		return nil, fmt.Errorf("resolve cli install dir: %w", err)
	}
	ledger, err := artifactledger.New(cleanHome)
	if err != nil {
		return nil, fmt.Errorf("resolve removal ledger: %w", err)
	}
	return &Manager{
		Root:       filepath.Clean(root),
		Home:       cleanHome,
		installDir: installDir,
		Installer:  GoInstaller{},
		removeFile: os.Remove,
		ledger:     ledger,
	}, nil
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
	return cliresolve.New(m.Home).Executable(item.BinaryName)
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
	servicePath, err := contract.ScenarioFile(m.Root, name, "service")
	if err != nil {
		return InstallableCLI{}, err
	}
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
	case managerGoModule:
		item.ModulePath = filepath.Join(scenarioRoot, filepath.FromSlash(manifest.CLI.Adapter.ModuleDir))
		if err := requireFile(filepath.Join(item.ModulePath, "go.mod")); err != nil {
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
	case managerGoModule:
		item.ModulePath = filepath.Join(resourceRoot, filepath.FromSlash(manifest.CLI.Adapter.ModuleDir))
		if err := requireFile(filepath.Join(item.ModulePath, "go.mod")); err != nil {
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
	scenarioDir, err := contract.TopLevelDir(m.Root, repocontractmeta.ScenarioDir)
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
	return m.installDir
}

// scenarioDeletionPredicate is the rule this removal path enforces, recorded on
// every receipt it writes. This path runs only as part of deleting a scenario:
// the CLI triple follows its scenario out of the repository rather than being
// reclaimed on its own judgement.
const scenarioDeletionPredicate = "scenario source deletion requested; the installed CLI triple follows its scenario"

// RemoveScenarioCLI removes the canonical installed triple for a scenario.
// It is intentionally idempotent: scenario deletion may be retried after the
// binary or either sidecar has already disappeared. Resource binaries and the
// root control-plane binaries are never accepted by this helper.
func (m *Manager) RemoveScenarioCLI(name string) (int, error) {
	report, err := m.RemoveScenarioCLIReport(name)
	return report.Removed, err
}

// RemoveScenarioCLIReport is the reporting form of RemoveScenarioCLI. It is
// intentionally idempotent and treats permission/locking failures as
// operator-visible skips instead of failing deletion of the scenario source.
func (m *Manager) RemoveScenarioCLIReport(name string) (ScenarioCLIRemovalReport, error) {
	name = ScenarioBinaryName(name)
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) {
		return ScenarioCLIRemovalReport{}, fmt.Errorf("invalid scenario CLI name %q", name)
	}
	if _, root := rootBinaryNames[name]; root || strings.HasPrefix(name, "resource-") {
		return ScenarioCLIRemovalReport{}, fmt.Errorf("refusing to remove non-scenario CLI %q", name)
	}
	removeFile := m.removeFile
	if removeFile == nil {
		removeFile = os.Remove
	}
	if m.ledger == nil {
		// Refusing is the safe direction. A removal nothing records is exactly
		// the fault this ledger exists to end, so a Manager assembled without
		// one does not get to delete.
		return ScenarioCLIRemovalReport{}, fmt.Errorf("remove scenario CLI %q: no removal ledger is configured", name)
	}
	report := ScenarioCLIRemovalReport{}
	for _, binaryName := range []string{name, name + ".exe"} {
		installed := filepath.Join(m.InstallDir(), binaryName)
		for _, artifact := range []struct{ kind, path string }{
			{"binary", installed},
			{"build-metadata", installedBuildMetadataPath(installed)},
			{"manifest", installedManifestPath(installed)},
		} {
			kind, path := artifact.kind, artifact.path
			err := m.ledger.Guard(artifactledger.Removal{
				Path:      path,
				Subject:   installed,
				Kind:      kind,
				Component: "cliinstall.RemoveScenarioCLIReport",
				Predicate: scenarioDeletionPredicate,
			}, func() error { return removeFile(path) })
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				if os.IsPermission(err) || errors.Is(err, fs.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "sharing violation") || strings.Contains(strings.ToLower(err.Error()), "used by another process") {
					report.Skipped = append(report.Skipped, path)
					continue
				}
				return report, fmt.Errorf("remove scenario CLI artifact %s: %w", path, err)
			}
			report.Removed++
		}
		// The ownership record describes an artifact that is now gone. Leaving it
		// would let a later install inherit a stale generation and, worse, a
		// stale recorded absence.
		if err := artifactlease.Remove(filepath.Join(m.InstallDir(), binaryName)); err != nil {
			report.Skipped = append(report.Skipped, artifactlease.Path(filepath.Join(m.InstallDir(), binaryName)))
		}
	}
	return report, nil
}

func (m *Manager) InstalledBinaryPath(item InstallableCLI) string {
	return filepath.Join(m.InstallDir(), item.BinaryName)
}

// rootBinaryNames lists the vrooli wrapper binaries that share the install
// directory with scenario/resource CLIs. They are excluded from
// InstalledScenarioCLINames so callers don't mistake them for scenario CLIs.
var rootBinaryNames = map[string]struct{}{
	"vrooli":               {},
	"vrooli-api":           {},
	"vrooli-buildmeta":     {},
	"vrooli-ports-migrate": {},
}

// InstalledScenarioCLINames returns the names of installed scenario/resource
// CLI binaries in InstallDir(). Sidecar metadata (.build.meta, .manifest.json)
// and the vrooli root binaries are filtered out. A missing install directory
// returns (nil, nil). Result is sorted for deterministic output.
func (m *Manager) InstalledScenarioCLINames() ([]string, error) {
	entries, err := os.ReadDir(m.InstallDir())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".build.meta") || strings.HasSuffix(name, ".manifest.json") {
			continue
		}
		if _, isRoot := rootBinaryNames[name]; isRoot {
			continue
		}
		if runtime.GOOS != "windows" {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&tuning.PermExecuteMask == 0 {
				continue
			}
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
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
			if err := m.recordInstalledCLI(item); err != nil {
				return err
			}
			// Already current: the artifact was not replaced, so this renews
			// the lease rather than claiming a new generation.
			m.noteOwnership(item, m.InstalledBinaryPath(item), false)
			return nil
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := m.install(ctx, item); err != nil {
		return err
	}
	// New bytes were written to this path, so this is a claim.
	m.noteOwnership(item, m.InstalledBinaryPath(item), true)
	return nil
}

func (m *Manager) installedBinaryCurrent(item InstallableCLI) (bool, error) {
	runnable, err := installedBinaryLooksRunnable(m.InstalledBinaryPath(item), item)
	if err != nil {
		return false, err
	}
	if !runnable {
		return false, nil
	}
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

func installedBinaryLooksRunnable(path string, item InstallableCLI) (bool, error) {
	if item.CLI == nil || item.CLI.Adapter.Kind != managerGoModule {
		return true, nil
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	header := make([]byte, managerParameterA)
	n, err := io.ReadFull(file, header)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	if n < len(header) {
		return false, nil
	}
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "netbsd":
		return string(header) == "\x7fELF", nil
	case "darwin":
		return isMachOMagic(header), nil
	case "windows":
		return header[0] == 'M' && header[1] == 'Z', nil
	default:
		return true, nil
	}
}

func isMachOMagic(header []byte) bool {
	if len(header) < managerParameterA {
		return false
	}
	switch string(header[:4]) {
	case "\xfe\xed\xfa\xce", "\xce\xfa\xed\xfe", "\xfe\xed\xfa\xcf", "\xcf\xfa\xed\xfe", "\xca\xfe\xba\xbe", "\xbe\xba\xfe\xca":
		return true
	default:
		return false
	}
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

// installAll ensures each item is installed and current, continuing past
// individual failures so one broken CLI doesn't abort the entire setup.
// Per-item errors are collected and reported as a single combined error
// at the end; the overall setup still fails (the operator needs to know
// something is broken) but every installable gets attempted, and the
// operator sees which ones failed in one pass instead of having to
// fix-and-retry one at a time.
//
// Why ensure (freshness-checked) and not install (unconditional rebuild):
// re-running `vrooli setup` is idempotent by design — repeat invocations
// should be quiet and fast. Each CLI's freshness fingerprint is recorded
// in its .build.meta sidecar; ensure() recomputes the fingerprint from
// current sources and skips the install when it matches. This means the
// 27+ resource/scenario CLIs only rebuild when something they depend on
// actually changed, instead of every setup spamming the terminal with
// "✅ installed CLI to ..." lines.
//
// Common failure cases this isolates:
//   - go.sum drift in one scenario CLI after a cli-core dep change.
//   - A scenario whose Go module fails to compile.
//   - Missing prerequisites for one shell-script-adapter CLI.
func (m *Manager) installAll(ctx context.Context, items []InstallableCLI) error {
	type failure struct {
		kind Kind
		name string
		err  error
	}
	var failures []failure
	for _, item := range items {
		if err := m.ensure(ctx, item); err != nil {
			failures = append(failures, failure{kind: item.Kind, name: item.Name, err: err})
			fmt.Fprintf(os.Stderr, "[WARN]    %s CLI %q install failed: %v\n", item.Kind, item.Name, err)
			continue
		}
	}
	if len(failures) == 0 {
		return nil
	}
	parts := make([]string, 0, len(failures))
	for _, f := range failures {
		parts = append(parts, fmt.Sprintf("%s %q (%v)", f.kind, f.name, f.err))
	}
	return fmt.Errorf("%d CLI install(s) failed: %s", len(failures), strings.Join(parts, "; "))
}
