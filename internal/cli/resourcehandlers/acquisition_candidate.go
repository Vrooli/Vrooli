package resourcehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/resources"
)

var composeCandidateRun = func(ctx context.Context, c *resources.Controller, name, output string) (resources.ComposeCandidateReceipt, error) {
	return c.ComposeCandidate(ctx, name, output)
}

func resourceComposeCandidateHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		parsed, err := commandtree.ParseArgs("resource acquisition compose-candidate", resourcecli.AcquisitionCommandHelpText(resourcecli.AcquisitionCommandComposeCandidate), resourcecli.AcquisitionComposeCandidateArgs(), args)
		if err != nil {
			if commandtree.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return err
		}
		if parsed.FlagValue("--out") == "" {
			return fmt.Errorf("resource acquisition compose-candidate requires --out")
		}
		operationCtx, cancel := context.WithTimeout(rootcli.ResolveOperationContext(deps, ctx), 15*time.Minute)
		defer cancel()
		if err := operationCtx.Err(); err != nil {
			return err
		}
		if parsed.HasFlag("--offline") {
			operationCtx = resources.WithOfflineCompose(operationCtx)
		}
		receipt, err := composeCandidateRun(operationCtx, controller, parsed.Positionals[0], parsed.FlagValue("--out"))
		if err != nil {
			return err
		}
		return json.NewEncoder(deps.Stdout(ctx)).Encode(receipt)
	}
}
