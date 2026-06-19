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

func (h *handlers) state(ctx cliapp.RunContext) error {
	resp, err := h.client.GetState(context.Background(), connect.NewRequest(&recoveryv1.GetStateRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get recovery state", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.State == nil {
		return fmt.Errorf("server returned no recovery state")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{"Recovery state."},
		ResultsHeading: "State",
		Results:        []string{formatState(resp.Msg.State)},
		RetrievalHints: []string{
			"`recovery events` — show the recovery event log",
			"`recovery run` — trigger a manual recovery",
		},
	})
}

func (h *handlers) events(ctx cliapp.RunContext) error {
	var limit int32
	if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--limit must be an integer: %w", err)
		}
		limit = int32(n)
	}
	resp, err := h.client.ListEvents(context.Background(), connect.NewRequest(&recoveryv1.ListEventsRequest{Limit: limit}))
	if err != nil {
		return cliapp.WrapAPIError("list recovery events", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no events response")
	}
	results := make([]string, 0, len(resp.Msg.Events))
	for _, e := range resp.Msg.Events {
		results = append(results, formatEvent(e))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d recovery event(s).", len(resp.Msg.Events))},
		ResultsHeading: "Events",
		Results:        results,
	})
}

func (h *handlers) recover(ctx cliapp.RunContext) error {
	force := false
	if v := strings.TrimSpace(ctx.Flag("force")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("--force must be true or false: %w", err)
		}
		force = b
	}
	resp, err := h.client.Recover(context.Background(), connect.NewRequest(&recoveryv1.RecoverRequest{Force: force}))
	if err != nil {
		return cliapp.WrapAPIError("trigger recovery", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no recover response")
	}
	changes := []string{fmt.Sprintf("Outcome: %s", strings.ToLower(strings.TrimPrefix(resp.Msg.Outcome.String(), "EVENT_OUTCOME_")))}
	if resp.Msg.Event != nil {
		changes = append(changes, formatEvent(resp.Msg.Event))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{"Recovery attempt complete."},
		Changes: changes,
	})
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
