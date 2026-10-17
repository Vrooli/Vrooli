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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
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
	ArtifactID  string
	SHA256      string
	SizeBytes   *int64
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
	return ResolveVideoSourceContext(context.Background(), artifacts, store)
}

// ResolveVideoSourceContext bounds local hashing and remote ingestion by job lifetime.
func ResolveVideoSourceContext(ctx context.Context, artifacts []executionwriter.ArtifactData, store storage.StorageInterface) (*VideoSource, error) {
	var lastErr error
	for _, artifact := range artifacts {
		if !isRecordedVideoArtifact(artifact.ArtifactType) {
			continue
		}
		source, err := videoFromArtifactContext(ctx, artifact, store)
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
	return videoFromArtifactContext(context.Background(), artifact, store)
}
func videoFromArtifactContext(ctx context.Context, artifact executionwriter.ArtifactData, store storage.StorageInterface) (*VideoSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	payload := artifact.Payload
	path := payloadString(payload, "path")
	if path == "" {
		path = filePathFromStorageURL(artifact.StorageURL)
	}
	if path != "" {
		info, err := os.Lstat(path)
		if err == nil && !info.Mode().IsRegular() {
			return nil, errors.New("recording must be a regular file without symlinks")
		}
		if err == nil && info.Mode().IsRegular() {
			contentType := firstNonEmptyString(artifact.ContentType, payloadString(payload, "content_type"))
			if contentType == "" {
				contentType = DetectVideoContentType(path)
			}
			resolved := &VideoSource{
				Path:        path,
				ContentType: contentType,
				ArtifactID:  artifact.ArtifactID,
				SHA256:      artifact.SHA256,
				SizeBytes:   artifact.SizeBytes,
			}
			if err := completeSourceReceipt(ctx, resolved); err != nil {
				return nil, err
			}
			return resolved, nil
		}
	}

	if store != nil {
		if objectName := artifactObjectNameFromURL(artifact.StorageURL); objectName != "" {
			source, err := downloadVideoFromStorage(ctx, store, objectName, artifact)
			if err != nil {
				return nil, err
			}
			if source != nil {
				source.ArtifactID = artifact.ArtifactID
				source.SHA256 = artifact.SHA256
				source.SizeBytes = artifact.SizeBytes
				if err := completeSourceReceipt(ctx, source); err != nil {
					if source.Cleanup != nil {
						source.Cleanup()
					}
					return nil, err
				}
				return source, nil
			}
		}
	}

	if payloadBool(payload, "inline") {
		encoded := payloadString(payload, "base64")
		if encoded == "" {
			return nil, nil
		}
		if base64.StdEncoding.DecodedLen(len(encoded)) > 64*1024*1024 {
			return nil, errors.New("inline recording exceeds 64 MiB budget")
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
		resolved := &VideoSource{
			Path:        file.Name(),
			ContentType: contentType,
			ArtifactID:  artifact.ArtifactID,
			SHA256:      artifact.SHA256,
			SizeBytes:   artifact.SizeBytes,
			Cleanup:     func() { _ = os.Remove(file.Name()) },
		}
		if err := completeSourceReceipt(ctx, resolved); err != nil {
			if resolved.Cleanup != nil {
				resolved.Cleanup()
			}
			return nil, err
		}
		return resolved, nil
	}

	return nil, nil
}

const maxRecordingBytes int64 = 512 * 1024 * 1024

func completeSourceReceipt(ctx context.Context, source *VideoSource) error {
	if source == nil || strings.TrimSpace(source.Path) == "" {
		return errors.New("video source path is required to complete its receipt")
	}
	file, err := openRegularRecording(ctx, source.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxRecordingBytes {
		return errors.New("recording must be a nonempty regular file within 512 MiB budget")
	}
	if source.SizeBytes != nil && *source.SizeBytes != info.Size() {
		return errors.New("recorded artifact size differs from source bytes")
	}
	hash := sha256.New()
	written, err := io.Copy(hash, &cancelReader{ctx: ctx, reader: io.LimitReader(file, maxRecordingBytes+1)})
	if err != nil {
		return err
	}
	after, err := file.Stat()
	if err != nil {
		return err
	}
	if written != info.Size() || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return errors.New("recording changed while computing its identity")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if source.SHA256 != "" && !strings.EqualFold(source.SHA256, actual) {
		return errors.New("recorded artifact hash differs from source bytes")
	}
	size := info.Size()
	source.SizeBytes = &size
	source.SHA256 = actual
	return nil
}

type cancelReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *cancelReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
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
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	}
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	return "video/webm"
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

func downloadVideoFromStorage(parent context.Context, store storage.StorageInterface, objectName string, artifact executionwriter.ArtifactData) (*VideoSource, error) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
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
	if written, err := io.Copy(file, &cancelReader{ctx: ctx, reader: io.LimitReader(reader, maxRecordingBytes+1)}); err != nil || written > maxRecordingBytes {
		if err == nil {
			err = fmt.Errorf("recording exceeds 512 MiB budget")
		}
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

// FreezeVideoSource copies verified bytes into a unique execution-owned directory.
// Rendering never rereads a mutable original after its provenance was recorded.
func FreezeVideoSource(ctx context.Context, original *VideoSource, assetsDir string) (*VideoSource, error) {
	if err := os.MkdirAll(assetsDir, 0700); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp(assetsDir, "frozen-source-")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	path := filepath.Join(directory, "source"+filepath.Ext(original.Path))
	input, err := openRegularRecording(ctx, original.Path)
	if err != nil {
		cleanup()
		return nil, err
	}
	defer input.Close()
	output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		cleanup()
		return nil, err
	}
	written, copyErr := io.Copy(output, &cancelReader{ctx: ctx, reader: io.LimitReader(input, maxRecordingBytes+1)})
	err = errors.Join(copyErr, output.Close())
	if err != nil || written > maxRecordingBytes {
		cleanup()
		if err == nil {
			err = errors.New("recording exceeds 512 MiB freeze budget")
		}
		return nil, err
	}
	frozen := *original
	frozen.Path = path
	frozen.Cleanup = cleanup
	if err = completeSourceReceipt(ctx, &frozen); err != nil {
		cleanup()
		return nil, err
	}
	return &frozen, nil
}
