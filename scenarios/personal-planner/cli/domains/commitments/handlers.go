package commitments

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/commitments"
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/commitments/commitments_v1connect"
)

type handlers struct{ client c.CommitmentsServiceClient }

func newHandlers(app *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(app)
	return &handlers{client: c.NewCommitmentsServiceClient(httpClient, base)}
}
func (h *handlers) listCall(_ cliapp.OperationContext) (*v.ListCommitmentsResponse, error) {
	r, err := h.client.ListCommitments(context.Background(), connect.NewRequest(&v.ListCommitmentsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list commitments", err, nil)
	}
	if r == nil || r.Msg == nil {
		return nil, fmt.Errorf("server returned no commitments")
	}
	return r.Msg, nil
}
func (h *handlers) listReport(_ cliapp.OperationContext, m *v.ListCommitmentsResponse) cliapp.ListReport {
	out := make([]string, 0, len(m.Commitments))
	for _, x := range m.Commitments {
		out = append(out, fmt.Sprintf("%s — %s [%s; promised %s]", x.Id, x.Result, x.State, x.PromisedBoundary))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d commitment(s).", len(m.Commitments))}, ResultsHeading: "Commitments", Results: out}
}
func (h *handlers) createCall(cxt cliapp.OperationContext) (*v.CreateCommitmentResponse, error) {
	r, err := h.client.CreateCommitment(context.Background(), connect.NewRequest(&v.CreateCommitmentRequest{Result: cxt.Flag("result"), DefinitionOfDone: cxt.Flag("definition-of-done"), PromisedBoundary: cxt.Flag("promised-boundary"), Timezone: cxt.Flag("timezone"), Beneficiary: cxt.Flag("beneficiary"), Assumptions: cxt.Flag("assumptions"), ScopeExclusions: cxt.Flag("scope-exclusions"), State: cxt.Flag("state")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create commitment", err, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Commitment == nil {
		return nil, fmt.Errorf("server returned no commitment")
	}
	return r.Msg, nil
}
func (h *handlers) createReport(_ cliapp.OperationContext, m *v.CreateCommitmentResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created commitment %s.", m.Commitment.Id)}, Changes: []string{m.Commitment.Result}, NextCommand: []string{"`commitments list` — inspect promises"}}
}
func (h *handlers) stateCall(cxt cliapp.OperationContext) (*v.UpdateCommitmentStateResponse, error) {
	revision, err := strconv.ParseInt(cxt.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	r, err := h.client.UpdateCommitmentState(context.Background(), connect.NewRequest(&v.UpdateCommitmentStateRequest{Id: cxt.Positional("id"), State: cxt.Flag("state"), ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("update commitment state", err, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Commitment == nil {
		return nil, fmt.Errorf("server returned no commitment")
	}
	return r.Msg, nil
}
func (h *handlers) stateReport(_ cliapp.OperationContext, m *v.UpdateCommitmentStateResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Commitment %s is now %s.", m.Commitment.Id, m.Commitment.State)}}
}
func (h *handlers) reviseCall(cxt cliapp.OperationContext) (*v.ReviseCommitmentResponse, error) {
	revision, err := strconv.ParseInt(cxt.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	r, err := h.client.ReviseCommitment(context.Background(), connect.NewRequest(&v.ReviseCommitmentRequest{Id: cxt.Positional("id"), PromisedBoundary: cxt.Flag("promised-boundary"), Assumptions: cxt.Flag("assumptions"), ScopeExclusions: cxt.Flag("scope-exclusions"), Reason: cxt.Flag("reason"), AcknowledgmentStatus: cxt.Flag("acknowledgment-status"), ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("revise commitment", err, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Commitment == nil {
		return nil, fmt.Errorf("server returned no commitment")
	}
	return r.Msg, nil
}
func (h *handlers) reviseReport(_ cliapp.OperationContext, m *v.ReviseCommitmentResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded revision %d for %s.", m.Commitment.Revision, m.Commitment.Id)}, Changes: []string{m.Commitment.PromisedBoundary}}
}
