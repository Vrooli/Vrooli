package sketch

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
)

func acceptanceCheckInput(ctx cliapp.RunContext) *sketchv1.CheckCandidateAcceptanceRequest {
	return &sketchv1.CheckCandidateAcceptanceRequest{Render: candidateRenderInput(ctx), ExpectedRenderHash: ctx.Flag("render-hash"), CritiqueIds: ctx.FlagValues("critique")}
}
func (h *handlers) acceptanceCheck(ctx cliapp.RunContext) error {
	response, err := h.client.CheckCandidateAcceptance(context.Background(), connect.NewRequest(acceptanceCheckInput(ctx)))
	if err != nil {
		return cliapp.WrapAPIError("check acceptance requirements", err, nil)
	}
	lines := []string{"Read-only acceptance check; no acceptance receipt was published."}
	for _, r := range response.Msg.Requirements {
		lines = append(lines, fmt.Sprintf("%s: %s — %s", r.Code, r.Status, r.Detail))
	}
	return render(ctx, response.Msg, lines)
}
func (h *handlers) accept(ctx cliapp.RunContext) error {
	response, err := h.client.AcceptCandidate(context.Background(), connect.NewRequest(&sketchv1.AcceptCandidateRequest{Check: acceptanceCheckInput(ctx), Actor: ctx.Flag("actor"), IdempotencyKey: ctx.Flag("key")}))
	if err != nil {
		return fmt.Errorf("%w; retry with identical inputs and the same key after a lost response", cliapp.WrapAPIError("record acceptance decision", err, nil))
	}
	return render(ctx, response.Msg, []string{"Decision: " + response.Msg.State, "Operation: " + response.Msg.Id, "This decision does not apply product code."})
}
func (h *handlers) acceptance(ctx cliapp.RunContext) error {
	response, err := h.client.GetAcceptance(context.Background(), connect.NewRequest(&sketchv1.GetAcceptanceRequest{Scenario: ctx.Flag("scenario"), DesignId: ctx.Flag("design"), Id: ctx.Flag("id")}))
	if err != nil {
		return cliapp.WrapAPIError("read acceptance decision", err, nil)
	}
	return render(ctx, response.Msg, []string{"Decision: " + response.Msg.State, "Operation: " + response.Msg.Id, "Recorded input hash: " + response.Msg.Hash})
}

func (h *handlers) renderCandidate(ctx cliapp.RunContext) error {
	response, err := h.client.RenderCandidate(context.Background(), connect.NewRequest(candidateRenderInput(ctx)))
	if err != nil {
		return cliapp.WrapAPIError("render candidate", err, nil)
	}
	return render(ctx, response.Msg, []string{"Render hash: " + response.Msg.RenderHash, "Use --json for generated HTML and complete rendering-input identities."})
}

func candidateRenderInput(ctx cliapp.RunContext) *sketchv1.RenderCandidateRequest {
	return &sketchv1.RenderCandidateRequest{Candidate: &sketchv1.CandidateReference{Scenario: ctx.Flag("scenario"), DesignId: ctx.Flag("design"), Hash: ctx.Flag("candidate")}, Kit: ctx.Flag("kit"), Theme: ctx.Flag("theme"), Direction: ctx.Flag("direction"), PreviewState: ctx.Flag("state"), MissingLabel: ctx.Flag("missing-label"), FailedLabel: ctx.Flag("failed-label")}
}
