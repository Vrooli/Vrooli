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

	go func(issueID, agent string, auto bool, template string) {
		cmd := exec.Command("bash", scriptPath, "resolve", issueID, agent, projectPath, template, autoFlag)
		cmd.Dir = s.config.ScenarioRoot

		output, err := cmd.CombinedOutput()
		if err != nil {
			outputStr := string(output)
			log.Printf("Investigation script failed for issue %s: %v\nOutput: %s", issueID, err, outputStr)

			// Check if failure is due to rate limit
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

			// Not a rate limit, move to failed
			s.moveIssue(issueID, "failed")
			return
		}

		var result struct {
			IssueID       string `json:"issue_id"`
			AgentID       string `json:"agent_id"`
			Status        string `json:"status"`
			Error         string `json:"error"`
			Investigation struct {
				Report       string   `json:"report"`
				RootCause    string   `json:"root_cause"`
				SuggestedFix string   `json:"suggested_fix"`
				Confidence   int      `json:"confidence_score"`
				Affected     []string `json:"affected_files"`
				StartedAt    string   `json:"started_at"`
				CompletedAt  string   `json:"completed_at"`
			} `json:"investigation"`
			Fix struct {
				Summary            string `json:"summary"`
				ImplementationPlan string `json:"implementation_plan"`
				TestPlan           string `json:"test_plan"`
				RollbackPlan       string `json:"rollback_plan"`
				Status             string `json:"status"`
			} `json:"fix"`
		}

		if err := json.Unmarshal(output, &result); err != nil {
			log.Printf("Failed to parse investigation output for issue %s: %v", issueID, err)
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
		if result.Investigation.Report != "" {
			issue.Investigation.Report = result.Investigation.Report
		}
		if result.Investigation.RootCause != "" {
			issue.Investigation.RootCause = result.Investigation.RootCause
		}
		if result.Investigation.SuggestedFix != "" {
			issue.Investigation.SuggestedFix = result.Investigation.SuggestedFix
		}
		if result.Investigation.Confidence > 0 {
			confidence := result.Investigation.Confidence
			issue.Investigation.ConfidenceScore = &confidence
		}
		if len(result.Investigation.Affected) > 0 {
			issue.ErrorContext.AffectedFiles = result.Investigation.Affected
		}
		if result.Investigation.StartedAt != "" {
			issue.Investigation.StartedAt = result.Investigation.StartedAt
		}
		if result.Investigation.CompletedAt != "" {
			issue.Investigation.CompletedAt = result.Investigation.CompletedAt
		} else {
			issue.Investigation.CompletedAt = now.Format(time.RFC3339)
		}

		if strings.TrimSpace(result.Fix.Summary) != "" {
			issue.Fix.SuggestedFix = strings.TrimSpace(result.Fix.Summary)
		}
		if strings.TrimSpace(result.Fix.ImplementationPlan) != "" {
			issue.Fix.ImplementationPlan = strings.TrimSpace(result.Fix.ImplementationPlan)
		}
		if strings.TrimSpace(result.Fix.TestPlan) != "" {
			issue.Fix.VerificationStatus = "pending-validation"
			if issue.Metadata.Extra == nil {
				issue.Metadata.Extra = make(map[string]string)
			}
			issue.Metadata.Extra["test_plan"] = strings.TrimSpace(result.Fix.TestPlan)
		}
		if strings.TrimSpace(result.Fix.RollbackPlan) != "" {
			issue.Fix.RollbackPlan = strings.TrimSpace(result.Fix.RollbackPlan)
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

		finalStatus := "active"
		switch strings.ToLower(originalStatus) {
		case "completed", "success":
			finalStatus = "completed"
		case "failed", "error":
			finalStatus = "failed"
		case "cancelled", "canceled":
			finalStatus = "failed"
		default:
			if strings.EqualFold(result.Fix.Status, "generated") || strings.EqualFold(result.Fix.Status, "completed") {
				finalStatus = "completed"
			}
		}

		if auto && finalStatus == "active" {
			finalStatus = "completed"
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
	}(issueID, agentID, autoResolve, promptTemplate)

	return nil
}
