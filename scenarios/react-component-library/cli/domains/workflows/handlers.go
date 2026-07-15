package workflows

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	workflowspb "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows"
	workflowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows/workflows_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client workflowsconnect.WorkflowsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: workflowsconnect.NewWorkflowsServiceClient(httpClient, baseURL)}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	kind, err := parseKind(ctx.Flag("kind"))
	if err != nil {
		return err
	}
	resp, err := h.client.StartWorkflow(context.Background(), connect.NewRequest(&workflowspb.StartWorkflowRequest{
		Kind: kind, AssetId: ctx.Flag("asset-id"), SourceScenario: ctx.Flag("source-scenario"), TargetScenario: ctx.Flag("target-scenario"), SourcePath: ctx.Flag("source-path"), RequestedVersion: ctx.Flag("version"), IdempotencyKey: ctx.Flag("idempotency-key"), ConfirmOverwrite: ctx.Flag("confirm-overwrite") == "true", OverrideValidation: ctx.Flag("override-validation") == "true",
	}))
	if err != nil {
		return cliapp.WrapAPIError("start assisted workflow", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Workflow == nil {
		return fmt.Errorf("server returned no workflow")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Workflow queued with depth %d.", resp.Msg.QueueDepth)}, ResultsHeading: "Workflow", Results: []string{format(resp.Msg.Workflow)}})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	req := &workflowspb.ListWorkflowsRequest{AssetId: ctx.Flag("asset-id"), TargetScenario: ctx.Flag("target-scenario"), ActiveOnly: ctx.Flag("active") == "true"}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListWorkflows(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list workflows", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no workflow list")
	}
	rows := make([]string, 0, len(resp.Msg.Workflows))
	for _, w := range resp.Msg.Workflows {
		rows = append(rows, format(w))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d workflow(s).", len(rows))}, ResultsHeading: "Workflows", Results: rows})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetWorkflow(context.Background(), connect.NewRequest(&workflowspb.GetWorkflowRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("get workflow", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Workflow == nil {
		return fmt.Errorf("server returned no workflow")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{ResultsHeading: "Workflow", Results: []string{format(resp.Msg.Workflow)}})
}
func (h *handlers) refresh(ctx cliapp.RunContext) error {
	resp, err := h.client.RefreshWorkflow(context.Background(), connect.NewRequest(&workflowspb.RefreshWorkflowRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("refresh workflow", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Workflow == nil {
		return fmt.Errorf("server returned no workflow")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{ResultsHeading: "Workflow", Results: []string{format(resp.Msg.Workflow)}})
}
func (h *handlers) stop(ctx cliapp.RunContext) error {
	resp, err := h.client.StopWorkflow(context.Background(), connect.NewRequest(&workflowspb.StopWorkflowRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("stop workflow", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Workflow == nil {
		return fmt.Errorf("server returned no workflow")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{ResultsHeading: "Workflow", Results: []string{format(resp.Msg.Workflow)}})
}
func (h *handlers) retry(ctx cliapp.RunContext) error {
	resp, err := h.client.RetryWorkflow(context.Background(), connect.NewRequest(&workflowspb.RetryWorkflowRequest{Id: ctx.Positional("id"), IdempotencyKey: ctx.Flag("idempotency-key")}))
	if err != nil {
		return cliapp.WrapAPIError("retry workflow", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Workflow == nil {
		return fmt.Errorf("server returned no workflow")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Workflow queued with depth %d.", resp.Msg.QueueDepth)}, ResultsHeading: "Workflow", Results: []string{format(resp.Msg.Workflow)}})
}

func parseKind(value string) (workflowspb.WorkflowKind, error) {
	switch value {
	case "extract":
		return workflowspb.WorkflowKind_WORKFLOW_KIND_EXTRACT, nil
	case "adopt":
		return workflowspb.WorkflowKind_WORKFLOW_KIND_ADOPT, nil
	default:
		return workflowspb.WorkflowKind_WORKFLOW_KIND_UNSPECIFIED, fmt.Errorf("--kind must be extract or adopt")
	}
}
func format(w *workflowspb.Workflow) string {
	if w == nil {
		return "(nil)"
	}
	detail := w.Summary
	if w.Error != "" {
		detail = w.Error
	}
	return fmt.Sprintf("%s [%s] asset=%s target=%s run=%s %s", w.Id, w.Status, w.AssetId, w.TargetScenario, w.AgentManagerRunId, detail)
}
