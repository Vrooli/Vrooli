package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestBuildPrompt(t *testing.T) {
	draft := Draft{
		ID:         "test-id",
		EntityType: "scenario",
		EntityName: "test-scenario",
		Content:    "# Test PRD\n\nSome content here",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Status:     "draft",
	}

	tests := []struct {
		name    string
		draft   Draft
		section string
		context string
		wantLen int // Approximate minimum length
	}{
		{
			name:    "basic prompt without context",
			draft:   draft,
			section: "Executive Summary",
			context: "",
			wantLen: 200,
		},
		{
			name:    "prompt with context",
			draft:   draft,
			section: "Technical Architecture",
			context: "Focus on microservices and API design",
			wantLen: 250,
		},
		{
			name: "prompt with empty draft content",
			draft: Draft{
				ID:         "empty-id",
				EntityType: "resource",
				EntityName: "test-resource",
				Content:    "",
			},
			section: "Success Metrics",
			context: "",
			wantLen: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPrompt(tt.draft, tt.section, tt.context, "")

			if len(result) < tt.wantLen {
				t.Errorf("buildPrompt() length = %d, want at least %d", len(result), tt.wantLen)
			}

			// Verify prompt contains key elements
			if !contains(result, tt.draft.EntityType) {
				t.Errorf("buildPrompt() missing entity_type %q", tt.draft.EntityType)
			}
			if !contains(result, tt.draft.EntityName) {
				t.Errorf("buildPrompt() missing entity_name %q", tt.draft.EntityName)
			}
			if !contains(result, tt.section) {
				t.Errorf("buildPrompt() missing section %q", tt.section)
			}

			if tt.context != "" && !contains(result, tt.context) {
				t.Errorf("buildPrompt() missing context %q", tt.context)
			}
		})
	}
}

func TestHandleAIGenerateSectionValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    any
		expectStatus   int
		expectContains string
	}{
		{
			name: "missing section field",
			requestBody: AIGenerateRequest{
				Section: "",
				Context: "some context",
			},
			expectStatus:   http.StatusBadRequest,
			expectContains: "Section is required",
		},
		{
			name:           "invalid JSON",
			requestBody:    "not a valid json",
			expectStatus:   http.StatusBadRequest,
			expectContains: "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			switch v := tt.requestBody.(type) {
			case string:
				body = []byte(v)
			default:
				body, err = json.Marshal(v)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts/test-id/ai/generate-section", bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"id": "test-draft-id"})

			w := httptest.NewRecorder()

			// Call handler directly (validation happens before DB access)
			handleAIGenerateSection(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("handleAIGenerateSection() status = %d, want %d, body: %s", w.Code, tt.expectStatus, w.Body.String())
			}

			if !contains(w.Body.String(), tt.expectContains) {
				t.Errorf("handleAIGenerateSection() body missing %q, got %q", tt.expectContains, w.Body.String())
			}
		})
	}
}

// TestHandleAIGenerateSectionNoDB is in handlers_test.go to avoid duplication

func TestGenerateAIContentCLI(t *testing.T) {
	draft := Draft{
		ID:         "test-id",
		EntityType: "scenario",
		EntityName: "test-scenario",
		Content:    "# Test Content",
	}

	// This test will fail if resource-openrouter is not installed,
	// which is expected behavior
	_, _, err := generateAIContentCLI(draft, "Executive Summary", "Test context", "")

	// We expect an error because resource-openrouter is likely not available in test environment
	if err == nil {
		t.Log("generateAIContentCLI() unexpectedly succeeded - resource-openrouter may be installed")
	} else {
		// Verify error message is informative
		if !contains(err.Error(), "resource-openrouter") {
			t.Errorf("generateAIContentCLI() error should mention resource-openrouter, got: %v", err)
		}
	}
}

func TestGenerateAIContentHTTP(t *testing.T) {
	// Set test API key
	oldKey := os.Getenv("OPENROUTER_API_KEY")
	os.Setenv("OPENROUTER_API_KEY", "test-key")
	defer func() {
		if oldKey != "" {
			os.Setenv("OPENROUTER_API_KEY", oldKey)
		} else {
			os.Unsetenv("OPENROUTER_API_KEY")
		}
	}()

	// Create a mock OpenRouter server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("Unexpected request method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Verify Content-Type header
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Missing or incorrect Content-Type header")
		}

		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-key" {
			t.Errorf("Expected Authorization header 'Bearer test-key', got '%s'", authHeader)
		}

		// Return mock response
		response := map[string]any{
			"model": "anthropic/claude-3.5-sonnet",
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "Generated executive summary content",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	draft := Draft{
		ID:         "test-id",
		EntityType: "scenario",
		EntityName: "test-scenario",
		Content:    "# Test PRD",
	}

	content, model, err := generateAIContentHTTP(mockServer.URL, draft, "Executive Summary", "Test context", "")

	if err != nil {
		t.Errorf("generateAIContentHTTP() unexpected error: %v", err)
	}

	if content != "Generated executive summary content" {
		t.Errorf("generateAIContentHTTP() content = %q, want %q", content, "Generated executive summary content")
	}

	if model != "anthropic/claude-3.5-sonnet" {
		t.Errorf("generateAIContentHTTP() model = %q, want %q", model, "anthropic/claude-3.5-sonnet")
	}
}

func TestGenerateAIContentHTTPError(t *testing.T) {
	// Set test API key
	oldKey := os.Getenv("OPENROUTER_API_KEY")
	os.Setenv("OPENROUTER_API_KEY", "test-key")
	defer func() {
		if oldKey != "" {
			os.Setenv("OPENROUTER_API_KEY", oldKey)
		} else {
			os.Unsetenv("OPENROUTER_API_KEY")
		}
	}()

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectContains string
	}{
		{
			name: "server returns error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal server error"))
			},
			expectContains: "OpenRouter API returned error",
		},
		{
			name: "invalid JSON response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte("not valid json"))
			},
			expectContains: "failed to decode response",
		},
		{
			name: "missing choices in response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"model": "test-model",
				})
			},
			expectContains: "no choices in response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(tt.serverHandler)
			defer mockServer.Close()

			draft := Draft{
				ID:         "test-id",
				EntityType: "scenario",
				EntityName: "test-scenario",
				Content:    "# Test",
			}

			_, _, err := generateAIContentHTTP(mockServer.URL, draft, "Executive Summary", "", "")

			if err == nil {
				t.Error("generateAIContentHTTP() expected error, got nil")
			}

			if !contains(err.Error(), tt.expectContains) {
				t.Errorf("generateAIContentHTTP() error should contain %q, got: %v", tt.expectContains, err)
			}
		})
	}
}

func TestGenerateAIContent(t *testing.T) {
	draft := Draft{
		ID:         "test-id",
		EntityType: "scenario",
		EntityName: "test-scenario",
		Content:    "# Test",
	}

	// Test with no RESOURCE_OPENROUTER_URL set (should fallback to CLI)
	t.Setenv("RESOURCE_OPENROUTER_URL", "")

	_, _, err := generateAIContent(draft, "Executive Summary", "", "")

	// Should attempt CLI and fail (unless resource-openrouter is installed)
	if err == nil {
		t.Log("generateAIContent() succeeded with CLI - resource-openrouter may be installed")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && bytes.Contains([]byte(s), []byte(substr))))
}
