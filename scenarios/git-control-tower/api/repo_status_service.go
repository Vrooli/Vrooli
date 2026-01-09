package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	parsed.Author = RepoAuthorStatus{}
	if name, err := deps.Git.ConfigGet(ctx, repoDir, "user.name"); err == nil {
		parsed.Author.Name = name
	}
	if email, err := deps.Git.ConfigGet(ctx, repoDir, "user.email"); err == nil {
		parsed.Author.Email = email
	}

	parsed.Summary = RepoStatusSummary{
		Staged:    len(parsed.Files.Staged),
		Unstaged:  len(parsed.Files.Unstaged),
		Untracked: len(parsed.Files.Untracked),
		Conflicts: len(parsed.Files.Conflicts),
		Ignored:   len(parsed.Files.Ignored),
	}
	parsed.Scopes = detectScopes(parsed.Files)

	stagedStats := map[string]DiffStats{}
	if numstat, err := deps.Git.DiffNumstat(ctx, repoDir, true); err == nil {
		stagedStats = parseNumstatOutput(numstat)
	}

	unstagedStats := map[string]DiffStats{}
	if numstat, err := deps.Git.DiffNumstat(ctx, repoDir, false); err == nil {
		unstagedStats = parseNumstatOutput(numstat)
	}

	untrackedStats := map[string]DiffStats{}
	for _, path := range parsed.Files.Untracked {
		stats, err := buildUntrackedStats(repoDir, path)
		if err != nil {
			continue
		}
		untrackedStats[path] = stats
	}

	if len(stagedStats) > 0 || len(unstagedStats) > 0 || len(untrackedStats) > 0 {
		parsed.FileStats = RepoFileStats{
			Staged:    stagedStats,
			Unstaged:  unstagedStats,
			Untracked: untrackedStats,
		}
	}

	binarySet := map[string]struct{}{}
	if numstat, err := deps.Git.DiffNumstat(ctx, repoDir, true); err == nil {
		addBinaryFiles(binarySet, numstat)
	}
	if numstat, err := deps.Git.DiffNumstat(ctx, repoDir, false); err == nil {
		addBinaryFiles(binarySet, numstat)
	}
	if len(binarySet) > 0 {
		parsed.Files.Binary = make([]string, 0, len(binarySet))
		for path := range binarySet {
			parsed.Files.Binary = append(parsed.Files.Binary, path)
		}
	}

	sort.Strings(parsed.Files.Staged)
	sort.Strings(parsed.Files.Unstaged)
	sort.Strings(parsed.Files.Untracked)
	sort.Strings(parsed.Files.Conflicts)
	sort.Strings(parsed.Files.Ignored)
	sort.Strings(parsed.Files.Binary)
	for key := range parsed.Scopes {
		sort.Strings(parsed.Scopes[key])
	}

	return parsed, nil
}

func addBinaryFiles(out map[string]struct{}, numstat []byte) {
	lines := strings.Split(strings.TrimSpace(string(numstat)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		additions := strings.TrimSpace(parts[0])
		deletions := strings.TrimSpace(parts[1])
		path := strings.TrimSpace(parts[2])
		if path == "" {
			continue
		}
		if additions == "-" || deletions == "-" {
			out[path] = struct{}{}
		}
	}
}

func parseNumstatOutput(out []byte) map[string]DiffStats {
	stats := map[string]DiffStats{}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return stats
	}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		path := strings.TrimSpace(parts[2])
		if path == "" {
			continue
		}
		stats[path] = DiffStats{
			Additions: parseNumstatValue(parts[0]),
			Deletions: parseNumstatValue(parts[1]),
			Files:     1,
		}
	}
	return stats
}

func parseNumstatValue(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "-" {
		return 0
	}
	num, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0
	}
	if num < 0 {
		return 0
	}
	return num
}

func buildUntrackedStats(repoDir string, path string) (DiffStats, error) {
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(repoDir, path)
	}
	lines, isBinary, err := countFileLines(fullPath)
	if err != nil {
		return DiffStats{}, err
	}
	stats := DiffStats{Files: 1}
	if !isBinary {
		stats.Additions = lines
	}
	return stats, nil
}

func countFileLines(path string) (int, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	lines := 0
	hasContent := false
	var lastByte byte

	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			hasContent = true
			if bytes.IndexByte(buf[:n], 0) >= 0 {
				return 0, true, nil
			}
			for _, b := range buf[:n] {
				if b == '\n' {
					lines++
				}
			}
			lastByte = buf[n-1]
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, false, readErr
		}
	}

	if hasContent && lastByte != '\n' {
		lines++
	}

	return lines, false, nil
}

func GetRepoHistory(ctx context.Context, deps RepoHistoryDeps) (*RepoHistory, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	limit := deps.Limit
	if limit <= 0 {
		limit = 30
	}

	out, err := deps.Git.LogGraph(ctx, repoDir, limit)
	if err != nil {
		return nil, err
	}

	raw := strings.TrimRight(string(out), "\n")
	lines := []string{}
	if raw != "" {
		lines = strings.Split(raw, "\n")
	}
	if len(lines) > 0 {
		lines = filterHistoryLines(lines)
	}

	history := &RepoHistory{
		RepoDir:   repoDir,
		Lines:     lines,
		Limit:     limit,
		Timestamp: time.Now().UTC(),
	}

	if deps.IncludeFiles {
		detailsRaw, err := deps.Git.LogDetails(ctx, repoDir, limit)
		if err != nil {
			return nil, err
		}
		entries := parseHistoryDetails(detailsRaw)
		history.Entries = entries
	}

	return history, nil
}

var commitHashPattern = regexp.MustCompile(`[0-9a-f]{7,}`)

func filterHistoryLines(lines []string) []string {
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if commitHashPattern.MatchString(line) {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func parseHistoryDetails(out []byte) []RepoHistoryEntry {
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return []RepoHistoryEntry{}
	}

	blocks := strings.Split(raw, "\n\n")
	entries := make([]RepoHistoryEntry, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}
		header := lines[0]
		parts := strings.Split(header, "\x00")
		if len(parts) < 4 {
			continue
		}
		entry := RepoHistoryEntry{
			Hash:    strings.TrimSpace(parts[0]),
			Author:  strings.TrimSpace(parts[1]),
			Date:    strings.TrimSpace(parts[2]),
			Subject: strings.TrimSpace(parts[3]),
			Files:   []string{},
		}
		for _, line := range lines[1:] {
			path := strings.TrimSpace(line)
			if path == "" {
				continue
			}
			entry.Files = append(entry.Files, path)
		}
		if entry.Hash == "" {
			continue
		}
		entries = append(entries, entry)
	}

	return entries
}

func detectScopes(files RepoFilesStatus) map[string][]string {
	scopes := map[string][]string{}
	for _, path := range append(append(append(files.Staged, files.Unstaged...), files.Untracked...), files.Conflicts...) {
		key := scopeKeyForPath(path)
		scopes[key] = append(scopes[key], path)
	}
	for _, path := range files.Ignored {
		key := scopeKeyForPath(path)
		scopes[key] = append(scopes[key], path)
	}
	return scopes
}

func scopeKeyForPath(path string) string {
	trimmed := strings.TrimLeft(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) >= 2 && parts[0] == "scenarios" {
		return "scenario:" + parts[1]
	}
	if len(parts) >= 2 && parts[0] == "resources" {
		return "resource:" + parts[1]
	}
	if len(parts) >= 2 && parts[0] == "packages" {
		return "package:" + parts[1]
	}
	return "other"
}
