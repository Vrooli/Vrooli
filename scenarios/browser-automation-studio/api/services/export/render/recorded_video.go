package render

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/proto"
)

// PrepareRecordedVideoSpec uses the same UI presentation renderer as replay
// frames. It measures the continuous source rather than using editorial replay
// durations. It cannot restyle cursor pixels baked into an existing recording.
func PrepareRecordedVideoSpec(ctx context.Context, incoming *ReplayMovieSpec, path, sourceURL string) (*ReplayMovieSpec, error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("recorded video requires an execution-owned HTTP asset URL")
	}
	meta, err := probeRecordedVideo(ctx, path)
	if err != nil {
		return nil, err
	}
	return recordedVideoSpec(incoming, sourceURL, meta.durationMs, meta.width, meta.height)
}

// recordedMetadata is measured from media, never inferred from its filename.
type recordedMetadata struct {
	width, height, durationMs int32
	format                    string
}

func probeRecordedVideo(ctx context.Context, path string) (recordedMetadata, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "ffprobe", "-v", "error", "-threads", "1", "-select_streams", "v:0", "-show_entries", "stream=width,height:format=duration,format_name", "-of", "json", path)
	cmd.WaitDelay = 2 * time.Second
	data, err := cmd.Output()
	if err != nil {
		return recordedMetadata{}, fmt.Errorf("probe recorded video: %w", err)
	}
	var meta struct {
		Streams []struct {
			Width  int32 `json:"width"`
			Height int32 `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Name     string `json:"format_name"`
		} `json:"format"`
	}
	if err = json.Unmarshal(data, &meta); err != nil {
		return recordedMetadata{}, err
	}
	seconds, err := strconv.ParseFloat(meta.Format.Duration, 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 || seconds > 600 || len(meta.Streams) != 1 || meta.Streams[0].Width < 1 || meta.Streams[0].Height < 1 {
		return recordedMetadata{}, fmt.Errorf("recorded video must have a measured raster and duration within 600 seconds")
	}
	return recordedMetadata{width: meta.Streams[0].Width, height: meta.Streams[0].Height, durationMs: int32(math.Round(seconds * 1000)), format: meta.Format.Name}, nil
}

func recordedVideoSpec(incoming *ReplayMovieSpec, sourceURL string, duration, width, height int32) (*ReplayMovieSpec, error) {
	spec := &ReplayMovieSpec{}
	if incoming != nil {
		spec = proto.Clone(incoming).(*ReplayMovieSpec)
	}
	if spec.GetIntroCard().GetEnabled() || spec.GetOutroCard().GetEnabled() {
		return nil, fmt.Errorf("recorded-video cards require a composition edit; disable them or choose replay_frames")
	}
	if spec.Decor == nil {
		spec.Decor = &exportsv1.ReplayDecor{}
	}
	// Editorial replay cursor tracks are not time-aligned to a continuous source.
	// Preserve source pixels; never fabricate a second moving pointer.
	spec.Decor.CursorTheme = "disabled"
	if spec.Cursor != nil {
		spec.Cursor.Style = "none"
	}
	spec.RecordedVideo = &exportsv1.ReplayRecordedVideo{Source: sourceURL, DurationMs: duration}
	if spec.Presentation == nil {
		spec.Presentation = &exportsv1.ReplayPresentation{}
	}
	spec.Presentation.Viewport = &exportsv1.ReplayDimensions{Width: width, Height: height}
	if spec.Presentation.GetCanvas().GetWidth() <= 0 || spec.Presentation.GetCanvas().GetHeight() <= 0 {
		spec.Presentation.Canvas = &exportsv1.ReplayDimensions{Width: width, Height: height}
	}
	spec.Presentation.DeviceScaleFactor = 1
	interval := int32(defaultCaptureInterval)
	if spec.GetPlayback().GetFrameIntervalMs() > 0 {
		interval = spec.Playback.FrameIntervalMs
	}
	if interval < 20 {
		interval = 20
	}
	spec.Playback = &exportsv1.ReplayPlayback{FrameIntervalMs: interval, Fps: int32(math.Round(1000 / float64(interval))), DurationMs: duration, TotalFrames: int32(math.Ceil(float64(duration) / float64(interval)))}
	spec.Summary = &exportsv1.ReplaySummary{FrameCount: 1, TotalDurationMs: duration, MaxFrameDurationMs: duration}
	spec.Frames = []*exportsv1.ReplayFrame{{Index: 0, NodeId: "Browser", DurationMs: duration, Viewport: spec.Presentation.Viewport}}
	// No obsolete screenshot requests should be loaded for continuous footage.
	spec.Assets = nil
	return spec, nil
}

// FinishRecordedVideo retains source audio and clips the final sample hold to
// the measured source duration. GIF has no audio channel by definition.
func FinishRecordedVideo(ctx context.Context, media *RenderedMedia, sourcePath string, durationMs int32, format string) error {
	if format == "gif" {
		return nil
	}
	temporary, err := os.CreateTemp(filepath.Dir(media.Path), "recorded-master-*.mp4")
	if err != nil {
		return err
	}
	name := temporary.Name()
	temporary.Close()
	defer os.Remove(name)
	args := []string{"-v", "error", "-threads", "2", "-y", "-i", media.Path, "-i", sourcePath, "-map", "0:v:0", "-map", "1:a?", "-c:v", "libx264", "-preset", "veryfast", "-crf", "18", "-threads", "2", "-pix_fmt", "yuv420p", "-c:a", "aac", "-t", strconv.FormatFloat(float64(durationMs)/1000, 'f', 3, 64), "-movflags", "+faststart", name}
	command := exec.CommandContext(ctx, DetectFFmpegBinary(), args...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("retain recorded audio: %w: %s", err, output)
	}
	return os.Rename(name, media.Path)
}

// BrowserCompatibleRecording retains the raw source and creates only an owned,
// temporary WebM derivative when a source container is not native WebM. The
// caller serves it through the existing recording-asset owner and cleans it up.
func BrowserCompatibleRecording(ctx context.Context, input, assetsDir string) (string, func(), error) {
	if strings.EqualFold(filepath.Ext(input), ".webm") {
		return input, func() {}, nil
	}
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return "", func() {}, err
	}
	dir, err := os.MkdirTemp(assetsDir, "video-source-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	output := filepath.Join(dir, "source.webm")
	limited, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	command := exec.CommandContext(limited, DetectFFmpegBinary(), "-v", "error", "-threads", "2", "-i", input, "-map", "0:v:0", "-an", "-c:v", "libvpx-vp9", "-deadline", "realtime", "-cpu-used", "6", "-threads", "2", output)
	if diagnostic, err := command.CombinedOutput(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("prepare browser-compatible recording: %w: %s", err, diagnostic)
	}
	return output, cleanup, nil
}
