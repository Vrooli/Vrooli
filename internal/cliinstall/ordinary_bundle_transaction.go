package cliinstall

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	watchdog "github.com/vrooli/vrooli/internal/safeguards/emergency-watchdog"
)

// InstallPreparedBundle serializes ordinary root activation and companions
// against rebaseline workflows, including same-root updates.
func installPreparedBundle(sourceDir, binDir, manifestSHA string) error {
	return installPreparedBundleWithSource(sourceDir, binDir, manifestSHA, "")
}

func installPreparedBundleWithSource(sourceDir, binDir, manifestSHA, sourceRoot string) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	canonicalBin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return err
	}
	if filepath.Clean(binDir) != filepath.Clean(canonicalBin) {
		return fmt.Errorf("prepared bundle destination is not the runtime bin directory")
	}
	release, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), home)
	if err != nil {
		return err
	}
	defer release()
	if err := buildinfo.GuardRebaselineCompletion(home); err != nil {
		return err
	}
	servingSHA, err := qualifyOrdinaryBundle(home, sourceDir, binDir, manifestSHA, false)
	if err != nil {
		return err
	}
	retained, err := beginOrdinaryBundleWithSource(home, sourceDir, binDir, manifestSHA, false, servingSHA, sourceRoot)
	if err != nil {
		return err
	}
	manifest, err := readPreparedManifest(retained, manifestSHA)
	if err != nil {
		return err
	}
	root := filepath.Join(binDir, "vrooli")
	current, err := installedImageSHA(root)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if current != manifest["vrooli"] {
		if err := atomicInstall(filepath.Join(retained, "vrooli"), root, "", atomicInstallOptions{purpose: atomicInstallPrepared, sourceRoot: sourceRoot}); err != nil {
			return err
		}
	}
	if err := installPreparedCompanionsOwned(retained, binDir, manifest["vrooli"], manifestSHA, true); err != nil {
		return err
	}
	return finishOrdinaryBundle(home, binDir, manifest)

}

// InstallPreparedBundleSameContract requalifies under the bundle workflow
// lock. Unenrolled publication binds the comparison to the root install lock;
// enrolled publication delegates qualification to the watchdog admission owner.
func installPreparedBundleSameContract(sourceDir, binDir, manifestSHA string) error {
	return installPreparedBundleSameContractWithSource(sourceDir, binDir, manifestSHA, "")
}

func installPreparedBundleSameContractWithSource(sourceDir, binDir, manifestSHA, sourceRoot string) error {
	return installPreparedBundleSameContractWithSourceChecked(sourceDir, binDir, manifestSHA, sourceRoot, nil)
}

// InstallPreparedBundleSameContractWithSourceChecked rechecks a caller's source
// binding while holding the bundle workflow fence and before recording intent.
func installPreparedBundleSameContractWithSourceChecked(sourceDir, binDir, manifestSHA, sourceRoot string, check func() error) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	canonicalBin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return err
	}
	if filepath.Clean(binDir) != filepath.Clean(canonicalBin) {
		return fmt.Errorf("prepared bundle destination is not the runtime bin directory")
	}
	release, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), home)
	if err != nil {
		return err
	}
	defer release()
	if err := buildinfo.GuardRebaselineCompletion(home); err != nil {
		return err
	}
	if check != nil {
		if err := check(); err != nil {
			return fmt.Errorf("recheck prepared source: %w", err)
		}
	}
	qualifiedPrior, err := qualifyOrdinaryBundle(home, sourceDir, binDir, manifestSHA, true)
	if err != nil {
		return err
	}
	if qualifiedPrior == "absent" {
		return fmt.Errorf("same-contract installation requires an existing serving root")
	}
	retained, err := beginOrdinaryBundleWithSource(home, sourceDir, binDir, manifestSHA, true, qualifiedPrior, sourceRoot)
	if err != nil {
		return err
	}
	manifest, err := readPreparedManifest(retained, manifestSHA)
	if err != nil {
		return err
	}
	privateRoot := filepath.Join(retained, "vrooli")
	root := filepath.Join(binDir, "vrooli")
	// Bind the observed identity to its digest while holding the same lock
	// used by root publication. AtomicInstallExpected checks it again.
	servingSHA := qualifiedPrior
	if servingSHA == "" {
		servingSHA, err = func() (string, error) {
			releaseRoot, err := buildinfo.AcquireBinaryInstallLock(root)
			if err != nil {
				return "", err
			}
			defer releaseRoot()
			digest, err := installedImageSHA(root)
			if err != nil {
				return "", err
			}
			serving, err := probeRootCLIIdentity(context.Background(), root)
			if err != nil {
				return "", err
			}
			candidate, err := probeRootCLIIdentity(context.Background(), privateRoot)
			if err != nil {
				return "", err
			}
			if err := coreactivation.QualifyRollbackIdentities(serving, candidate); err != nil {
				return "", fmt.Errorf("serving root changed or is incompatible with prepared candidate: %w", err)
			}
			return digest, nil
		}()
		if err != nil {
			return err
		}
	}
	enrolled, err := buildinfo.EnrolledRootUpdate(root)
	if err != nil {
		return err
	}
	if servingSHA == manifest["vrooli"] {
		// The exact root already serves. Companion publication rechecks it
		// under the publication lock, so no root trial is necessary.
	} else if enrolled {
		source := sourceRoot
		if source == "" {
			source, err = buildinfo.ResolveSourceRoot()
			if err != nil {
				return err
			}
		}
		if err := buildinfo.RequestRootUpdate(context.Background(), privateRoot, source); err != nil {
			return err
		}
	} else if err := atomicInstall(privateRoot, root, servingSHA, atomicInstallOptions{purpose: atomicInstallPrepared, sourceRoot: sourceRoot}); err != nil {
		return err
	}
	if err := installPreparedCompanionsOwned(retained, binDir, manifest["vrooli"], manifestSHA, true); err != nil {
		return err
	}
	return finishOrdinaryBundle(home, binDir, manifest)
}

// ResumePreparedBundle completes the exact durable bundle after interruption.
func resumePreparedBundle(binDir string) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	intent, err := buildinfo.ReadPreparedBundleIntent(home)
	if err != nil {
		return err
	}
	if intent == nil {
		return ordinaryBundleNoRecoveryError()
	}
	_, retained, err := buildinfo.PreparedBundlePaths(home)
	if err != nil {
		return err
	}
	if intent.SameContract {
		return installPreparedBundleSameContractWithSource(retained, binDir, intent.Manifest, intent.SourceRoot)
	}
	return installPreparedBundleWithSource(retained, binDir, intent.Manifest, intent.SourceRoot)
}

// AbortPreparedBundle discards an unstarted or fully rolled-back transaction
// only when every installed image still matches the recorded prior state.
func abortPreparedBundle(binDir string) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	canonical, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return err
	}
	if filepath.Clean(binDir) != filepath.Clean(canonical) {
		return fmt.Errorf("prepared bundle destination is not the runtime bin directory")
	}
	workflow, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), home)
	if err != nil {
		return err
	}
	defer workflow()
	transition, err := buildinfo.AcquireRebaselineTransition(context.Background(), home)
	if err != nil {
		return err
	}
	defer transition()
	publication, err := watchdog.AcquirePublicationLock(context.Background(), home)
	if err != nil {
		return err
	}
	defer publication()
	root, err := buildinfo.AcquireBinaryInstallLock(filepath.Join(binDir, "vrooli"))
	if err != nil {
		return err
	}
	defer root()
	if err := buildinfo.GuardRootActivationOperation(context.Background(), filepath.Join(binDir, "vrooli")); err != nil {
		return err
	}
	intent, err := buildinfo.ReadPreparedBundleIntent(home)
	if err != nil {
		return err
	}
	if intent == nil {
		return ordinaryBundleNoRecoveryError()
	}
	for _, name := range append([]string{"vrooli"}, companionNames...) {
		prior, present := intent.Prior[name]
		if !present {
			return fmt.Errorf("recovery intent lacks prior identity for %s", name)
		}
		actual, err := installedImageSHA(filepath.Join(binDir, name))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if actual != prior {
			return fmt.Errorf("cannot abort: installed %s differs from its prior identity; resume the retained bundle", name)
		}
	}
	return clearOrdinaryBundleIntent(home)
}

// qualifyOrdinaryBundle prevents direct unenrolled upgrades across persistence
// contracts. A missing root is a first install; a durable intent was already
// qualified at admission and resumes using its recorded mode.
func qualifyOrdinaryBundle(home, source, bin, manifestSHA string, requireIdentity bool) (string, error) {
	intent, err := buildinfo.ReadPreparedBundleIntent(home)
	if err != nil {
		return "", err
	}
	if intent != nil {
		return "", nil
	}
	if err := VerifyPreparedBundle(source, manifestSHA); err != nil {
		return "", err
	}
	manifest, err := readPreparedManifest(source, manifestSHA)
	if err != nil {
		return "", err
	}
	root := filepath.Join(bin, "vrooli")
	lock, err := buildinfo.AcquireBinaryInstallLock(root)
	if err != nil {
		return "", err
	}
	defer lock()
	prior, err := installedImageSHA(root)
	if os.IsNotExist(err) {
		return "absent", nil
	}
	if err != nil {
		return "", err
	}
	if prior == manifest["vrooli"] && !requireIdentity {
		return prior, nil
	}
	serving, err := probeRootCLIIdentity(context.Background(), root)
	if err != nil {
		return "", fmt.Errorf("qualify serving root: %w", err)
	}
	private, err := os.MkdirTemp(source, ".qualified-root-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(private)
	candidatePath := filepath.Join(private, "vrooli")
	if _, err := copyRootCLIImage(filepath.Join(source, "vrooli"), candidatePath, manifest["vrooli"], 0700); err != nil {
		return "", err
	}
	candidate, err := probeRootCLIIdentity(context.Background(), candidatePath)
	if err != nil {
		return "", fmt.Errorf("qualify candidate root: %w", err)
	}
	if err := coreactivation.QualifyRollbackIdentities(serving, candidate); err != nil {
		return "", fmt.Errorf("ordinary bundle requires a compatible serving contract; use rebaseline: %w", err)
	}
	return prior, nil
}

// beginOrdinaryBundle retains the complete image set before recording intent.
// All readers of the fence serialize with at least one of these locks.
func beginOrdinaryBundle(home, source, bin, manifestSHA string, sameContract bool, qualifiedPrior string) (string, error) {
	return beginOrdinaryBundleWithSource(home, source, bin, manifestSHA, sameContract, qualifiedPrior, "")
}

func beginOrdinaryBundleWithSource(home, source, bin, manifestSHA string, sameContract bool, qualifiedPrior, sourceRoot string) (string, error) {
	if sourceRoot != "" && (!filepath.IsAbs(sourceRoot) || filepath.Clean(sourceRoot) != sourceRoot) {
		return "", fmt.Errorf("source root must be a canonical absolute path")
	}
	_, retained, err := buildinfo.PreparedBundlePaths(home)
	if err != nil {
		return "", err
	}
	transition, err := buildinfo.AcquireRebaselineTransition(context.Background(), home)
	if err != nil {
		return "", err
	}
	defer transition()
	publication, err := watchdog.AcquirePublicationLock(context.Background(), home)
	if err != nil {
		return "", err
	}
	defer publication()
	rootLock, err := buildinfo.AcquireBinaryInstallLock(filepath.Join(bin, "vrooli"))
	if err != nil {
		return "", err
	}
	defer rootLock()
	if err := buildinfo.GuardRebaselineCompletion(home); err != nil {
		return "", err
	}
	if err := buildinfo.GuardRootActivationOperation(context.Background(), filepath.Join(bin, "vrooli")); err != nil {
		return "", err
	}
	existing, err := buildinfo.ReadPreparedBundleIntent(home)
	if err != nil {
		return "", err
	}
	if existing != nil {
		if existing.Manifest != manifestSHA || existing.SameContract != sameContract || existing.SourceRoot != sourceRoot {
			return "", fmt.Errorf("an ordinary bundle install is incomplete; resume with %q --resume-bundle %q (manifest %s)", filepath.Join(retained, "vrooli-atomic-install"), bin, existing.Manifest)
		}
		if err := VerifyPreparedBundle(retained, existing.Manifest); err != nil {
			return "", fmt.Errorf("retained bundle recovery failed: %w", err)
		}
		return retained, nil
	}
	if qualifiedPrior != "" {
		actual, err := installedImageSHA(filepath.Join(bin, "vrooli"))
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if actual == "" {
			actual = "absent"
		}
		if actual != qualifiedPrior {
			return "", fmt.Errorf("serving root changed after bundle qualification")
		}
	}
	if err := VerifyPreparedBundle(source, manifestSHA); err != nil {
		return "", err
	}
	manifest, err := readPreparedManifest(source, manifestSHA)
	if err != nil {
		return "", err
	}
	if err := os.RemoveAll(retained); err != nil {
		return "", err
	}
	if _, err := config.EnsureOwnedDirMode(retained, 0700); err != nil {
		return "", err
	}
	// Persist the newly created recovery-directory entry as well as the files
	// inside it before making the intent durable.
	parent, err := os.Open(filepath.Dir(filepath.Dir(retained)))
	if err != nil {
		return "", err
	}
	err = parent.Sync()
	closeErr := parent.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}

	for _, name := range append(append([]string{}, manifestBundleNames(manifest)...), "SHA256SUMS") {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			return "", err
		}
		if err := config.WriteOwnedFileDurable(filepath.Join(retained, name), data, 0700); err != nil {
			return "", err
		}
	}
	if err := VerifyPreparedBundle(retained, manifestSHA); err != nil {
		return "", err
	}
	prior := map[string]string{}
	for _, name := range append([]string{"vrooli"}, companionNames...) {
		digest, err := installedImageSHA(filepath.Join(bin, name))
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		prior[name] = digest
	}
	if err := buildinfo.WritePreparedBundleIntent(home, buildinfo.PreparedBundleIntent{Version: 1, Manifest: manifestSHA, Root: manifest["vrooli"], Prior: prior, SameContract: sameContract, SourceRoot: sourceRoot}); err != nil {
		return "", err
	}
	return retained, nil
}

func finishOrdinaryBundle(home, bin string, expected map[string]string) error {
	lock, err := buildinfo.AcquireBinaryInstallLock(filepath.Join(bin, "vrooli"))
	if err != nil {
		return err
	}
	defer lock()
	for _, name := range append([]string{"vrooli"}, companionNames...) {
		actual, err := installedImageSHA(filepath.Join(bin, name))
		if err != nil {
			return err
		}
		if actual != expected[name] {
			return fmt.Errorf("installed bundle image %s differs before completion", name)
		}
	}
	intent, err := buildinfo.ReadPreparedBundleIntent(home)
	if err != nil {
		return err
	}
	if intent == nil {
		return fmt.Errorf("ordinary bundle intent disappeared before completion")
	}
	if intent.SourceRoot != "" {
		if err := config.WriteOwnedFileDurable(repocontract.SourceRootPointerPath(home), []byte(intent.SourceRoot+"\n"), 0600); err != nil {
			return err
		}
	}
	return clearOrdinaryBundleIntent(home)
}

func clearOrdinaryBundleIntent(home string) error {
	path, _, err := buildinfo.PreparedBundlePaths(home)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
