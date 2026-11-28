package cliinstall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/rootmetrics"
)

var companionNames = []string{"vrooli-api", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog"}
var legacyBundleNames = append([]string{"vrooli", "vrooli-atomic-install"}, companionNames...)
var bundleNames = append(append([]string{}, legacyBundleNames...), "vrooli-watchdog")

func manifestBundleNames(manifest map[string]string) []string {
	if _, ok := manifest["vrooli-watchdog"]; ok {
		return bundleNames
	}
	return legacyBundleNames
}

// PrepareBundle creates a private, immutable-by-convention copy of a complete
// build. A source changing during the copy causes the whole preparation to fail.
func PrepareBundle(sourceDir, outputDir string) error {
	return prepareBundle(sourceDir, outputDir, syncPreparedPath)
}

func prepareBundle(sourceDir, outputDir string, syncPath func(string) error) (err error) {
	if err = os.Mkdir(outputDir, 0700); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(outputDir)
		}
	}()
	// Publish the lease before the first image write. If the process is killed
	// during preparation, age-based pruning can still distinguish this
	// abandoned bundle from one held by a live installer.
	if err = config.WriteOwnedFileDurable(filepath.Join(outputDir, ".lease"), nil, 0o600); err != nil {
		return fmt.Errorf("publish bundle lease: %w", err)
	}
	if err = config.ChownToInvokingUser(outputDir); err != nil {
		return err
	}
	var manifest strings.Builder
	for _, name := range bundleNames {
		path := filepath.Join(sourceDir, name)
		before, readErr := installedImageSHA(path)
		if readErr != nil {
			return fmt.Errorf("inspect %s: %w", name, readErr)
		}
		f, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		data, readErr := io.ReadAll(io.LimitReader(f, rootCLIImageLimit+1))
		closeErr := f.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if len(data) == 0 || int64(len(data)) > rootCLIImageLimit {
			return fmt.Errorf("invalid prepared image %s", name)
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		after, readErr := installedImageSHA(path)
		if readErr != nil {
			return readErr
		}
		if before != digest || after != digest {
			return fmt.Errorf("prepared image %s changed during snapshot", name)
		}
		if err = os.WriteFile(filepath.Join(outputDir, name), data, 0700); err != nil {
			return err
		}
		if err = config.ChownToInvokingUser(filepath.Join(outputDir, name)); err != nil {
			return err
		}
		if err = syncPath(filepath.Join(outputDir, name)); err != nil {
			return fmt.Errorf("sync prepared image %s: %w", name, err)
		}
		fmt.Fprintf(&manifest, "%s  %s\n", digest, name)
	}
	manifestPath := filepath.Join(outputDir, "SHA256SUMS")
	if err = os.WriteFile(manifestPath, []byte(manifest.String()), 0600); err != nil {
		return err
	}
	if err = config.ChownToInvokingUser(manifestPath); err != nil {
		return err
	}
	for _, path := range []string{manifestPath, outputDir, filepath.Dir(outputDir)} {
		if err = syncPath(path); err != nil {
			return fmt.Errorf("sync prepared bundle %s: %w", path, err)
		}
	}
	return nil
}

// InstallCompanions serializes ordinary publication with root activation and
// refuses to publish companions for a root image that has since been replaced.
func InstallCompanions(sourceDir, binDir, rootSHA string) error {
	if !rootCLIBuildIdentity.MatchString(rootSHA) {
		return fmt.Errorf("exact expected root SHA-256 required")
	}
	root := filepath.Join(binDir, "vrooli")
	releaseWorkflow, err := buildinfo.AcquireRootBundleWorkflowForTarget(context.Background(), root)
	if err != nil {
		return err
	}
	defer releaseWorkflow()
	release, err := buildinfo.AcquireBinaryInstallLock(root)
	if err != nil {
		return err
	}
	defer release()
	if err := buildinfo.GuardRootActivation(context.Background(), root); err != nil {
		return err
	}
	observed, err := installedImageSHA(root)
	if err != nil {
		return err
	}
	if observed != rootSHA {
		return fmt.Errorf("installed root changed before companion publication")
	}
	expected := make(map[string]string, len(companionNames))
	for _, name := range companionNames {
		sha, err := installedImageSHA(filepath.Join(sourceDir, name))
		if err != nil {
			return err
		}
		expected[name] = sha
	}
	return installCompanions(sourceDir, binDir, expected, nil)
}

// InstallPreparedCompanionsLocked requires the root install lock. The caller
// must verify the serving root digest while holding it. The approved manifest
// binds every source hash; each installer verifies its copied bytes again.
var beginBundleObservation = rootmetrics.BeginDeferred

func InstallPreparedCompanionsLocked(sourceDir, binDir, manifestSHA string) (resultErr error) {
	expected, err := readPreparedManifest(sourceDir, manifestSHA)
	if err != nil {
		return err
	}
	// Observations are only associated with the canonical runtime installation;
	// arbitrary callers and test destinations must not write into operator state.
	var attempt *rootmetrics.Deferred
	if home, err := config.HomeDir(); err == nil {
		if bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin); err == nil && filepath.Clean(binDir) == filepath.Clean(bin) {
			attempt = beginBundleObservation(home, "bundle_publication")
			attempt.Bundle(manifestSHA, expected)
			attempt.Event("qualified_manifest", "")
		}
	}
	defer func() {
		if resultErr != nil {
			attempt.Event("terminal", "failed")
		} else {
			attempt.Event("terminal", "succeeded")
		}
		attempt.Finish()
	}()
	return installCompanions(sourceDir, binDir, expected, attempt)
}

// PublishPreparedBundleSourceLocked completes the source binding while the
// caller holds the workflow and root publication locks.
func PublishPreparedBundleSourceLocked(sourceDir, binDir, manifestSHA, sourceRoot, home string) error {
	if !filepath.IsAbs(sourceRoot) || filepath.Clean(sourceRoot) != sourceRoot {
		return fmt.Errorf("source root must be a canonical absolute path")
	}
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return err
	}
	if filepath.Clean(binDir) != filepath.Clean(bin) {
		return fmt.Errorf("invalid runtime bin directory")
	}
	if err := VerifyPreparedBundle(sourceDir, manifestSHA); err != nil {
		return err
	}
	manifest, err := readPreparedManifest(sourceDir, manifestSHA)
	if err != nil {
		return err
	}
	for _, name := range append([]string{"vrooli"}, companionNames...) {
		actual, err := installedImageSHA(filepath.Join(bin, name))
		if err != nil || actual != manifest[name] {
			return fmt.Errorf("installed bundle changed before source pointer publication: %s", name)
		}
	}
	return config.WriteOwnedFileDurable(repocontract.SourceRootPointerPath(home), []byte(sourceRoot+"\n"), 0600)
}

// VerifyPreparedBundle checks every image before any root activation. The
// manifest digest must be retained by the caller through publication.
func VerifyPreparedBundle(sourceDir, manifestSHA string) error {
	expected, err := readPreparedManifest(sourceDir, manifestSHA)
	if err != nil {
		return err
	}
	for _, name := range manifestBundleNames(expected) {
		actual, err := installedImageSHA(filepath.Join(sourceDir, name))
		if err != nil {
			return fmt.Errorf("inspect prepared %s: %w", name, err)
		}
		if actual != expected[name] {
			return fmt.Errorf("prepared image %s changed", name)
		}
	}
	return nil
}

// PreparedBundleRootSHA returns the root identity bound by the verified manifest.
func PreparedBundleRootSHA(sourceDir, manifestSHA string) (string, error) {
	expected, err := readPreparedManifest(sourceDir, manifestSHA)
	if err != nil {
		return "", err
	}
	return expected["vrooli"], nil
}

// PreparedBundleWatchdogSHA returns the manifest-bound watchdog digest, or empty for a legacy bundle.
func PreparedBundleWatchdogSHA(sourceDir, manifestSHA string) (string, error) {
	expected, err := readPreparedManifest(sourceDir, manifestSHA)
	if err != nil {
		return "", err
	}
	return expected["vrooli-watchdog"], nil
}

// InstallPreparedCompanions fences ordinary publication against root changes.
func InstallPreparedCompanions(sourceDir, binDir, rootSHA, manifestSHA string) error {
	release, err := buildinfo.AcquireRootBundleWorkflowForTarget(context.Background(), filepath.Join(binDir, "vrooli"))
	if err != nil {
		return err
	}
	defer release()
	return installPreparedCompanions(sourceDir, binDir, rootSHA, manifestSHA)
}

// PublishPreparedBundleSource publishes the checkout only while its exact
// bundle still owns the installed root and companions. A later competing installation
// cannot be overwritten by a delayed source pointer update.
func PublishPreparedBundleSource(sourceDir, binDir, manifestSHA, sourceRoot, pointer string) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return err
	}
	if filepath.Clean(binDir) != filepath.Clean(bin) || filepath.Clean(pointer) != repocontract.SourceRootPointerPath(home) || !filepath.IsAbs(sourceRoot) || filepath.Clean(sourceRoot) != sourceRoot {
		return fmt.Errorf("source pointer publication requires canonical runtime paths and an absolute checkout")
	}
	release, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), home)
	if err != nil {
		return err
	}
	defer release()
	root := filepath.Join(bin, "vrooli")
	rootRelease, err := buildinfo.AcquireBinaryInstallLock(root)
	if err != nil {
		return err
	}
	defer rootRelease()
	if err := buildinfo.GuardRootActivation(context.Background(), root); err != nil {
		return err
	}
	return PublishPreparedBundleSourceLocked(sourceDir, binDir, manifestSHA, sourceRoot, home)
}

// installPreparedCompanions requires the canonical bundle workflow lock.
func installPreparedCompanions(sourceDir, binDir, rootSHA, manifestSHA string) error {
	return installPreparedCompanionsOwned(sourceDir, binDir, rootSHA, manifestSHA, false)
}

func installPreparedCompanionsOwned(sourceDir, binDir, rootSHA, manifestSHA string, owned bool) error {
	if !rootCLIBuildIdentity.MatchString(rootSHA) {
		return fmt.Errorf("exact expected root SHA-256 required")
	}
	expected, err := readPreparedManifest(sourceDir, manifestSHA)
	if err != nil {
		return err
	}
	if expected["vrooli"] != rootSHA {
		return fmt.Errorf("prepared bundle does not match expected root")
	}
	root := filepath.Join(binDir, "vrooli")
	release, err := buildinfo.AcquireBinaryInstallLock(root)
	if err != nil {
		return err
	}
	defer release()
	guard := buildinfo.GuardRootActivation
	if owned {
		guard = func(ctx context.Context, target string) error {
			return buildinfo.GuardRootActivationForPrepared(ctx, target, filepath.Join(sourceDir, "vrooli"))
		}
	}
	if err := guard(context.Background(), root); err != nil {
		return err
	}
	observed, err := installedImageSHA(root)
	if err != nil {
		return err
	}
	if observed != rootSHA {
		return fmt.Errorf("installed root changed before companion publication")
	}
	return InstallPreparedCompanionsLocked(sourceDir, binDir, manifestSHA)
}

func readPreparedManifest(sourceDir, manifestSHA string) (map[string]string, error) {
	if !rootCLIBuildIdentity.MatchString(manifestSHA) {
		return nil, fmt.Errorf("exact approved bundle SHA-256 required")
	}
	data, err := os.ReadFile(filepath.Join(sourceDir, "SHA256SUMS"))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > 8192 {
		return nil, fmt.Errorf("invalid bundle manifest")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != manifestSHA {
		return nil, fmt.Errorf("prepared bundle differs from the approved manifest")
	}
	expected := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !rootCLIBuildIdentity.MatchString(fields[0]) {
			return nil, fmt.Errorf("invalid bundle manifest entry")
		}
		if _, exists := expected[fields[1]]; exists {
			return nil, fmt.Errorf("duplicate bundle manifest entry")
		}
		expected[fields[1]] = fields[0]
	}
	for _, name := range legacyBundleNames {
		if !rootCLIBuildIdentity.MatchString(expected[name]) {
			return nil, fmt.Errorf("missing prepared image %s", name)
		}
	}
	if sha, ok := expected["vrooli-watchdog"]; ok && !rootCLIBuildIdentity.MatchString(sha) {
		return nil, fmt.Errorf("invalid watchdog digest")
	}
	if len(expected) != len(manifestBundleNames(expected)) {
		return nil, fmt.Errorf("unexpected bundle manifest entry")
	}
	return expected, nil
}
func installCompanions(sourceDir, binDir string, expected map[string]string, attempt *rootmetrics.Deferred) error {
	// Refuse a known incomplete bundle before publishing the first companion.
	// Each source is checked again while copied to close replacement races.
	for _, name := range companionNames {
		sha := expected[name]
		if !rootCLIBuildIdentity.MatchString(sha) {
			return fmt.Errorf("missing prepared companion %s", name)
		}
		actual, err := installedImageSHA(filepath.Join(sourceDir, name))
		if err != nil {
			return err
		}
		if actual != sha {
			return fmt.Errorf("prepared companion %s changed", name)
		}
	}
	for _, name := range companionNames {
		sha := expected[name]
		if !rootCLIBuildIdentity.MatchString(sha) {
			return fmt.Errorf("missing prepared companion %s", name)
		}
		if err := atomicInstallVerifiedSource(filepath.Join(sourceDir, name), filepath.Join(binDir, name), sha); err != nil {
			return fmt.Errorf("install companion %s: %w", name, err)
		}
		attempt.Event(strings.ReplaceAll(name, "-", "_"), "published")
	}
	return nil
}
