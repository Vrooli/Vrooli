package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AudioProcessor struct {
	WorkDir string
	DataDir string
}

func NewAudioProcessor(workDir, dataDir string) *AudioProcessor {
	return &AudioProcessor{
		WorkDir: workDir,
		DataDir: dataDir,
	}
}

type EditOperation struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

type AudioFile struct {
	ID              string    `json:"id"`
	Path            string    `json:"path"`
	Format          string    `json:"format"`
	DurationSeconds float64   `json:"duration_seconds"`
	SampleRate      int       `json:"sample_rate"`
	Channels        int       `json:"channels"`
	Bitrate         int       `json:"bitrate"`
	FileSizeBytes   int64     `json:"file_size_bytes"`
	CreatedAt       time.Time `json:"created_at"`
}

type AudioMetadata struct {
	Duration   string            `json:"duration"`
	Format     string            `json:"format"`
	Codec      string            `json:"codec"`
	SampleRate string            `json:"sample_rate"`
	Channels   int               `json:"channels"`
	Bitrate    string            `json:"bitrate"`
	Size       int64             `json:"size"`
	Tags       map[string]string `json:"tags"`
}

// ExtractMetadata uses ffprobe to extract audio file metadata
func (p *AudioProcessor) ExtractMetadata(filePath string) (*AudioMetadata, error) {
	return p.ExtractMetadataWithContext(context.Background(), filePath)
}

// ExtractMetadataWithContext uses ffprobe to extract audio file metadata with timeout support
func (p *AudioProcessor) ExtractMetadataWithContext(ctx context.Context, filePath string) (*AudioMetadata, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Create context with timeout if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	// Use ffprobe to get metadata
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		// Return error for invalid audio files
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	var ffprobeOutput map[string]interface{}
	if err := json.Unmarshal(output, &ffprobeOutput); err != nil {
		return nil, err
	}

	metadata := &AudioMetadata{
		Size: fileInfo.Size(),
		Tags: make(map[string]string),
	}

	// Parse format information
	if format, ok := ffprobeOutput["format"].(map[string]interface{}); ok {
		if duration, ok := format["duration"].(string); ok {
			metadata.Duration = duration
		}
		if formatName, ok := format["format_name"].(string); ok {
			metadata.Format = formatName
		}
		if bitrate, ok := format["bit_rate"].(string); ok {
			metadata.Bitrate = bitrate
		}

		// Extract tags
		if tags, ok := format["tags"].(map[string]interface{}); ok {
			for k, v := range tags {
				metadata.Tags[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Parse stream information
	if streams, ok := ffprobeOutput["streams"].([]interface{}); ok && len(streams) > 0 {
		if stream, ok := streams[0].(map[string]interface{}); ok {
			if codecName, ok := stream["codec_name"].(string); ok {
				metadata.Codec = codecName
			}
			if sampleRate, ok := stream["sample_rate"].(string); ok {
				metadata.SampleRate = sampleRate
			}
			if channels, ok := stream["channels"].(float64); ok {
				metadata.Channels = int(channels)
			}
		}
	}

	return metadata, nil
}

// ConvertFormat converts audio file to a different format
func (p *AudioProcessor) ConvertFormat(inputPath, outputFormat string, quality map[string]interface{}) (string, error) {
	return p.ConvertFormatWithContext(context.Background(), inputPath, outputFormat, quality)
}

// ConvertFormatWithContext converts audio file to a different format with timeout support
func (p *AudioProcessor) ConvertFormatWithContext(ctx context.Context, inputPath, outputFormat string, quality map[string]interface{}) (string, error) {
	// Generate output filename
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s.%s", outputID, outputFormat))

	// Build ffmpeg command
	args := []string{"-i", inputPath}

	// Add quality settings if provided
	if bitrate, ok := quality["bitrate"].(float64); ok {
		args = append(args, "-b:a", fmt.Sprintf("%dk", int(bitrate)))
	}
	if sampleRate, ok := quality["sample_rate"].(float64); ok {
		args = append(args, "-ar", strconv.Itoa(int(sampleRate)))
	}
	if channels, ok := quality["channels"].(float64); ok {
		args = append(args, "-ac", strconv.Itoa(int(channels)))
	}

	// Add codec based on format
	switch outputFormat {
	case "mp3":
		args = append(args, "-codec:a", "libmp3lame")
	case "wav":
		args = append(args, "-codec:a", "pcm_s16le")
	case "flac":
		args = append(args, "-codec:a", "flac")
	case "aac":
		args = append(args, "-codec:a", "aac")
	case "ogg":
		args = append(args, "-codec:a", "libvorbis")
	}

	// Add -y flag to overwrite at the beginning
	args = append([]string{"-y", "-loglevel", "error"}, args...)
	args = append(args, outputPath)

	// Create context with timeout if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	// Wait for it to complete with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Timeout reached, kill the process
		if err := cmd.Process.Kill(); err != nil {
			// Process might have already exited
		}
		return "", fmt.Errorf("conversion operation timed out")
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("conversion failed: %v", err)
		}
	}

	return outputPath, nil
}

// Trim audio file
func (p *AudioProcessor) Trim(inputPath string, startTime, endTime float64) (string, error) {
	return p.TrimWithContext(context.Background(), inputPath, startTime, endTime)
}

// TrimWithContext trims audio file with timeout support
func (p *AudioProcessor) TrimWithContext(ctx context.Context, inputPath string, startTime, endTime float64) (string, error) {
	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("input file not found: %s", inputPath)
	}

	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_trimmed%s", outputID, filepath.Ext(inputPath)))

	duration := endTime - startTime
	if duration <= 0 {
		return "", fmt.Errorf("invalid time range: start=%.2f, end=%.2f", startTime, endTime)
	}

	// Create context with timeout if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y", // Overwrite output files without asking
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-t", fmt.Sprintf("%.2f", duration),
		"-c", "copy",
		"-loglevel", "error", // Only show errors
		outputPath)

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	// Wait for it to complete with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Timeout reached, kill the process
		if err := cmd.Process.Kill(); err != nil {
			// Process might have already exited
		}
		return "", fmt.Errorf("trim operation timed out after 30 seconds")
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("trim failed: %v", err)
		}
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("trim operation completed but output file not created")
	}

	return outputPath, nil
}

// Merge multiple audio files
func (p *AudioProcessor) Merge(inputPaths []string, outputFormat string) (string, error) {
	return p.MergeWithContext(context.Background(), inputPaths, outputFormat)
}

// MergeWithContext merges multiple audio files with timeout support
func (p *AudioProcessor) MergeWithContext(ctx context.Context, inputPaths []string, outputFormat string) (string, error) {
	if len(inputPaths) < 2 {
		return "", fmt.Errorf("at least 2 files required for merge")
	}

	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_merged.%s", outputID, outputFormat))

	// Create concat file
	concatFile := filepath.Join(p.WorkDir, fmt.Sprintf("%s_concat.txt", outputID))
	f, err := os.Create(concatFile)
	if err != nil {
		return "", err
	}
	defer os.Remove(concatFile)

	for _, path := range inputPaths {
		fmt.Fprintf(f, "file '%s'\n", path)
	}
	f.Close()

	// Create context with timeout if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
		"-c", "copy",
		"-loglevel", "error",
		outputPath)

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	// Wait for it to complete with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Timeout reached, kill the process
		if err := cmd.Process.Kill(); err != nil {
			// Process might have already exited
		}
		return "", fmt.Errorf("merge operation timed out")
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("merge failed: %v", err)
		}
	}

	return outputPath, nil
}

// AdjustVolume adjusts the volume of an audio file
func (p *AudioProcessor) AdjustVolume(inputPath string, volumeFactor float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_volume%s", outputID, filepath.Ext(inputPath)))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-filter:a", fmt.Sprintf("volume=%.2f", volumeFactor),
		"-loglevel", "error",
		outputPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("volume adjustment failed: %v - ffmpeg output: %s", err, string(output))
	}

	return outputPath, nil
}

// FadeIn applies fade-in effect
func (p *AudioProcessor) FadeIn(inputPath string, duration float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_fadein%s", outputID, filepath.Ext(inputPath)))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-af", fmt.Sprintf("afade=t=in:st=0:d=%.2f", duration),
		"-loglevel", "error",
		outputPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("fade-in failed: %v - ffmpeg output: %s", err, string(output))
	}

	return outputPath, nil
}

// FadeOut applies fade-out effect
func (p *AudioProcessor) FadeOut(inputPath string, duration float64) (string, error) {
	// First get the total duration of the file
	metadata, err := p.ExtractMetadata(inputPath)
	if err != nil {
		return "", err
	}

	totalDuration, err := strconv.ParseFloat(metadata.Duration, 64)
	if err != nil {
		return "", fmt.Errorf("could not parse duration: %v", err)
	}

	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_fadeout%s", outputID, filepath.Ext(inputPath)))

	startTime := totalDuration - duration

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-af", fmt.Sprintf("afade=t=out:st=%.2f:d=%.2f", startTime, duration),
		"-loglevel", "error",
		outputPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("fade-out failed: %v - ffmpeg output: %s", err, string(output))
	}

	return outputPath, nil
}

// Normalize audio levels
func (p *AudioProcessor) Normalize(inputPath string, targetLevel float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_normalized%s", outputID, filepath.Ext(inputPath)))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use loudnorm filter for EBU R128 normalization
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-loglevel", "error",
		"-i", inputPath,
		"-af", fmt.Sprintf("loudnorm=I=%.1f:TP=-1.5:LRA=11", targetLevel),
		outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("normalization failed: %v", err)
	}

	return outputPath, nil
}

// ApplyNoiseReduction applies basic noise reduction
func (p *AudioProcessor) ApplyNoiseReduction(inputPath string, intensity float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_denoised%s", outputID, filepath.Ext(inputPath)))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use highpass and lowpass filters for basic noise reduction
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-loglevel", "error",
		"-i", inputPath,
		"-af", fmt.Sprintf("highpass=f=200,lowpass=f=3000,afftdn=nf=%.2f", intensity),
		outputPath)

	if err := cmd.Run(); err != nil {
		// Fallback to simpler noise reduction if afftdn is not available
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-y",
			"-loglevel", "error",
			"-i", inputPath,
			"-af", "highpass=f=200,lowpass=f=3000",
			outputPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("noise reduction failed: %v", err)
		}
	}

	return outputPath, nil
}

// VoiceActivityDetection represents VAD results
type VoiceActivityDetection struct {
	SpeechSegments  []SpeechSegment `json:"speech_segments"`
	TotalDuration   float64         `json:"total_duration_seconds"`
	SpeechDuration  float64         `json:"speech_duration_seconds"`
	SilenceDuration float64         `json:"silence_duration_seconds"`
	SpeechRatio     float64         `json:"speech_ratio"`
}

// SpeechSegment represents a segment of detected speech
type SpeechSegment struct {
	StartTime float64 `json:"start_time_seconds"`
	EndTime   float64 `json:"end_time_seconds"`
	Duration  float64 `json:"duration_seconds"`
}

// DetectVoiceActivity performs voice activity detection on audio
func (p *AudioProcessor) DetectVoiceActivity(inputPath string, threshold float64) (*VoiceActivityDetection, error) {
	// Use silencedetect filter to find speech segments
	if threshold <= 0 {
		threshold = -40 // Default threshold in dB
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First get total duration
	metadata, err := p.ExtractMetadata(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %v", err)
	}

	// Parse duration string to float64
	totalDuration, err := strconv.ParseFloat(metadata.Duration, 64)
	if err != nil {
		// Try to parse duration in format "HH:MM:SS.ms"
		parts := strings.Split(metadata.Duration, ":")
		if len(parts) == 3 {
			hours, _ := strconv.ParseFloat(parts[0], 64)
			minutes, _ := strconv.ParseFloat(parts[1], 64)
			seconds, _ := strconv.ParseFloat(parts[2], 64)
			totalDuration = hours*3600 + minutes*60 + seconds
		} else {
			return nil, fmt.Errorf("failed to parse duration: %v", err)
		}
	}

	// Detect silence (which gives us speech segments by inversion)
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-af", fmt.Sprintf("silencedetect=noise=%.0fdB:d=0.3", threshold),
		"-f", "null",
		"-")

	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	vad := &VoiceActivityDetection{
		TotalDuration:  totalDuration,
		SpeechSegments: []SpeechSegment{},
	}

	// Parse silence detection output to find speech segments
	lines := strings.Split(outputStr, "\n")
	var silenceStart float64 = 0
	var lastSilenceEnd float64 = 0

	for _, line := range lines {
		if strings.Contains(line, "silence_start:") {
			// Extract silence start time
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "silence_start:" && i+1 < len(parts) {
					if val, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
						silenceStart = val
						// Add speech segment from last silence end to this silence start
						if silenceStart > lastSilenceEnd {
							segment := SpeechSegment{
								StartTime: lastSilenceEnd,
								EndTime:   silenceStart,
								Duration:  silenceStart - lastSilenceEnd,
							}
							vad.SpeechSegments = append(vad.SpeechSegments, segment)
							vad.SpeechDuration += segment.Duration
						}
					}
				}
			}
		} else if strings.Contains(line, "silence_end:") {
			// Extract silence end time
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "silence_end:" && i+1 < len(parts) {
					if val, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
						lastSilenceEnd = val
					}
				}
			}
		}
	}

	// Add final speech segment if audio doesn't end with silence
	if lastSilenceEnd < totalDuration {
		segment := SpeechSegment{
			StartTime: lastSilenceEnd,
			EndTime:   totalDuration,
			Duration:  totalDuration - lastSilenceEnd,
		}
		vad.SpeechSegments = append(vad.SpeechSegments, segment)
		vad.SpeechDuration += segment.Duration
	}

	vad.SilenceDuration = totalDuration - vad.SpeechDuration
	if totalDuration > 0 {
		vad.SpeechRatio = vad.SpeechDuration / totalDuration
	}

	return vad, nil
}

// RemoveSilence removes silence from audio, keeping only speech segments
func (p *AudioProcessor) RemoveSilence(inputPath string, threshold float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_speech_only%s", outputID, filepath.Ext(inputPath)))

	if threshold <= 0 {
		threshold = -40 // Default threshold in dB
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use silenceremove filter to remove silence
	filterStr := fmt.Sprintf("silenceremove=start_periods=1:start_duration=0.1:start_threshold=%.0fdB:detection=peak,areverse,silenceremove=start_periods=1:start_duration=0.1:start_threshold=%.0fdB:detection=peak,areverse", threshold, threshold)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-loglevel", "error",
		"-i", inputPath,
		"-af", filterStr,
		outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("silence removal failed: %v", err)
	}

	return outputPath, nil
}

// ChangeSpeed changes the speed/tempo of audio
func (p *AudioProcessor) ChangeSpeed(inputPath string, speedFactor float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_speed%s", outputID, filepath.Ext(inputPath)))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-loglevel", "error",
		"-i", inputPath,
		"-filter:a", fmt.Sprintf("atempo=%.2f", speedFactor),
		outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("speed change failed: %v", err)
	}

	return outputPath, nil
}

// ChangePitch changes the pitch without affecting tempo
func (p *AudioProcessor) ChangePitch(inputPath string, pitchSemitones int) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_pitch%s", outputID, filepath.Ext(inputPath)))

	// Calculate frequency ratio from semitones
	ratio := math.Pow(2, float64(pitchSemitones)/12.0)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-loglevel", "error",
		"-i", inputPath,
		"-af", fmt.Sprintf("asetrate=%d*%.4f,aresample=%d", 44100, ratio, 44100),
		outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pitch change failed: %v", err)
	}

	return outputPath, nil
}

// ApplyEqualizer applies EQ settings
func (p *AudioProcessor) ApplyEqualizer(inputPath string, eqSettings map[string]float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_eq%s", outputID, filepath.Ext(inputPath)))

	// Build equalizer string
	var eqFilters []string
	for freq, gain := range eqSettings {
		eqFilters = append(eqFilters, fmt.Sprintf("equalizer=f=%s:g=%.1f", freq, gain))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-loglevel", "error",
		"-i", inputPath,
		"-af", strings.Join(eqFilters, ","),
		outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("equalizer failed: %v", err)
	}

	return outputPath, nil
}

// Split audio file at specified points
func (p *AudioProcessor) Split(inputPath string, splitPoints []float64) ([]string, error) {
	var outputPaths []string

	// Add 0 at the beginning if not present
	if len(splitPoints) == 0 || splitPoints[0] != 0 {
		splitPoints = append([]float64{0}, splitPoints...)
	}

	// Get total duration
	metadata, err := p.ExtractMetadata(inputPath)
	if err != nil {
		return nil, err
	}

	totalDuration, err := strconv.ParseFloat(metadata.Duration, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse duration: %v", err)
	}

	// Add end time if not present
	if splitPoints[len(splitPoints)-1] < totalDuration {
		splitPoints = append(splitPoints, totalDuration)
	}

	// Create segments
	for i := 0; i < len(splitPoints)-1; i++ {
		outputID := uuid.New().String()
		outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_part%d%s", outputID, i+1, filepath.Ext(inputPath)))

		startTime := splitPoints[i]
		duration := splitPoints[i+1] - startTime

		// Create context with timeout for each split operation
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-y",
			"-loglevel", "error",
			"-i", inputPath,
			"-ss", fmt.Sprintf("%.2f", startTime),
			"-t", fmt.Sprintf("%.2f", duration),
			"-c", "copy",
			outputPath)

		if err := cmd.Run(); err != nil {
			// Clean up any created files
			for _, path := range outputPaths {
				os.Remove(path)
			}
			return nil, fmt.Errorf("split failed at segment %d: %v", i+1, err)
		}

		outputPaths = append(outputPaths, outputPath)
	}

	return outputPaths, nil
}

// AnalyzeQuality analyzes audio quality metrics
func (p *AudioProcessor) AnalyzeQuality(inputPath string) (*QualityMetrics, error) {
	// Get basic metadata first
	metadata, err := p.ExtractMetadata(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %v", err)
	}

	metrics := &QualityMetrics{
		Format:     metadata.Format,
		Codec:      metadata.Codec,
		SampleRate: metadata.SampleRate,
		Channels:   metadata.Channels,
		Bitrate:    metadata.Bitrate,
	}

	// Analyze audio levels using ffmpeg's astats filter
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-af", "astats=metadata=1:reset=1",
		"-f", "null",
		"-")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try simpler analysis without astats
		return p.analyzeBasicQuality(inputPath, metrics)
	}

	// Parse the output for quality metrics
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "RMS level dB") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				if val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					metrics.RMSLevel = val
				}
			}
		} else if strings.Contains(line, "Peak level dB") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				if val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					metrics.PeakLevel = val
				}
			}
		} else if strings.Contains(line, "Dynamic range") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				if val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					metrics.DynamicRange = val
				}
			}
		}
	}

	// Calculate quality score based on metrics
	metrics.QualityScore = p.calculateQualityScore(metrics)

	// Detect issues
	metrics.Issues = p.detectQualityIssues(metrics)

	return metrics, nil
}

// analyzeBasicQuality provides basic quality analysis when astats is not available
func (p *AudioProcessor) analyzeBasicQuality(inputPath string, metrics *QualityMetrics) (*QualityMetrics, error) {
	// Use volumedetect filter as a fallback
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-af", "volumedetect",
		"-f", "null",
		"-")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Return basic metrics without audio analysis
		metrics.QualityScore = p.calculateBasicQualityScore(metrics)
		return metrics, nil
	}

	// Parse volumedetect output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "mean_volume:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "mean_volume:" && i+1 < len(parts) {
					if val, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
						metrics.RMSLevel = val
					}
				}
			}
		} else if strings.Contains(line, "max_volume:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "max_volume:" && i+1 < len(parts) {
					if val, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
						metrics.PeakLevel = val
					}
				}
			}
		}
	}

	metrics.QualityScore = p.calculateQualityScore(metrics)
	metrics.Issues = p.detectQualityIssues(metrics)

	return metrics, nil
}

// calculateQualityScore calculates an overall quality score (0-100)
func (p *AudioProcessor) calculateQualityScore(metrics *QualityMetrics) float64 {
	score := 100.0

	// Bitrate scoring (for compressed formats)
	if metrics.Format != "wav" && metrics.Format != "flac" {
		bitrate, _ := strconv.Atoi(metrics.Bitrate)
		if bitrate < 128000 {
			score -= 20
		} else if bitrate < 192000 {
			score -= 10
		}
	}

	// Sample rate scoring
	sampleRate, _ := strconv.Atoi(metrics.SampleRate)
	if sampleRate < 44100 {
		score -= 15
	} else if sampleRate < 48000 {
		score -= 5
	}

	// Audio level scoring
	if metrics.PeakLevel > -1.0 {
		score -= 15 // Likely clipping
	} else if metrics.PeakLevel < -20.0 {
		score -= 10 // Too quiet
	}

	if metrics.RMSLevel < -30.0 {
		score -= 10 // Very low average level
	}

	// Dynamic range scoring
	if metrics.DynamicRange > 0 && metrics.DynamicRange < 6.0 {
		score -= 10 // Over-compressed
	}

	return math.Max(0, math.Min(100, score))
}

// calculateBasicQualityScore for when we only have format metrics
func (p *AudioProcessor) calculateBasicQualityScore(metrics *QualityMetrics) float64 {
	score := 100.0

	// Basic scoring based on format/codec quality
	if metrics.Format != "wav" && metrics.Format != "flac" {
		bitrate, _ := strconv.Atoi(metrics.Bitrate)
		if bitrate < 128000 {
			score -= 30
		} else if bitrate < 192000 {
			score -= 15
		}
	}

	sampleRate, _ := strconv.Atoi(metrics.SampleRate)
	if sampleRate < 44100 {
		score -= 20
	}

	return math.Max(0, math.Min(100, score))
}

// detectQualityIssues identifies common audio quality issues
func (p *AudioProcessor) detectQualityIssues(metrics *QualityMetrics) []string {
	var issues []string

	if metrics.PeakLevel > -1.0 {
		issues = append(issues, "Possible clipping detected")
	}

	if metrics.PeakLevel < -20.0 {
		issues = append(issues, "Audio level too low")
	}

	if metrics.RMSLevel < -30.0 {
		issues = append(issues, "Very low average level")
	}

	if metrics.DynamicRange > 0 && metrics.DynamicRange < 6.0 {
		issues = append(issues, "Over-compressed (low dynamic range)")
	}

	bitrate, _ := strconv.Atoi(metrics.Bitrate)
	if metrics.Format != "wav" && metrics.Format != "flac" && bitrate < 128000 {
		issues = append(issues, "Low bitrate may affect quality")
	}

	sampleRate, _ := strconv.Atoi(metrics.SampleRate)
	if sampleRate < 44100 {
		issues = append(issues, "Low sample rate")
	}

	return issues
}

// QualityMetrics holds audio quality analysis results
type QualityMetrics struct {
	Format       string   `json:"format"`
	Codec        string   `json:"codec"`
	SampleRate   string   `json:"sample_rate"`
	Channels     int      `json:"channels"`
	Bitrate      string   `json:"bitrate"`
	RMSLevel     float64  `json:"rms_level_db"`
	PeakLevel    float64  `json:"peak_level_db"`
	DynamicRange float64  `json:"dynamic_range_db"`
	QualityScore float64  `json:"quality_score"`
	Issues       []string `json:"issues,omitempty"`
}
