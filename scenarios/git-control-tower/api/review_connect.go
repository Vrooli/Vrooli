package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/review/review_v1connect"
)

var _ reviewconnect.ReviewServiceHandler = (*reviewConnectServer)(nil)

type reviewConnectServer struct{ server *Server }

func (s reviewConnectServer) Start(ctx context.Context, req *connect.Request[reviewv1.StartReviewRequest]) (*connect.Response[reviewv1.StartReviewResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("review request is required"))
	}
	resolved, err := s.server.resolveRepoForConnect(ctx, fmt.Sprintf("%d", req.Msg.GetRepositoryId()), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	request := ReviewRunRequest{
		ScenarioName: strings.TrimSpace(req.Msg.GetScenarioName()),
		Checks:       append([]string(nil), req.Msg.GetChecks()...),
		Details:      int(req.Msg.GetDetails()),
	}
	if thresholds := req.Msg.GetThresholds(); thresholds != nil {
		request.Thresholds = &ReadinessThresholds{
			CodeQualityMinScore:   thresholds.GetCodeQualityMinScore(),
			TestMinPassRate:       thresholds.GetTestMinPassRate(),
			MaxBlockingViolations: int(thresholds.GetMaxBlockingViolations()),
			MaxWarnings:           int(thresholds.GetMaxWarnings()),
			RequireScreenshots:    thresholds.GetRequireScreenshots(),
			RequireTests:          thresholds.GetRequireTests(),
		}
	}
	response, err := s.server.startReviewRun(ctx, resolved.ID, resolved.Path, request)
	if err != nil {
		if conflict, ok := err.(*reviewRunConflictError); ok {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("a review run is already in progress for this scenario: %s", conflict.JobID))
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&reviewv1.StartReviewResponse{JobId: response.JobID}), nil
}
