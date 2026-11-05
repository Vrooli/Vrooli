package resourcehandlers

import (
	"context"
	"io"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/packages/harness-lifecycle"
)

// harnessActionTimeout covers a governed npm install (the gateway allows three
// minutes), its approval, and the checks before and after it.
const harnessActionTimeout = 6 * time.Minute

func resourceUpdateHandler[C any](deps rootcli.HandlerDeps[C], parse func([]string) (resourcecli.UpdateRequest, error)) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		parseMapped := func(args []string) (resourcecli.UpdateRequest, error) {
			req, err := parse(args)
			return req, mapResourceParseError("resource update", err)
		}
		return rootcli.BindResourceCommand(deps.Stdout, deps.OutputFormat, parseMapped,
			func(ctx C, controller *resources.Controller, req resourcecli.UpdateRequest) (agentinstall.HarnessActionResult, error) {
				if _, err := cliutil.CodingAgentBinary(req.Name); err != nil {
					return agentinstall.HarnessActionResult{}, rootcli.UsageErrorf("resource update", "unknown coding-agent resource %q (known: %s)", req.Name, knownCodingAgentResourceNames())
				}
				operation, cancel := context.WithTimeout(rootcli.ResolveOperationContext(deps, ctx), harnessActionTimeout)
				defer cancel()
				return runHarnessUpdate(operation, controller.Root, req)
			},
			func(w io.Writer, _ cliout.Format, result agentinstall.HarnessActionResult) error {
				if err := resourcecli.WriteHarnessUpdate(w, format, result); err != nil {
					return err
				}
				if !result.Success {
					// The result is already rendered; only the exit status is left to say.
					return rootcli.ExitCodeError{Code: 1, Silent_: true}
				}
				return nil
			},
		)(ctx, controller, args)
	}
}

func runHarnessUpdate(ctx context.Context, root string, req resourcecli.UpdateRequest) (agentinstall.HarnessActionResult, error) {
	return agentinstall.ManageHarness(ctx, root, req.Name, req.Action, cliutil.ResolveCallerActor(req.OnBehalfOf), req.CleanupPlanID)
}
