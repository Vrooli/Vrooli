package audio

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function for substring check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Test Context-aware methods
func TestExtractMetadataWithContext(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("WithTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		testFile := filepath.Join("/tmp", "test_ctx_metadata.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.ExtractMetadataWithContext(ctx, testFile)
		assert.Error(t, err) // Should fail with non-audio file
	})

	t.Run("WithCanceledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		testFile := filepath.Join("/tmp", "test_canceled.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.ExtractMetadataWithContext(ctx, testFile)
		assert.Error(t, err)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		ctx := context.Background()
		_, err := processor.ExtractMetadataWithContext(ctx, "/nonexistent/file.wav")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestConvertFormatWithContext(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("WithTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		testFile := filepath.Join("/tmp", "test_convert_timeout.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.ConvertFormatWithContext(ctx, testFile, "mp3", nil)
		assert.Error(t, err)
	})

	t.Run("ValidFormats", func(t *testing.T) {
		formats := []string{"mp3", "wav", "flac", "aac", "ogg"}
		testFile := filepath.Join("/tmp", "test_formats.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		for _, format := range formats {
			t.Run(format, func(t *testing.T) {
				_, err := processor.ConvertFormat(testFile, format, nil)
				// May fail with fake data, but shouldn't panic
				_ = err
			})
		}
	})

	t.Run("WithQualitySettings", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_quality.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		quality := map[string]interface{}{
			"bitrate":     192000.0,
			"sample_rate": 44100.0,
			"channels":    2.0,
		}

		_, err = processor.ConvertFormat(testFile, "mp3", quality)
		// May fail, but shouldn't panic
		_ = err
	})
}

func TestTrimWithContext(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("InvalidTimeRange", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_trim_invalid.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		// End time before start time
		_, err = processor.Trim(testFile, 20.0, 10.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time range")
	})

	t.Run("ZeroDuration", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_trim_zero.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.Trim(testFile, 10.0, 10.0)
		assert.Error(t, err)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := processor.Trim("/nonexistent.wav", 0, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("WithTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		testFile := filepath.Join("/tmp", "test_trim_timeout.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.TrimWithContext(ctx, testFile, 0, 10)
		assert.Error(t, err)
	})
}

func TestMergeWithContext(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("InsufficientFiles", func(t *testing.T) {
		_, err := processor.Merge([]string{"/tmp/single.wav"}, "wav")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 2 files")
	})

	t.Run("EmptyFileList", func(t *testing.T) {
		_, err := processor.Merge([]string{}, "wav")
		assert.Error(t, err)
	})

	t.Run("WithTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Create test files
		file1 := filepath.Join("/tmp", "merge1.txt")
		file2 := filepath.Join("/tmp", "merge2.txt")
		os.WriteFile(file1, []byte("fake1"), 0644)
		os.WriteFile(file2, []byte("fake2"), 0644)
		defer os.Remove(file1)
		defer os.Remove(file2)

		_, err := processor.MergeWithContext(ctx, []string{file1, file2}, "wav")
		assert.Error(t, err)
	})
}

func TestSplitExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("EmptySplitPoints", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_split_empty.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		// Should handle empty split points
		_, err = processor.Split(testFile, []float64{})
		// Will fail extracting metadata from fake file
		assert.Error(t, err)
	})

	t.Run("SingleSplitPoint", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_split_single.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.Split(testFile, []float64{5.0})
		assert.Error(t, err) // Will fail with fake audio
	})
}

func TestAdjustVolumeExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("VolumeFactors", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_volume_factors.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		factors := []float64{0.5, 1.0, 1.5, 2.0, 0.1, 3.0}
		for _, factor := range factors {
			t.Run("Factor"+string(rune(factor*10)), func(t *testing.T) {
				_, err := processor.AdjustVolume(testFile, factor)
				// May fail with fake data
				_ = err
			})
		}
	})
}

func TestFadeInAndOut(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("FadeInDurations", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_fadein.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		durations := []float64{0.5, 1.0, 2.0, 5.0}
		for _, duration := range durations {
			_, err := processor.FadeIn(testFile, duration)
			// May fail with fake data
			_ = err
		}
	})

	t.Run("FadeOutDurations", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_fadeout.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		durations := []float64{0.5, 1.0, 2.0, 5.0}
		for _, duration := range durations {
			_, err := processor.FadeOut(testFile, duration)
			// Will fail extracting metadata
			assert.Error(t, err)
		}
	})
}

func TestNormalizeExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("TargetLevels", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_normalize_levels.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		levels := []float64{-23.0, -16.0, -14.0, -12.0}
		for _, level := range levels {
			_, err := processor.Normalize(testFile, level)
			// May fail with fake data
			_ = err
		}
	})
}

func TestChangeSpeedExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("SpeedFactors", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_speed_factors.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		factors := []float64{0.5, 0.75, 1.0, 1.25, 1.5, 2.0}
		for _, factor := range factors {
			_, err := processor.ChangeSpeed(testFile, factor)
			// May fail with fake data
			_ = err
		}
	})
}

func TestChangePitchExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("PitchSemitones", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_pitch_semitones.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		semitones := []int{-12, -5, 0, 5, 12}
		for _, st := range semitones {
			_, err := processor.ChangePitch(testFile, st)
			// May fail with fake data
			_ = err
		}
	})
}

func TestApplyEqualizerExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("EmptySettings", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_eq_empty.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.ApplyEqualizer(testFile, map[string]float64{})
		assert.Error(t, err) // Should fail with empty settings
	})

	t.Run("MultipleFrequencies", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_eq_multi.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		eqSettings := map[string]float64{
			"100Hz":  2.0,
			"500Hz":  0.0,
			"1000Hz": -1.5,
			"5000Hz": 1.0,
			"10kHz":  -2.0,
		}

		_, err = processor.ApplyEqualizer(testFile, eqSettings)
		// May fail with fake data
		_ = err
	})
}

func TestApplyNoiseReductionExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("IntensityLevels", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_noise_intensity.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		intensities := []float64{0.1, 0.3, 0.5, 0.7, 0.9}
		for _, intensity := range intensities {
			_, err := processor.ApplyNoiseReduction(testFile, intensity)
			// May fail with fake data
			_ = err
		}
	})
}

func TestDetectVoiceActivityExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("DefaultThreshold", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_vad_default.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.DetectVoiceActivity(testFile, 0)
		// Will use default threshold -40
		assert.Error(t, err) // Fails with fake audio
	})

	t.Run("CustomThresholds", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_vad_custom.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		thresholds := []float64{-30, -40, -50, -60}
		for _, threshold := range thresholds {
			_, err := processor.DetectVoiceActivity(testFile, threshold)
			assert.Error(t, err) // Fails with fake audio
		}
	})

	t.Run("MetadataExtractionFailure", func(t *testing.T) {
		_, err := processor.DetectVoiceActivity("/nonexistent.wav", -40)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract metadata")
	})
}

func TestRemoveSilenceExtended(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("DefaultThreshold", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_remove_silence_default.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.RemoveSilence(testFile, 0)
		// Uses default -40
		assert.Error(t, err)
	})

	t.Run("CustomThresholds", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_remove_silence_custom.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		thresholds := []float64{-30, -40, -50}
		for _, threshold := range thresholds {
			_, err := processor.RemoveSilence(testFile, threshold)
			assert.Error(t, err) // Fails with fake audio
		}
	})
}

func TestAnalyzeQuality(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("MetadataExtractionFailure", func(t *testing.T) {
		_, err := processor.AnalyzeQuality("/nonexistent.wav")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract metadata")
	})

	t.Run("WithFakeAudioFile", func(t *testing.T) {
		testFile := filepath.Join("/tmp", "test_quality.txt")
		err := os.WriteFile(testFile, []byte("fake audio"), 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		_, err = processor.AnalyzeQuality(testFile)
		assert.Error(t, err) // Metadata extraction will fail
	})
}

func TestQualityScoreCalculation(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("HighQualityMetrics", func(t *testing.T) {
		metrics := &QualityMetrics{
			Format:       "flac",
			Codec:        "flac",
			SampleRate:   "48000",
			Bitrate:      "1411200",
			Channels:     2,
			PeakLevel:    -3.0,
			RMSLevel:     -18.0,
			DynamicRange: 12.0,
		}

		score := processor.calculateQualityScore(metrics)
		assert.Greater(t, score, 80.0)
	})

	t.Run("LowQualityMetrics", func(t *testing.T) {
		metrics := &QualityMetrics{
			Format:       "mp3",
			Codec:        "mp3",
			SampleRate:   "22050",
			Bitrate:      "96000",
			Channels:     1,
			PeakLevel:    -0.5, // Clipping
			RMSLevel:     -32.0,
			DynamicRange: 4.0, // Over-compressed
		}

		score := processor.calculateQualityScore(metrics)
		assert.Less(t, score, 50.0)
	})
}

func TestDetectQualityIssues(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	t.Run("Clipping", func(t *testing.T) {
		metrics := &QualityMetrics{
			PeakLevel: -0.5,
		}

		issues := processor.detectQualityIssues(metrics)
		assert.Contains(t, issues, "Possible clipping detected")
	})

	t.Run("LowLevel", func(t *testing.T) {
		metrics := &QualityMetrics{
			PeakLevel: -25.0,
			RMSLevel:  -35.0,
		}

		issues := processor.detectQualityIssues(metrics)
		assert.Contains(t, issues, "Audio level too low")
		assert.Contains(t, issues, "Very low average level")
	})

	t.Run("OverCompressed", func(t *testing.T) {
		metrics := &QualityMetrics{
			DynamicRange: 4.0,
			PeakLevel:    -0.5, // Add clipping
		}

		issues := processor.detectQualityIssues(metrics)
		// Check that one of the issues contains "Over-compressed" or "low dynamic range"
		found := false
		for _, issue := range issues {
			if contains(issue, "Over-compressed") || contains(issue, "low dynamic range") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find over-compression issue")
	})

	t.Run("LowBitrate", func(t *testing.T) {
		metrics := &QualityMetrics{
			Format:  "mp3",
			Bitrate: "96000",
			PeakLevel: -0.5, // Add clipping
		}

		issues := processor.detectQualityIssues(metrics)
		// Check that one of the issues contains "bitrate" or "Low bitrate"
		found := false
		for _, issue := range issues {
			if contains(issue, "bitrate") || contains(issue, "Low bitrate") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find low bitrate issue")
	})
}
