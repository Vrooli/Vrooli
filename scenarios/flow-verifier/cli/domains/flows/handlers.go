package flows

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows"
	flowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows/flows_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client flowsconnect.FlowsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: flowsconnect.NewFlowsServiceClient(httpClient, baseURL)}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListFlows(context.Background(), connect.NewRequest(&flowsv1.ListFlowsRequest{Root: ctx.Flag("root")}))
	if err != nil {
		return cliapp.WrapAPIError("list flows", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Flows))
	for _, f := range resp.Msg.Flows {
		results = append(results, formatSummary(f))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d flow(s).", len(resp.Msg.Flows))},
		ResultsHeading: "Flows",
		Results:        results,
		RetrievalHints: []string{
			"`flows show --flow <id>` — show the typed flow.json projection",
			"`flows explain --flow <id>` — print the human-readable explain report",
		},
	})
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	resp, err := h.client.ValidateFlow(context.Background(), connect.NewRequest(&flowsv1.ValidateFlowRequest{
		Root:   ctx.Flag("root"),
		FlowId: ctx.Flag("flow"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate flows", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Flows))
	for _, f := range resp.Msg.Flows {
		results = append(results, formatSummary(f))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Validated %d flow(s) successfully.", len(resp.Msg.Flows))},
		ResultsHeading: "Validated",
		Results:        results,
	})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	lang := ctx.Flag("lang")
	if lang == "" {
		lang = "typescript"
	}
	resp, err := h.client.CreateFlow(context.Background(), connect.NewRequest(&flowsv1.CreateFlowRequest{
		ParentDir: ctx.Positional("feature-dir"),
		FlowId:    ctx.Flag("flow-id"),
		Language:  lang,
		Root:      ctx.Flag("root"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("create flow", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created flow directory %s.", resp.Msg.FlowDir)},
		Changes: []string{resp.Msg.FlowDir},
	})
}

func (h *handlers) explain(ctx cliapp.RunContext) error {
	resp, err := h.client.ExplainFlow(context.Background(), connect.NewRequest(&flowsv1.ExplainFlowRequest{
		Root:   ctx.Flag("root"),
		FlowId: ctx.Flag("flow"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("explain flow", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Explain report for %s", ctx.Flag("flow"))},
		ResultsHeading: "Report",
		Results:        []string{resp.Msg.Report},
	})
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	resp, err := h.client.GetFlow(context.Background(), connect.NewRequest(&flowsv1.GetFlowRequest{
		Root:   ctx.Flag("root"),
		FlowId: ctx.Flag("flow"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("show flow", err, nil)
	}
	d := resp.Msg.Flow
	if d == nil {
		return fmt.Errorf("server returned no flow")
	}
	results := []string{
		fmt.Sprintf("flowId       = %s", d.FlowId),
		fmt.Sprintf("language     = %s", d.Language),
		fmt.Sprintf("domain       = %s", d.Domain),
		fmt.Sprintf("description  = %s", d.Description),
		fmt.Sprintf("states       = %d  events = %d  transitions = %d  traces = %d  invariants = %d",
			len(d.States), len(d.Events), len(d.Transitions), len(d.Traces), len(d.Invariants)),
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Flow %s", d.FlowId)},
		ResultsHeading: "Detail",
		Results:        results,
	})
}

func formatSummary(f *flowsv1.FlowSummary) string {
	if f == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s | %s | v%d | %s", f.FlowId, f.Language, f.SchemaVersion, f.ContractPath)
}
