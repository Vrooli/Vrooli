package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/browser-automation-studio/handlers/export"
)

// Type aliases for backward compatibility with replay_config.go.
// These delegate to handlers/export types.
type (
	executionExportOverrides = export.Overrides
	themePresetOverride      = export.ThemePreset
	cursorPresetOverride     = export.CursorPreset
)

// normalizeExportFilename normalizes a filename for export, adding the extension if missing.
func normalizeExportFilename(filename, defaultBase, ext string) string {
	cleaned := strings.TrimSpace(filename)
	if cleaned == "" {
		cleaned = defaultBase
	}
	if ext == "" {
		return cleaned
	}
	if strings.HasSuffix(strings.ToLower(cleaned), strings.ToLower(ext)) {
		return cleaned
	}
	return cleaned + ext
}

// requestBaseURL extracts the base URL (scheme + host) from an HTTP request.
func requestBaseURL(r *http.Request) string {
	if r == nil {
		return ""
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			scheme = strings.TrimSpace(parts[0])
		}
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = strings.TrimSpace(r.URL.Host)
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

// ExecutionExportability contains lightweight exportability info for an execution.
// This is computed via file existence checks rather than loading full timeline data.
type ExecutionExportability struct {
	HasTimeline      bool `json:"has_timeline"`
	HasScreenshots   bool `json:"has_screenshots"`
	HasRecordedVideo bool `json:"has_recorded_video"`
	IsExportable     bool `json:"is_exportable"`
}

// checkExecutionExportability performs lightweight file existence checks to determine
// if an execution has exportable content. This avoids loading full timeline data.
func checkExecutionExportability(resultPath, recordingsRoot, executionID string) ExecutionExportability {
	result := ExecutionExportability{}

	if strings.TrimSpace(resultPath) == "" {
		return result
	}

	// Check for timeline.proto.json
	resultDir := filepath.Dir(resultPath)
	timelinePath := filepath.Join(resultDir, "timeline.proto.json")
	if info, err := os.Stat(timelinePath); err == nil && !info.IsDir() && info.Size() > 0 {
		result.HasTimeline = true
	}

	// Check for screenshots directory with content
	screenshotsDir := filepath.Join(resultDir, "screenshots")
	if entries, err := os.ReadDir(screenshotsDir); err == nil && len(entries) > 0 {
		for _, entry := range entries {
			if !entry.IsDir() {
				result.HasScreenshots = true
				break
			}
		}
	}

	// Check for recorded video
	if strings.TrimSpace(recordingsRoot) != "" && strings.TrimSpace(executionID) != "" {
		videoDir := filepath.Join(recordingsRoot, executionID, "artifacts", "videos")
		if entries, err := os.ReadDir(videoDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					result.HasRecordedVideo = true
					break
				}
			}
		}
	}

	// An execution is exportable if it has timeline data or recorded video
	result.IsExportable = result.HasTimeline || result.HasRecordedVideo

	return result
}
