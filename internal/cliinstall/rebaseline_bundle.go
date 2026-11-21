package cliinstall

import (
	"fmt"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

// RebaselineBundlePath resolves the runtime-owned, content-addressed recovery
// location. The manifest digest is a locator and does not itself grant authority.
func RebaselineBundlePath(home, manifestSHA string) (string, error) {
	if !rootCLIBuildIdentity.MatchString(manifestSHA) {
		return "", fmt.Errorf("exact bundle SHA-256 required")
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "root-rebaseline", "prepared-bundles", manifestSHA), nil
}

// RetainRebaselineBundle persists the exact manifest-bound image set before a
// progress record may refer to it. Callers serialize this operation with the
// root bundle workflow lock, including cleanup. A previous complete copy is
// reused only after checking every digest. Interrupted staging is discarded.
func RetainRebaselineBundle(home, source, manifestSHA string) (string, error) {
	return retainRebaselineBundle(home, source, manifestSHA, copyRootCLIImage, config.SyncOwnedDirectoryChain)
}

func retainRebaselineBundle(home, source, manifestSHA string, copyImage func(string, string, string, os.FileMode) (string, error), syncDirectories func(string) error) (string, error) {
	dest, err := RebaselineBundlePath(home, manifestSHA)
	if err != nil {
		return "", err
	}
	if info, err := os.Lstat(dest); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("retained bundle is not a directory")
		}
		if err := VerifyPreparedBundle(dest, manifestSHA); err != nil {
			return "", fmt.Errorf("retained bundle invalid: %w", err)
		}
		return dest, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	manifest, err := readPreparedManifest(source, manifestSHA)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(dest)
	if _, err := config.EnsureOwnedDirMode(parent, 0700); err != nil {
		return "", err
	}
	if err := syncDirectories(parent); err != nil {
		return "", err
	}
	stage := dest + ".staging"
	if err := os.RemoveAll(stage); err != nil {
		return "", err
	}
	if _, err := config.EnsureOwnedDirMode(stage, 0700); err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	for _, name := range manifestBundleNames(manifest) {
		if _, err := copyImage(filepath.Join(source, name), filepath.Join(stage, name), manifest[name], 0700); err != nil {
			return "", fmt.Errorf("retain %s: %w", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(source, "SHA256SUMS"))
	if err != nil {
		return "", err
	}
	if err := config.WriteOwnedFileDurable(filepath.Join(stage, "SHA256SUMS"), data, 0600); err != nil {
		return "", err
	}
	if err := VerifyPreparedBundle(stage, manifestSHA); err != nil {
		return "", fmt.Errorf("verify retained bundle: %w", err)
	}
	if err := syncDirectories(stage); err != nil {
		return "", err
	}
	if err := os.Rename(stage, dest); err != nil {
		return "", err
	}
	if err := syncDirectories(parent); err != nil {
		// Rename may already have made the bundle visible even though directory
		// durability is uncertain. Return its known location so recovery guidance
		// can inspect it instead of losing the content-addressed path.
		return dest, err
	}
	return dest, nil
}

// RemoveRebaselineBundle retires only the named recovery bundle after its
// owner has durably recorded completion. Call under the workflow lock.
func RemoveRebaselineBundle(home, manifestSHA string) error {
	path, err := RebaselineBundlePath(home, manifestSHA)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return config.SyncOwnedDirectoryChain(filepath.Dir(path))
}
