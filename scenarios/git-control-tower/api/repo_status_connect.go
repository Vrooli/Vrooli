package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	repodomain "git-control-tower/internal/repo"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (s *Server) getRepoStatusConnect(ctx context.Context, req *repov1.GetRepoStatusRequest) (*repov1.GetRepoStatusResponse, error) {
	if req.GetPageSize() > 512 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must be at most 512"))
	}
	if req.GetPageToken() != "" && req.GetPageSize() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size is required with page_token"))
	}
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), req.GetRepoPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	pageCache := s.statusPageCache
	if pageCache == nil {
		pageCache = defaultRepoStatusPageCache
	}
	if req.GetPageToken() != "" {
		return pageCache.Continue(ctx, resolved.Path, req.GetPageToken(), req.GetPageSize())
	}
	statusCache := s.statusCache
	if statusCache == nil {
		statusCache = defaultRepoStatusCache
	}
	configCache := s.configCache
	for attempt := 0; attempt < 3; attempt++ {
		generation := statusCache.Generation(resolved.Path)
		status, err := GetRepoStatus(ctx, RepoStatusDeps{
			Git:             s.git,
			RepoDir:         resolved.Path,
			ConfigCache:     configCache,
			StatusCache:     statusCache,
			IncludeHotspots: req.GetIncludeHotspots(),
			FastStatus:      req.GetPageSize() > 0,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		identity, err := repodomain.NewService(newWorktreeInspector()).GetRepoStatusWithSnapshot(ctx, resolved.Path, status.Branch.Head, status.Branch.OID)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		response := repoStatusResponseProto(status, identity)
		if req.GetPageSize() == 0 {
			return boundRepoStatusResponse(response, resolved.Path, 0, "")
		}

		// Keep every paged status snapshot briefly, even when it fits on one
		// page. Publish it under the same generation as its status result so a
		// mutation between cache lookup and snapshot storage cannot resurrect a
		// stale continuation after invalidation.
		var pagedResponse *repov1.GetRepoStatusResponse
		current, err := statusCache.WithGeneration(resolved.Path, generation, func() error {
			pagedResponse, err = pageCache.Store(ctx, resolved.Path, response, req.GetPageSize())
			return err
		})
		if err != nil {
			return nil, err
		}
		if current {
			return pagedResponse, nil
		}
	}
	return nil, connect.NewError(connect.CodeAborted, errors.New("repository changed repeatedly while preparing status; retry"))
}

// getRepoChangeMetricsConnect keeps line-counting off the initial status path.
// The request is bound to the path/status snapshot already shown by the UI, so
// a stale metrics response cannot be attached to a different file list.
func (s *Server) getRepoChangeMetricsConnect(ctx context.Context, req *repov1.GetRepoChangeMetricsRequest) (*repov1.GetRepoChangeMetricsResponse, error) {
	expected := strings.TrimSpace(req.GetStatusSnapshotId())
	if expected == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("status_snapshot_id is required"))
	}
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	pageCache := s.statusPageCache
	if pageCache == nil {
		pageCache = defaultRepoStatusPageCache
	}
	snapshot, err := pageCache.Snapshot(resolved.Path, expected)
	if err != nil {
		return nil, err
	}
	files := repoFilesStatusFromProto(snapshot.GetFiles())
	stats, _, err := collectRepoFileStats(ctx, RepoStatusDeps{Git: s.git, RepoDir: resolved.Path}, resolved.Path, files)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &repov1.GetRepoChangeMetricsResponse{
		StatusSnapshotId: expected,
		FileStats:        repoFileStatsProto(stats),
	}, nil
}

func repoFilesStatusFromProto(files *repov1.FilesStatus) RepoFilesStatus {
	if files == nil {
		return RepoFilesStatus{}
	}
	return RepoFilesStatus{
		Staged: files.GetStaged(), Unstaged: files.GetUnstaged(),
		Untracked: files.GetUntracked(), Conflicts: files.GetConflicts(),
		Binary: files.GetBinary(), Ignored: files.GetIgnored(),
		Statuses: files.GetStatuses(), Renames: files.GetRenames(),
	}
}

func repoStatusResponseProto(status *RepoStatus, identity repodomain.Status) *repov1.GetRepoStatusResponse {
	response := &repov1.GetRepoStatusResponse{
		Branch:   status.Branch.Head,
		Detached: status.Branch.Head == "" || identity.Detached,
		Worktree: repoWorktreeIdentityProto(identity),
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
	}
	return response
}

func repoWorktreeIdentityProto(identity repodomain.Status) *repov1.WorktreeIdentity {
	return &repov1.WorktreeIdentity{IsLinkedWorktree: identity.Identity.IsLinkedWorktree, CommonRepoRoot: identity.Identity.CommonRepoRoot, WorktreeName: identity.Identity.WorktreeName, WorktreeHead: identity.Identity.WorktreeHead, LinkedWorktreeCount: int32(identity.Identity.LinkedWorktreeCount)}
}

const repoStatusResponseBudget = 256 * 1024

type repoStatusCursor struct {
	Snapshot string `json:"s"`
	Offset   int    `json:"o"`
}

// boundRepoStatusResponse paginates the stable category-entry sequence. The
// cursor fingerprints the full semantic response (excluding request timestamp)
// so a caller cannot combine pages from different worktree states.
func boundRepoStatusResponse(full *repov1.GetRepoStatusResponse, repoPath string, requested uint32, token string) (*repov1.GetRepoStatusResponse, error) {
	snapshot, err := repoStatusSnapshotID(full, repoPath)
	if err != nil {
		return nil, err
	}
	total := statusEntryCount(full.GetFiles())
	offset := 0
	if token != "" {
		decoded, e := base64.RawURLEncoding.DecodeString(token)
		var cursor repoStatusCursor
		if e != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.Offset < 0 || cursor.Offset > total {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid repo status page token"))
		}
		if cursor.Snapshot != snapshot {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("repository status changed between pages; restart from the first page"))
		}
		offset = cursor.Offset
	}
	pageSize := int(requested)
	if pageSize == 0 {
		pageSize = total
	}
	if requested == 0 && statusResponseBytes(full) > repoStatusResponseBudget {
		return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("repository status exceeds 256 KiB; retry with page_size and follow page.next_page_token"))
	}
	if pageSize > total-offset {
		pageSize = total - offset
	}
	var page *repov1.GetRepoStatusResponse
	for {
		page = projectRepoStatusPage(full, snapshot, offset, pageSize, total)
		// Reserve the encoded width of this field before checking the cap.
		page.Page.ResponseBytes = repoStatusResponseBudget
		pageBytes := statusResponseBytes(page)
		if pageBytes <= repoStatusResponseBudget {
			page.Page.ResponseBytes = uint32(pageBytes)
			return page, nil
		}
		if requested == 0 {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("repository status exceeds 256 KiB with pagination metadata; retry with page_size and follow page.next_page_token"))
		}
		if pageSize <= 1 {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("a single repository status entry exceeds the 256 KiB response budget"))
		}
		pageSize /= 2
	}
}

// repoStatusSnapshotID identifies the paths, categories and repository state
// represented by GetRepoStatus. Line statistics are deliberately excluded:
// the paged status request skips them and the UI loads them through a separate
// metrics RPC after the fast file list is usable.
func repoStatusSnapshotID(full *repov1.GetRepoStatusResponse, repoPath string) (string, error) {
	semantic := proto.Clone(full).(*repov1.GetRepoStatusResponse)
	semantic.Timestamp = ""
	semantic.Page = nil
	semantic.FileStats = nil
	canonical, err := proto.MarshalOptions{Deterministic: true}.Marshal(semantic)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, err)
	}
	h := sha256.Sum256(append([]byte(repoPath+"\x00"), canonical...))
	return fmt.Sprintf("%x", h[:]), nil
}

func statusResponseBytes(response *repov1.GetRepoStatusResponse) int {
	protobufBytes := proto.Size(response) + 5 // Connect envelope
	jsonBytes, err := protojson.Marshal(response)
	if err == nil && len(jsonBytes) > protobufBytes {
		protobufBytes = len(jsonBytes)
	}
	return protobufBytes
}

func statusEntryCount(files *repov1.FilesStatus) int {
	return len(files.GetStaged()) + len(files.GetUnstaged()) + len(files.GetUntracked()) + len(files.GetConflicts()) + len(files.GetIgnored())
}

func projectRepoStatusPage(full *repov1.GetRepoStatusResponse, snapshot string, offset, limit, total int) *repov1.GetRepoStatusResponse {
	// Build a page projection directly. Cloning the full response here copied
	// every path and map for every page (N pages × the entire worktree), even
	// though each response contains only a small slice of paths.
	out := &repov1.GetRepoStatusResponse{
		RepoDir: full.GetRepoDir(), Branch: full.GetBranch(), Detached: full.GetDetached(),
		BranchStatus: full.GetBranchStatus(), FileStats: &repov1.FileStats{},
		FileHotspots: map[string]int32{}, Scopes: map[string]*repov1.StringList{},
		Summary: full.GetSummary(), Author: full.GetAuthor(), Timestamp: full.GetTimestamp(),
		Worktree: full.GetWorktree(),
	}
	f := &repov1.FilesStatus{}
	out.Files = f
	source := full.GetFiles()
	remainingSkip, remainingTake := offset, limit
	selectPart := func(paths []string) []string {
		if remainingSkip >= len(paths) {
			remainingSkip -= len(paths)
			return []string{}
		}
		start := remainingSkip
		remainingSkip = 0
		n := len(paths) - start
		if n > remainingTake {
			n = remainingTake
		}
		remainingTake -= n
		return append([]string(nil), paths[start:start+n]...)
	}
	f.Staged = selectPart(source.GetStaged())
	f.Unstaged = selectPart(source.GetUnstaged())
	f.Untracked = selectPart(source.GetUntracked())
	f.Conflicts = selectPart(source.GetConflicts())
	f.Ignored = selectPart(source.GetIgnored())
	selected := map[string]bool{}
	selectedPaths := make([]string, 0, limit)
	for _, group := range [][]string{f.Staged, f.Unstaged, f.Untracked, f.Conflicts, f.Ignored} {
		for _, path := range group {
			if selected[path] {
				continue
			}
			selected[path] = true
			selectedPaths = append(selectedPaths, path)
		}
	}
	filterString := func(src map[string]string) map[string]string {
		dst := make(map[string]string)
		for _, path := range selectedPaths {
			if value, ok := src[path]; ok {
				dst[path] = value
			}
		}
		return dst
	}
	f.Statuses = filterString(source.GetStatuses())
	f.Renames = filterString(source.GetRenames())
	filterStats := func(src map[string]*repov1.DiffStats, paths []string) map[string]*repov1.DiffStats {
		dst := map[string]*repov1.DiffStats{}
		for _, p := range paths {
			if v := src[p]; v != nil {
				dst[p] = v
			}
		}
		return dst
	}
	out.FileStats = &repov1.FileStats{Staged: filterStats(full.GetFileStats().GetStaged(), f.Staged), Unstaged: filterStats(full.GetFileStats().GetUnstaged(), f.Unstaged), Untracked: filterStats(full.GetFileStats().GetUntracked(), f.Untracked)}
	f.Binary = filterPaths(source.GetBinary(), selected)
	out.FileHotspots = make(map[string]int32)
	for _, path := range selectedPaths {
		if value, ok := full.GetFileHotspots()[path]; ok {
			out.FileHotspots[path] = value
		}
	}
	out.Scopes = make(map[string]*repov1.StringList)
	for _, path := range selectedPaths {
		if value, ok := full.GetScopes()[path]; ok {
			out.Scopes[path] = value
		}
	}
	end := offset + limit
	if end > total {
		end = total
	}
	complete := end >= total
	out.Page = &repov1.RepoStatusPage{ReturnedEntries: uint32(end - offset), TotalEntries: uint32(total), SnapshotId: snapshot, Complete: complete}
	if !complete {
		raw, _ := json.Marshal(repoStatusCursor{Snapshot: snapshot, Offset: end})
		out.Page.NextPageToken = base64.RawURLEncoding.EncodeToString(raw)
	}
	return out
}

func filterPaths(paths []string, selected map[string]bool) []string {
	var out []string
	for _, p := range paths {
		if selected[p] {
			out = append(out, p)
		}
	}
	return out
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
