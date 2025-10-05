package main

import (
	"net/http"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, env.Server.HealthHandler)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response != nil {
			Validator.ValidateRequiredFields(t, response, []string{"status", "timestamp", "database", "resources", "version"})

			// Validate version
			if version, ok := response["version"].(string); ok {
				if version != "1.0.0" {
					t.Errorf("Expected version 1.0.0, got %s", version)
				}
			}
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, _ := makeHTTPRequest(req, env.Server.HealthHandler)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})
}

func TestResourcesHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/resources",
		}

		w, err := makeHTTPRequest(req, env.Server.ResourcesHandler)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateRequiredFields(t, response, []string{"timestamp", "resources", "metrics"})
		}
	})
}

func TestDiffHandlerV1(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_SimpleDiff", func(t *testing.T) {
		reqBody := DiffRequest{
			Text1: "hello",
			Text2: "world",
			Options: DiffOptions{
				Type: "line",
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/diff",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.DiffHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateRequiredFields(t, response, []string{"changes", "similarity_score", "summary"})
		}
	})

	t.Run("Success_IgnoreCase", func(t *testing.T) {
		reqBody := DiffRequest{
			Text1: "HELLO",
			Text2: "hello",
			Options: DiffOptions{
				Type:       "line",
				IgnoreCase: true,
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/diff",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.DiffHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		// When ignoring case, "HELLO" and "hello" should be similar
		if response != nil {
			if changes, ok := response["changes"].([]interface{}); ok {
				if len(changes) > 0 {
					t.Errorf("Expected no changes with ignore case, got %d changes", len(changes))
				}
			}
		}
	})

	t.Run("Error_MissingText1", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"text2": "world",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/diff",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.DiffHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Both text1 and text2 are required")
	})

	t.Run("Error_MissingText2", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"text1": "hello",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/diff",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.DiffHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Both text1 and text2 are required")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/diff",
			Body:   `{"invalid": json}`,
		}

		w, err := makeHTTPRequest(req, env.Server.DiffHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("DiffTypes", func(t *testing.T) {
		diffTypes := []string{"line", "word", "character"}

		for _, diffType := range diffTypes {
			t.Run(diffType, func(t *testing.T) {
				reqBody := DiffRequest{
					Text1: "hello world",
					Text2: "goodbye world",
					Options: DiffOptions{
						Type: diffType,
					},
				}

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/diff",
					Body:   reqBody,
				}

				w, err := makeHTTPRequest(req, env.Server.DiffHandlerV1)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 for diff type %s, got %d", diffType, w.Code)
				}
			})
		}
	})
}

func TestSearchHandlerV1(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_SimpleSearch", func(t *testing.T) {
		reqBody := SearchRequest{
			Text:    "hello world\ntest line\nhello again",
			Pattern: "hello",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.SearchHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateRequiredFields(t, response, []string{"matches", "total_matches"})

			if totalMatches, ok := response["total_matches"].(float64); ok {
				if totalMatches != 2 {
					t.Errorf("Expected 2 matches for 'hello', got %v", totalMatches)
				}
			}
		}
	})

	t.Run("Success_CaseSensitive", func(t *testing.T) {
		reqBody := SearchRequest{
			Text:    "Hello HELLO hello",
			Pattern: "hello",
			Options: SearchOptions{
				CaseSensitive: true,
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.SearchHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if totalMatches, ok := response["total_matches"].(float64); ok {
				if totalMatches != 1 {
					t.Errorf("Expected 1 case-sensitive match, got %v", totalMatches)
				}
			}
		}
	})

	t.Run("Error_MissingPattern", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"text": "hello world",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.SearchHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Pattern is required")
	})

	t.Run("Success_RegexSearch", func(t *testing.T) {
		reqBody := SearchRequest{
			Text:    "test123 test456 test",
			Pattern: "test[0-9]+",
			Options: SearchOptions{
				Regex: true,
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.SearchHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if totalMatches, ok := response["total_matches"].(float64); ok {
				if totalMatches != 2 {
					t.Errorf("Expected 2 regex matches, got %v", totalMatches)
				}
			}
		}
	})
}

func TestTransformHandlerV1(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_Uppercase", func(t *testing.T) {
		reqBody := TransformRequest{
			Text: "hello world",
			Transformations: []Transformation{
				{Type: "upper"},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/transform",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.TransformHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if result, ok := response["result"].(string); ok {
				if result != "HELLO WORLD" {
					t.Errorf("Expected 'HELLO WORLD', got '%s'", result)
				}
			}
		}
	})

	t.Run("Success_MultipleTransformations", func(t *testing.T) {
		reqBody := TransformRequest{
			Text: "Hello World",
			Transformations: []Transformation{
				{Type: "lower"},
				{Type: "upper"},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/transform",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.TransformHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if result, ok := response["result"].(string); ok {
				if result != "HELLO WORLD" {
					t.Errorf("Expected 'HELLO WORLD' after chained transforms, got '%s'", result)
				}
			}
		}
	})

	t.Run("Error_MissingText", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"transformations": []interface{}{
				map[string]interface{}{"type": "upper"},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/transform",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.TransformHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Text is required")
	})
}

func TestAnalyzeHandlerV1(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_Statistics", func(t *testing.T) {
		reqBody := AnalyzeRequest{
			Text:     "Hello world. This is a test.",
			Analyses: []string{"statistics"},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/analyze",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.AnalyzeHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if stats, ok := response["statistics"].(map[string]interface{}); ok {
				if wordCount, ok := stats["word_count"].(float64); ok {
					if wordCount != 6 {
						t.Errorf("Expected word count 6, got %v", wordCount)
					}
				}
			}
		}
	})

	t.Run("Success_Entities", func(t *testing.T) {
		reqBody := AnalyzeRequest{
			Text:     "Contact me at test@example.com or visit https://example.com",
			Analyses: []string{"entities"},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/analyze",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.AnalyzeHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if entities, ok := response["entities"].([]interface{}); ok {
				if len(entities) == 0 {
					t.Error("Expected to find entities in text")
				}
			}
		}
	})

	t.Run("Error_MissingText", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"analyses": []string{"statistics"},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/analyze",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.AnalyzeHandlerV1)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Text is required")
	})
}
