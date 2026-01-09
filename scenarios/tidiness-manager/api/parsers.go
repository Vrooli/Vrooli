package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Issue represents a structured lint/type issue
type Issue struct {
	Scenario string `json:"scenario"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity"`
	Tool     string `json:"tool"`
	Category string `json:"category"` // "lint" or "type"
}

// ParseLintOutput parses various linter formats into structured issues
func ParseLintOutput(scenario, tool, output string) []Issue {
	issues := []Issue{}

	// Support common formats:
	// eslint: path/file.ts:10:5: message [rule-name]
	// golangci-lint: path/file.go:10:5: message (linter)
	// Generic: file.ext:line:col: message

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		issue := parseGenericLintLine(scenario, tool, line)
		if issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues
}

// ParseTypeOutput parses type checker outputs into structured issues
func ParseTypeOutput(scenario, tool, output string) []Issue {
	issues := []Issue{}

	// TypeScript: src/file.ts(10,5): error TS2304: Message
	// Go: file.go:10:5: Message

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var issue *Issue

		// Try TypeScript format first
		issue = parseTypeScriptLine(scenario, tool, line)
		if issue != nil {
			issues = append(issues, *issue)
			continue
		}

		// Fall back to generic format
		issue = parseGenericTypeLine(scenario, tool, line)
		if issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues
}

// parseGenericLintLine handles generic format: file:line:col: message
func parseGenericLintLine(scenario, tool, line string) *Issue {
	// Regex: filepath:line:col: message [optional-rule]
	re := regexp.MustCompile(`^(.+?):(\d+):(\d+):\s*(.+)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 5 {
		return nil
	}

	file := matches[1]
	lineNum := parseInt(matches[2])
	col := parseInt(matches[3])
	message := matches[4]

	// Extract rule if present
	rule := ""
	ruleRe := regexp.MustCompile(`\[([^\]]+)\]$`)
	if ruleMatches := ruleRe.FindStringSubmatch(message); len(ruleMatches) > 1 {
		rule = ruleMatches[1]
		message = strings.TrimSpace(ruleRe.ReplaceAllString(message, ""))
	}

	// Determine severity from keywords
	severity := "warning"
	lowerMsg := strings.ToLower(message)
	if strings.Contains(lowerMsg, "error") {
		severity = "error"
	}

	return &Issue{
		Scenario: scenario,
		File:     file,
		Line:     lineNum,
		Column:   col,
		Message:  message,
		Rule:     rule,
		Severity: severity,
		Tool:     tool,
		Category: "lint",
	}
}

// parseTypeScriptLine handles TypeScript format: file.ts(line,col): error TS1234: message
func parseTypeScriptLine(scenario, tool, line string) *Issue {
	re := regexp.MustCompile(`^(.+?)\((\d+),(\d+)\):\s*(error|warning)\s*TS(\d+):\s*(.+)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 7 {
		return nil
	}

	return &Issue{
		Scenario: scenario,
		File:     matches[1],
		Line:     parseInt(matches[2]),
		Column:   parseInt(matches[3]),
		Severity: matches[4],
		Rule:     "TS" + matches[5],
		Message:  matches[6],
		Tool:     tool,
		Category: "type",
	}
}

// parseGenericTypeLine handles generic type errors: file:line:col: message
func parseGenericTypeLine(scenario, tool, line string) *Issue {
	re := regexp.MustCompile(`^(.+?):(\d+):(\d+):\s*(.+)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 5 {
		return nil
	}

	severity := "error"
	if strings.Contains(strings.ToLower(matches[4]), "warning") {
		severity = "warning"
	}

	return &Issue{
		Scenario: scenario,
		File:     matches[1],
		Line:     parseInt(matches[2]),
		Column:   parseInt(matches[3]),
		Message:  matches[4],
		Severity: severity,
		Tool:     tool,
		Category: "type",
	}
}

// parseInt safely converts string to int
func parseInt(s string) int {
	var n int
	// Ignore errors, return 0 if invalid
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}
