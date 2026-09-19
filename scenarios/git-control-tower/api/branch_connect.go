package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"git-control-tower/internal/policygate"
	"github.com/vrooli/cli-core/cliutil"
	branchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/branch"
)

func (s *Server) listBranchesConnect(ctx context.Context, req *branchv1.ListBranchesRequest) (*branchv1.ListBranchesResponse, error) {
	repo, err := s.mutationRepo(ctx, req.GetRepositoryId())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := ListBranches(readCtx, BranchDeps{Git: s.git, RepoDir: repo.Path})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	enrichBranchesWithWorktreeClaims(readCtx, result, repo.Path)
	return branchListResponseProto(result), nil
}

func (s *Server) createBranchConnect(ctx context.Context, req *branchv1.CreateBranchRequest) (*branchv1.CreateBranchResponse, error) {
	result, repoPath, err := s.runBranchMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.branch.create", "create branch", func(writeCtx context.Context, deps BranchDeps) (any, error) {
		return CreateBranch(writeCtx, deps, CreateBranchRequest{
			Name: req.GetName(), From: req.GetFrom(), Checkout: req.GetCheckout(), AllowDirty: req.GetAllowDirty(),
		})
	})
	if err != nil {
		return nil, err
	}
	branchResult, _ := result.(*BranchCreateResponse)
	logBranchAudit(s, repoPath, AuditOpBranchCreate, req.GetName(), branchResult != nil && branchResult.Success, nil)
	return branchCreateResponseProto(branchResult), nil
}

func (s *Server) switchBranchConnect(ctx context.Context, req *branchv1.SwitchBranchRequest) (*branchv1.SwitchBranchResponse, error) {
	result, repoPath, err := s.runBranchMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.branch.switch", "switch branch", func(writeCtx context.Context, deps BranchDeps) (any, error) {
		return SwitchBranch(writeCtx, deps, SwitchBranchRequest{
			Name: req.GetName(), AllowDirty: req.GetAllowDirty(), TrackRemote: req.GetTrackRemote(),
		})
	})
	if err != nil {
		return nil, err
	}
	branchResult, _ := result.(*BranchSwitchResponse)
	logBranchAudit(s, repoPath, AuditOpBranchSwitch, req.GetName(), branchResult != nil && branchResult.Success, nil)
	return branchSwitchResponseProto(branchResult), nil
}

func (s *Server) publishBranchConnect(ctx context.Context, req *branchv1.PublishBranchRequest) (*branchv1.PublishBranchResponse, error) {
	result, repoPath, err := s.runBranchMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.branch.publish", "publish branch", func(writeCtx context.Context, deps BranchDeps) (any, error) {
		return PublishBranch(writeCtx, deps, PublishBranchRequest{
			Remote: req.GetRemote(), Branch: req.GetBranch(), SetUpstream: req.GetSetUpstream(), Fetch: req.GetFetch(),
		})
	})
	if err != nil {
		return nil, err
	}
	branchResult, _ := result.(*BranchPublishResponse)
	branchName := req.GetBranch()
	if branchName == "" && branchResult != nil {
		branchName = branchResult.Branch
	}
	logBranchAudit(s, repoPath, AuditOpBranchPublish, branchName, branchResult != nil && branchResult.Success, nil)
	return branchPublishResponseProto(branchResult), nil
}

func (s *Server) runBranchMutation(ctx context.Context, repositoryID, intentID, operation, writerName string, run func(context.Context, BranchDeps) (any, error)) (any, string, error) {
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, errors.New("authenticate through the configured provider before mutating branches"))
	}
	if principal.Kind != cliutil.CallerKindHuman {
		return nil, "", connect.NewError(connect.CodePermissionDenied, errors.New("agent callers cannot mutate branches"))
	}
	if strings.TrimSpace(intentID) == "" {
		return nil, "", connect.NewError(connect.CodePermissionDenied, errors.New("review the exact branch change and confirm it before mutating branches"))
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

	result, err := run(writeCtx, BranchDeps{Git: s.git, RepoDir: repo.Path})
	if err != nil {
		return nil, repo.Path, connect.NewError(connect.CodeInternal, errors.New(writerName+": "+err.Error()))
	}
	return result, repo.Path, nil
}

func branchListResponseProto(result *RepoBranchesResponse) *branchv1.ListBranchesResponse {
	if result == nil {
		return &branchv1.ListBranchesResponse{}
	}
	response := &branchv1.ListBranchesResponse{
		Current:   result.Current,
		Timestamp: branchTimestamp(result.Timestamp),
		Locals:    make([]*branchv1.BranchInfo, 0, len(result.Locals)),
		Remotes:   make([]*branchv1.BranchInfo, 0, len(result.Remotes)),
	}
	for _, branch := range result.Locals {
		response.Locals = append(response.Locals, branchInfoProto(branch))
	}
	for _, branch := range result.Remotes {
		response.Remotes = append(response.Remotes, branchInfoProto(branch))
	}
	return response
}

func branchInfoProto(value BranchInfo) *branchv1.BranchInfo {
	return &branchv1.BranchInfo{
		Name: value.Name, Upstream: value.Upstream, Oid: value.OID,
		LastCommitAt: branchTimestamp(value.LastCommitAt), Ahead: int32(value.Ahead), Behind: int32(value.Behind),
		IsCurrent: value.IsCurrent, CheckedOutInWorktree: value.CheckedOutInWorktree,
	}
}

func branchWarningProto(value *BranchWarning) *branchv1.BranchWarning {
	if value == nil {
		return nil
	}
	warning := &branchv1.BranchWarning{
		Message: value.Message, RequiresConfirmation: value.RequiresConfirmation,
		RequiresTracking: value.RequiresTracking, RequiresFetch: value.RequiresFetch,
	}
	if value.DirtySummary != nil {
		warning.DirtySummary = &branchv1.DirtySummary{
			Staged: int32(value.DirtySummary.Staged), Unstaged: int32(value.DirtySummary.Unstaged),
			Untracked: int32(value.DirtySummary.Untracked), Conflicts: int32(value.DirtySummary.Conflicts),
		}
	}
	return warning
}

func branchTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

func branchCreateResponseProto(value *BranchCreateResponse) *branchv1.CreateBranchResponse {
	if value == nil {
		return &branchv1.CreateBranchResponse{}
	}
	response := &branchv1.CreateBranchResponse{
		Success: value.Success, Error: value.Error, ValidationErrors: append([]string(nil), value.ValidationErrors...),
		Timestamp: branchTimestamp(value.Timestamp), Warning: branchWarningProto(value.Warning),
	}
	if value.Branch != nil {
		response.Branch = branchInfoProto(*value.Branch)
	}
	return response
}

func branchSwitchResponseProto(value *BranchSwitchResponse) *branchv1.SwitchBranchResponse {
	if value == nil {
		return &branchv1.SwitchBranchResponse{}
	}
	response := &branchv1.SwitchBranchResponse{Success: value.Success, Error: value.Error, Timestamp: branchTimestamp(value.Timestamp), Warning: branchWarningProto(value.Warning)}
	if value.Branch != nil {
		response.Branch = branchInfoProto(*value.Branch)
	}
	return response
}

func branchPublishResponseProto(value *BranchPublishResponse) *branchv1.PublishBranchResponse {
	if value == nil {
		return &branchv1.PublishBranchResponse{}
	}
	return &branchv1.PublishBranchResponse{Success: value.Success, Remote: value.Remote, Branch: value.Branch, Error: value.Error, Timestamp: branchTimestamp(value.Timestamp), Warning: branchWarningProto(value.Warning)}
}
