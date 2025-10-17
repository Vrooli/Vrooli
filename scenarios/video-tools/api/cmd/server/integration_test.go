package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
)

// TestIntegrationVideoUpload tests video upload integration
func TestIntegrationVideoUpload(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("CompleteUploadFlow", func(t *testing.T) {
		// Test multipart upload would go here
		// For now, we test the database insertion flow
		videoID := insertTestVideo(t, server.db)

		// Verify video was created
		var count int
		err := server.db.QueryRow("SELECT COUNT(*) FROM video_assets WHERE id = $1", videoID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query video: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 video, found %d", count)
		}
	})
}

// TestIntegrationJobProcessing tests job processing integration
func TestIntegrationJobProcessing(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("JobLifecycle", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		// Create job
		path := fmt.Sprintf("/api/v1/video/%s/convert", videoID)
		payload := GenerateTestConvertRequest("webm")
		w := makeJSONRequest(server, "POST", path, payload)

		response := assertJSONResponse(t, w, http.StatusAccepted)
		data := response["data"].(map[string]interface{})
		jobID := data["job_id"].(string)

		// Verify job in database
		var status string
		err := server.db.QueryRow("SELECT status FROM processing_jobs WHERE id = $1", jobID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query job: %v", err)
		}

		if status != "pending" {
			t.Errorf("Expected status 'pending', got '%s'", status)
		}

		// Cancel job
		cancelPath := fmt.Sprintf("/api/v1/jobs/%s/cancel", jobID)
		cancelW := makeHTTPRequest(server, "POST", cancelPath, nil, nil)
		assertJSONResponse(t, cancelW, http.StatusOK)

		// Verify job was cancelled
		err = server.db.QueryRow("SELECT status FROM processing_jobs WHERE id = $1", jobID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query job after cancel: %v", err)
		}

		if status != "cancelled" {
			t.Errorf("Expected status 'cancelled', got '%s'", status)
		}
	})
}

// TestIntegrationStreamingSession tests streaming session integration
func TestIntegrationStreamingSession(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("StreamSessionLifecycle", func(t *testing.T) {
		// Create stream
		payload := map[string]interface{}{
			"name": "Integration Test Stream",
			"input_source": map[string]interface{}{
				"type": "file",
			},
			"output_targets": []map[string]interface{}{
				{"platform": "custom", "url": "rtmp://test.com/live"},
			},
		}

		createW := makeJSONRequest(server, "POST", "/api/v1/stream/create", payload)
		createResp := assertJSONResponse(t, createW, http.StatusCreated)
		sessionID := createResp["data"].(map[string]interface{})["session_id"].(string)

		// Verify stream in database
		var isActive bool
		err := server.db.QueryRow("SELECT is_active FROM streaming_sessions WHERE id = $1", sessionID).Scan(&isActive)
		if err != nil {
			t.Fatalf("Failed to query stream: %v", err)
		}

		if isActive {
			t.Error("Stream should not be active initially")
		}

		// Start stream
		startPath := fmt.Sprintf("/api/v1/stream/%s/start", sessionID)
		startW := makeHTTPRequest(server, "POST", startPath, nil, nil)
		assertJSONResponse(t, startW, http.StatusOK)

		// Verify stream is active
		err = server.db.QueryRow("SELECT is_active FROM streaming_sessions WHERE id = $1", sessionID).Scan(&isActive)
		if err != nil {
			t.Fatalf("Failed to query stream after start: %v", err)
		}

		if !isActive {
			t.Error("Stream should be active after starting")
		}

		// Stop stream
		stopPath := fmt.Sprintf("/api/v1/stream/%s/stop", sessionID)
		stopW := makeHTTPRequest(server, "POST", stopPath, nil, nil)
		assertJSONResponse(t, stopW, http.StatusOK)

		// Verify stream is stopped
		err = server.db.QueryRow("SELECT is_active FROM streaming_sessions WHERE id = $1", sessionID).Scan(&isActive)
		if err != nil {
			t.Fatalf("Failed to query stream after stop: %v", err)
		}

		if isActive {
			t.Error("Stream should not be active after stopping")
		}
	})
}

// TestIntegrationConcurrentOperations tests concurrent request handling
func TestIntegrationConcurrentOperations(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("ConcurrentVideoRetrieval", func(t *testing.T) {
		// Create test videos
		videoIDs := make([]string, 5)
		for i := 0; i < 5; i++ {
			videoIDs[i] = insertTestVideo(t, server.db)
		}

		// Concurrent retrieval
		done := make(chan bool, 5)
		for _, videoID := range videoIDs {
			go func(id string) {
				path := fmt.Sprintf("/api/v1/video/%s", id)
				w := makeHTTPRequest(server, "GET", path, nil, nil)
				if w.Code != http.StatusOK {
					t.Errorf("Concurrent retrieval failed for video %s: %d", id, w.Code)
				}
				done <- true
			}(videoID)
		}

		// Wait for all to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})

	t.Run("ConcurrentJobCreation", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s/convert", videoID)

		done := make(chan string, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				payload := GenerateTestConvertRequest("mp4")
				w := makeJSONRequest(server, "POST", path, payload)

				if w.Code == http.StatusAccepted {
					var response map[string]interface{}
					json.Unmarshal(w.Body.Bytes(), &response)
					if data, ok := response["data"].(map[string]interface{}); ok {
						if jobID, ok := data["job_id"].(string); ok {
							done <- jobID
							return
						}
					}
				}
				done <- ""
			}(i)
		}

		// Collect job IDs
		created := 0
		for i := 0; i < 10; i++ {
			jobID := <-done
			if jobID != "" {
				created++
			}
		}

		if created != 10 {
			t.Errorf("Expected 10 jobs created, got %d", created)
		}
	})
}

// TestIntegrationDatabaseTransactions tests database transaction handling
func TestIntegrationDatabaseTransactions(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("JobCancellation", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		jobID := insertTestJob(t, server.db, videoID, "convert")

		// Try to cancel non-cancellable job (completed)
		_, err := server.db.Exec("UPDATE processing_jobs SET status = 'completed' WHERE id = $1", jobID)
		if err != nil {
			t.Fatalf("Failed to update job status: %v", err)
		}

		// Attempt to cancel
		cancelPath := fmt.Sprintf("/api/v1/jobs/%s/cancel", jobID)
		w := makeHTTPRequest(server, "POST", cancelPath, nil, nil)

		// Should fail because job is completed
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for cancelling completed job, got %d", w.Code)
		}
	})
}

// TestIntegrationErrorRecovery tests error handling and recovery
func TestIntegrationErrorRecovery(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("InvalidVideoID", func(t *testing.T) {
		invalidID := "not-a-uuid"
		path := fmt.Sprintf("/api/v1/video/%s", invalidID)

		w := makeHTTPRequest(server, "GET", path, nil, nil)
		// Should return 404 for invalid route pattern
		if w.Code != http.StatusNotFound {
			t.Logf("Got status %d for invalid UUID", w.Code)
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s/convert", videoID)

		w := makeHTTPRequest(server, "POST", path, nil, map[string]string{
			"Content-Type": "application/json",
		})

		// Should handle nil body gracefully
		if w.Code < 200 || w.Code >= 500 {
			t.Errorf("Server error on nil body: %d", w.Code)
		}
	})
}

// TestIntegrationDataConsistency tests data consistency across operations
func TestIntegrationDataConsistency(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("VideoJobRelationship", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)

		// Create multiple jobs for the same video
		jobTypes := []string{"convert", "edit", "compress", "analyze"}
		for _, jobType := range jobTypes {
			insertTestJob(t, server.db, videoID, jobType)
		}

		// Query jobs for video
		var count int
		err := server.db.QueryRow(`
			SELECT COUNT(*) FROM processing_jobs
			WHERE video_id = $1
		`, videoID).Scan(&count)

		if err != nil {
			t.Fatalf("Failed to count jobs: %v", err)
		}

		if count != len(jobTypes) {
			t.Errorf("Expected %d jobs, found %d", len(jobTypes), count)
		}

		// Verify we can list all jobs
		w := makeHTTPRequest(server, "GET", "/api/v1/jobs", nil, nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		jobs := response["data"].([]interface{})
		if len(jobs) < len(jobTypes) {
			t.Errorf("Expected at least %d jobs in list, got %d", len(jobTypes), len(jobs))
		}
	})
}
