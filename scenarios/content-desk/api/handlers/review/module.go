// Package review mounts the review-run Connect surface.
package review

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"content-desk/internal/module"
	internalreview "content-desk/internal/review"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review/review_v1connect"
)

type handler struct{ service internalreview.Service }

var _ reviewconnect.ReviewServiceHandler = handler{}

func (h handler) ListReviewRuns(ctx context.Context, _ *connect.Request[reviewv1.ListReviewRunsRequest]) (*connect.Response[reviewv1.ListReviewRunsResponse], error) {
	runs, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &reviewv1.ListReviewRunsResponse{}
	for _, run := range runs {
		response.ReviewRuns = append(response.ReviewRuns, &reviewv1.ReviewRun{Id: run.ID, DraftId: run.DraftID, Outcome: run.Outcome})
	}
	return connect.NewResponse(response), nil
}

func (h handler) RecordReviewRun(ctx context.Context, request *connect.Request[reviewv1.RecordReviewRunRequest]) (*connect.Response[reviewv1.RecordReviewRunResponse], error) {
	verdicts := make([]internalreview.Verdict, 0, len(request.Msg.Verdicts))
	for _, verdict := range request.Msg.Verdicts {
		if verdict == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("review verdict cannot be empty"))
		}
		verdicts = append(verdicts, internalreview.Verdict{Mode: verdict.Mode, Passed: verdict.Passed, Evidence: verdict.Evidence, Finding: verdict.Finding})
	}
	run, err := h.service.Record(ctx, request.Msg.DraftId, verdicts)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&reviewv1.RecordReviewRunResponse{ReviewRun: reviewRunMessage(run)}), nil
}

func reviewRunMessage(run internalreview.Run) *reviewv1.ReviewRun {
	return &reviewv1.ReviewRun{Id: run.ID, DraftId: run.DraftID, Outcome: run.Outcome}
}

func Module(db *database.RoutedDB) module.Module {
	path, h := reviewconnect.NewReviewServiceHandler(handler{service: internalreview.NewService(db)})
	return module.Module{Name: "review", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalreview.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "review_list", Path: reviewconnect.ReviewServiceListReviewRunsProcedure, Method: "POST", Summary: "List review runs", Category: "review"},
	{ID: "review_record", Path: reviewconnect.ReviewServiceRecordReviewRunProcedure, Method: "POST", Summary: "Record re-runnable craft and policy verdicts", Category: "review"},
}
