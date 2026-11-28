package sketch

import (
	"context"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

func (h *handlers) renderCandidateCall(ctx cliapp.OperationContext) (*sharedv1.RenderCompositionResponse, error) {
	response, err := h.candidateClient.RenderCandidate(context.Background(), connect.NewRequest(candidateRenderInput(ctx)))
	if err != nil {
		return nil, cliapp.WrapAPIError("render candidate", err, nil)
	}
	return response.Msg, nil
}

func renderCandidateHuman(ctx cliapp.RunContext, response *sharedv1.RenderCompositionResponse) error {
	return ctx.RenderList(cliapp.ListReport{Summary: []string{"Render hash: " + response.GetRenderHash(), "Use --json for generated HTML and complete rendering-input identities."}})
}

func candidateRenderInput(ctx cliapp.OperationContext) *sharedv1.RenderCandidateRequest {
	return &sharedv1.RenderCandidateRequest{Candidate: &sharedv1.CandidateReference{Scenario: ctx.Flag("scenario"), DesignId: ctx.Flag("design"), Hash: ctx.Flag("candidate")}, Kit: ctx.Flag("kit"), Theme: ctx.Flag("theme"), Direction: ctx.Flag("direction"), PreviewState: ctx.Flag("state"), MissingLabel: ctx.Flag("missing-label"), FailedLabel: ctx.Flag("failed-label")}
}
