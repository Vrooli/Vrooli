// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})
}

// TestCreateDownloadHandler tests the download creation endpoint
func TestCreateDownloadHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	// Set global db for handlers
	db = testDB.DB

	t.Run("Success_BasicDownload", func(t *testing.T) {
		reqBody := DownloadRequest{
			URL:     "https://youtube.com/watch?v=test123",
			Quality: "720p",
			Format:  "mp4",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/download",
			Body:   reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createDownloadHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "queued",
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify download_id is present
		if _, ok := response["download_id"]; !ok {
			t.Error("Expected download_id in response")
		}
	})

	t.Run("Success_WithTranscript", func(t *testing.T) {
		reqBody := DownloadRequest{
			URL:                "https://youtube.com/watch?v=test456",
			GenerateTranscript: true,
			WhisperModel:       "base",
			TargetLanguage:     "en",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/download",
			Body:   reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createDownloadHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":               "queued",
			"transcript_requested": true,
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("Success_AudioOnly", func(t *testing.T) {
		reqBody := DownloadRequest{
			URL:          "https://youtube.com/watch?v=test789",
			AudioOnly:    true,
			AudioFormat:  "mp3",
			AudioQuality: "320k",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/download",
			Body:   reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createDownloadHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		if audioFormat, ok := response["audio_format"].(string); ok {
			if audioFormat != "mp3" {
				t.Errorf("Expected audio_format mp3, got %s", audioFormat)
			}
		}
	})

	t.Run("Error_MissingURL", func(t *testing.T) {
		reqBody := DownloadRequest{
			Quality: "720p",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/download",
			Body:   reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createDownloadHandler(w, httpReq)
		assertHTMLErrorResponse(t, w, http.StatusBadRequest, "URL is required")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/download",
			Body:   `{"invalid json`,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createDownloadHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/download",
			Body:   "",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createDownloadHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestGetQueueHandler tests the queue retrieval endpoint
func TestGetQueueHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success_EmptyQueue", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/queue",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getQueueHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Success_WithItems", func(t *testing.T) {
		// Create test downloads
		download1 := setupTestDownload(t, testDB.DB, "https://test.com/1", map[string]interface{}{
			"status": "pending",
		})
		defer download1.Cleanup()

		// Add to queue
		_, err := testDB.DB.Exec("INSERT INTO download_queue (download_id, position, priority) VALUES ($1, 1, 1)", download1.Download.ID)
		if err != nil {
			t.Fatalf("Failed to add to queue: %v", err)
		}

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/queue",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getQueueHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestGetHistoryHandler tests the history retrieval endpoint
func TestGetHistoryHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success_EmptyHistory", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/history",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getHistoryHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Success_WithHistory", func(t *testing.T) {
		// Create completed download
		download := setupTestDownload(t, testDB.DB, "https://test.com/completed", map[string]interface{}{
			"status": "completed",
		})
		defer download.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/history",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getHistoryHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestDeleteDownloadHandler tests the download deletion endpoint
func TestDeleteDownloadHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		download := setupTestDownload(t, testDB.DB, "https://test.com/delete", nil)
		defer download.Cleanup()

		// Add to queue
		_, err := testDB.DB.Exec("INSERT INTO download_queue (download_id, position) VALUES ($1, 1)", download.Download.ID)
		if err != nil {
			t.Fatalf("Failed to add to queue: %v", err)
		}

		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/download/%d", download.Download.ID),
			URLVars: map[string]string{"id": fmt.Sprintf("%d", download.Download.ID)},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteDownloadHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "cancelled",
		})
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/download/invalid",
			URLVars: map[string]string{"id": "invalid"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteDownloadHandler(w, httpReq)
		assertHTMLErrorResponse(t, w, http.StatusBadRequest, "Invalid ID")
	})

	t.Run("Error_NonExistent", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/download/999999",
			URLVars: map[string]string{"id": "999999"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteDownloadHandler(w, httpReq)

		// Should still return OK even for non-existent (idempotent)
		if w.Code != http.StatusOK {
			t.Logf("Non-existent delete returned status %d", w.Code)
		}
	})
}

// TestAnalyzeURLHandler tests the URL analysis endpoint
func TestAnalyzeURLHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		reqBody := map[string]string{
			"url": "https://youtube.com/watch?v=test",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/analyze",
			Body:   reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		analyzeURLHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"url": "https://youtube.com/watch?v=test",
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/analyze",
			Body:   `{invalid}`,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		analyzeURLHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestProcessQueueHandler tests the queue processing endpoint
func TestProcessQueueHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/queue/process",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		processQueueHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "processing",
		})
	})
}

// TestGetTranscriptHandler tests the transcript retrieval endpoint
func TestGetTranscriptHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success_WithSegments", func(t *testing.T) {
		download := setupTestDownload(t, testDB.DB, "https://test.com/transcript", nil)
		defer download.Cleanup()

		transcriptID := setupTestTranscript(t, testDB.DB, download.Download.ID)
		if transcriptID == 0 {
			t.Fatal("Failed to create test transcript")
		}

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/transcript/%d", download.Download.ID),
			URLVars: map[string]string{"download_id": fmt.Sprintf("%d", download.Download.ID)},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getTranscriptHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"download_id": float64(download.Download.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Check segments are included
		if segments, ok := response["segments"].([]interface{}); ok {
			if len(segments) == 0 {
				t.Error("Expected segments in response")
			}
		}
	})

	t.Run("Success_NoSegments", func(t *testing.T) {
		download := setupTestDownload(t, testDB.DB, "https://test.com/transcript2", nil)
		defer download.Cleanup()

		transcriptID := setupTestTranscript(t, testDB.DB, download.Download.ID)
		if transcriptID == 0 {
			t.Fatal("Failed to create test transcript")
		}

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        fmt.Sprintf("/api/transcript/%d", download.Download.ID),
			URLVars:     map[string]string{"download_id": fmt.Sprintf("%d", download.Download.ID)},
			QueryParams: map[string]string{"include_segments": "false"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getTranscriptHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/transcript/invalid",
			URLVars: map[string]string{"download_id": "invalid"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getTranscriptHandler(w, httpReq)
		assertHTMLErrorResponse(t, w, http.StatusBadRequest, "Invalid download ID")
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/transcript/999999",
			URLVars: map[string]string{"download_id": "999999"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getTranscriptHandler(w, httpReq)
		assertHTMLErrorResponse(t, w, http.StatusNotFound, "Transcript not found")
	})
}

// TestSearchTranscriptHandler tests the transcript search endpoint
func TestSearchTranscriptHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		download := setupTestDownload(t, testDB.DB, "https://test.com/search", nil)
		defer download.Cleanup()

		transcriptID := setupTestTranscript(t, testDB.DB, download.Download.ID)
		if transcriptID == 0 {
			t.Fatal("Failed to create test transcript")
		}

		reqBody := TranscriptSearchRequest{
			Query: "test",
		}

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/transcript/%d/search", download.Download.ID),
			URLVars: map[string]string{"download_id": fmt.Sprintf("%d", download.Download.ID)},
			Body:    reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchTranscriptHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Should have matches field
		if _, ok := response["matches"]; !ok {
			t.Error("Expected matches field in response")
		}
	})

	t.Run("Error_MissingQuery", func(t *testing.T) {
		download := setupTestDownload(t, testDB.DB, "https://test.com/search2", nil)
		defer download.Cleanup()

		reqBody := TranscriptSearchRequest{}

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/transcript/%d/search", download.Download.ID),
			URLVars: map[string]string{"download_id": fmt.Sprintf("%d", download.Download.ID)},
			Body:    reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchTranscriptHandler(w, httpReq)
		assertHTMLErrorResponse(t, w, http.StatusBadRequest, "Search query is required")
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		reqBody := TranscriptSearchRequest{
			Query: "test",
		}

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/transcript/invalid/search",
			URLVars: map[string]string{"download_id": "invalid"},
			Body:    reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchTranscriptHandler(w, httpReq)
		assertHTMLErrorResponse(t, w, http.StatusBadRequest, "Invalid download ID")
	})
}

// TestGenerateTranscriptHandler tests the transcript generation endpoint
func TestGenerateTranscriptHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		// Create download with audio path
		var downloadID int
		err := testDB.DB.QueryRow(`
			INSERT INTO downloads (url, title, status, audio_path, whisper_model)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			"https://test.com/generate", "Test", "completed", "/path/to/audio.mp3", "base").Scan(&downloadID)
		if err != nil {
			t.Fatalf("Failed to create download: %v", err)
		}

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/transcript/%d/generate", downloadID),
			URLVars: map[string]string{"download_id": fmt.Sprintf("%d", downloadID)},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateTranscriptHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "processing",
		})
	})

	t.Run("Error_NoAudio", func(t *testing.T) {
		download := setupTestDownload(t, testDB.DB, "https://test.com/noaudio", nil)
		defer download.Cleanup()

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/transcript/%d/generate", download.Download.ID),
			URLVars: map[string]string{"download_id": fmt.Sprintf("%d", download.Download.ID)},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateTranscriptHandler(w, httpReq)
		assertHTMLErrorResponse(t, w, http.StatusBadRequest, "No audio file available for transcription")
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/transcript/invalid/generate",
			URLVars: map[string]string{"download_id": "invalid"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateTranscriptHandler(w, httpReq)
		assertHTMLErrorResponse(t, w, http.StatusBadRequest, "Invalid download ID")
	})
}

// Benchmark tests
func BenchmarkHealthHandler(b *testing.B) {
	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}

	BenchmarkHandler(b, healthHandler, req)
}

func BenchmarkCreateDownloadHandler(b *testing.B) {
	testDB := setupTestDatabase(&testing.T{})
	if testDB == nil {
		b.Skip("Test database not available")
	}
	defer testDB.Cleanup()

	db = testDB.DB

	reqBody := DownloadRequest{
		URL:     "https://youtube.com/watch?v=bench",
		Quality: "720p",
	}

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/download",
		Body:   reqBody,
	}

	BenchmarkHandler(b, createDownloadHandler, req)
}
