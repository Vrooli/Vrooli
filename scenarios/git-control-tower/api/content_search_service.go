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

// validateSearchQuery validates the query string and regex syntax.
func validateSearchQuery(query string, isRegex bool) error {
	if query == "" {
		return fmt.Errorf("query is required")
	}
	if len(query) < ContentSearchMinQueryLen {
		return fmt.Errorf("query must be at least %d characters", ContentSearchMinQueryLen)
	}
	if len(query) > ContentSearchMaxQueryLen {
		return fmt.Errorf("query must be at most %d characters", ContentSearchMaxQueryLen)
	}
	if isRegex {
		if _, err := regexp.Compile(query); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	return nil
}

// clampInt clamps a value between min and max, using defaultVal when value <= 0.
func clampInt(value, defaultVal, max int) int {
	if value <= 0 {
		value = defaultVal
	}
	if value > max {
		value = max
	}
	return value
}

// ValidateContentSearchRequest validates and normalizes the request.
func ValidateContentSearchRequest(req *ContentSearchRequest) error {
	if err := validateSearchQuery(strings.TrimSpace(req.Query), req.Regex); err != nil {
		return err
	}

	req.Limit = clampInt(req.Limit, ContentSearchDefaultLimit, ContentSearchMaxLimit)
	req.Timeout = clampInt(req.Timeout, ContentSearchDefaultTimeout, ContentSearchMaxTimeout)

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

// grepParseState tracks state across grep output lines.
type grepParseState struct {
	matches              []ContentSearchMatch
	total                int
	truncated            bool
	pendingContextBefore []string
	currentMatch         *ContentSearchMatch
	limit                int
	contextLines         int
}

func (s *grepParseState) finalizeCurrentMatch() {
	if s.currentMatch == nil {
		return
	}
	s.matches = append(s.matches, *s.currentMatch)
	s.currentMatch = nil
}

func (s *grepParseState) handleSeparator() {
	s.finalizeCurrentMatch()
	s.pendingContextBefore = nil
}

func (s *grepParseState) handleMatchLine(path string, lineNum int, content string) {
	s.total++

	if len(s.matches) >= s.limit {
		s.truncated = true
		s.finalizeCurrentMatch()
		return
	}

	s.finalizeCurrentMatch()

	s.currentMatch = &ContentSearchMatch{
		Path:       path,
		LineNumber: lineNum,
		Content:    truncateLine(content, ContentSearchMaxLineLen),
	}

	if len(s.pendingContextBefore) > 0 && s.contextLines > 0 {
		s.currentMatch.ContextBefore = strings.Join(s.pendingContextBefore, "\n")
	}
	s.pendingContextBefore = nil
}

func (s *grepParseState) handleContextLine(content string) {
	if s.contextLines <= 0 {
		return
	}
	truncatedContent := truncateLine(content, ContentSearchMaxLineLen)

	if s.currentMatch != nil {
		if s.currentMatch.ContextAfter == "" {
			s.currentMatch.ContextAfter = truncatedContent
		} else {
			s.currentMatch.ContextAfter += "\n" + truncatedContent
		}
		return
	}

	s.pendingContextBefore = append(s.pendingContextBefore, truncatedContent)
	if len(s.pendingContextBefore) > s.contextLines {
		s.pendingContextBefore = s.pendingContextBefore[len(s.pendingContextBefore)-s.contextLines:]
	}
}

// parseGrepOutput parses git grep output into ContentSearchMatch structs.
// Git grep output format with -n: file:line:content
// With context (-C): file-line-content for context, file:line:content for matches
func parseGrepOutput(output string, limit int, contextLines int) ([]ContentSearchMatch, int, bool) {
	if strings.TrimSpace(output) == "" {
		return []ContentSearchMatch{}, 0, false
	}

	state := &grepParseState{
		limit:        limit,
		contextLines: contextLines,
	}

	for _, line := range strings.Split(output, "\n") {
		if line == "" || line == "--" {
			state.handleSeparator()
			continue
		}

		isMatch, path, lineNum, content := parseGrepLine(line)
		if path == "" {
			continue
		}

		if isMatch {
			state.handleMatchLine(path, lineNum, content)
		} else {
			state.handleContextLine(content)
		}
	}

	if state.currentMatch != nil && len(state.matches) < limit {
		state.matches = append(state.matches, *state.currentMatch)
	}

	return state.matches, state.total, state.truncated
}

// isSepChar returns true if the byte is a grep separator character (: or -).
func isSepChar(b byte) bool {
	return b == ':' || b == '-'
}

// scanDigitsAfter returns the index after consuming digits starting at pos.
func scanDigitsAfter(line string, pos int) int {
	j := pos
	for j < len(line) && line[j] >= '0' && line[j] <= '9' {
		j++
	}
	return j
}

// findGrepSeparator locates the first separator position where a :number: or -number- pattern
// starts. Returns the index and whether it's a match line (: separator) vs context (- separator).
func findGrepSeparator(line string) (int, bool) {
	for i := 0; i < len(line); i++ {
		if !isSepChar(line[i]) {
			continue
		}
		j := scanDigitsAfter(line, i+1)
		if j > i+1 && j < len(line) && isSepChar(line[j]) {
			return i, line[i] == ':'
		}
	}
	return -1, false
}

// extractGrepContent extracts the line number and content after the separator index.
func extractGrepContent(line string, sepIdx int) (int, string, bool) {
	numStart := sepIdx + 1
	numEnd := numStart
	for numEnd < len(line) && line[numEnd] >= '0' && line[numEnd] <= '9' {
		numEnd++
	}
	if numEnd >= len(line) {
		return 0, "", false
	}
	lineNum, err := strconv.Atoi(line[numStart:numEnd])
	if err != nil {
		return 0, "", false
	}
	content := ""
	if numEnd+1 < len(line) {
		content = line[numEnd+1:]
	}
	return lineNum, content, true
}

// parseGrepLine parses a single line of git grep output.
// Returns: isMatch (true for match, false for context), path, lineNumber, content
func parseGrepLine(line string) (bool, string, int, string) {
	sepIdx, isMatch := findGrepSeparator(line)
	if sepIdx == -1 {
		return false, "", 0, ""
	}

	lineNum, content, ok := extractGrepContent(line, sepIdx)
	if !ok {
		return false, "", 0, ""
	}

	return isMatch, line[:sepIdx], lineNum, content
}

// truncateLine truncates a line to maxLen characters.
func truncateLine(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
