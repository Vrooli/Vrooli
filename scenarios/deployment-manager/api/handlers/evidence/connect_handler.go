package evidencehandler

import (
	"context"
	"fmt"

	internalEvidence "deployment-manager/internal/evidence"
	"deployment-manager/internal/evidence/conformance"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
)

type ConnectHandler struct {
	evidenceconnect.UnimplementedEvidenceServiceHandler
	repo       internalEvidence.Repository
	authorize  func(context.Context) error
	readAuth   func(context.Context) error
	targetAuth func(context.Context, *commonv1.EvidenceTarget) error
}

func NewConnectHandler(repo internalEvidence.Repository) *ConnectHandler {
	return &ConnectHandler{repo: repo}
}

// WithAuthorization installs the verified producer boundary for evidence
// writes. The handler fails closed when production wiring omits the boundary.
func (h *ConnectHandler) WithAuthorization(authorize func(context.Context) error) *ConnectHandler {
	h.authorize = authorize
	return h
}

// WithTargetAuthorization binds the authenticated producer to the submitted
// target identity after the evidence contract has been validated.
func (h *ConnectHandler) WithTargetAuthorization(authorize func(context.Context, *commonv1.EvidenceTarget) error) *ConnectHandler {
	h.targetAuth = authorize
	return h
}

// WithReadAuthorization installs the authenticated reviewer boundary for
// evidence retrieval. Evidence is release input and is not public state.
func (h *ConnectHandler) WithReadAuthorization(authorize func(context.Context) error) *ConnectHandler {
	h.readAuth = authorize
	return h
}

func (h *ConnectHandler) ReportTargetVerdict(ctx context.Context, req *connect.Request[evidencev1.ReportTargetVerdictRequest]) (*connect.Response[evidencev1.ReportTargetVerdictResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if h.authorize == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("evidence authorization is not configured"))
	}
	if err := h.authorize(ctx); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("verified evidence producer is required: %w", err))
	}
	if violations := conformance.Validate(req.Msg.Verdict); len(violations) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, violations[0])
	}
	if h.targetAuth != nil {
		if err := h.targetAuth(ctx, req.Msg.Verdict.Target); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("evidence producer is not authorized for target ramp: %w", err))
		}
	}
	if err := h.repo.Save(ctx, req.Msg.ProfileId, req.Msg.GitCommitHash, req.Msg.Verdict); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&evidencev1.ReportTargetVerdictResponse{Verdict: req.Msg.Verdict}), nil
}

func (h *ConnectHandler) ListTargetVerdicts(ctx context.Context, req *connect.Request[evidencev1.ListTargetVerdictsRequest]) (*connect.Response[evidencev1.ListTargetVerdictsResponse], error) {
	if err := h.requireRead(ctx); err != nil {
		return nil, err
	}
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
	if err := h.requireRead(ctx); err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil || req.Msg.ProfileId == "" || req.Msg.GitCommitHash == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("profile_id and git_commit_hash are required"))
	}
	verdicts, err := h.repo.List(ctx, req.Msg.ProfileId, req.Msg.GitCommitHash, 1000)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ready := len(verdicts) > 0
	reason := ""
	seenTargets := make(map[string]struct{}, len(verdicts))
	for _, verdict := range verdicts {
		if verdict == nil {
			ready = false
			reason = "target_evidence_invalid"
			break
		}
		if verdict.Target == nil {
			ready = false
			reason = "target_evidence_invalid"
			break
		}
		// Bridge job IDs identify an execution attempt, not a distinct target.
		// Exclude them so a second passing attempt cannot make one target look
		// like two independent target reports.
		targetKey := fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%s", verdict.Target.GetRamp(), verdict.Target.GetPlatform(), verdict.Target.GetOs(), verdict.Target.GetDeviceKind(), verdict.Target.GetBridgeNodeId())
		if _, duplicate := seenTargets[targetKey]; duplicate {
			ready = false
			reason = "duplicate_target_evidence"
			break
		}
		seenTargets[targetKey] = struct{}{}
		if verdict.Disposition != commonv1.Disposition_DISPOSITION_PASSED {
			ready = false
			reason = "one_or_more_targets_not_passed"
			break
		}
		if violations := conformance.Validate(verdict); len(violations) > 0 {
			ready = false
			reason = "target_evidence_invalid"
			break
		}
	}
	if len(verdicts) == 0 {
		reason = "no_target_evidence"
	}
	return connect.NewResponse(&evidencev1.EvidenceReview{ProfileId: req.Msg.ProfileId, GitCommitHash: req.Msg.GitCommitHash, Verdicts: verdicts, Ready: ready, Reason: reason}), nil
}

func (h *ConnectHandler) requireRead(ctx context.Context) error {
	if h.readAuth == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("evidence read authorization is not configured"))
	}
	if err := h.readAuth(ctx); err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("verified evidence reader is required: %w", err))
	}
	return nil
}
