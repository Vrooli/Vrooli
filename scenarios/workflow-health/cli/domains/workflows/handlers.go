package workflows

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/workflow-health/v1/workflows"
	workflowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/workflow-health/v1/workflows/workflows_v1connect"
)

type handlers struct {
	client workflowsconnect.WorkflowSearchServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: workflowsconnect.NewWorkflowSearchServiceClient(httpClient, baseURL)}
}

func (h *handlers) search(ctx cliapp.RunContext) error {
	query := strings.TrimSpace(ctx.Positional("query"))
	if query == "" {
		return fmt.Errorf("query is required")
	}
	resp, err := h.client.SearchWorkflows(context.Background(), connect.NewRequest(&workflowsv1.SearchWorkflowsRequest{
		Scenario:         strings.TrimSpace(ctx.Flag("scenario")),
		Path:             strings.TrimSpace(ctx.Flag("path")),
		Query:            query,
		Types:            splitCSV(ctx.Flag("type")),
		IncludeFragments: ctx.BoolFlag("include-fragments"),
		Limit:            parseLimit(ctx.Flag("limit")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("search workflows for %q", query), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no workflow search response")
	}
	results := make([]string, 0, len(resp.Msg.GetResults()))
	for _, result := range resp.Msg.GetResults() {
		results = append(results, formatResult(result))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Found %d workflow result(s) for %s.", len(resp.Msg.GetResults()), resp.Msg.GetScenario()),
		},
		ResultsHeading: "Workflow results",
		Results:        results,
		RetrievalHints: []string{
			"`--type workflow.flow` - runnable journey candidates",
			"`--type workflow.test` - validation evidence cases",
			"`--type workflow.fragment` - reusable dependency fragments",
			"`--include-fragments` - include fragments alongside default flow/test results",
		},
	})
}

func formatResult(result *workflowsv1.WorkflowSearchResult) string {
	title := strings.TrimSpace(result.GetTitle())
	if title == "" {
		title = result.GetId()
	}
	line := fmt.Sprintf("%s [%s score=%.2f] %s", title, result.GetLeafType(), result.GetScore(), result.GetPath())
	if result.GetSafetySummary() != "" {
		line += "\n    safety: " + result.GetSafetySummary()
	}
	if len(result.GetRequirementIds()) > 0 {
		line += "\n    requirements: " + strings.Join(result.GetRequirementIds(), ", ")
	}
	if len(result.GetGuardrails()) > 0 {
		line += "\n    guardrails: " + strings.Join(result.GetGuardrails(), "; ")
	}
	return line
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseLimit(value string) int32 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil || n < 1 {
		return 0
	}
	return int32(n)
}
