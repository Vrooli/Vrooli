package services

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtractSessionIDFromPaths attempts to extract agent session ID from various file paths and patterns.
// Codex creates marker files like: .codex-run-{SESSION_ID}.marker
// We also check transcript paths for embedded session identifiers.
func ExtractSessionIDFromPaths(transcriptPath, scenarioRoot string) string {
	if transcriptPath == "" {
		return ""
	}

	// Extract from transcript filename
	// Format: app-issue-tracker-{issueID}-{timestamp}-conversation.jsonl
	// or: {tag}-{sessionID}-conversation.jsonl
	basename := filepath.Base(transcriptPath)
	if sessionID := extractFromFilename(basename); sessionID != "" {
		return sessionID
	}

	// Check for Codex marker files in scenario root
	if scenarioRoot != "" {
		if sessionID := extractFromCodexMarkers(scenarioRoot, transcriptPath); sessionID != "" {
			return sessionID
		}
	}

	return ""
}

// extractFromFilename tries to extract a session ID from a transcript filename
// Looks for numeric patterns that could be session IDs
func extractFromFilename(filename string) string {
	// Remove extension
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Pattern: *-{timestamp/sessionID}-conversation
	// Extract the numeric part before "-conversation" or "-last"
	parts := strings.Split(nameWithoutExt, "-")
	if len(parts) >= 2 {
		// Look for the numeric timestamp/session ID (last numeric part before suffix)
		for i := len(parts) - 1; i >= 0; i-- {
			part := parts[i]
			// Check if it's a large numeric value (likely timestamp or session ID)
			if matched, _ := regexp.MatchString(`^\d{6,}$`, part); matched {
				return part
			}
		}
	}

	return ""
}

// extractFromCodexMarkers scans for .codex-run-*.marker files that match the transcript
// Looks for the most recently created marker file within a reasonable time window
func extractFromCodexMarkers(scenarioRoot, transcriptPath string) string {
	// Extract timestamp from transcript path to correlate with marker files
	transcriptTimestamp := extractTimestampFromFilename(filepath.Base(transcriptPath))
	if transcriptTimestamp == 0 {
		return ""
	}

	// Look for marker files in scenario root
	markerPattern := filepath.Join(scenarioRoot, ".codex-run-*.marker")
	matches, err := filepath.Glob(markerPattern)
	if err != nil || len(matches) == 0 {
		return ""
	}

	// Pattern: .codex-run-{SESSION_ID}.marker
	sessionIDPattern := regexp.MustCompile(`\.codex-run-(\d+)\.marker$`)

	// Find marker file closest to transcript timestamp (within 5 minutes)
	var bestMatch string
	var bestTimeDiff int64 = 5 * 60 // 5 minutes in seconds

	for _, markerPath := range matches {
		// Extract session ID from marker filename
		markerFilename := filepath.Base(markerPath)
		submatch := sessionIDPattern.FindStringSubmatch(markerFilename)
		if len(submatch) < 2 {
			continue
		}

		sessionID := submatch[1]

		// Get file modification time as proxy for creation time
		info, statErr := os.Stat(markerPath)
		if statErr != nil {
			continue
		}

		// Calculate time difference between marker and transcript
		markerTimestamp := info.ModTime().Unix()
		timeDiff := abs(transcriptTimestamp - markerTimestamp)

		// If this marker is closer in time to the transcript, it's a better match
		if timeDiff < bestTimeDiff {
			bestTimeDiff = timeDiff
			bestMatch = sessionID
		}
	}

	return bestMatch
}

// extractTimestampFromFilename extracts Unix timestamp from filename
func extractTimestampFromFilename(filename string) int64 {
	// Extract the numeric timestamp portion
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(nameWithoutExt, "-")

	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// Look for nanosecond timestamp (very large number)
		if matched, _ := regexp.MatchString(`^\d{16,19}$`, part); matched {
			// Convert nanoseconds to seconds
			if len(part) >= 18 {
				// Nanosecond timestamp - take first 10 digits (seconds)
				return parseTimestamp(part[:10])
			}
		}
		// Look for second timestamp
		if matched, _ := regexp.MatchString(`^\d{10}$`, part); matched {
			return parseTimestamp(part)
		}
	}

	return 0
}

func parseTimestamp(s string) int64 {
	// Simple integer parsing without error handling (we know it's numeric)
	var result int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int64(c-'0')
		}
	}
	return result
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// ExtractSessionIDFromAgentTag extracts a session identifier from an agent tag
// Tag format: app-issue-tracker-{issueID}
// This isn't a true session ID, but can be used as a fallback identifier
func ExtractSessionIDFromAgentTag(agentTag string) string {
	// Extract the issue ID portion which can serve as a session identifier
	prefix := "app-issue-tracker-"
	if strings.HasPrefix(agentTag, prefix) {
		return strings.TrimPrefix(agentTag, prefix)
	}
	return agentTag
}
