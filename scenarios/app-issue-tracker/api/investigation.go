package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const investigationPromptPrefix = `# Issue Investigation Request

You are a senior software engineer investigating a reported issue. Please analyze the codebase and provide a comprehensive investigation report.

## Task
`

const investigationPromptSuffix = `

## Investigation Steps
1. **Analyze the codebase** at the specified path
2. **Identify the root cause** of the issue
3. **Examine related files** and dependencies
4. **Check for similar patterns** in the codebase
5. **Provide actionable recommendations**

## Expected Output Format
Please structure your response as follows:

### Investigation Summary
Brief overview of the issue and investigation approach.

### Root Cause Analysis
Detailed explanation of what is causing the issue.

### Affected Components
List of files, functions, or systems impacted.

### Recommended Solutions
Prioritized list of potential fixes with implementation details.

### Testing Strategy
How to verify the fix and prevent regression.

### Related Issues
Any similar issues or patterns to watch for.

### Confidence Level
Rate your confidence in the analysis (1-10) and explain any uncertainties.

## Context
- Issue ID: %s
- Agent ID: %s
- Project Path: %s
- Timestamp: %s

Please begin your investigation now.
`

// buildInvestigationPromptTemplate builds a prompt template from an issue
func buildInvestigationPromptTemplate(issue *Issue) string {
	if issue == nil {
		return "Perform a full investigation and resolution for the reported issue."
	}

	title := strings.TrimSpace(issue.Title)
	if title == "" {
		title = strings.TrimSpace(issue.ID)
	}
	if title == "" {
		title = "Unspecified issue"
	}

	prompt := fmt.Sprintf("Perform a full investigation and resolution for issue: %s", title)

	if msg := strings.TrimSpace(issue.ErrorContext.ErrorMessage); msg != "" {
		prompt += fmt.Sprintf(". Error: %s", msg)
	}

	return prompt
}

// buildInvestigationPromptMarkdown builds a complete markdown investigation prompt
func buildInvestigationPromptMarkdown(template, issueID, agentID, projectPath, timestamp string) string {
	issueRef := strings.TrimSpace(issueID)
	if issueRef == "" {
		issueRef = "preview-issue"
	}

	agentRef := strings.TrimSpace(agentID)
	if agentRef == "" {
		agentRef = "unified-resolver"
	}

	projectRef := strings.TrimSpace(projectPath)
	if projectRef == "" {
		projectRef = "(not specified)"
	}

	timeRef := strings.TrimSpace(timestamp)
	if timeRef == "" {
		timeRef = time.Now().UTC().Format(time.RFC3339)
	}

	var builder strings.Builder
	builder.WriteString(investigationPromptPrefix)
	builder.WriteString(template)
	builder.WriteString(fmt.Sprintf(investigationPromptSuffix, issueRef, agentRef, projectRef, timeRef))
	return builder.String()
}

// detectRateLimit checks if script output indicates a rate limit
// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mGKHf]`)
	return ansiRegex.ReplaceAllString(str, "")
}

// extractJSON finds and extracts the first JSON object from a string that may contain log messages
func extractJSON(str string) string {
	// Find the first '{' and matching '}'
	start := strings.Index(str, "{")
	if start == -1 {
		return ""
	}

	// Count braces to find the matching closing brace
	depth := 0
	inString := false
	escape := false

	for i := start; i < len(str); i++ {
		c := str[i]

		if escape {
			escape = false
			continue
		}

		if c == '\\' {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return str[start : i+1]
			}
		}
	}

	return ""
}

func detectRateLimit(output string) (bool, string) {
	outputLower := strings.ToLower(output)

	// Check for rate limit keywords
	if strings.Contains(outputLower, "rate limit") ||
		strings.Contains(outputLower, "rate_limit") ||
		strings.Contains(outputLower, "quota exceeded") ||
		strings.Contains(outputLower, "too many requests") ||
		strings.Contains(outputLower, "429") {

		// Try to extract reset time from output
		resetTime := ""

		// Look for ISO timestamp patterns
		isoPattern := regexp.MustCompile(`20\d{2}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
		if match := isoPattern.FindString(output); match != "" {
			resetTime = match
		} else {
			// Default to 5 minutes from now
			resetTime = time.Now().Add(5 * time.Minute).Format(time.RFC3339)
		}

		return true, resetTime
	}

	return false, ""
}

// triggerInvestigation runs an investigation for a single issue
func (s *Server) triggerInvestigation(issueID, agentID string, autoResolve bool) error {
	issueDir, currentFolder, err := s.findIssueDirectory(issueID)
	if err != nil {
		return fmt.Errorf("issue not found: %w", err)
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		return fmt.Errorf("failed to load issue: %w", err)
	}

	if agentID == "" {
		agentID = "unified-resolver"
	}

	log.Printf("Starting investigation for issue %s (agent: %s, auto-resolve: %v)", issueID, agentID, autoResolve)

	now := time.Now().UTC().Format(time.RFC3339)
	issue.Investigation.AgentID = agentID
	issue.Investigation.StartedAt = now
	issue.Metadata.UpdatedAt = now

	if err := s.writeIssueMetadata(issueDir, issue); err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	if currentFolder != "active" {
		if err := s.moveIssue(issueID, "active"); err != nil {
			return fmt.Errorf("failed to move issue to active: %w", err)
		}
	}

	promptTemplate := buildInvestigationPromptTemplate(issue)
	scriptPath := filepath.Join(s.config.ScenarioRoot, "scripts", "claude-investigator.sh")
	projectPath := s.config.ScenarioRoot

	autoFlag := "false"
	if autoResolve {
		autoFlag = "true"
	}

	// Register this process as running
	s.registerRunningProcess(issueID, agentID, now)

	go func(issueID, agent string, auto bool, template string) {
		defer s.unregisterRunningProcess(issueID)

		cmd := exec.Command("bash", scriptPath, "resolve", issueID, agent, projectPath, template, autoFlag)
		cmd.Dir = s.config.ScenarioRoot

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// CRITICAL: resource-claude-code exit codes cannot be trusted (ecosystem-manager lesson)
		// The agent may exit non-zero even when it successfully completes its work:
		// - Test commands that fail during investigation
		// - Permission warnings on non-critical files
		// - Shell error handling for edge cases
		// - Timeout signals after agent finishes useful work
		//
		// Solution: Check output QUALITY, not exit codes. Only treat as failure if BOTH
		// non-zero exit AND no valid report generated.

		// First, always check for rate limit (regardless of exit code)
		isRateLimit, resetTime := detectRateLimit(outputStr)
		if isRateLimit {
			log.Printf("Rate limit detected for issue %s, moving to waiting status until %s", issueID, resetTime)

			// Load issue and update metadata with rate limit info
			issueDir, _, findErr := s.findIssueDirectory(issueID)
			if findErr == nil {
				issue, loadErr := s.loadIssueFromDir(issueDir)
				if loadErr == nil {
					if issue.Metadata.Extra == nil {
						issue.Metadata.Extra = make(map[string]string)
					}
					issue.Metadata.Extra["rate_limit_until"] = resetTime
					issue.Metadata.Extra["rate_limit_agent"] = agent
					s.writeIssueMetadata(issueDir, issue)
				}
			}

			// Move to waiting status instead of failed
			s.moveIssue(issueID, "waiting")
			return
		}

		// STRATEGY CHANGE: Don't check raw output for structure.
		// Always try to parse JSON and check the report field.
		// This ensures we show user what happened even on errors.
		if err != nil {
			log.Printf("Script exited non-zero for issue %s: %v (will check JSON for actual report quality)", issueID, err)
		}

		// Strip ANSI escape codes before parsing JSON
		cleanedOutput := stripANSI(outputStr)

		// Extract JSON portion from output (may contain log messages before/after JSON)
		jsonOutput := extractJSON(cleanedOutput)
		if jsonOutput == "" {
			log.Printf("No JSON found in output for issue %s", issueID)
			log.Printf("Output preview (first 500 chars): %s", cleanedOutput[:min(500, len(cleanedOutput))])

			// Build user-visible error with actual output
			errorMsg := fmt.Sprintf("Agent execution failed: no valid JSON response")
			if err != nil {
				errorMsg = fmt.Sprintf("Agent execution failed: %v", err)
			}
			if len(cleanedOutput) > 0 {
				if len(cleanedOutput) > 500 {
					errorMsg = fmt.Sprintf("%s\n\nAgent output:\n%s...", errorMsg, cleanedOutput[:500])
				} else {
					errorMsg = fmt.Sprintf("%s\n\nAgent output:\n%s", errorMsg, cleanedOutput)
				}
			}

			// Save error and move to failed
			issueDir, _, findErr := s.findIssueDirectory(issueID)
			if findErr == nil {
				issue, loadErr := s.loadIssueFromDir(issueDir)
				if loadErr == nil {
					if issue.Metadata.Extra == nil {
						issue.Metadata.Extra = make(map[string]string)
					}
					issue.Metadata.Extra["agent_last_error"] = errorMsg
					issue.Metadata.Extra["agent_last_status"] = "no_json_output"
					issue.Metadata.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
					s.writeIssueMetadata(issueDir, issue)
				}
			}

			s.moveIssue(issueID, "failed")
			s.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
				IssueID:   issueID,
				AgentID:   agent,
				Success:   false,
				EndTime:   time.Now(),
				NewStatus: "failed",
			}))
			return
		}

		var result struct {
			IssueID       string `json:"issue_id"`
			AgentID       string `json:"agent_id"`
			Status        string `json:"status"`
			Error         string `json:"error"`
			Investigation struct {
				Report            string `json:"report"`
				StartedAt         string `json:"started_at"`
				CompletedAt       string `json:"completed_at"`
				MaxTurnsExceeded  bool   `json:"max_turns_exceeded"`
				ExitCode          int    `json:"exit_code"`
			} `json:"investigation"`
		}

		if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
			log.Printf("Failed to parse investigation JSON for issue %s: %v", issueID, err)
			log.Printf("Raw output (first 500 chars): %s", cleanedOutput[:min(500, len(cleanedOutput))])

			// Load issue and mark as failed with parsing error
			issueDir, _, findErr := s.findIssueDirectory(issueID)
			if findErr != nil {
				log.Printf("Cannot locate issue %s to mark failed: %v", issueID, findErr)
				return
			}

			issue, loadErr := s.loadIssueFromDir(issueDir)
			if loadErr != nil {
				log.Printf("Cannot load issue %s to mark failed: %v", issueID, loadErr)
				return
			}

			// Store error information in investigation
			errorSummary := fmt.Sprintf("Investigation failed: output parsing error - %v", err)
			issue.Investigation.Report = fmt.Sprintf("Investigation failed due to output parsing error: %v\n\nRaw output:\n%s", err, cleanedOutput)
			issue.Investigation.CompletedAt = time.Now().UTC().Format(time.RFC3339)
			issue.Metadata.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

			// Also store in metadata.extra so it shows in the UI card
			if issue.Metadata.Extra == nil {
				issue.Metadata.Extra = make(map[string]string)
			}
			issue.Metadata.Extra["agent_last_error"] = errorSummary
			issue.Metadata.Extra["agent_last_status"] = "parsing_failed"

			if err := s.writeIssueMetadata(issueDir, issue); err != nil {
				log.Printf("Failed to save parsing error for issue %s: %v", issueID, err)
			}

			// Move to failed status
			s.moveIssue(issueID, "failed")

			// Publish agent failed event
			s.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
				IssueID:   issueID,
				AgentID:   agent,
				Success:   false,
				EndTime:   time.Now(),
				NewStatus: "failed",
			}))

			return
		}

		issueDir, _, err := s.findIssueDirectory(issueID)
		if err != nil {
			log.Printf("Failed to locate issue %s after investigation: %v", issueID, err)
			return
		}

		issue, err := s.loadIssueFromDir(issueDir)
		if err != nil {
			log.Printf("Failed to reload issue %s: %v", issueID, err)
			return
		}

		now := time.Now().UTC()

		// Store the full investigation report (unified-resolver.md includes fixes)
		if result.Investigation.Report != "" {
			issue.Investigation.Report = result.Investigation.Report
		}
		if result.Investigation.StartedAt != "" {
			issue.Investigation.StartedAt = result.Investigation.StartedAt
		}
		if result.Investigation.CompletedAt != "" {
			issue.Investigation.CompletedAt = result.Investigation.CompletedAt
		} else {
			issue.Investigation.CompletedAt = now.Format(time.RFC3339)
		}

		issue.Metadata.UpdatedAt = now.Format(time.RFC3339)

		if issue.Investigation.StartedAt != "" {
			if startTime, err := time.Parse(time.RFC3339, issue.Investigation.StartedAt); err == nil {
				duration := int(time.Since(startTime).Minutes())
				issue.Investigation.InvestigationDurationMinutes = &duration
			}
		}

		if err := s.writeIssueMetadata(issueDir, issue); err != nil {
			log.Printf("Failed to save investigation results for issue %s: %v", issueID, err)
		}

		if issue.Metadata.Extra == nil {
			issue.Metadata.Extra = make(map[string]string)
		}

		originalStatus := strings.TrimSpace(result.Status)
		if originalStatus == "" {
			originalStatus = "active"
		}
		issue.Metadata.Extra["agent_last_status"] = strings.ToLower(originalStatus)

		trimmedErr := strings.TrimSpace(result.Error)
		if trimmedErr != "" {
			issue.Metadata.Extra["agent_last_error"] = trimmedErr
		} else {
			delete(issue.Metadata.Extra, "agent_last_error")
		}

		// Handle max turns exceeded (ecosystem-manager pattern)
		if result.Investigation.MaxTurnsExceeded {
			issue.Metadata.Extra["max_turns_exceeded"] = "true"
			log.Printf("Issue %s: MAX_TURNS limit reached - output may be partial", issueID)
		}

		// Check report quality from parsed JSON (ecosystem-manager pattern)
		reportContent := strings.TrimSpace(result.Investigation.Report)
		hasValidReport := false
		if len(reportContent) > 500 {
			lowerReport := strings.ToLower(reportContent)
			hasValidReport = strings.Contains(lowerReport, "investigation summary") ||
				strings.Contains(lowerReport, "root cause") ||
				strings.Contains(lowerReport, "remediation") ||
				strings.Contains(lowerReport, "validation plan") ||
				strings.Contains(lowerReport, "confidence assessment")
		}

		// Log report quality for debugging
		if result.Investigation.ExitCode != 0 {
			log.Printf("Agent exit code %d for issue %s: report length=%d, hasValidReport=%v",
				result.Investigation.ExitCode, issueID, len(reportContent), hasValidReport)
			if !hasValidReport && len(reportContent) > 0 {
				// Show what we got if it's not a valid structured report
				preview := reportContent
				if len(preview) > 500 {
					preview = preview[:500] + "..."
				}
				log.Printf("Report preview: %s", preview)
			}
		}

		finalStatus := "active"
		switch strings.ToLower(originalStatus) {
		case "completed", "success":
			finalStatus = "completed"
		case "failed", "error":
			// Only fail if truly no valid report
			if hasValidReport && result.Investigation.ExitCode != 0 {
				// Override failure - agent produced valid work despite non-zero exit
				log.Printf("Overriding failure status for issue %s: exit code %d but valid report exists",
					issueID, result.Investigation.ExitCode)
				finalStatus = "completed"
			} else {
				finalStatus = "failed"
			}
		case "cancelled", "canceled":
			finalStatus = "failed"
		default:
			// Check if we have a valid report to determine status
			if hasValidReport {
				finalStatus = "completed"
			} else if result.Investigation.ExitCode != 0 {
				finalStatus = "failed"
			} else {
				finalStatus = "completed"
			}
		}

		if finalStatus == "completed" && strings.TrimSpace(issue.Metadata.ResolvedAt) == "" {
			issue.Metadata.ResolvedAt = now.Format(time.RFC3339)
			if err := s.writeIssueMetadata(issueDir, issue); err != nil {
				log.Printf("Failed to set resolution timestamp for issue %s: %v", issueID, err)
			}
		}

		if finalStatus != "active" {
			if err := s.moveIssue(issueID, finalStatus); err != nil {
				log.Printf("Failed to move issue %s to %s: %v", issueID, finalStatus, err)
			}
		}

		log.Printf("Investigation completed for issue %s: status=%s", issueID, finalStatus)

		// Publish agent completed event
		s.hub.Publish(NewEvent(EventAgentCompleted, AgentCompletedData{
			IssueID:   issueID,
			AgentID:   agent,
			Success:   finalStatus == "completed",
			EndTime:   now,
			NewStatus: finalStatus,
		}))
	}(issueID, agentID, autoResolve, promptTemplate)

	return nil
}
