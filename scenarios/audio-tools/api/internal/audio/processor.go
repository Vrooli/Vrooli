package audio

import (
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
	ID               string    `json:"id"`
	Path             string    `json:"path"`
	Format           string    `json:"format"`
	DurationSeconds  float64   `json:"duration_seconds"`
	SampleRate       int       `json:"sample_rate"`
	Channels         int       `json:"channels"`
	Bitrate          int       `json:"bitrate"`
	FileSizeBytes    int64     `json:"file_size_bytes"`
	CreatedAt        time.Time `json:"created_at"`
}

type AudioMetadata struct {
	Duration    string            `json:"duration"`
	Format      string            `json:"format"`
	Codec       string            `json:"codec"`
	SampleRate  string            `json:"sample_rate"`
	Channels    int               `json:"channels"`
	Bitrate     string            `json:"bitrate"`
	Size        int64             `json:"size"`
	Tags        map[string]string `json:"tags"`
}

// ExtractMetadata uses ffprobe to extract audio file metadata
func (p *AudioProcessor) ExtractMetadata(filePath string) (*AudioMetadata, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Use ffprobe to get metadata
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		// Fallback to basic metadata if ffprobe is not available
		return &AudioMetadata{
			Size:   fileInfo.Size(),
			Format: filepath.Ext(filePath)[1:], // Remove the dot
		}, nil
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

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("conversion failed: %v - ffmpeg output: %s", err, string(output))
	}

	return outputPath, nil
}

// Trim audio file
func (p *AudioProcessor) Trim(inputPath string, startTime, endTime float64) (string, error) {
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

	cmd := exec.Command("ffmpeg",
		"-y", // Overwrite output files without asking
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-t", fmt.Sprintf("%.2f", duration),
		"-c", "copy",
		"-loglevel", "error", // Only show errors
		outputPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("trim failed: %v - ffmpeg output: %s", err, string(output))
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("trim operation completed but output file not created")
	}

	return outputPath, nil
}

// Merge multiple audio files
func (p *AudioProcessor) Merge(inputPaths []string, outputFormat string) (string, error) {
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

	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
		"-c", "copy",
		"-loglevel", "error",
		outputPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("merge failed: %v - ffmpeg output: %s", err, string(output))
	}

	return outputPath, nil
}

// AdjustVolume adjusts the volume of an audio file
func (p *AudioProcessor) AdjustVolume(inputPath string, volumeFactor float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_volume%s", outputID, filepath.Ext(inputPath)))

	cmd := exec.Command("ffmpeg",
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

	cmd := exec.Command("ffmpeg",
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
	cmd := exec.Command("ffmpeg",
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

	// Use loudnorm filter for EBU R128 normalization
	cmd := exec.Command("ffmpeg",
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

	// Use highpass and lowpass filters for basic noise reduction
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-af", fmt.Sprintf("highpass=f=200,lowpass=f=3000,afftdn=nf=%.2f", intensity),
		outputPath)

	if err := cmd.Run(); err != nil {
		// Fallback to simpler noise reduction if afftdn is not available
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-af", "highpass=f=200,lowpass=f=3000",
			outputPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("noise reduction failed: %v", err)
		}
	}

	return outputPath, nil
}

// ChangeSpeed changes the speed/tempo of audio
func (p *AudioProcessor) ChangeSpeed(inputPath string, speedFactor float64) (string, error) {
	outputID := uuid.New().String()
	outputPath := filepath.Join(p.WorkDir, fmt.Sprintf("%s_speed%s", outputID, filepath.Ext(inputPath)))

	cmd := exec.Command("ffmpeg",
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
	
	cmd := exec.Command("ffmpeg",
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

	cmd := exec.Command("ffmpeg",
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
		
		cmd := exec.Command("ffmpeg",
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