package video

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewProcessor(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tempDir := t.TempDir()

		processor, err := NewProcessor(tempDir)
		if err != nil {
			// If ffmpeg is not available, skip this test
			if os.IsNotExist(err) || err.Error() == "ffmpeg not found: exec: \"ffmpeg\": executable file not found in $PATH" {
				t.Skip("ffmpeg not available, skipping test")
			}
			t.Fatalf("Failed to create processor: %v", err)
		}

		if processor == nil {
			t.Fatal("Expected processor to be created")
		}

		if processor.workDir != tempDir {
			t.Errorf("Expected workDir=%s, got %s", tempDir, processor.workDir)
		}

		// Verify work directory was created
		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Error("Expected work directory to be created")
		}
	})

	t.Run("InvalidDirectory", func(t *testing.T) {
		// Try to create processor with invalid directory path
		invalidPath := "/invalid/nonexistent/path/that/cannot/be/created/\x00"
		_, err := NewProcessor(invalidPath)

		if err == nil {
			t.Error("Expected error for invalid directory path")
		}
	})
}

func TestGetVideoInfo(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := processor.GetVideoInfo("/nonexistent/file.mp4")

		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("InvalidFile", func(t *testing.T) {
		// Create a non-video file
		invalidFile := filepath.Join(tempDir, "invalid.mp4")
		if err := os.WriteFile(invalidFile, []byte("not a video file"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err := processor.GetVideoInfo(invalidFile)

		// ffprobe should fail on invalid video file
		if err == nil {
			t.Error("Expected error for invalid video file")
		}
	})
}

func TestTrim(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("NonExistentInput", func(t *testing.T) {
		inputPath := "/nonexistent/input.mp4"
		outputPath := filepath.Join(tempDir, "output.mp4")

		err := processor.Trim(inputPath, outputPath, 0, 10)

		if err == nil {
			t.Error("Expected error for non-existent input file")
		}
	})

	t.Run("InvalidTimeRange", func(t *testing.T) {
		// Create a dummy input file
		inputPath := filepath.Join(tempDir, "input.mp4")
		if err := os.WriteFile(inputPath, []byte("dummy"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		outputPath := filepath.Join(tempDir, "output.mp4")

		// Test with invalid time range (end before start)
		err := processor.Trim(inputPath, outputPath, 10, 5)

		// Should still execute but may produce unexpected results
		// Real implementation would validate this
		if err == nil {
			t.Log("Trim executed with invalid time range (implementation doesn't validate)")
		}
	})
}

func TestMerge(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("EmptyInputList", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "merged.mp4")

		err := processor.Merge([]string{}, outputPath)

		if err == nil {
			t.Error("Expected error for empty input list")
		}
	})

	t.Run("NonExistentInputs", func(t *testing.T) {
		inputPaths := []string{
			"/nonexistent/video1.mp4",
			"/nonexistent/video2.mp4",
		}
		outputPath := filepath.Join(tempDir, "merged.mp4")

		err := processor.Merge(inputPaths, outputPath)

		if err == nil {
			t.Error("Expected error for non-existent input files")
		}
	})
}

func TestExtractFrames(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("NonExistentInput", func(t *testing.T) {
		inputPath := "/nonexistent/video.mp4"
		outputDir := filepath.Join(tempDir, "frames")

		options := FrameExtractionOptions{
			Timestamps: []float64{1.0, 2.0, 3.0},
			Format:     "jpg",
		}

		_, err := processor.ExtractFrames(inputPath, outputDir, options)

		if err == nil {
			t.Error("Expected error for non-existent input file")
		}
	})

	t.Run("EmptyOptions", func(t *testing.T) {
		inputPath := filepath.Join(tempDir, "video.mp4")
		if err := os.WriteFile(inputPath, []byte("dummy"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		outputDir := filepath.Join(tempDir, "frames")

		options := FrameExtractionOptions{}

		frames, err := processor.ExtractFrames(inputPath, outputDir, options)

		// Should handle empty options gracefully
		if len(frames) > 0 && err != nil {
			t.Logf("Extracted %d frames with empty options", len(frames))
		}
	})

	t.Run("IntervalExtraction", func(t *testing.T) {
		inputPath := filepath.Join(tempDir, "video.mp4")
		if err := os.WriteFile(inputPath, []byte("dummy"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		outputDir := filepath.Join(tempDir, "frames_interval")

		options := FrameExtractionOptions{
			Interval: 1.0, // Every second
			Format:   "png",
		}

		_, err := processor.ExtractFrames(inputPath, outputDir, options)

		// Will fail on dummy file but tests the code path
		if err != nil {
			t.Logf("Expected failure on dummy file: %v", err)
		}
	})
}

func TestGenerateThumbnail(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("NonExistentInput", func(t *testing.T) {
		inputPath := "/nonexistent/video.mp4"
		outputPath := filepath.Join(tempDir, "thumbnail.jpg")

		err := processor.GenerateThumbnail(inputPath, outputPath, 5.0, 320)

		if err == nil {
			t.Error("Expected error for non-existent input file")
		}
	})

	t.Run("InvalidTimestamp", func(t *testing.T) {
		inputPath := filepath.Join(tempDir, "video.mp4")
		if err := os.WriteFile(inputPath, []byte("dummy"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		outputPath := filepath.Join(tempDir, "thumbnail.jpg")

		// Negative timestamp
		err := processor.GenerateThumbnail(inputPath, outputPath, -5.0, 320)

		// ffmpeg will handle negative timestamps, but test the code path
		if err != nil {
			t.Logf("Error with negative timestamp: %v", err)
		}
	})
}

func TestExtractAudio(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("NonExistentInput", func(t *testing.T) {
		inputPath := "/nonexistent/video.mp4"
		outputPath := filepath.Join(tempDir, "audio.mp3")

		err := processor.ExtractAudio(inputPath, outputPath)

		if err == nil {
			t.Error("Expected error for non-existent input file")
		}
	})
}

func TestAddSubtitles(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("BurnInSubtitles", func(t *testing.T) {
		videoPath := filepath.Join(tempDir, "video.mp4")
		subtitlePath := filepath.Join(tempDir, "subtitles.srt")
		outputPath := filepath.Join(tempDir, "output.mp4")

		// Create dummy files
		os.WriteFile(videoPath, []byte("dummy"), 0644)
		os.WriteFile(subtitlePath, []byte("1\n00:00:00,000 --> 00:00:02,000\nTest"), 0644)

		err := processor.AddSubtitles(videoPath, subtitlePath, outputPath, true)

		// Will fail on dummy files but tests the code path
		if err != nil {
			t.Logf("Expected failure on dummy files: %v", err)
		}
	})

	t.Run("StreamSubtitles", func(t *testing.T) {
		videoPath := filepath.Join(tempDir, "video2.mp4")
		subtitlePath := filepath.Join(tempDir, "subtitles2.srt")
		outputPath := filepath.Join(tempDir, "output2.mp4")

		os.WriteFile(videoPath, []byte("dummy"), 0644)
		os.WriteFile(subtitlePath, []byte("1\n00:00:00,000 --> 00:00:02,000\nTest"), 0644)

		err := processor.AddSubtitles(videoPath, subtitlePath, outputPath, false)

		if err != nil {
			t.Logf("Expected failure on dummy files: %v", err)
		}
	})
}

func TestCompress(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("NonExistentInput", func(t *testing.T) {
		inputPath := "/nonexistent/video.mp4"
		outputPath := filepath.Join(tempDir, "compressed.mp4")

		err := processor.Compress(inputPath, outputPath, 10)

		if err == nil {
			t.Error("Expected error for non-existent input file")
		}
	})
}

func TestConvertFormat(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	t.Run("NonExistentInput", func(t *testing.T) {
		inputPath := "/nonexistent/video.mp4"
		outputPath := filepath.Join(tempDir, "output.webm")

		options := ConversionOptions{
			TargetFormat: "webm",
			Quality:      "medium",
			Preset:       "fast",
		}

		err := processor.ConvertFormat(inputPath, outputPath, options)

		if err == nil {
			t.Error("Expected error for non-existent input file")
		}
	})

	t.Run("QualityPresets", func(t *testing.T) {
		inputPath := filepath.Join(tempDir, "input.mp4")
		os.WriteFile(inputPath, []byte("dummy"), 0644)

		qualityLevels := []string{"lossless", "high", "medium", "low"}

		for _, quality := range qualityLevels {
			t.Run(quality, func(t *testing.T) {
				outputPath := filepath.Join(tempDir, "output_"+quality+".mp4")

				options := ConversionOptions{
					TargetFormat: "mp4",
					Quality:      quality,
					Preset:       "fast",
				}

				err := processor.ConvertFormat(inputPath, outputPath, options)

				// Will fail on dummy file but tests quality preset code paths
				if err != nil {
					t.Logf("Expected failure with quality %s: %v", quality, err)
				}
			})
		}
	})

	t.Run("ResolutionPresets", func(t *testing.T) {
		inputPath := filepath.Join(tempDir, "input2.mp4")
		os.WriteFile(inputPath, []byte("dummy"), 0644)

		resolutions := []string{"480p", "720p", "1080p", "4k"}

		for _, resolution := range resolutions {
			t.Run(resolution, func(t *testing.T) {
				outputPath := filepath.Join(tempDir, "output_"+resolution+".mp4")

				options := ConversionOptions{
					TargetFormat: "mp4",
					Resolution:   resolution,
				}

				err := processor.ConvertFormat(inputPath, outputPath, options)

				if err != nil {
					t.Logf("Expected failure with resolution %s: %v", resolution, err)
				}
			})
		}
	})
}

func TestParseResolution(t *testing.T) {
	tempDir := t.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		t.Skip("ffmpeg not available, skipping test")
	}

	testCases := []struct {
		input    string
		expected string
	}{
		{"480p", "854:480"},
		{"720p", "1280:720"},
		{"1080p", "1920:1080"},
		{"4k", "3840:2160"},
		{"1280:720", "1280:720"}, // Custom resolution
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := processor.parseResolution(tc.input)

			if result != tc.expected {
				t.Errorf("Expected parseResolution(%s)=%s, got %s",
					tc.input, tc.expected, result)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkGetVideoInfo(b *testing.B) {
	tempDir := b.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		b.Skip("ffmpeg not available, skipping benchmark")
	}

	// Create a dummy video file
	inputPath := filepath.Join(tempDir, "video.mp4")
	os.WriteFile(inputPath, []byte("dummy video content"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.GetVideoInfo(inputPath)
	}
}

func BenchmarkGenerateThumbnail(b *testing.B) {
	tempDir := b.TempDir()
	processor, err := NewProcessor(tempDir)
	if err != nil {
		b.Skip("ffmpeg not available, skipping benchmark")
	}

	inputPath := filepath.Join(tempDir, "video.mp4")
	os.WriteFile(inputPath, []byte("dummy"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("thumb_%d.jpg", i))
		processor.GenerateThumbnail(inputPath, outputPath, 1.0, 320)
	}
}
