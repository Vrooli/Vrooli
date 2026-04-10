// Package source provides video source resolution for export operations.
//
// This package handles the resolution of recorded video sources from various
// storage backends and artifact types. It supports:
//   - Local file system paths
//   - Object storage (via storage.StorageInterface)
//   - Base64-encoded inline video data
//
// The VideoSource type encapsulates the resolved video with cleanup handling
// for temporary files downloaded from remote storage.
package source

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/storage"
)

// VideoSource represents a resolved video source ready for processing.
type VideoSource struct {
	Path        string
	ContentType string
	Cleanup     func()
}

// ErrVideoNotFound indicates no recorded video is available for the execution.
var ErrVideoNotFound = errors.New("recorded video not available")

// RenderSource constants define the available render source types.
const (
	RenderSourceAuto          = "auto"
	RenderSourceRecordedVideo = "recorded_video"
	RenderSourceReplayFrames  = "replay_frames"
)

// NormalizeRenderSource validates and normalizes a render source string.
// Returns the normalized source and true if valid, or empty string and false if invalid.
func NormalizeRenderSource(value string) (string, bool) {
	renderSource := strings.ToLower(strings.TrimSpace(value))
	if renderSource == "" {
		return RenderSourceAuto, true
	}
	switch renderSource {
	case RenderSourceAuto, RenderSourceRecordedVideo, RenderSourceReplayFrames:
		return renderSource, true
	default:
		return "", false
	}
}

// ResolveVideoSource finds and returns the first valid video source from the provided artifacts.
// It tries local paths, object storage, and inline base64 data in order.
func ResolveVideoSource(artifacts []executionwriter.ArtifactData, store storage.StorageInterface) (*VideoSource, error) {
	var lastErr error
	for _, artifact := range artifacts {
		if !isRecordedVideoArtifact(artifact.ArtifactType) {
			continue
		}
		source, err := videoFromArtifact(artifact, store)
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
	return nil, ErrVideoNotFound
}

func isRecordedVideoArtifact(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "video_meta", "video":
		return true
	default:
		return false
	}
}

func videoFromArtifact(artifact executionwriter.ArtifactData, store storage.StorageInterface) (*VideoSource, error) {
	payload := artifact.Payload
	path := payloadString(payload, "path")
	if path == "" {
		path = filePathFromStorageURL(artifact.StorageURL)
	}
	if path != "" {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			contentType := firstNonEmptyString(artifact.ContentType, payloadString(payload, "content_type"))
			if contentType == "" {
				contentType = DetectVideoContentType(path)
			}
			return &VideoSource{
				Path:        path,
				ContentType: contentType,
			}, nil
		}
	}

	if store != nil {
		if objectName := artifactObjectNameFromURL(artifact.StorageURL); objectName != "" {
			source, err := downloadVideoFromStorage(store, objectName, artifact)
			if err != nil {
				return nil, err
			}
			if source != nil {
				return source, nil
			}
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
			contentType = DetectVideoContentType(path)
		}
		ext := ExtensionForContentType(contentType)
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
		return &VideoSource{
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

// DetectVideoContentType returns the MIME type for a video file based on extension.
func DetectVideoContentType(path string) string {
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

// ExtensionForContentType returns a file extension for a given content type.
func ExtensionForContentType(contentType string) string {
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

func artifactObjectNameFromURL(storageURL string) string {
	trimmed := strings.TrimSpace(storageURL)
	if trimmed == "" {
		return ""
	}
	path := trimmed
	if parsed, err := url.Parse(trimmed); err == nil {
		if parsed.Path != "" {
			path = parsed.Path
		}
	}
	switch {
	case strings.HasPrefix(path, "/api/v1/artifacts/"):
		return strings.TrimPrefix(path, "/api/v1/artifacts/")
	case strings.HasPrefix(path, "/api/v1/screenshots/"):
		return strings.TrimPrefix(path, "/api/v1/screenshots/")
	default:
		return ""
	}
}

func filePathFromStorageURL(storageURL string) string {
	trimmed := strings.TrimSpace(storageURL)
	if trimmed == "" {
		return ""
	}
	return strings.TrimPrefix(trimmed, "file://")
}

func downloadVideoFromStorage(store storage.StorageInterface, objectName string, artifact executionwriter.ArtifactData) (*VideoSource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	reader, info, err := store.GetArtifact(ctx, objectName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	contentType := firstNonEmptyString(artifact.ContentType, info.ContentType, payloadString(artifact.Payload, "content_type"))
	ext := ExtensionForContentType(contentType)
	if ext == "" {
		ext = filepath.Ext(objectName)
	}
	if ext == "" {
		ext = ".webm"
	}

	file, err := os.CreateTemp("", "bas-recorded-video-*"+ext)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(file.Name())
		return nil, err
	}

	return &VideoSource{
		Path:        file.Name(),
		ContentType: contentType,
		Cleanup:     func() { _ = os.Remove(file.Name()) },
	}, nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
