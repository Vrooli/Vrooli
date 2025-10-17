package main

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}, healthHandler)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "qr-code-generator",
		})

		if response != nil {
			// Verify features exist
			features, ok := response["features"].(map[string]interface{})
			if !ok {
				t.Error("Expected features map in response")
			} else {
				if features["generate"] != true {
					t.Error("Expected generate feature to be true")
				}
				if features["batch"] != true {
					t.Error("Expected batch feature to be true")
				}
				if features["formats"] != true {
					t.Error("Expected formats feature to be true")
				}
			}

			// Verify timestamp exists
			if _, exists := response["timestamp"]; !exists {
				t.Error("Expected timestamp in response")
			}
		}
	})
}

// TestGenerateHandler tests the QR code generation endpoint
func TestGenerateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_BasicGeneration", func(t *testing.T) {
		req := GenerateRequest{
			Text: "Hello World",
			Size: 256,
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   req,
		}, generateHandler)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
			"format":  "base64",
		})

		if response != nil {
			// Verify data field contains base64 encoded PNG
			data, ok := response["data"].(string)
			if !ok {
				t.Fatal("Expected data field to be string")
			}

			// Verify it's valid base64
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				t.Errorf("Failed to decode base64 data: %v", err)
			}

			// Verify it starts with PNG magic bytes
			if len(decoded) < 8 || string(decoded[1:4]) != "PNG" {
				t.Error("Decoded data is not a valid PNG")
			}
		}
	})

	t.Run("Success_WithCustomSize", func(t *testing.T) {
		req := GenerateRequest{
			Text: "Test QR Code",
			Size: 512,
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   req,
		}, generateHandler)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			data, ok := response["data"].(string)
			if !ok || data == "" {
				t.Error("Expected non-empty data field")
			}
		}
	})

	t.Run("Success_WithErrorCorrection", func(t *testing.T) {
		errorLevels := []string{"Low", "Medium", "High", "Highest"}

		for _, level := range errorLevels {
			t.Run("ErrorCorrection_"+level, func(t *testing.T) {
				req := GenerateRequest{
					Text:            "Test",
					Size:            256,
					ErrorCorrection: level,
				}

				w := makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/generate",
					Body:   req,
				}, generateHandler)

				assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
					"success": true,
				})
			})
		}
	})

	t.Run("Success_DefaultSize", func(t *testing.T) {
		req := GenerateRequest{
			Text: "Default size test",
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   req,
		}, generateHandler)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})

	t.Run("Error_MissingText", func(t *testing.T) {
		req := GenerateRequest{
			Size: 256,
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   req,
		}, generateHandler)

		assertErrorResponse(t, w, http.StatusBadRequest, "Text is required")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   `{"text": "test"`, // Invalid JSON
		}, generateHandler)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/generate",
		}, generateHandler)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	t.Run("EdgeCase_EmptyText", func(t *testing.T) {
		req := GenerateRequest{
			Text: "",
			Size: 256,
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   req,
		}, generateHandler)

		assertErrorResponse(t, w, http.StatusBadRequest, "Text is required")
	})

	t.Run("EdgeCase_LongText", func(t *testing.T) {
		longText := strings.Repeat("A", 2000)
		req := GenerateRequest{
			Text: longText,
			Size: 512,
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   req,
		}, generateHandler)

		// Should still succeed or fail gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("EdgeCase_SpecialCharacters", func(t *testing.T) {
		req := GenerateRequest{
			Text: "特殊文字 🎉 émojis & symbols: @#$%",
			Size: 256,
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   req,
		}, generateHandler)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})
}

// TestBatchHandler tests the batch QR code generation endpoint
func TestBatchHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_MultipleCodes", func(t *testing.T) {
		items := []BatchItem{
			{Text: "Item 1", Label: "First"},
			{Text: "Item 2", Label: "Second"},
			{Text: "Item 3", Label: "Third"},
		}

		req := BatchRequest{
			Items: items,
			Options: GenerateRequest{
				Size: 256,
			},
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   req,
		}, batchHandler)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			results, ok := response["results"].([]interface{})
			if !ok {
				t.Fatal("Expected results to be an array")
			}

			if len(results) != 3 {
				t.Errorf("Expected 3 results, got %d", len(results))
			}

			// Verify each result
			for i, result := range results {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Result %d is not a map", i)
					continue
				}

				if resultMap["success"] != true {
					t.Errorf("Result %d should be successful", i)
				}

				data, ok := resultMap["data"].(string)
				if !ok || data == "" {
					t.Errorf("Result %d missing data field", i)
				}
			}
		}
	})

	t.Run("Success_SingleItem", func(t *testing.T) {
		items := []BatchItem{
			{Text: "Single Item", Label: "One"},
		}

		req := BatchRequest{
			Items: items,
			Options: GenerateRequest{
				Size: 256,
			},
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   req,
		}, batchHandler)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			results, ok := response["results"].([]interface{})
			if !ok || len(results) != 1 {
				t.Error("Expected exactly 1 result")
			}
		}
	})

	t.Run("Success_WithCustomOptions", func(t *testing.T) {
		items := []BatchItem{
			{Text: "Custom 1", Label: "First"},
			{Text: "Custom 2", Label: "Second"},
		}

		req := BatchRequest{
			Items: items,
			Options: GenerateRequest{
				Size:            512,
				ErrorCorrection: "High",
			},
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   req,
		}, batchHandler)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})

	t.Run("Error_EmptyItems", func(t *testing.T) {
		req := BatchRequest{
			Items: []BatchItem{},
			Options: GenerateRequest{
				Size: 256,
			},
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   req,
		}, batchHandler)

		assertErrorResponse(t, w, http.StatusBadRequest, "Items array is required")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   `{"items": [`, // Invalid JSON
		}, batchHandler)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/batch",
		}, batchHandler)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	t.Run("EdgeCase_ManyItems", func(t *testing.T) {
		items := make([]BatchItem, 50)
		for i := 0; i < 50; i++ {
			items[i] = BatchItem{
				Text:  "Item " + string(rune(i)),
				Label: "Label " + string(rune(i)),
			}
		}

		req := BatchRequest{
			Items: items,
			Options: GenerateRequest{
				Size: 256,
			},
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   req,
		}, batchHandler)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			results, ok := response["results"].([]interface{})
			if !ok || len(results) != 50 {
				t.Errorf("Expected 50 results, got %d", len(results))
			}
		}
	})

	t.Run("PartialFailure_MixedValidInvalid", func(t *testing.T) {
		items := []BatchItem{
			{Text: "Valid 1", Label: "Good"},
			{Text: "", Label: "Empty"},  // Will fail
			{Text: "Valid 2", Label: "Good"},
		}

		req := BatchRequest{
			Items: items,
			Options: GenerateRequest{
				Size: 256,
			},
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   req,
		}, batchHandler)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			results, ok := response["results"].([]interface{})
			if !ok || len(results) != 3 {
				t.Fatal("Expected 3 results")
			}

			// First should succeed
			if results[0].(map[string]interface{})["success"] != true {
				t.Error("First item should succeed")
			}

			// Second should fail (empty text)
			if results[1].(map[string]interface{})["success"] != false {
				t.Error("Second item should fail")
			}

			// Third should succeed
			if results[2].(map[string]interface{})["success"] != true {
				t.Error("Third item should succeed")
			}
		}
	})
}

// TestFormatsHandler tests the formats information endpoint
func TestFormatsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/formats",
		}, formatsHandler)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Verify formats array exists
			formats, ok := response["formats"].([]interface{})
			if !ok {
				t.Fatal("Expected formats array")
			}

			if len(formats) < 1 {
				t.Error("Expected at least one format")
			}

			// Verify sizes array exists
			sizes, ok := response["sizes"].([]interface{})
			if !ok {
				t.Fatal("Expected sizes array")
			}

			if len(sizes) < 1 {
				t.Error("Expected at least one size")
			}

			// Verify error corrections array exists
			errorCorrections, ok := response["errorCorrections"].([]interface{})
			if !ok {
				t.Fatal("Expected errorCorrections array")
			}

			if len(errorCorrections) < 1 {
				t.Error("Expected at least one error correction level")
			}
		}
	})
}

// TestGenerateQRCode tests the internal QR code generation function
func TestGenerateQRCode(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_BasicGeneration", func(t *testing.T) {
		req := GenerateRequest{
			Text:            "Test",
			Size:            256,
			ErrorCorrection: "Medium",
		}

		response, err := generateQRCode(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}

		if response.Data == "" {
			t.Error("Expected non-empty data")
		}

		if response.Format != "base64" {
			t.Errorf("Expected format 'base64', got '%s'", response.Format)
		}

		// Verify base64 validity
		_, err = base64.StdEncoding.DecodeString(response.Data)
		if err != nil {
			t.Errorf("Invalid base64 data: %v", err)
		}
	})

	t.Run("Success_DifferentErrorLevels", func(t *testing.T) {
		levels := []string{"Low", "L", "Medium", "M", "High", "H", "Highest", "Q"}

		for _, level := range levels {
			t.Run("Level_"+level, func(t *testing.T) {
				req := GenerateRequest{
					Text:            "Test",
					Size:            256,
					ErrorCorrection: level,
				}

				response, err := generateQRCode(req)
				if err != nil {
					t.Errorf("Error with level %s: %v", level, err)
				}

				if !response.Success {
					t.Errorf("Failed with level %s", level)
				}
			})
		}
	})

	t.Run("Success_DifferentSizes", func(t *testing.T) {
		sizes := []int{128, 256, 512, 1024}

		for _, size := range sizes {
			t.Run("Size_"+string(rune(size)), func(t *testing.T) {
				req := GenerateRequest{
					Text:            "Test",
					Size:            size,
					ErrorCorrection: "Medium",
				}

				response, err := generateQRCode(req)
				if err != nil {
					t.Errorf("Error with size %d: %v", size, err)
				}

				if !response.Success {
					t.Errorf("Failed with size %d", size)
				}
			})
		}
	})

	t.Run("EdgeCase_UnicodeText", func(t *testing.T) {
		req := GenerateRequest{
			Text:            "日本語 中文 한국어 العربية",
			Size:            256,
			ErrorCorrection: "High",
		}

		response, err := generateQRCode(req)
		if err != nil {
			t.Fatalf("Unexpected error with unicode: %v", err)
		}

		if !response.Success {
			t.Error("Expected success with unicode text")
		}
	})

	t.Run("EdgeCase_URL", func(t *testing.T) {
		req := GenerateRequest{
			Text:            "https://example.com/path?query=value&foo=bar",
			Size:            256,
			ErrorCorrection: "Medium",
		}

		response, err := generateQRCode(req)
		if err != nil {
			t.Fatalf("Unexpected error with URL: %v", err)
		}

		if !response.Success {
			t.Error("Expected success with URL")
		}
	})
}

// TestGetEnv tests the environment variable helper function
func TestGetEnv(t *testing.T) {
	t.Run("ExistingVariable", func(t *testing.T) {
		key := "TEST_VAR_EXISTS"
		expected := "test_value"
		os.Setenv(key, expected)
		defer os.Unsetenv(key)

		result := getEnv(key, "default")
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("MissingVariable", func(t *testing.T) {
		key := "TEST_VAR_MISSING"
		defaultValue := "default_value"

		result := getEnv(key, defaultValue)
		if result != defaultValue {
			t.Errorf("Expected %s, got %s", defaultValue, result)
		}
	})

	t.Run("EmptyVariable", func(t *testing.T) {
		key := "TEST_VAR_EMPTY"
		defaultValue := "default"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		result := getEnv(key, defaultValue)
		if result != defaultValue {
			t.Errorf("Expected default for empty var, got %s", result)
		}
	})
}

// TestErrorScenarios uses the pattern builder for systematic error testing
func TestErrorScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GenerateHandler_Errors", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "generateHandler",
			Handler:     generateHandler,
			BaseURL:     "/generate",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/generate").
			AddMethodNotAllowed("GET", "/generate").
			AddMethodNotAllowed("DELETE", "/generate").
			AddMethodNotAllowed("PUT", "/generate").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("BatchHandler_Errors", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "batchHandler",
			Handler:     batchHandler,
			BaseURL:     "/batch",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/batch").
			AddMethodNotAllowed("GET", "/batch").
			AddMethodNotAllowed("DELETE", "/batch").
			AddMethodNotAllowed("PUT", "/batch").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestGenerateQRCodeErrors tests error paths in QR code generation
func TestGenerateQRCodeErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Error_EmptyText", func(t *testing.T) {
		req := GenerateRequest{
			Text:            "",
			Size:            256,
			ErrorCorrection: "Medium",
		}

		response, err := generateQRCode(req)
		if err == nil {
			t.Error("Expected error for empty text")
		}
		if response.Success {
			t.Error("Expected success to be false")
		}
	})
}

// TestGenerateHandlerInternalError tests internal server error path
func TestGenerateHandlerInternalError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InternalError_VeryLargeSize", func(t *testing.T) {
		req := GenerateRequest{
			Text: "Test",
			Size: 100000, // Extremely large size may cause issues
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   req,
		}, generateHandler)

		// Should either succeed or fail gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})
}

// TestBatchHandlerDefaultOptions tests batch with no custom options
func TestBatchHandlerDefaultOptions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DefaultOptions_Success", func(t *testing.T) {
		items := []BatchItem{
			{Text: "Default 1", Label: "First"},
			{Text: "Default 2", Label: "Second"},
		}

		// Don't set options, let defaults apply
		req := BatchRequest{
			Items: items,
			Options: GenerateRequest{}, // Empty options
		}

		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   req,
		}, batchHandler)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})
}

// TestHandlerContentType verifies content-type headers
func TestHandlerContentType(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handlers := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		body    interface{}
	}{
		{"health", healthHandler, "GET", "/health", nil},
		{"formats", formatsHandler, "GET", "/formats", nil},
		{"generate", generateHandler, "POST", "/generate", GenerateRequest{Text: "test", Size: 256}},
	}

	for _, h := range handlers {
		t.Run("ContentType_"+h.name, func(t *testing.T) {
			w := makeHTTPRequest(HTTPTestRequest{
				Method: h.method,
				Path:   h.path,
				Body:   h.body,
			}, h.handler)

			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkGenerateQRCode(b *testing.B) {
	req := GenerateRequest{
		Text:            "Benchmark test",
		Size:            256,
		ErrorCorrection: "Medium",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generateQRCode(req)
	}
}

func BenchmarkBatchGeneration(b *testing.B) {
	items := make([]BatchItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = BatchItem{
			Text:  "Batch item " + string(rune(i)),
			Label: "Label " + string(rune(i)),
		}
	}

	req := BatchRequest{
		Items: items,
		Options: GenerateRequest{
			Size:            256,
			ErrorCorrection: "Medium",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/batch",
			Body:   req,
		}, batchHandler)
		_ = w
	}
}
