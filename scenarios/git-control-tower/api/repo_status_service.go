package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// GetRepoStatus gathers the full repository status by running git operations
// concurrently where possible. The initial `git status --porcelain=v2` must
// complete first because subsequent steps depend on the parsed file lists.
// After that, author resolution, diff stats, and optional hotspots all run
// in parallel via an errgroup.
func GetRepoStatus(ctx context.Context, deps RepoStatusDeps) (*RepoStatus, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	// Phase 1: git status (must complete before anything else).
	out, err := deps.Git.StatusPorcelainV2(ctx, repoDir)
	if err != nil {
		return nil, err
	}

	parsed, err := ParsePorcelainV2Status(out)
	if err != nil {
		return nil, err
	}
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

	// Pre-detect binary files so we can exclude them from git diff --numstat.
	// This is a critical performance optimization: git diff --numstat must read
	// the full content of each file to compute line stats, which is extremely
	// slow for large compiled binaries (e.g., 14 Go binaries at 7-18MB each
	// turned a 34ms diff into a 1,700ms diff). By checking the first 512 bytes
	// for null bytes (the same heuristic git uses internally), we can skip
	// binaries before invoking the expensive diff and report them directly.
	stagedText, stagedPreBinaries := partitionBinaryPaths(repoDir, parsed.Files.Staged)
	unstagedText, unstagedPreBinaries := partitionBinaryPaths(repoDir, parsed.Files.Unstaged)

	// Phase 2: run independent git operations concurrently.
	var (
		author           RepoAuthorStatus
		stagedStats      map[string]DiffStats
		stagedBinaries   []string
		unstagedStats    map[string]DiffStats
		unstagedBinaries []string
		untrackedStats   map[string]DiffStats
		fileHotspots     map[string]int
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		author = resolveAuthor(gctx, deps, repoDir)
		return nil
	})

	g.Go(func() error {
		stagedStats, stagedBinaries = collectDiffStats(gctx, deps, repoDir, true, stagedText)
		return nil
	})

	g.Go(func() error {
		unstagedStats, unstagedBinaries = collectDiffStats(gctx, deps, repoDir, false, unstagedText)
		return nil
	})

	g.Go(func() error {
		untrackedStats = collectUntrackedStats(repoDir, parsed.Files.Untracked)
		return nil
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

	if len(stagedStats) > 0 || len(unstagedStats) > 0 || len(untrackedStats) > 0 {
		parsed.FileStats = RepoFileStats{
			Staged:    stagedStats,
			Unstaged:  unstagedStats,
			Untracked: untrackedStats,
		}
	}

	parsed.Files.Binary = collectBinaryFiles(
		append(stagedBinaries, stagedPreBinaries...),
		append(unstagedBinaries, unstagedPreBinaries...),
		untrackedStats,
	)
	parsed.FileHotspots = fileHotspots
	sortFileStatus(&parsed.Files, parsed.Scopes)

	return parsed, nil
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

func collectUntrackedStats(repoDir string, untracked []string) map[string]DiffStats {
	stats := map[string]DiffStats{}
	for _, path := range untracked {
		s, err := buildUntrackedStats(repoDir, path)
		if err != nil {
			continue
		}
		stats[path] = s
	}
	return stats
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
