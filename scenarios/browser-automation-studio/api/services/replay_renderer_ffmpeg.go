package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

// assembleVideoFromSequence uses ffmpeg to compile a sequence of frames into an MP4 video.
func (r *ReplayRenderer) assembleVideoFromSequence(ctx context.Context, pattern string, fps int, outputPath string) error {
	if fps <= 0 {
		fps = 25
	}
	args := []string{
		"-y",
		"-framerate", strconv.Itoa(fps),
		"-start_number", "0",
		"-i", pattern,
		"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2,format=yuv420p",
		"-c:v", "libx264",
		"-profile:v", "high",
		"-level", "4.1",
		"-crf", "21",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, r.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg sequence assembly failed: %w (%s)", err, stderr.String())
	}
	return nil
}

// convertToGIF uses ffmpeg to convert a video file to an animated GIF.
func (r *ReplayRenderer) convertToGIF(ctx context.Context, inputPath, outputPath string, targetWidth int, fps int) error {
	if fps <= 0 {
		fps = 12
	}
	if targetWidth <= 0 {
		targetWidth = defaultPresentationWidth
	}
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos", fps, targetWidth),
		outputPath,
	}
	cmd := exec.CommandContext(ctx, r.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg gif conversion failed: %w (%s)", err, stderr.String())
	}
	return nil
}
