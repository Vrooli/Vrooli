package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/storage"
)

func TestVideoSourceCompletesMissingHashAndSizeReceipt(t *testing.T) {
	content := []byte("neutral recorded video bytes")
	path := filepath.Join(t.TempDir(), "recording.webm")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := videoFromArtifact(executionwriter.ArtifactData{
		ArtifactID:   "video-1",
		ArtifactType: "video",
		Payload:      map[string]any{"path": path},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved == nil {
		t.Fatal("expected a resolved video")
	}
	wantHash := sha256.Sum256(content)
	if resolved.SHA256 != fmt.Sprintf("%x", wantHash) {
		t.Fatalf("source hash = %q, want %x", resolved.SHA256, wantHash)
	}
	if resolved.SizeBytes == nil || *resolved.SizeBytes != int64(len(content)) {
		t.Fatalf("source size = %v, want %d", resolved.SizeBytes, len(content))
	}
	if resolved.ArtifactID != "video-1" {
		t.Fatalf("source artifact id = %q", resolved.ArtifactID)
	}
}

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
	wantHash := sha256.Sum256([]byte("fake-video-storage"))
	if source.SHA256 != fmt.Sprintf("%x", wantHash) || source.SizeBytes == nil || *source.SizeBytes != int64(len("fake-video-storage")) {
		t.Fatalf("download receipt = %s/%v, want %x/%d", source.SHA256, source.SizeBytes, wantHash, len("fake-video-storage"))
	}
	source.Cleanup()
}

func TestResolveVideoSource_StorageReceiptMismatchCleansTempFile(t *testing.T) {
	store := storage.NewMemoryStorage()
	data := []byte("verified bytes")
	info, err := store.StoreArtifact(context.Background(), "recordings/video.webm", data, "video/webm")
	if err != nil {
		t.Fatal(err)
	}
	actualHash := fmt.Sprintf("%x", sha256.Sum256(data))
	actualSize := int64(len(data))
	badSize := int64(1)
	for _, tc := range []struct {
		name string
		hash string
		size *int64
	}{
		{name: "size", hash: actualHash, size: &badSize},
		{name: "hash", hash: "stale", size: &actualSize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, err := filepath.Glob(filepath.Join(os.TempDir(), "bas-recorded-video-*"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = ResolveVideoSourceContext(t.Context(), []executionwriter.ArtifactData{{
				ArtifactType: "video",
				StorageURL:   info.URL,
				ContentType:  "video/webm",
				SHA256:       tc.hash,
				SizeBytes:    tc.size,
			}}, store)
			if err == nil {
				t.Fatal("accepted storage video with mismatched declared receipt")
			}
			after, err := filepath.Glob(filepath.Join(os.TempDir(), "bas-recorded-video-*"))
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range after {
				found := false
				for _, prior := range before {
					if path == prior {
						found = true
						break
					}
				}
				if !found {
					_ = os.Remove(path)
					t.Fatalf("failed storage receipt left a temp file: %s", path)
				}
			}
		})
	}
}

type byteCountReader struct {
	reader *bytes.Reader
	bytes  int64
}

func (r *byteCountReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytes += int64(n)
	return n, err
}

func TestCopyVideoSourceProducesReceiptInOneBoundedPass(t *testing.T) {
	const limit = 16
	data := []byte("0123456789abcdef")
	reader := &byteCountReader{reader: bytes.NewReader(data)}
	file, err := os.CreateTemp(t.TempDir(), "video-copy-*.webm")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	receipt, err := copyVideoSourceLimit(t.Context(), reader, file, limit)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := sha256.Sum256(data)
	if reader.bytes != int64(len(data)) || receipt.SizeBytes != int64(len(data)) || receipt.SHA256 != fmt.Sprintf("%x", wantHash) {
		t.Fatalf("read=%d receipt=%d/%s, want %d/%x", reader.bytes, receipt.SizeBytes, receipt.SHA256, len(data), wantHash)
	}
	written, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(written, data) {
		t.Fatalf("copied bytes = %q, err=%v", written, err)
	}
}

func TestCopyVideoSourceLimitUsesOnlyOneSentinelByte(t *testing.T) {
	reader := &byteCountReader{reader: bytes.NewReader([]byte("0123456789"))}
	file, err := os.CreateTemp(t.TempDir(), "video-copy-limit-*.webm")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	_, err = copyVideoSourceLimit(t.Context(), reader, file, 4)
	if !errors.Is(err, errVideoSourceTooLarge) {
		t.Fatalf("copy error = %v, want size limit error", err)
	}
	if reader.bytes != 5 {
		t.Fatalf("read %d bytes before rejecting, want limit plus one sentinel (5)", reader.bytes)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("caller-owned path should remain available for cleanup: %v", err)
	}
}

func TestCopyVideoSourceCancellationClosesDestination(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	file, err := os.CreateTemp(t.TempDir(), "video-copy-cancel-*.webm")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	_, err = copyVideoSourceLimit(ctx, bytes.NewReader([]byte("payload")), file, 32)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("copy error = %v, want context cancellation", err)
	}
	if _, err := file.Write([]byte("after close")); err == nil {
		t.Fatal("destination remained open after cancellation")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("caller must retain cleanup ownership: %v", err)
	}
}

func BenchmarkVideoSourceReceiptPasses(b *testing.B) {
	fixture := bytes.Repeat([]byte("v"), 256*1024)
	b.Run("copy-then-readback", func(b *testing.B) {
		dir := b.TempDir()
		b.ReportAllocs()
		b.SetBytes(int64(len(fixture)))
		for i := 0; i < b.N; i++ {
			file, err := os.CreateTemp(dir, "video-baseline-*.webm")
			if err != nil {
				b.Fatal(err)
			}
			path := file.Name()
			if _, err := io.Copy(file, bytes.NewReader(fixture)); err != nil {
				b.Fatal(err)
			}
			if err := file.Close(); err != nil {
				b.Fatal(err)
			}
			if err := completeSourceReceipt(b.Context(), &VideoSource{Path: path}); err != nil {
				b.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("copy-and-receipt", func(b *testing.B) {
		dir := b.TempDir()
		b.ReportAllocs()
		b.SetBytes(int64(len(fixture)))
		for i := 0; i < b.N; i++ {
			file, err := os.CreateTemp(dir, "video-single-pass-*.webm")
			if err != nil {
				b.Fatal(err)
			}
			path := file.Name()
			if _, err := copyVideoSource(b.Context(), bytes.NewReader(fixture), file); err != nil {
				b.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestDetectVideoContentType(t *testing.T) {
	// OS MIME databases can register WebM as audio. Known video outputs must
	// retain their declared media kind regardless of the host's registration.
	previous := mime.TypeByExtension(".webm")
	t.Cleanup(func() { _ = mime.AddExtensionType(".webm", previous) })
	if err := mime.AddExtensionType(".webm", "audio/webm"); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path     string
		expected string
	}{
		{"video.mp4", "video/mp4"},
		{"video.webm", "video/webm"},
		{"VIDEO.WEBM", "video/webm"},
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

func TestRecordingSourceRejectsStaleIdentityAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.webm")
	if err := os.WriteFile(path, []byte("actual bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	artifacts := []executionwriter.ArtifactData{{ArtifactType: "video", SHA256: "stale", Payload: map[string]any{"path": path}}}
	if _, err := ResolveVideoSourceContext(t.Context(), artifacts, nil); err == nil {
		t.Fatal("trusted stale declared hash")
	}
	artifacts[0].SHA256 = ""
	size := int64(1)
	artifacts[0].SizeBytes = &size
	if _, err := ResolveVideoSourceContext(t.Context(), artifacts, nil); err == nil {
		t.Fatal("trusted stale declared size")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := ResolveVideoSourceContext(ctx, artifacts, nil); !errors.Is(err, context.Canceled) {
		t.Fatal("lost cancellation", err)
	}
}

func TestFreezeRecordingRendersStableVerifiedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "original.webm")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	original, err := ResolveVideoSourceContext(t.Context(), []executionwriter.ArtifactData{{ArtifactType: "video", Payload: map[string]any{"path": path}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assets := t.TempDir()
	frozen, err := FreezeVideoSource(t.Context(), original, assets)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(frozen.Path)
	if err != nil || string(bytes) != "original" || frozen.SHA256 != original.SHA256 {
		t.Fatal("render input not stable", err)
	}
	frozen.Cleanup()
	entries, err := os.ReadDir(assets)
	if err != nil || len(entries) != 0 {
		t.Fatal("owned frozen source leaked", err)
	}
	if _, err = FreezeVideoSource(t.Context(), original, assets); err == nil {
		t.Fatal("froze bytes different from recorded identity")
	}
}
