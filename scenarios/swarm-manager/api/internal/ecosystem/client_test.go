package ecosystem

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockHTTPDoer is a mock implementation of HTTPDoer for testing.
type mockHTTPDoer struct {
	postFunc func(url, contentType string, body *strings.Reader) (*http.Response, error)
}

func (m *mockHTTPDoer) Post(url, contentType string, body *strings.Reader) (*http.Response, error) {
	return m.postFunc(url, contentType, body)
}

// makeResponse creates an http.Response with the given status and body.
func makeResponse(status int, body interface{}) *http.Response {
	jsonBytes, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(string(jsonBytes))),
	}
}

func TestHTTPClient_CreateTask(t *testing.T) {
	tests := []struct {
		name          string
		req           CreateTaskRequest
		portResolver  func() (string, error)
		httpResponse  *http.Response
		httpError     error
		wantTaskID    string
		wantErrType   error
		wantErrSubstr string
	}{
		{
			name: "successful task creation",
			req: CreateTaskRequest{
				Title:     "Test Task",
				Operation: "generator",
				Priority:  1,
				Category:  "test",
			},
			portResolver: func() (string, error) { return "12345", nil },
			httpResponse: makeResponse(http.StatusCreated, Task{ID: "task-123"}),
			wantTaskID:   "task-123",
		},
		{
			name: "successful with 200 OK",
			req: CreateTaskRequest{
				Title:     "Test Task",
				Operation: "improver",
				Priority:  5,
			},
			portResolver: func() (string, error) { return "12345", nil },
			httpResponse: makeResponse(http.StatusOK, Task{ID: "task-456"}),
			wantTaskID:   "task-456",
		},
		{
			name: "port resolution fails",
			req: CreateTaskRequest{
				Title:     "Test Task",
				Operation: "generator",
				Priority:  1,
			},
			portResolver: func() (string, error) { return "", ErrNotAvailable },
			wantErrType:  ErrNotAvailable,
		},
		{
			name: "HTTP request fails",
			req: CreateTaskRequest{
				Title:     "Test Task",
				Operation: "generator",
				Priority:  1,
			},
			portResolver:  func() (string, error) { return "12345", nil },
			httpError:     errors.New("connection refused"),
			wantErrType:   ErrNotAvailable,
			wantErrSubstr: "connection refused",
		},
		{
			name: "server returns error status",
			req: CreateTaskRequest{
				Title:     "Test Task",
				Operation: "generator",
				Priority:  1,
			},
			portResolver: func() (string, error) { return "12345", nil },
			httpResponse: makeResponse(http.StatusInternalServerError, nil),
			wantErrType:  ErrTaskCreationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &mockHTTPDoer{
				postFunc: func(url, contentType string, body *strings.Reader) (*http.Response, error) {
					if tt.httpError != nil {
						return nil, tt.httpError
					}
					return tt.httpResponse, nil
				},
			}

			client := NewHTTPClientWithResolver(tt.portResolver, mockHTTP)

			taskID, err := client.CreateTask(tt.req)

			if tt.wantErrType != nil {
				if err == nil {
					t.Fatalf("expected error containing %v, got nil", tt.wantErrType)
				}
				if !errors.Is(err, tt.wantErrType) && !strings.Contains(err.Error(), tt.wantErrType.Error()) {
					t.Errorf("expected error type %v, got %v", tt.wantErrType, err)
				}
				if tt.wantErrSubstr != "" && !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.wantErrSubstr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if taskID != tt.wantTaskID {
				t.Errorf("expected task ID %q, got %q", tt.wantTaskID, taskID)
			}
		})
	}
}

func TestMapPriorityToString(t *testing.T) {
	tests := []struct {
		priority int
		want     string
	}{
		{1, "high"},
		{2, "high"},
		{3, "medium"},
		{5, "medium"},
		{7, "medium"},
		{8, "low"},
		{9, "low"},
		{10, "low"},
	}

	for _, tt := range tests {
		t.Run(string(rune('0'+tt.priority)), func(t *testing.T) {
			got := mapPriorityToString(tt.priority)
			if got != tt.want {
				t.Errorf("mapPriorityToString(%d) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

// TestMapPriorityToString_Boundaries tests boundary values for priority mapping
func TestMapPriorityToString_Boundaries(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     string
	}{
		{"zero (below range)", 0, "high"},
		{"negative", -1, "high"},
		{"boundary 2->3", 2, "high"},
		{"boundary 3", 3, "medium"},
		{"boundary 7->8", 7, "medium"},
		{"boundary 8", 8, "low"},
		{"very high number", 100, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapPriorityToString(tt.priority)
			if got != tt.want {
				t.Errorf("mapPriorityToString(%d) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

// TestHTTPClient_CreateTask_ResponseDecodeError tests response decode error path
func TestHTTPClient_CreateTask_ResponseDecodeError(t *testing.T) {
	mockHTTP := &mockHTTPDoer{
		postFunc: func(url, contentType string, body *strings.Reader) (*http.Response, error) {
			// Return success but with invalid JSON body
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader("invalid json")),
			}, nil
		},
	}

	client := NewHTTPClientWithResolver(
		func() (string, error) { return "12345", nil },
		mockHTTP,
	)

	_, err := client.CreateTask(CreateTaskRequest{
		Title:     "Test",
		Operation: "generator",
		Priority:  1,
	})

	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("expected decode error, got: %v", err)
	}
}

// TestHTTPClient_CreateTask_RequestConstruction verifies the HTTP request is built correctly
func TestHTTPClient_CreateTask_RequestConstruction(t *testing.T) {
	var capturedURL string
	var capturedContentType string
	var capturedBody string

	mockHTTP := &mockHTTPDoer{
		postFunc: func(url, contentType string, body *strings.Reader) (*http.Response, error) {
			capturedURL = url
			capturedContentType = contentType
			bodyBytes, _ := io.ReadAll(body)
			capturedBody = string(bodyBytes)
			return makeResponse(http.StatusCreated, Task{ID: "test-id"}), nil
		},
	}

	client := NewHTTPClientWithResolver(
		func() (string, error) { return "54321", nil },
		mockHTTP,
	)

	_, err := client.CreateTask(CreateTaskRequest{
		Title:     "My Test Task",
		Operation: "improver",
		Priority:  2,
		Category:  "testing",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify URL
	expectedURL := "http://localhost:54321/api/tasks"
	if capturedURL != expectedURL {
		t.Errorf("URL = %q, want %q", capturedURL, expectedURL)
	}

	// Verify content type
	if capturedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want 'application/json'", capturedContentType)
	}

	// Verify body contains expected fields
	if !strings.Contains(capturedBody, `"title":"My Test Task"`) {
		t.Errorf("body missing title: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, `"operation":"improver"`) {
		t.Errorf("body missing operation: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, `"priority":"high"`) {
		t.Errorf("body missing priority mapping (2 should be 'high'): %s", capturedBody)
	}
	if !strings.Contains(capturedBody, `"category":"testing"`) {
		t.Errorf("body missing category: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, `"type":"scenario"`) {
		t.Errorf("body missing type: %s", capturedBody)
	}
}

// TestNewHTTPClient verifies the default client creation
func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}
	if client.portResolver == nil {
		t.Error("portResolver is nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
}
