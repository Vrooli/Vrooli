// Package media owns measured source-media facts shared by capture and export.
package media

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"time"
)

// Video describes measured media, independent of requested viewport or DPR.
type Video struct {
	Width, Height, DurationMs int32
	Format                    string
}

// ProbeVideo bounds the external probe and rejects missing or invalid measurements.
func ProbeVideo(ctx context.Context, path string) (Video, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "ffprobe", "-v", "error", "-threads", "1", "-select_streams", "v:0", "-show_entries", "stream=width,height:format=duration,format_name", "-of", "json", path)
	cmd.WaitDelay = 2 * time.Second
	data, err := cmd.Output()
	if err != nil {
		return Video{}, fmt.Errorf("probe recorded video: %w", err)
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
		return Video{}, err
	}
	seconds, err := strconv.ParseFloat(meta.Format.Duration, 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 || seconds > 600 || len(meta.Streams) != 1 || meta.Streams[0].Width < 1 || meta.Streams[0].Height < 1 {
		return Video{}, fmt.Errorf("recorded video must have a measured raster and duration within 600 seconds")
	}
	return Video{Width: meta.Streams[0].Width, Height: meta.Streams[0].Height, DurationMs: int32(math.Round(seconds * 1000)), Format: meta.Format.Name}, nil
}
