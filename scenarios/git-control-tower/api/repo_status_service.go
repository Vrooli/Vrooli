package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

var defaultRepoStatusCache *RepoStatusCache

// SetDefaultRepoStatusCache wires the process-wide cache used by internal
// status call sites that do not carry an HTTP server dependency. Tests can
// leave it unset or inject a per-call cache explicitly.
func SetDefaultRepoStatusCache(cache *RepoStatusCache) {
	defaultRepoStatusCache = cache
}

// GetRepoStatus gathers the full repository status by running git operations
// concurrently where possible. The initial `git status --porcelain=v2` must
// complete first because subsequent steps depend on the parsed file lists.
// After that, author resolution, diff stats, and optional hotspots all run
// in parallel via an errgroup.
func GetRepoStatus(ctx context.Context, deps RepoStatusDeps) (*RepoStatus, error) {
	if deps.StatusCache == nil {
		deps.StatusCache = defaultRepoStatusCache
	}
	if deps.StatusCache != nil {
		key := deps.RepoDir
		if deps.IncludeHotspots {
			key += "\x00hotspots"
		}
		if deps.FastStatus {
			key += "\x00fast"
		}
		return deps.StatusCache.Get(ctx, key, func(ctx context.Context) (*RepoStatus, error) {
			deps.StatusCache = nil
			return getRepoStatus(ctx, deps)
		})
	}
	return getRepoStatus(ctx, deps)
}

// readRepoStatusSnapshot reads fresh branch/index/worktree identities without
// presentation enrichment. Authorization and target resolution must not compute
// line counts, open untracked contents, or resolve display-only author settings.
func readRepoStatusSnapshot(ctx context.Context, git GitRunner, repoDir string) (*RepoStatus, error) {
	if git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	out, err := git.StatusPorcelainV2(ctx, repoDir)
	if err != nil {
		return nil, err
	}
	return ParsePorcelainV2Status(out)
}

func getRepoStatus(ctx context.Context, deps RepoStatusDeps) (*RepoStatus, error) {
	repoDir, err := validateGitRepoDeps(deps.Git, deps.RepoDir)
	if err != nil {
		return nil, err
	}

	// Phase 1: git status (must complete before anything else).
	statusPorcelain, err := deps.Git.StatusPorcelainV2(ctx, repoDir)
	if err != nil {
		return nil, err
	}
	parsed, err := ParsePorcelainV2Status(statusPorcelain)
	if err != nil {
		return nil, err
	}
	parsed.statusPorcelain = append([]byte(nil), statusPorcelain...)
	parsed.RepoDir = repoDir
	parsed.Timestamp = time.Now().UTC()

	parsed.Summary = RepoStatusSummary{
		Staged:    len(parsed.Files.Staged),
		Unstaged:  len(parsed.Files.Unstaged),
		Untracked: len(parsed.Files.Untracked),
		Conflicts: len(parsed.Files.Conflicts),
		Ignored:   len(parsed.Files.Ignored),
	}
	parsed.Scopes = detectScopes(repoDir, parsed.Files)
	if deps.FastStatus {
		// The Changes panel needs paths and status categories immediately. Diff
		// line counts and untracked-file scans are presentation enrichment and
		// can dominate latency on a large dirty worktree, so defer them to the
		// separate change-metrics request.
		parsed.Author = resolveAuthor(ctx, deps, repoDir)
		sortFileStatus(&parsed.Files, parsed.Scopes)
		return parsed, nil
	}

	// Phase 2: run independent Git and filesystem enrichment concurrently.
	var (
		author       RepoAuthorStatus
		fileStats    RepoFileStats
		binaryFiles  []string
		fileHotspots map[string]int
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		author = resolveAuthor(gctx, deps, repoDir)
		return nil
	})

	g.Go(func() error {
		var collectErr error
		fileStats, binaryFiles, collectErr = collectRepoFileStats(gctx, deps, repoDir, parsed.Files)
		return collectErr
	})

	if deps.IncludeHotspots {
		g.Go(func() error {
			fileHotspots = computeFileHotspots(gctx, deps, repoDir, parsed.Files)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	parsed.Author = author

	parsed.FileStats = fileStats
	parsed.Files.Binary = binaryFiles
	parsed.FileHotspots = fileHotspots
	sortFileStatus(&parsed.Files, parsed.Scopes)

	return parsed, nil
}

// collectRepoFileStats computes the line metrics that enrich status views.
// It accepts a captured path snapshot so callers can defer this work until
// after the path list has rendered without running `git status` a second time.
func collectRepoFileStats(ctx context.Context, deps RepoStatusDeps, repoDir string, files RepoFilesStatus) (RepoFileStats, []string, error) {
	// Pre-detect binary files so `git diff --numstat` does not read their full
	// contents. This keeps compiled binaries out of the expensive diff path.
	stagedText, stagedPreBinaries := partitionBinaryPaths(repoDir, files.Staged)
	unstagedText, unstagedPreBinaries := partitionBinaryPaths(repoDir, files.Unstaged)
	// Include rename origins so Git reports edits to a moved file rather than a
	// whole-file add.
	stagedText = withRenameOrigins(stagedText, files.Renames)
	unstagedText = withRenameOrigins(unstagedText, files.Renames)

	var (
		stagedStats      map[string]DiffStats
		stagedBinaries   []string
		unstagedStats    map[string]DiffStats
		unstagedBinaries []string
		untrackedStats   map[string]DiffStats
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		stagedStats, stagedBinaries = collectDiffStats(gctx, deps, repoDir, true, stagedText)
		return nil
	})
	g.Go(func() error {
		unstagedStats, unstagedBinaries = collectDiffStats(gctx, deps, repoDir, false, unstagedText)
		return nil
	})
	g.Go(func() error {
		var err error
		untrackedStats, err = collectUntrackedStats(gctx, repoDir, files.Untracked)
		return err
	})
	if err := g.Wait(); err != nil {
		return RepoFileStats{}, nil, err
	}
	stats := RepoFileStats{Staged: stagedStats, Unstaged: unstagedStats, Untracked: untrackedStats}
	binaries := collectBinaryFiles(
		append(stagedBinaries, stagedPreBinaries...),
		append(unstagedBinaries, unstagedPreBinaries...),
		untrackedStats,
	)
	return stats, binaries, nil
}

// resolveAuthor reads user.name and user.email from git config.
// Uses the config cache when available to avoid spawning git processes.
func resolveAuthor(ctx context.Context, deps RepoStatusDeps, repoDir string) RepoAuthorStatus {
	author := RepoAuthorStatus{}
	if deps.ConfigCache != nil {
		if name, err := deps.ConfigCache.ConfigGet(ctx, deps.Git, repoDir, "user.name"); err == nil {
			author.Name = name
		}
		if email, err := deps.ConfigCache.ConfigGet(ctx, deps.Git, repoDir, "user.email"); err == nil {
			author.Email = email
		}
	} else {
		if name, err := deps.Git.ConfigGet(ctx, repoDir, "user.name"); err == nil {
			author.Name = name
		}
		if email, err := deps.Git.ConfigGet(ctx, repoDir, "user.email"); err == nil {
			author.Email = email
		}
	}
	return author
}

func collectDiffStats(ctx context.Context, deps RepoStatusDeps, repoDir string, staged bool, paths []string) (map[string]DiffStats, []string) {
	numstat, err := deps.Git.DiffNumstat(ctx, repoDir, staged, paths...)
	if err != nil {
		return map[string]DiffStats{}, nil
	}
	return parseNumstatOutput(numstat)
}

func collectUntrackedStats(ctx context.Context, repoDir string, untracked []string) (map[string]DiffStats, error) {
	stats := map[string]DiffStats{}
	if len(untracked) == 0 {
		return stats, nil
	}
	// Counting untracked files is an independent local read for every path.
	// A small fixed pool bounds concurrent file descriptors and disk pressure
	// while avoiding a serial full-file scan on repositories with large
	// untracked sets.
	workers := min(8, len(untracked))
	jobs := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				s, err := buildUntrackedStats(repoDir, path)
				if err == nil {
					mu.Lock()
					stats[path] = s
					mu.Unlock()
				}
			}
		}()
	}
send:
	for _, path := range untracked {
		select {
		case jobs <- path:
		case <-ctx.Done():
			break send
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func collectBinaryFiles(stagedBinaries, unstagedBinaries []string, untrackedStats map[string]DiffStats) []string {
	binarySet := map[string]struct{}{}
	for _, p := range stagedBinaries {
		binarySet[p] = struct{}{}
	}
	for _, p := range unstagedBinaries {
		binarySet[p] = struct{}{}
	}
	for path, stats := range untrackedStats {
		if stats.IsBinary {
			binarySet[path] = struct{}{}
		}
	}
	if len(binarySet) == 0 {
		return nil
	}
	result := make([]string, 0, len(binarySet))
	for path := range binarySet {
		result = append(result, path)
	}
	return result
}

// partitionBinaryPaths splits file paths into text and binary lists by
// reading the first 512 bytes and checking for null bytes (the same
// heuristic git itself uses in buffer_is_binary). This is cheap — at most
// 512 bytes of I/O per file — and lets us skip expensive diff computation
// on large compiled artifacts.
func partitionBinaryPaths(repoDir string, paths []string) (text, binary []string) {
	for _, p := range paths {
		if isBinaryFile(filepath.Join(repoDir, p)) {
			binary = append(binary, p)
		} else {
			text = append(text, p)
		}
	}
	return text, binary
}

// withRenameOrigins appends the origin path of every rename in paths, so a
// pathspec built from paths shows git both halves of the pair. Without both
// halves git cannot recognize the rename and scores the destination as a
// whole-file add with no matching deletion, inflating the reported additions
// by the full length of every moved file.
//
// Origins are not run through the binary partition: a rename's origin shares
// its destination's content, which has already been classified, and the origin
// no longer exists on disk to be sampled.
func withRenameOrigins(paths []string, renames map[string]string) []string {
	if len(renames) == 0 {
		return paths
	}
	known := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		known[p] = struct{}{}
	}
	var origins []string
	for _, p := range paths {
		orig, ok := renames[p]
		if !ok {
			continue
		}
		if _, seen := known[orig]; seen {
			continue
		}
		known[orig] = struct{}{}
		origins = append(origins, orig)
	}
	if len(origins) == 0 {
		return paths
	}
	result := make([]string, 0, len(paths)+len(origins))
	result = append(result, paths...)
	return append(result, origins...)
}

// isBinaryFile returns true if the file appears to contain binary content.
// It reads up to 512 bytes and checks for null bytes, matching the heuristic
// git uses internally (see buffer_is_binary in the git source).
func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		return false
	}
	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}

func computeFileHotspots(ctx context.Context, deps RepoStatusDeps, repoDir string, files RepoFilesStatus) map[string]int {
	hotspots, err := deps.Git.LogFileFrequency(ctx, repoDir, 50)
	if err != nil {
		return nil
	}
	changedFiles := map[string]struct{}{}
	for _, p := range files.Staged {
		changedFiles[p] = struct{}{}
	}
	for _, p := range files.Unstaged {
		changedFiles[p] = struct{}{}
	}
	for _, p := range files.Untracked {
		changedFiles[p] = struct{}{}
	}
	filtered := map[string]int{}
	for path, count := range hotspots {
		if _, ok := changedFiles[path]; ok {
			filtered[path] = count
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func sortFileStatus(files *RepoFilesStatus, scopes map[string][]string) {
	sort.Strings(files.Staged)
	sort.Strings(files.Unstaged)
	sort.Strings(files.Untracked)
	sort.Strings(files.Conflicts)
	sort.Strings(files.Ignored)
	sort.Strings(files.Binary)
	for key := range scopes {
		sort.Strings(scopes[key])
	}
}
