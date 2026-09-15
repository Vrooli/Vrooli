package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	"git-control-tower/internal/policygate"
	"github.com/vrooli/cli-core/cliutil"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

// createCommitConnect is the single typed commit application boundary. The
// generated RepoService handler is only a transport adapter; all repository
// resolution, intent consumption, locking, audit, and Git execution remain in
// this scenario-owned domain boundary.
func (s *Server) createCommitConnect(ctx context.Context, req *repov1.CreateCommitRequest) (*repov1.CreateCommitResponse, error) {
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticate through the configured provider before committing"))
	}
	if principal.Kind != cliutil.CallerKindHuman {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("agent callers cannot commit repositories"))
	}
	if strings.TrimSpace(req.GetIntentId()) == "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("review the exact staged change and confirm it before committing"))
	}

	writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	repo, err := s.mutationRepo(writeCtx, req.GetRepositoryId())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	cleanStaleLock(repo.Path)
	unlock, err := s.repoLock.Acquire(writeCtx, repo.Path)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("repository is busy, please retry"))
	}
	defer func() {
		unlock()
		s.repos.InvalidateStatus(repo.Path)
	}()

	preview, err := s.prepareMutation(writeCtx, repositoryIDFor(repo), mutationOperationCommit)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	consumedIntent, err := s.intentService.Consume(writeCtx, principal, req.GetIntentId(), preview.RepositoryID, mutationOperationCommit, preview.ExpectedRevision, preview.SubjectDigest)
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	writeCtx = policygate.WithIntent(writeCtx, consumedIntent.AsHumanIntent())

	request := CommitRequest{
		IntentID:             req.GetIntentId(),
		Message:              req.GetMessage(),
		ValidateConventional: req.GetValidateConventional(),
		Amend:                req.GetAmend(),
		AuthorName:           req.GetAuthorName(),
		AuthorEmail:          req.GetAuthorEmail(),
		SkipPrecommitOnce:    req.GetSkipPrecommitOnce(),
	}
	stagedFiles, _ := s.git.ListStagedFiles(writeCtx, repo.Path)
	result, err := CreateCommit(writeCtx, CommitDeps{
		Git:       s.git,
		RepoDir:   repo.Path,
		Precommit: s.precommit,
		Checks:    s.commitChecks,
	}, request)
	s.logCommitAudit(repo.Path, request, result, err)
	s.notifyCommitToSandbox(repo.Path, stagedFiles, result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return commitResponseProto(result), nil
}

func commitResponseProto(result *CommitResponse) *repov1.CreateCommitResponse {
	if result == nil {
		return &repov1.CreateCommitResponse{}
	}
	response := &repov1.CreateCommitResponse{
		Success:          result.Success,
		Hash:             result.Hash,
		Message:          result.Message,
		Amended:          result.Amended,
		ValidationErrors: append([]string(nil), result.ValidationErrors...),
		Error:            result.Error,
		Timestamp:        result.Timestamp.UTC().Format(time.RFC3339Nano),
	}
	if result.Precommit != nil {
		response.Precommit = &repov1.PrecommitRunResult{
			Status:          result.Precommit.Status,
			Command:         result.Precommit.Command,
			ExitCode:        int32(result.Precommit.ExitCode),
			Summary:         result.Precommit.Summary,
			Stdout:          result.Precommit.Stdout,
			Stderr:          result.Precommit.Stderr,
			DurationMs:      result.Precommit.DurationMs,
			OverrideAllowed: result.Precommit.OverrideAllowed,
			Timestamp:       result.Precommit.Timestamp.UTC().Format(time.RFC3339Nano),
		}
	}
	return response
}
