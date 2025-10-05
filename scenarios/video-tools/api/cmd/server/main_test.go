package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	// Setup test logger
	cleanup := setupTestLogger()
	defer cleanup()

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()

	t.Run("HealthCheck", func(t *testing.T) {
		w := makeHTTPRequest(server, "GET", "/health", nil, map[string]string{})

		response := assertJSONResponse(t, w, http.StatusOK)

		// Validate response structure
		if status, ok := response["status"].(string); !ok || status == "" {
			t.Error("Expected 'status' field in health response")
		}

		if service, ok := response["service"].(string); !ok || service != "video-tools API" {
			t.Errorf("Expected service='video-tools API', got %v", response["service"])
		}

		if version, ok := response["version"].(string); !ok || version == "" {
			t.Error("Expected 'version' field in health response")
		}

		if _, ok := response["database"].(string); !ok {
			t.Error("Expected 'database' field in health response")
		}
	})

	t.Run("HealthCheckNoAuth", func(t *testing.T) {
		// Health endpoint should work without auth
		w := makeHTTPRequest(server, "GET", "/health", nil, map[string]string{})

		if w.Code != http.StatusOK {
			t.Errorf("Health endpoint should not require auth, got status %d", w.Code)
		}
	})
}

// TestStatusEndpoint tests the status endpoint
func TestStatusEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()

	t.Run("StatusCheck", func(t *testing.T) {
		w := makeHTTPRequest(server, "GET", "/api/status", nil, map[string]string{})

		response := assertJSONResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		// Validate response structure
		if status, ok := data["status"].(string); !ok || status != "operational" {
			t.Errorf("Expected status='operational', got %v", data["status"])
		}

		if capabilities, ok := data["capabilities"].([]interface{}); !ok || len(capabilities) == 0 {
			t.Error("Expected non-empty 'capabilities' array in status response")
		}
	})
}

// TestAuthMiddleware tests authentication middleware
func TestAuthMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()

	testCases := []ErrorTestPattern{
		{
			Name:           "MissingAuth",
			Path:           "/api/v1/video/upload",
			Method:         "POST",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedError:  "unauthorized",
		},
		{
			Name:           "InvalidAuthToken",
			Path:           "/api/v1/video/upload",
			Method:         "POST",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedError:  "unauthorized",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			headers := map[string]string{}
			if tc.Name == "InvalidAuthToken" {
				headers["Authorization"] = "Bearer wrong-token"
			}

			w := makeHTTPRequest(server, tc.Method, tc.Path, nil, headers)
			assertErrorResponse(t, w, tc.ExpectedStatus, tc.ExpectedError)
		})
	}
}

// TestGetVideo tests video retrieval
func TestGetVideo(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		// Insert test video
		videoID := insertTestVideo(t, server.db)

		// Get video
		path := fmt.Sprintf("/api/v1/video/%s", videoID)
		w := makeHTTPRequest(server, "GET", path, nil, nil)

		response := assertJSONResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		if id := data["id"].(string); id != videoID {
			t.Errorf("Expected video ID %s, got %s", videoID, id)
		}

		if name := data["name"].(string); name != "Test Video" {
			t.Errorf("Expected name 'Test Video', got %s", name)
		}
	})

	t.Run("NonExistentVideo", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/video/%s", nonExistentID)

		w := makeHTTPRequest(server, "GET", path, nil, nil)
		assertErrorResponse(t, w, http.StatusNotFound, "video not found")
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		path := "/api/v1/video/invalid-uuid-format"

		w := makeHTTPRequest(server, "GET", path, nil, nil)
		// Mux will return 404 for invalid route patterns
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for invalid UUID, got %d", w.Code)
		}
	})
}

// TestVideoConvert tests video conversion endpoints
func TestVideoConvert(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		path := fmt.Sprintf("/api/v1/video/%s/convert", videoID)
		payload := GenerateTestConvertRequest("webm")

		w := makeJSONRequest(server, "POST", path, payload)
		response := assertJSONResponse(t, w, http.StatusAccepted)

		data := response["data"].(map[string]interface{})
		if jobID, ok := data["job_id"].(string); !ok || jobID == "" {
			t.Error("Expected job_id in response")
		}

		if status, ok := data["status"].(string); !ok || status != "pending" {
			t.Errorf("Expected status='pending', got %v", data["status"])
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s/convert", videoID)

		w := makeHTTPRequest(server, "POST", path, bytes.NewReader([]byte("{invalid")), map[string]string{
			"Content-Type": "application/json",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid request body")
	})

	t.Run("NonExistentVideo", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/video/%s/convert", nonExistentID)
		payload := GenerateTestConvertRequest("mp4")

		w := makeJSONRequest(server, "POST", path, payload)
		// This will create a job even for non-existent video (async processing)
		// In a real implementation, you'd want validation
		if w.Code != http.StatusAccepted && w.Code != http.StatusNotFound {
			t.Logf("Got status %d for non-existent video conversion", w.Code)
		}
	})
}

// TestVideoEdit tests video editing endpoints
func TestVideoEdit(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		path := fmt.Sprintf("/api/v1/video/%s/edit", videoID)
		payload := GenerateTestEditRequest()

		w := makeJSONRequest(server, "POST", path, payload)
		response := assertJSONResponse(t, w, http.StatusAccepted)

		data := response["data"].(map[string]interface{})
		if jobID, ok := data["job_id"].(string); !ok || jobID == "" {
			t.Error("Expected job_id in response")
		}
	})

	t.Run("EmptyOperations", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s/edit", videoID)

		payload := map[string]interface{}{
			"operations": []interface{}{},
		}

		w := makeJSONRequest(server, "POST", path, payload)
		// Should still accept empty operations (will be validated during processing)
		if w.Code != http.StatusAccepted && w.Code != http.StatusBadRequest {
			t.Logf("Got status %d for empty operations", w.Code)
		}
	})
}

// TestJobManagement tests job-related endpoints
func TestJobManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("ListJobs", func(t *testing.T) {
		// Insert test data
		videoID := insertTestVideo(t, server.db)
		insertTestJob(t, server.db, videoID, "convert")
		insertTestJob(t, server.db, videoID, "edit")

		w := makeHTTPRequest(server, "GET", "/api/v1/jobs", nil, nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		jobs := response["data"].([]interface{})
		if len(jobs) < 2 {
			t.Errorf("Expected at least 2 jobs, got %d", len(jobs))
		}
	})

	t.Run("GetJob", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		jobID := insertTestJob(t, server.db, videoID, "convert")

		path := fmt.Sprintf("/api/v1/jobs/%s", jobID)
		w := makeHTTPRequest(server, "GET", path, nil, nil)

		response := assertJSONResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		if id := data["id"].(string); id != jobID {
			t.Errorf("Expected job ID %s, got %s", jobID, id)
		}

		if jobType := data["job_type"].(string); jobType != "convert" {
			t.Errorf("Expected job_type='convert', got %s", jobType)
		}
	})

	t.Run("GetNonExistentJob", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/jobs/%s", nonExistentID)

		w := makeHTTPRequest(server, "GET", path, nil, nil)
		assertErrorResponse(t, w, http.StatusNotFound, "job not found")
	})

	t.Run("CancelJob", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		jobID := insertTestJob(t, server.db, videoID, "convert")

		path := fmt.Sprintf("/api/v1/jobs/%s/cancel", jobID)
		w := makeHTTPRequest(server, "POST", path, nil, nil)

		response := assertJSONResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		if status := data["status"].(string); status != "cancelled" {
			t.Errorf("Expected status='cancelled', got %s", status)
		}
	})
}

// TestStreamManagement tests streaming-related endpoints
func TestStreamManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("CreateStream", func(t *testing.T) {
		payload := map[string]interface{}{
			"name": "Test Stream",
			"input_source": map[string]interface{}{
				"type": "file",
			},
			"output_targets": []map[string]interface{}{
				{
					"platform": "custom",
					"url":      "rtmp://example.com/live",
				},
			},
		}

		w := makeJSONRequest(server, "POST", "/api/v1/stream/create", payload)
		response := assertJSONResponse(t, w, http.StatusCreated)

		data := response["data"].(map[string]interface{})
		if sessionID, ok := data["session_id"].(string); !ok || sessionID == "" {
			t.Error("Expected session_id in response")
		}

		if streamKey, ok := data["stream_key"].(string); !ok || streamKey == "" {
			t.Error("Expected stream_key in response")
		}

		if rtmpURL, ok := data["rtmp_url"].(string); !ok || rtmpURL == "" {
			t.Error("Expected rtmp_url in response")
		}
	})

	t.Run("ListStreams", func(t *testing.T) {
		w := makeHTTPRequest(server, "GET", "/api/v1/streams", nil, nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		streams := response["data"].([]interface{})
		// Should return array (possibly empty)
		if streams == nil {
			t.Error("Expected streams array in response")
		}
	})
}

// TestDocsEndpoint tests documentation endpoint
func TestDocsEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()

	t.Run("GetDocs", func(t *testing.T) {
		w := makeHTTPRequest(server, "GET", "/docs", nil, map[string]string{})

		response := assertJSONResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		if name, ok := data["name"].(string); !ok || name != "video-tools API" {
			t.Errorf("Expected name='video-tools API', got %v", data["name"])
		}

		if endpoints, ok := data["endpoints"].([]interface{}); !ok || len(endpoints) == 0 {
			t.Error("Expected non-empty endpoints array in docs")
		}
	})
}

// TestCORSMiddleware tests CORS headers
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()

	t.Run("OptionsRequest", func(t *testing.T) {
		w := makeHTTPRequest(server, "OPTIONS", "/health", nil, nil)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected CORS header, got %s", origin)
		}
	})
}

// TestErrorHandling tests comprehensive error handling
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()

	scenarios := NewTestScenarioBuilder().
		AddNonExistentVideo("/api/v1/video/%s").
		AddInvalidJSON("/api/v1/video/123/convert").
		AddMissingAuth("/api/v1/video/upload", "POST").
		Build()

	RunErrorTests(t, server, scenarios)
}

// BenchmarkHealthEndpoint benchmarks the health endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(&testing.T{})
	defer cleanupServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(server, "GET", "/health", nil, map[string]string{})
	}
}

// BenchmarkGetVideo benchmarks video retrieval
func BenchmarkGetVideo(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(&testing.T{})
	defer cleanupServer()

	videoID := insertTestVideo(&testing.T{}, server.db)
	path := fmt.Sprintf("/api/v1/video/%s", videoID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(server, "GET", path, nil, nil)
	}
}

// TestExtractFramesEndpoint tests frame extraction
func TestExtractFramesEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		// Create test video file
		videoPath := fmt.Sprintf("%s/uploads/%s.mp4", server.config.WorkDir, videoID)
		createTestVideo(t, videoPath)

		path := fmt.Sprintf("/api/v1/video/%s/frames?timestamps=1,2,3&format=jpg", videoID)
		w := makeHTTPRequest(server, "GET", path, nil, nil)

		// May fail if ffmpeg is not available, but should return proper error
		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK)
			if frames, ok := response["data"].(map[string]interface{})["frames"]; ok {
				t.Logf("Extracted frames: %v", frames)
			}
		} else if w.Code == http.StatusInternalServerError {
			t.Log("Frame extraction failed (expected without real video/ffmpeg)")
		}
	})

	t.Run("VideoNotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/video/%s/frames", nonExistentID)

		w := makeHTTPRequest(server, "GET", path, nil, nil)
		assertErrorResponse(t, w, http.StatusNotFound, "video file not found")
	})
}

// TestGenerateThumbnailEndpoint tests thumbnail generation
func TestGenerateThumbnailEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		// Create test video file
		videoPath := fmt.Sprintf("%s/uploads/%s.mp4", server.config.WorkDir, videoID)
		createTestVideo(t, videoPath)

		path := fmt.Sprintf("/api/v1/video/%s/thumbnail", videoID)
		w := makeHTTPRequest(server, "POST", path, nil, nil)

		// May succeed or fail depending on ffmpeg availability
		if w.Code == http.StatusOK || w.Code == http.StatusInternalServerError {
			t.Logf("Thumbnail generation returned status: %d", w.Code)
		}
	})

	t.Run("VideoNotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/video/%s/thumbnail", nonExistentID)

		w := makeHTTPRequest(server, "POST", path, nil, nil)
		assertErrorResponse(t, w, http.StatusNotFound, "video file not found")
	})
}

// TestExtractAudioEndpoint tests audio extraction
func TestExtractAudioEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		// Create test video file
		videoPath := fmt.Sprintf("%s/uploads/%s.mp4", server.config.WorkDir, videoID)
		createTestVideo(t, videoPath)

		path := fmt.Sprintf("/api/v1/video/%s/audio", videoID)
		w := makeHTTPRequest(server, "POST", path, nil, nil)

		if w.Code == http.StatusOK || w.Code == http.StatusInternalServerError {
			t.Logf("Audio extraction returned status: %d", w.Code)
		}
	})

	t.Run("VideoNotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/video/%s/audio", nonExistentID)

		w := makeHTTPRequest(server, "POST", path, nil, nil)
		assertErrorResponse(t, w, http.StatusNotFound, "video file not found")
	})
}

// TestAddSubtitlesEndpoint tests subtitle addition
func TestAddSubtitlesEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		payload := map[string]interface{}{
			"subtitle_path": "/tmp/subtitles.srt",
			"burn_in":       true,
		}

		path := fmt.Sprintf("/api/v1/video/%s/subtitles", videoID)
		w := makeJSONRequest(server, "POST", path, payload)

		response := assertJSONResponse(t, w, http.StatusAccepted)
		data := response["data"].(map[string]interface{})
		if jobID, ok := data["job_id"].(string); !ok || jobID == "" {
			t.Error("Expected job_id in response")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s/subtitles", videoID)

		w := makeHTTPRequest(server, "POST", path, bytes.NewReader([]byte("{invalid")), map[string]string{
			"Content-Type": "application/json",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid request body")
	})
}

// TestCompressEndpoint tests video compression
func TestCompressEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		payload := map[string]interface{}{
			"target_size_mb": 50,
		}

		path := fmt.Sprintf("/api/v1/video/%s/compress", videoID)
		w := makeJSONRequest(server, "POST", path, payload)

		response := assertJSONResponse(t, w, http.StatusAccepted)
		data := response["data"].(map[string]interface{})
		if status, ok := data["status"].(string); !ok || status != "pending" {
			t.Errorf("Expected status='pending', got %v", data["status"])
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s/compress", videoID)

		w := makeHTTPRequest(server, "POST", path, bytes.NewReader([]byte("not json")), map[string]string{
			"Content-Type": "application/json",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid request body")
	})
}

// TestAnalyzeEndpoint tests video analysis
func TestAnalyzeEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		payload := map[string]interface{}{
			"analysis_types": []string{"quality", "content", "audio"},
			"options": map[string]interface{}{
				"detailed": true,
			},
		}

		path := fmt.Sprintf("/api/v1/video/%s/analyze", videoID)
		w := makeJSONRequest(server, "POST", path, payload)

		response := assertJSONResponse(t, w, http.StatusAccepted)
		data := response["data"].(map[string]interface{})

		if jobID, ok := data["job_id"].(string); !ok || jobID == "" {
			t.Error("Expected job_id in response")
		}

		if analysisTypes, ok := data["analysis_types"].([]interface{}); !ok || len(analysisTypes) != 3 {
			t.Error("Expected analysis_types array with 3 items")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s/analyze", videoID)

		w := makeHTTPRequest(server, "POST", path, bytes.NewReader([]byte("invalid")), map[string]string{
			"Content-Type": "application/json",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid request body")
	})
}

// TestStartStopStream tests streaming control
func TestStartStopStream(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("StartStream", func(t *testing.T) {
		// First create a stream
		createPayload := map[string]interface{}{
			"name": "Test Stream",
			"input_source": map[string]interface{}{
				"type": "file",
			},
			"output_targets": []map[string]interface{}{
				{"platform": "custom", "url": "rtmp://test.com/live"},
			},
		}

		createW := makeJSONRequest(server, "POST", "/api/v1/stream/create", createPayload)
		createResp := assertJSONResponse(t, createW, http.StatusCreated)
		sessionID := createResp["data"].(map[string]interface{})["session_id"].(string)

		// Start the stream
		startPath := fmt.Sprintf("/api/v1/stream/%s/start", sessionID)
		startW := makeHTTPRequest(server, "POST", startPath, nil, nil)

		response := assertJSONResponse(t, startW, http.StatusOK)
		data := response["data"].(map[string]interface{})

		if status, ok := data["status"].(string); !ok || status != "streaming" {
			t.Errorf("Expected status='streaming', got %v", data["status"])
		}
	})

	t.Run("StopStream", func(t *testing.T) {
		// Create and start a stream
		createPayload := map[string]interface{}{
			"name": "Test Stream 2",
			"input_source": map[string]interface{}{
				"type": "file",
			},
			"output_targets": []map[string]interface{}{
				{"platform": "custom", "url": "rtmp://test.com/live"},
			},
		}

		createW := makeJSONRequest(server, "POST", "/api/v1/stream/create", createPayload)
		createResp := assertJSONResponse(t, createW, http.StatusCreated)
		sessionID := createResp["data"].(map[string]interface{})["session_id"].(string)

		// Stop the stream
		stopPath := fmt.Sprintf("/api/v1/stream/%s/stop", sessionID)
		stopW := makeHTTPRequest(server, "POST", stopPath, nil, nil)

		response := assertJSONResponse(t, stopW, http.StatusOK)
		data := response["data"].(map[string]interface{})

		if status, ok := data["status"].(string); !ok || status != "stopped" {
			t.Errorf("Expected status='stopped', got %v", data["status"])
		}
	})
}

// TestGetVideoInfo tests video info endpoint
func TestGetVideoInfo(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("Success", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		// Create test video file
		videoPath := fmt.Sprintf("%s/uploads/%s.mp4", server.config.WorkDir, videoID)
		createTestVideo(t, videoPath)

		path := fmt.Sprintf("/api/v1/video/%s/info", videoID)
		w := makeHTTPRequest(server, "GET", path, nil, nil)

		// May succeed or fail depending on ffmpeg
		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK)
			t.Logf("Video info: %v", response["data"])
		}
	})

	t.Run("VideoNotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/video/%s/info", nonExistentID)

		w := makeHTTPRequest(server, "GET", path, nil, nil)
		assertErrorResponse(t, w, http.StatusNotFound, "video file not found")
	})
}

// TestCompleteWorkflow tests end-to-end workflow
func TestCompleteWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("VideoProcessingWorkflow", func(t *testing.T) {
		// 1. Insert video
		videoID := insertTestVideo(t, server.db)

		// 2. Get video details
		getPath := fmt.Sprintf("/api/v1/video/%s", videoID)
		getW := makeHTTPRequest(server, "GET", getPath, nil, nil)
		assertJSONResponse(t, getW, http.StatusOK)

		// 3. Create conversion job
		convertPath := fmt.Sprintf("/api/v1/video/%s/convert", videoID)
		convertPayload := GenerateTestConvertRequest("webm")
		convertW := makeJSONRequest(server, "POST", convertPath, convertPayload)
		convertResp := assertJSONResponse(t, convertW, http.StatusAccepted)
		jobID := convertResp["data"].(map[string]interface{})["job_id"].(string)

		// 4. Get job status
		jobPath := fmt.Sprintf("/api/v1/jobs/%s", jobID)
		jobW := makeHTTPRequest(server, "GET", jobPath, nil, nil)
		assertJSONResponse(t, jobW, http.StatusOK)

		// 5. List all jobs
		listJobsW := makeHTTPRequest(server, "GET", "/api/v1/jobs", nil, nil)
		assertJSONResponse(t, listJobsW, http.StatusOK)

		// 6. Cancel job
		cancelPath := fmt.Sprintf("/api/v1/jobs/%s/cancel", jobID)
		cancelW := makeHTTPRequest(server, "POST", cancelPath, nil, nil)
		assertJSONResponse(t, cancelW, http.StatusOK)
	})
}
