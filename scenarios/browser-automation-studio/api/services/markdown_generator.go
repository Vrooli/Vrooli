package services

import (
	"fmt"
	"strings"
	"time"
)

// GenerateTimelineMarkdown creates a human-readable markdown report from execution timeline data.
func GenerateTimelineMarkdown(timeline *ExecutionTimeline, workflowName string) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# Workflow Execution: %s\n\n", sanitizeMarkdown(workflowName)))

	// Metadata
	status := timeline.Status
	emoji := statusEmoji(status)

	sb.WriteString(fmt.Sprintf("**Status**: %s %s  \n", emoji, strings.Title(status)))

	// Duration calculation
	if timeline.CompletedAt != nil {
		duration := timeline.CompletedAt.Sub(timeline.StartedAt)
		sb.WriteString(fmt.Sprintf("**Duration**: %.1fs  \n", duration.Seconds()))
	}

	// Step completion
	completedSteps, totalSteps, failedStep := calculateStepMetrics(timeline.Frames)
	sb.WriteString(fmt.Sprintf("**Steps Completed**: %d/%d  \n", completedSteps, totalSteps))

	// Assertion count
	assertPassed, assertTotal := calculateAssertionMetrics(timeline.Frames)
	if assertTotal > 0 {
		sb.WriteString(fmt.Sprintf("**Assertions**: %d/%d passed  \n", assertPassed, assertTotal))
	}

	sb.WriteString(fmt.Sprintf("**Execution ID**: `%s`\n\n", timeline.ExecutionID))

	// Error Summary (if failed)
	if status == "failed" && failedStep >= 0 {
		frame := timeline.Frames[failedStep]
		sb.WriteString("## Error Summary\n\n")
		sb.WriteString(fmt.Sprintf("**Failed at step %d** (%s)  \n", frame.StepIndex+1, frame.NodeID))
		if frame.Error != "" {
			sb.WriteString(fmt.Sprintf("**Error**: %s\n\n", sanitizeMarkdown(frame.Error)))
		}
	}

	// Execution Timeline
	sb.WriteString("## Execution Timeline\n\n")
	for i, frame := range timeline.Frames {
		emoji := statusEmoji(frame.Status)

		duration := formatDuration(frame.TotalDurationMs)
		if duration == "" {
			duration = formatDuration(frame.DurationMs)
		}

		stepLabel := frame.StepType
		if stepLabel == "" {
			stepLabel = "unknown"
		}

		sb.WriteString(fmt.Sprintf("%d. %s **%s** `%s`%s", i+1, emoji, stepLabel, frame.NodeID, duration))

		// Add retry info if applicable
		if frame.RetryAttempt > 1 {
			sb.WriteString(fmt.Sprintf(" *(retry %d/%d)*", frame.RetryAttempt, frame.RetryMaxAttempts))
		}

		sb.WriteString("  \n")

		// Add error inline if present
		if frame.Error != "" && frame.Status == "failed" {
			sb.WriteString(fmt.Sprintf("   → Error: %s  \n", sanitizeMarkdown(frame.Error)))
		}

		// Add assertion details
		if frame.Assertion != nil && !frame.Success {
			sb.WriteString(fmt.Sprintf("   → Assertion failed: %s  \n", sanitizeMarkdown(frame.Assertion.Message)))
		}
	}
	sb.WriteString("\n")

	// Artifact Directory
	sb.WriteString("## Artifacts\n\n")
	sb.WriteString("This execution generated the following artifacts:\n\n")
	sb.WriteString("- **timeline.json** - Full machine-readable execution data\n")
	sb.WriteString("- **execution-summary.md** - Detailed step-by-step breakdown\n")
	sb.WriteString("- **console-logs.md** - Console output organized by step\n")
	sb.WriteString("- **network-activity.md** - Network requests organized by step\n")
	sb.WriteString("- **assertions.md** - Detailed assertion results\n")

	// Count screenshots
	screenshotCount := countScreenshots(timeline.Frames)
	if screenshotCount > 0 {
		sb.WriteString(fmt.Sprintf("- **screenshots/** - %d screenshots captured during execution\n", screenshotCount))
	}

	sb.WriteString("\n")

	// Debugging Hints (for failed executions)
	if status == "failed" {
		sb.WriteString("## Debugging Hints\n\n")

		// Analyze error patterns
		hasNetworkErrors, hasSelectorErrors, hasTimeouts := detectErrorPatterns(timeline.Frames)

		if hasNetworkErrors {
			sb.WriteString("- **Network Issues Detected**: Check `network-activity.md` for failed requests\n")
		}
		if hasSelectorErrors {
			sb.WriteString("- **Selector Issues Detected**: Review screenshots to verify element visibility\n")
		}
		if hasTimeouts {
			sb.WriteString("- **Timeout Issues Detected**: Consider increasing wait times or checking for async loading\n")
		}

		// Point to relevant artifacts
		if failedStep >= 0 {
			frame := timeline.Frames[failedStep]
			if frame.ConsoleLogCount > 0 {
				sb.WriteString(fmt.Sprintf("- Check console logs for step %d in `console-logs.md`\n", frame.StepIndex+1))
			}
			if frame.Screenshot != nil {
				sb.WriteString(fmt.Sprintf("- Review screenshot from failed step in `screenshots/`\n"))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateExecutionSummaryMarkdown creates a detailed step-by-step breakdown.
func GenerateExecutionSummaryMarkdown(timeline *ExecutionTimeline) string {
	var sb strings.Builder

	sb.WriteString("# Execution Summary\n\n")
	sb.WriteString("Detailed step-by-step breakdown of workflow execution.\n\n")

	for i, frame := range timeline.Frames {
		sb.WriteString(fmt.Sprintf("## Step %d: %s\n\n", i+1, frame.NodeID))

		// Metadata table
		writeMarkdownTable(&sb)
		writeTableRow(&sb, "Type", frame.StepType)
		writeTableRow(&sb, "Status", frame.Status)
		writeTableRow(&sb, "Success", fmt.Sprintf("%t", frame.Success))

		if frame.TotalDurationMs > 0 {
			writeTableRow(&sb, "Duration", fmt.Sprintf("%dms (%.2fs)", frame.TotalDurationMs, float64(frame.TotalDurationMs)/1000.0))
		} else if frame.DurationMs > 0 {
			writeTableRow(&sb, "Duration", fmt.Sprintf("%dms (%.2fs)", frame.DurationMs, float64(frame.DurationMs)/1000.0))
		}

		if ts := formatTimestamp(frame.StartedAt); ts != "" {
			writeTableRow(&sb, "Started", ts)
		}

		if ts := formatTimestamp(frame.CompletedAt); ts != "" {
			writeTableRow(&sb, "Completed", ts)
		}

		if frame.FinalURL != "" {
			writeTableRow(&sb, "Final URL", sanitizeMarkdown(frame.FinalURL))
		}

		if frame.RetryAttempt > 0 {
			writeTableRow(&sb, "Retry Attempt", fmt.Sprintf("%d/%d", frame.RetryAttempt, frame.RetryMaxAttempts))
		}

		sb.WriteString("\n")

		// Error details
		if frame.Error != "" {
			sb.WriteString("### Error\n\n")
			sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", frame.Error))
		}

		// Assertion details
		if frame.Assertion != nil {
			sb.WriteString("### Assertion\n\n")
			sb.WriteString(fmt.Sprintf("**Success**: %t  \n", frame.Assertion.Success))
			if frame.Assertion.Message != "" {
				sb.WriteString(fmt.Sprintf("**Message**: %s  \n", sanitizeMarkdown(frame.Assertion.Message)))
			}
			if frame.Assertion.Actual != nil {
				sb.WriteString(fmt.Sprintf("**Actual**: `%v`  \n", frame.Assertion.Actual))
			}
			if frame.Assertion.Expected != nil {
				sb.WriteString(fmt.Sprintf("**Expected**: `%v`  \n", frame.Assertion.Expected))
			}
			sb.WriteString("\n")
		}

		// Retry history
		if len(frame.RetryHistory) > 0 {
			sb.WriteString("### Retry History\n\n")
			for _, retry := range frame.RetryHistory {
				emoji := statusEmoji(map[bool]string{true: "completed", false: "failed"}[retry.Success])
				sb.WriteString(fmt.Sprintf("- %s Attempt %d: ", emoji, retry.Attempt))
				if retry.Success {
					sb.WriteString(fmt.Sprintf("Success (%dms)\n", retry.DurationMs))
				} else {
					sb.WriteString(fmt.Sprintf("Failed - %s (%dms)\n", sanitizeMarkdown(retry.Error), retry.DurationMs))
				}
			}
			sb.WriteString("\n")
		}

		// Extracted data
		if frame.ExtractedDataPreview != nil {
			sb.WriteString("### Extracted Data\n\n")
			sb.WriteString(fmt.Sprintf("```json\n%v\n```\n\n", frame.ExtractedDataPreview))
		}

		// Artifacts
		if len(frame.Artifacts) > 0 || frame.Screenshot != nil {
			sb.WriteString("### Artifacts\n\n")
			if frame.Screenshot != nil {
				sb.WriteString(fmt.Sprintf("- Screenshot: `screenshots/step-%02d-%s.png`\n", i+1, frame.NodeID))
			}
			for _, artifact := range frame.Artifacts {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", artifact.Type, artifact.Label))
			}
			sb.WriteString("\n")
		}

		// Console and network counts
		if frame.ConsoleLogCount > 0 || frame.NetworkEventCount > 0 {
			sb.WriteString("### Telemetry\n\n")
			if frame.ConsoleLogCount > 0 {
				sb.WriteString(fmt.Sprintf("- Console logs: %d entries (see console-logs.md)\n", frame.ConsoleLogCount))
			}
			if frame.NetworkEventCount > 0 {
				sb.WriteString(fmt.Sprintf("- Network events: %d requests (see network-activity.md)\n", frame.NetworkEventCount))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("---\n\n")
	}

	return sb.String()
}

// GenerateConsoleLogsMarkdown formats console logs by step.
func GenerateConsoleLogsMarkdown(timeline *ExecutionTimeline) string {
	var sb strings.Builder

	sb.WriteString("# Console Logs\n\n")
	sb.WriteString("Console output organized by execution step.\n\n")

	for i, frame := range timeline.Frames {
		// Find console artifacts for this step
		var consoleLogs []TimelineArtifact
		for _, artifact := range frame.Artifacts {
			if artifact.Type == "console" {
				consoleLogs = append(consoleLogs, artifact)
			}
		}

		if len(consoleLogs) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## Step %d: %s\n\n", i+1, frame.NodeID))

		for _, artifact := range consoleLogs {
			if artifact.Payload == nil {
				continue
			}

			if entries, ok := artifact.Payload["entries"].([]interface{}); ok {
				for _, entry := range entries {
					if entryMap, ok := entry.(map[string]interface{}); ok {
						timestamp := ""
						if ts, ok := entryMap["timestamp"].(float64); ok {
							t := time.UnixMilli(int64(ts))
							timestamp = t.Format("15:04:05.000")
						}

						logType := "LOG"
						if t, ok := entryMap["type"].(string); ok {
							logType = strings.ToUpper(t)
						}

						text := ""
						if t, ok := entryMap["text"].(string); ok {
							text = t
						}

						sb.WriteString(fmt.Sprintf("`[%s]` [%s] %s  \n", timestamp, logType, sanitizeMarkdown(text)))
					}
				}
			}
		}

		sb.WriteString("\n")
	}

	if sb.Len() == len("# Console Logs\n\nConsole output organized by execution step.\n\n") {
		sb.WriteString("*No console logs captured during execution.*\n")
	}

	return sb.String()
}

// GenerateNetworkActivityMarkdown formats network requests by step.
func GenerateNetworkActivityMarkdown(timeline *ExecutionTimeline) string {
	var sb strings.Builder

	sb.WriteString("# Network Activity\n\n")
	sb.WriteString("Network requests organized by execution step.\n\n")

	for i, frame := range timeline.Frames {
		// Find network artifacts for this step
		var networkArtifacts []TimelineArtifact
		for _, artifact := range frame.Artifacts {
			if artifact.Type == "network" {
				networkArtifacts = append(networkArtifacts, artifact)
			}
		}

		if len(networkArtifacts) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## Step %d: %s\n\n", i+1, frame.NodeID))

		for _, artifact := range networkArtifacts {
			if artifact.Payload == nil {
				continue
			}

			if events, ok := artifact.Payload["events"].([]interface{}); ok {
				for _, event := range events {
					if eventMap, ok := event.(map[string]interface{}); ok {
						eventType := ""
						if t, ok := eventMap["type"].(string); ok {
							eventType = t
						}

						if eventType == "request" {
							method := ""
							if m, ok := eventMap["method"].(string); ok {
								method = m
							}

							url := ""
							if u, ok := eventMap["url"].(string); ok {
								url = u
							}

							sb.WriteString(fmt.Sprintf("**%s** `%s`  \n", method, sanitizeMarkdown(url)))

						} else if eventType == "response" {
							status := 0
							if s, ok := eventMap["status"].(float64); ok {
								status = int(s)
							}

							url := ""
							if u, ok := eventMap["url"].(string); ok {
								url = u
							}

							statusEmoji := "✅"
							if status >= 400 {
								statusEmoji = "❌"
							} else if status >= 300 {
								statusEmoji = "↪️"
							}

							sb.WriteString(fmt.Sprintf("  → %s %d `%s`  \n", statusEmoji, status, sanitizeMarkdown(url)))
						}
					}
				}
			}
		}

		sb.WriteString("\n")
	}

	if sb.Len() == len("# Network Activity\n\nNetwork requests organized by execution step.\n\n") {
		sb.WriteString("*No network activity captured during execution.*\n")
	}

	return sb.String()
}

// GenerateAssertionsMarkdown creates detailed assertion results.
func GenerateAssertionsMarkdown(timeline *ExecutionTimeline) string {
	var sb strings.Builder

	sb.WriteString("# Assertions\n\n")
	sb.WriteString("Detailed results of all assertions performed during execution.\n\n")

	assertionCount := 0
	for i, frame := range timeline.Frames {
		if frame.Assertion == nil {
			continue
		}

		assertionCount++

		emoji := statusEmoji(map[bool]string{true: "completed", false: "failed"}[frame.Assertion.Success])

		sb.WriteString(fmt.Sprintf("## %s Step %d: %s\n\n", emoji, i+1, frame.NodeID))

		writeMarkdownTable(&sb)
		writeTableRow(&sb, "Success", fmt.Sprintf("%t", frame.Assertion.Success))

		if frame.Assertion.Message != "" {
			writeTableRow(&sb, "Message", sanitizeMarkdown(frame.Assertion.Message))
		}

		if frame.Assertion.Expected != nil {
			writeTableRow(&sb, "Expected", fmt.Sprintf("`%v`", frame.Assertion.Expected))
		}

		if frame.Assertion.Actual != nil {
			actualStr := fmt.Sprintf("%v", frame.Assertion.Actual)
			// Truncate very long actual values
			if len(actualStr) > 200 {
				actualStr = actualStr[:200] + "..."
			}
			writeTableRow(&sb, "Actual", fmt.Sprintf("`%s`", sanitizeMarkdown(actualStr)))
		}

		sb.WriteString("\n")

		// Add detailed actual value if it's a map/object
		if actualMap, ok := frame.Assertion.Actual.(map[string]interface{}); ok {
			sb.WriteString("### Actual Value Details\n\n")
			sb.WriteString("```json\n")
			for key, value := range actualMap {
				sb.WriteString(fmt.Sprintf("  \"%s\": %v\n", key, value))
			}
			sb.WriteString("```\n\n")
		}

		sb.WriteString("---\n\n")
	}

	if assertionCount == 0 {
		sb.WriteString("*No assertions were performed during this execution.*\n")
	}

	return sb.String()
}
