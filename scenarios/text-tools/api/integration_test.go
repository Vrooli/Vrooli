package main

import (
	"net/http"
	"testing"
)

// Integration tests covering combined workflows

func TestEndToEndDiffWorkflow(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	// Test complete diff workflow with various options
	testCases := []struct {
		name    string
		text1   string
		text2   string
		options DiffOptions
	}{
		{
			name:  "Simple Text Diff",
			text1: "Hello World",
			text2: "Hello Universe",
			options: DiffOptions{
				Type: "line",
			},
		},
		{
			name:  "Case Insensitive Diff",
			text1: "HELLO",
			text2: "hello",
			options: DiffOptions{
				Type:       "line",
				IgnoreCase: true,
			},
		},
		{
			name:  "Whitespace Ignored",
			text1: "  Hello  World  ",
			text2: "Hello World",
			options: DiffOptions{
				Type:             "line",
				IgnoreWhitespace: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := DiffRequest{
				Text1:   tc.text1,
				Text2:   tc.text2,
				Options: tc.options,
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
				t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestEndToEndSearchWorkflow(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	testCases := []struct {
		name    string
		text    string
		pattern string
		options SearchOptions
	}{
		{
			name:    "Basic Search",
			text:    "The quick brown fox jumps over the lazy dog",
			pattern: "fox",
			options: SearchOptions{},
		},
		{
			name:    "Regex Search",
			text:    "test123 test456 test",
			pattern: "test[0-9]+",
			options: SearchOptions{
				Regex: true,
			},
		},
		{
			name:    "Case Sensitive",
			text:    "Hello hello HELLO",
			pattern: "hello",
			options: SearchOptions{
				CaseSensitive: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := SearchRequest{
				Text:    tc.text,
				Pattern: tc.pattern,
				Options: tc.options,
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

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		})
	}
}

func TestEndToEndTransformWorkflow(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	// Test chained transformations
	t.Run("ChainedTransforms", func(t *testing.T) {
		reqBody := TransformRequest{
			Text: "hello world",
			Transformations: []Transformation{
				{Type: "upper"},
				{Type: "lower"},
				{Type: "title"},
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
			if applied, ok := response["transformations_applied"].([]interface{}); ok {
				if len(applied) != 3 {
					t.Errorf("Expected 3 transformations applied, got %d", len(applied))
				}
			}
		}
	})

	// Test encoding transforms
	t.Run("EncodingTransforms", func(t *testing.T) {
		reqBody := TransformRequest{
			Text: "hello world",
			Transformations: []Transformation{
				{
					Type:       "encode",
					Parameters: map[string]interface{}{"type": "base64"},
				},
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

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test sanitization
	t.Run("SanitizeHTML", func(t *testing.T) {
		reqBody := TransformRequest{
			Text: "<p>Hello <b>World</b></p>",
			Transformations: []Transformation{
				{Type: "sanitize"},
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
				if result != "Hello World" {
					t.Errorf("Expected 'Hello World', got '%s'", result)
				}
			}
		}
	})
}

func TestEndToEndAnalyzeWorkflow(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("MultipleAnalyses", func(t *testing.T) {
		reqBody := AnalyzeRequest{
			Text: "Contact me at test@example.com. This is a great product! Visit https://example.com for more.",
			Analyses: []string{"entities", "sentiment", "statistics", "keywords", "language"},
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
			// Should have entities (email, URL)
			if entities, ok := response["entities"].([]interface{}); ok {
				if len(entities) == 0 {
					t.Error("Expected to find entities")
				}
			}

			// Should have sentiment
			if _, ok := response["sentiment"].(map[string]interface{}); !ok {
				t.Error("Expected sentiment in response")
			}

			// Should have statistics
			if _, ok := response["statistics"].(map[string]interface{}); !ok {
				t.Error("Expected statistics in response")
			}

			// Should have keywords (may be empty array or populated)
			if keywords, ok := response["keywords"]; ok {
				if keywords == nil {
					t.Error("Expected keywords field (even if empty)")
				}
			}

			// Should have language
			if _, ok := response["language"].(map[string]interface{}); !ok {
				t.Error("Expected language in response")
			}
		}
	})

	t.Run("SummaryGeneration", func(t *testing.T) {
		reqBody := AnalyzeRequest{
			Text: "This is a long text that should be summarized into a shorter version for easy reading",
			Analyses: []string{"summary"},
			Options: AnalyzeOptions{
				SummaryLength: 5,
			},
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
			if summary, ok := response["summary"].(string); ok {
				if len(summary) == 0 {
					t.Error("Expected non-empty summary")
				}
			}
		}
	})
}

func TestErrorHandlingConsistency(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	// Test that all handlers return consistent error structure
	testCases := []struct {
		name    string
		handler http.HandlerFunc
		path    string
		body    interface{}
	}{
		{
			name:    "Diff Invalid JSON",
			handler: env.Server.DiffHandlerV1,
			path:    "/api/v1/diff",
			body:    `{"invalid": json}`,
		},
		{
			name:    "Search Invalid JSON",
			handler: env.Server.SearchHandlerV1,
			path:    "/api/v1/search",
			body:    `{"invalid": json}`,
		},
		{
			name:    "Transform Invalid JSON",
			handler: env.Server.TransformHandlerV1,
			path:    "/api/v1/transform",
			body:    `{"invalid": json}`,
		},
		{
			name:    "Analyze Invalid JSON",
			handler: env.Server.AnalyzeHandlerV1,
			path:    "/api/v1/analyze",
			body:    `{"invalid": json}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   tc.path,
				Body:   tc.body,
			}

			w, err := makeHTTPRequest(req, tc.handler)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// All should return 400
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			// All should have error structure
			assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
		})
	}
}
