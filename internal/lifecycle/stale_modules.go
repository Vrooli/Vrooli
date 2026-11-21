package lifecycle

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type ownedInstancesKey struct{}

// withOwnedInstances marks registry instances that belong to the operation the
// context runs: a start's own starting instance, or the running instance a
// restart is replacing. They are not other occupants of the scenario.
func withOwnedInstances(ctx context.Context, ids ...string) context.Context {
	owned := map[string]bool{}
	if existing, ok := ctx.Value(ownedInstancesKey{}).(map[string]bool); ok {
		for id := range existing {
			owned[id] = true
		}
	}
	for _, id := range ids {
		if id != "" {
			owned[id] = true
		}
	}
	return context.WithValue(ctx, ownedInstancesKey{}, owned)
}

func ownedInstances(ctx context.Context) map[string]bool {
	owned, _ := ctx.Value(ownedInstancesKey{}).(map[string]bool)
	return owned
}

const staleModulesInfix = ".stale-"

// setAsideStaleModules moves a consumer's own node_modules out of the way when
// it was linked against a pnpm store layout this pnpm cannot select, so the
// install that follows builds it afresh. pnpm would purge it itself, but only
// with a terminal or silently under CI=true; neither is acceptable while
// something may be serving from the directory.
//
// It is allowed only for a setup step; when every active registry instance of
// the scenario belongs to the operation the context runs (a start's own
// instance, or the instance a restart is replacing) and none belongs to anyone
// else; and for a real node_modules directory (not a symlink) directly under the
// step's working directory, which the step declaration already confines to the
// scenario. A registry that cannot be read is not evidence the scenario is
// stopped, so it refuses.
//
// The old directory is renamed, not deleted, so a failed install puts it back:
// settle(true) deletes it after a successful install, settle(false) restores it.
// A node process that is already serving has loaded its modules, and the
// replacement has the same locked versions, so the window is the install.
func (r *Runner) setAsideStaleModules(ctx context.Context, item scenario.Scenario, phase string, mismatch *packagegov.PnpmStoreMismatchError, logWriter io.Writer) (settle func(installed bool), err error) {
	if phase != phasesSetup {
		return nil, fmt.Errorf("node_modules is only recreated during setup, not %s", phase)
	}
	modules := filepath.Join(mismatch.Dir, "node_modules")
	info, err := os.Lstat(modules)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory (it is %s), so it is not recreated", modules, info.Mode().Type())
	}
	instances, err := r.activeInstances(ctx, item.Slug)
	if err != nil {
		return nil, fmt.Errorf("could not confirm scenario %s has no other occupant: %w", item.Slug, err)
	}
	owned := ownedInstances(ctx)
	var others []string
	replacing := false
	for _, instance := range instances {
		if owned[instance.InstanceID] {
			replacing = replacing || instance.Status == scenarioruntime.StatusRunning
			continue
		}
		variant := instance.Variant
		if variant == "" {
			variant = "live"
		}
		others = append(others, instance.Scenario+"@"+variant)
	}
	if len(others) > 0 {
		return nil, fmt.Errorf("scenario %s is occupied by %s that this operation does not own, so its node_modules is not recreated", item.Slug, strings.Join(others, ", "))
	}
	if err := removeAbandonedStaleModules(mismatch.Dir); err != nil {
		return nil, err
	}
	aside := fmt.Sprintf("%s%s%d", modules, staleModulesInfix, time.Now().UnixNano())
	if err := os.Rename(modules, aside); err != nil {
		return nil, fmt.Errorf("set aside %s: %w", modules, err)
	}
	state := "stopped"
	if replacing {
		state = "serving and being replaced by this restart"
	}
	fmt.Fprintf(logWriter, "lifecycle: scenario %s is %s and %s was linked against pnpm store %s, which this pnpm cannot select (it selects %s); set it aside so the install below recreates it, and it is restored if the install fails\n", item.Slug, state, modules, mismatch.Recorded, mismatch.Effective)
	return func(installed bool) {
		if installed {
			_ = os.RemoveAll(aside)
			return
		}
		// A failed install may have left a partial node_modules; the working
		// legacy one replaces it.
		_ = os.RemoveAll(modules)
		if err := os.Rename(aside, modules); err != nil {
			fmt.Fprintf(logWriter, "lifecycle: could not restore %s from %s: %v\n", modules, aside, err)
			return
		}
		fmt.Fprintf(logWriter, "lifecycle: install failed; restored the previous %s\n", modules)
	}, nil
}

// removeAbandonedStaleModules removes directories an interrupted earlier repair
// left beside node_modules. Only the exact prefix this repair writes is touched.
func removeAbandonedStaleModules(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "node_modules"+staleModulesInfix) {
			if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
				return fmt.Errorf("remove abandoned %s: %w", entry.Name(), err)
			}
		}
	}
	return nil
}
