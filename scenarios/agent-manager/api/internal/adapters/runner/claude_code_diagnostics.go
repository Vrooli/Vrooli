package runner

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Diagnostic helpers used by the Claude Code runner to enrich
// execution_error events and surface stream-stall information.

// maxStderrTailBytes bounds the amount of stderr we attach to an error event.
// Keeps payload sizes predictable without silently dropping the most recent
// (usually most informative) output from the child process.
const maxStderrTailBytes = 2048

// streamIdleHeartbeatThreshold is the stream-stall duration at which the
// heartbeat goroutine starts emitting debug log events. Kept strictly less
// than DefaultStreamIdleTimeout (which is 0 — disabled — by default, but
// operators may raise it per run). Exposed as a var for test injection.
//
// Note: expressed as an int millisecond so tests can shrink it without
// waiting multiple seconds.
var streamIdleHeartbeatMillis int64 = 30_000

// streamIdleHeartbeatTickMillis is how often the heartbeat goroutine wakes
// up to evaluate whether the threshold has been crossed. Kept aggressive
// enough to emit promptly after a gap crosses the threshold.
var streamIdleHeartbeatTickMillis int64 = 2_000

// secretRedactors strips obvious credential patterns out of stderr before
// we attach the tail to an event. Not a full DLP pass — just the common
// ones that have historically leaked via CLI wrappers.
var secretRedactors = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-]{8,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9_\-]{8,}`),
	regexp.MustCompile(`(?i)api[_-]?key[=:\s]+[A-Za-z0-9._\-]{8,}`),
}

// redactSecrets replaces credential-shaped substrings with "<redacted>".
func redactSecrets(s string) string {
	for _, re := range secretRedactors {
		s = re.ReplaceAllString(s, "<redacted>")
	}
	return s
}

// tailBytesUTF8Safe returns the last max bytes of s, rewound to the nearest
// UTF-8 rune boundary so we never emit half a multi-byte character.
// Returns s unchanged if it already fits.
func tailBytesUTF8Safe(s string, max int) string {
	if len(s) <= max {
		return s
	}
	start := len(s) - max
	// Rewind to the next rune boundary.
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}

// buildErrorDetails collects the structured context we can derive from a
// Claude Code `result` stream event when `is_error` is true. Exposed for
// testability via export_test.go.
func buildErrorDetails(subtype string, numTurns int, durationMs int, sessionID, resultText, stderrTail string) map[string]interface{} {
	details := map[string]interface{}{
		"subtype":     subtype,
		"num_turns":   numTurns,
		"duration_ms": durationMs,
		"result_text": resultText,
	}
	if sessionID != "" {
		details["session_id"] = sessionID
	}
	if stderrTail != "" {
		details["stderr_tail"] = stderrTail
	}
	return details
}

// formatErrorMessage builds a human-readable summary for an execution_error
// event. Always produces a non-empty string, even when the CLI sent no
// `result` text — that's the whole point.
func formatErrorMessage(subtype string, numTurns int, durationMs int, resultText string) string {
	summary := "claude-code terminated with is_error=true"
	parts := []string{}
	if subtype != "" {
		parts = append(parts, "subtype="+subtype)
	}
	parts = append(parts, "turns="+strconv.Itoa(numTurns), "duration_ms="+strconv.Itoa(durationMs))
	summary += " (" + strings.Join(parts, ", ") + ")"

	if strings.TrimSpace(resultText) != "" {
		summary += ": " + strings.TrimSpace(resultText)
	}
	return summary
}

// isAutoCompactMarker recognizes the log-style strings Claude Code emits
// around an automatic (non-user-triggered) compaction. Broader than
// isCompactionSummary, which only matches the final summary payload.
func isAutoCompactMarker(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	markers := []string{
		"auto-compacting",
		"auto compacting",
		"conversation history has been compacted",
		"context has been compacted",
		"automatic compaction",
	}
	for _, m := range markers {
		if strings.Contains(c, m) {
			return true
		}
	}
	return false
}
