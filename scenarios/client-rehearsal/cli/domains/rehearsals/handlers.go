package rehearsals

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/rehearsals"
	connectclient "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/rehearsals/rehearsals_v1connect"
)

type handlers struct {
	client connectclient.RehearsalServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: connectclient.NewRehearsalServiceClient(httpClient, baseURL)}
}
func (h *handlers) createCall(ctx cliapp.OperationContext) (*v1.CreateAndRunRehearsalResponse, error) {
	var requiredOutputPaths []string
	if raw := strings.TrimSpace(ctx.Flag("required-output-paths")); raw != "" {
		requiredOutputPaths = strings.Split(raw, ",")
	}
	resp, err := h.client.CreateAndRunRehearsal(context.Background(), connect.NewRequest(&v1.CreateAndRunRehearsalRequest{Title: ctx.Flag("title"), WorkflowJson: ctx.Flag("workflow-json"), TriggerNodeId: ctx.Flag("trigger-node-id"), SimulatorSourceJson: ctx.Flag("simulator-source-json"), InjectSeededFault: ctx.Flag("inject-seeded-fault") == "true", RequiredOutputPaths: requiredOutputPaths, RepairOfId: ctx.Flag("repair-of-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create and run rehearsal", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Rehearsal == nil {
		return nil, fmt.Errorf("server returned no rehearsal")
	}
	return resp.Msg, nil
}
func (h *handlers) createReport(_ cliapp.OperationContext, msg *v1.CreateAndRunRehearsalResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created rehearsal %s (%s).", msg.Rehearsal.Id, msg.Rehearsal.State)}, Changes: []string{formatRehearsal(msg.Rehearsal)}, NextCommand: []string{fmt.Sprintf("`rehearsals get %s` — inspect trace and handoff", msg.Rehearsal.Id), fmt.Sprintf("`rehearsals review %s --decision approve` — record human approval", msg.Rehearsal.Id)}}
}
func (h *handlers) listCall(ctx cliapp.OperationContext) (*v1.ListRehearsalsResponse, error) {
	limit := uint32(50)
	if value := strings.TrimSpace(ctx.Flag("limit")); value != "" {
		n, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("limit must be a positive integer")
		}
		limit = uint32(n)
	}
	resp, err := h.client.ListRehearsals(context.Background(), connect.NewRequest(&v1.ListRehearsalsRequest{Limit: limit}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list rehearsals", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) listReport(_ cliapp.OperationContext, msg *v1.ListRehearsalsResponse) cliapp.ListReport {
	items := make([]string, 0, len(msg.Rehearsals))
	for _, item := range msg.Rehearsals {
		items = append(items, formatRehearsal(item))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d rehearsal(s).", len(items))}, ResultsHeading: "Rehearsals", Results: items, RetrievalHints: []string{"`rehearsals get <id>` — inspect trace and handoff", "`rehearsals review <id> --decision approve|request_changes` — record a human decision"}}
}
func (h *handlers) getCall(ctx cliapp.OperationContext) (*v1.GetRehearsalResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetRehearsal(context.Background(), connect.NewRequest(&v1.GetRehearsalRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("get rehearsal %q", id), err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) getReport(_ cliapp.OperationContext, msg *v1.GetRehearsalResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Rehearsal %s (%s).", msg.Rehearsal.Id, msg.Rehearsal.State)}, ResultsHeading: "Handoff", Results: []string{formatRehearsal(msg.Rehearsal), msg.Rehearsal.HandoffMarkdown}}
}
func (h *handlers) reviewCall(ctx cliapp.OperationContext) (*v1.RecordReviewResponse, error) {
	decision := v1.ReviewDecision_REVIEW_DECISION_UNSPECIFIED
	switch strings.TrimSpace(ctx.Flag("decision")) {
	case "approve":
		decision = v1.ReviewDecision_REVIEW_DECISION_APPROVE
	case "request_changes":
		decision = v1.ReviewDecision_REVIEW_DECISION_REQUEST_CHANGES
	default:
		return nil, fmt.Errorf("decision must be approve or request_changes")
	}
	resp, err := h.client.RecordReview(context.Background(), connect.NewRequest(&v1.RecordReviewRequest{Id: ctx.Positional("id"), Decision: decision, Note: ctx.Flag("note")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record rehearsal review", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) reviewReport(_ cliapp.OperationContext, msg *v1.RecordReviewResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded review: %s.", msg.Rehearsal.State)}, Changes: []string{formatRehearsal(msg.Rehearsal)}}
}
func formatRehearsal(r *v1.Rehearsal) string {
	if r == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s [state=%s credentials-replaced=%d trace-events=%d]", r.Id, r.Title, r.State, r.CredentialReplacements, len(r.Trace))
}
