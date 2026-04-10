package source

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/storage"
)

func TestNormalizeRenderSource(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"", RenderSourceAuto, true},
		{"auto", RenderSourceAuto, true},
		{"recorded_video", RenderSourceRecordedVideo, true},
		{"replay_frames", RenderSourceReplayFrames, true},
		{"RECORDED_VIDEO", RenderSourceRecordedVideo, true},
		{"  replay_frames  ", RenderSourceReplayFrames, true},
		{"invalid", "", false},
		{"nope", "", false},
	}

	for _, tc := range tests {
		source, ok := NormalizeRenderSource(tc.input)
		if ok != tc.ok {
			t.Errorf("NormalizeRenderSource(%q): expected ok=%v, got ok=%v", tc.input, tc.ok, ok)
		}
		if source != tc.expected {
			t.Errorf("NormalizeRenderSource(%q): expected %q, got %q", tc.input, tc.expected, source)
		}
	}
}

func TestResolveVideoSource_Path(t *testing.T) {
	tmp, err := os.CreateTemp("", "bas-video-*.webm")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmp.Write([]byte("fake-video")); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	artifact := executionwriter.ArtifactData{
		ArtifactType: "video_meta",
		ContentType:  "video/webm",
		Payload: map[string]any{
			"path": tmp.Name(),
		},
	}

	source, err := ResolveVideoSource([]executionwriter.ArtifactData{artifact}, nil)
	if err != nil {
		t.Fatalf("expected video source, got error: %v", err)
	}
	if source == nil || source.Path != tmp.Name() {
		t.Fatalf("expected video source path %q, got %#v", tmp.Name(), source)
	}
	if source.ContentType != "video/webm" {
		t.Fatalf("expected content type video/webm, got %q", source.ContentType)
	}
}

func TestResolveVideoSource_Inline(t *testing.T) {
	payload := map[string]any{
		"inline":       true,
		"base64":       base64.StdEncoding.EncodeToString([]byte("fake-video-inline")),
		"content_type": "video/webm",
	}
	artifact := executionwriter.ArtifactData{
		ArtifactType: "video_meta",
		Payload:      payload,
	}

	source, err := ResolveVideoSource([]executionwriter.ArtifactData{artifact}, nil)
	if err != nil {
		t.Fatalf("expected video source, got error: %v", err)
	}
	if source == nil {
		t.Fatalf("expected non-nil source")
	}
	if _, statErr := os.Stat(source.Path); statErr != nil {
		t.Fatalf("expected inline file to exist, got error: %v", statErr)
	}
	if source.Cleanup == nil {
		t.Fatalf("expected cleanup for inline video source")
	}
	source.Cleanup()
	if _, statErr := os.Stat(source.Path); !os.IsNotExist(statErr) {
		t.Fatalf("expected inline file to be removed, got error: %v", statErr)
	}
}

func TestResolveVideoSource_Missing(t *testing.T) {
	artifact := executionwriter.ArtifactData{
		ArtifactType: "video_meta",
		Payload:      map[string]any{"path": "/nope/video.webm"},
	}
	_, err := ResolveVideoSource([]executionwriter.ArtifactData{artifact}, nil)
	if err == nil {
		t.Fatalf("expected error for missing video")
	}
	if !errors.Is(err, ErrVideoNotFound) {
		t.Fatalf("expected ErrVideoNotFound, got %v", err)
	}
}

func TestResolveVideoSource_StorageURL(t *testing.T) {
	tmp, err := os.CreateTemp("", "bas-video-*.webm")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmp.Write([]byte("fake-video-storage")); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	store := storage.NewMemoryStorage()
	info, err := store.StoreArtifactFromFile(context.Background(), uuid.New(), "video-1", tmp.Name(), "video/webm")
	if err != nil {
		t.Fatalf("failed to store artifact: %v", err)
	}

	artifact := executionwriter.ArtifactData{
		ArtifactType: "video_meta",
		StorageURL:   info.URL,
		ContentType:  "video/webm",
		Payload:      map[string]any{},
	}

	source, err := ResolveVideoSource([]executionwriter.ArtifactData{artifact}, store)
	if err != nil {
		t.Fatalf("expected video source, got error: %v", err)
	}
	if source == nil {
		t.Fatalf("expected non-nil source")
	}
	if _, statErr := os.Stat(source.Path); statErr != nil {
		t.Fatalf("expected downloaded file to exist, got error: %v", statErr)
	}
	if source.Cleanup == nil {
		t.Fatalf("expected cleanup for downloaded video source")
	}
	source.Cleanup()
}

func TestDetectVideoContentType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"video.mp4", "video/mp4"},
		{"video.webm", "video/webm"},
		{"video", "video/webm"},
		{"", "video/webm"},
	}

	for _, tc := range tests {
		result := DetectVideoContentType(tc.path)
		if result != tc.expected {
			t.Errorf("DetectVideoContentType(%q): expected %q, got %q", tc.path, tc.expected, result)
		}
	}
}

func TestExtensionForContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"video/mp4", ".mp4"},
		{"video/webm", ".webm"},
		{"image/gif", ".gif"},
		{"application/octet-stream", ""},
	}

	for _, tc := range tests {
		result := ExtensionForContentType(tc.contentType)
		if result != tc.expected {
			t.Errorf("ExtensionForContentType(%q): expected %q, got %q", tc.contentType, tc.expected, result)
		}
	}
}
