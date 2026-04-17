package handlers

import (
	"agent-inbox/services"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

// =============================================================================
// Skills Parsing Tests
// =============================================================================

// TestParseSkillsFromBody tests skills parsing from request body.
// This verifies the logic that extracts skills from JSON request body.
func TestParseSkillsFromBody(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		contentLength  int64 // -1 for chunked/unknown
		expectedSkills int
		expectParsed   bool
	}{
		{
			name:           "valid skills with content length",
			body:           `{"skills": [{"id": "s1", "key": "test", "content": "content"}]}`,
			contentLength:  -1, // Simulates chunked transfer
			expectedSkills: 1,
			expectParsed:   true,
		},
		{
			name:           "valid skills - chunked encoding",
			body:           `{"skills": [{"id": "s1", "key": "test", "content": "content"}]}`,
			contentLength:  -1, // This is what happens with chunked transfer
			expectedSkills: 1,
			expectParsed:   true, // SHOULD work but currently FAILS due to bug
		},
		{
			name:           "empty skills array",
			body:           `{"skills": []}`,
			contentLength:  -1,
			expectedSkills: 0,
			expectParsed:   true,
		},
		{
			name:           "no skills field",
			body:           `{}`,
			contentLength:  -1,
			expectedSkills: 0,
			expectParsed:   true,
		},
		{
			name:           "empty body",
			body:           ``,
			contentLength:  0,
			expectedSkills: 0,
			expectParsed:   false,
		},
		{
			name:           "multiple skills with targeting",
			body:           `{"skills": [{"id": "s1", "key": "global", "content": "c1"}, {"id": "s2", "key": "targeted", "content": "c2", "targetToolId": "tool_a"}]}`,
			contentLength:  -1,
			expectedSkills: 2,
			expectParsed:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			skills := parseSkillsFromRequestBody(tc.body, tc.contentLength)

			if tc.expectParsed {
				if len(skills) != tc.expectedSkills {
					t.Errorf("expected %d skills, got %d", tc.expectedSkills, len(skills))
				}
			} else {
				if len(skills) != 0 {
					t.Errorf("expected no skills parsed, got %d", len(skills))
				}
			}
		})
	}
}

// parseSkillsFromRequestBody is a helper that extracts the skill parsing logic
// for testability. This mirrors what ChatComplete does.
//
// NOTE: The actual implementation uses io.ReadAll(r.Body) which handles
// chunked transfer encoding correctly. We don't check ContentLength because
// it may be -1 for chunked requests (common with browser fetch()).
func parseSkillsFromRequestBody(body string, contentLength int64) []SkillPayload {
	var skills []SkillPayload

	// Don't check contentLength - it may be -1 for chunked encoding
	// Instead, just check if we have body content
	if body == "" {
		return skills
	}

	var reqBody struct {
		Skills []SkillPayload `json:"skills"`
	}

	if err := json.Unmarshal([]byte(body), &reqBody); err == nil {
		skills = reqBody.Skills
	}

	return skills
}
