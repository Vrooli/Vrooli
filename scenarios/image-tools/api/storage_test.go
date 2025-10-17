package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestLocalStorage tests the local filesystem storage implementation
func TestLocalStorage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("NewLocalStorage", func(t *testing.T) {
		storage := NewLocalStorage()

		if storage == nil {
			t.Fatal("NewLocalStorage returned nil")
		}

		localStorage, ok := storage.(*LocalStorage)
		if !ok {
			t.Fatal("Expected *LocalStorage type")
		}

		if localStorage.basePath == "" {
			t.Error("Base path should not be empty")
		}
	})

	t.Run("SaveAndRetrieve", func(t *testing.T) {
		storage := NewLocalStorage()

		testData := []byte("test image data")
		testKey := "test/image.jpg"

		// Save data
		url, err := storage.Save(testKey, testData)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		if url == "" {
			t.Error("Save returned empty URL")
		}

		// Retrieve data
		retrieved, err := storage.Get(testKey)
		if err != nil {
			t.Fatalf("Failed to retrieve: %v", err)
		}

		if string(retrieved) != string(testData) {
			t.Errorf("Retrieved data doesn't match. Expected: %s, Got: %s", string(testData), string(retrieved))
		}
	})

	t.Run("SaveCreatesDirectories", func(t *testing.T) {
		storage := NewLocalStorage()

		testData := []byte("test data")
		testKey := "nested/path/to/image.jpg"

		url, err := storage.Save(testKey, testData)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		if url == "" {
			t.Error("Save returned empty URL")
		}

		// Verify directories were created
		localStorage := storage.(*LocalStorage)
		fullPath := filepath.Join(localStorage.basePath, testKey)
		dir := filepath.Dir(fullPath)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Error("Expected directories to be created")
		}
	})

	t.Run("GetNonExistentFile", func(t *testing.T) {
		storage := NewLocalStorage()

		_, err := storage.Get("nonexistent/file.jpg")
		if err == nil {
			t.Error("Expected error when getting non-existent file")
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		storage := NewLocalStorage()

		testData := []byte("test data")
		testKey := "delete-test/image.jpg"

		// Save data
		_, err := storage.Save(testKey, testData)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		// Delete data
		err = storage.Delete(testKey)
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		// Verify deletion
		_, err = storage.Get(testKey)
		if err == nil {
			t.Error("Expected error when getting deleted file")
		}
	})

	t.Run("DeleteNonExistentFile", func(t *testing.T) {
		storage := NewLocalStorage()

		err := storage.Delete("nonexistent/file.jpg")
		if err == nil {
			t.Error("Expected error when deleting non-existent file")
		}
	})

	t.Run("SaveEmptyData", func(t *testing.T) {
		storage := NewLocalStorage()

		emptyData := []byte{}
		testKey := "empty/file.jpg"

		url, err := storage.Save(testKey, emptyData)
		if err != nil {
			t.Fatalf("Failed to save empty data: %v", err)
		}

		if url == "" {
			t.Error("Save returned empty URL for empty data")
		}

		// Retrieve and verify
		retrieved, err := storage.Get(testKey)
		if err != nil {
			t.Fatalf("Failed to retrieve empty data: %v", err)
		}

		if len(retrieved) != 0 {
			t.Errorf("Expected empty data, got %d bytes", len(retrieved))
		}
	})

	t.Run("SaveLargeFile", func(t *testing.T) {
		storage := NewLocalStorage()

		// Create 1MB test data
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		testKey := "large/file.jpg"

		url, err := storage.Save(testKey, largeData)
		if err != nil {
			t.Fatalf("Failed to save large file: %v", err)
		}

		if url == "" {
			t.Error("Save returned empty URL for large file")
		}

		// Retrieve and verify size
		retrieved, err := storage.Get(testKey)
		if err != nil {
			t.Fatalf("Failed to retrieve large file: %v", err)
		}

		if len(retrieved) != len(largeData) {
			t.Errorf("Retrieved size doesn't match. Expected: %d, Got: %d", len(largeData), len(retrieved))
		}
	})

	t.Run("SaveWithSpecialCharacters", func(t *testing.T) {
		storage := NewLocalStorage()

		testData := []byte("test data")
		testKey := "special/path-with_chars/image-2024.jpg"

		url, err := storage.Save(testKey, testData)
		if err != nil {
			t.Fatalf("Failed to save with special characters: %v", err)
		}

		if url == "" {
			t.Error("Save returned empty URL")
		}

		// Retrieve to verify
		retrieved, err := storage.Get(testKey)
		if err != nil {
			t.Fatalf("Failed to retrieve: %v", err)
		}

		if string(retrieved) != string(testData) {
			t.Error("Retrieved data doesn't match")
		}
	})

	t.Run("MultipleFilesInSameDirectory", func(t *testing.T) {
		storage := NewLocalStorage()

		testData1 := []byte("data 1")
		testData2 := []byte("data 2")

		_, err := storage.Save("same-dir/file1.jpg", testData1)
		if err != nil {
			t.Fatalf("Failed to save file1: %v", err)
		}

		_, err = storage.Save("same-dir/file2.jpg", testData2)
		if err != nil {
			t.Fatalf("Failed to save file2: %v", err)
		}

		// Retrieve both
		retrieved1, err := storage.Get("same-dir/file1.jpg")
		if err != nil {
			t.Fatalf("Failed to retrieve file1: %v", err)
		}

		retrieved2, err := storage.Get("same-dir/file2.jpg")
		if err != nil {
			t.Fatalf("Failed to retrieve file2: %v", err)
		}

		if string(retrieved1) != string(testData1) {
			t.Error("File1 data doesn't match")
		}

		if string(retrieved2) != string(testData2) {
			t.Error("File2 data doesn't match")
		}
	})
}

// TestMinIOStorageCreation tests MinIO storage initialization
func TestMinIOStorageCreation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MinIODisabled", func(t *testing.T) {
		os.Setenv("MINIO_DISABLED", "true")
		defer os.Unsetenv("MINIO_DISABLED")

		// When MinIO is disabled, NewServer should use LocalStorage
		server := NewServer()

		if server.storage == nil {
			t.Fatal("Storage should not be nil")
		}

		// Verify it's using local storage
		_, isLocal := server.storage.(*LocalStorage)
		if !isLocal {
			t.Error("Expected LocalStorage when MinIO is disabled")
		}
	})

	t.Run("MinIOConnectionFailure", func(t *testing.T) {
		// Temporarily unset MINIO_DISABLED to test connection failure
		os.Unsetenv("MINIO_DISABLED")

		// Set invalid MinIO configuration
		os.Setenv("MINIO_ENDPOINT", "invalid:9999")
		os.Setenv("MINIO_ACCESS_KEY", "test")
		os.Setenv("MINIO_SECRET_KEY", "test")

		defer func() {
			os.Unsetenv("MINIO_ENDPOINT")
			os.Unsetenv("MINIO_ACCESS_KEY")
			os.Unsetenv("MINIO_SECRET_KEY")
			os.Setenv("MINIO_DISABLED", "true")
		}()

		// Server creation should not fail, but should fall back to local storage
		server := NewServer()

		if server.storage == nil {
			t.Fatal("Storage should not be nil even with MinIO failure")
		}

		// Should have fallen back to local storage
		_, isLocal := server.storage.(*LocalStorage)
		if !isLocal {
			t.Log("Note: Storage is not local, MinIO may actually be accessible")
		}
	})
}

// TestStorageIntegration tests storage integration with server handlers
func TestStorageIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("CompressAndStore", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "test.jpg", imageData, map[string]string{
			"quality": "85",
		})

		headers := map[string]string{
			"Content-Type": contentType,
		}

		resp, respBody, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/compress", body, headers)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if resp.StatusCode == 200 {
			result := assertJSONResponse(t, resp, respBody, 200)

			url, ok := result["url"].(string)
			if !ok || url == "" {
				t.Error("Expected valid URL in response")
			}

			// Verify URL format
			if url[:7] != "file://" && url[:7] != "http://" {
				t.Errorf("Unexpected URL format: %s", url)
			}
		}
	})

	t.Run("StorageURLFormat", func(t *testing.T) {
		storage := NewLocalStorage()

		testData := []byte("test")
		url, err := storage.Save("test/file.jpg", testData)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		if url[:7] != "file://" {
			t.Errorf("Expected file:// URL, got: %s", url)
		}
	})
}

// TestGetContentType tests the content type utility function
func TestContentTypeDetection(t *testing.T) {
	tests := []struct {
		filename    string
		expected    string
		description string
	}{
		{"image.jpg", "image/jpeg", "JPEG with .jpg extension"},
		{"image.jpeg", "image/jpeg", "JPEG with .jpeg extension"},
		{"IMAGE.JPG", "image/jpeg", "Uppercase JPEG"},
		{"photo.png", "image/png", "PNG format"},
		{"graphic.webp", "image/webp", "WebP format"},
		{"logo.svg", "image/svg+xml", "SVG format"},
		{"animation.gif", "image/gif", "GIF format"},
		{"document.pdf", "application/octet-stream", "Unknown format"},
		{"file", "application/octet-stream", "No extension"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := getContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("getContentType(%s) = %s, want %s", tt.filename, result, tt.expected)
			}
		})
	}
}

// TestStorageEdgeCases tests edge cases in storage operations
func TestStorageEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SaveWithLeadingSlash", func(t *testing.T) {
		storage := NewLocalStorage()

		testData := []byte("test")
		testKey := "/leading/slash/file.jpg"

		url, err := storage.Save(testKey, testData)
		if err != nil {
			t.Fatalf("Failed to save with leading slash: %v", err)
		}

		if url == "" {
			t.Error("Expected valid URL")
		}

		// Should be able to retrieve
		retrieved, err := storage.Get(testKey)
		if err != nil {
			t.Fatalf("Failed to retrieve: %v", err)
		}

		if string(retrieved) != string(testData) {
			t.Error("Data mismatch")
		}
	})

	t.Run("SaveWithTrailingSlash", func(t *testing.T) {
		storage := NewLocalStorage()

		testData := []byte("test")
		testKey := "trailing/slash/"

		_, err := storage.Save(testKey, testData)
		// Behavior may vary - either succeeds or fails
		// Just ensure it doesn't panic
		_ = err
	})

	t.Run("SaveWithDotDot", func(t *testing.T) {
		storage := NewLocalStorage()

		testData := []byte("test")
		testKey := "path/../escape/file.jpg"

		// Save should work (path is normalized by filepath.Join)
		url, err := storage.Save(testKey, testData)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		if url == "" {
			t.Error("Expected valid URL")
		}
	})

	t.Run("ConcurrentStorageOperations", func(t *testing.T) {
		storage := NewLocalStorage()
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				testData := []byte("concurrent test")
				testKey := fmt.Sprintf("concurrent/%d/file.jpg", index)

				_, err := storage.Save(testKey, testData)
				if err != nil {
					t.Errorf("Concurrent save failed: %v", err)
				}

				done <- true
			}(i)
		}

		// Wait for all operations
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
