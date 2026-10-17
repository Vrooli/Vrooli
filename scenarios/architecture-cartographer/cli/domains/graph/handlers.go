package graph

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client graphconnect.GraphServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: graphconnect.NewGraphServiceClient(httpClient, baseURL),
	}
}

// extract calls the generated Connect-RPC GraphService.ExtractGraph
// method. Human consumers see a ListReport summarizing snapshot
// counts and top packages; --json consumers see the proto-typed
// ExtractGraphResponse wire shape.
func (h *handlers) extract(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	idempotencyKey := ctx.Flag("idempotency-key")

	resp, err := h.client.ExtractGraph(context.Background(), connect.NewRequest(&graphv1.ExtractGraphRequest{
		Scenario:       scenario,
		IdempotencyKey: idempotencyKey,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("extract %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no extract response")
	}

	snap := resp.Msg.GetSnapshot()
	var (
		files, packages, symbols, imports int
		topPackages                       []string
	)
	if snap != nil {
		files = len(snap.GetFiles())
		packages = len(snap.GetPackages())
		symbols = len(snap.GetSymbols())
		imports = len(snap.GetImports())
		for i, p := range snap.GetPackages() {
			if i >= 10 {
				break
			}
			topPackages = append(topPackages, p.GetImportPath())
		}
	}

	cacheTag := "fresh"
	if resp.Msg.GetFromCache() {
		cacheTag = "cache-hit"
	}
	summary := []string{
		fmt.Sprintf(
			"Snapshot %s (%s, %dms): %d file(s), %d package(s), %d symbol(s), %d import(s).",
			snap.GetId(), cacheTag, snap.GetExtractionMs(),
			files, packages, symbols, imports,
		),
	}

	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Top packages",
		Results:        topPackages,
		RetrievalHints: []string{
			fmt.Sprintf("`graph extract %s` again returns the cached snapshot when nothing changed.", scenario),
		},
	})
}
