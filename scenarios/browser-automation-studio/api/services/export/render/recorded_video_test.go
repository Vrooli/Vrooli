package render

import (
	"context"
	"encoding/json"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/proto"
)

func TestRecordedVideoKeepsPresentationAndFullSourceTimeline(t *testing.T) {
	original := &ReplayMovieSpec{Decor: &exportsv1.ReplayDecor{ChromeTheme: "minimal", DeviceFrameTheme: "slim", ShowDesktop: proto.Bool(true), ShowBrowserFrame: proto.Bool(false), ShowDeviceFrame: proto.Bool(true), CursorTheme: "arrowNeon", BrowserScale: proto.Float64(.7)}, Presentation: &exportsv1.ReplayPresentation{Canvas: &exportsv1.ReplayDimensions{Width: 1080, Height: 1920}}, Frames: []*exportsv1.ReplayFrame{{DurationMs: 1000}}}
	spec, err := recordedVideoSpec(original, "http://localhost:1234/api/v1/recordings/assets/id/artifacts/videos/page.webm", 8446, 430, 880)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Decor.GetDeviceFrameTheme() != "slim" || spec.Decor.GetShowBrowserFrame() || !spec.Decor.GetShowDeviceFrame() || spec.Decor.GetBrowserScale() != .7 {
		t.Fatalf("lost styling: %v", spec.Decor)
	}
	if spec.Summary.TotalDurationMs != 8446 || spec.Frames[0].DurationMs != 8446 || spec.RecordedVideo.DurationMs != 8446 {
		t.Fatal("continuous timeline shortened to replay duration")
	}
	if spec.Presentation.Viewport.Width != 430 || spec.Presentation.Canvas.Width != 1080 {
		t.Fatal("raster/canvas confused")
	}
	if spec.Decor.CursorTheme != "disabled" || original.Decor.CursorTheme != "arrowNeon" {
		t.Fatal("must preserve original and avoid invented cursor overlay")
	}
}
func TestRecordedVideoDoesNotSilentlyIgnoreCards(t *testing.T) {
	_, err := recordedVideoSpec(&ReplayMovieSpec{IntroCard: &exportsv1.ReplayIntroCard{Enabled: true}}, "http://localhost/source", 1000, 400, 300)
	if err == nil {
		t.Fatal("unsupported cards silently ignored")
	}
}

func TestRecordedMasterPreservesAudioAndRemovesExtraCaptureHold(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("FFmpeg required")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	rendered := filepath.Join(dir, "rendered.mp4")
	for _, item := range []struct {
		path     string
		duration string
		audio    bool
	}{{source, "1", true}, {rendered, "1.04", false}} {
		args := []string{"-v", "error", "-f", "lavfi", "-i", "color=red:s=160x120:r=25:d=" + item.duration}
		if item.audio {
			args = append(args, "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-c:a", "aac")
		}
		args = append(args, "-c:v", "libx264", "-threads", "2", "-pix_fmt", "yuv420p", item.path)
		if raw, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
			t.Fatalf("fixture %v: %s", err, raw)
		}
	}
	if err := FinishRecordedVideo(context.Background(), &RenderedMedia{Path: rendered}, source, 1000, "mp4"); err != nil {
		t.Fatal(err)
	}
	raw, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "stream=codec_type,duration", "-of", "json", rendered).Output()
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		Streams []struct {
			Type     string `json:"codec_type"`
			Duration string `json:"duration"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	if len(meta.Streams) != 2 {
		t.Fatalf("lost audio: %s", raw)
	}
	for _, s := range meta.Streams {
		duration, err := strconv.ParseFloat(s.Duration, 64)
		if err != nil || math.Abs(duration-1) > .025 {
			t.Fatalf("wrong final %s duration: %s", s.Type, raw)
		}
	}
}
