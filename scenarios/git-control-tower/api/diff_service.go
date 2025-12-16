package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DiffDeps contains dependencies for diff operations
type DiffDeps struct {
	Git     GitRunner
	RepoDir string
}

// GetDiff retrieves and parses a git diff
func GetDiff(ctx context.Context, deps DiffDeps, req DiffRequest) (*DiffResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	out, err := deps.Git.Diff(ctx, repoDir, req.Path, req.Staged)
	if err != nil {
		return nil, err
	}

	parsed := ParseDiffOutput(string(out))
	parsed.RepoDir = repoDir
	parsed.Path = req.Path
	parsed.Staged = req.Staged
	parsed.Base = req.Base
	parsed.Timestamp = time.Now().UTC()

	return parsed, nil
}

// ParseDiffOutput parses raw git diff output into structured form
func ParseDiffOutput(raw string) *DiffResponse {
	resp := &DiffResponse{
		Raw:   raw,
		Hunks: []DiffHunk{},
		Stats: DiffStats{},
	}

	if strings.TrimSpace(raw) == "" {
		resp.HasDiff = false
		return resp
	}

	resp.HasDiff = true
	lines := strings.Split(raw, "\n")
	filesSet := make(map[string]bool)

	var currentHunk *DiffHunk
	hunkRegex := regexp.MustCompile(`^@@\s*-(\d+)(?:,(\d+))?\s*\+(\d+)(?:,(\d+))?\s*@@(.*)$`)

	for _, line := range lines {
		// Track file changes for stats
		if strings.HasPrefix(line, "diff --git") {
			filesSet[line] = true
			continue
		}

		// Parse hunk headers
		if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
			if currentHunk != nil {
				resp.Hunks = append(resp.Hunks, *currentHunk)
			}
			currentHunk = &DiffHunk{
				OldStart: atoi(matches[1]),
				OldCount: atoiWithDefault(matches[2], 1),
				NewStart: atoi(matches[3]),
				NewCount: atoiWithDefault(matches[4], 1),
				Header:   strings.TrimSpace(matches[5]),
				Lines:    []string{},
			}
			continue
		}

		// Collect hunk lines and track stats
		if currentHunk != nil {
			currentHunk.Lines = append(currentHunk.Lines, line)
		}

		// Count additions and deletions
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			resp.Stats.Additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			resp.Stats.Deletions++
		}
	}

	// Add last hunk
	if currentHunk != nil {
		resp.Hunks = append(resp.Hunks, *currentHunk)
	}

	resp.Stats.Files = len(filesSet)

	return resp
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func atoiWithDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
