package buildinfo

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
)

// GuardRootActivation keeps ordinary installers and self-rebuilders from
// crossing an admitted root operation. Callers already own the binary lock.
// Scratch and scenario binaries do not mutate the canonical serving image.
func GuardRootActivation(ctx context.Context, executable string) error {
	home, canonical, err := canonicalRootTarget(executable)
	if err != nil || !canonical {
		return err
	}
	// A rebaseline owns this lock across its multi-owner transition. Do not
	// wait while holding the install lock: the transition may need that lock.
	if err := GuardRebaselineTransition(ctx, home); err != nil {
		return err
	}
	if err := GuardRebaselineCompletion(home); err != nil {
		return err
	}
	if err := GuardPreparedBundle(home, ""); err != nil {
		return err
	}
	return GuardRootActivationOperation(ctx, executable)
}

// GuardRootActivationForPrepared allows only the retained root of an unfinished
// ordinary bundle through the same transition and operation guards.
func GuardRootActivationForPrepared(ctx context.Context, executable, candidate string) error {
	home, canonical, err := canonicalRootTarget(executable)
	if err != nil || !canonical {
		return err
	}
	if err := GuardRebaselineTransition(ctx, home); err != nil {
		return err
	}
	if err := GuardRebaselineCompletion(home); err != nil {
		return err
	}
	if err := GuardPreparedBundle(home, candidate); err != nil {
		return err
	}
	return GuardRootActivationOperation(ctx, executable)
}

// GuardRebaselineCompletion is called while the transition or root install
// lock is held, so a transition cannot publish across this check.
func GuardRebaselineCompletion(home string) error {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return err
	}
	progress, err := rebaselineprogress.Read(filepath.Join(state, "root-rebaseline", "progress.json"))
	if err != nil {
		return err
	}
	if progress != nil && !progress.Complete {
		if digest, err := hex.DecodeString(progress.Bundle); err == nil && len(digest) == 32 && hex.EncodeToString(digest) == progress.Bundle {
			candidate := filepath.Join(state, "root-rebaseline", "prepared-bundles", progress.Bundle, "vrooli")
			quoted := "'" + strings.ReplaceAll(candidate, "'", "'\\''") + "'"
			return fmt.Errorf("a rebaseline is incomplete; resume it before changing the root image: %s runtime update rebaseline %s", quoted, quoted)
		}
		return fmt.Errorf("a rebaseline is incomplete; resume it before changing the root image")
	}
	return nil
}

// GuardRebaselineTransition fails rather than waiting while a caller holds an
// owner lock needed by a rebaseline. The caller retains that owner lock after
// this check so a transition that starts immediately afterwards must wait.
func GuardRebaselineTransition(ctx context.Context, home string) error {
	check, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	release, err := AcquireRebaselineTransition(check, home)
	if err != nil {
		return fmt.Errorf("root rebaseline transition is busy or unavailable: %w", err)
	}
	release()
	return nil
}

// AcquireRebaselineTransition serializes a full admission or rebaseline with
// other transitions. Acquire it before any operation, publication or install
// lock; owner-lock holders must instead use GuardRebaselineTransition.
func AcquireRebaselineTransition(ctx context.Context, home string) (func(), error) {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, err
	}
	return artifactlock.AcquireWithPathContext(ctx, "root-rebaseline:"+home, func(string) (string, error) { return filepath.Join(state, "root-rebaseline", "transition.lock"), nil })
}

// AcquireRootBundleWorkflow serializes root-and-companion publication. It is
// outside the rebaseline/activation locks so the watchdog can acquire those
// locks while an enrolled ordinary installation waits for activation.
func AcquireRootBundleWorkflow(ctx context.Context, home string) (func(), error) {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, err
	}
	return artifactlock.AcquireWithPathContext(ctx, "root-bundle:"+home, func(string) (string, error) {
		return filepath.Join(state, "root-rebaseline", "bundle-workflow.lock"), nil
	})
}

// AcquireRootBundleWorkflowForTarget serializes only canonical root bundles;
// unrelated scratch installations do not share the operator workflow.
func AcquireRootBundleWorkflowForTarget(ctx context.Context, executable string) (func(), error) {
	home, canonical, err := canonicalRootTarget(executable)
	if err != nil {
		return nil, err
	}
	if !canonical {
		return func() {}, nil
	}
	return AcquireRootBundleWorkflow(ctx, home)
}

// GuardRootActivationOperation checks admitted activation state without the
// transition lock. Only the rebaseline owner may bypass the transition check.
func GuardRootActivationOperation(ctx context.Context, executable string) error {
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
