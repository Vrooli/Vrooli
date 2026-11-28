package page

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
	pageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page/page_v1connect"
	"react-component-library/cli/internal/runtimeenv"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := pageconnect.NewPageServiceClient(httpClient, baseURL)
	call := func(ctx cliapp.OperationContext) (*pagev1.InspectResponse, error) {
		result, err := client.Inspect(context.Background(), connect.NewRequest(&pagev1.InspectRequest{Scenario: ctx.Positional("scenario"), Route: ctx.Positional("route"), WaitSelector: ctx.Flag("wait-selector")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("inspect page", err, nil)
		}
		return result.Msg, nil
	}
	deps := runtimeenv.System()
	captures := captureHandlers{
		bas:  func(ctx context.Context) (basClient, error) { return newBASClient(ctx, deps.Environment) },
		deps: deps,
	}
	return cliapp.LoadFromManifestPrimitives(manifest, "page", map[string]cliapp.PrimitiveHandler{
		"PageService.Inspect": cliapp.ProtoList(call, inspectReport),
		"capture":             {Run: captures.capture},
		"render":              {Run: captures.render},
	})
}

func inspectReport(_ cliapp.OperationContext, result *pagev1.InspectResponse) cliapp.ListReport {
	rows := []string{}
	if result.ScreenshotPath != "" {
		rows = append(rows, "Screenshot: "+result.ScreenshotPath)
	}
	if result.ScreenshotUrl != "" {
		rows = append(rows, "Open: "+result.ScreenshotUrl)
	}
	seen := map[string]bool{}
	var walk func(*pagev1.PageNode)
	walk = func(node *pagev1.PageNode) {
		if node == nil {
			return
		}
		source := node.Source
		if source != nil && source.Status != "unstamped" {
			key := source.StampedAsset + "@" + source.StampedVersion
			if !seen[key] {
				seen[key] = true
				detail := source.Reason
				if source.Status == "resolved" {
					detail = source.SourcePath + " (" + source.ResolutionRule + ")"
				}
				rows = append(rows, key+" → "+detail)
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(result.Tree)
	rows = append(rows, result.Warnings...)
	return cliapp.ListReport{Summary: []string{result.Url, fmt.Sprintf("%d nodes; %d stamped; %d sources resolved; %d unstamped.", result.NodeCount, result.StampedCount, result.ResolvedCount, result.NodeCount-result.StampedCount)}, ResultsHeading: "Observation", Results: rows}
}
