// Package report provides execution report generation and failure analysis.
package report

import (
	"fmt"
	"os"
	"strings"
	"time"

	execTypes "test-genie/cli/internal/execute"
	"test-genie/cli/internal/repo"
)

// StatusIcon returns an icon for the given phase status.
// Uses consistent emoji icons that match the legacy testing output.
func StatusIcon(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "success":
		return "✅"
	case "skipped":
		return "⏭️"
	case "failed", "error":
		return "❌"
	default:
		return "⏳"
	}
}

// FormatPhaseDuration formats a phase duration in seconds.
func FormatPhaseDuration(seconds float64) string {
	if seconds <= 0 {
		return "0s"
	}
	return fmt.Sprintf("%.1fs", seconds)
}

// FormatRunDuration formats the total run duration.
func FormatRunDuration(summarySeconds int, started, completed string) string {
	if summarySeconds > 0 {
		return (time.Duration(summarySeconds) * time.Second).String()
	}
	start, okStart := parseTime(started)
	complete, okComplete := parseTime(completed)
	if okStart && okComplete && complete.After(start) {
		return complete.Sub(start).String()
	}
	return "n/a"
}

// FormatTimestampShort converts an RFC3339 timestamp to a human-readable short format.
// Returns fallback if the timestamp is empty or invalid.
func FormatTimestampShort(timestamp, fallback string) string {
	t, ok := parseTime(timestamp)
	if !ok {
		return fallback
	}
	return t.Local().Format("15:04:05")
}

func parseTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// DefaultValue returns the value if non-empty, otherwise the fallback.
func DefaultValue(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

// NormalizeName lowercases and trims a name.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// DescribeLogPath returns a formatted log path with existence status.
func DescribeLogPath(path string) string {
	if path == "" {
		return ""
	}
	exists, empty := repo.FileState(path)
	switch {
	case !exists:
		return fmt.Sprintf("%s (missing)", path)
	case empty:
		return fmt.Sprintf("%s (empty)", path)
	default:
		return path
	}
}

// CleanObservations filters empty observations.
func CleanObservations(obs []string) []string {
	var cleaned []string
	for _, o := range obs {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		cleaned = append(cleaned, o)
	}
	return cleaned
}

// CountLinesContaining counts lines in a file containing a substring.
func CountLinesContaining(path, substr string) int {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, substr) {
			count++
		}
	}
	return count
}

// ReadLogSnippet reads the last maxBytes of a log file.
func ReadLogSnippet(path string, maxBytes int) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
	}
	return string(data)
}

// TailLines returns the last n lines from content.
func TailLines(content string, max int) []string {
	if max <= 0 {
		return nil
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

// FilterFailedPhases returns only failed phases.
func FilterFailedPhases(phases []execTypes.Phase) []execTypes.Phase {
	var failed []execTypes.Phase
	for _, phase := range phases {
		if strings.EqualFold(phase.Status, "passed") || strings.EqualFold(phase.Status, "skipped") {
			continue
		}
		failed = append(failed, phase)
	}
	return failed
}
