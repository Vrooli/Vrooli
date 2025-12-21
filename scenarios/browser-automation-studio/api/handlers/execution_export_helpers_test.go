package handlers

import (
	"encoding/base64"
	"errors"
	"os"
	"testing"

	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
)

func TestNormalizeRenderSource(t *testing.T) {
	if source, ok := normalizeRenderSource(""); !ok || source != renderSourceAuto {
		t.Fatalf("expected auto render source, got %q (ok=%v)", source, ok)
	}
	if source, ok := normalizeRenderSource("recorded_video"); !ok || source != renderSourceRecordedVideo {
		t.Fatalf("expected recorded_video render source, got %q (ok=%v)", source, ok)
	}
	if source, ok := normalizeRenderSource("replay_frames"); !ok || source != renderSourceReplayFrames {
		t.Fatalf("expected replay_frames render source, got %q (ok=%v)", source, ok)
	}
	if _, ok := normalizeRenderSource("nope"); ok {
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

	source, err := resolveRecordedVideoSource([]executionwriter.ArtifactData{artifact})
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

	source, err := resolveRecordedVideoSource([]executionwriter.ArtifactData{artifact})
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

func TestResolveRecordedVideoSource_Missing(t *testing.T) {
	artifact := executionwriter.ArtifactData{
		ArtifactType: "video_meta",
		Payload:      map[string]any{"path": "/nope/video.webm"},
	}
	_, err := resolveRecordedVideoSource([]executionwriter.ArtifactData{artifact})
	if err == nil {
		t.Fatalf("expected error for missing video")
	}
	if !errors.Is(err, errRecordedVideoNotFound) {
		t.Fatalf("expected errRecordedVideoNotFound, got %v", err)
	}
}
