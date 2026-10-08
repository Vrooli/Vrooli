package main

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

// getRepoDiffConnect adapts the existing repository diff domain service to the
// typed RepoService contract. The domain remains the owner of diff semantics;
// this function only resolves the repository and maps the wire types.
func (s *Server) getRepoDiffConnect(ctx context.Context, req *repov1.GetRepoDiffRequest) (*repov1.GetRepoDiffResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), req.GetRepoPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	mode := ViewMode(req.GetMode())
	switch mode {
	case ViewModeDiff, ViewModeFullDiff, ViewModeSource, ViewModePreview, "":
	default:
		mode = ViewModeDiff
	}

	diff, err := GetDiff(ctx, DiffDeps{Git: s.git, RepoDir: resolved.Path}, DiffRequest{
		Path:      req.GetPath(),
		Staged:    req.GetStaged(),
		Untracked: req.GetUntracked(),
		Base:      req.GetBase(),
		Commit:    req.GetCommit(),
		Mode:      mode,
		Any:       req.GetAny(),
	})
	if err != nil {
		var tooLarge *FileTooLargeError
		if errors.As(err, &tooLarge) {
			return nil, connect.NewError(connect.CodeResourceExhausted, tooLarge)
		}
		var unsupported *UnsupportedBinaryError
		if errors.As(err, &unsupported) {
			return nil, connect.NewError(connect.CodeInvalidArgument, unsupported)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &repov1.GetRepoDiffResponse{
		RepoDir:        diff.RepoDir,
		Path:           diff.Path,
		Staged:         diff.Staged,
		Untracked:      diff.Untracked,
		Base:           diff.Base,
		HasDiff:        diff.HasDiff,
		Hunks:          diffHunksProto(diff.Hunks),
		Stats:          diffStatsProto(diff.Stats),
		Raw:            diff.Raw,
		FullContent:    diff.FullContent,
		ContentHash:    diff.ContentHash,
		AnnotatedLines: annotatedLinesProto(diff.AnnotatedLines),
		Mode:           string(diff.Mode),
		Timestamp:      diff.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}, nil
}

func (s *Server) getRepoGroupsConnect(ctx context.Context, req *repov1.GetRepoGroupsRequest) (*repov1.GetRepoGroupsResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), req.GetRepoPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	configPath, err := s.groupingConfigPath(ctx, resolved.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	config, err := LoadGroupingRules(GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: s.git, RepoDir: resolved.Path})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	groups := ResolveChangeGroups(resolved.Path, status.Files, *config)
	result := make([]*repov1.ChangeGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, &repov1.ChangeGroup{
			Key: group.Key, Kind: group.Kind, Id: group.ID, Label: group.Label,
			Root: group.Root, Source: group.Source, Files: group.Files,
		})
	}
	return &repov1.GetRepoGroupsResponse{Groups: result}, nil
}

func (s *Server) getSyncStatusConnect(ctx context.Context, req *repov1.GetSyncStatusRequest) (*repov1.GetSyncStatusResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), req.GetRepoPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	status, err := GetSyncStatus(ctx, SyncStatusDeps{Git: s.git, RepoDir: resolved.Path, CredStore: s.credStore}, SyncStatusRequest{
		Fetch: req.GetFetch(), Remote: req.GetRemote(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &repov1.GetSyncStatusResponse{
		Branch: status.Branch, Upstream: status.Upstream, RemoteUrl: status.RemoteURL,
		Ahead: int32(status.Ahead), Behind: int32(status.Behind), HasUpstream: status.HasUpstream,
		CanPush: status.CanPush, CanPull: status.CanPull, NeedsPull: status.NeedsPull,
		NeedsPush: status.NeedsPush, HasUncommittedChanges: status.HasUncommittedChanges,
		SafetyWarnings: status.SafetyWarnings, Recommendations: status.Recommendations,
		Fetched: status.Fetched, FetchError: status.FetchError,
		Timestamp: status.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}, nil
}

func diffStatsProto(stats DiffStats) *repov1.DiffStats {
	return &repov1.DiffStats{
		Additions:        int32(stats.Additions),
		Deletions:        int32(stats.Deletions),
		Files:            int32(stats.Files),
		NetLines:         int32(stats.NetLines),
		HunkCount:        int32(stats.HunkCount),
		LargestHunk:      int32(stats.LargestHunk),
		Density:          stats.Density,
		IsBinary:         stats.IsBinary,
		IsRename:         stats.IsRename,
		OldPath:          stats.OldPath,
		CommentAdditions: int32(stats.CommentAdditions),
		CommentDeletions: int32(stats.CommentDeletions),
		IsNewFile:        stats.IsNewFile,
		IsDeletedFile:    stats.IsDeletedFile,
	}
}

func diffHunksProto(hunks []DiffHunk) []*repov1.DiffHunk {
	result := make([]*repov1.DiffHunk, 0, len(hunks))
	for _, hunk := range hunks {
		result = append(result, &repov1.DiffHunk{
			OldStart: int32(hunk.OldStart),
			OldCount: int32(hunk.OldCount),
			NewStart: int32(hunk.NewStart),
			NewCount: int32(hunk.NewCount),
			Header:   hunk.Header,
			Lines:    hunk.Lines,
		})
	}
	return result
}

func annotatedLinesProto(lines []AnnotatedLine) []*repov1.AnnotatedLine {
	result := make([]*repov1.AnnotatedLine, 0, len(lines))
	for _, line := range lines {
		result = append(result, &repov1.AnnotatedLine{
			Number:    int32(line.Number),
			Content:   line.Content,
			Change:    string(line.Change),
			OldNumber: int32(line.OldNumber),
		})
	}
	return result
}
