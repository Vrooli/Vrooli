package bountycases

import (
	"client-rehearsal/internal/bountycases"
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/bountycases"
)

type ConnectHandler struct{ service bountycases.Service }

func NewConnectHandler(service bountycases.Service) *ConnectHandler {
	if service == nil {
		panic("bountycases.NewConnectHandler requires service")
	}
	return &ConnectHandler{service: service}
}

func (h *ConnectHandler) ResolveBountyReplayCase(ctx context.Context, req *connect.Request[v1.ResolveBountyReplayCaseRequest]) (*connect.Response[v1.ResolveBountyReplayCaseResponse], error) {
	resolved, err := h.service.Resolve(ctx, bountycases.Lookup{CaseID: req.Msg.GetCaseId(), PublicURL: req.Msg.GetPublicUrl()})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.ResolveBountyReplayCaseResponse{Case: toProto(resolved)}), nil
}

func (h *ConnectHandler) GetFrozenEvaluatorBundle(ctx context.Context, req *connect.Request[v1.GetFrozenEvaluatorBundleRequest]) (*connect.Response[v1.GetFrozenEvaluatorBundleResponse], error) {
	bundle, err := h.service.ResolveEvaluatorBundle(ctx, req.Msg.GetEvaluatorBundleId())
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.GetFrozenEvaluatorBundleResponse{EvaluatorBundle: evaluatorBundleToProto(bundle)}), nil
}

func toConnectError(err error) error {
	var invalid bountycases.ErrInvalid
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var missing bountycases.ErrNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var conflict bountycases.ErrConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, errors.New("bounty case resolution failed"))
}

func toProto(item bountycases.CaseBundle) *v1.BountyReplayCase {
	return &v1.BountyReplayCase{
		SchemaVersion: item.SchemaVersion, BundleVersion: item.BundleVersion, CaseId: item.CaseID,
		Project: item.Project, Repository: item.Repository, Title: item.Title, Task: item.Task,
		PublicUrl: item.PublicURL, IssueNumber: item.IssueNumber, PullRequestUrl: item.PullRequestURL,
		PullRequestNumber: item.PullRequestNumber, SourceCommit: item.SourceCommit, ReferenceCommit: item.ReferenceCommit,
		SourceProjectRoot: item.SourceProjectRoot, ReferenceProjectRoot: item.ReferenceProjectRoot,
		FrozenEvaluatorDigest: item.FrozenEvaluatorDigest, FrozenEvaluatorRoot: item.FrozenEvaluatorRoot,
		TestGenieProjectId: item.TestGenieProjectID,
		TestGeniePlan: &v1.BountyTestGeniePlan{
			ValidationPurpose: item.TestGeniePlan.ValidationPurpose, RequiredStrength: item.TestGeniePlan.RequiredStrength, Phases: append([]string(nil), item.TestGeniePlan.Phases...),
			EvidencePolicy: &v1.BountyReplayEvidencePolicy{RequireBehavioralBefore: item.TestGeniePlan.EvidencePolicy.RequireBehavioralBefore, RequireSourceSnapshot: item.TestGeniePlan.EvidencePolicy.RequireSourceSnapshot, AllowAuthorizedDegradation: item.TestGeniePlan.EvidencePolicy.AllowAuthorizedDegradation, RequiredEvidenceKinds: append([]string(nil), item.TestGeniePlan.EvidencePolicy.RequiredEvidenceKinds...)},
			DeadlinePolicy: &v1.BountyReplayDeadlinePolicy{QueueBudgetMilliseconds: item.TestGeniePlan.DeadlinePolicy.QueueBudgetMilliseconds, ExecutionBudgetMilliseconds: item.TestGeniePlan.DeadlinePolicy.ExecutionBudgetMilliseconds, MaximumAttempts: item.TestGeniePlan.DeadlinePolicy.MaximumAttempts},
		},
		BundleDigest: item.BundleDigest, FrozenEvaluatorEntrypoint: item.FrozenEvaluatorEntrypoint,
		FrozenEvaluatorBundleDigest: item.FrozenEvaluatorBundleDigest, EvaluatorBundleId: evaluatorBundleID(item),
	}
}

func evaluatorBundleToProto(item bountycases.FrozenEvaluatorBundle) *v1.FrozenEvaluatorBundle {
	out := &v1.FrozenEvaluatorBundle{EvaluatorBundleId: item.EvaluatorBundleID, CaseId: item.CaseID, BundleVersion: item.BundleVersion, Entrypoint: item.Entrypoint, EvaluatorDigest: item.EvaluatorDigest, ContentDigest: item.ContentDigest, Files: make([]*v1.FrozenEvaluatorFile, 0, len(item.Files))}
	for _, file := range item.Files {
		out.Files = append(out.Files, &v1.FrozenEvaluatorFile{RelativePath: file.RelativePath, Content: append([]byte(nil), file.Content...), Sha256: file.SHA256})
	}
	return out
}

func evaluatorBundleID(item bountycases.CaseBundle) string {
	return fmt.Sprintf("%s@%d:%s", item.CaseID, item.BundleVersion, item.BundleDigest)
}
