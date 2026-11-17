package runtimehandlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	"github.com/vrooli/vrooli/internal/rootdecision"
)

type preparedRootBundleIdentity struct {
	manifestSHA string
	rootSHA     string
}

type preparedSnapshotDecisionIdentity struct {
	terms rootDecisionTerms
}

// loadPreparedHubDecisionLocator and savePreparedHubDecisionLocator are the
// only runtimehandler path through the local durable Hub locator codec.
func loadPreparedHubDecisionLocator(path string) (rootDecisionTerms, error) {
	return rootdecision.Load(path)
}

func savePreparedHubDecisionLocator(path string, terms rootDecisionTerms) error {
	return rootdecision.Save(path, terms)
}

func loadPreparedSnapshotDecision(snapshot string) (preparedSnapshotDecisionIdentity, error) {
	terms, err := loadPreparedHubDecisionLocator(filepath.Join(snapshot, rootdecision.PendingFilename))
	if err != nil {
		return preparedSnapshotDecisionIdentity{}, fmt.Errorf("snapshot decision identity unavailable")
	}
	if terms.RequesterRef == "" {
		return preparedSnapshotDecisionIdentity{}, fmt.Errorf("snapshot decision identity unavailable")
	}
	manifest, err := os.ReadFile(filepath.Join(snapshot, "SHA256SUMS"))
	if err != nil || len(manifest) == 0 || len(manifest) > 8192 {
		return preparedSnapshotDecisionIdentity{}, fmt.Errorf("snapshot manifest unavailable")
	}
	sum := sha256.Sum256(manifest)
	manifestSHA := hex.EncodeToString(sum[:])
	if manifestSHA != terms.BundleSHA {
		return preparedSnapshotDecisionIdentity{}, fmt.Errorf("snapshot decision does not match bundle")
	}
	return preparedSnapshotDecisionIdentity{terms: terms}, nil
}

// readPreparedRootBundle verifies the durable bundle manifest and binds the
// exact root image whose contract will be examined or activated.
func readPreparedRootBundle(candidate string) (preparedRootBundleIdentity, error) {
	manifestSHA, err := preparedBundleManifestSHA(filepath.Dir(candidate))
	if err != nil {
		return preparedRootBundleIdentity{}, fmt.Errorf("prepared bundle manifest unavailable or invalid")
	}
	if err := cliinstall.VerifyPreparedBundle(filepath.Dir(candidate), manifestSHA); err != nil {
		return preparedRootBundleIdentity{}, err
	}
	rootSHA, err := cliinstall.PreparedBundleRootSHA(filepath.Dir(candidate), manifestSHA)
	if err != nil {
		return preparedRootBundleIdentity{}, err
	}
	return preparedRootBundleIdentity{manifestSHA: manifestSHA, rootSHA: rootSHA}, nil
}

func preparedBundleManifestSHA(bundleDir string) (string, error) {
	manifest, err := os.ReadFile(filepath.Join(bundleDir, "SHA256SUMS"))
	if err != nil || len(manifest) == 0 || len(manifest) > 8192 {
		return "", fmt.Errorf("prepared bundle manifest unavailable or invalid")
	}
	sum := sha256.Sum256(manifest)
	return hex.EncodeToString(sum[:]), nil
}

type preparedRootRecovery struct {
	home        string
	candidate   string
	bundleSHA   string
	pendingPath string
	hubDecision bool
	recovery    bool
}

func newPreparedRootRecovery(home, candidate, bundleSHA string, hubDecision, recovery bool) (*preparedRootRecovery, bool, error) {
	workflow := &preparedRootRecovery{
		home:        home,
		candidate:   candidate,
		bundleSHA:   bundleSHA,
		hubDecision: hubDecision,
		recovery:    recovery,
	}
	if !hubDecision {
		return workflow, false, nil
	}
	workflow.pendingPath = filepath.Join(filepath.Dir(candidate), rootdecision.PendingFilename)
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, false, err
	}
	progress, err := rebaselineprogress.Read(filepath.Join(state, "root-rebaseline", "progress.json"))
	if err != nil {
		return nil, false, err
	}
	if progress != nil && !progress.Complete && progress.Bundle == bundleSHA && progress.Mode == "" {
		terms, err := loadPreparedHubDecisionLocator(workflow.pendingPath)
		if err != nil {
			return nil, false, err
		}
		return workflow, legacyFinishingNeedsDecision(progress, bundleSHA, terms), nil
	}
	return workflow, false, nil
}

func (r *preparedRootRecovery) beforeEffects(ctx context.Context, admission runtimeapp.RebaselineAdmission) error {
	if !r.hubDecision {
		return nil
	}
	client, err := rootDecisionClientFactory(ctx)
	if err != nil {
		return err
	}
	return verifyConsumedHubDecision(ctx, client, r.pendingPath, r.bundleSHA, admission.CandidateSHA, admission.WatchdogSHA)
}

func (r *preparedRootRecovery) prepare(ctx context.Context, recovery runtimeapp.RebaselineRecovery) error {
	if !r.hubDecision {
		return nil
	}
	client, err := rootDecisionClientFactory(ctx)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(r.pendingPath)
	if err != nil {
		return err
	}
	if len(data) == 0 || len(data) > 16384 {
		return fmt.Errorf("invalid Hub recovery locator")
	}
	retainedPending := filepath.Join(recovery.BundleDir, rootdecision.PendingFilename)
	if r.pendingPath != retainedPending {
		if err := config.WriteOwnedFileDurable(retainedPending, data, 0600); err != nil {
			return err
		}
	}
	return verifyConsumedHubDecision(ctx, client, retainedPending, r.bundleSHA, recovery.Admission.CandidateSHA, recovery.Admission.WatchdogSHA)
}

func (r *preparedRootRecovery) afterActivation(ctx context.Context, recovery runtimeapp.RebaselineRecovery, source string) error {
	if r.hubDecision {
		client, err := rootDecisionClientFactory(ctx)
		if err != nil {
			return err
		}
		retainedPending := filepath.Join(recovery.BundleDir, rootdecision.PendingFilename)
		if err := verifyConsumedHubDecision(ctx, client, retainedPending, r.bundleSHA, recovery.Admission.CandidateSHA, recovery.Admission.WatchdogSHA); err != nil {
			return err
		}
	}
	binDir, err := repocontract.RuntimeHomeEntryPath(r.home, repocontract.HomeKeyBin)
	if err != nil {
		return err
	}
	if err := cliinstall.InstallPreparedCompanionsLocked(recovery.BundleDir, binDir, r.bundleSHA); err != nil {
		return err
	}
	return cliinstall.PublishPreparedBundleSourceLocked(recovery.BundleDir, binDir, r.bundleSHA, source, r.home)
}

func (r *preparedRootRecovery) watchdogSnapshot() string {
	if !r.hubDecision {
		return ""
	}
	if r.recovery {
		state, _ := repocontract.RuntimeHomeEntryPath(r.home, repocontract.HomeKeyState)
		return filepath.Join(state, "root-rebaseline", "watchdog-pending")
	}
	return filepath.Join(filepath.Dir(r.candidate), rootdecision.WatchdogFilename)
}

func legacyFinishingNeedsDecision(p *rebaselineprogress.Progress, bundle string, terms rootDecisionTerms) bool {
	return p != nil && !p.Complete && p.Bundle == bundle && p.Mode == "" && (terms.RequesterRef == "" || terms.Operation == "continuation")
}

// retainedRebaselineSource resolves only the runtime-owned bundle named by an
// incomplete transition. Progress is a locator, never human authority.
func retainedRebaselineSource(home, candidate string) (string, bool, error) {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", false, err
	}
	p, err := rebaselineprogress.Read(filepath.Join(state, "root-rebaseline", "progress.json"))
	if err != nil {
		return "", false, err
	}
	if p == nil || p.Complete || p.Bundle == "" {
		return "", false, nil
	}
	retained, err := cliinstall.RebaselineBundlePath(home, p.Bundle)
	if err != nil {
		return "", false, err
	}
	if filepath.Clean(candidate) != filepath.Join(retained, "vrooli") {
		return "", false, nil
	}
	if !filepath.IsAbs(p.SourceRoot) || filepath.Clean(p.SourceRoot) != p.SourceRoot {
		return "", false, fmt.Errorf("interrupted rebaseline lacks a canonical source binding")
	}
	return p.SourceRoot, true, nil
}
