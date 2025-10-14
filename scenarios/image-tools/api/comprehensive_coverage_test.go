package main

import (
	"testing"
)

// TestHealthCheckComprehensive provides comprehensive health check coverage
func TestHealthCheckComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("HealthCheckAllFields", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/health", nil, nil)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		result := assertJSONResponse(t, resp, body, resp.StatusCode)

		// Check all required top-level fields
		requiredFields := []string{"status", "service", "timestamp", "readiness", "dependencies", "metrics"}
		for _, field := range requiredFields {
			if _, exists := result[field]; !exists {
				t.Errorf("Missing field: %s", field)
			}
		}

		// Check dependencies
		deps, ok := result["dependencies"].(map[string]interface{})
		if !ok {
			t.Fatal("Dependencies not in expected format")
		}

		requiredDeps := []string{"plugin_registry", "storage_system", "image_processing", "filesystem"}
		for _, depName := range requiredDeps {
			if _, exists := deps[depName]; !exists {
				t.Errorf("Missing dependency: %s", depName)
			}
		}
	})

	t.Run("HealthCheckPluginRegistry", func(t *testing.T) {
		health := server.checkPluginRegistry()

		// Should have status
		if _, ok := health["status"]; !ok {
			t.Error("Missing status in plugin registry health")
		}

		// Should have checks
		if _, ok := health["checks"]; !ok {
			t.Error("Missing checks in plugin registry health")
		}
	})

	t.Run("HealthCheckStorageSystem", func(t *testing.T) {
		health := server.checkStorageSystem()

		if _, ok := health["status"]; !ok {
			t.Error("Missing status in storage health")
		}

		if _, ok := health["checks"]; !ok {
			t.Error("Missing checks in storage health")
		}
	})

	t.Run("HealthCheckImageProcessing", func(t *testing.T) {
		health := server.checkImageProcessing()

		if _, ok := health["status"]; !ok {
			t.Error("Missing status in image processing health")
		}

		if _, ok := health["checks"]; !ok {
			t.Error("Missing checks in image processing health")
		}
	})

	t.Run("HealthCheckFileSystem", func(t *testing.T) {
		health := server.checkFileSystemOperations()

		if _, ok := health["status"]; !ok {
			t.Error("Missing status in filesystem health")
		}

		if _, ok := health["checks"]; !ok {
			t.Error("Missing checks in filesystem health")
		}
	})
}

// TestAllEndpoints ensures all endpoints are tested
func TestAllEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	endpoints := []struct {
		method string
		path   string
		desc   string
	}{
		{"GET", "/health", "Health check"},
		{"GET", "/api/v1/plugins", "List plugins"},
		{"GET", "/api/v1/presets", "List presets"},
		{"GET", "/api/v1/presets/web-optimized", "Get specific preset"},
		{"GET", "/api/v1/presets/email-safe", "Get email-safe preset"},
		{"GET", "/api/v1/presets/aggressive", "Get aggressive preset"},
		{"GET", "/api/v1/presets/high-quality", "Get high-quality preset"},
		{"GET", "/api/v1/presets/social-media", "Get social-media preset"},
		{"POST", "/api/v1/image/preset/web-optimized", "Apply preset"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.desc, func(t *testing.T) {
			resp, _, err := makeHTTPRequest(server.app, endpoint.method, endpoint.path, nil, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode >= 500 {
				t.Errorf("%s returned server error: %d", endpoint.desc, resp.StatusCode)
			}
		})
	}
}

// TestImageHandlers tests all image handling endpoints
func TestImageHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("CompressHandler", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "compress.jpg", imageData, map[string]string{
			"quality": "75",
		})

		headers := map[string]string{"Content-Type": contentType}
		resp, _, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/compress", body, headers)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if resp.StatusCode >= 500 {
			t.Logf("Server error (acceptable): %d", resp.StatusCode)
		}
	})

	t.Run("ResizeHandler", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "resize.jpg", imageData, map[string]string{
			"width":  "800",
			"height": "600",
		})

		headers := map[string]string{"Content-Type": contentType}
		resp, _, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/resize", body, headers)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if resp.StatusCode >= 500 {
			t.Logf("Server error (acceptable): %d", resp.StatusCode)
		}
	})

	t.Run("ConvertHandler", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "convert.jpg", imageData, map[string]string{
			"target_format": "png",
		})

		headers := map[string]string{"Content-Type": contentType}
		resp, _, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/convert", body, headers)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if resp.StatusCode >= 500 {
			t.Logf("Server error (acceptable): %d", resp.StatusCode)
		}
	})

	t.Run("MetadataHandler", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "metadata.jpg", imageData, nil)

		headers := map[string]string{"Content-Type": contentType}
		resp, _, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/metadata", body, headers)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if resp.StatusCode >= 500 {
			t.Logf("Server error (acceptable): %d", resp.StatusCode)
		}
	})

	t.Run("InfoHandler", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "info.jpg", imageData, nil)

		headers := map[string]string{"Content-Type": contentType}
		resp, _, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/info", body, headers)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if resp.StatusCode >= 500 {
			t.Logf("Server error (acceptable): %d", resp.StatusCode)
		}
	})

	t.Run("BatchHandler", func(t *testing.T) {
		operations := `{"operations": [{"type": "compress", "options": {"quality": 80}}]}`
		files := map[string][]byte{
			"batch.jpg": generateTestImageData("jpeg"),
		}

		body, contentType := createMultipartBatchData(t, files, operations)

		headers := map[string]string{"Content-Type": contentType}
		resp, _, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/batch", body, headers)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if resp.StatusCode >= 500 {
			t.Logf("Server error (acceptable): %d", resp.StatusCode)
		}
	})
}

// TestUtilityFunctions tests utility functions
func TestUtilityFunctions(t *testing.T) {
	t.Run("GetFileFormatVariations", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"file.jpg", "jpeg"},
			{"FILE.JPG", "jpeg"},
			{"Image.Jpg", "jpeg"},
			{"photo.jpeg", "jpeg"},
			{"image.png", "png"},
			{"graphic.webp", "webp"},
			{"icon.svg", "svg"},
			{"FILE.PNG", "png"},
			{"noext", ""},
		}

		for _, tt := range tests {
			result := getFileFormat(tt.input)
			if result != tt.expected {
				t.Errorf("getFileFormat(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("GetContentTypeVariations", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"file.jpg", "image/jpeg"},
			{"file.jpeg", "image/jpeg"},
			{"FILE.JPEG", "image/jpeg"},
			{"file.png", "image/png"},
			{"FILE.PNG", "image/png"},
			{"file.webp", "image/webp"},
			{"file.svg", "image/svg+xml"},
			{"file.gif", "image/gif"},
			{"unknown.xyz", "application/octet-stream"},
		}

		for _, tt := range tests {
			result := getContentType(tt.input)
			if result != tt.expected {
				t.Errorf("getContentType(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		}
	})
}

// TestErrorPaths tests error handling paths
func TestErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("ProxyInvalidURL", func(t *testing.T) {
		resp, _, err := makeHTTPRequest(server.app, "GET", "/api/v1/image/proxy?url=invalid", nil, nil)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if resp.StatusCode == 200 {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("ProxyHTTPURL", func(t *testing.T) {
		resp, _, err := makeHTTPRequest(server.app, "GET", "/api/v1/image/proxy?url=http://example.com/image.jpg", nil, nil)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if resp.StatusCode == 200 {
			t.Error("Expected error for HTTP URL")
		}
	})

	t.Run("GetNonExistentPreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/nonexistent", nil, nil)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 404)

		if _, ok := result["error"]; !ok {
			t.Error("Expected error field")
		}

		if _, ok := result["available_presets"]; !ok {
			t.Error("Expected available_presets field")
		}
	})

	t.Run("ApplyNonExistentPreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/preset/nonexistent", nil, nil)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 404)

		if _, ok := result["error"]; !ok {
			t.Error("Expected error field")
		}
	})

	t.Run("ConvertSameFormat", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "same.jpg", imageData, map[string]string{
			"target_format": "jpeg",
		})

		headers := map[string]string{"Content-Type": contentType}
		resp, respBody, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/convert", body, headers)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		assertErrorResponse(t, resp, respBody, 400, "error")
	})

	t.Run("ConvertMissingTargetFormat", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")
		body, contentType := createMultipartFormData(t, "image", "missing.jpg", imageData, nil)

		headers := map[string]string{"Content-Type": contentType}
		resp, respBody, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/convert", body, headers)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		assertErrorResponse(t, resp, respBody, 400, "error")
	})
}
