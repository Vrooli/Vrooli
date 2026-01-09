package artifacts

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// GenerateWorkflowReadme creates a human-readable README.md for a workflow execution.
func GenerateWorkflowReadme(result *WorkflowResult) string {
	var b strings.Builder

	// Header with status
	statusEmoji := "✅"
	statusText := "Passed"
	if !result.Success {
		statusEmoji = "❌"
		statusText = "Failed"
	}

	workflowName := filepath.Base(result.WorkflowFile)
	b.WriteString(fmt.Sprintf("# %s Playbook Execution Results\n\n", workflowName))
	b.WriteString(fmt.Sprintf("**Status:** %s %s\n\n", statusEmoji, statusText))

	// Execution Info Table
	b.WriteString("## Execution Information\n\n")
	b.WriteString("| Property | Value |\n")
	b.WriteString("|----------|-------|\n")
	b.WriteString(fmt.Sprintf("| Workflow | `%s` |\n", result.WorkflowFile))
	if result.Description != "" {
		b.WriteString(fmt.Sprintf("| Description | %s |\n", result.Description))
	}
	b.WriteString(fmt.Sprintf("| Timestamp | %s |\n", result.Timestamp.Format("2006-01-02 15:04:05 UTC")))
	if result.DurationMs > 0 {
		b.WriteString(fmt.Sprintf("| Duration | %dms |\n", result.DurationMs))
	}
	if result.ExecutionID != "" {
		b.WriteString(fmt.Sprintf("| Execution ID | `%s` |\n", result.ExecutionID))
	}
	b.WriteString("\n")

	// Requirements Coverage
	if len(result.Requirements) > 0 {
		b.WriteString("## Requirements Validated\n\n")
		for _, req := range result.Requirements {
			reqStatus := "✅"
			if !result.Success {
				reqStatus = "❌"
			}
			b.WriteString(fmt.Sprintf("- %s `%s`\n", reqStatus, req))
		}
		b.WriteString("\n")
	}

	// Execution Summary
	b.WriteString("## Execution Summary\n\n")
	if result.Summary.TotalSteps > 0 {
		b.WriteString(fmt.Sprintf("- **Total Steps:** %d\n", result.Summary.TotalSteps))
	}
	if result.Summary.TotalAsserts > 0 {
		passRate := float64(result.Summary.AssertsPassed) / float64(result.Summary.TotalAsserts) * 100
		b.WriteString(fmt.Sprintf("- **Assertions:** %d/%d passed (%.0f%%)\n",
			result.Summary.AssertsPassed, result.Summary.TotalAsserts, passRate))
	}
	b.WriteString("\n")

	// Error Details (if failed)
	if result.Error != "" {
		b.WriteString("## Error Details\n\n")
		b.WriteString("```\n")
		b.WriteString(result.Error)
		b.WriteString("\n```\n\n")
	}

	// Assertion Details (if available)
	if result.ParsedSummary != nil && len(result.ParsedSummary.Assertions) > 0 {
		b.WriteString("## Assertion Results\n\n")
		b.WriteString("| Step | Type | Status | Details |\n")
		b.WriteString("|------|------|--------|--------|\n")
		for _, a := range result.ParsedSummary.Assertions {
			status := "✅ Passed"
			if !a.Passed {
				status = "❌ Failed"
			}
			details := ""
			if a.Selector != "" {
				details = fmt.Sprintf("`%s`", truncateString(a.Selector, 40))
			}
			if a.Message != "" && !a.Passed {
				details = truncateString(a.Message, 50)
			}
			if a.Error != "" {
				details = truncateString(a.Error, 50)
			}
			b.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n",
				a.StepIndex, a.AssertionType, status, details))
		}
		b.WriteString("\n")
	}

	// Failed Frame Details
	if result.ParsedSummary != nil && result.ParsedSummary.FailedFrame != nil {
		frame := result.ParsedSummary.FailedFrame
		b.WriteString("## Failed Step Details\n\n")
		b.WriteString(fmt.Sprintf("- **Step Index:** %d\n", frame.StepIndex))
		b.WriteString(fmt.Sprintf("- **Step Type:** %s\n", frame.StepType))
		if frame.NodeID != "" {
			b.WriteString(fmt.Sprintf("- **Node ID:** `%s`\n", frame.NodeID))
		}
		if frame.Error != "" {
			b.WriteString(fmt.Sprintf("- **Error:** %s\n", frame.Error))
		}
		if frame.FinalURL != "" {
			b.WriteString(fmt.Sprintf("- **URL:** %s\n", frame.FinalURL))
		}
		b.WriteString("\n")
	}

	// Collected Artifacts
	b.WriteString("## Collected Artifacts\n\n")
	hasArtifacts := false

	if len(result.Artifacts.Screenshots) > 0 {
		hasArtifacts = true
		b.WriteString("### Screenshots\n\n")
		b.WriteString(fmt.Sprintf("Captured %d screenshots during execution.\n\n", len(result.Artifacts.Screenshots)))
		for i, ss := range result.Artifacts.Screenshots {
			filename := filepath.Base(ss)
			b.WriteString(fmt.Sprintf("%d. [%s](./%s)\n", i+1, filename, filepath.Join("screenshots", filename)))
		}
		b.WriteString("\n")
	}

	if result.Artifacts.Timeline != "" {
		hasArtifacts = true
		b.WriteString("### Timeline\n\n")
		b.WriteString("Full execution timeline with step-by-step details.\n\n")
		b.WriteString(fmt.Sprintf("📋 [timeline.json](./%s)\n\n", filepath.Base(result.Artifacts.Timeline)))
	}

	if result.Artifacts.Console != "" {
		hasArtifacts = true
		b.WriteString("### Console Logs\n\n")
		b.WriteString("Browser console output captured during execution.\n\n")
		b.WriteString(fmt.Sprintf("📝 [console.json](./%s)\n\n", filepath.Base(result.Artifacts.Console)))
	}

	if result.Artifacts.DOM != "" {
		hasArtifacts = true
		b.WriteString("### DOM Snapshot\n\n")
		b.WriteString("Final HTML of the page at the end of execution (or at failure).\n\n")
		b.WriteString(fmt.Sprintf("📄 [dom.html](./%s)\n\n", filepath.Base(result.Artifacts.DOM)))
	}

	if result.Artifacts.Assertions != "" {
		hasArtifacts = true
		b.WriteString("### Assertion Details\n\n")
		b.WriteString("Structured assertion results for programmatic analysis.\n\n")
		b.WriteString(fmt.Sprintf("✓ [assertions.json](./%s)\n\n", filepath.Base(result.Artifacts.Assertions)))
	}

	if result.Artifacts.Latest != "" {
		hasArtifacts = true
		b.WriteString("### Result Summary\n\n")
		b.WriteString("Machine-readable execution result summary.\n\n")
		b.WriteString(fmt.Sprintf("🔧 [latest.json](./%s)\n\n", filepath.Base(result.Artifacts.Latest)))
	}

	if !hasArtifacts {
		b.WriteString("*No artifacts were collected for this execution.*\n\n")
	}

	// Troubleshooting (for failures)
	if !result.Success {
		b.WriteString("## Troubleshooting\n\n")
		b.WriteString("### Debugging Steps\n\n")
		b.WriteString("1. **Review Screenshots:** Check the step screenshots to see the visual state at each step\n")
		b.WriteString("2. **Check Console Logs:** Look for JavaScript errors or warnings in console.json\n")
		b.WriteString("3. **Inspect DOM:** Review dom.html for unexpected page content\n")
		b.WriteString("4. **Review Timeline:** Check timeline.json for detailed step-by-step execution data\n")
		b.WriteString("5. **Check Assertions:** Review assertions.json for expected vs actual values\n\n")

		if result.ParsedSummary != nil && result.ParsedSummary.FailedFrame != nil {
			frame := result.ParsedSummary.FailedFrame
			b.WriteString("### Possible Causes\n\n")

			switch frame.StepType {
			case "navigate":
				b.WriteString("- The target URL may be incorrect or unreachable\n")
				b.WriteString("- The scenario may not be running\n")
				b.WriteString("- Network issues may be preventing page load\n")
			case "click":
				b.WriteString("- The element selector may not match any element\n")
				b.WriteString("- The element may be hidden, disabled, or covered by another element\n")
				b.WriteString("- The page may not have finished loading\n")
			case "assert":
				b.WriteString("- The expected condition may not be met\n")
				b.WriteString("- The selector may not match the expected element\n")
				b.WriteString("- The page content may have changed unexpectedly\n")
			case "wait":
				b.WriteString("- The wait condition may have timed out\n")
				b.WriteString("- The expected element may never appear\n")
			case "type", "input":
				b.WriteString("- The input field may not be found or not be editable\n")
				b.WriteString("- The element may be hidden or disabled\n")
			default:
				b.WriteString("- Review the error message for specific details\n")
				b.WriteString("- Check the timeline for the exact failure point\n")
			}
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("*Generated by test-genie playbooks phase at %s*\n",
		time.Now().UTC().Format("2006-01-02 15:04:05 UTC")))

	return b.String()
}

// truncateString truncates a string to maxLen, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
