package recovery

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	recoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery"
	recoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery/recovery_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client recoveryconnect.RecoveryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: recoveryconnect.NewRecoveryServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) stateCall(_ cliapp.OperationContext) (*recoveryv1.GetStateResponse, error) {
	resp, err := h.client.GetState(context.Background(), connect.NewRequest(&recoveryv1.GetStateRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get recovery state", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.State == nil {
		return nil, fmt.Errorf("server returned no recovery state")
	}
	return resp.Msg, nil
}

func (h *handlers) stateReport(_ cliapp.OperationContext, message *recoveryv1.GetStateResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary:        []string{"Recovery state."},
		ResultsHeading: "State",
		Results:        []string{formatState(message.State)},
		RetrievalHints: []string{
			"`recovery events` — show the recovery event log",
			"`recovery run` — trigger a manual recovery",
		},
	}
}

func (h *handlers) eventsCall(ctx cliapp.OperationContext) (*recoveryv1.ListEventsResponse, error) {
	var limit int32
	if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("--limit must be an integer: %w", err)
		}
		limit = int32(n)
	}
	resp, err := h.client.ListEvents(context.Background(), connect.NewRequest(&recoveryv1.ListEventsRequest{Limit: limit}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list recovery events", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no events response")
	}
	return resp.Msg, nil
}

func (h *handlers) eventsReport(_ cliapp.OperationContext, message *recoveryv1.ListEventsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Events))
	for _, e := range message.Events {
		results = append(results, formatEvent(e))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d recovery event(s).", len(message.Events))},
		ResultsHeading: "Events",
		Results:        results,
	}
}

func (h *handlers) recoverCall(ctx cliapp.OperationContext) (*recoveryv1.RecoverResponse, error) {
	force := false
	if v := strings.TrimSpace(ctx.Flag("force")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("--force must be true or false: %w", err)
		}
		force = b
	}
	resp, err := h.client.Recover(context.Background(), connect.NewRequest(&recoveryv1.RecoverRequest{Force: force}))
	if err != nil {
		return nil, cliapp.WrapAPIError("trigger recovery", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no recover response")
	}
	return resp.Msg, nil
}

func (h *handlers) recoverReport(_ cliapp.OperationContext, message *recoveryv1.RecoverResponse) cliapp.MutationReport {
	changes := []string{fmt.Sprintf("Outcome: %s", strings.ToLower(strings.TrimPrefix(message.Outcome.String(), "EVENT_OUTCOME_")))}
	if message.Event != nil {
		changes = append(changes, formatEvent(message.Event))
	}
	return cliapp.MutationReport{
		Result:  []string{"Recovery attempt complete."},
		Changes: changes,
	}
}

func formatState(s *recoveryv1.RecoveryState) string {
	if s == nil {
		return "(nil)"
	}
	status := strings.ToLower(strings.TrimPrefix(s.Status.String(), "RECOVERY_STATUS_"))
	return fmt.Sprintf("status=%s consec_failures=%d backoff_level=%d failed=%d circuit_open=%t",
		status, s.ConsecFailures, s.BackoffLevel, s.FailedRecoveries, s.CircuitOpen)
}

func formatEvent(e *recoveryv1.RecoveryEvent) string {
	if e == nil {
		return "(nil)"
	}
	outcome := strings.ToLower(strings.TrimPrefix(e.Outcome.String(), "EVENT_OUTCOME_"))
	ts := ""
	if e.CreatedAt != nil {
		ts = e.CreatedAt.AsTime().Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s [%s] %s — %s (attempt %d) %s", ts, outcome, e.Trigger, e.Action, e.Attempt, e.Details)
}
