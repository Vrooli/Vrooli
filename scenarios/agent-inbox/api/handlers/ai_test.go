package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-inbox/services"
)

// =============================================================================
// Helper Function Tests
// =============================================================================
//
// These tests cover helper functions used in the AI handler flow.
// The main ChatComplete handler involves complex external dependencies
// (OpenRouter API, database, tool registry) and is tested via integration tests.

// TestIsStreamingRequest verifies query parameter parsing for streaming mode.
func TestIsStreamingRequest_True(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/chats/123/complete?stream=true", nil)
	if !isStreamingRequest(req) {
		t.Error("expected streaming=true when ?stream=true")
	}
}

func TestIsStreamingRequest_False(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"no param", "/api/v1/chats/123/complete"},
		{"stream=false", "/api/v1/chats/123/complete?stream=false"},
		{"stream=1", "/api/v1/chats/123/complete?stream=1"},
		{"stream empty", "/api/v1/chats/123/complete?stream="},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			if isStreamingRequest(req) {
				t.Errorf("expected streaming=false for %s", tc.url)
			}
		})
	}
}

// TestMapCompletionErrorToStatus verifies error mapping to HTTP status codes.
func TestMapCompletionErrorToStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "ErrChatNotFound",
			err:        services.ErrChatNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ErrNoMessages",
			err:        services.ErrNoMessages,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ErrDatabaseError",
			err:        services.ErrDatabaseError,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "ErrMessagesFailed",
			err:        services.ErrMessagesFailed,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "generic error",
			err:        http.ErrAbortHandler,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status := mapCompletionErrorToStatus(tc.err)
			if status != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, status)
			}
		})
	}
}

// TestForcedToolQueryParam verifies the force_tool query parameter is extracted.
// This is a simplified test - the actual forced tool handling is tested in
// services/completion_test.go via TestGetForcedToolDefinition_*.
func TestForcedToolQueryParam_Extraction(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"with force_tool", "/api/v1/chats/123/complete?force_tool=scenario:tool_name", "scenario:tool_name"},
		{"no force_tool", "/api/v1/chats/123/complete", ""},
		{"empty force_tool", "/api/v1/chats/123/complete?force_tool=", ""},
		{"with other params", "/api/v1/chats/123/complete?stream=true&force_tool=s:t", "s:t"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			got := req.URL.Query().Get("force_tool")
			if got != tc.expected {
				t.Errorf("expected force_tool=%q, got %q", tc.expected, got)
			}
		})
	}
}

// =============================================================================
// ChatComplete Handler Testing Notes
// =============================================================================
//
// The ChatComplete handler involves multiple external dependencies that make
// unit testing impractical:
//
// 1. Repository - for chat/message retrieval
// 2. ToolRegistry - for tool discovery and configuration
// 3. OpenRouter API - for AI completion
// 4. AsyncTracker - for async tool operations
//
// Instead, we test at two levels:
//
// 1. **Service-level tests** (services/completion_test.go):
//    - TestGetForcedToolDefinition_* tests forced tool handling
//    - TestGetToolsForOpenAI_* tests internal tool filtering
//    - TestBuildAsyncGuidanceMessage_* tests async guidance injection
//
// 2. **Integration tests** (services/completion_integration_test.go):
//    - Full flow tests with real (or stubbed) dependencies
//    - End-to-end validation of forced tool + async guidance behavior
//
// This approach provides better coverage than mocking every dependency,
// while keeping tests maintainable and meaningful.
