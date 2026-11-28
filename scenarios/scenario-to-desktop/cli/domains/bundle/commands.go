// Package bundle provides bundle-related CLI commands.
package bundle

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
)

// Commands provides bundle CLI commands.
type Commands struct {
	rpc pipelineRPC
}

// New creates a new bundle Commands instance.
func New(deps support.Dependencies) *Commands {
	return &Commands{rpc: newPipelineRPC(deps.ScenarioApp())}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{
		Name:        "bundle",
		Description: "Bundle output utilities (run 'bundle help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			(cliapp.Command{Name: "clean", Description: "Clean bundle output directory: clean <scenario> [--location-mode ...]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "location-mode", Default: "proper", Values: []string{"proper", "staging", "temp"}}, {Name: "pipeline-id"}}}}).WithPrimitive(cmds.cleanPrimitive()),
		},
	}
}

func (c *Commands) cleanPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*pipelinev1.BundleCleanResponse, error) {
		locationMode := strings.TrimSpace(ctx.Flag("location-mode"))
		pipelineID := strings.TrimSpace(ctx.Flag("pipeline-id"))
		if (locationMode == "staging" || locationMode == "temp") && pipelineID == "" {
			return nil, fmt.Errorf("--pipeline-id is required when --location-mode is staging/temp")
		}
		request := &pipelinev1.BundleCleanRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}
		if locationMode != "" {
			request.LocationMode = &locationMode
		}
		if pipelineID != "" {
			request.PipelineId = &pipelineID
		}
		response, err := c.rpc.CleanBundle(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("clean bundle", err, nil)
		}
		if response.Msg.GetPath() == "" {
			return nil, fmt.Errorf("clean bundle response did not include a path")
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.BundleCleanResponse) cliapp.MutationReport {
		result := "Bundle already clean: " + response.GetPath()
		if response.GetRemoved() {
			result = "Bundle cleaned: " + response.GetPath()
		}
		return cliapp.MutationReport{Result: []string{result}}
	})
}
