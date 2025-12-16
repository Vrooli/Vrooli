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

	parsed.Summary = RepoStatusSummary{
		Staged:    len(parsed.Files.Staged),
		Unstaged:  len(parsed.Files.Unstaged),
		Untracked: len(parsed.Files.Untracked),
		Conflicts: len(parsed.Files.Conflicts),
		Ignored:   len(parsed.Files.Ignored),
	}
	parsed.Scopes = detectScopes(parsed.Files)

	sort.Strings(parsed.Files.Staged)
	sort.Strings(parsed.Files.Unstaged)
	sort.Strings(parsed.Files.Untracked)
	sort.Strings(parsed.Files.Conflicts)
	sort.Strings(parsed.Files.Ignored)
	for key := range parsed.Scopes {
		sort.Strings(parsed.Scopes[key])
	}

	return parsed, nil
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

