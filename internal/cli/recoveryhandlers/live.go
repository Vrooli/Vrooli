package recoveryhandlers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	recoveryapp "github.com/vrooli/vrooli/internal/app/recovery"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// liveControl is the production recovery.LiveControl: the runtime registry
// says where instances run from, and the scenario lifecycle restarts or
// stops them.
type liveControl[C any] struct {
	deps rootcli.HandlerDeps[C]
	ctx  C
}

func (l liveControl[C]) operationContext() context.Context {
	return rootcli.ResolveOperationContext(l.deps, l.ctx)
}

func (l liveControl[C]) Instances(scenario string) ([]recoveryapp.ServingInstance, error) {
	home := ""
	if l.deps.HomeDir != nil {
		resolved, err := l.deps.HomeDir(l.ctx)
		if err != nil {
			return nil, err
		}
		home = resolved
	}
	dbPath, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dbPath); errors.Is(err, fs.ErrNotExist) {
		return nil, nil // no registry yet: nothing has ever run
	}
	ctx := l.operationContext()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer store.Close()
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: scenario,
		Statuses: append(scenarioruntime.ActiveInstanceStatuses(), scenarioruntime.StatusStopping),
	})
	if err != nil {
		return nil, err
	}
	out := make([]recoveryapp.ServingInstance, 0, len(instances))
	for _, instance := range instances {
		if strings.TrimSpace(instance.WorkingDir) == "" {
			continue
		}
		out = append(out, recoveryapp.ServingInstance{
			Key:        recoveryapp.InstanceKey(instance.Scenario, instance.Variant),
			WorkingDir: instance.WorkingDir,
		})
	}
	return out, nil
}

func (l liveControl[C]) Restart(instanceKey string) error {
	if l.deps.ScenarioOperations == nil {
		return fmt.Errorf("scenario lifecycle is unavailable to restart %s", instanceKey)
	}
	ops, err := l.deps.ScenarioOperations(l.ctx)
	if err != nil {
		return err
	}
	_, err = ops.RestartDetailed(instanceKey, lifecycle.StartOptions{Context: l.operationContext()})
	return err
}

func (l liveControl[C]) Stop(instanceKey string) error {
	if l.deps.LifecycleRunner == nil {
		return fmt.Errorf("scenario lifecycle is unavailable to stop %s", instanceKey)
	}
	runner, err := l.deps.LifecycleRunner(l.ctx)
	if err != nil {
		return err
	}
	return runner.Stop(instanceKey, lifecycle.StopOptions{Context: l.operationContext()})
}
