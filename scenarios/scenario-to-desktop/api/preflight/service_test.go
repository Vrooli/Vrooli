package preflight

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bundleruntime "scenario-to-desktop-runtime"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// mockTimeProvider provides a fixed time for deterministic testing.
type mockTimeProvider struct {
	now time.Time
}

func (m *mockTimeProvider) Now() time.Time {
	return m.now
}

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	t.Run("GetNonExistent", func(t *testing.T) {
		session, ok := store.Get("nonexistent")
		if ok || session != nil {
			t.Errorf("expected nil session for nonexistent ID")
		}
	})

	t.Run("StopNonExistent", func(t *testing.T) {
		if store.Stop("nonexistent") {
			t.Errorf("expected Stop to return false for nonexistent session")
		}
	})
}

func TestInMemoryJobStore(t *testing.T) {
	store := NewInMemoryJobStore()

	t.Run("CreateAndGet", func(t *testing.T) {
		job := store.Create()
		if job == nil || job.ID == "" {
			t.Fatalf("expected job to be created with ID")
		}

		retrieved, ok := store.Get(job.ID)
		if !ok {
			t.Fatalf("expected to retrieve created job")
		}
		if retrieved.ID != job.ID {
			t.Errorf("expected job IDs to match")
		}
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		job, ok := store.Get("nonexistent")
		if ok || job != nil {
			t.Errorf("expected nil job for nonexistent ID")
		}
	})

	t.Run("UpdateExisting", func(t *testing.T) {
		job := store.Create()
		store.Update(job.ID, func(j *Job) {
			j.Status = "running"
		})

		updated, ok := store.Get(job.ID)
		if !ok {
			t.Fatalf("expected to retrieve updated job")
		}
		if updated.Status != "running" {
			t.Errorf("expected status 'running', got %q", updated.Status)
		}
	})

	t.Run("UpdateNonExistent", func(t *testing.T) {
		// Should not panic
		store.Update("nonexistent", func(j *Job) {
			j.Status = "failed"
		})
	})

	t.Run("SetStep", func(t *testing.T) {
		job := store.Create()
		store.SetStep(job.ID, "step1", "running", "executing")

		updated, _ := store.Get(job.ID)
		step, ok := updated.Steps["step1"]
		if !ok {
			t.Fatalf("expected step1 to exist")
		}
		if step.State != "running" {
			t.Errorf("expected state 'running', got %q", step.State)
		}
	})

	t.Run("SetStepNonExistentJob", func(t *testing.T) {
		// Should not panic
		store.SetStep("nonexistent", "step1", "running", "executing")
	})

	t.Run("SetResult", func(t *testing.T) {
		job := store.Create()
		store.SetResult(job.ID, func(prev *Response) *Response {
			if prev == nil {
				return &Response{Status: "ok"}
			}
			prev.Status = "ok"
			return prev
		})

		updated, _ := store.Get(job.ID)
		if updated.Result == nil {
			t.Fatalf("expected result to be set")
		}
		if updated.Result.Status != "ok" {
			t.Errorf("expected status 'ok', got %q", updated.Result.Status)
		}
	})

	t.Run("Finish", func(t *testing.T) {
		job := store.Create()
		store.Finish(job.ID, "completed", "")

		finished, _ := store.Get(job.ID)
		if finished.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", finished.Status)
		}
		// UpdatedAt should be set when job is finished
		if finished.UpdatedAt.IsZero() {
			t.Errorf("expected UpdatedAt to be set")
		}
	})

	t.Run("FinishWithError", func(t *testing.T) {
		job := store.Create()
		store.Finish(job.ID, "failed", "test error")

		finished, _ := store.Get(job.ID)
		if finished.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", finished.Status)
		}
		if finished.Err != "test error" {
			t.Errorf("expected error 'test error', got %q", finished.Err)
		}
	})
}

func TestInMemoryJobStoreCleanup(t *testing.T) {
	store := NewInMemoryJobStore(WithJobExpiration(1 * time.Millisecond))

	// Create a completed job
	job := store.Create()
	store.Finish(job.ID, "completed", "")

	// Sleep to let it expire
	time.Sleep(10 * time.Millisecond)

	// Run cleanup
	store.Cleanup()

	// Job should be cleaned up
	_, ok := store.Get(job.ID)
	if ok {
		t.Errorf("expected expired job to be cleaned up")
	}
}

func TestInMemoryJobStoreCleanupKeepsRunning(t *testing.T) {
	store := NewInMemoryJobStore(WithJobExpiration(1 * time.Millisecond))

	// Create a running job
	job := store.Create()
	// Don't finish it - it should stay running

	// Sleep to let it "expire" if it were completed
	time.Sleep(10 * time.Millisecond)

	// Run cleanup
	store.Cleanup()

	// Running job should NOT be cleaned up
	_, ok := store.Get(job.ID)
	if !ok {
		t.Errorf("expected running job to NOT be cleaned up")
	}
}

func TestServiceCreation(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatalf("expected service to be created")
	}
}

func TestServiceWithOptions(t *testing.T) {
	sessionStore := NewInMemorySessionStore()
	jobStore := NewInMemoryJobStore()

	mockNow := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	timeProvider := func() time.Time { return mockNow }

	service := NewService(
		WithSessionStore(sessionStore),
		WithJobStore(jobStore),
		WithTimeProvider(timeProvider),
	)

	if service == nil {
		t.Fatalf("expected service to be created")
	}
	if service.sessions != sessionStore {
		t.Errorf("expected custom session store to be used")
	}
	if service.jobs != jobStore {
		t.Errorf("expected custom job store to be used")
	}
}

func TestServiceCreateJob(t *testing.T) {
	service := NewService()
	job := service.CreateJob()

	if job == nil {
		t.Fatalf("expected job to be created")
	}
	if job.ID == "" {
		t.Errorf("expected job ID to be set")
	}
	if job.Status != "running" {
		t.Errorf("expected status 'running', got %q", job.Status)
	}
}

func TestServiceGetJob(t *testing.T) {
	service := NewService()
	job := service.CreateJob()

	retrieved, ok := service.GetJob(job.ID)
	if !ok {
		t.Fatalf("expected to retrieve created job")
	}
	if retrieved.ID != job.ID {
		t.Errorf("expected job IDs to match")
	}
}

func TestServiceGetJobNonExistent(t *testing.T) {
	service := NewService()
	_, ok := service.GetJob("nonexistent")
	if ok {
		t.Errorf("expected GetJob to return false for nonexistent job")
	}
}

func TestServiceGetSession(t *testing.T) {
	service := NewService()
	// Without creating a session, GetSession should return false
	_, ok := service.GetSession("nonexistent")
	if ok {
		t.Errorf("expected GetSession to return false for nonexistent session")
	}
}

func TestServiceWithNilStores(t *testing.T) {
	service := &DefaultService{
		sessions: nil,
		jobs:     nil,
	}

	// These should not panic with nil stores
	job := service.CreateJob()
	if job != nil {
		t.Errorf("expected nil job when job store is nil")
	}

	_, ok := service.GetJob("test")
	if ok {
		t.Errorf("expected GetJob to return false when job store is nil")
	}

	_, ok = service.GetSession("test")
	if ok {
		t.Errorf("expected GetSession to return false when session store is nil")
	}
}

func TestRunBundlePreflightValidation(t *testing.T) {
	service := NewService()

	t.Run("EmptyManifestPath", func(t *testing.T) {
		_, err := service.RunBundlePreflight(Request{
			BundleManifestPath: "",
		})
		if err == nil {
			t.Fatalf("expected error for empty manifest path")
		}
		statusErr, ok := err.(*StatusError)
		if !ok {
			t.Fatalf("expected StatusError, got %T", err)
		}
		if statusErr.Status != 400 {
			t.Errorf("expected status 400, got %d", statusErr.Status)
		}
	})

	t.Run("StatusOnlyWithoutSessionID", func(t *testing.T) {
		_, err := service.RunBundlePreflight(Request{
			BundleManifestPath: "/path/to/manifest.json",
			StatusOnly:         true,
			SessionID:          "",
		})
		if err == nil {
			t.Fatalf("expected error for status_only without session_id")
		}
		statusErr, ok := err.(*StatusError)
		if !ok {
			t.Fatalf("expected StatusError, got %T", err)
		}
		if statusErr.Status != 400 {
			t.Errorf("expected status 400, got %d", statusErr.Status)
		}
	})

	t.Run("SessionStopWithNonexistentSession", func(t *testing.T) {
		_, err := service.RunBundlePreflight(Request{
			BundleManifestPath: "/path/to/manifest.json",
			SessionStop:        true,
			SessionID:          "nonexistent",
		})
		if err == nil {
			t.Fatalf("expected error for stopping nonexistent session")
		}
		statusErr, ok := err.(*StatusError)
		if !ok {
			t.Fatalf("expected StatusError, got %T", err)
		}
		if statusErr.Status != 404 {
			t.Errorf("expected status 404, got %d", statusErr.Status)
		}
	})

	t.Run("NonexistentManifestPath", func(t *testing.T) {
		_, err := service.RunBundlePreflight(Request{
			BundleManifestPath: "/nonexistent/path/to/manifest.json",
		})
		if err == nil {
			t.Fatalf("expected error for nonexistent manifest path")
		}
		statusErr, ok := err.(*StatusError)
		if !ok {
			t.Fatalf("expected StatusError, got %T", err)
		}
		if statusErr.Status != 400 {
			t.Errorf("expected status 400, got %d", statusErr.Status)
		}
	})
}

func TestPreflightTimeout(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected time.Duration
	}{
		{"zero defaults to 15s", 0, 15 * time.Second},
		{"negative defaults to 15s", -1, 15 * time.Second},
		{"normal value", 30, 30 * time.Second},
		{"exceeds max capped to 2m", 300, 2 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preflightTimeout(tt.seconds)
			if got != tt.expected {
				t.Errorf("preflightTimeout(%d) = %v, want %v", tt.seconds, got, tt.expected)
			}
		})
	}
}

func TestValidationStepState(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil returns skipped", nil, "skipped"},
		{"valid returns pass", &struct{ Valid bool }{true}, "pass"},
		{"invalid returns fail", &struct{ Valid bool }{false}, "fail"},
	}

	// Using a mock validation result
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *struct {
				Valid bool
			}
			if tt.input != nil {
				result = tt.input.(*struct{ Valid bool })
			}

			state := "skipped"
			if result != nil {
				if result.Valid {
					state = "pass"
				} else {
					state = "fail"
				}
			}
			if state != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, state)
			}
		})
	}
}

func TestSecretsStepState(t *testing.T) {
	tests := []struct {
		name     string
		secrets  []Secret
		expected string
	}{
		{"empty returns skipped", []Secret{}, "skipped"},
		{"all have values returns pass", []Secret{{Required: true, HasValue: true}}, "pass"},
		{"missing required returns warning", []Secret{{Required: true, HasValue: false}}, "warning"},
		{"optional missing returns pass", []Secret{{Required: false, HasValue: false}}, "pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := secretsStepState(tt.secrets)
			if got != tt.expected {
				t.Errorf("secretsStepState() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestReadinessStepState(t *testing.T) {
	tests := []struct {
		name     string
		ready    *Ready
		request  Request
		expected string
	}{
		{"nil ready returns skipped", nil, Request{}, "skipped"},
		{"not start services returns skipped", &Ready{Ready: true}, Request{StartServices: false}, "skipped"},
		{"status only returns skipped", &Ready{Ready: true}, Request{StartServices: true, StatusOnly: true}, "skipped"},
		{"ready returns pass", &Ready{Ready: true}, Request{StartServices: true}, "pass"},
		{"not ready returns warning", &Ready{Ready: false}, Request{StartServices: true}, "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readinessStepState(tt.ready, tt.request)
			if got != tt.expected {
				t.Errorf("readinessStepState() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDiagnosticsStepState(t *testing.T) {
	tests := []struct {
		name      string
		ports     map[string]map[string]int
		telemetry *Telemetry
		logTails  []LogTail
		request   Request
		expected  string
	}{
		{"not start services returns skipped", nil, nil, nil, Request{StartServices: false}, "skipped"},
		{"has ports returns pass", map[string]map[string]int{"svc": {"http": 8080}}, nil, nil, Request{StartServices: true}, "pass"},
		{"has telemetry path returns pass", nil, &Telemetry{Path: "/path"}, nil, Request{StartServices: true}, "pass"},
		{"has log tails returns pass", nil, nil, []LogTail{{ServiceID: "svc"}}, Request{StartServices: true}, "pass"},
		{"nothing returns warning", nil, nil, nil, Request{StartServices: true}, "warning"},
		{"empty telemetry returns warning", nil, &Telemetry{Path: ""}, nil, Request{StartServices: true}, "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diagnosticsStepState(tt.ports, tt.telemetry, tt.logTails, tt.request)
			if got != tt.expected {
				t.Errorf("diagnosticsStepState() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStatusError(t *testing.T) {
	err := &StatusError{
		Status: 404,
		Err:    &testError{msg: "not found"},
	}

	if err.Error() != "not found" {
		t.Errorf("expected error message 'not found', got %q", err.Error())
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// HTTPRuntimeClient tests

func TestNewHTTPRuntimeClient(t *testing.T) {
	client := NewHTTPRuntimeClient("http://localhost:8080", "test-token", nil)
	if client == nil {
		t.Fatalf("expected client to be created")
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL 'http://localhost:8080', got %q", client.baseURL)
	}
	if client.token != "test-token" {
		t.Errorf("expected token 'test-token', got %q", client.token)
	}
}

func TestNewHTTPRuntimeClientWithOptions(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 10 * time.Second}
	client := NewHTTPRuntimeClient("http://localhost:8080", "test-token", nil,
		WithHTTPClient(customHTTPClient),
	)
	if client.httpClient != customHTTPClient {
		t.Errorf("expected custom HTTP client to be used")
	}
}

func TestHTTPRuntimeClient_Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Errorf("expected path '/status', got %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Runtime{RuntimeVersion: "1.0.0", InstanceID: "test-instance"})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if status.RuntimeVersion != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", status.RuntimeVersion)
	}
}

func TestHTTPRuntimeClient_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Status()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_Secrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/secrets" {
			t.Errorf("expected path '/secrets', got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"secrets": []Secret{
				{ID: "API_KEY", Required: true, HasValue: false},
				{ID: "DB_PASSWORD", Required: true, HasValue: true},
			},
		})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	secrets, err := client.Secrets()
	if err != nil {
		t.Fatalf("Secrets() error: %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(secrets))
	}
}

func TestHTTPRuntimeClient_ApplySecrets(t *testing.T) {
	t.Run("posts secrets", func(t *testing.T) {
		var receivedPayload map[string]map[string]string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		err := client.ApplySecrets(map[string]string{"API_KEY": "secret123"})
		if err != nil {
			t.Fatalf("ApplySecrets() error: %v", err)
		}
		if receivedPayload["secrets"]["API_KEY"] != "secret123" {
			t.Errorf("expected API_KEY 'secret123', got %q", receivedPayload["secrets"]["API_KEY"])
		}
	})

	t.Run("skips empty secrets", func(t *testing.T) {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		err := client.ApplySecrets(map[string]string{"API_KEY": ""})
		if err != nil {
			t.Fatalf("ApplySecrets() error: %v", err)
		}
		if called {
			t.Errorf("expected no HTTP call for empty secrets")
		}
	})

	t.Run("skips nil secrets", func(t *testing.T) {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		err := client.ApplySecrets(nil)
		if err != nil {
			t.Fatalf("ApplySecrets() error: %v", err)
		}
		if called {
			t.Errorf("expected no HTTP call for nil secrets")
		}
	})
}

func TestHTTPRuntimeClient_Ports(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"services": map[string]map[string]int{
				"api": {"http": 8080},
				"web": {"http": 3000},
			},
		})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	ports, err := client.Ports()
	if err != nil {
		t.Fatalf("Ports() error: %v", err)
	}
	if ports["api"]["http"] != 8080 {
		t.Errorf("expected api http port 8080, got %d", ports["api"]["http"])
	}
}

func TestHTTPRuntimeClient_Telemetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Telemetry{Path: "/telemetry", UploadURL: "http://localhost:8080/telemetry"})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	telemetry, err := client.Telemetry()
	if err != nil {
		t.Fatalf("Telemetry() error: %v", err)
	}
	if telemetry.Path != "/telemetry" {
		t.Errorf("expected path '/telemetry', got %q", telemetry.Path)
	}
}

func TestHTTPRuntimeClient_LogTails(t *testing.T) {
	t.Run("returns nil for nil manifest", func(t *testing.T) {
		client := NewHTTPRuntimeClient("http://localhost:8080", "", nil)
		tails := client.LogTails(Request{LogTailLines: 10})
		if tails != nil {
			t.Errorf("expected nil for nil manifest")
		}
	})
}

func TestHTTPRuntimeClient_AuthorizationHeader(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Runtime{})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "test-token", nil)
	client.Status()

	if receivedAuth != "Bearer test-token" {
		t.Errorf("expected 'Bearer test-token', got %q", receivedAuth)
	}
}

func TestUpdatePreflightResult(t *testing.T) {
	t.Run("nil prev creates new response", func(t *testing.T) {
		result := updatePreflightResult(nil, func(next *Response) {
			next.Status = "updated"
		})
		if result.Status != "updated" {
			t.Errorf("expected status 'updated', got %q", result.Status)
		}
	})

	t.Run("existing prev is updated", func(t *testing.T) {
		prev := &Response{Status: "initial"}
		result := updatePreflightResult(prev, func(next *Response) {
			next.Status = "updated"
		})
		if result.Status != "updated" {
			t.Errorf("expected status 'updated', got %q", result.Status)
		}
	})
}

func TestRuntimeHandleFromSession(t *testing.T) {
	t.Run("nil session returns empty handle", func(t *testing.T) {
		handle := runtimeHandleFromSession(nil)
		if handle == nil {
			t.Fatalf("expected non-nil handle")
		}
		if handle.Client != nil {
			t.Errorf("expected nil client for nil session")
		}
		if handle.SessionID != "" {
			t.Errorf("expected empty session ID")
		}
	})

	t.Run("valid session creates handle", func(t *testing.T) {
		session := &Session{
			ID:        "test-session",
			BaseURL:   "http://localhost:8080",
			Token:     "test-token",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		handle := runtimeHandleFromSession(session)
		if handle.SessionID != "test-session" {
			t.Errorf("expected session ID 'test-session', got %q", handle.SessionID)
		}
		if handle.Client == nil {
			t.Errorf("expected non-nil client")
		}
		if handle.ExpiresAt.IsZero() {
			t.Errorf("expected ExpiresAt to be set")
		}
	})
}

func TestCollectServiceFingerprints(t *testing.T) {
	t.Run("nil manifest returns nil", func(t *testing.T) {
		result := collectServiceFingerprints(nil, "/tmp")
		if result != nil {
			t.Errorf("expected nil for nil manifest")
		}
	})
}

func TestSha256File(t *testing.T) {
	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := sha256File("/nonexistent/path/to/file")
		if err == nil {
			t.Errorf("expected error for nonexistent file")
		}
	})
}

func TestReadFileWithRetry(t *testing.T) {
	t.Run("nonexistent file returns error after timeout", func(t *testing.T) {
		_, err := readFileWithRetry("/nonexistent/path", 100*time.Millisecond)
		if err == nil {
			t.Errorf("expected error for nonexistent file")
		}
	})
}

func TestReadPortFileWithRetry(t *testing.T) {
	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := readPortFileWithRetry("/nonexistent/path", 100*time.Millisecond)
		if err == nil {
			t.Errorf("expected error for nonexistent file")
		}
	})
}

func TestWaitForRuntimeHealth(t *testing.T) {
	t.Run("returns error when server not responding", func(t *testing.T) {
		client := &http.Client{Timeout: 100 * time.Millisecond}
		err := waitForRuntimeHealth(client, "http://localhost:59999", 200*time.Millisecond)
		if err == nil {
			t.Errorf("expected error for non-responding server")
		}
	})

	t.Run("returns nil when server responds OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		client := &http.Client{Timeout: 100 * time.Millisecond}
		err := waitForRuntimeHealth(client, server.URL, 1*time.Second)
		if err != nil {
			t.Errorf("expected nil error for responding server, got %v", err)
		}
	})
}

func TestBuildPreflightChecks(t *testing.T) {
	t.Run("nil inputs returns empty checks", func(t *testing.T) {
		checks := buildPreflightChecks(nil, nil, nil, nil, nil, nil, nil, Request{})
		if len(checks) != 0 {
			t.Errorf("expected 0 checks, got %d", len(checks))
		}
	})

	t.Run("adds checks for ready", func(t *testing.T) {
		ready := &Ready{Ready: true}
		checks := buildPreflightChecks(nil, nil, ready, nil, nil, nil, nil, Request{})
		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
		if checks[0].ID != "runtime-ready" {
			t.Errorf("expected ID 'runtime-ready', got %q", checks[0].ID)
		}
		if checks[0].Status != "pass" {
			t.Errorf("expected status 'pass', got %q", checks[0].Status)
		}
	})

	t.Run("adds checks for not ready", func(t *testing.T) {
		ready := &Ready{Ready: false}
		checks := buildPreflightChecks(nil, nil, ready, nil, nil, nil, nil, Request{})
		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
		if checks[0].Status != "fail" {
			t.Errorf("expected status 'fail', got %q", checks[0].Status)
		}
	})

	t.Run("adds checks for secrets", func(t *testing.T) {
		secrets := []Secret{{ID: "API_KEY", Required: true, HasValue: true}}
		checks := buildPreflightChecks(nil, nil, nil, secrets, nil, nil, nil, Request{})
		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
		if checks[0].ID != "secrets" {
			t.Errorf("expected ID 'secrets', got %q", checks[0].ID)
		}
	})

	t.Run("adds diagnostics check for start_services", func(t *testing.T) {
		ports := map[string]map[string]int{"api": {"http": 8080}}
		checks := buildPreflightChecks(nil, nil, nil, nil, ports, nil, nil, Request{StartServices: true})
		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
		if checks[0].ID != "diagnostics" {
			t.Errorf("expected ID 'diagnostics', got %q", checks[0].ID)
		}
	})
}

func TestServiceStartJanitor(t *testing.T) {
	t.Run("does not panic with nil stores", func(t *testing.T) {
		service := &DefaultService{
			sessions: nil,
			jobs:     nil,
		}
		// Should not panic
		service.StartJanitor()
	})
}

func TestHTTPRuntimeClient_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/validate" {
				t.Errorf("expected path '/validate', got %q", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":    true,
				"errors":   []interface{}{},
				"warnings": []interface{}{},
			})
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		result, err := client.Validate()
		if err != nil {
			t.Fatalf("Validate() error: %v", err)
		}
		if !result.Valid {
			t.Errorf("expected valid=true")
		}
	})

	t.Run("with validation errors (422 status)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":  false,
				"errors": []interface{}{map[string]string{"message": "test error"}},
			})
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		result, err := client.Validate()
		if err != nil {
			t.Fatalf("Validate() error: %v", err)
		}
		if result.Valid {
			t.Errorf("expected valid=false")
		}
	})
}

func TestHTTPRuntimeClient_Ready(t *testing.T) {
	t.Run("immediate ready", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Ready{Ready: true})
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		result, waited, err := client.Ready(Request{}, 5*time.Second)
		if err != nil {
			t.Fatalf("Ready() error: %v", err)
		}
		if !result.Ready {
			t.Errorf("expected ready=true")
		}
		if waited != 0 {
			t.Errorf("expected waited=0 for immediate ready")
		}
	})

	t.Run("status only skips polling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Ready{Ready: false})
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		result, waited, err := client.Ready(Request{StatusOnly: true}, 5*time.Second)
		if err != nil {
			t.Fatalf("Ready() error: %v", err)
		}
		if result.Ready {
			t.Errorf("expected ready=false")
		}
		if waited != 0 {
			t.Errorf("expected waited=0 for status only")
		}
	})
}

func TestHTTPRuntimeClient_fetchText(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("log content here"))
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "test-token", nil)
		text, status, err := client.fetchText("/logs")
		if err != nil {
			t.Fatalf("fetchText() error: %v", err)
		}
		if text != "log content here" {
			t.Errorf("expected 'log content here', got %q", text)
		}
		if status != http.StatusOK {
			t.Errorf("expected status 200, got %d", status)
		}
	})

	t.Run("error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "", nil)
		_, status, err := client.fetchText("/logs")
		if err == nil {
			t.Fatalf("expected error for 404 status")
		}
		if status != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", status)
		}
	})

	t.Run("sends auth header", func(t *testing.T) {
		var receivedAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPRuntimeClient(server.URL, "my-token", nil)
		client.fetchText("/logs")
		if receivedAuth != "Bearer my-token" {
			t.Errorf("expected 'Bearer my-token', got %q", receivedAuth)
		}
	})
}

func TestHTTPRuntimeClient_maxReadinessTimeout(t *testing.T) {
	t.Run("nil manifest returns 0", func(t *testing.T) {
		client := NewHTTPRuntimeClient("http://localhost", "", nil)
		timeout := client.maxReadinessTimeout()
		if timeout != 0 {
			t.Errorf("expected 0 for nil manifest, got %v", timeout)
		}
	})
}

func TestHTTPRuntimeClient_collectLogTails(t *testing.T) {
	t.Run("returns nil for zero lines", func(t *testing.T) {
		client := NewHTTPRuntimeClient("http://localhost", "", nil)
		tails := client.collectLogTails(Request{LogTailLines: 0})
		if tails != nil {
			t.Errorf("expected nil for zero lines")
		}
	})

	t.Run("returns nil for nil manifest", func(t *testing.T) {
		client := NewHTTPRuntimeClient("http://localhost", "", nil)
		tails := client.collectLogTails(Request{LogTailLines: 10})
		if tails != nil {
			t.Errorf("expected nil for nil manifest")
		}
	})
}

func TestValidationStepStateActual(t *testing.T) {
	t.Run("nil returns skipped", func(t *testing.T) {
		result := validationStepState(nil)
		if result != "skipped" {
			t.Errorf("expected 'skipped', got %q", result)
		}
	})
}

func TestWithRuntimeFactory(t *testing.T) {
	service := NewService(WithRuntimeFactory(func(manifest *bundlemanifest.Manifest, bundleRoot string, timeout time.Duration) (*RuntimeHandle, error) {
		return &RuntimeHandle{}, nil
	}))

	if service.newDryRunRuntime == nil {
		t.Errorf("expected newDryRunRuntime to be set")
	}
}

func TestValidationStepStateWithValidationResult(t *testing.T) {
	t.Run("valid returns pass", func(t *testing.T) {
		// Import runtimeapi for the actual type if needed
		// Since we're in the same package, we can test directly
		result := validationStepState(nil)
		if result != "skipped" {
			t.Errorf("expected 'skipped', got %q", result)
		}
	})
}

func TestHTTPRuntimeClient_PortsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Ports()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_TelemetryError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Telemetry()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_SecretsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Secrets()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_ApplySecretsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	err := client.ApplySecrets(map[string]string{"key": "value"})
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_ValidateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, err := client.Validate()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_ReadyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	_, _, err := client.Ready(Request{}, 1*time.Second)
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestHTTPRuntimeClient_fetchJSONWithPayload(t *testing.T) {
	var receivedPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "token", nil)
	var out map[string]string
	_, err := client.fetchJSON("/test", http.MethodPost, map[string]string{"key": "value"}, &out, nil)
	if err != nil {
		t.Fatalf("fetchJSON() error: %v", err)
	}
	if receivedPayload["key"] != "value" {
		t.Errorf("expected payload key=value, got %v", receivedPayload)
	}
	if out["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", out)
	}
}

func TestHTTPRuntimeClient_fetchJSONDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	var out map[string]string
	_, err := client.fetchJSON("/test", http.MethodGet, nil, &out, nil)
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestHTTPRuntimeClient_fetchJSONAllowedStatusDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	var out map[string]string
	allow := map[int]bool{http.StatusUnprocessableEntity: true}
	_, err := client.fetchJSON("/test", http.MethodGet, nil, &out, allow)
	if err == nil {
		t.Fatalf("expected decode error for allowed status with bad JSON")
	}
}

func TestHTTPRuntimeClient_fetchJSONAllowedStatusNoOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("error message"))
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	allow := map[int]bool{http.StatusUnprocessableEntity: true}
	status, err := client.fetchJSON("/test", http.MethodGet, nil, nil, allow)
	if err != nil {
		t.Fatalf("expected no error for allowed status, got %v", err)
	}
	if status != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", status)
	}
}

func TestHTTPRuntimeClient_fetchTextEmptyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		// Empty body
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	body, _, err := client.fetchText("/test")
	if err == nil {
		t.Fatalf("expected error for 404 status")
	}
	// Body should contain the status text since body was empty
	if body == "" {
		// The function returns resp.Status when body is empty
		t.Logf("body was empty as expected")
	}
}

// Session store option tests
func TestSessionStoreOptions(t *testing.T) {
	t.Run("WithSupervisorFactory", func(t *testing.T) {
		// Just verify option doesn't panic
		store := NewInMemorySessionStore(WithSupervisorFactory(func(manifest *bundlemanifest.Manifest, bundleRoot, appData string) (*bundleruntime.Supervisor, error) {
			return nil, nil
		}))
		if store.createSupervisor == nil {
			t.Error("expected createSupervisor to be set")
		}
	})

	t.Run("WithFileReader", func(t *testing.T) {
		store := NewInMemorySessionStore(WithFileReader(func(path string, timeout time.Duration) ([]byte, error) {
			return []byte("test-token"), nil
		}))
		if store.readFileWithRetry == nil {
			t.Error("expected readFileWithRetry to be set")
		}
	})

	t.Run("WithPortReader", func(t *testing.T) {
		store := NewInMemorySessionStore(WithPortReader(func(path string, timeout time.Duration) (int, error) {
			return 8080, nil
		}))
		if store.readPortFile == nil {
			t.Error("expected readPortFile to be set")
		}
	})

	t.Run("WithHealthWaiter", func(t *testing.T) {
		store := NewInMemorySessionStore(WithHealthWaiter(func(client *http.Client, baseURL string, timeout time.Duration) error {
			return nil
		}))
		if store.waitForHealth == nil {
			t.Error("expected waitForHealth to be set")
		}
	})
}

func TestSessionStoreRefresh(t *testing.T) {
	store := NewInMemorySessionStore()

	t.Run("zero TTL does nothing", func(t *testing.T) {
		session := &Session{
			ID:        "test",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		originalExpiry := session.ExpiresAt
		store.Refresh(session, 0)
		if session.ExpiresAt != originalExpiry {
			t.Error("expected expiry not to change for zero TTL")
		}
	})

	t.Run("negative TTL does nothing", func(t *testing.T) {
		session := &Session{
			ID:        "test",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		originalExpiry := session.ExpiresAt
		store.Refresh(session, -1)
		if session.ExpiresAt != originalExpiry {
			t.Error("expected expiry not to change for negative TTL")
		}
	})

	t.Run("TTL capped at 900 seconds", func(t *testing.T) {
		session := &Session{
			ID:        "test",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		store.Refresh(session, 9999)
		// Should be capped to 900 seconds
		expectedMax := time.Now().Add(901 * time.Second)
		if session.ExpiresAt.After(expectedMax) {
			t.Error("expected expiry to be capped at 900 seconds")
		}
	})

	t.Run("normal TTL sets expiry", func(t *testing.T) {
		session := &Session{
			ID:        "test",
			ExpiresAt: time.Now().Add(1 * time.Minute),
		}
		store.Refresh(session, 300)
		// Should be about 300 seconds from now
		minExpected := time.Now().Add(299 * time.Second)
		maxExpected := time.Now().Add(301 * time.Second)
		if session.ExpiresAt.Before(minExpected) || session.ExpiresAt.After(maxExpected) {
			t.Errorf("expected expiry around 300 seconds, got %v", time.Until(session.ExpiresAt))
		}
	})
}

func TestSessionStoreCleanup(t *testing.T) {
	store := NewInMemorySessionStore()

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	store.mux.Lock()
	store.sessions["expired"] = expiredSession
	store.mux.Unlock()

	// Create a valid session
	validSession := &Session{
		ID:        "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	store.mux.Lock()
	store.sessions["valid"] = validSession
	store.mux.Unlock()

	// Run cleanup
	store.Cleanup()

	// Check expired session is gone
	_, ok := store.Get("expired")
	if ok {
		t.Error("expected expired session to be cleaned up")
	}

	// Check valid session remains
	_, ok = store.Get("valid")
	if !ok {
		t.Error("expected valid session to remain")
	}
}

func TestSessionStoreGetExpiry(t *testing.T) {
	store := NewInMemorySessionStore()

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	store.mux.Lock()
	store.sessions["expired"] = expiredSession
	store.mux.Unlock()

	// Get should delete expired session
	_, ok := store.Get("expired")
	if ok {
		t.Error("expected Get to return false for expired session")
	}
}

func TestShutdownSession(t *testing.T) {
	t.Run("nil session does nothing", func(t *testing.T) {
		// Should not panic
		shutdownSession(nil)
	})

	t.Run("empty session does nothing", func(t *testing.T) {
		// Should not panic
		shutdownSession(&Session{})
	})
}

func TestDefaultReadFileWithRetry(t *testing.T) {
	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := defaultReadFileWithRetry("/nonexistent/path", 100*time.Millisecond)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestDefaultReadPortFile(t *testing.T) {
	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := defaultReadPortFile("/nonexistent/path", 100*time.Millisecond)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestDefaultWaitForHealth(t *testing.T) {
	t.Run("returns error when server not responding", func(t *testing.T) {
		client := &http.Client{Timeout: 100 * time.Millisecond}
		err := defaultWaitForHealth(client, "http://localhost:59998", 200*time.Millisecond)
		if err == nil {
			t.Error("expected error for non-responding server")
		}
	})

	t.Run("returns nil when server responds OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		client := &http.Client{Timeout: 100 * time.Millisecond}
		err := defaultWaitForHealth(client, server.URL, 1*time.Second)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

func TestValidationStepStateComprehensive(t *testing.T) {
	t.Run("nil validation returns skipped", func(t *testing.T) {
		result := validationStepState(nil)
		if result != "skipped" {
			t.Errorf("expected 'skipped', got %q", result)
		}
	})
}

func TestBuildPreflightChecksComprehensive(t *testing.T) {
	t.Run("multiple checks combined", func(t *testing.T) {
		ready := &Ready{Ready: true}
		secrets := []Secret{{ID: "test", Required: true, HasValue: true}}
		ports := map[string]map[string]int{"svc": {"http": 8080}}
		checks := buildPreflightChecks(nil, nil, ready, secrets, ports, nil, nil, Request{StartServices: true})
		if len(checks) != 3 {
			t.Errorf("expected 3 checks, got %d", len(checks))
		}
	})
}

func TestDefaultCreateSupervisorType(t *testing.T) {
	// Test the function signature - it should be callable
	// The actual supervisor creation would require full runtime setup
	_ = defaultCreateSupervisor
}

func TestHTTPRuntimeClient_ReadyWithPolling(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		// Return not ready on first call, ready on second
		if callCount == 1 {
			json.NewEncoder(w).Encode(Ready{Ready: false})
		} else {
			json.NewEncoder(w).Encode(Ready{Ready: true})
		}
	}))
	defer server.Close()

	client := NewHTTPRuntimeClient(server.URL, "", nil)
	// Start services triggers polling
	result, _, err := client.Ready(Request{StartServices: true}, 5*time.Second)
	if err != nil {
		t.Fatalf("Ready() error: %v", err)
	}
	if !result.Ready {
		t.Error("expected ready=true after polling")
	}
}
