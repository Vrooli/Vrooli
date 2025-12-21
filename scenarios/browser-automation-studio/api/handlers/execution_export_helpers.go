package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/handlers/export"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
)

// executionExportRequest represents the JSON payload for execution export endpoints.
// It wraps the export.Request type for backwards compatibility with existing handler code.
type executionExportRequest = export.Request

// executionExportOverrides allows clients to customize export themes and cursor configuration.
// It wraps the export.Overrides type for backwards compatibility with existing handler code.
type executionExportOverrides = export.Overrides

// themePresetOverride specifies which chrome and background preset themes to apply.
// It wraps the export.ThemePreset type for backwards compatibility with existing handler code.
type themePresetOverride = export.ThemePreset

// cursorPresetOverride specifies which cursor preset theme to apply and additional options.
// It wraps the export.CursorPreset type for backwards compatibility with existing handler code.
type cursorPresetOverride = export.CursorPreset

var errMovieSpecUnavailable = export.ErrMovieSpecUnavailable

// applyExportOverrides applies client-provided overrides to a movie spec.
func applyExportOverrides(spec *exportservices.ReplayMovieSpec, overrides *executionExportOverrides) {
	export.Apply(spec, overrides)
}

// buildExportSpec constructs a validated ReplayMovieSpec for export by merging client-provided
// and server-generated specs, validating execution ID matching, and filling in defaults.
func buildExportSpec(baseline, incoming *exportservices.ReplayMovieSpec, executionID uuid.UUID) (*exportservices.ReplayMovieSpec, error) {
	return export.BuildSpec(baseline, incoming, executionID)
}

// cloneMovieSpec creates a deep copy of a ReplayMovieSpec.
func cloneMovieSpec(spec *exportservices.ReplayMovieSpec) (*exportservices.ReplayMovieSpec, error) {
	return export.Clone(spec)
}

type recordedVideoSource struct {
	Path        string
	ContentType string
	Cleanup     func()
}

var errRecordedVideoNotFound = errors.New("recorded video not available")

const (
	renderSourceAuto          = "auto"
	renderSourceRecordedVideo = "recorded_video"
	renderSourceReplayFrames  = "replay_frames"
)

func normalizeRenderSource(value string) (string, bool) {
	renderSource := strings.ToLower(strings.TrimSpace(value))
	if renderSource == "" {
		return renderSourceAuto, true
	}
	switch renderSource {
	case renderSourceAuto, renderSourceRecordedVideo, renderSourceReplayFrames:
		return renderSource, true
	default:
		return "", false
	}
}

func resolveRecordedVideoSource(artifacts []executionwriter.ArtifactData) (*recordedVideoSource, error) {
	var lastErr error
	for _, artifact := range artifacts {
		if !isRecordedVideoArtifact(artifact.ArtifactType) {
			continue
		}
		source, err := recordedVideoFromArtifact(artifact)
		if err != nil {
			lastErr = err
			continue
		}
		if source != nil {
			return source, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errRecordedVideoNotFound
}

func isRecordedVideoArtifact(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "video_meta", "video":
		return true
	default:
		return false
	}
}

func recordedVideoFromArtifact(artifact executionwriter.ArtifactData) (*recordedVideoSource, error) {
	payload := artifact.Payload
	path := payloadString(payload, "path")
	if path == "" {
		if storagePath := strings.TrimPrefix(strings.TrimSpace(artifact.StorageURL), "file://"); storagePath != "" {
			path = storagePath
		}
	}
	if path != "" {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			contentType := firstNonEmptyString(artifact.ContentType, payloadString(payload, "content_type"))
			if contentType == "" {
				contentType = detectVideoContentType(path)
			}
			return &recordedVideoSource{
				Path:        path,
				ContentType: contentType,
			}, nil
		}
	}

	if payloadBool(payload, "inline") {
		encoded := payloadString(payload, "base64")
		if encoded == "" {
			return nil, nil
		}
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
		contentType := firstNonEmptyString(artifact.ContentType, payloadString(payload, "content_type"))
		if contentType == "" {
			contentType = detectVideoContentType(path)
		}
		ext := extensionForContentType(contentType)
		if ext == "" {
			ext = ".webm"
		}
		file, err := os.CreateTemp("", "bas-recorded-video-*"+ext)
		if err != nil {
			return nil, err
		}
		if _, err := file.Write(raw); err != nil {
			_ = file.Close()
			_ = os.Remove(file.Name())
			return nil, err
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(file.Name())
			return nil, err
		}
		return &recordedVideoSource{
			Path:        file.Name(),
			ContentType: contentType,
			Cleanup:     func() { _ = os.Remove(file.Name()) },
		}, nil
	}

	return nil, nil
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

func payloadBool(payload map[string]any, key string) bool {
	if payload == nil {
		return false
	}
	if value, ok := payload[key]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func detectVideoContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "video/webm"
	}
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return "video/webm"
	}
}

func extensionForContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(contentType, "mp4"):
		return ".mp4"
	case strings.Contains(contentType, "webm"):
		return ".webm"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	default:
		return ""
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

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
