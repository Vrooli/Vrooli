package services

import (
	"fmt"
	"strings"
	"time"
)

// statusEmoji returns the appropriate emoji for a given status string.
func statusEmoji(status string) string {
	switch status {
	case "failed":
		return "❌"
	case "cancelled":
		return "⚠️"
	case "running":
		return "🔄"
	case "skipped":
		return "⏭️"
	default:
		return "✅"
	}
}

// formatDuration formats a duration in milliseconds as a human-readable string.
func formatDuration(durationMs int) string {
	if durationMs == 0 {
		return ""
	}
	return fmt.Sprintf(" (%.2fs)", float64(durationMs)/1000.0)
}

// formatTimestamp formats a time as RFC3339 or returns empty string for nil.
func formatTimestamp(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// calculateStepMetrics calculates completion and failure statistics from timeline frames.
func calculateStepMetrics(frames []TimelineFrame) (completed, total, failedStep int) {
	total = len(frames)
	failedStep = -1
	for i, frame := range frames {
		if frame.Status == "completed" {
			completed++
		} else if frame.Status == "failed" && failedStep == -1 {
			failedStep = i
		}
	}
	return completed, total, failedStep
}

// calculateAssertionMetrics calculates assertion pass/fail statistics from timeline frames.
func calculateAssertionMetrics(frames []TimelineFrame) (passed, total int) {
	for _, frame := range frames {
		if frame.StepType == "assert" {
			total++
			if frame.Success {
				passed++
			}
		}
	}
	return passed, total
}

// countScreenshots counts the number of frames with screenshots.
func countScreenshots(frames []TimelineFrame) int {
	count := 0
	for _, frame := range frames {
		if frame.Screenshot != nil {
			count++
		}
	}
	return count
}

// detectErrorPatterns analyzes errors in timeline frames and returns common patterns.
func detectErrorPatterns(frames []TimelineFrame) (hasNetworkErrors, hasSelectorErrors, hasTimeouts bool) {
	for _, frame := range frames {
		if frame.Error != "" {
			errorLower := strings.ToLower(frame.Error)
			if strings.Contains(errorLower, "network") || strings.Contains(errorLower, "404") || strings.Contains(errorLower, "500") {
				hasNetworkErrors = true
			}
			if strings.Contains(errorLower, "selector") || strings.Contains(errorLower, "element") {
				hasSelectorErrors = true
			}
			if strings.Contains(errorLower, "timeout") || strings.Contains(errorLower, "timed out") {
				hasTimeouts = true
			}
		}
	}
	return hasNetworkErrors, hasSelectorErrors, hasTimeouts
}

// writeMarkdownTable writes a markdown table header.
func writeMarkdownTable(sb *strings.Builder) {
	sb.WriteString("| Property | Value |\n")
	sb.WriteString("|----------|-------|\n")
}

// writeTableRow writes a markdown table row with proper formatting.
func writeTableRow(sb *strings.Builder, property, value string) {
	sb.WriteString(fmt.Sprintf("| **%s** | %s |\n", property, value))
}

// sanitizeMarkdown escapes markdown special characters to prevent formatting issues.
func sanitizeMarkdown(text string) string {
	// Basic escaping for common markdown characters
	text = strings.ReplaceAll(text, "|", "\\|")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", "")
	return text
}
