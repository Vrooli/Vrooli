package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test HandleVAD
func TestHandleVAD_Comprehensive(t *testing.T) {
	handler := createTestHandler()

	t.Run("JSONRequest", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"audio_file": "/tmp/test.wav",
			"threshold":  -35.0,
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/audio/vad", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.HandleVAD(w, req)

		// Should fail with fake audio but shouldn't crash
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/audio/vad", bytes.NewBufferString("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.HandleVAD(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("OptionsRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/audio/vad", nil)
		w := httptest.NewRecorder()

		handler.HandleVAD(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Test HandleRemoveSilence
func TestHandleRemoveSilence_Comprehensive(t *testing.T) {
	handler := createTestHandler()

	t.Run("JSONRequest", func(t *testing.T) {
		// Create temp file
		tempFile, err := os.CreateTemp("/tmp", "test*.wav")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.WriteString("test audio data")
		tempFile.Close()

		requestBody := map[string]interface{}{
			"audio_file":    tempFile.Name(),
			"threshold":     -45.0,
			"output_format": "wav",
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/audio/remove-silence", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.HandleRemoveSilence(w, req)

		// Should fail with fake audio but shouldn't crash
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("OptionsRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/audio/remove-silence", nil)
		w := httptest.NewRecorder()

		handler.HandleRemoveSilence(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/audio/remove-silence", bytes.NewBufferString("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.HandleRemoveSilence(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Test getFloat helper
func TestGetFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		ok       bool
	}{
		{"float64", float64(10.5), 10.5, true},
		{"float32", float32(5.5), 5.5, true},
		{"int", 10, 10.0, true},
		{"int64", int64(20), 20.0, true},
		{"string", "15.5", 15.5, true},
		{"invalid string", "not-a-number", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := getFloat(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test sendJSON helper
func TestSendJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]interface{}{
		"message": "test",
		"status":  "ok",
	}

	sendJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "test", response["message"])
}

// Test mustMarshal helper
func TestMustMarshal(t *testing.T) {
	data := map[string]interface{}{
		"test": "data",
	}

	result := mustMarshal(data)
	assert.NotNil(t, result)
	assert.Contains(t, string(result), "test")
}

// Test getEQPreset helper
func TestGetEQPreset(t *testing.T) {
	environments := []string{"podcast", "meeting", "music", "unknown"}

	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			preset := getEQPreset(env)
			assert.NotNil(t, preset)
			assert.Greater(t, len(preset), 0)
		})
	}
}

// Test HandleEdit with multipart
func TestHandleEdit_MultipartForm(t *testing.T) {
	handler := createTestHandler()

	t.Run("MultipartTrimOperation", func(t *testing.T) {
		// Create temp file
		tempFile, err := os.CreateTemp("/tmp", "test*.wav")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.WriteString("test audio data")
		tempFile.Close()

		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file
		file, err := os.Open(tempFile.Name())
		require.NoError(t, err)
		defer file.Close()

		part, err := writer.CreateFormFile("audio", filepath.Base(tempFile.Name()))
		require.NoError(t, err)
		_, err = io.Copy(part, file)
		require.NoError(t, err)

		// Add operation
		writer.WriteField("operation", "trim")
		params := map[string]interface{}{
			"start_time": 0.0,
			"end_time":   10.0,
		}
		paramsJSON, _ := json.Marshal(params)
		writer.WriteField("parameters", string(paramsJSON))

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/audio/edit", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleEdit(w, req)

		// May fail with fake audio but shouldn't crash
		_ = w.Code
	})

	t.Run("MultipartVolumeOperation", func(t *testing.T) {
		tempFile, err := os.CreateTemp("/tmp", "test*.wav")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.WriteString("test audio data")
		tempFile.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		file, err := os.Open(tempFile.Name())
		require.NoError(t, err)
		defer file.Close()

		part, err := writer.CreateFormFile("audio", filepath.Base(tempFile.Name()))
		require.NoError(t, err)
		_, _ = io.Copy(part, file)

		writer.WriteField("operation", "volume")
		params := map[string]interface{}{
			"volume_factor": 1.5,
		}
		paramsJSON, _ := json.Marshal(params)
		writer.WriteField("parameters", string(paramsJSON))
		writer.WriteField("output_format", "mp3")

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/audio/edit", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleEdit(w, req)

		_ = w.Code
	})
}

// Test HandleEnhance with multipart
func TestHandleEnhance_MultipartForm(t *testing.T) {
	handler := createTestHandler()

	t.Run("MultipartNoiseReduction", func(t *testing.T) {
		tempFile, err := os.CreateTemp("/tmp", "test*.wav")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.WriteString("test audio data")
		tempFile.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		file, err := os.Open(tempFile.Name())
		require.NoError(t, err)
		defer file.Close()

		part, err := writer.CreateFormFile("audio", filepath.Base(tempFile.Name()))
		require.NoError(t, err)
		_, _ = io.Copy(part, file)

		writer.WriteField("environment", "podcast")

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/audio/enhance", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleEnhance(w, req)

		_ = w.Code
	})
}

// Test HandleAnalyze with multipart
func TestHandleAnalyze_MultipartForm(t *testing.T) {
	handler := createTestHandler()

	t.Run("MultipartQualityAnalysis", func(t *testing.T) {
		tempFile, err := os.CreateTemp("/tmp", "test*.wav")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.WriteString("test audio data")
		tempFile.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		file, err := os.Open(tempFile.Name())
		require.NoError(t, err)
		defer file.Close()

		part, err := writer.CreateFormFile("audio", filepath.Base(tempFile.Name()))
		require.NoError(t, err)
		_, _ = io.Copy(part, file)

		analysisTypes := []string{"quality", "spectral"}
		typesJSON, _ := json.Marshal(analysisTypes)
		writer.WriteField("analysis_types", string(typesJSON))

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/audio/analyze", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.HandleAnalyze(w, req)

		_ = w.Code
	})
}

// Test various edit operations
func TestHandleEdit_AllOperations(t *testing.T) {
	handler := createTestHandler()

	operations := []struct {
		name       string
		opType     string
		parameters map[string]interface{}
	}{
		{"fade", "fade", map[string]interface{}{"fade_in": 2.0, "fade_out": 2.0}},
		{"normalize", "normalize", map[string]interface{}{"target_level": -14.0}},
		{"speed", "speed", map[string]interface{}{"speed_factor": 1.5}},
		{"pitch", "pitch", map[string]interface{}{"semitones": 5.0}},
		{"equalizer", "equalizer", map[string]interface{}{
			"eq_settings": map[string]interface{}{
				"100Hz":  2.0,
				"1000Hz": -1.0,
			},
		}},
		{"noise_reduction", "noise_reduction", map[string]interface{}{"intensity": 0.6}},
	}

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			requestBody := map[string]interface{}{
				"audio_file": "/tmp/test.wav",
				"operations": []map[string]interface{}{
					{
						"type":       op.opType,
						"parameters": op.parameters,
					},
				},
			}

			body, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/v1/audio/edit", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.HandleEdit(w, req)

			// May fail with fake audio
			_ = w.Code
		})
	}
}

// Test edge cases
func TestHandleEdit_EdgeCases(t *testing.T) {
	handler := createTestHandler()

	t.Run("UnsupportedOperation", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"audio_file": "/tmp/test.wav",
			"operations": []map[string]interface{}{
				{
					"type":       "unsupported_operation",
					"parameters": map[string]interface{}{},
				},
			},
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/audio/edit", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.HandleEdit(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("InvalidAudioFileFormat", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"audio_file": 12345, // Invalid format
			"operations": []map[string]interface{}{
				{
					"type":       "trim",
					"parameters": map[string]interface{}{"start_time": 0.0, "end_time": 10.0},
				},
			},
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/audio/edit", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.HandleEdit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
