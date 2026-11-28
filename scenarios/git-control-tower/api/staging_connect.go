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

func (s *Server) stageFilesConnect(ctx context.Context, req *repov1.StageFilesRequest) (*repov1.StageFilesResponse, error) {
	result, repoPath, err := s.runStagingMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.stage", "stage files", func(writeCtx context.Context, deps StagingDeps) (any, error) {
		return StageFiles(writeCtx, deps, StageRequest{Paths: req.GetPaths(), Scope: req.GetScope()})
	})
	if err != nil {
		return nil, err
	}
	stageResult, _ := result.(*StageResponse)
	if stageResult != nil {
		s.logStagingAudit(repoPath, AuditOpStage, req.GetPaths(), stageResult.Staged, stageResult.Success, stageResult.Errors)
	}
	return stageResponseProto(stageResult), nil
}

func (s *Server) unstageFilesConnect(ctx context.Context, req *repov1.UnstageFilesRequest) (*repov1.UnstageFilesResponse, error) {
	result, repoPath, err := s.runStagingMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.unstage", "unstage files", func(writeCtx context.Context, deps StagingDeps) (any, error) {
		return UnstageFiles(writeCtx, deps, UnstageRequest{Paths: req.GetPaths(), Scope: req.GetScope()})
	})
	if err != nil {
		return nil, err
	}
	unstageResult, _ := result.(*UnstageResponse)
	if unstageResult != nil {
		s.logStagingAudit(repoPath, AuditOpUnstage, req.GetPaths(), unstageResult.Unstaged, unstageResult.Success, unstageResult.Errors)
	}
	return unstageResponseProto(unstageResult), nil
}

func (s *Server) runStagingMutation(ctx context.Context, repositoryID, intentID, operation, writerName string, run func(context.Context, StagingDeps) (any, error)) (any, string, error) {
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, errors.New("authenticate through the configured provider before changing the index"))
	}
	if principal.Kind != cliutil.CallerKindHuman {
		return nil, "", connect.NewError(connect.CodePermissionDenied, errors.New("agent callers cannot change the repository index"))
	}
	if strings.TrimSpace(intentID) == "" {
		return nil, "", connect.NewError(connect.CodePermissionDenied, errors.New("review the exact index change and confirm it before staging files"))
	}

	writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	repo, err := s.mutationRepo(writeCtx, repositoryID)
	if err != nil {
		return nil, "", connect.NewError(connect.CodeFailedPrecondition, err)
	}
	cleanStaleLock(repo.Path)
	unlock, err := s.repoLock.Acquire(writeCtx, repo.Path)
	if err != nil {
		return nil, "", connect.NewError(connect.CodeUnavailable, errors.New("repository is busy, please retry"))
	}
	defer func() {
		unlock()
		s.repos.InvalidateStatus(repo.Path)
	}()

	preview, err := s.prepareMutation(writeCtx, repositoryIDFor(repo), operation)
	if err != nil {
		return nil, "", connect.NewError(connect.CodeFailedPrecondition, err)
	}
	consumedIntent, err := s.intentService.Consume(writeCtx, principal, intentID, preview.RepositoryID, operation, preview.ExpectedRevision, preview.SubjectDigest)
	if err != nil {
		return nil, "", connect.NewError(connect.CodeAborted, err)
	}
	writeCtx = policygate.WithIntent(writeCtx, consumedIntent.AsHumanIntent())

	result, err := run(writeCtx, StagingDeps{Git: s.git, RepoDir: repo.Path})
	if err != nil {
		return nil, repo.Path, connect.NewError(connect.CodeInternal, errors.New(writerName+": "+err.Error()))
	}
	return result, repo.Path, nil
}

func (s *Server) logStagingAudit(repoPath string, operation AuditOperation, requested, affected []string, success bool, errs []string) {
	entry := AuditEntry{Operation: operation, RepoDir: repoPath, Paths: append([]string(nil), requested...), Success: success, Timestamp: time.Now().UTC()}
	if len(affected) > 0 {
		entry.Paths = append([]string(nil), affected...)
	}
	if len(errs) > 0 {
		entry.Error = strings.Join(errs, "; ")
	}
	go func() {
		logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.audit.Log(logCtx, entry)
	}()
}

func stageResponseProto(value *StageResponse) *repov1.StageFilesResponse {
	if value == nil {
		return &repov1.StageFilesResponse{}
	}
	return &repov1.StageFilesResponse{
		Success: value.Success, Staged: value.Staged, Failed: value.Failed, Errors: value.Errors,
		Warnings: value.Warnings, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano),
	}
}

func unstageResponseProto(value *UnstageResponse) *repov1.UnstageFilesResponse {
	if value == nil {
		return &repov1.UnstageFilesResponse{}
	}
	return &repov1.UnstageFilesResponse{
		Success: value.Success, Unstaged: value.Unstaged, Failed: value.Failed, Errors: value.Errors,
		Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano),
	}
}
