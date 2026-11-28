package scenariohandlers

import (
	"context"
	"fmt"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle/retainedruntime"
)

type runtimeReleaseRunner interface {
	CaptureRuntimeRelease(context.Context, string, string) (retainedruntime.Release, error)
	VerifyRuntimeRelease(context.Context, string, string) (retainedruntime.Release, error)
}

func buildRuntimeReleaseHandlers[C any](deps rootcli.HandlerDeps[C]) map[CommandID]rootcli.Handler[C] {
	result := make(map[CommandID]rootcli.Handler[C])
	for _, id := range []CommandID{CommandCaptureRuntime, CommandVerifyRuntime} {
		result[id] = func(ctx C, args []string) error {
			req, err := ParseRuntimeReleaseRequest(id, deps.Globals(ctx).JSON, args)
			if err != nil {
				if commandtree.HandleHelp(deps.Stdout(ctx), err) {
					return nil
				}
				return err
			}
			runner, err := deps.LifecycleRunner(ctx)
			if err != nil {
				return err
			}
			releases, ok := runner.(runtimeReleaseRunner)
			if !ok {
				return fmt.Errorf("lifecycle runner does not support retained runtime releases")
			}
			var release retainedruntime.Release
			if id == CommandCaptureRuntime {
				release, err = releases.CaptureRuntimeRelease(rootcli.ResolveOperationContext(deps, ctx), req.Name, req.CustomPath)
			} else {
				release, err = releases.VerifyRuntimeRelease(rootcli.ResolveOperationContext(deps, ctx), req.Name, req.ReleaseID)
			}
			if err != nil {
				return err
			}
			if req.JSON {
				return cliout.WriteJSONValue(deps.Stdout(ctx), release)
			}
			if id == CommandCaptureRuntime {
				_, err = fmt.Fprintf(deps.Stdout(ctx), "Captured runtime release %s for %s. No processes started or stopped; capture is not a health or data-compatibility proof.\n", release.ID, req.Name)
			} else {
				_, err = fmt.Fprintf(deps.Stdout(ctx), "Verified runtime release %s for %s. No processes started or stopped.\n", release.ID, req.Name)
			}
			return err
		}
	}
	return result
}

// Source CLI preparation follows the lifecycle owner's durable selection. It
// does not select, verify, or launch a runtime; those stay under the owner lock.
func ensureSourceScenarioCLIs[C any](deps rootcli.HandlerDeps[C], ctx C, requested string, names ...string) error {
	if deps.EnsureCLI == nil {
		return nil
	}
	if requested != "" && requested != "source" {
		return nil
	}
	if requested == "source" {
		return ensureScenarioCLIs(deps, ctx, names...)
	}
	source := make([]string, 0, len(names))
	for _, name := range names {
		selected := ""
		if deps.LifecycleRunner != nil {
			runner, err := deps.LifecycleRunner(ctx)
			if err != nil {
				return err
			}
			if reader, ok := runner.(interface {
				SelectedRuntimeReleaseID(context.Context, string) (string, error)
			}); ok {
				selected, err = reader.SelectedRuntimeReleaseID(rootcli.ResolveOperationContext(deps, ctx), name)
				if err != nil {
					return err
				}
			}
		}
		if selected == "" {
			source = append(source, name)
		}
	}
	return ensureScenarioCLIs(deps, ctx, source...)
}
