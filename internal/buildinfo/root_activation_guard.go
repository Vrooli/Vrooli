package buildinfo

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

// GuardRootActivation keeps ordinary installers and self-rebuilders from
// crossing an admitted root operation. Callers already own the binary lock.
// Scratch and scenario binaries do not mutate the canonical serving image.
func GuardRootActivation(ctx context.Context, executable string) error {
	home, canonical, err := canonicalRootTarget(executable)
	if err != nil || !canonical {
		return err
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		return err
	}
	return store.GuardOrdinaryMutation(ctx)
}

func canonicalRootTarget(executable string) (string, bool, error) {
	if filepath.Base(executable) != "vrooli" && filepath.Base(executable) != "vrooli.exe" {
		return "", false, nil
	}
	home, err := homeDirFn()
	if err != nil {
		return "", false, err
	}
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return "", false, err
	}
	if filepath.Clean(filepath.Dir(executable)) != filepath.Clean(bin) {
		destinationParent, err := filepath.EvalSymlinks(filepath.Dir(executable))
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		if err != nil {
			return "", false, err
		}
		canonicalParent, err := filepath.EvalSymlinks(bin)
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		if err != nil {
			return "", false, err
		}
		if destinationParent != canonicalParent {
			return "", false, nil
		}
	}
	return home, true, nil
}
