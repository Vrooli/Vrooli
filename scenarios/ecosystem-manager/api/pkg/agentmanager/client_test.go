package agentmanager

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// testClient wraps Client with a fixed base URL for testing.
type testClient struct {
	*Client
	baseURL string
}

// newTestClient creates a client that uses the test server.
func newTestClient(server *httptest.Server) *testClient {
	return &testClient{
		Client: &Client{
			httpClient: server.Client(),
			jsonOpts:   protojson.MarshalOptions{UseProtoNames: false},
		},
		baseURL: server.URL,
	}
}

// doRequestTest overrides the base URL resolution for tests.
func (c *testClient) doRequestTest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// Health tests the health endpoint.
func (c *testClient) Health(ctx context.Context) (bool, error) {
	resp, err := c.doRequestTest(ctx, "GET", "/health", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

// EnsureProfile tests the ensure profile endpoint.
func (c *testClient) EnsureProfile(ctx context.Context, req *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error) {
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequestTest(ctx, "POST", "/api/v1/profiles/ensure", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result apipb.EnsureProfileResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTask tests the create task endpoint.
func (c *testClient) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	req := &apipb.CreateTaskRequest{Task: task}
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequestTest(ctx, "POST", "/api/v1/tasks", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result apipb.CreateTaskResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Task, nil
}

// CreateRun tests the create run endpoint.
func (c *testClient) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequestTest(ctx, "POST", "/api/v1/runs", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result apipb.CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// GetRun tests the get run endpoint.
func (c *testClient) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	resp, err := c.doRequestTest(ctx, "GET", "/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.GetRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// StopRun tests the stop run endpoint.
func (c *testClient) StopRun(ctx context.Context, runID string) error {
	resp, err := c.doRequestTest(ctx, "POST", "/api/v1/runs/"+runID+"/stop", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// GetRunEvents tests the get run events endpoint.
func (c *testClient) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	path := "/api/v1/runs/" + runID + "/events"
	if afterSequence > 0 {
		path += "?after_sequence=" + string(rune(afterSequence+'0'))
	}

	resp, err := c.doRequestTest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.GetRunEventsResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Events, nil
}

// =============================================================================
// TESTS
// =============================================================================

func TestClient_Health(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantOK     bool
	}{
		{
			name:       "healthy service returns true",
			statusCode: http.StatusOK,
			wantOK:     true,
		},
		{
			name:       "unhealthy service returns false",
			statusCode: http.StatusServiceUnavailable,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health" {
					t.Errorf("expected path /health, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server)
			ok, err := client.Health(context.Background())
			if err != nil {
				t.Errorf("Health() error = %v", err)
				return
			}
			if ok != tt.wantOK {
				t.Errorf("Health() = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestClient_EnsureProfile(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   map[string]any
		wantErr    bool
	}{
		{
			name:       "success creates profile",
			statusCode: http.StatusOK,
			response: map[string]any{
				"profile": map[string]any{
					"id":         "test-profile-id",
					"profileKey": "test-key",
				},
				"created": true,
			},
			wantErr: false,
		},
		{
			name:       "error returns error",
			statusCode: http.StatusInternalServerError,
			response:   map[string]any{"error": "internal error"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/profiles/ensure") {
					t.Errorf("expected path ending with /profiles/ensure, got %s", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := newTestClient(server)
			req := &apipb.EnsureProfileRequest{
				ProfileKey: "test-key",
			}

			resp, err := client.EnsureProfile(context.Background(), req)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && resp == nil {
				t.Error("EnsureProfile() returned nil response without error")
			}
		})
	}
}

func TestClient_CreateTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/tasks") {
			t.Errorf("expected path ending with /tasks, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"task": map[string]any{
				"id":    "created-task-id",
				"title": "Test Task",
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	task := &domainpb.Task{
		Title:       "Test Task",
		Description: "Test Description",
	}

	result, err := client.CreateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if result == nil {
		t.Fatal("CreateTask() returned nil task")
	}
}

func TestClient_CreateRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/runs") {
			t.Errorf("expected path ending with /runs, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"run": map[string]any{
				"id":     "created-run-id",
				"taskId": "task-123",
				"status": "RUN_STATUS_QUEUED",
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	profileID := "profile-456"
	req := &apipb.CreateRunRequest{
		TaskId:         "task-123",
		AgentProfileId: &profileID,
	}

	result, err := client.CreateRun(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	if result == nil {
		t.Fatal("CreateRun() returned nil run")
	}
}

func TestClient_GetRun(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   map[string]any
		wantNil    bool
		wantErr    bool
	}{
		{
			name:       "found run returns run",
			statusCode: http.StatusOK,
			response: map[string]any{
				"run": map[string]any{
					"id":     "run-123",
					"status": "RUN_STATUS_RUNNING",
				},
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name:       "not found returns nil",
			statusCode: http.StatusNotFound,
			response:   nil,
			wantNil:    true,
			wantErr:    false,
		},
		{
			name:       "error returns error",
			statusCode: http.StatusInternalServerError,
			response:   map[string]any{"error": "internal error"},
			wantNil:    true,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := newTestClient(server)
			result, err := client.GetRun(context.Background(), "run-123")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (result == nil) != tt.wantNil {
				t.Errorf("GetRun() result = %v, wantNil %v", result, tt.wantNil)
			}
		})
	}
}

func TestClient_StopRun(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "success stops run",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "error returns error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if !strings.HasSuffix(r.URL.Path, "/stop") {
					t.Errorf("expected path ending with /stop, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server)
			err := client.StopRun(context.Background(), "run-123")
			if (err != nil) != tt.wantErr {
				t.Errorf("StopRun() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetRunEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/events") {
			t.Errorf("expected path containing /events, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"events": []map[string]any{
				{"sequence": 11, "type": "RUN_EVENT_TYPE_LOG"},
				{"sequence": 12, "type": "RUN_EVENT_TYPE_TOOL_CALL"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	events, err := client.GetRunEvents(context.Background(), "run-123", 0)
	if err != nil {
		t.Fatalf("GetRunEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Errorf("GetRunEvents() returned %d events, want 2", len(events))
	}
}

func TestWaitForRun_Completes(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Return running on first call, complete on second
		status := "RUN_STATUS_RUNNING"
		if callCount >= 2 {
			status = "RUN_STATUS_COMPLETE"
		}
		json.NewEncoder(w).Encode(map[string]any{
			"run": map[string]any{
				"id":     "run-123",
				"status": status,
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a simple polling loop to test WaitForRun behavior
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var result *domainpb.Run
	var err error
	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout waiting for run to complete")
		case <-ticker.C:
			result, err = client.GetRun(ctx, "run-123")
			if err != nil {
				t.Fatalf("GetRun() error = %v", err)
			}
			if result != nil && result.Status == domainpb.RunStatus_RUN_STATUS_COMPLETE {
				goto done
			}
		}
	}
done:
	if result == nil {
		t.Fatal("expected non-nil run result")
	}
	if callCount < 2 {
		t.Errorf("polled %d times, expected at least 2", callCount)
	}
}
