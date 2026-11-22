package main

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	repositoryID := ""
	if id := req.Msg.GetRepositoryId(); id > 0 {
		repositoryID = fmt.Sprintf("%d", id)
	}
	resolved, err := s.server.resolveRepoForConnect(ctx, repositoryID, "")
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

func (s reviewConnectServer) GetSummary(ctx context.Context, req *connect.Request[reviewv1.GetReviewSummaryRequest]) (*connect.Response[reviewv1.ReviewSummary], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("review summary request is required"))
	}
	name := strings.TrimSpace(req.Msg.GetScenarioName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario_name is required"))
	}
	if !IsValidScenarioName(name) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario_name contains invalid characters"))
	}
	repositoryID := ""
	if id := req.Msg.GetRepositoryId(); id > 0 {
		repositoryID = fmt.Sprintf("%d", id)
	}
	resolved, err := s.server.resolveRepoForConnect(ctx, repositoryID, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	summary := s.server.buildReviewSummary(ctx, resolved.ID, resolved.Path, name, clampNonNegative(int(req.Msg.GetDetails())), DefaultReadinessThresholds(), "", nil)
	return connect.NewResponse(reviewSummaryToProto(summary)), nil
}

func (s reviewConnectServer) GetJobStatus(ctx context.Context, req *connect.Request[reviewv1.GetReviewJobStatusRequest]) (*connect.Response[reviewv1.ReviewJobStatus], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("review job status request is required"))
	}
	repositoryID := ""
	if id := req.Msg.GetRepositoryId(); id > 0 {
		repositoryID = fmt.Sprintf("%d", id)
	}
	if _, err := s.server.resolveRepoForConnect(ctx, repositoryID, ""); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	jobID := strings.TrimSpace(req.Msg.GetJobId())
	if jobID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("job_id is required"))
	}
	job, ok := s.server.reviewJobStore.Get(jobID)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("review job not found"))
	}
	checks := make(map[string]string, len(job.Checks))
	for k, v := range job.Checks {
		checks[k] = string(v)
	}
	return connect.NewResponse(&reviewv1.ReviewJobStatus{JobId: job.JobID, Status: job.Status, Checks: checks, CheckExecutionIds: job.CheckExecutionIDs, Summary: reviewSummaryToProto(job.Summary), StartedAt: job.StartedAt, Error: job.Error}), nil
}

func reviewSummaryToProto(v *ReviewSummaryResponse) *reviewv1.ReviewSummary {
	if v == nil {
		return nil
	}
	out := &reviewv1.ReviewSummary{ScenarioName: v.ScenarioName, Readiness: string(v.Readiness), DimensionStatuses: v.DimensionStatuses, CheckStatuses: map[string]string{}, Capabilities: v.Capabilities, Timestamp: v.Timestamp}
	for k, status := range v.CheckStatuses {
		out.CheckStatuses[k] = string(status)
	}
	d := &reviewv1.ReviewDimensions{}
	out.Dimensions = d
	if x := v.Dimensions.CodeQuality; x != nil {
		d.CodeQuality = &reviewv1.CodeQualityDimension{Available: x.Available, Score: x.Score, Violations: int32(x.Violations), Stale: x.Stale, LastScan: x.LastScan}
		for _, i := range x.TopIssues {
			d.CodeQuality.TopIssues = append(d.CodeQuality.TopIssues, &reviewv1.CodeQualityIssue{Category: i.Category, Count: int32(i.Count)})
		}
	}
	if x := v.Dimensions.Tests; x != nil {
		d.Tests = &reviewv1.TestsDimension{Available: x.Available, Passed: x.Passed, Total: int32(x.Total), PassedCount: int32(x.PassedCount), FailedCount: int32(x.FailedCount), LastRun: x.LastRun}
		for _, i := range x.Failures {
			d.Tests.Failures = append(d.Tests.Failures, &reviewv1.TestFailure{Phase: i.Phase, Error: i.Error, Classification: i.Classification, Remediation: i.Remediation})
		}
	}
	if x := v.Dimensions.Standards; x != nil {
		d.Standards = &reviewv1.StandardsDimension{Available: x.Available, BlockingViolations: int32(x.BlockingViolations), Warnings: int32(x.Warnings), TotalViolations: int32(x.TotalViolations)}
		for _, i := range x.TopViolations {
			d.Standards.TopViolations = append(d.Standards.TopViolations, &reviewv1.StandardsViolationDetail{FilePath: i.FilePath, LineNumber: int32(i.LineNumber), Title: i.Title, Severity: i.Severity, Recommendation: i.Recommendation})
		}
	}
	if x := v.Dimensions.Visual; x != nil {
		d.Visual = &reviewv1.VisualDimension{Available: x.Available, ScreenshotCount: int32(x.ScreenshotCount), Stale: x.Stale}
		if i := x.LatestCapture; i != nil {
			d.Visual.LatestCapture = &reviewv1.VisualCaptureMeta{CapturedAt: i.CapturedAt, CommitHash: i.CommitHash, ScreenshotCount: int32(i.ScreenshotCount)}
		}
	}
	if x := v.Dimensions.Provenance; x != nil {
		d.Provenance = &reviewv1.ProvenanceDimension{Available: x.Available, TracedFiles: int32(x.TracedFiles), UntracedFiles: x.UntracedFiles}
	}
	return out
}
