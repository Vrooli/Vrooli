package screenrecording

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// MediaInspection is the producer-side integrity record for a recorded video.
// It deliberately contains only derived metadata; the capture service remains
// the owner of the bytes and checksum.
type MediaInspection struct {
	Container   string
	Codec       string
	Width       int
	Height      int
	FrameRate   string
	FrameCount  int64
	DurationMs  int64
	UsefulFrame bool
}

// minimumUsefulApplicationLuma is deliberately close to the supported Xvfb
// desktop background (~50 Y). The generated apps can use a dark theme, so a
// large absolute brightness cutoff rejects real application frames. A small
// measured delta still rejects an otherwise uniform desktop.
const minimumUsefulApplicationLuma = 52

const (
	minimumUsefulApplicationPeak       = 96
	minimumUsefulApplicationLumaSpread = 24
)

type ffprobeResult struct {
	Streams []struct {
		CodecName  string `json:"codec_name"`
		CodecType  string `json:"codec_type"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		FrameRate  string `json:"avg_frame_rate"`
		Duration   string `json:"duration"`
		FrameCount string `json:"nb_frames"`
	} `json:"streams"`
	Format struct {
		Name     string `json:"format_name"`
		Duration string `json:"duration"`
	} `json:"format"`
}

// mediaCommand is a seam for deterministic tests without replacing ffmpeg in
// production. It is intentionally package-local: callers use InspectVideo.
var mediaCommand = exec.CommandContext

// InspectVideo validates that path is a decodable, non-empty MP4 recording and
// rejects the blank/static desktop regression through a bounded frame sample.
func InspectVideo(ctx context.Context, path string) (MediaInspection, error) {
	info, err := os.Stat(path)
	if err != nil {
		return MediaInspection{}, fmt.Errorf("stat recording: %w", err)
	}
	if info.Size() <= 0 {
		return MediaInspection{}, fmt.Errorf("recording is empty")
	}

	probeCmd := mediaCommand(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name,codec_type,width,height,avg_frame_rate,duration,nb_frames", "-show_entries", "format=format_name,duration", "-of", "json", path)
	probeOutput, err := probeCmd.Output()
	if err != nil {
		return MediaInspection{}, fmt.Errorf("probe recording: %w", err)
	}
	var probe ffprobeResult
	if err := json.Unmarshal(probeOutput, &probe); err != nil {
		return MediaInspection{}, fmt.Errorf("decode media metadata: %w", err)
	}
	if len(probe.Streams) == 0 || probe.Streams[0].CodecType != "video" {
		return MediaInspection{}, fmt.Errorf("recording has no video stream")
	}
	if !strings.Contains(strings.ToLower(probe.Format.Name), "mp4") {
		return MediaInspection{}, fmt.Errorf("recording container is not MP4: %q", probe.Format.Name)
	}
	stream := probe.Streams[0]
	duration, err := firstPositiveFloat(stream.Duration, probe.Format.Duration)
	if err != nil || duration < 1 {
		return MediaInspection{}, fmt.Errorf("recording duration is missing or too short")
	}
	frameCount := int64(0)
	if stream.FrameCount != "" {
		frameCount, _ = strconv.ParseInt(stream.FrameCount, 10, 64)
	}
	if stream.CodecName == "" || stream.Width <= 0 || stream.Height <= 0 || frameCount == 0 && duration <= 0 {
		return MediaInspection{}, fmt.Errorf("recording video metadata is incomplete")
	}

	// signalstats metadata is emitted for frames sampled every two seconds across
	// the recording. Sampling the full duration matters because Electron startup
	// can leave the first few seconds on a dark Xvfb desktop. The app-owned
	// display background is about 50 Y on the supported Xvfb profile; requiring
	// a materially brighter sampled frame (rather than the old 80-Y cutoff)
	// accepts dark application themes while still rejecting the uniform desktop.
	filter := "select='not(mod(n,30))',signalstats,metadata=print:file=-"
	frameCmd := mediaCommand(ctx, "ffmpeg", "-v", "error", "-i", path, "-vf", filter, "-frames:v", "20", "-f", "null", "-")
	frameOutput, err := frameCmd.CombinedOutput()
	if err != nil {
		return MediaInspection{}, fmt.Errorf("decode recording frames: %w", err)
	}
	if !usefulFrames(string(frameOutput)) {
		return MediaInspection{}, fmt.Errorf("recording has no useful application frames")
	}

	return MediaInspection{
		Container:   probe.Format.Name,
		Codec:       stream.CodecName,
		Width:       stream.Width,
		Height:      stream.Height,
		FrameRate:   stream.FrameRate,
		FrameCount:  frameCount,
		DurationMs:  int64(duration * 1000),
		UsefulFrame: true,
	}, nil
}

func firstPositiveFloat(values ...string) (float64, error) {
	for _, value := range values {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && parsed > 0 {
			return parsed, nil
		}
	}
	return 0, fmt.Errorf("no positive duration")
}

func usefulFrames(output string) bool {
	var maxAverage float64
	var maxPeak float64
	minLuma := 255.0
	for _, line := range strings.Split(output, "\n") {
		for key, destination := range map[string]*float64{
			"lavfi.signalstats.YAVG=": &maxAverage,
			"lavfi.signalstats.YMAX=": &maxPeak,
			"lavfi.signalstats.YMIN=": &minLuma,
		} {
			if !strings.Contains(line, key) {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(line, key))
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			switch destination {
			case &maxAverage, &maxPeak:
				if parsed > *destination {
					*destination = parsed
				}
			default:
				if parsed < *destination {
					*destination = parsed
				}
			}
		}
	}
	return maxAverage >= minimumUsefulApplicationLuma ||
		(maxPeak >= minimumUsefulApplicationPeak && maxPeak-minLuma >= minimumUsefulApplicationLumaSpread)
}
