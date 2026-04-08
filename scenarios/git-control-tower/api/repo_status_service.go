package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func GetRepoStatus(ctx context.Context, deps RepoStatusDeps) (*RepoStatus, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

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
	parsed.Author = resolveAuthor(ctx, deps, repoDir)

	parsed.Summary = RepoStatusSummary{
		Staged:    len(parsed.Files.Staged),
		Unstaged:  len(parsed.Files.Unstaged),
		Untracked: len(parsed.Files.Untracked),
		Conflicts: len(parsed.Files.Conflicts),
		Ignored:   len(parsed.Files.Ignored),
	}
	parsed.Scopes = detectScopes(parsed.Files)

	stagedStats, stagedBinaries := collectDiffStats(ctx, deps, repoDir, true)
	unstagedStats, unstagedBinaries := collectDiffStats(ctx, deps, repoDir, false)
	untrackedStats := collectUntrackedStats(repoDir, parsed.Files.Untracked)

	if len(stagedStats) > 0 || len(unstagedStats) > 0 || len(untrackedStats) > 0 {
		parsed.FileStats = RepoFileStats{
			Staged:    stagedStats,
			Unstaged:  unstagedStats,
			Untracked: untrackedStats,
		}
	}

	parsed.Files.Binary = collectBinaryFiles(stagedBinaries, unstagedBinaries, untrackedStats)
	parsed.FileHotspots = computeFileHotspots(ctx, deps, repoDir, parsed.Files)
	sortFileStatus(&parsed.Files, parsed.Scopes)

	return parsed, nil
}

func resolveAuthor(ctx context.Context, deps RepoStatusDeps, repoDir string) RepoAuthorStatus {
	author := RepoAuthorStatus{}
	if name, err := deps.Git.ConfigGet(ctx, repoDir, "user.name"); err == nil {
		author.Name = name
	}
	if email, err := deps.Git.ConfigGet(ctx, repoDir, "user.email"); err == nil {
		author.Email = email
	}
	return author
}

func collectDiffStats(ctx context.Context, deps RepoStatusDeps, repoDir string, staged bool) (map[string]DiffStats, []string) {
	numstat, err := deps.Git.DiffNumstat(ctx, repoDir, staged)
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
