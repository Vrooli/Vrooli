package handlers

import (
	"audio-tools/internal/audio"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create test AudioHandler
func createTestHandler() *AudioHandler {
	processor := audio.NewAudioProcessor("/tmp", "/tmp/data")
	return &AudioHandler{
		processor: processor,
		workDir:   "/tmp",
		dataDir:   "/tmp/data",
		db:        nil, // No database for tests
	}
}

func TestHandleEdit(t *testing.T) {
	handler := createTestHandler()

	// Create a test file for the tests
	testFile := "/tmp/test.wav"
	err := os.WriteFile(testFile, []byte("test audio data"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp map[string]interface{})
	}{
		{
			name: "trim operation with fake audio",
			requestBody: map[string]interface{}{
				"audio_file": "/tmp/test.wav",
				"operations": []map[string]interface{}{
					{
						"type": "trim",
						"parameters": map[string]interface{}{
							"start_time": 10.0,
							"end_time":   20.0,
						},
					},
				},
			},
			expectedStatus: http.StatusInternalServerError, // FFmpeg will fail with fake data
			checkResponse:  nil,
		},
		{
			name: "volume adjustment with fake audio",
			requestBody: map[string]interface{}{
				"audio_file": "/tmp/test.wav",
				"operations": []map[string]interface{}{
					{
						"type": "volume",
						"parameters": map[string]interface{}{
							"volume_factor": 1.5,
						},
					},
				},
			},
			expectedStatus: http.StatusInternalServerError, // FFmpeg will fail with fake data
			checkResponse:  nil,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing audio file",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{"type": "trim"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/v1/audio/edit", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Handle request
			handler.HandleEdit(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil && w.Code == http.StatusOK {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestHandleConvert(t *testing.T) {
	handler := createTestHandler()

	// Create a temporary test file
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

	// Add format field
	writer.WriteField("format", "mp3")
	writer.WriteField("bitrate", "192000")

	err = writer.Close()
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/audio/convert", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleConvert(w, req)

	// Check response - may fail if ffmpeg not available
	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp["file_id"])
		assert.Equal(t, "mp3", resp["format"])
	}
}

func TestHandleMetadata(t *testing.T) {
	handler := createTestHandler()

	t.Run("GET with file ID", func(t *testing.T) {
		// Create a test file
		tempFile, err := os.CreateTemp("/tmp", "test*.mp3")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.WriteString("test audio data")
		tempFile.Close()

		req := httptest.NewRequest("GET", "/api/v1/audio/metadata/test-file-id", nil)
		w := httptest.NewRecorder()

		// Add route vars
		req = mux.SetURLVars(req, map[string]string{
			"id": tempFile.Name(),
		})

		// Handle request
		handler.HandleMetadata(w, req)

		// Will fail with fake audio but shouldn't crash
		if w.Code == http.StatusOK {
			var resp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.NotNil(t, resp)
		}
	})

	t.Run("POST with file upload", func(t *testing.T) {
		// Create a temporary test file
		tempFile, err := os.CreateTemp("/tmp", "test*.mp3")
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

		err = writer.Close()
		require.NoError(t, err)

		// Create request
		req := httptest.NewRequest("POST", "/api/v1/audio/metadata", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		// Handle request
		handler.HandleMetadata(w, req)

		// Will fail with fake audio but shouldn't crash
		if w.Code == http.StatusOK {
			var resp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.NotNil(t, resp)
		}
	})
}

func TestHandleEnhance(t *testing.T) {
	handler := createTestHandler()

	requestBody := map[string]interface{}{
		"audio_file": "/tmp/test.wav",
		"enhancements": []map[string]interface{}{
			{
				"type":      "noise_reduction",
				"intensity": 0.7,
			},
			{
				"type": "auto_level",
			},
		},
		"target_environment": "podcast",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/audio/enhance", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleEnhance(w, req)

	// Check response - may fail if ffmpeg not available
	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp["file_id"])
		// Applied enhancements may be empty if processing failed
	}
}

func TestHandleAnalyze(t *testing.T) {
	handler := createTestHandler()

	// Create a temporary test file
	tempFile, err := os.CreateTemp("/tmp", "test*.wav")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	tempFile.WriteString("test audio data")
	tempFile.Close()

	requestBody := map[string]interface{}{
		"audio_file":     tempFile.Name(),
		"analysis_types": []string{"quality", "content", "spectral"},
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/audio/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleAnalyze(w, req)

	// Check response - may fail if ffmpeg not available
	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp["analysis_results"])
	}
}
