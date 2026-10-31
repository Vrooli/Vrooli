package focus

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus"
	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus/focus_v1connect"
)

type handlers struct {
	client focusconnect.FocusServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: focusconnect.NewFocusServiceClient(httpClient, baseURL)}
}

func (h *handlers) currentCall(_ cliapp.OperationContext) (*focusv1.GetCurrentSessionResponse, error) {
	resp, err := h.client.GetCurrentSession(context.Background(), connect.NewRequest(&focusv1.GetCurrentSessionRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get current focus", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no focus response")
	}
	return resp.Msg, nil
}

func (h *handlers) currentReport(_ cliapp.OperationContext, msg *focusv1.GetCurrentSessionResponse) cliapp.ListReport {
	if !msg.HasSession || msg.Session == nil {
		return cliapp.ListReport{Summary: []string{"No focus session is active."}}
	}
	return cliapp.ListReport{Summary: []string{"Current focus session."}, ResultsHeading: "Focus", Results: []string{formatSession(msg.Session)}}
}

func (h *handlers) startCall(ctx cliapp.OperationContext) (*focusv1.FocusSessionResponse, error) {
	mode := ctx.Flag("mode")
	if mode == "" {
		mode = "open"
	}
	resp, err := h.client.StartFocus(context.Background(), connect.NewRequest(&focusv1.StartFocusRequest{WorkItemId: ctx.Flag("work-item-id"), Title: ctx.Flag("title"), Mode: mode}))
	if err != nil {
		return nil, cliapp.WrapAPIError("start focus", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Session == nil {
		return nil, fmt.Errorf("server returned no focus session")
	}
	return resp.Msg, nil
}

func (h *handlers) startReport(_ cliapp.OperationContext, msg *focusv1.FocusSessionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Started focus session %s.", msg.Session.Id)}, Changes: []string{formatSession(msg.Session)}, NextCommand: []string{"`focus current` — inspect the active session", fmt.Sprintf("`focus pause %s --revision %d` — pause it", msg.Session.Id, msg.Session.Revision)}}
}

func (h *handlers) transitionCall(ctx cliapp.OperationContext, action string) (*focusv1.FocusSessionResponse, error) {
	revision, err := strconv.ParseInt(ctx.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	req := connect.NewRequest(&focusv1.TransitionFocusRequest{SessionId: ctx.Positional("id"), ExpectedRevision: revision})
	var resp *connect.Response[focusv1.FocusSessionResponse]
	switch action {
	case "pause":
		resp, err = h.client.PauseFocus(context.Background(), req)
	case "resume":
		resp, err = h.client.ResumeFocus(context.Background(), req)
	default:
		return nil, fmt.Errorf("unsupported focus transition %q", action)
	}
	if err != nil {
		return nil, cliapp.WrapAPIError(action+" focus", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Session == nil {
		return nil, fmt.Errorf("server returned no focus session")
	}
	return resp.Msg, nil
}

func (h *handlers) pauseCall(ctx cliapp.OperationContext) (*focusv1.FocusSessionResponse, error) {
	return h.transitionCall(ctx, "pause")
}

func (h *handlers) resumeCall(ctx cliapp.OperationContext) (*focusv1.FocusSessionResponse, error) {
	return h.transitionCall(ctx, "resume")
}

func (h *handlers) transitionReport(_ cliapp.OperationContext, msg *focusv1.FocusSessionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Focus session is now %s.", msg.Session.State)}, Changes: []string{formatSession(msg.Session)}}
}

func (h *handlers) endCall(ctx cliapp.OperationContext) (*focusv1.FocusSessionResponse, error) {
	revision, err := strconv.ParseInt(ctx.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	remaining, err := strconv.ParseInt(ctx.Flag("reported-remaining-minutes"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("reported-remaining-minutes must be an integer: %w", err)
	}
	resp, err := h.client.EndFocus(context.Background(), connect.NewRequest(&focusv1.EndFocusRequest{SessionId: ctx.Positional("id"), ExpectedRevision: revision, ReportedRemainingMinutes: remaining}))
	if err != nil {
		return nil, cliapp.WrapAPIError("end focus", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Session == nil {
		return nil, fmt.Errorf("server returned no focus session")
	}
	return resp.Msg, nil
}

func (h *handlers) endReport(_ cliapp.OperationContext, msg *focusv1.FocusSessionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Ended focus session %s.", msg.Session.Id)}, Changes: []string{formatSession(msg.Session)}}
}

func (h *handlers) recordActualCall(ctx cliapp.OperationContext) (*focusv1.ActualResponse, error) {
	minutes, err := strconv.ParseInt(ctx.Flag("minutes"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("minutes must be an integer: %w", err)
	}
	resp, err := h.client.RecordManualActual(context.Background(), connect.NewRequest(&focusv1.RecordManualActualRequest{WorkItemId: ctx.Flag("work-item-id"), Title: ctx.Flag("title"), LocalDate: ctx.Flag("date"), ReportedMinutes: minutes, Certainty: ctx.Flag("certainty"), Note: ctx.Flag("note")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record actual", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Actual == nil {
		return nil, fmt.Errorf("server returned no actual")
	}
	return resp.Msg, nil
}

func (h *handlers) recordActualReport(_ cliapp.OperationContext, msg *focusv1.ActualResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded %d minutes for %s.", msg.Actual.ReportedMinutes, msg.Actual.LocalDate)}, Changes: []string{formatActual(msg.Actual)}}
}

func (h *handlers) listActualsCall(ctx cliapp.OperationContext) (*focusv1.ListActualsResponse, error) {
	resp, err := h.client.ListActuals(context.Background(), connect.NewRequest(&focusv1.ListActualsRequest{LocalDate: ctx.Flag("date")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list actuals", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no actuals")
	}
	return resp.Msg, nil
}

func (h *handlers) listActualsReport(_ cliapp.OperationContext, msg *focusv1.ListActualsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Actuals))
	for _, actual := range msg.Actuals {
		results = append(results, formatActual(actual))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d actual record(s).", len(msg.Actuals))}, ResultsHeading: "Actuals", Results: results}
}

func (h *handlers) listCorrectionsCall(ctx cliapp.OperationContext) (*focusv1.ListActualCorrectionsResponse, error) {
	limit := int32(100)
	if raw := ctx.Flag("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("limit must be an integer: %w", err)
		}
		limit = int32(parsed)
	}
	resp, err := h.client.ListActualCorrections(context.Background(), connect.NewRequest(&focusv1.ListActualCorrectionsRequest{ActualId: ctx.Flag("actualId"), LocalDate: ctx.Flag("date"), Limit: limit}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list actual corrections", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no actual corrections")
	}
	return resp.Msg, nil
}

func (h *handlers) listCorrectionsReport(_ cliapp.OperationContext, msg *focusv1.ListActualCorrectionsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Corrections))
	for _, correction := range msg.Corrections {
		results = append(results, formatCorrection(correction))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d correction record(s).", len(msg.Corrections))}, ResultsHeading: "Actual corrections", Results: results}
}

func (h *handlers) correctActualCall(ctx cliapp.OperationContext) (*focusv1.ActualResponse, error) {
	minutes, err := strconv.ParseInt(ctx.Flag("minutes"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("minutes must be an integer: %w", err)
	}
	revision, err := strconv.ParseInt(ctx.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	resp, err := h.client.CorrectActual(context.Background(), connect.NewRequest(&focusv1.CorrectActualRequest{Id: ctx.Positional("id"), ExpectedRevision: revision, ReportedMinutes: minutes, Certainty: ctx.Flag("certainty"), Note: ctx.Flag("note")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("correct actual", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Actual == nil {
		return nil, fmt.Errorf("server returned no actual")
	}
	return resp.Msg, nil
}

func (h *handlers) correctActualReport(_ cliapp.OperationContext, msg *focusv1.ActualResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Corrected actual %s.", msg.Actual.Id)}, Changes: []string{formatActual(msg.Actual)}}
}

func formatActual(actual *focusv1.Actual) string {
	if actual == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s: %d min [%s, revision=%d]", actual.Id, actual.LocalDate, actual.ReportedMinutes, actual.Certainty, actual.Revision)
}

func formatCorrection(correction *focusv1.ActualCorrection) string {
	if correction == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s: %d → %d min [%s → %s] — %s", correction.Id, correction.ActualId, correction.PreviousMinutes, correction.NewMinutes, correction.PreviousCertainty, correction.NewCertainty, correction.Reason)
}

func formatSession(session *focusv1.FocusSession) string {
	if session == nil {
		return "(nil)"
	}
	started := time.Unix(session.StartedAtUnixSeconds, 0).UTC().Format(time.RFC3339)
	return fmt.Sprintf("%s — %s [%s, active=%ds, wall=%ds, started=%s]", session.Id, session.Title, session.State, session.ActiveSeconds, session.WallSeconds, started)
}
