package main

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	repodomain "git-control-tower/internal/repo"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func (s *Server) getRepoStatusConnect(ctx context.Context, req *repov1.GetRepoStatusRequest) (*repov1.GetRepoStatusResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), req.GetRepoPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:             s.git,
		RepoDir:         resolved.Path,
		ConfigCache:     s.configCache,
		StatusCache:     s.statusCache,
		IncludeHotspots: req.GetIncludeHotspots(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	identity, err := repodomain.NewService(newWorktreeInspector()).GetRepoStatus(ctx, resolved.Path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &repov1.GetRepoStatusResponse{
		Branch:   status.Branch.Head,
		Detached: status.Branch.Head == "" || identity.Detached,
		Worktree: &repov1.WorktreeIdentity{
			IsLinkedWorktree:    identity.Identity.IsLinkedWorktree,
			CommonRepoRoot:      identity.Identity.CommonRepoRoot,
			WorktreeName:        identity.Identity.WorktreeName,
			WorktreeHead:        identity.Identity.WorktreeHead,
			LinkedWorktreeCount: int32(identity.Identity.LinkedWorktreeCount),
		},
		BranchStatus: &repov1.BranchStatus{
			Head:     status.Branch.Head,
			Upstream: status.Branch.Upstream,
			Ahead:    int32(status.Branch.Ahead),
			Behind:   int32(status.Branch.Behind),
			Oid:      status.Branch.OID,
		},
		Files:        repoFilesStatusProto(status.Files),
		FileStats:    repoFileStatsProto(status.FileStats),
		FileHotspots: intMapProto(status.FileHotspots),
		Scopes:       repoScopesProto(status.Scopes),
		Summary: &repov1.StatusSummary{
			Staged:    int32(status.Summary.Staged),
			Unstaged:  int32(status.Summary.Unstaged),
			Untracked: int32(status.Summary.Untracked),
			Conflicts: int32(status.Summary.Conflicts),
			Ignored:   int32(status.Summary.Ignored),
		},
		Author: &repov1.AuthorStatus{
			Name:  status.Author.Name,
			Email: status.Author.Email,
		},
		RepoDir:   status.RepoDir,
		Timestamp: status.Timestamp.Format("2006-01-02T15:04:05.999999999Z07:00"),
	}, nil
}

func (s *Server) resolveRepoForConnect(ctx context.Context, repositoryID, repoPath string) (ResolvedRepo, error) {
	if value := strings.TrimSpace(repositoryID); value != "" {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return ResolvedRepo{}, errors.New("repository_id must be a positive integer")
		}
		repo, err := s.repos.resolveByID(ctx, id)
		if err != nil {
			return ResolvedRepo{}, err
		}
		return ResolvedRepo{ID: repo.ID, Path: repo.Path, Source: "repository_id"}, nil
	}
	if value := strings.TrimSpace(repoPath); value != "" {
		root, err := s.repos.resolveRepoRoot(ctx, value)
		if err != nil {
			return ResolvedRepo{}, err
		}
		return ResolvedRepo{Path: root, Source: "repo_path"}, nil
	}
	if resolved, tried, err := s.repos.resolveFromActive(ctx); tried {
		if err != nil {
			return ResolvedRepo{}, err
		}
		return resolved, nil
	}
	return s.repos.resolveFromRoot(ctx)
}

func repoFilesStatusProto(files RepoFilesStatus) *repov1.FilesStatus {
	return &repov1.FilesStatus{
		Staged:    files.Staged,
		Unstaged:  files.Unstaged,
		Untracked: files.Untracked,
		Conflicts: files.Conflicts,
		Binary:    files.Binary,
		Ignored:   files.Ignored,
		Statuses:  files.Statuses,
		Renames:   files.Renames,
	}
}

func repoFileStatsProto(stats RepoFileStats) *repov1.FileStats {
	return &repov1.FileStats{
		Staged:    diffStatsMapProto(stats.Staged),
		Unstaged:  diffStatsMapProto(stats.Unstaged),
		Untracked: diffStatsMapProto(stats.Untracked),
	}
}

func diffStatsMapProto(stats map[string]DiffStats) map[string]*repov1.DiffStats {
	result := make(map[string]*repov1.DiffStats, len(stats))
	for path, stat := range stats {
		result[path] = &repov1.DiffStats{
			Additions:        int32(stat.Additions),
			Deletions:        int32(stat.Deletions),
			Files:            int32(stat.Files),
			NetLines:         int32(stat.NetLines),
			HunkCount:        int32(stat.HunkCount),
			LargestHunk:      int32(stat.LargestHunk),
			Density:          stat.Density,
			IsBinary:         stat.IsBinary,
			IsRename:         stat.IsRename,
			OldPath:          stat.OldPath,
			CommentAdditions: int32(stat.CommentAdditions),
			CommentDeletions: int32(stat.CommentDeletions),
			IsNewFile:        stat.IsNewFile,
			IsDeletedFile:    stat.IsDeletedFile,
		}
	}
	return result
}

func repoScopesProto(scopes map[string][]string) map[string]*repov1.StringList {
	result := make(map[string]*repov1.StringList, len(scopes))
	for key, values := range scopes {
		result[key] = &repov1.StringList{Values: values}
	}
	return result
}

func intMapProto(values map[string]int) map[string]int32 {
	result := make(map[string]int32, len(values))
	for key, value := range values {
		result[key] = int32(value)
	}
	return result
}
