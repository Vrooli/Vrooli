package work

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	workv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work"
	workconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work/work_v1connect"
)

type handlers struct{ client workconnect.WorkServiceClient }

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: workconnect.NewWorkServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*workv1.ListWorkItemsResponse, error) {
	resp, err := h.client.ListWorkItems(context.Background(), connect.NewRequest(&workv1.ListWorkItemsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list work items", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no work response")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *workv1.ListWorkItemsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.WorkItems))
	for _, item := range msg.WorkItems {
		results = append(results, formatItem(item))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d work item(s).", len(msg.WorkItems))}, ResultsHeading: "Work", Results: results, RetrievalHints: []string{"`work create --title <title> --remaining-minutes <minutes>` — capture work", "`work get <id>` — inspect one item"}}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*workv1.CreateWorkItemResponse, error) {
	minutes, err := strconv.ParseInt(ctx.Flag("remaining-minutes"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("remaining-minutes must be an integer: %w", err)
	}
	resp, err := h.client.CreateWorkItem(context.Background(), connect.NewRequest(&workv1.CreateWorkItemRequest{Title: ctx.Flag("title"), Description: ctx.Flag("description"), RemainingMinutes: int32(minutes), SourceLabel: ctx.Flag("source-label")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create work item", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.WorkItem == nil {
		return nil, fmt.Errorf("server returned no work item")
	}
	return resp.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, msg *workv1.CreateWorkItemResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Captured work item %s.", msg.WorkItem.Id)}, Changes: []string{formatItem(msg.WorkItem)}, NextCommand: []string{fmt.Sprintf("`work get %s` — inspect this item", msg.WorkItem.Id), "`work list` — show the backlog"}}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*workv1.GetWorkItemResponse, error) {
	resp, err := h.client.GetWorkItem(context.Background(), connect.NewRequest(&workv1.GetWorkItemRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get work item", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.WorkItem == nil {
		return nil, fmt.Errorf("server returned no work item")
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *workv1.GetWorkItemResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Fetched work item %s.", msg.WorkItem.Id)}, ResultsHeading: "Work item", Results: []string{formatItem(msg.WorkItem)}}
}

func (h *handlers) planCall(_ cliapp.OperationContext) (*workv1.GetTodayPlanResponse, error) {
	resp, err := h.client.GetTodayPlan(context.Background(), connect.NewRequest(&workv1.GetTodayPlanRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get today's plan", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no today plan")
	}
	return resp.Msg, nil
}

func (h *handlers) planReport(_ cliapp.OperationContext, msg *workv1.GetTodayPlanResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Entries))
	for _, entry := range msg.Entries {
		results = append(results, fmt.Sprintf("%s — %d min at %02d:%02d [%s]", entry.Title, entry.DurationMinutes, entry.StartMinutes/60, entry.StartMinutes%60, entry.SourceLabel))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d min planned, %d min available, %d min breathing room.", msg.PlannedMinutes, msg.AvailableMinutes, msg.BreathingRoomMinutes)}, ResultsHeading: "Today plan", Results: results}
}

func formatItem(item *workv1.WorkItem) string {
	if item == nil {
		return "(nil)"
	}
	created := ""
	if item.CreatedAt != nil {
		created = item.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [%d min remaining, source=%s, created=%s]", item.Id, item.Title, item.RemainingMinutes, item.SourceLabel, created)
}
