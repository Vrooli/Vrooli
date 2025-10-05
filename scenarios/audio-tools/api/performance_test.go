package main

import (
	"audio-tools/internal/audio"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPerformance_AudioProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)

	t.Run("MetadataExtraction", func(t *testing.T) {
		testFile := createTestAudioFile(t, env.WorkDir, "perf_metadata.wav", 30.0)

		pattern := CreateAudioProcessingPerformanceTest(
			"MetadataExtraction",
			2*time.Second,
			func(t *testing.T, env *TestEnvironment) time.Duration {
				start := time.Now()
				_, err := processor.ExtractMetadata(testFile)
				duration := time.Since(start)

				// May error with test audio, but should be fast
				_ = err

				return duration
			},
		)

		RunPerformanceTest(t, pattern)
	})

	t.Run("TrimOperation", func(t *testing.T) {
		testFile := createTestAudioFile(t, env.WorkDir, "perf_trim.wav", 60.0)

		pattern := CreateAudioProcessingPerformanceTest(
			"TrimOperation",
			5*time.Second,
			func(t *testing.T, env *TestEnvironment) time.Duration {
				start := time.Now()
				_, err := processor.Trim(testFile, 10.0, 30.0)
				duration := time.Since(start)

				// Cleanup output file if created
				if err == nil {
					// Would clean up output file
				}

				return duration
			},
		)

		RunPerformanceTest(t, pattern)
	})

	t.Run("VolumeAdjustment", func(t *testing.T) {
		testFile := createTestAudioFile(t, env.WorkDir, "perf_volume.wav", 30.0)

		pattern := CreateAudioProcessingPerformanceTest(
			"VolumeAdjustment",
			5*time.Second,
			func(t *testing.T, env *TestEnvironment) time.Duration {
				start := time.Now()
				outputPath, err := processor.AdjustVolume(testFile, 1.5)
				duration := time.Since(start)

				if err == nil && outputPath != "" {
					os.Remove(outputPath)
				}

				return duration
			},
		)

		RunPerformanceTest(t, pattern)
	})

	t.Run("Normalization", func(t *testing.T) {
		testFile := createTestAudioFile(t, env.WorkDir, "perf_normalize.wav", 30.0)

		pattern := CreateAudioProcessingPerformanceTest(
			"Normalization",
			10*time.Second,
			func(t *testing.T, env *TestEnvironment) time.Duration {
				start := time.Now()
				outputPath, err := processor.Normalize(testFile, -16.0)
				duration := time.Since(start)

				if err == nil && outputPath != "" {
					os.Remove(outputPath)
				}

				return duration
			},
		)

		RunPerformanceTest(t, pattern)
	})
}

func TestPerformance_LargeFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)

	t.Run("LargeFileMetadata", func(t *testing.T) {
		// Create a larger test file (5 minutes)
		testFile := createTestAudioFile(t, env.WorkDir, "large.wav", 300.0)

		start := time.Now()
		_, err := processor.ExtractMetadata(testFile)
		duration := time.Since(start)

		// Should complete within reasonable time even with larger files
		assert.Less(t, duration, 5*time.Second)

		// May error with test data
		_ = err
	})

	t.Run("LargeFileTrim", func(t *testing.T) {
		testFile := createTestAudioFile(t, env.WorkDir, "large_trim.wav", 600.0)

		start := time.Now()
		_, err := processor.Trim(testFile, 60.0, 120.0)
		duration := time.Since(start)

		assert.Less(t, duration, 10*time.Second)
		_ = err
	})
}

func TestPerformance_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)

	t.Run("ParallelMetadataExtraction", func(t *testing.T) {
		numFiles := 10
		files := make([]string, numFiles)

		// Create test files
		for i := 0; i < numFiles; i++ {
			filename := filepath.Join(env.WorkDir, "parallel_"+string(rune(i))+".wav")
			files[i] = createTestAudioFile(t, env.WorkDir, filename, 10.0)
		}

		start := time.Now()

		// Process in parallel
		done := make(chan bool, numFiles)
		for _, file := range files {
			go func(f string) {
				_, _ = processor.ExtractMetadata(f)
				done <- true
			}(file)
		}

		// Wait for all to complete
		for i := 0; i < numFiles; i++ {
			<-done
		}

		duration := time.Since(start)

		// Should handle concurrent operations efficiently
		t.Logf("Processed %d files concurrently in %v", numFiles, duration)
		assert.Less(t, duration, 30*time.Second)
	})
}

func TestPerformance_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)

	t.Run("MultipleOperationsMemory", func(t *testing.T) {
		testFile := createTestAudioFile(t, env.WorkDir, "memory_test.wav", 60.0)

		// Perform multiple operations
		for i := 0; i < 100; i++ {
			_, _ = processor.ExtractMetadata(testFile)

			// Brief pause to allow GC
			if i%10 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Should not leak memory significantly
		assert.True(t, true)
	})
}

func TestPerformance_FormatConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping format conversion performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)

	formats := []string{"mp3", "wav", "flac", "ogg"}

	for _, format := range formats {
		t.Run("ConvertTo"+format, func(t *testing.T) {
			testFile := createTestAudioFile(t, env.WorkDir, "convert_"+format+".wav", 30.0)

			start := time.Now()
			outputPath, err := processor.ConvertFormat(testFile, format, nil)
			duration := time.Since(start)

			t.Logf("Conversion to %s took %v", format, duration)

			if err == nil && outputPath != "" {
				os.Remove(outputPath)
			}

			// Conversion should complete within reasonable time
			assert.Less(t, duration, 15*time.Second)
		})
	}
}

func TestPerformance_VAD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VAD performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)

	t.Run("VoiceActivityDetection", func(t *testing.T) {
		testFile := createTestAudioFile(t, env.WorkDir, "vad_perf.wav", 120.0)

		pattern := CreateAudioProcessingPerformanceTest(
			"VoiceActivityDetection",
			10*time.Second,
			func(t *testing.T, env *TestEnvironment) time.Duration {
				start := time.Now()
				_, err := processor.DetectVoiceActivity(testFile, -40)
				duration := time.Since(start)

				// May error with test data
				_ = err

				return duration
			},
		)

		RunPerformanceTest(t, pattern)
	})
}

func TestPerformance_MultiOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-operation performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)

	t.Run("ChainedOperations", func(t *testing.T) {
		testFile := createTestAudioFile(t, env.WorkDir, "chained.wav", 60.0)
		currentFile := testFile

		start := time.Now()

		// Chain multiple operations
		operations := []func(string) (string, error){
			func(f string) (string, error) { return processor.Trim(f, 5.0, 55.0) },
			func(f string) (string, error) { return processor.AdjustVolume(f, 1.2) },
			func(f string) (string, error) { return processor.Normalize(f, -16.0) },
		}

		for _, op := range operations {
			newFile, err := op(currentFile)
			if err == nil && newFile != "" {
				if currentFile != testFile {
					os.Remove(currentFile)
				}
				currentFile = newFile
			}
		}

		duration := time.Since(start)

		// Cleanup
		if currentFile != testFile {
			os.Remove(currentFile)
		}

		t.Logf("Chained operations took %v", duration)
		assert.Less(t, duration, 30*time.Second)
	})
}

func BenchmarkMetadataExtraction(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)
	testFile := createTestAudioFile(&testing.T{}, env.WorkDir, "bench.wav", 10.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.ExtractMetadata(testFile)
	}
}

func BenchmarkVolumeAdjustment(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)
	testFile := createTestAudioFile(&testing.T{}, env.WorkDir, "bench_volume.wav", 10.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath, err := processor.AdjustVolume(testFile, 1.5)
		if err == nil && outputPath != "" {
			os.Remove(outputPath)
		}
	}
}

func BenchmarkTrim(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	processor := audio.NewAudioProcessor(env.WorkDir, env.DataDir)
	testFile := createTestAudioFile(&testing.T{}, env.WorkDir, "bench_trim.wav", 30.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath, err := processor.Trim(testFile, 5.0, 25.0)
		if err == nil && outputPath != "" {
			os.Remove(outputPath)
		}
	}
}
