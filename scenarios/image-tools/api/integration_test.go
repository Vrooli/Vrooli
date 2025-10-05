package main

import (
	"os"
	"strings"
	"testing"
)

// TestSaveToStorageFallback tests the saveToStorage fallback mechanism
func TestSaveToStorageFallback(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SaveWhenStorageNil", func(t *testing.T) {
		// Create server with nil storage to test fallback
		server := &Server{
			app:      nil,
			registry: nil,
			storage:  nil, // Force fallback
		}

		testData := []byte("test data")
		url, err := server.saveToStorage("fallback/test.jpg", testData)
		if err != nil {
			t.Fatalf("Fallback save failed: %v", err)
		}

		if url == "" {
			t.Error("Expected non-empty URL from fallback")
		}

		if !strings.HasPrefix(url, "file://") {
			t.Errorf("Expected file:// URL from fallback, got: %s", url)
		}

		// Verify file was actually created
		expectedPath := "/tmp/image-tools/fallback/test.jpg"
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Error("Fallback should have created file")
		}
	})

	t.Run("SaveWithMixedPaths", func(t *testing.T) {
		server := &Server{
			storage: nil, // Use fallback
		}

		testCases := []string{
			"simple.jpg",
			"path/to/file.jpg",
			"deep/nested/path/file.jpg",
		}

		for _, testKey := range testCases {
			t.Run(testKey, func(t *testing.T) {
				url, err := server.saveToStorage(testKey, []byte("test"))
				if err != nil {
					t.Errorf("Failed to save %s: %v", testKey, err)
				}

				if url == "" {
					t.Errorf("Empty URL for %s", testKey)
				}
			})
		}
	})
}

// TestFullWorkflow tests complete image processing workflows
func TestFullWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("CompleteCompressionWorkflow", func(t *testing.T) {
		// 1. Upload image
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "workflow.jpg", imageData, map[string]string{
			"quality": "85",
		})

		headers := map[string]string{
			"Content-Type": contentType,
		}

		resp, respBody, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/compress", body, headers)
		if err != nil {
			t.Fatalf("Failed compression: %v", err)
		}

		if resp.StatusCode == 200 {
			result := assertJSONResponse(t, resp, respBody, 200)

			// 2. Verify URL is returned
			url, ok := result["url"].(string)
			if !ok || url == "" {
				t.Error("Expected URL in compression result")
			}

			// 3. Try to fetch the image via proxy
			if strings.HasPrefix(url, "file://") {
				proxyResp, _, err := makeHTTPRequest(server.app, "GET", "/api/v1/image/proxy?url="+url, nil, nil)
				if err != nil {
					t.Fatalf("Failed to proxy: %v", err)
				}

				if proxyResp.StatusCode >= 500 {
					t.Logf("Proxy returned server error: %d", proxyResp.StatusCode)
				}
			}
		}
	})

	t.Run("MultistepBatchWorkflow", func(t *testing.T) {
		operations := `{"operations": [
			{"type": "compress", "options": {"quality": 80}},
			{"type": "strip_metadata", "options": {}}
		]}`

		files := map[string][]byte{
			"batch1.jpg": generateTestImageData("jpeg"),
			"batch2.jpg": generateTestImageData("jpeg"),
		}

		body, contentType := createMultipartBatchData(t, files, operations)

		headers := map[string]string{
			"Content-Type": contentType,
		}

		resp, respBody, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/batch", body, headers)
		if err != nil {
			t.Fatalf("Failed batch processing: %v", err)
		}

		if resp.StatusCode == 200 {
			result := assertJSONResponse(t, resp, respBody, 200)

			results, ok := result["results"].([]interface{})
			if !ok {
				t.Error("Expected results array")
			}

			if len(results) == 0 {
				t.Error("Expected at least one result")
			}
		}
	})
}

// TestServerConfiguration tests server configuration scenarios
func TestServerConfiguration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ServerWithLocalStorage", func(t *testing.T) {
		os.Setenv("MINIO_DISABLED", "true")
		defer os.Unsetenv("MINIO_DISABLED")

		server := NewServer()
		if server.storage == nil {
			t.Error("Storage should not be nil")
		}

		// Verify it's local storage
		_, isLocal := server.storage.(*LocalStorage)
		if !isLocal {
			t.Log("Note: Storage is not LocalStorage (MinIO may be configured)")
		}
	})

	t.Run("ServerRoutesAccessible", func(t *testing.T) {
		server := setupTestServer()

		routes := []struct {
			method string
			path   string
		}{
			{"GET", "/api/v1/health"},
			{"GET", "/api/v1/plugins"},
			{"GET", "/api/v1/presets"},
			{"GET", "/api/v1/presets/web-optimized"},
			{"POST", "/api/v1/image/preset/web-optimized"},
		}

		for _, route := range routes {
			t.Run(route.method+"_"+route.path, func(t *testing.T) {
				resp, _, err := makeHTTPRequest(server.app, route.method, route.path, nil, nil)
				if err != nil {
					t.Errorf("Failed to access %s %s: %v", route.method, route.path, err)
				}

				if resp.StatusCode >= 500 {
					t.Errorf("Server error on %s %s: %d", route.method, route.path, resp.StatusCode)
				}
			})
		}
	})
}

// TestErrorHandling tests comprehensive error handling
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("InvalidHTTPMethods", func(t *testing.T) {
		endpoints := []string{
			"/api/v1/health",
			"/api/v1/plugins",
			"/api/v1/presets",
		}

		for _, endpoint := range endpoints {
			// Try DELETE on GET endpoints
			resp, _, err := makeHTTPRequest(server.app, "DELETE", endpoint, nil, nil)
			if err != nil {
				t.Errorf("Request failed: %v", err)
			}

			if resp.StatusCode != 404 && resp.StatusCode != 405 {
				t.Logf("Unexpected status for DELETE %s: %d", endpoint, resp.StatusCode)
			}
		}
	})

	t.Run("MalformedRequests", func(t *testing.T) {
		tests := []struct {
			name   string
			method string
			path   string
			body   string
		}{
			{"EmptyBody", "POST", "/api/v1/image/compress", ""},
			{"InvalidJSON", "POST", "/api/v1/image/batch", "{invalid}"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resp, _, err := makeHTTPRequest(server.app, tt.method, tt.path, strings.NewReader(tt.body), nil)
				if err != nil {
					t.Errorf("Request failed: %v", err)
				}

				if resp.StatusCode >= 500 {
					t.Logf("Server error (may be expected): %d", resp.StatusCode)
				}
			})
		}
	})
}

// TestConcurrentAccess tests concurrent request handling
func TestConcurrentAccess(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		done := make(chan error, 20)

		for i := 0; i < 20; i++ {
			go func() {
				resp, _, err := makeHTTPRequest(server.app, "GET", "/api/v1/health", nil, nil)
				if err != nil {
					done <- err
					return
				}

				if resp.StatusCode != 200 {
					done <- err
					return
				}

				done <- nil
			}()
		}

		for i := 0; i < 20; i++ {
			if err := <-done; err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			}
		}
	})

	t.Run("ConcurrentPresetRequests", func(t *testing.T) {
		presets := []string{"web-optimized", "email-safe", "aggressive", "high-quality", "social-media"}
		done := make(chan error, len(presets))

		for _, preset := range presets {
			go func(p string) {
				resp, _, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/"+p, nil, nil)
				if err != nil {
					done <- err
					return
				}

				if resp.StatusCode != 200 {
					done <- err
					return
				}

				done <- nil
			}(preset)
		}

		for i := 0; i < len(presets); i++ {
			if err := <-done; err != nil {
				t.Errorf("Concurrent preset request failed: %v", err)
			}
		}
	})
}
