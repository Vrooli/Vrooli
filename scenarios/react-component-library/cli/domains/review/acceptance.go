package review

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

func acceptanceCheckInput(ctx cliapp.OperationContext) *reviewv1.CheckCandidateAcceptanceRequest {
	return &reviewv1.CheckCandidateAcceptanceRequest{
		Render: &sharedv1.RenderCandidateRequest{
			Candidate: &sharedv1.CandidateReference{Scenario: ctx.Flag("scenario"), DesignId: ctx.Flag("design"), Hash: ctx.Flag("candidate")},
			Kit:       ctx.Flag("kit"), Theme: ctx.Flag("theme"), Direction: ctx.Flag("direction"), PreviewState: ctx.Flag("state"),
			MissingLabel: ctx.Flag("missing-label"), FailedLabel: ctx.Flag("failed-label"),
		},
		ExpectedRenderHash: ctx.Flag("render-hash"), CritiqueIds: ctx.FlagValues("critique"),
	}
}

func (h *handlers) acceptanceCheckCall(ctx cliapp.OperationContext) (*reviewv1.CandidateAcceptanceCheck, error) {
	response, err := h.client.CheckCandidateAcceptance(context.Background(), connect.NewRequest(acceptanceCheckInput(ctx)))
	if err != nil {
		return nil, cliapp.WrapAPIError("check acceptance requirements", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) acceptanceCheckReport(_ cliapp.OperationContext, result *reviewv1.CandidateAcceptanceCheck) cliapp.ListReport {
	rows := make([]string, 0, len(result.Requirements))
	for _, requirement := range result.Requirements {
		rows = append(rows, fmt.Sprintf("%s: %s — %s", requirement.Code, requirement.Status, requirement.Detail))
	}
	return cliapp.ListReport{Summary: []string{"Read-only acceptance check; no acceptance receipt was published."}, ResultsHeading: "Requirements", Results: rows}
}

func (h *handlers) acceptCall(ctx cliapp.OperationContext) (*reviewv1.AcceptanceDecision, error) {
	response, err := h.client.AcceptCandidate(context.Background(), connect.NewRequest(&reviewv1.AcceptCandidateRequest{
		Check: acceptanceCheckInput(ctx), Actor: ctx.Flag("actor"), IdempotencyKey: ctx.Flag("key"),
	}))
	if err != nil {
		return nil, fmt.Errorf("%w; retry with identical inputs and the same key after a lost response", cliapp.WrapAPIError("record acceptance decision", err, nil))
	}
	return response.Msg, nil
}

func (h *handlers) acceptReport(_ cliapp.OperationContext, result *reviewv1.AcceptanceDecision) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Decision: " + result.State}, Changes: []string{"Operation: " + result.Id, "This decision does not apply product code."}}
}

func (h *handlers) acceptanceCall(ctx cliapp.OperationContext) (*reviewv1.AcceptanceDecision, error) {
	response, err := h.client.GetAcceptance(context.Background(), connect.NewRequest(&reviewv1.GetAcceptanceRequest{Scenario: ctx.Flag("scenario"), DesignId: ctx.Flag("design"), Id: ctx.Flag("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read acceptance decision", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) acceptanceReport(_ cliapp.OperationContext, result *reviewv1.AcceptanceDecision) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Decision: " + result.State, "Operation: " + result.Id, "Recorded input hash: " + result.Hash}}
}
