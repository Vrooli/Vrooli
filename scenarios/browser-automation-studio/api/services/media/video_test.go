package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeVideoMeasuresRealFileAndRejectsInvalidMedia(t *testing.T) {
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " unavailable")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "source.webm")
	if out, err := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=black:s=64x48:r=10:d=0.5", "-c:v", "libvpx-vp9", "-threads", "1", path).CombinedOutput(); err != nil {
		t.Fatalf("fixture encode: %v %s", err, out)
	}
	got, err := ProbeVideo(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 64 || got.Height != 48 || got.DurationMs < 400 || got.DurationMs > 600 {
		t.Fatalf("measured media = %+v", got)
	}
	bad := filepath.Join(t.TempDir(), "invalid.webm")
	if err := os.WriteFile(bad, []byte("not media"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ProbeVideo(ctx, bad); err == nil {
		t.Fatal("invalid media acquired measured geometry")
	}
}
