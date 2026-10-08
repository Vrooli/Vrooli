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
	resp, err := h.client.ListFlows(context.Background(), connect.NewRequest(&flowsv1.ListFlowsRequest{
		Root: ctx.Flag("root"),
		Kind: ctx.Flag("kind"),
	}))
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
	kindName := ctx.Flag("kind")
	lang := ctx.Flag("lang")
	if kindName == "" || kindName == "temporal" {
		if lang == "" {
			lang = "typescript"
		}
	}
	resp, err := h.client.CreateFlow(context.Background(), connect.NewRequest(&flowsv1.CreateFlowRequest{
		ParentDir: ctx.Positional("feature-dir"),
		FlowId:    ctx.Flag("flow-id"),
		Kind:      kindName,
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

func (h *handlers) codegen(ctx cliapp.RunContext) error {
	write := ctx.Flag("write") == "true"
	resp, err := h.client.CodegenFlow(context.Background(), connect.NewRequest(&flowsv1.CodegenFlowRequest{
		Root:     ctx.Flag("root"),
		FlowId:   ctx.Flag("flow"),
		Language: ctx.Flag("lang"),
		Write:    write,
	}))
	if err != nil {
		return cliapp.WrapAPIError("codegen flow", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Artifacts)+len(resp.Msg.Written))
	for _, a := range resp.Msg.Artifacts {
		results = append(results, fmt.Sprintf("artifact: %s (%d bytes)", a.Path, len(a.Content)))
	}
	for _, w := range resp.Msg.Written {
		results = append(results, fmt.Sprintf("written:  %s", w))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Codegen %s — %d artifact(s), %d written.", ctx.Flag("flow"), len(resp.Msg.Artifacts), len(resp.Msg.Written))},
		ResultsHeading: "Artifacts",
		Results:        results,
	})
}

func (h *handlers) reconcile(ctx cliapp.RunContext) error {
	resp, err := h.client.ReconcileFlow(context.Background(), connect.NewRequest(&flowsv1.ReconcileFlowRequest{
		Root:         ctx.Flag("root"),
		FlowId:       ctx.Flag("flow"),
		ScenarioRoot: ctx.Flag("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("reconcile flow", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Findings))
	for _, f := range resp.Msg.Findings {
		loc := ""
		if f.SourceFile != "" {
			loc = fmt.Sprintf(" (%s:%d)", f.SourceFile, f.SourceLine)
		}
		results = append(results, fmt.Sprintf("[%s] %s%s", f.Severity, f.Message, loc))
	}
	status := "PASSED"
	if !resp.Msg.Passed {
		status = "FAILED"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Reconcile %s — %s (%d file(s) scanned, %d finding(s)).", ctx.Flag("flow"), status, resp.Msg.FilesScanned, len(resp.Msg.Findings))},
		ResultsHeading: "Findings",
		Results:        results,
	})
}

func (h *handlers) studio(ctx cliapp.RunContext) error {
	resp, err := h.client.GetNavigationStudio(context.Background(), connect.NewRequest(&flowsv1.GetNavigationStudioRequest{
		Root:   ctx.Flag("root"),
		FlowId: ctx.Flag("flow"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("navigation studio", err, nil)
	}
	d := resp.Msg.GetDescriptor_()
	if d == nil {
		return fmt.Errorf("server returned no descriptor")
	}
	results := []string{
		fmt.Sprintf("renderer    = %s", d.Renderer),
		fmt.Sprintf("routes      = %d", len(d.Routes)),
		fmt.Sprintf("affordances = %d", len(d.Affordances)),
		fmt.Sprintf("containers  = %d", len(d.Containers)),
		fmt.Sprintf("contexts    = %d", len(d.Contexts)),
		fmt.Sprintf("invariants  = %d", len(d.Invariants)),
	}
	for _, inv := range d.Invariants {
		status := "PASS"
		if !inv.Passed {
			status = "FAIL"
		}
		results = append(results, fmt.Sprintf("  [%s] %s — %s", status, inv.Id, inv.Message))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Studio descriptor for %s", ctx.Flag("flow"))},
		ResultsHeading: "Descriptor",
		Results:        results,
	})
}

func formatSummary(f *flowsv1.FlowSummary) string {
	if f == nil {
		return "(nil)"
	}
	lang := f.Language
	if lang == "" {
		lang = "-"
	}
	return fmt.Sprintf("%s | %s | %s | v%d | %s", f.FlowId, f.Kind, lang, f.SchemaVersion, f.ContractPath)
}
