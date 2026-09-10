package transitioncatalog

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/development"
)

// PreviewDevelopment intentionally cannot reach the runner or mutate approvals.
func (s *Service) PreviewDevelopment(_ context.Context, req *connect.Request[api.PreviewDevelopmentRequest]) (*connect.Response[api.PreviewDevelopmentResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if s.developmentReviewer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("development review source is not configured"))
	}
	p, err := development.ProposalFromProto(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	review, err := s.developmentReviewer.Preview(p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	response := &api.PreviewDevelopmentResponse{ProposalDigest: review.ProposalDigest, GoalMessage: review.GoalMessage, ReviewComplete: review.ReviewComplete, LaunchReady: false, LaunchBlockers: review.LaunchBlockers}
	for _, a := range review.Artifacts {
		response.Artifacts = append(response.Artifacts, &api.DevelopmentArtifact{Path: a.Path, Sha256: a.SHA256, SizeBytes: a.SizeBytes})
	}
	for _, f := range review.Findings {
		response.Findings = append(response.Findings, &api.DevelopmentReviewFinding{Code: f.Code, Detail: f.Detail})
	}
	return connect.NewResponse(response), nil
}
