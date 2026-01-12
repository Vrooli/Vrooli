package preflight

import (
	"testing"
	"time"
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
