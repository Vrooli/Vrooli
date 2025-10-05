package main

import (
	"net/http"
	"testing"
)

func TestExtractHandlerV1(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_StringSource", func(t *testing.T) {
		reqBody := ExtractRequest{
			Source: "test content",
			Format: "text",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateRequiredFields(t, response, []string{"text", "metadata"})
		}
	})

	t.Run("Success_MapSource", func(t *testing.T) {
		reqBody := ExtractRequest{
			Source: map[string]interface{}{
				"text": "Extracted content",
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if text, ok := response["text"].(string); ok {
				if text != "Extracted content" {
					t.Errorf("Expected 'Extracted content', got '%s'", text)
				}
			}
		}
	})

	t.Run("Error_MissingSource", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"format": "text",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Source is required")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extract",
			Body:   `{"invalid": json}`,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})
}

func TestExtractRequestValidation(t *testing.T) {
	t.Run("WithOptions", func(t *testing.T) {
		reqBody := ExtractRequest{
			Source: "test",
			Options: ExtractOptions{
				OCR:                true,
				PreserveFormatting: true,
				ExtractMetadata:    true,
			},
		}

		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		env := setupTestServer(t)
		defer env.Cleanup()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("DocumentIDSource", func(t *testing.T) {
		reqBody := ExtractRequest{
			Source: map[string]interface{}{
				"document_id": "doc123",
			},
		}

		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		env := setupTestServer(t)
		defer env.Cleanup()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if metadata, ok := response["metadata"].(map[string]interface{}); ok {
				if metadata["type"] != "document" {
					t.Errorf("Expected type 'document', got '%v'", metadata["type"])
				}
			}
		}
	})

	t.Run("URLSource", func(t *testing.T) {
		reqBody := ExtractRequest{
			Source: map[string]interface{}{
				"url": "https://example.com",
			},
		}

		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		env := setupTestServer(t)
		defer env.Cleanup()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if metadata, ok := response["metadata"].(map[string]interface{}); ok {
				if metadata["type"] != "url" {
					t.Errorf("Expected type 'url', got '%v'", metadata["type"])
				}
			}
		}
	})

	t.Run("FileSource", func(t *testing.T) {
		reqBody := ExtractRequest{
			Source: map[string]interface{}{
				"file": "base64encodeddata",
			},
		}

		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		env := setupTestServer(t)
		defer env.Cleanup()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if metadata, ok := response["metadata"].(map[string]interface{}); ok {
				if metadata["type"] != "file" {
					t.Errorf("Expected type 'file', got '%v'", metadata["type"])
				}
			}
		}
	})
}
