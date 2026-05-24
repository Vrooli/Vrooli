package graph

import (
	"context"
	"fmt"
	"sort"

	"connectrpc.com/connect"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client graphconnect.GoCodeGraphServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: graphconnect.NewGoCodeGraphServiceClient(httpClient, baseURL),
	}
}

// extract calls the generated Connect-RPC GoCodeGraphService.Extract
// method. Output routing: human consumers see a ListReport summarizing
// node/edge counts and top-level packages; --json consumers see the
// proto-typed ExtractResponse wire shape.
func (h *handlers) extract(ctx cliapp.RunContext) error {
	path := ctx.Positional("path")
	includeVendor := ctx.BoolFlag("include-vendor")

	resp, err := h.client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{
		ScenarioPath:  path,
		IncludeVendor: includeVendor,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("extract %q", path), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no extract response")
	}

	nodes := 0
	edges := 0
	var packages []string
	if g := resp.Msg.GetGraph(); g != nil {
		nodes = len(g.GetNodes())
		edges = len(g.GetEdges())
		for _, n := range g.GetNodes() {
			if n.GetKind() == commonv1.NodeKind_NODE_KIND_PACKAGE {
				packages = append(packages, n.GetName())
			}
		}
	}
	sort.Strings(packages)
	if len(packages) > 10 {
		packages = packages[:10]
	}

	summary := []string{
		fmt.Sprintf(
			"Extracted %d node(s), %d edge(s) in %dms (hash=%s).",
			nodes, edges, resp.Msg.GetExtractionMs(), resp.Msg.GetGraphHash(),
		),
	}
	if w := len(resp.Msg.GetWarnings()); w > 0 {
		summary = append(summary, fmt.Sprintf("%d warning(s) returned.", w))
	}

	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Top-level packages",
		Results:        packages,
		RetrievalHints: []string{
			fmt.Sprintf("`rewrite plan <ops.json> --scenario-path %s` — plan a rewrite for this module", path),
		},
	})
}
