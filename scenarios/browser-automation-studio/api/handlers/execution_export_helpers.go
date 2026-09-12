package handlers

import (
	"fmt"
	"net/http"
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
