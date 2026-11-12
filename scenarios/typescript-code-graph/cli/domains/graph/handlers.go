package graph

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/graphreport"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client graphconnect.TypeScriptCodeGraphServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: graphconnect.NewTypeScriptCodeGraphServiceClient(httpClient, baseURL),
	}
}

// extract calls the generated Connect-RPC TypeScriptCodeGraphService.Extract
// method. Output routing: human consumers see a ListReport summarizing
// node/edge/warning counts plus a short top-level package list; --json
// consumers see the proto-typed ExtractResponse wire shape.
func (h *handlers) extract(ctx cliapp.RunContext) error {
	path := ctx.Positional("path")

	resp, err := h.client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{
		ProjectPath: path,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("extract %q", path), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no extract response")
	}

	var packages []string
	if g := resp.Msg.GetGraph(); g != nil {
		for _, node := range g.GetNodes() {
			if node.GetKind() == commonv1.NodeKind_NODE_KIND_PACKAGE {
				packages = append(packages, node.GetName())
			}
		}
	}
	return graphreport.RenderExtraction(ctx, resp.Msg, graphreport.Extraction{
		Nodes: len(resp.Msg.GetGraph().GetNodes()), Edges: len(resp.Msg.GetGraph().GetEdges()),
		Warnings: len(resp.Msg.GetWarnings()), ElapsedMS: int(resp.Msg.GetExtractionMs()), GraphHash: resp.Msg.GetGraphHash(),
		Path: path, RewriteFlag: "project-path", RewriteTarget: "project", Packages: packages, ShowWarningsAlways: true,
	})
}

// listFixtures calls TypeScriptCodeGraphService.ListFixtures and renders the
// golden determinism fixtures the server ships (the browser/CLI can't read
// bas/fixtures/* directly, so this RPC is the only path to the list).
func (h *handlers) listFixtures(ctx cliapp.RunContext) error {
	resp, err := h.client.ListFixtures(context.Background(), connect.NewRequest(&graphv1.ListFixturesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list fixtures", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no list-fixtures response")
	}

	fixtures := make([]graphreport.Fixture, 0, len(resp.Msg.GetFixtures()))
	for _, fixture := range resp.Msg.GetFixtures() {
		fixtures = append(fixtures, graphreport.Fixture{Name: fixture.GetName(), Path: fixture.GetPath(), HasExpected: fixture.GetHasExpected()})
	}
	return graphreport.RenderFixtures(ctx, resp.Msg, fixtures)
}

// validateFixture calls TypeScriptCodeGraphService.ValidateFixture, which
// re-extracts a named fixture server-side and byte-compares the canonical JSON
// against its expected-graph.json. A failing comparison exits non-zero so the
// command is usable as a determinism gate in scripts.
func (h *handlers) validateFixture(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")

	resp, err := h.client.ValidateFixture(context.Background(), connect.NewRequest(&graphv1.ValidateFixtureRequest{
		Name: name,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate fixture %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validate-fixture response")
	}

	return graphreport.RenderValidation(ctx, resp.Msg, graphreport.Validation{
		Name: name, Passed: resp.Msg.GetPassed(), Diff: resp.Msg.GetDiff(),
		ExpectedBytes: resp.Msg.GetExpectedBytes(), ActualBytes: resp.Msg.GetActualBytes(), GraphHash: resp.Msg.GetGraphHash(),
	})
}
