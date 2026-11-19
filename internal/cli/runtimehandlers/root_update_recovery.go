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
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
)

type preparedRootBundleIdentity struct {
	manifestSHA string
	rootSHA     string
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

// resumePreparedCandidate resolves the prepared bundle a paused transition
// points at. It is the recovery owner's read; the dispatcher only asks whether
// there is one and where it is.
func resumePreparedCandidate(home string) (string, bool, error) {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", false, err
	}
	progress, err := rebaselineprogress.Read(filepath.Join(state, "root-rebaseline", "progress.json"))
	if err != nil || progress == nil || progress.Complete {
		return "", false, err
	}
	candidate := filepath.Join(state, "root-rebaseline", "prepared-bundles", progress.Bundle, "vrooli")
	if _, err := os.Stat(candidate); err != nil {
		return "", false, fmt.Errorf("the prepared bundle for the interrupted transition is gone (%v); run `make rebaseline` to prepare a new one", err)
	}
	return candidate, true, nil
}

// preparedRootRecovery is the prepared transition's companion publication. It
// holds no authority: the operator gate and the plan confirmation have already
// happened by the time it runs.
type preparedRootRecovery struct {
	home      string
	bundleSHA string
}

func newPreparedRootRecovery(home, bundleSHA string) *preparedRootRecovery {
	return &preparedRootRecovery{home: home, bundleSHA: bundleSHA}
}

func (r *preparedRootRecovery) afterActivation(ctx context.Context, recovery runtimeapp.RebaselineRecovery, source string) error {
	binDir, err := repocontract.RuntimeHomeEntryPath(r.home, repocontract.HomeKeyBin)
	if err != nil {
		return err
	}
	if err := cliinstall.InstallPreparedCompanionsLocked(recovery.BundleDir, binDir, r.bundleSHA); err != nil {
		return err
	}
	return cliinstall.PublishPreparedBundleSourceLocked(recovery.BundleDir, binDir, r.bundleSHA, source, r.home)
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

// preparedBundleCandidate reports whether candidate names a root image inside a
// directory carrying a bundle manifest.
func preparedBundleCandidate(candidate string) bool {
	if filepath.Base(candidate) != "vrooli" {
		return false
	}
	info, err := os.Stat(filepath.Join(filepath.Dir(candidate), "SHA256SUMS"))
	return err == nil && !info.IsDir()
}
