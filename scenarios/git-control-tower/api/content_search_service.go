package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ContentSearchDeps holds dependencies for content search operations.
type ContentSearchDeps struct {
	Git     GitRunner
	RepoDir string
}

// ValidateContentSearchRequest validates and normalizes the request.
func ValidateContentSearchRequest(req *ContentSearchRequest) error {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return fmt.Errorf("query is required")
	}
	if len(query) < ContentSearchMinQueryLen {
		return fmt.Errorf("query must be at least %d characters", ContentSearchMinQueryLen)
	}
	if len(query) > ContentSearchMaxQueryLen {
		return fmt.Errorf("query must be at most %d characters", ContentSearchMaxQueryLen)
	}

	// Validate regex syntax if regex mode is enabled
	if req.Regex {
		if _, err := regexp.Compile(query); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Normalize and cap limits
	if req.Limit <= 0 {
		req.Limit = ContentSearchDefaultLimit
	}
	if req.Limit > ContentSearchMaxLimit {
		req.Limit = ContentSearchMaxLimit
	}

	if req.Timeout <= 0 {
		req.Timeout = ContentSearchDefaultTimeout
	}
	if req.Timeout > ContentSearchMaxTimeout {
		req.Timeout = ContentSearchMaxTimeout
	}

	if req.ContextLines < 0 {
		req.ContextLines = 0
	}
	if req.ContextLines > ContentSearchMaxContextLines {
		req.ContextLines = ContentSearchMaxContextLines
	}

	return nil
}

// parseGlobs splits a comma-separated glob string into a slice.
func parseGlobs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// SearchContent performs a content search using git grep.
func SearchContent(ctx context.Context, deps ContentSearchDeps, req ContentSearchRequest) (*ContentSearchResponse, error) {
	if err := ValidateContentSearchRequest(&req); err != nil {
		return nil, err
	}

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Millisecond)
	defer cancel()

	// Build grep options
	opts := GrepOptions{
		Pattern:       req.Query,
		CaseSensitive: req.CaseSensitive,
		WholeWord:     req.WholeWord,
		ExtendedRegex: req.Regex,
		IncludeGlobs:  parseGlobs(req.Include),
		ExcludeGlobs:  parseGlobs(req.Exclude),
		ContextLines:  req.ContextLines,
		MaxCount:      ContentSearchMaxPerFile, // Limit matches per file
	}

	// Execute git grep
	out, err := deps.Git.GrepContent(timeoutCtx, deps.RepoDir, opts)

	// Check if context was cancelled (timeout)
	cancelled := false
	if timeoutCtx.Err() == context.DeadlineExceeded {
		cancelled = true
	} else if err != nil {
		return nil, err
	}

	// Parse the output
	matches, total, truncated := parseGrepOutput(string(out), req.Limit, req.ContextLines)

	return &ContentSearchResponse{
		Matches:   matches,
		Total:     total,
		Truncated: truncated,
		Cancelled: cancelled,
		Query:     req.Query,
		Timestamp: time.Now(),
	}, nil
}

// parseGrepOutput parses git grep output into ContentSearchMatch structs.
// Git grep output format with -n: file:line:content
// With context (-C): file-line-content for context, file:line:content for matches
func parseGrepOutput(output string, limit int, contextLines int) ([]ContentSearchMatch, int, bool) {
	if strings.TrimSpace(output) == "" {
		return []ContentSearchMatch{}, 0, false
	}

	lines := strings.Split(output, "\n")
	var matches []ContentSearchMatch
	total := 0
	truncated := false

	// Track context lines for grouping
	var pendingContextBefore []string
	var currentMatch *ContentSearchMatch

	for _, line := range lines {
		if line == "" {
			// Empty line often separates groups in context mode
			if currentMatch != nil {
				matches = append(matches, *currentMatch)
				currentMatch = nil
			}
			pendingContextBefore = nil
			continue
		}

		// Check if we're at the separator between file groups (--) in context mode
		if line == "--" {
			if currentMatch != nil {
				matches = append(matches, *currentMatch)
				currentMatch = nil
			}
			pendingContextBefore = nil
			continue
		}

		// Parse the line - look for file:line:content or file-line-content pattern
		// Match line format: file:linenum:content
		// Context line format: file-linenum-content
		isMatch, path, lineNum, content := parseGrepLine(line)

		if path == "" {
			// Couldn't parse - might be continuation or malformed
			continue
		}

		if isMatch {
			total++

			// Check limit
			if len(matches) >= limit {
				truncated = true
				// Finalize current match if any
				if currentMatch != nil {
					matches = append(matches, *currentMatch)
				}
				continue
			}

			// Finalize previous match
			if currentMatch != nil {
				matches = append(matches, *currentMatch)
			}

			// Create new match
			currentMatch = &ContentSearchMatch{
				Path:       path,
				LineNumber: lineNum,
				Content:    truncateLine(content, ContentSearchMaxLineLen),
			}

			// Add pending context as context_before
			if len(pendingContextBefore) > 0 && contextLines > 0 {
				currentMatch.ContextBefore = strings.Join(pendingContextBefore, "\n")
			}
			pendingContextBefore = nil

		} else if contextLines > 0 {
			// This is a context line
			truncatedContent := truncateLine(content, ContentSearchMaxLineLen)

			if currentMatch != nil {
				// This is context_after for the current match
				if currentMatch.ContextAfter == "" {
					currentMatch.ContextAfter = truncatedContent
				} else {
					currentMatch.ContextAfter += "\n" + truncatedContent
				}
			} else {
				// This is context_before for an upcoming match
				pendingContextBefore = append(pendingContextBefore, truncatedContent)
				// Keep only the last N context lines
				if len(pendingContextBefore) > contextLines {
					pendingContextBefore = pendingContextBefore[len(pendingContextBefore)-contextLines:]
				}
			}
		}
	}

	// Don't forget the last match
	if currentMatch != nil && len(matches) < limit {
		matches = append(matches, *currentMatch)
	}

	return matches, total, truncated
}

// parseGrepLine parses a single line of git grep output.
// Returns: isMatch (true for match, false for context), path, lineNumber, content
func parseGrepLine(line string) (bool, string, int, string) {
	// Git grep uses : for matches and - for context lines
	// Format: path:linenum:content or path-linenum-content
	// We need to find the first separator after the path

	// Find the line number separator
	// The path can contain : so we look for the pattern where we have a number after the separator
	isMatch := true

	// Try to find :number: pattern first (match line)
	matchIdx := -1
	for i := 0; i < len(line); i++ {
		if line[i] == ':' || line[i] == '-' {
			// Check if followed by digits and another separator
			j := i + 1
			for j < len(line) && line[j] >= '0' && line[j] <= '9' {
				j++
			}
			if j > i+1 && j < len(line) && (line[j] == ':' || line[j] == '-') {
				matchIdx = i
				isMatch = line[i] == ':'
				break
			}
		}
	}

	if matchIdx == -1 {
		return false, "", 0, ""
	}

	path := line[:matchIdx]

	// Find the end of the line number
	numStart := matchIdx + 1
	numEnd := numStart
	for numEnd < len(line) && line[numEnd] >= '0' && line[numEnd] <= '9' {
		numEnd++
	}

	if numEnd >= len(line) {
		return false, "", 0, ""
	}

	lineNum, err := strconv.Atoi(line[numStart:numEnd])
	if err != nil {
		return false, "", 0, ""
	}

	// Content starts after the second separator
	content := ""
	if numEnd+1 < len(line) {
		content = line[numEnd+1:]
	}

	return isMatch, path, lineNum, content
}

// truncateLine truncates a line to maxLen characters.
func truncateLine(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
