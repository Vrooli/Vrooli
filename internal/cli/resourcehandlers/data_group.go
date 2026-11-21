package resourcehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/resources"
	"time"
)

func resourceDataGroupHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		parsed, err := commandtree.ParseArgs("resource acquisition data", resourcecli.AcquisitionCommandHelpText(resourcecli.AcquisitionCommandDataGroup), resourcecli.AcquisitionDataGroupArgs(), args)
		if err != nil {
			if commandtree.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return err
		}
		group := parsed.FlagValue("--group")
		if group == "" {
			return fmt.Errorf("resource acquisition data requires --group")
		}
		operation, cancel := context.WithTimeout(rootcli.ResolveOperationContext(deps, ctx), 30*time.Minute)
		defer cancel()
		count, err := controller.AcquireManagedServiceDataGroup(operation, parsed.Positionals[0], group)
		if err != nil {
			return err
		}
		return json.NewEncoder(deps.Stdout(ctx)).Encode(map[string]any{"resource": parsed.Positionals[0], "group": group, "status": "verified", "artifacts": count})
	}
}
