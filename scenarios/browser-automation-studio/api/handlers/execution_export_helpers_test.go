package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/services/export/source"
	"github.com/vrooli/browser-automation-studio/storage"
)

func TestNormalizeRenderSource(t *testing.T) {
	if result, ok := source.NormalizeRenderSource(""); !ok || result != source.RenderSourceAuto {
		t.Fatalf("expected auto render source, got %q (ok=%v)", result, ok)
	}
	if result, ok := source.NormalizeRenderSource("recorded_video"); !ok || result != source.RenderSourceRecordedVideo {
		t.Fatalf("expected recorded_video render source, got %q (ok=%v)", result, ok)
	}
	if result, ok := source.NormalizeRenderSource("replay_frames"); !ok || result != source.RenderSourceReplayFrames {
		t.Fatalf("expected replay_frames render source, got %q (ok=%v)", result, ok)
	}
	if _, ok := source.NormalizeRenderSource("nope"); ok {
		t.Fatalf("expected invalid render source to fail")
	}
}

func TestResolveRecordedVideoSource_Path(t *testing.T) {
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

	videoSource, err := source.ResolveVideoSource([]executionwriter.ArtifactData{artifact}, nil)
	if err != nil {
		t.Fatalf("expected video source, got error: %v", err)
	}
	if videoSource == nil || videoSource.Path != tmp.Name() {
		t.Fatalf("expected video source path %q, got %#v", tmp.Name(), videoSource)
	}
	if videoSource.ContentType != "video/webm" {
		t.Fatalf("expected content type video/webm, got %q", videoSource.ContentType)
	}
}

func TestResolveRecordedVideoSource_Inline(t *testing.T) {
	payload := map[string]any{
		"inline":       true,
		"base64":       base64.StdEncoding.EncodeToString([]byte("fake-video-inline")),
		"content_type": "video/webm",
	}
	artifact := executionwriter.ArtifactData{
		ArtifactType: "video_meta",
		Payload:      payload,
	}

	videoSource, err := source.ResolveVideoSource([]executionwriter.ArtifactData{artifact}, nil)
	if err != nil {
		t.Fatalf("expected video source, got error: %v", err)
	}
	if videoSource == nil {
		t.Fatalf("expected non-nil source")
	}
	if _, statErr := os.Stat(videoSource.Path); statErr != nil {
		t.Fatalf("expected inline file to exist, got error: %v", statErr)
	}
	if videoSource.Cleanup == nil {
		t.Fatalf("expected cleanup for inline video source")
	}
	videoSource.Cleanup()
	if _, statErr := os.Stat(videoSource.Path); !os.IsNotExist(statErr) {
		t.Fatalf("expected inline file to be removed, got error: %v", statErr)
	}
}

func TestResolveRecordedVideoSource_Missing(t *testing.T) {
	artifact := executionwriter.ArtifactData{
		ArtifactType: "video_meta",
		Payload:      map[string]any{"path": "/nope/video.webm"},
	}
	_, err := source.ResolveVideoSource([]executionwriter.ArtifactData{artifact}, nil)
	if err == nil {
		t.Fatalf("expected error for missing video")
	}
	if !errors.Is(err, source.ErrVideoNotFound) {
		t.Fatalf("expected ErrVideoNotFound, got %v", err)
	}
}

func TestResolveRecordedVideoSource_StorageURL(t *testing.T) {
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

	videoSource, err := source.ResolveVideoSource([]executionwriter.ArtifactData{artifact}, store)
	if err != nil {
		t.Fatalf("expected video source, got error: %v", err)
	}
	if videoSource == nil {
		t.Fatalf("expected non-nil source")
	}
	if _, statErr := os.Stat(videoSource.Path); statErr != nil {
		t.Fatalf("expected downloaded file to exist, got error: %v", statErr)
	}
	if videoSource.Cleanup == nil {
		t.Fatalf("expected cleanup for downloaded video source")
	}
	videoSource.Cleanup()
}
