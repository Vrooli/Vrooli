package resourcehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/resources"
)

func resourceInspectDataHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		parsed, err := commandtree.ParseArgs("resource acquisition inspect-data", resourcecli.AcquisitionCommandHelpText(resourcecli.AcquisitionCommandInspectData), resourcecli.AcquisitionInspectDataArgs(), args)
		if err != nil {
			if commandtree.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return err
		}
		for _, flag := range []string{"--url", "--git-blob-sha1", "--max-bytes", "--out", "--logical-id", "--field"} {
			if parsed.FlagValue(flag) == "" {
				return fmt.Errorf("resource acquisition inspect-data requires %s", flag)
			}
		}
		maxBytes, err := strconv.ParseInt(parsed.FlagValue("--max-bytes"), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid max-bytes")
		}
		operationCtx, cancel := context.WithTimeout(rootcli.ResolveOperationContext(deps, ctx), 2*time.Minute)
		defer cancel()
		result, err := controller.InspectData(operationCtx, parsed.Positionals[0], resources.ResourceDataInspection{URL: parsed.FlagValue("--url"), GitBlobSHA1: parsed.FlagValue("--git-blob-sha1"), MaxBytes: maxBytes, Destination: parsed.FlagValue("--out"), LogicalID: parsed.FlagValue("--logical-id"), Field: parsed.FlagValue("--field")})
		if err != nil {
			return err
		}
		return json.NewEncoder(deps.Stdout(ctx)).Encode(result)
	}
}
