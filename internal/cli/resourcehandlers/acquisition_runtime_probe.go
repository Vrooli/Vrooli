package resourcehandlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/resources"
)

func resourceRuntimeProbeHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		parsed, err := commandtree.ParseArgs("resource acquisition runtime-probe", resourcecli.AcquisitionCommandHelpText(resourcecli.AcquisitionCommandRuntimeProbe), resourcecli.AcquisitionRuntimeProbeArgs(), args)
		if err != nil {
			if commandtree.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return err
		}
		if parsed.FlagValue("--imports") == "" || parsed.FlagValue("--out") == "" {
			return rootcli.UsageErrorf("resource acquisition runtime-probe", "requires --imports and --out")
		}
		operationCtx, cancel := context.WithTimeout(rootcli.ResolveOperationContext(deps, ctx), time.Minute)
		defer cancel()
		receipt, err := controller.ProbeManagedRuntime(operationCtx, parsed.Positionals[0], strings.Split(parsed.FlagValue("--imports"), ","), parsed.FlagValue("--out"))
		if receipt.Status != "" {
			if writeErr := json.NewEncoder(deps.Stdout(ctx)).Encode(receipt); writeErr != nil {
				return errors.Join(err, writeErr)
			}
		}
		return err
	}
}
