package video

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VideoInfo contains metadata about a video file
type VideoInfo struct {
	Format          string  `json:"format"`
	Duration        float64 `json:"duration"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	FrameRate       float64 `json:"frame_rate"`
	BitRate         int     `json:"bitrate"`
	Codec           string  `json:"codec"`
	HasAudio        bool    `json:"has_audio"`
	AudioCodec      string  `json:"audio_codec,omitempty"`
	AudioChannels   int     `json:"audio_channels,omitempty"`
	AudioSampleRate int     `json:"audio_sample_rate,omitempty"`
	FileSize        int64   `json:"file_size"`
}

// EditOperation represents a video editing operation
type EditOperation struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ConversionOptions represents format conversion parameters
type ConversionOptions struct {
	TargetFormat string `json:"target_format"`
	Resolution   string `json:"resolution,omitempty"`
	Quality      string `json:"quality,omitempty"`
	Preset       string `json:"preset,omitempty"`
	VideoBitrate string `json:"video_bitrate,omitempty"`
	AudioBitrate string `json:"audio_bitrate,omitempty"`
	AudioCodec   string `json:"audio_codec,omitempty"`
}

// FrameExtractionOptions represents frame extraction parameters
type FrameExtractionOptions struct {
	Timestamps []float64 `json:"timestamps,omitempty"`
	Interval   float64   `json:"interval,omitempty"`
	Count      int       `json:"count,omitempty"`
	Format     string    `json:"format,omitempty"`
}

// Processor handles video processing operations
type Processor struct {
	workDir string
	ffmpeg  string
	ffprobe string
}

// NewProcessor creates a new video processor
func NewProcessor(workDir string) (*Processor, error) {
	// Ensure work directory exists
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	// Find ffmpeg and ffprobe
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found: %w", err)
	}

	return &Processor{
		workDir: workDir,
		ffmpeg:  ffmpeg,
		ffprobe: ffprobe,
	}, nil
}

// GetVideoInfo extracts metadata from a video file
func (p *Processor) GetVideoInfo(filePath string) (*VideoInfo, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Run ffprobe to get video metadata
	cmd := exec.Command(p.ffprobe,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Parse JSON output
	var probeData map[string]interface{}
	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &VideoInfo{
		FileSize: fileInfo.Size(),
	}

	// Extract format info
	if format, ok := probeData["format"].(map[string]interface{}); ok {
		if formatName, ok := format["format_name"].(string); ok {
			info.Format = strings.Split(formatName, ",")[0]
		}
		if duration, ok := format["duration"].(string); ok {
			info.Duration, _ = strconv.ParseFloat(duration, 64)
		}
		if bitrate, ok := format["bit_rate"].(string); ok {
			br, _ := strconv.Atoi(bitrate)
			info.BitRate = br / 1000 // Convert to kbps
		}
	}

	// Extract stream info
	if streams, ok := probeData["streams"].([]interface{}); ok {
		for _, stream := range streams {
			s := stream.(map[string]interface{})
			codecType, _ := s["codec_type"].(string)

			if codecType == "video" {
				info.Codec, _ = s["codec_name"].(string)
				if width, ok := s["width"].(float64); ok {
					info.Width = int(width)
				}
				if height, ok := s["height"].(float64); ok {
					info.Height = int(height)
				}

				// Calculate frame rate
				if fpsStr, ok := s["r_frame_rate"].(string); ok {
					parts := strings.Split(fpsStr, "/")
					if len(parts) == 2 {
						num, _ := strconv.ParseFloat(parts[0], 64)
						den, _ := strconv.ParseFloat(parts[1], 64)
						if den > 0 {
							info.FrameRate = num / den
						}
					}
				}
			} else if codecType == "audio" {
				info.HasAudio = true
				info.AudioCodec, _ = s["codec_name"].(string)
				if channels, ok := s["channels"].(float64); ok {
					info.AudioChannels = int(channels)
				}
				if sampleRate, ok := s["sample_rate"].(string); ok {
					info.AudioSampleRate, _ = strconv.Atoi(sampleRate)
				}
			}
		}
	}

	return info, nil
}

// ConvertFormat converts a video to a different format
func (p *Processor) ConvertFormat(inputPath string, outputPath string, options ConversionOptions) error {
	args := []string{"-i", inputPath}

	// Add quality/compression settings
	switch options.Quality {
	case "lossless":
		args = append(args, "-crf", "0")
	case "high":
		args = append(args, "-crf", "18")
	case "medium":
		args = append(args, "-crf", "23")
	case "low":
		args = append(args, "-crf", "28")
	}

	// Add preset for encoding speed
	if options.Preset != "" {
		args = append(args, "-preset", options.Preset)
	}

	// Add resolution if specified
	if options.Resolution != "" {
		args = append(args, "-vf", fmt.Sprintf("scale=%s", p.parseResolution(options.Resolution)))
	}

	// Add bitrate settings
	if options.VideoBitrate != "" {
		args = append(args, "-b:v", options.VideoBitrate)
	}
	if options.AudioBitrate != "" {
		args = append(args, "-b:a", options.AudioBitrate)
	}

	// Add audio codec if specified
	if options.AudioCodec != "" {
		args = append(args, "-c:a", options.AudioCodec)
	}

	// Add output file
	args = append(args, "-y", outputPath)

	// Execute ffmpeg
	cmd := exec.Command(p.ffmpeg, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	return nil
}

// Trim cuts a video between start and end times
func (p *Processor) Trim(inputPath string, outputPath string, startTime, endTime float64) error {
	duration := endTime - startTime

	args := []string{
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-i", inputPath,
		"-t", fmt.Sprintf("%.2f", duration),
		"-c", "copy", // Copy codecs for faster processing
		"-y", outputPath,
	}

	cmd := exec.Command(p.ffmpeg, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg trim failed: %w", err)
	}

	return nil
}

// Merge combines multiple video files
func (p *Processor) Merge(inputPaths []string, outputPath string) error {
	// Create a temporary file list for ffmpeg concat
	listFile := filepath.Join(p.workDir, fmt.Sprintf("merge_%d.txt", time.Now().Unix()))
	defer os.Remove(listFile)

	// Write file list
	var content strings.Builder
	for _, path := range inputPaths {
		content.WriteString(fmt.Sprintf("file '%s'\n", path))
	}
	if err := os.WriteFile(listFile, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to create file list: %w", err)
	}

	// Execute ffmpeg concat
	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-y", outputPath,
	}

	cmd := exec.Command(p.ffmpeg, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg merge failed: %w", err)
	}

	return nil
}

// ExtractFrames extracts frames from a video
func (p *Processor) ExtractFrames(inputPath string, outputDir string, options FrameExtractionOptions) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	format := options.Format
	if format == "" {
		format = "jpg"
	}

	var framePaths []string

	if len(options.Timestamps) > 0 {
		// Extract specific timestamps
		for i, timestamp := range options.Timestamps {
			outputPath := filepath.Join(outputDir, fmt.Sprintf("frame_%04d.%s", i, format))
			args := []string{
				"-ss", fmt.Sprintf("%.3f", timestamp),
				"-i", inputPath,
				"-vframes", "1",
				"-y", outputPath,
			}

			cmd := exec.Command(p.ffmpeg, args...)
			if err := cmd.Run(); err != nil {
				return framePaths, fmt.Errorf("failed to extract frame at %.3f: %w", timestamp, err)
			}
			framePaths = append(framePaths, outputPath)
		}
	} else if options.Interval > 0 {
		// Extract frames at intervals
		outputPattern := filepath.Join(outputDir, fmt.Sprintf("frame_%%04d.%s", format))
		fps := 1.0 / options.Interval
		args := []string{
			"-i", inputPath,
			"-vf", fmt.Sprintf("fps=%.3f", fps),
			"-y", outputPattern,
		}

		cmd := exec.Command(p.ffmpeg, args...)
		if err := cmd.Run(); err != nil {
			return framePaths, fmt.Errorf("failed to extract frames: %w", err)
		}

		// Collect generated frame paths
		files, _ := filepath.Glob(filepath.Join(outputDir, fmt.Sprintf("frame_*.%s", format)))
		framePaths = files
	}

	return framePaths, nil
}

// GenerateThumbnail creates a thumbnail from a video
func (p *Processor) GenerateThumbnail(inputPath string, outputPath string, timestamp float64, width int) error {
	args := []string{
		"-ss", fmt.Sprintf("%.3f", timestamp),
		"-i", inputPath,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:-1", width),
		"-y", outputPath,
	}

	cmd := exec.Command(p.ffmpeg, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return nil
}

// ExtractAudio extracts audio track from video
func (p *Processor) ExtractAudio(inputPath string, outputPath string) error {
	args := []string{
		"-i", inputPath,
		"-vn", // No video
		"-acodec", "copy",
		"-y", outputPath,
	}

	cmd := exec.Command(p.ffmpeg, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract audio: %w", err)
	}

	return nil
}

// AddSubtitles burns subtitles into video
func (p *Processor) AddSubtitles(videoPath string, subtitlePath string, outputPath string, burnIn bool) error {
	if burnIn {
		// Burn subtitles into video
		args := []string{
			"-i", videoPath,
			"-vf", fmt.Sprintf("subtitles=%s", subtitlePath),
			"-c:a", "copy",
			"-y", outputPath,
		}

		cmd := exec.Command(p.ffmpeg, args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to burn subtitles: %w", err)
		}
	} else {
		// Add subtitles as a stream
		args := []string{
			"-i", videoPath,
			"-i", subtitlePath,
			"-c", "copy",
			"-c:s", "mov_text",
			"-y", outputPath,
		}

		cmd := exec.Command(p.ffmpeg, args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add subtitle stream: %w", err)
		}
	}

	return nil
}

// Compress reduces video file size
func (p *Processor) Compress(inputPath string, outputPath string, targetSizeMB int) error {
	// Get video info to calculate bitrate
	info, err := p.GetVideoInfo(inputPath)
	if err != nil {
		return err
	}

	// Calculate target bitrate
	targetBitrate := int((float64(targetSizeMB) * 8192) / info.Duration) // kbps

	args := []string{
		"-i", inputPath,
		"-b:v", fmt.Sprintf("%dk", targetBitrate),
		"-maxrate", fmt.Sprintf("%dk", int(float64(targetBitrate)*1.5)),
		"-bufsize", fmt.Sprintf("%dk", targetBitrate*2),
		"-y", outputPath,
	}

	cmd := exec.Command(p.ffmpeg, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compression failed: %w", err)
	}

	return nil
}

// parseResolution converts resolution strings to ffmpeg scale format
func (p *Processor) parseResolution(resolution string) string {
	switch resolution {
	case "480p":
		return "854:480"
	case "720p":
		return "1280:720"
	case "1080p":
		return "1920:1080"
	case "4k":
		return "3840:2160"
	default:
		return resolution
	}
}
