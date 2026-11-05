package resourcehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/resources"
)

func resourceModelCacheHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		parsed, err := commandtree.ParseArgs("resource acquisition model-cache", resourcecli.AcquisitionCommandHelpText(resourcecli.AcquisitionCommandModelCache), resourcecli.AcquisitionModelCacheArgs(), args)
		if err != nil {
			if commandtree.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return err
		}
		entry := parsed.FlagValue("--entry")
		if entry == "" {
			entry = "models"
		}
		var pin *bool
		if parsed.HasFlag("--pin") {
			value, err := strconv.ParseBool(parsed.FlagValue("--pin"))
			if err != nil {
				return fmt.Errorf("--pin requires true or false")
			}
			pin = &value
		}
		var reviewed *resources.ModelCachePreview
		if file := parsed.FlagValue("--apply-preview"); file != "" {
			stream, err := os.Open(file)
			if err != nil {
				return err
			}
			defer stream.Close()
			data, err := io.ReadAll(io.LimitReader(stream, 65537))
			if err != nil {
				return err
			}
			if len(data) > 65536 {
				return fmt.Errorf("model-cache preview exceeds 64KiB")
			}
			var saved resources.ModelCacheResult
			if err := json.Unmarshal(data, &saved); err != nil {
				return err
			}
			reviewed = &saved.Preview
		}
		operationCtx, cancel := context.WithTimeout(rootcli.ResolveOperationContext(deps, ctx), 60*time.Second)
		defer cancel()
		result, err := controller.ModelCache(operationCtx, parsed.Positionals[0], entry, pin, reviewed)
		if err != nil {
			return err
		}
		return json.NewEncoder(deps.Stdout(ctx)).Encode(result)
	}
}
