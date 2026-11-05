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
	"slices"
	"sort"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/artifactlease"
	"github.com/vrooli/vrooli/internal/artifactledger"

	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/scenario"
)

const (
	installationGoModule = "go_module"
	cliBuildGroupProbe   = 25 * time.Millisecond
)

func (m *Manager) install(ctx context.Context, item InstallableCLI) error {
	installer := m.Installer
	if installer == nil {
		installer = GoInstaller{}
	}
	if err := installer.Install(ctx, item, m.InstallDir()); err != nil {
		return err
	}
	// The canonical installer writes the digest and freshness manifest.  Reuse
	// that digest instead of immediately hashing the same source tree again.
	// Test/custom installers from older callers may not write metadata, so keep
	// the full-hash fallback for compatibility.
	installedMeta, metaOK, metaErr := m.readInstallMetadata(item)
	if metaErr != nil {
		return metaErr
	}
	manifestPath := cliutil.FreshnessManifestPath(m.InstalledBinaryPath(item))
	freshnessManifest, manifestOK, manifestErr := cliutil.ReadFreshnessManifest(manifestPath)
	if manifestErr != nil {
		manifestOK = false
	}
	if !manifestOK || freshnessManifest.Digest == "" {
		// Older/custom installers do not emit the manifest.  Bootstrap it once;
		// future checks use the stat-cache path and avoid repeated full hashing.
		freshnessManifest, manifestErr = m.computeInstallManifest(item)
		if manifestErr != nil {
			return manifestErr
		}
		manifestOK = false
	}
	// Prefer the manifest digest because it was computed from the same source
	// snapshot as the newly installed binary.  The metadata fallback keeps
	// compatibility with a legacy installer that emitted only .build.meta.
	fingerprint := freshnessManifest.Digest
	if fingerprint == "" && metaOK {
		fingerprint = strings.TrimSpace(installedMeta.Fingerprint)
	}
	if fingerprint == "" {
		computed, fingerprintErr := m.computeInstallFingerprint(item)
		if fingerprintErr != nil {
			return fingerprintErr
		}
		fingerprint = computed
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
	if !manifestOK {
		if err := cliutil.WriteFreshnessManifest(manifestPath, freshnessManifest); err != nil {
			return err
		}
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

// leaseTTL is how long a freshly installed CLI is protected outright.
//
// It matches the reclamation grace window on purpose. An artifact whose
// scenario is genuinely gone stops being reinstalled, so its lease lapses a day
// after the last install and the absence clock then has to run its own course.
// An artifact that is still in use is reinstalled long before that and is never
// eligible. The conservative direction costs only disk.
var leaseTTL = artifactlease.DefaultGrace

// noteOwnership records ownership of an installed CLI.
//
// replaced distinguishes the two callers. An actual install claims, which bumps
// the generation; a freshness check that found the binary already current
// renews, which does not. Collapsing them made the generation count
// invocations instead of replacements.
//
// Failures warn rather than fail. An unwritable ownership record leaves the
// artifact unprotected, which is bad; refusing to install because of it would
// leave the operator with no CLI at all, which is worse. The warning goes to
// stderr so it survives --json output.
func (m *Manager) noteOwnership(item InstallableCLI, path string, replaced bool) {
	identity := artifactledger.CurrentIdentity()
	owner := artifactlease.Owner{
		Node:  identity.Node,
		User:  identity.User,
		Agent: identity.Claimed.Session,
	}
	record := artifactlease.Renew
	if replaced {
		record = artifactlease.Claim
	}
	if _, err := record(path, owner, item.ModulePath, leaseTTL, time.Now().UTC()); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN]    ownership record for %s could not be written: %v\n", item.Name, err)
	}
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
	return config.WriteOwnedFile(m.InstallMetadataPath(item), data, tuning.PermFile)
}

func (m *Manager) computeInstallFingerprint(item InstallableCLI) (string, error) {
	spec, err := item.FreshnessSpec()
	if err != nil {
		return "", err
	}
	return cliutil.ComputeFreshnessFingerprint(spec)
}

func (m *Manager) computeInstallManifest(item InstallableCLI) (cliutil.FreshnessManifest, error) {
	spec, err := item.FreshnessSpec()
	if err != nil {
		return cliutil.FreshnessManifest{}, err
	}
	return cliutil.ComputeFreshnessManifest(spec, string(item.Kind)+"-cli", nil, time.Now().UnixNano())
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
		case installationGoModule:
			return cliutil.CanonicalScenarioGoModuleFreshnessSpec(item.ScenarioPath, item.ModulePath, item.BinaryName, customInputs), nil
		default:
			return cliutil.FreshnessSpec{}, fmt.Errorf("unsupported scenario CLI adapter kind %q", item.CLI.Adapter.Kind)
		}
	case KindResource:
		switch item.CLI.Adapter.Kind {
		case installationGoModule:
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
	manifest, err := scenario.LoadServiceManifest(filepath.Join(configDir, "service.json"))
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(manifest.Dependencies.Resources))
	for name, entry := range manifest.Dependencies.Resources {
		if entry.Enabled {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names, nil
}

type GoInstaller struct{}

func (GoInstaller) Install(ctx context.Context, item InstallableCLI, installDir string) error {
	switch item.Kind {
	case KindScenario:
		if item.CLI == nil {
			return errors.New("scenario CLI manifest is required")
		}
		switch item.CLI.Adapter.Kind {
		case installationGoModule:
			return withCLIInstallAdmission(ctx, string(item.Kind)+"-"+item.Name, func(buildCtx context.Context) error {
				return runGoModuleInstaller(buildCtx, item, installDir)
			})
		default:
			return fmt.Errorf("unsupported scenario CLI adapter kind %q", item.CLI.Adapter.Kind)
		}
	case KindResource:
		if item.CLI == nil {
			return errors.New("resource CLI manifest is required")
		}
		switch item.CLI.Adapter.Kind {
		case installationGoModule:
			return withCLIInstallAdmission(ctx, string(item.Kind)+"-"+item.Name, func(buildCtx context.Context) error {
				return runGoModuleInstaller(buildCtx, item, installDir)
			})
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

	cmd := shell.NewCommandContext(ctx, "go", args...)
	cmd.Dir = installerDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return runOwnedCLICommand(ctx, cmd)
}

func runOwnedCLICommand(ctx context.Context, cmd *exec.Cmd) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if cmd == nil {
		return fmt.Errorf("owned CLI build command is nil")
	}
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return fmt.Errorf("configure owned CLI build process group: %w", err)
	}
	cmd.Cancel = func() error { return platform.GracefulStopProcess(cmd.Process) }
	cmd.WaitDelay = cliBuildStopGrace
	if err := cmd.Start(); err != nil {
		return err
	}
	// DetachedProcessAttrs creates a new session/process group whose identity
	// is the child PID on Unix. Capture that identity while the child is alive;
	// looking it up after Wait can fail once a short-lived root exits.
	groupID := cmd.Process.Pid
	runErr := cmd.Wait()
	if ctx.Err() != nil {
		cleanupErr := terminateAndWaitCLIProcessGroup(ctx, groupID)
		if errors.Is(cleanupErr, platform.ErrUnsupported) {
			// Preserve the pre-admission cross-platform behavior when the native
			// backend cannot verify descendants. The root command has exited, but
			// this path does not claim that its child processes have.
			return errors.Join(runErr, ctx.Err(), errors.New("descendant cleanup could not be verified on this platform"))
		}
		return errors.Join(runErr, ctx.Err(), cleanupErr)
	}

	// A successful root process is not sufficient evidence that its compiler
	// descendants exited. The installer owns this process group, so keep the
	// admission until the whole group is empty. If a descendant outlives a
	// successful root, stop the exact group and fail this install.
	waitCtx, cancelWait := context.WithTimeout(context.Background(), cliBuildCleanupTimeout)
	waitErr := platform.WaitProcessGroupContext(waitCtx, groupID, cliBuildGroupPollInterval)
	waitTimedOut := errors.Is(waitErr, context.DeadlineExceeded)
	cancelWait()
	if waitErr == nil {
		return runErr
	}
	if errors.Is(waitErr, platform.ErrUnsupported) {
		// On unsupported hosts, keep the existing install result after the root
		// exits. Descendant completion is explicitly outside this guarantee.
		return runErr
	}

	cleanupErr := terminateAndWaitCLIProcessGroup(context.Background(), groupID)
	if cleanupErr != nil {
		return errors.Join(runErr, fmt.Errorf("owned CLI build descendants could not be verified after root exit: %w", waitErr), cleanupErr)
	}
	if waitTimedOut {
		return errors.Join(runErr, fmt.Errorf("owned CLI build descendants remained after root exit and were stopped"))
	}
	return errors.Join(runErr, fmt.Errorf("owned CLI build process-group verification failed and the group was stopped: %w", waitErr))
}

func terminateAndWaitCLIProcessGroup(ctx context.Context, groupID int) error {
	probeCtx, cancelProbe := context.WithTimeout(context.WithoutCancel(ctx), cliBuildGroupProbe)
	probeErr := platform.WaitProcessGroupContext(probeCtx, groupID, cliBuildGroupPollInterval)
	cancelProbe()
	if probeErr == nil {
		return nil
	}
	if errors.Is(probeErr, platform.ErrUnsupported) {
		return errors.Join(errCLIProcessGroupCleanupUnverified, fmt.Errorf("verify owned CLI build process group %d before stopping it: %w", groupID, probeErr))
	}
	if !errors.Is(probeErr, context.DeadlineExceeded) {
		return errors.Join(errCLIProcessGroupCleanupUnverified, fmt.Errorf("inspect owned CLI build process group %d before stopping it: %w", groupID, probeErr))
	}
	signalErr := platform.SignalProcessGroup(groupID, true)
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cliBuildCleanupTimeout)
	defer cancel()
	if err := platform.WaitProcessGroupContext(cleanupCtx, groupID, cliBuildGroupPollInterval); err != nil {
		waitErr := fmt.Errorf("wait for owned CLI build process group %d to exit: %w", groupID, err)
		if signalErr != nil {
			return errors.Join(errCLIProcessGroupCleanupUnverified, fmt.Errorf("force-stop owned CLI build process group %d: %w", groupID, signalErr), waitErr)
		}
		return errors.Join(errCLIProcessGroupCleanupUnverified, waitErr)
	}
	return nil
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
	slices.Sort(names)
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
	if hostreqspec.PlatformFromGOOS(runtime.GOOS) == hostreqspec.PlatformWindows {
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
		Failures: make([]scenario.Failure, 0),
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
		report.Failures = append(report.Failures, scenario.Failure{
			Kind:  string(kind),
			Name:  name,
			Stage: "discover_cli",
			Error: err.Error(),
		})
	}
	sort.Slice(report.Items, func(i, j int) bool { return report.Items[i].Name < report.Items[j].Name })
	return report, nil
}
