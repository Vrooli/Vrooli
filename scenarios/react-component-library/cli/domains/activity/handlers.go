package activity

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity"
	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity/activity_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp + the generated
// Connect-Go client. Mirrors cli/domains/adoptions/.
type handlers struct {
	core   *cliapp.ScenarioApp
	client activityconnect.ActivityServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: activityconnect.NewActivityServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) eventsCall(ctx cliapp.OperationContext) (*activityv1.ListActivityResponse, error) {
	req := &activityv1.ListActivityRequest{
		SubjectKind: ctx.Flag("subject-kind"),
		SubjectId:   ctx.Flag("subject-id"),
		Actor:       ctx.Flag("actor"),
		Since:       ctx.Flag("since"),
	}
	limit, err := parseNonNegativeInt32(ctx.Flag("limit"), "--limit")
	if err != nil {
		return nil, err
	}
	req.Limit = limit
	resp, err := h.client.ListActivity(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("list activity", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no activity events")
	}
	return resp.Msg, nil
}

func (h *handlers) eventsReport(_ cliapp.OperationContext, msg *activityv1.ListActivityResponse) cliapp.ListReport {
	events := msg.GetEvents()
	rows := make([]string, 0, len(events))
	for _, event := range events {
		rows = append(rows, formatEvent(event))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d activity event(s).", len(events))},
		ResultsHeading: "Events (newest first)",
		Results:        rows,
		RetrievalHints: []string{
			"`activity summary --days 7` — see bounded counts for the Stats view",
		},
	}
}

func (h *handlers) summaryCall(ctx cliapp.OperationContext) (*activityv1.GetActivitySummaryResponse, error) {
	days, err := parseNonNegativeInt32(ctx.Flag("days"), "--days")
	if err != nil {
		return nil, err
	}
	resp, err := h.client.GetActivitySummary(context.Background(), connect.NewRequest(&activityv1.GetActivitySummaryRequest{Days: days}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get activity summary", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no activity summary")
	}
	return resp.Msg, nil
}

func (h *handlers) summaryReport(_ cliapp.OperationContext, msg *activityv1.GetActivitySummaryResponse) cliapp.ListReport {
	rows := make([]string, 0, len(msg.GetByDay())+len(msg.GetByActor())+len(msg.GetByAction())+3)
	if msg.GetEmpty() {
		// Empty is a measured zero, not an unavailable source: say so instead of
		// letting an empty table imply a failed read.
		rows = append(rows, "activity log: empty (measured zero events)")
	} else {
		rows = append(rows, fmt.Sprintf("total events: %d", msg.GetTotal()))
	}
	if days := msg.GetByDay(); len(days) > 0 {
		rows = append(rows, "by day:")
		for _, day := range days {
			if day == nil {
				continue
			}
			rows = append(rows, fmt.Sprintf("  %s: %d", day.GetDate(), day.GetCount()))
		}
	}
	if actors := msg.GetByActor(); len(actors) > 0 {
		rows = append(rows, "by actor:")
		for _, actor := range actors {
			if actor == nil {
				continue
			}
			rows = append(rows, fmt.Sprintf("  %s: %d", nonEmpty(actor.GetActor(), "(unattributed)"), actor.GetCount()))
		}
	}
	if actions := msg.GetByAction(); len(actions) > 0 {
		rows = append(rows, "by action:")
		for _, action := range actions {
			if action == nil {
				continue
			}
			rows = append(rows, fmt.Sprintf("  %s: %d", nonEmpty(action.GetAction(), "(unnamed)"), action.GetCount()))
		}
	}
	summary := fmt.Sprintf("Activity summary: %d event(s).", msg.GetTotal())
	if msg.GetEmpty() {
		summary = "Activity summary: the log is empty (measured zero events)."
	}
	return cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Activity",
		Results:        rows,
		RetrievalHints: []string{
			"`activity events --limit 20` — inspect the recent append-only log",
		},
	}
}

func formatEvent(event *activityv1.ActivityEvent) string {
	if event == nil {
		return "(nil event)"
	}
	subject := event.GetSubjectKind()
	if subject == "" {
		subject = "subject"
	}
	if event.GetSubjectId() != "" {
		subject += "/" + event.GetSubjectId()
	}
	line := fmt.Sprintf("%s %s [%s] %s — %s",
		nonEmpty(event.GetAt(), "(unknown time)"),
		nonEmpty(event.GetActor(), "(unattributed)"),
		nonEmpty(event.GetAction(), "(no action)"),
		subject,
		nonEmpty(event.GetSummary(), "(no summary)"))
	if trace := strings.TrimSpace(event.GetTraceId()); trace != "" {
		line += " trace=" + trace
	}
	return line
}

func parseNonNegativeInt32(raw, flag string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer (got %q)", flag, raw)
	}
	return int32(value), nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
