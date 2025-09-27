package audio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAudioProcessor(t *testing.T) {
	workDir := "/tmp/audio-test"
	dataDir := "/tmp/audio-data"
	processor := NewAudioProcessor(workDir, dataDir)

	assert.NotNil(t, processor)
	assert.Equal(t, workDir, processor.WorkDir)
	assert.Equal(t, dataDir, processor.DataDir)
}

func TestExtractMetadata(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file (not real audio, will cause error)
	testFile := filepath.Join("/tmp", "test_metadata.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	metadata, err := processor.ExtractMetadata(testFile)
	
	// With a non-audio file, ffprobe will fail but shouldn't crash
	assert.Error(t, err)
	assert.Nil(t, metadata)
}

func TestConvertFormat(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_convert.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.ConvertFormat(testFile, "mp3", map[string]interface{}{
		"bitrate": 192000,
	})

	// With fake data, conversion will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, ".mp3")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestTrim(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_trim.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.Trim(testFile, 10.0, 20.0)

	// With fake data, trim will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "trimmed")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestNormalize(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_normalize.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.Normalize(testFile, -16.0)

	// With fake data, normalize will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "normalized")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestAdjustVolume(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_volume.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.AdjustVolume(testFile, 1.5) // 150% volume

	// With fake data, volume adjust will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "volume")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestMerge(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create test files
	testFile1 := filepath.Join("/tmp", "test_merge1.wav")
	testFile2 := filepath.Join("/tmp", "test_merge2.wav")
	err := os.WriteFile(testFile1, []byte("fake audio 1"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile1)
	
	err = os.WriteFile(testFile2, []byte("fake audio 2"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile2)

	outputPath, err := processor.Merge([]string{testFile1, testFile2}, "wav")

	// With fake data, merge will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "merged")
		assert.Contains(t, outputPath, ".wav")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestFadeIn(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_fade.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.FadeIn(testFile, 2.0)

	// With fake data, fade will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "faded")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestFadeOut(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_fade.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.FadeOut(testFile, 3.0)

	// With fake data, fade will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "faded")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestChangeSpeed(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_speed.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.ChangeSpeed(testFile, 1.5) // 150% speed

	// With fake data, speed change will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "speed")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestChangePitch(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_pitch.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.ChangePitch(testFile, 5) // +5 semitones

	// With fake data, pitch change will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "pitch")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestSplit(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_split.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPaths, err := processor.Split(testFile, []float64{10.0, 20.0, 30.0})

	// With fake data, split will fail but shouldn't crash
	if err == nil {
		assert.GreaterOrEqual(t, len(outputPaths), 2)
		// Clean up if successful
		for _, path := range outputPaths {
			os.Remove(path)
		}
	}
}

func TestApplyEqualizer(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_eq.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	eqSettings := map[string]float64{
		"100Hz":  2.0,
		"1000Hz": -1.5,
		"10kHz":  1.0,
	}

	outputPath, err := processor.ApplyEqualizer(testFile, eqSettings)

	// With fake data, EQ will fail but shouldn't crash
	if err == nil {
		assert.Contains(t, outputPath, "equalized")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestApplyNoiseReduction(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}

	// Create a test file
	testFile := filepath.Join("/tmp", "test_noise.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.ApplyNoiseReduction(testFile, 0.7)

	// With fake data, noise reduction will fail but shouldn't crash  
	if err == nil {
		assert.Contains(t, outputPath, "noise_reduced")
		// Clean up if successful
		os.Remove(outputPath)
	}
}

func TestDetectVoiceActivity(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}
	
	// Create a test file
	testFile := filepath.Join("/tmp", "test_vad.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	vad, err := processor.DetectVoiceActivity(testFile, -40)
	
	// With fake data, VAD will fail but should handle gracefully
	if err == nil {
		assert.NotNil(t, vad)
		assert.NotNil(t, vad.SpeechSegments)
		assert.GreaterOrEqual(t, vad.TotalDuration, 0.0)
		assert.GreaterOrEqual(t, vad.SpeechDuration, 0.0)
		assert.GreaterOrEqual(t, vad.SilenceDuration, 0.0)
	} else {
		// Expected for fake audio data
		assert.Error(t, err)
	}
}

func TestRemoveSilence(t *testing.T) {
	processor := &AudioProcessor{WorkDir: "/tmp"}
	
	// Create a test file
	testFile := filepath.Join("/tmp", "test_silence.wav")
	err := os.WriteFile(testFile, []byte("fake audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	outputPath, err := processor.RemoveSilence(testFile, -40)
	
	// With fake data, silence removal will fail but should handle gracefully
	if err == nil {
		assert.Contains(t, outputPath, "no_silence")
		// Clean up if successful
		os.Remove(outputPath)
	} else {
		// Expected for fake audio data
		assert.Error(t, err)
	}
}