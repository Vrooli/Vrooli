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
