package scenariohandlers

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/runtimeprofile"
)

func profileHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		request, err := ParseProfileRequest(deps.Globals(ctx).JSON, args)
		if err != nil {
			if commandtree.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return err
		}
		home, err := deps.HomeDir(ctx)
		if err != nil {
			return err
		}
		result, err := runtimeprofile.Capture(rootcli.ResolveOperationContext(deps, ctx), home, request.Request)
		if err != nil {
			return err
		}
		if request.JSON {
			return cliout.WriteJSONValue(deps.Stdout(ctx), result)
		}
		_, err = fmt.Fprintf(deps.Stdout(ctx), "CPU profile: %s (%d bytes, PID %d, %ds)\nInspect: go tool pprof -top %s\n", result.Path, result.Bytes, result.Identity.PID, result.Seconds, result.Path)
		return err
	}
}
