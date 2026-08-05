package evidencehandler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	internalEvidence "deployment-manager/internal/evidence"
	"deployment-manager/internal/evidence/conformance"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
)

type ConnectHandler struct {
	evidenceconnect.UnimplementedEvidenceServiceHandler
	repo internalEvidence.Repository
}

func NewConnectHandler(repo internalEvidence.Repository) *ConnectHandler {
	return &ConnectHandler{repo: repo}
}

func (h *ConnectHandler) ReportTargetVerdict(ctx context.Context, req *connect.Request[evidencev1.ReportTargetVerdictRequest]) (*connect.Response[evidencev1.ReportTargetVerdictResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if violations := conformance.Validate(req.Msg.Verdict); len(violations) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, violations[0])
	}
	if err := h.repo.Save(ctx, req.Msg.ProfileId, req.Msg.GitCommitHash, req.Msg.Verdict); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&evidencev1.ReportTargetVerdictResponse{Verdict: req.Msg.Verdict}), nil
}

func (h *ConnectHandler) ListTargetVerdicts(ctx context.Context, req *connect.Request[evidencev1.ListTargetVerdictsRequest]) (*connect.Response[evidencev1.ListTargetVerdictsResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.ProfileId == "" || req.Msg.GitCommitHash == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("profile_id and git_commit_hash are required"))
	}
	verdicts, err := h.repo.List(ctx, req.Msg.ProfileId, req.Msg.GitCommitHash, int(req.Msg.PageSize))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&evidencev1.ListTargetVerdictsResponse{
		Verdicts: verdicts,
		Count:    int32(len(verdicts)),
	}), nil
}

func (h *ConnectHandler) GetEvidenceReview(ctx context.Context, req *connect.Request[evidencev1.GetEvidenceReviewRequest]) (*connect.Response[evidencev1.EvidenceReview], error) {
	if req == nil || req.Msg == nil || req.Msg.ProfileId == "" || req.Msg.GitCommitHash == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("profile_id and git_commit_hash are required"))
	}
	verdicts, err := h.repo.List(ctx, req.Msg.ProfileId, req.Msg.GitCommitHash, 1000)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ready := len(verdicts) > 0
	reason := ""
	for _, verdict := range verdicts {
		if verdict.Disposition != commonv1.Disposition_DISPOSITION_PASSED {
			ready = false
			reason = "one_or_more_targets_not_passed"
			break
		}
	}
	if len(verdicts) == 0 {
		reason = "no_target_evidence"
	}
	return connect.NewResponse(&evidencev1.EvidenceReview{ProfileId: req.Msg.ProfileId, GitCommitHash: req.Msg.GitCommitHash, Verdicts: verdicts, Ready: ready, Reason: reason}), nil
}
