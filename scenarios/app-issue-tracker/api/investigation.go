package main

import (
	"context"
	"fmt"
	"log"
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

// failInvestigation marks an investigation as failed with error details
func (s *Server) failInvestigation(issueID, errorMsg, output string) {
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

	nowUTC := time.Now().UTC()

	// Store error in both investigation report and metadata
	if output != "" {
		issue.Investigation.Report = fmt.Sprintf("Investigation failed: %s\n\nOutput:\n%s", errorMsg, output)
	} else {
		issue.Investigation.Report = fmt.Sprintf("Investigation failed: %s", errorMsg)
	}
	issue.Investigation.CompletedAt = nowUTC.Format(time.RFC3339)
	issue.Metadata.UpdatedAt = nowUTC.Format(time.RFC3339)

	if issue.Metadata.Extra == nil {
		issue.Metadata.Extra = make(map[string]string)
	}
	issue.Metadata.Extra["agent_last_error"] = errorMsg
	issue.Metadata.Extra["agent_last_status"] = "failed"

	if err := s.writeIssueMetadata(issueDir, issue); err != nil {
		log.Printf("Failed to save error for issue %s: %v", issueID, err)
	}

	s.moveIssue(issueID, "failed")

	s.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   issue.Investigation.AgentID,
		Success:   false,
		EndTime:   nowUTC,
		NewStatus: "failed",
	}))
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

	// Register this process as running
	s.registerRunningProcess(issueID, agentID, now)

	go func(issueID, agent string) {
		defer s.unregisterRunningProcess(issueID)

		// Build investigation prompt
		issueDir, _, findErr := s.findIssueDirectory(issueID)
		if findErr != nil {
			log.Printf("Failed to find issue %s: %v", issueID, findErr)
			return
		}

		issue, loadErr := s.loadIssueFromDir(issueDir)
		if loadErr != nil {
			log.Printf("Failed to load issue %s: %v", issueID, loadErr)
			return
		}

		promptTemplate := buildInvestigationPromptTemplate(issue)
		fullPrompt := buildInvestigationPromptMarkdown(promptTemplate, issueID, agent, s.config.ScenarioRoot, now)

		// Get agent settings and create timeout
		settings := GetAgentSettings()
		timeoutDuration := time.Duration(settings.TimeoutSeconds) * time.Second
		startTime := time.Now()

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		// Execute Claude Code directly (ecosystem-manager pattern)
		result, err := s.executeClaudeCode(ctx, fullPrompt, issueID, startTime, timeoutDuration)

		if err != nil {
			log.Printf("Failed to execute Claude Code for issue %s: %v", issueID, err)
			s.failInvestigation(issueID, fmt.Sprintf("Execution error: %v", err), "")
			return
		}

		// Handle rate limits (move to waiting, not failed)
		if strings.Contains(result.Error, "RATE_LIMIT") {
			isRateLimit, resetTime := detectRateLimit(result.Output)
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
		}

		// Handle failures (timeout, max turns, errors)
		if !result.Success {
			log.Printf("Investigation failed for issue %s: %s", issueID, result.Error)

			// Store error metadata
			issueDir, _, findErr := s.findIssueDirectory(issueID)
			if findErr == nil {
				issue, loadErr := s.loadIssueFromDir(issueDir)
				if loadErr == nil {
					if issue.Metadata.Extra == nil {
						issue.Metadata.Extra = make(map[string]string)
					}
					issue.Metadata.Extra["agent_last_error"] = result.Error
					if result.MaxTurnsExceeded {
						issue.Metadata.Extra["max_turns_exceeded"] = "true"
					}
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

		// SUCCESS: Process the investigation report
		log.Printf("Investigation succeeded for issue %s (execution time: %v)", issueID, result.ExecutionTime.Round(time.Second))

		// Load issue to store results
		var findErr2 error
		issueDir, _, findErr2 = s.findIssueDirectory(issueID)
		if findErr2 != nil {
			log.Printf("Failed to locate issue %s after investigation: %v", issueID, findErr2)
			return
		}

		var loadErr2 error
		issue, loadErr2 = s.loadIssueFromDir(issueDir)
		if loadErr2 != nil {
			log.Printf("Failed to reload issue %s: %v", issueID, loadErr2)
			return
		}

		nowUTC := time.Now().UTC()

		// Store the investigation report (clean ecosystem-manager pattern)
		issue.Investigation.Report = result.Output
		issue.Investigation.CompletedAt = nowUTC.Format(time.RFC3339)
		issue.Metadata.UpdatedAt = nowUTC.Format(time.RFC3339)

		// Calculate investigation duration
		if issue.Investigation.StartedAt != "" {
			if parsedStart, err := time.Parse(time.RFC3339, issue.Investigation.StartedAt); err == nil {
				durationMinutes := int(time.Since(parsedStart).Minutes())
				issue.Investigation.InvestigationDurationMinutes = &durationMinutes
			}
		}

		// Clear any previous error metadata (CRITICAL: no false timeout warnings)
		if issue.Metadata.Extra == nil {
			issue.Metadata.Extra = make(map[string]string)
		}
		delete(issue.Metadata.Extra, "agent_last_error")
		delete(issue.Metadata.Extra, "max_turns_exceeded")
		issue.Metadata.Extra["agent_last_status"] = "completed"

		// Set resolved timestamp
		issue.Metadata.ResolvedAt = nowUTC.Format(time.RFC3339)

		// Save updated metadata
		if saveErr := s.writeIssueMetadata(issueDir, issue); saveErr != nil {
			log.Printf("Failed to save investigation results for issue %s: %v", issueID, saveErr)
		}

		// Move to completed status
		if moveErr := s.moveIssue(issueID, "completed"); moveErr != nil {
			log.Printf("Failed to move issue %s to completed: %v", issueID, moveErr)
		}

		log.Printf("Investigation completed successfully for issue %s", issueID)

		// Publish success event
		s.hub.Publish(NewEvent(EventAgentCompleted, AgentCompletedData{
			IssueID:   issueID,
			AgentID:   agent,
			Success:   true,
			EndTime:   nowUTC,
			NewStatus: "completed",
		}))
	}(issueID, agentID)

	return nil
}
