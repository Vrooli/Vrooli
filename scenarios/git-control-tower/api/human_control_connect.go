package main

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"git-control-tower/internal/policygate"
	"github.com/vrooli/cli-core/cliutil"
	humancontrol "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control"
)

func (s *Server) GetAuthorityStatus(ctx context.Context, _ *connect.Request[humancontrol.GetAuthorityStatusRequest]) (*connect.Response[humancontrol.AuthorityStatus], error) {
	status := authorityStatusFromContext(ctx)
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok {
		return connect.NewResponse(authorityStatusProto(status)), nil
	}
	status.Authenticated = true
	status.PrincipalID, status.Email, status.Realm = principal.Subject, principal.Email, principal.Realm
	status.CallerKind = principal.Kind.String()
	status.CanMutate = principal.Kind == cliutil.CallerKindHuman
	if status.CanMutate {
		status.Capabilities = []string{mutationOperationCommit}
	} else {
		status.Reason = "agent callers may inspect and prepare changes, but cannot mutate repositories"
	}
	return connect.NewResponse(authorityStatusProto(status)), nil
}

func (s *Server) PrepareMutation(ctx context.Context, req *connect.Request[humancontrol.PrepareMutationRequest]) (*connect.Response[humancontrol.MutationPreview], error) {
	preview, err := s.prepareMutationWithContext(ctx, req.Msg.GetRepositoryId(), req.Msg.GetOperation(), req.Msg.GetSubjectContext())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(mutationPreviewProto(preview)), nil
}

func (s *Server) ConfirmMutation(ctx context.Context, req *connect.Request[humancontrol.ConfirmMutationRequest]) (*connect.Response[humancontrol.MutationIntent], error) {
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticate through the configured provider before confirming a mutation"))
	}
	if principal.Kind != cliutil.CallerKindHuman {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("agent callers cannot issue human mutation intents"))
	}
	operation := req.Msg.GetOperation()
	if operation == "" {
		operation = mutationOperationCommit
	}
	preview, err := s.prepareMutationWithContext(ctx, req.Msg.GetRepositoryId(), operation, req.Msg.GetSubjectContext())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if preview.ExpectedRevision != req.Msg.GetExpectedRevision() || preview.SubjectDigest != req.Msg.GetSubjectDigest() {
		return nil, connect.NewError(connect.CodeAborted, errors.New("repository changed; review the exact mutation preview again"))
	}
	stepUpRequired := requiresMutationStepUp(operation)
	if stepUpRequired && !req.Msg.GetStepUpConfirmed() {
		return nil, connect.NewError(connect.CodePermissionDenied, policygate.ErrStepUpRequired)
	}
	intent, err := s.intentService.Issue(ctx, principal, policygate.IntentRequest{
		RepositoryID: preview.RepositoryID, Operation: operation,
		ExpectedRevision: preview.ExpectedRevision, SubjectDigest: preview.SubjectDigest,
		PolicyVersion: "gct-human-control-v1", StepUpRequired: stepUpRequired,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewResponse(&humancontrol.MutationIntent{
		IntentId: intent.ID, PrincipalId: intent.PrincipalID, RepositoryId: intent.RepositoryID,
		Operation: intent.Operation, ExpectedRevision: intent.ExpectedRevision, SubjectDigest: intent.SubjectDigest,
		ExpiresAt: timestamppb.New(intent.ExpiresAt), SingleUse: true, StepUpRequired: intent.StepUpRequired,
	}), nil
}

func authorityStatusProto(status AuthorityStatusResponse) *humancontrol.AuthorityStatus {
	return &humancontrol.AuthorityStatus{Authenticated: status.Authenticated, PrincipalId: status.PrincipalID, Email: status.Email, Realm: status.Realm, CallerKind: status.CallerKind, CanMutate: status.CanMutate, Reason: status.Reason, Capabilities: status.Capabilities, AuthSource: status.AuthSource, AuthState: status.AuthState, RecoveryUrl: status.RecoveryURL, FailureClass: status.FailureClass, AuthSources: status.AuthSources}
}

func mutationPreviewProto(preview MutationPreviewResponse) *humancontrol.MutationPreview {
	return &humancontrol.MutationPreview{RepositoryId: preview.RepositoryID, RepositoryPath: preview.RepositoryPath, Operation: preview.Operation, Branch: preview.Branch, ExpectedRevision: preview.ExpectedRevision, SubjectDigest: preview.SubjectDigest, StagedFiles: preview.StagedFiles, FileCount: int32(preview.FileCount), GeneratedAt: timestamppb.New(preview.GeneratedAt), SubjectContext: preview.SubjectContext}
}
