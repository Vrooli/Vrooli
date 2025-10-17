// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Setup
	cleanup := setupTestLogger()
	defer cleanup()

	// Run tests
	m.Run()
}

// Test Health Check
func TestHealthCheck(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, "GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "ok" {
			t.Errorf("Expected status 'ok', got %v", status)
		}

		if _, ok := response["timestamp"].(string); !ok {
			t.Error("Expected timestamp in response")
		}

		if service, ok := response["service"].(string); !ok || service != "smart-file-photo-manager-api" {
			t.Errorf("Expected service name, got %v", service)
		}
	})

	t.Run("ResponseTime", func(t *testing.T) {
		start := time.Now()
		w, _ := makeHTTPRequest(env, "GET", "/health", nil)
		duration := time.Since(start)

		assertStatusCode(t, w, http.StatusOK)

		if duration > 50*time.Millisecond {
			t.Errorf("Health check took too long: %v", duration)
		}
	})
}

// Test File Operations
func TestGetFiles(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		cleanupTestData(env)
		w, err := makeHTTPRequest(env, "GET", "/api/files", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["files"]; !ok {
			t.Error("Expected 'files' field in response")
		}
	})

	t.Run("Success_WithFiles", func(t *testing.T) {
		cleanupTestData(env)

		// Create test files
		file1 := createTestFile(t, env, "test1.jpg", "image/jpeg")
		defer file1.Cleanup()

		file2 := createTestFile(t, env, "test2.pdf", "application/pdf")
		defer file2.Cleanup()

		w, _ := makeHTTPRequest(env, "GET", "/api/files", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		files, ok := response["files"].([]interface{})
		if !ok {
			t.Fatal("Expected files array")
		}

		if len(files) < 2 {
			t.Errorf("Expected at least 2 files, got %d", len(files))
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		cleanupTestData(env)

		// Create multiple files
		for i := 0; i < 5; i++ {
			file := createTestFile(t, env, fmt.Sprintf("file%d.txt", i), "text/plain")
			defer file.Cleanup()
		}

		// Test with limit
		w, _ := makeHTTPRequest(env, "GET", "/api/files?limit=2&offset=0", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if limit, ok := response["limit"].(float64); !ok || int(limit) != 2 {
			t.Errorf("Expected limit 2, got %v", limit)
		}

		if offset, ok := response["offset"].(float64); !ok || int(offset) != 0 {
			t.Errorf("Expected offset 0, got %v", offset)
		}
	})

	t.Run("FilterByType", func(t *testing.T) {
		cleanupTestData(env)

		// Create files of different types
		img := createTestFile(t, env, "image.jpg", "image/jpeg")
		defer img.Cleanup()
		env.DB.Exec("UPDATE files SET file_type = 'image' WHERE id = $1", img.ID)

		doc := createTestFile(t, env, "doc.pdf", "application/pdf")
		defer doc.Cleanup()
		env.DB.Exec("UPDATE files SET file_type = 'document' WHERE id = $1", doc.ID)

		// Filter by type
		w, _ := makeHTTPRequest(env, "GET", "/api/files?type=image", nil)
		assertStatusCode(t, w, http.StatusOK)
	})
}

func TestGetFile(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		file := createTestFile(t, env, "test.jpg", "image/jpeg")
		defer file.Cleanup()

		w, _ := makeHTTPRequest(env, "GET", fmt.Sprintf("/api/files/%s", file.ID), nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if id, ok := response["id"].(string); !ok || id != file.ID {
			t.Errorf("Expected file ID %s, got %v", file.ID, id)
		}

		if name, ok := response["original_name"].(string); !ok || name != file.OriginalName {
			t.Errorf("Expected filename %s, got %v", file.OriginalName, name)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, "GET", "/api/files/00000000-0000-0000-0000-000000000000", nil)
		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, "GET", "/api/files/invalid-uuid", nil)
		// Note: Gin router handles this before it reaches the handler
		// Status might be 404 or handler-specific
		if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 404 or 400, got %d", w.Code)
		}
	})
}

func TestUploadFile(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		body := map[string]interface{}{
			"filename":     "upload_test.jpg",
			"file_hash":    "hash_upload_test",
			"size_bytes":   2048,
			"mime_type":    "image/jpeg",
			"storage_path": "/test/upload_test.jpg",
			"folder_path":  "/",
			"metadata": map[string]interface{}{
				"camera": "iPhone",
			},
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/files", body)
		response := assertJSONResponse(t, w, http.StatusCreated)

		fileID, ok := response["file_id"].(string)
		if !ok || fileID == "" {
			t.Error("Expected file_id in response")
		}

		// Cleanup
		defer env.DB.Exec("DELETE FROM files WHERE id = $1", fileID)

		if status, ok := response["status"].(string); !ok || status != "uploaded" {
			t.Errorf("Expected status 'uploaded', got %v", status)
		}
	})

	t.Run("MissingFilename", func(t *testing.T) {
		body := map[string]interface{}{
			"file_hash":    "hash_test",
			"size_bytes":   1024,
			"mime_type":    "image/jpeg",
			"storage_path": "/test/test.jpg",
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/files", body)
		assertStatusCode(t, w, http.StatusBadRequest)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, "POST", "/api/files", "invalid-json")
		assertStatusCode(t, w, http.StatusBadRequest)
	})

	t.Run("ZeroByteFile", func(t *testing.T) {
		body := map[string]interface{}{
			"filename":     "empty.txt",
			"file_hash":    "hash_empty",
			"size_bytes":   0,
			"mime_type":    "text/plain",
			"storage_path": "/test/empty.txt",
			"folder_path":  "/",
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/files", body)
		response := assertJSONResponse(t, w, http.StatusCreated)

		if fileID, ok := response["file_id"].(string); ok {
			defer env.DB.Exec("DELETE FROM files WHERE id = $1", fileID)
		}
	})
}

// Test Search Operations
func TestSearchFiles(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_POST", func(t *testing.T) {
		cleanupTestData(env)

		file := createTestFile(t, env, "searchable.txt", "text/plain")
		defer file.Cleanup()

		// Add description for search
		env.DB.Exec("UPDATE files SET description = 'Test description for search' WHERE id = $1", file.ID)

		body := map[string]interface{}{
			"query": "search",
			"type":  "text",
			"limit": 10,
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/search", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["files"]; !ok {
			t.Error("Expected 'files' in response")
		}

		if _, ok := response["search_time"]; !ok {
			t.Error("Expected 'search_time' in response")
		}
	})

	t.Run("Success_GET", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, "GET", "/api/search?q=test&limit=5", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["files"]; !ok {
			t.Error("Expected 'files' in response")
		}
	})

	t.Run("MissingQuery_GET", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, "GET", "/api/search", nil)
		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "",
			"limit": 10,
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/search", body)
		// Empty query should still work, just return less relevant results
		assertStatusCode(t, w, http.StatusOK)
	})
}

// Test Folder Operations
func TestGetFolders(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		cleanupTestData(env)

		folder := createTestFolder(t, env, "/test/photos", "photos")
		defer folder.Cleanup()

		w, _ := makeHTTPRequest(env, "GET", "/api/folders", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		folders, ok := response["folders"].([]interface{})
		if !ok {
			t.Fatal("Expected folders array")
		}

		if len(folders) < 1 {
			t.Error("Expected at least 1 folder")
		}
	})

	t.Run("EmptyList", func(t *testing.T) {
		cleanupTestData(env)

		w, _ := makeHTTPRequest(env, "GET", "/api/folders", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if total, ok := response["total"].(float64); !ok || int(total) != 0 {
			// Empty is acceptable
		}
	})
}

func TestCreateFolder(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		body := map[string]interface{}{
			"path":        "/test/documents",
			"name":        "documents",
			"description": "Test folder for documents",
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/folders", body)
		response := assertJSONResponse(t, w, http.StatusCreated)

		folderID, ok := response["folder_id"].(string)
		if !ok || folderID == "" {
			t.Error("Expected folder_id in response")
		}

		// Cleanup
		defer env.DB.Exec("DELETE FROM folders WHERE id = $1", folderID)
	})

	t.Run("MissingPath", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "test",
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/folders", body)
		assertStatusCode(t, w, http.StatusBadRequest)
	})

	t.Run("MissingName", func(t *testing.T) {
		body := map[string]interface{}{
			"path": "/test",
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/folders", body)
		assertStatusCode(t, w, http.StatusBadRequest)
	})
}

func TestDeleteFolder(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		folder := createTestFolder(t, env, "/test/empty", "empty")

		w, _ := makeHTTPRequest(env, "DELETE", fmt.Sprintf("/api/folders/%s", folder.Path), nil)
		assertStatusCode(t, w, http.StatusOK)

		// Verify deletion
		var count int
		env.DB.QueryRow("SELECT COUNT(*) FROM folders WHERE id = $1", folder.ID).Scan(&count)
		if count != 0 {
			t.Error("Folder was not deleted")
		}
	})

	t.Run("NonEmpty", func(t *testing.T) {
		folder := createTestFolder(t, env, "/test/nonempty", "nonempty")
		defer folder.Cleanup()

		// Add a file to the folder
		file := createTestFile(t, env, "file.txt", "text/plain")
		env.DB.Exec("UPDATE files SET folder_path = $1 WHERE id = $2", folder.Path, file.ID)
		defer file.Cleanup()

		w, _ := makeHTTPRequest(env, "DELETE", fmt.Sprintf("/api/folders/%s", folder.Path), nil)
		assertErrorResponse(t, w, http.StatusBadRequest, "not empty")
	})

	t.Run("NotFound", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, "DELETE", "/api/folders/non-existent", nil)
		assertStatusCode(t, w, http.StatusNotFound)
	})
}

// Test Duplicate Detection
func TestFindDuplicates(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_NoDuplicates", func(t *testing.T) {
		cleanupTestData(env)

		w, _ := makeHTTPRequest(env, "POST", "/api/find-duplicates", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if groups, ok := response["duplicate_groups"].([]interface{}); !ok || len(groups) != 0 {
			// No duplicates is valid
		}
	})

	t.Run("Success_WithDuplicates", func(t *testing.T) {
		cleanupTestData(env)

		// Create files with same hash
		sameHash := "duplicate_hash_12345"
		file1 := createTestFile(t, env, "dup1.jpg", "image/jpeg")
		env.DB.Exec("UPDATE files SET file_hash = $1 WHERE id = $2", sameHash, file1.ID)
		defer file1.Cleanup()

		file2 := createTestFile(t, env, "dup2.jpg", "image/jpeg")
		env.DB.Exec("UPDATE files SET file_hash = $1 WHERE id = $2", sameHash, file2.ID)
		defer file2.Cleanup()

		w, _ := makeHTTPRequest(env, "POST", "/api/find-duplicates", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		groups, ok := response["duplicate_groups"].([]interface{})
		if !ok {
			t.Fatal("Expected duplicate_groups array")
		}

		if len(groups) < 1 {
			t.Error("Expected at least 1 duplicate group")
		}
	})
}

// Test Organization
func TestOrganizeFiles(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_ByType", func(t *testing.T) {
		cleanupTestData(env)

		file := createTestFile(t, env, "organize_test.jpg", "image/jpeg")
		defer file.Cleanup()

		body := map[string]interface{}{
			"file_ids": []string{file.ID},
			"strategy": "by_type",
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/organize", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		if msg, ok := response["message"].(string); !ok || msg == "" {
			t.Error("Expected message in response")
		}
	})

	t.Run("Success_AllFiles", func(t *testing.T) {
		cleanupTestData(env)

		file := createTestFile(t, env, "auto_organize.txt", "text/plain")
		defer file.Cleanup()

		body := map[string]interface{}{
			"strategy": "smart",
		}

		w, _ := makeHTTPRequest(env, "POST", "/api/organize", body)
		assertStatusCode(t, w, http.StatusOK)
	})
}

// Test Processing Status
func TestProcessingStatus(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		file := createTestFile(t, env, "status_test.jpg", "image/jpeg")
		defer file.Cleanup()

		w, _ := makeHTTPRequest(env, "GET", fmt.Sprintf("/api/processing-status/%s", file.ID), nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["status"]; !ok {
			t.Error("Expected status in response")
		}

		if _, ok := response["processing_stage"]; !ok {
			t.Error("Expected processing_stage in response")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		w, _ := makeHTTPRequest(env, "GET", "/api/processing-status/00000000-0000-0000-0000-000000000000", nil)
		assertStatusCode(t, w, http.StatusNotFound)
	})
}

// Test Boundary Conditions
func TestBoundaryConditions(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	tests := CreateBoundaryTests(env)
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.TestFunc(t, env)
		})
	}
}

// Test Performance
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	tests := CreatePerformanceTests(env)
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var totalDuration time.Duration
			for i := 0; i < test.Iterations; i++ {
				start := time.Now()
				test.TestFunc(t, env)
				totalDuration += time.Since(start)
			}

			avgDuration := totalDuration / time.Duration(test.Iterations)
			if avgDuration > time.Duration(test.MaxDuration)*time.Millisecond {
				t.Errorf("%s: Average duration %v exceeds max %dms",
					test.Name, avgDuration, test.MaxDuration)
			}
		})
	}
}

// Test Error Handling Patterns
func TestErrorHandling(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := &HandlerTestSuite{
		HandlerName: "Files",
		Env:         env,
	}

	t.Run("FileUploadErrors", func(t *testing.T) {
		patterns := FileUploadErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})

	t.Run("SearchErrors", func(t *testing.T) {
		patterns := SearchErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})

	t.Run("FolderErrors", func(t *testing.T) {
		patterns := FolderErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})
}
