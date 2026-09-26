package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func (s *Server) getBlameConnect(ctx context.Context, req *repov1.GetBlameRequest) (*repov1.GetBlameResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	runner, ok := s.git.(BlameRunner)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("configured git runner does not support blame"))
	}
	result, err := ReadBlame(ctx, runner, BlameRequest{
		RepoDir: resolved.Path, Revision: req.GetRevision(), Paths: req.GetPaths(),
		StartLine: int(req.GetStartLine()), EndLine: int(req.GetEndLine()),
		MaxPaths: int(req.GetMaxPaths()), MaxLines: int(req.GetMaxLines()), MaxBytes: int(req.GetMaxBytes()),
		Enrich: req.GetEnrich(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.GetEnrich() {
		s.enrichBlame(ctx, resolved.Path, result)
	}
	return blameResponseProto(result), nil
}

func blameResponseProto(result *BlameResponse) *repov1.GetBlameResponse {
	if result == nil {
		return &repov1.GetBlameResponse{}
	}
	response := &repov1.GetBlameResponse{Revision: result.Revision, Truncated: result.Truncated, Warnings: append([]string(nil), result.Warnings...)}
	for _, file := range result.Files {
		out := &repov1.BlameFile{Path: file.Path, Status: file.Status, ContentDigest: file.ContentDigest, Reason: file.Reason, Standing: string(file.Standing), DowngradeReasons: append([]string(nil), file.DowngradeReasons...)}
		for _, line := range file.Lines {
			out.Lines = append(out.Lines, &repov1.BlameLine{Line: int32(line.Number), Content: line.Content, Commit: line.Commit, Author: line.Author, AuthorTime: line.AuthorTime, Subject: line.Subject, OriginalLine: int32(line.OriginalLine), OriginalPath: line.OriginalPath})
		}
		for _, evidence := range file.Evidence {
			fact := &repov1.BlameEvidence{RunId: evidence.RunID, SandboxId: evidence.SandboxID, ApplicationReceipt: evidence.ApplicationReceipt, ContentDigest: evidence.ContentDigest, CommitId: evidence.CommitID, Visibility: evidence.Visibility, CommitState: evidence.CommitState, RunOutcome: evidence.RunOutcome, ConversationId: evidence.ConversationID, CostUsd: evidence.CostUSD, CommittedAt: evidence.CommittedAt, Unavailable: append([]string(nil), evidence.Unavailable...)}
			for _, ref := range evidence.WorkReferences {
				fact.WorkReferences = append(fact.WorkReferences, &repov1.ProvenanceWorkReference{Kind: ref.Kind, Id: ref.ID, Revision: ref.Revision, Relationship: ref.Relationship, Verified: ref.Verified, Visibility: ref.Visibility, State: ref.State, UnavailableReason: ref.UnavailableReason})
			}
			out.Evidence = append(out.Evidence, fact)
		}
		response.Files = append(response.Files, out)
	}
	for _, bundle := range result.Bundles {
		out := &repov1.ProvenanceChangeBundle{RunId: bundle.RunID, SandboxId: bundle.SandboxID, Files: append([]string(nil), bundle.Files...), RunOutcome: bundle.RunOutcome, ConversationId: bundle.ConversationID, CostUsd: bundle.CostUSD, Gaps: append([]string(nil), bundle.Gaps...)}
		for _, ref := range bundle.WorkReferences {
			out.WorkReferences = append(out.WorkReferences, &repov1.ProvenanceWorkReference{Kind: ref.Kind, Id: ref.ID, Revision: ref.Revision, Relationship: ref.Relationship, Verified: ref.Verified, Visibility: ref.Visibility, State: ref.State, UnavailableReason: ref.UnavailableReason})
		}
		response.ChangeBundles = append(response.ChangeBundles, out)
	}
	return response
}

func (s *Server) getRepoHistoryConnect(ctx context.Context, req *repov1.GetRepoHistoryRequest) (*repov1.GetRepoHistoryResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 30
	}
	limit = capHistoryLimit(limit, req.GetGrepPattern(), req.GetLimit() > 0)
	history, err := GetRepoHistory(ctx, RepoHistoryDeps{
		Git: s.git, RepoDir: resolved.Path, Limit: limit,
		IncludeFiles: req.GetIncludeFiles(), IncludeChecks: req.GetIncludeChecks(),
		CommitChecks: s.commitChecks, GrepPattern: strings.TrimSpace(req.GetGrepPattern()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	entries := make([]*repov1.RepoHistoryEntry, 0, len(history.Entries))
	for _, entry := range history.Entries {
		checks := make([]*repov1.CommitCheckRun, 0, len(entry.Checks))
		for _, check := range entry.Checks {
			checks = append(checks, &repov1.CommitCheckRun{
				Kind: string(check.Kind), Status: string(check.Status), Command: check.Command,
				ExitCode: int32(check.ExitCode), Summary: check.Summary, Stdout: check.Stdout,
				Stderr: check.Stderr, DurationMs: check.DurationMs,
				Timestamp: check.Timestamp.UTC().Format(time.RFC3339Nano),
			})
		}
		entries = append(entries, &repov1.RepoHistoryEntry{
			Hash: entry.Hash, Author: entry.Author, Date: entry.Date, Subject: entry.Subject,
			Files: entry.Files, Checks: checks,
		})
	}
	return &repov1.GetRepoHistoryResponse{
		RepoDir: history.RepoDir, Lines: history.Lines, Entries: entries,
		Limit: int32(history.Limit), GrepPattern: history.GrepPattern,
		Timestamp: history.Timestamp.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) getApprovedChangesConnect(ctx context.Context, req *repov1.GetApprovedChangesRequest) (*repov1.GetApprovedChangesResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !s.capabilities.IsAvailable(ctx, "workspace-sandbox") {
		return &repov1.GetApprovedChangesResponse{Warning: "Workspace Sandbox is not running"}, nil
	}
	var preview *workspaceSandboxCommitPreview
	if len(req.GetPaths()) == 0 {
		preview, err = s.sandbox.GetCommitPreview(ctx, resolved.Path)
	} else {
		preview, err = s.sandbox.GetCommitPreviewForPaths(ctx, resolved.Path, req.GetPaths())
	}
	if err != nil {
		return &repov1.GetApprovedChangesResponse{Warning: err.Error()}, nil
	}
	return approvedChangesProto(preview), nil
}

func (s *Server) getProvenanceConnect(ctx context.Context, req *repov1.GetProvenanceRequest) (*repov1.GetProvenanceResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !s.capabilities.IsAvailable(ctx, "workspace-sandbox") {
		return &repov1.GetProvenanceResponse{Warning: "Workspace Sandbox is not running"}, nil
	}
	result, err := s.sandbox.GetProvenanceByRun(ctx, resolved.Path)
	if err != nil {
		return &repov1.GetProvenanceResponse{Warning: err.Error()}, nil
	}
	groups := make([]*repov1.ProvenanceRunGroup, 0, len(result.RunGroups))
	for _, group := range result.RunGroups {
		files := make([]*repov1.ProvenanceFile, 0, len(group.Files))
		for _, file := range group.Files {
			files = append(files, &repov1.ProvenanceFile{
				FilePath: file.FilePath, RelativePath: file.RelativePath, ChangeType: file.ChangeType,
				AppliedAt: file.AppliedAt, Visibility: file.Visibility,
			})
		}
		groups = append(groups, &repov1.ProvenanceRunGroup{
			RunId: group.RunID, SandboxId: group.SandboxID, SandboxOwner: group.SandboxOwner,
			Files: files, LatestAppliedAt: group.LatestAppliedAt,
		})
	}
	return &repov1.GetProvenanceResponse{Available: true, RunGroups: groups}, nil
}

func (s *Server) searchProvenanceConnect(ctx context.Context, req *repov1.SearchProvenanceRequest) (*repov1.SearchProvenanceResponse, error) {
	if strings.TrimSpace(req.GetQuery()) == "" || req.GetRepositoryId() <= 0 {
		return &repov1.SearchProvenanceResponse{Warning: "query and explicit repo scope are required"}, nil
	}
	resolved, err := s.resolveRepoForConnect(ctx, fmt.Sprintf("%d", req.GetRepositoryId()), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !s.capabilities.IsAvailable(ctx, "workspace-sandbox") {
		return &repov1.SearchProvenanceResponse{Warning: "Workspace Sandbox is not running"}, nil
	}
	result, err := s.sandbox.GetProvenanceByRun(ctx, resolved.Path)
	if err != nil {
		return &repov1.SearchProvenanceResponse{Warning: err.Error()}, nil
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	hits := rankProvenanceSearch(result.RunGroups, req.GetQuery(), limit)
	protoHits := make([]*repov1.ProvenanceSearchHit, 0, len(hits))
	for _, hit := range hits {
		protoHits = append(protoHits, &repov1.ProvenanceSearchHit{
			Id: hit.ID, Title: hit.Title, Snippet: hit.Snippet, Score: hit.Score,
			RunId: hit.RunID, SandboxId: hit.SandboxID, RelativePath: hit.RelativePath,
			EvidenceStanding: hit.EvidenceStanding,
		})
	}
	return &repov1.SearchProvenanceResponse{Available: true, Results: protoHits}, nil
}

func approvedChangesProto(preview *workspaceSandboxCommitPreview) *repov1.GetApprovedChangesResponse {
	if preview == nil {
		return &repov1.GetApprovedChangesResponse{}
	}
	files := make([]*repov1.ApprovedChangeFile, 0, len(preview.Files))
	for _, file := range preview.Files {
		files = append(files, &repov1.ApprovedChangeFile{
			RelativePath: file.RelativePath, Status: file.Status, SandboxId: file.SandboxID,
			SandboxOwner: file.SandboxOwner, ChangeType: file.ChangeType,
			AgentManagerRunId: file.AgentManagerRunID,
		})
	}
	return &repov1.GetApprovedChangesResponse{
		Available: true, CommittableFiles: int32(preview.CommittableFiles),
		SuggestedMessage: preview.SuggestedMessage, Files: files,
	}
}

// normalizeApprovedChanges remains the domain-level normalization seam used
// by focused tests and by the typed adapter. It does not own transport.
func normalizeApprovedChanges(preview *workspaceSandboxCommitPreview) ApprovedChangesResponse {
	if preview == nil {
		return ApprovedChangesResponse{Available: false}
	}
	files := make([]ApprovedChangeFile, 0, len(preview.Files))
	for _, file := range preview.Files {
		files = append(files, ApprovedChangeFile{
			RelativePath: file.RelativePath, Status: file.Status, SandboxID: file.SandboxID,
			SandboxOwner: file.SandboxOwner, ChangeType: file.ChangeType,
			AgentManagerRunID: file.AgentManagerRunID,
		})
	}
	return ApprovedChangesResponse{
		Available: true, CommittableFiles: preview.CommittableFiles,
		SuggestedMessage: preview.SuggestedMessage, Files: files,
	}
}
