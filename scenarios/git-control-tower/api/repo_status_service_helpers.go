package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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

	out, err := deps.Git.LogGraph(ctx, repoDir, limit, deps.GrepPattern)
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
		RepoDir:     repoDir,
		Lines:       lines,
		Limit:       limit,
		GrepPattern: deps.GrepPattern,
		Timestamp:   time.Now().UTC(),
	}

	if deps.IncludeFiles || deps.IncludeChecks {
		detailsRaw, err := deps.Git.LogDetails(ctx, repoDir, limit, deps.GrepPattern)
		if err != nil {
			return nil, err
		}
		entries := parseHistoryDetails(detailsRaw)
		if deps.IncludeChecks && deps.CommitChecks != nil && len(entries) > 0 {
			hashes := make([]string, 0, len(entries))
			for _, entry := range entries {
				hashes = append(hashes, entry.Hash)
			}
			checksByHash, err := deps.CommitChecks.ListForCommits(ctx, repoDir, hashes)
			if err != nil {
				return nil, err
			}
			for index := range entries {
				entries[index].Checks = checksByHash[entries[index].Hash]
			}
		}
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

// scopePrefixMap maps top-level directories to their scope prefix.
var scopePrefixMap = map[string]string{
	"scenarios": "scenario:",
	"resources": "resource:",
	"packages":  "package:",
	"apps":      "app:",
	"services":  "service:",
}

func scopeKeyForPath(path string) string {
	trimmed := strings.TrimLeft(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) >= 2 {
		if prefix, ok := scopePrefixMap[parts[0]]; ok {
			return prefix + parts[1]
		}
	}
	return "other"
}

func parseNumstatOutput(out []byte) (map[string]DiffStats, []string) {
	stats := map[string]DiffStats{}
	var binaries []string
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return stats, binaries
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
		additions := strings.TrimSpace(parts[0])
		deletions := strings.TrimSpace(parts[1])
		if additions == "-" || deletions == "-" {
			binaries = append(binaries, path)
			stats[path] = DiffStats{Files: 1, IsBinary: true}
			continue
		}
		add := parseNumstatValue(parts[0])
		del := parseNumstatValue(parts[1])
		stats[path] = DiffStats{
			Additions: add,
			Deletions: del,
			Files:     1,
			NetLines:  add - del,
		}
	}
	return stats, binaries
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
	if isBinary {
		return DiffStats{Files: 1, IsBinary: true, IsNewFile: true}, nil
	}
	return DiffStats{Files: 1, Additions: lines, NetLines: lines, IsNewFile: true}, nil
}

// processChunk checks a buffer chunk for binary content (null bytes) and counts newlines.
// Returns the line count, whether binary was detected, and the last byte in the chunk.
func processChunk(buf []byte) (int, bool, byte) {
	if bytes.IndexByte(buf, 0) >= 0 {
		return 0, true, 0
	}
	lines := 0
	for _, b := range buf {
		if b == '\n' {
			lines++
		}
	}
	return lines, false, buf[len(buf)-1]
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
			chunkLines, isBinary, last := processChunk(buf[:n])
			if isBinary {
				return 0, true, nil
			}
			lines += chunkLines
			lastByte = last
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
