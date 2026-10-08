package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"mime"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	hexport "github.com/vrooli/browser-automation-studio/handlers/export"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/export/source"
	"github.com/vrooli/browser-automation-studio/storage"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
)

func TestLoadRecordedVideoUsesVideoMediaType(t *testing.T) {
	previous := mime.TypeByExtension(".webm")
	t.Cleanup(func() { _ = mime.AddExtensionType(".webm", previous) })
	if err := mime.AddExtensionType(".webm", "audio/webm"); err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	root := t.TempDir()
	path := filepath.Join(root, id.String(), "artifacts", "videos", "recording.webm")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("synthetic video"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := NewMockRepository()
	repo.AddExecution(&database.ExecutionIndex{ID: id, ResultPath: "result.json"})
	h := &Handler{repo: repo, recordingsRoot: root}
	video, err := h.loadRecordedVideo(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if video.ContentType != "video/webm" {
		t.Fatalf("video content type = %q", video.ContentType)
	}
}

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

func TestBuildExportReceiptSettingsRecordsSourceStyleAndEditMapAvailability(t *testing.T) {
	receipt := buildExportReceiptSettings(hexport.Request{
		Overrides: &hexport.Overrides{CursorPreset: &exportservices.CursorPreset{Theme: "arrow-light"}},
	}, source.RenderSourceAuto)
	if receipt["requested_source"] != source.RenderSourceAuto || receipt["selected_source"] != "pending" {
		t.Fatalf("source receipt = %#v", receipt)
	}
	if receipt["cursor_style"] != "arrow-light" {
		t.Fatalf("cursor style receipt = %#v", receipt["cursor_style"])
	}
	if receipt["edit_map_status"] != "not_represented_by_export_contract" {
		t.Fatalf("edit map receipt = %#v", receipt["edit_map_status"])
	}
}

func TestBuildRenderedSpecReceiptIdentifiesFinalReplaySpec(t *testing.T) {
	spec := &exportsv1.ReplaySpec{
		Frames: []*exportsv1.ReplayFrame{{Index: 0, StepIndex: 3}, {Index: 1, StepIndex: 5}},
		Cursor: &exportsv1.ReplayCursor{Style: "arrow-light"},
	}
	receipt := buildRenderedSpecReceipt(spec)
	if receipt["rendered_frame_count"] != 2 {
		t.Fatalf("rendered frame count = %#v", receipt["rendered_frame_count"])
	}
	hash, ok := receipt["rendered_spec_sha256"].(string)
	if !ok || len(hash) != 64 {
		t.Fatalf("rendered spec hash = %#v", receipt["rendered_spec_sha256"])
	}
	spec.Cursor.Style = "hidden"
	if buildRenderedSpecReceipt(spec)["rendered_spec_sha256"] == hash {
		t.Fatal("rendered spec hash did not change when the final cursor style changed")
	}
}

func TestSetExportReceiptHeadersExposeSelectedReplaySourceAndStyle(t *testing.T) {
	response := httptest.NewRecorder()
	setExportReceiptHeaders(response, &exportsv1.ReplaySpec{Cursor: &exportsv1.ReplayCursor{Style: "arrow-light"}}, source.RenderSourceReplayFrames)
	if got := response.Header().Get("X-BAS-Export-Selected-Source"); got != source.RenderSourceReplayFrames {
		t.Fatalf("selected source header = %q", got)
	}
	if got := response.Header().Get("X-BAS-Export-Cursor-Style"); got != "arrow-light" {
		t.Fatalf("cursor style header = %q", got)
	}
	if got := response.Header().Get("X-BAS-Export-Edit-Map-Status"); got != "not_represented_by_export_contract" {
		t.Fatalf("edit map status header = %q", got)
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
		ArtifactID:   "video-artifact-1",
		SHA256:       "recorded-video-sha256",
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
	if videoSource.ArtifactID != "video-artifact-1" || videoSource.SHA256 != "recorded-video-sha256" {
		t.Fatalf("video source identity was not preserved: %#v", videoSource)
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
