package preflight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"test-genie/internal/structure/smoke/orchestrator"
)

func TestDefaultHealthThresholds(t *testing.T) {
	thresholds := DefaultHealthThresholds()
	if thresholds.MaxChromeProcesses != 50 {
		t.Errorf("expected MaxChromeProcesses=50, got %d", thresholds.MaxChromeProcesses)
	}
	if thresholds.MaxMemoryPercent != 80.0 {
		t.Errorf("expected MaxMemoryPercent=80.0, got %f", thresholds.MaxMemoryPercent)
	}
}

func TestDiagnoseBrowserlessFailure_Offline(t *testing.T) {
	// Use invalid URL to simulate offline
	checker := NewChecker("http://127.0.0.1:99999")
	// Also set executor to fail
	checker.cmdExecutor = &mockCmdExecutor{
		err: context.DeadlineExceeded,
	}

	diagnosis := checker.DiagnoseBrowserlessFailure(context.Background(), "test-scenario")

	if diagnosis.Type != orchestrator.DiagnosisOffline {
		t.Errorf("expected DiagnosisOffline, got %s", diagnosis.Type)
	}
	if diagnosis.IsBrowserlessIssue != "true" {
		t.Errorf("expected IsBrowserlessIssue=true, got %s", diagnosis.IsBrowserlessIssue)
	}
}

func TestDiagnoseBrowserlessFailure_ProcessLeak(t *testing.T) {
	// Server that reports healthy pressure but we'll mock the executor to return high chrome count
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":0,"queued":0,"concurrent":10}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	// Override the executor to return high chrome count
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("75\n"), // More than 50 processes
			"docker": []byte("45.00%\n"),
		},
	}

	diagnosis := checker.DiagnoseBrowserlessFailure(context.Background(), "test-scenario")

	if diagnosis.Type != orchestrator.DiagnosisProcessLeak {
		t.Errorf("expected DiagnosisProcessLeak, got %s", diagnosis.Type)
	}
	if diagnosis.IsBrowserlessIssue != "true" {
		t.Errorf("expected IsBrowserlessIssue=true, got %s", diagnosis.IsBrowserlessIssue)
	}
	if diagnosis.Diagnostics == nil {
		t.Error("expected diagnostics to be populated")
	} else if diagnosis.Diagnostics.ChromeProcessCount != 75 {
		t.Errorf("expected ChromeProcessCount=75, got %d", diagnosis.Diagnostics.ChromeProcessCount)
	}
}

func TestDiagnoseBrowserlessFailure_MemoryExhaustion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":0,"queued":0,"concurrent":10}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("10\n"), // Normal process count
			"docker": []byte("95.50%\n"), // High memory
		},
	}

	diagnosis := checker.DiagnoseBrowserlessFailure(context.Background(), "test-scenario")

	if diagnosis.Type != orchestrator.DiagnosisMemoryExhaustion {
		t.Errorf("expected DiagnosisMemoryExhaustion, got %s", diagnosis.Type)
	}
}

func TestDiagnoseBrowserlessFailure_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":0,"queued":0,"concurrent":10}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("5\n"),  // Low process count
			"docker": []byte("30.00%\n"), // Low memory
		},
	}

	diagnosis := checker.DiagnoseBrowserlessFailure(context.Background(), "test-scenario")

	if diagnosis.Type != orchestrator.DiagnosisUnknown {
		t.Errorf("expected DiagnosisUnknown for healthy status, got %s", diagnosis.Type)
	}
	if diagnosis.IsBrowserlessIssue != "false" {
		t.Errorf("expected IsBrowserlessIssue=false for healthy status, got %s", diagnosis.IsBrowserlessIssue)
	}
}

func TestGetHealthDiagnostics_PressureEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":3,"queued":2,"concurrent":10}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("15\n"),
			"docker": []byte("45.00%\n"),
		},
	}

	diag, err := checker.GetHealthDiagnostics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diag.RunningSessions != 3 {
		t.Errorf("expected RunningSessions=3, got %d", diag.RunningSessions)
	}
	if diag.QueuedSessions != 2 {
		t.Errorf("expected QueuedSessions=2, got %d", diag.QueuedSessions)
	}
	if diag.MaxConcurrent != 10 {
		t.Errorf("expected MaxConcurrent=10, got %d", diag.MaxConcurrent)
	}
	if diag.ChromeProcessCount != 15 {
		t.Errorf("expected ChromeProcessCount=15, got %d", diag.ChromeProcessCount)
	}
	if diag.MemoryUsagePercent != 45.0 {
		t.Errorf("expected MemoryUsagePercent=45.0, got %f", diag.MemoryUsagePercent)
	}
}

func TestIsHealthy(t *testing.T) {
	tests := []struct {
		name           string
		chromeCount    string
		memoryPercent  string
		expectedHealthy bool
	}{
		{"healthy", "10\n", "30.00%\n", true},
		{"unhealthy_chrome", "60\n", "30.00%\n", false},
		{"unhealthy_memory", "10\n", "85.00%\n", false},
		{"unhealthy_both", "60\n", "85.00%\n", false},
		{"at_threshold_chrome", "50\n", "30.00%\n", true},
		{"at_threshold_memory", "10\n", "80.00%\n", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/pressure" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"pressure":{"running":0,"queued":0,"concurrent":10}}`))
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()

			checker := NewChecker(server.URL)
			checker.cmdExecutor = &mockCmdExecutor{
				responses: map[string][]byte{
					"pgrep":  []byte(tc.chromeCount),
					"docker": []byte(tc.memoryPercent),
				},
			}

			healthy, diag, err := checker.IsHealthy(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if healthy != tc.expectedHealthy {
				t.Errorf("expected healthy=%v, got %v (chrome=%d, mem=%.1f%%)",
					tc.expectedHealthy, healthy, diag.ChromeProcessCount, diag.MemoryUsagePercent)
			}
		})
	}
}

func TestEnsureHealthy_AlreadyHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":0,"queued":0,"concurrent":10}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("5\n"),
			"docker": []byte("30.00%\n"),
		},
	}

	opts := orchestrator.AutoRecoveryOptions{
		SharedMode:    false,
		MaxRetries:    1,
		ContainerName: "test-browserless",
	}

	result, err := checker.EnsureHealthy(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected Success=true for already healthy")
	}
	if result.Attempted {
		t.Error("expected Attempted=false for already healthy")
	}
}

func TestEnsureHealthy_SharedModeWithActiveSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":5,"queued":0,"concurrent":10}}`)) // 5 active sessions
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("60\n"), // Unhealthy
			"docker": []byte("85.00%\n"),
		},
	}

	opts := orchestrator.AutoRecoveryOptions{
		SharedMode:    true, // Shared mode enabled
		MaxRetries:    1,
		ContainerName: "test-browserless",
	}

	result, err := checker.EnsureHealthy(context.Background(), opts)
	if err == nil {
		t.Error("expected error for shared mode with active sessions")
	}

	if result.Success {
		t.Error("expected Success=false for shared mode blocking recovery")
	}
	if result.Attempted {
		t.Error("expected Attempted=false when blocked by shared mode")
	}
}

// mockCmdExecutor is a test double for command execution
type mockCmdExecutor struct {
	responses map[string][]byte
	err       error
}

func (m *mockCmdExecutor) Execute(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Build full command for matching
	fullCmd := name
	for _, arg := range args {
		fullCmd += " " + arg
	}

	// Check for pgrep (chrome process count)
	if name == "docker" && len(args) >= 3 && args[0] == "exec" && args[2] == "pgrep" {
		if response, ok := m.responses["pgrep"]; ok {
			return response, nil
		}
	}

	// Check for docker stats (memory percent)
	if name == "docker" && len(args) >= 1 && args[0] == "stats" {
		if response, ok := m.responses["docker"]; ok {
			return response, nil
		}
	}

	// Check for docker logs (chrome crashes)
	if name == "docker" && len(args) >= 1 && args[0] == "logs" {
		if response, ok := m.responses["logs"]; ok {
			return response, nil
		}
		return []byte(""), nil // Return empty logs by default
	}

	// Check for resource-browserless
	if name == "resource-browserless" {
		if response, ok := m.responses["resource-browserless"]; ok {
			return response, nil
		}
		// Return empty to indicate command not found
		return nil, context.DeadlineExceeded
	}

	return nil, nil
}

func TestDiagnoseBrowserlessFailure_ChromeCrashes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":0,"queued":0,"concurrent":10}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("10\n"),     // Normal process count
			"docker": []byte("40.00%\n"), // Normal memory
			"logs":   []byte("Some log output\nPage crashed!\nMore logs\nPage crashed!\n"), // 2 crashes
		},
	}

	diagnosis := checker.DiagnoseBrowserlessFailure(context.Background(), "test-scenario")

	if diagnosis.Type != orchestrator.DiagnosisChromeCrashes {
		t.Errorf("expected DiagnosisChromeCrashes, got %s", diagnosis.Type)
	}
	if diagnosis.IsBrowserlessIssue != "true" {
		t.Errorf("expected IsBrowserlessIssue=true, got %s", diagnosis.IsBrowserlessIssue)
	}
	if diagnosis.Diagnostics == nil || diagnosis.Diagnostics.ChromeCrashes != 2 {
		t.Errorf("expected ChromeCrashes=2, got %d", diagnosis.Diagnostics.ChromeCrashes)
	}
}

func TestGetHealthDiagnostics_WithChromeCrashes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":0,"queued":0,"concurrent":10}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("5\n"),
			"docker": []byte("30.00%\n"),
			"logs":   []byte("Page crashed!\nOther log\nPage crashed!\nPage crashed!\n"), // 3 crashes
		},
	}

	diag, err := checker.GetHealthDiagnostics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diag.ChromeCrashes != 3 {
		t.Errorf("expected ChromeCrashes=3, got %d", diag.ChromeCrashes)
	}
	if diag.Status != "degraded" {
		t.Errorf("expected status=degraded when crashes detected, got %s", diag.Status)
	}
}

func TestGetHealthDiagnostics_NoCrashes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pressure" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pressure":{"running":0,"queued":0,"concurrent":10}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	checker := NewChecker(server.URL)
	checker.cmdExecutor = &mockCmdExecutor{
		responses: map[string][]byte{
			"pgrep":  []byte("5\n"),
			"docker": []byte("30.00%\n"),
			"logs":   []byte("Normal log output\nNo crashes here\n"),
		},
	}

	diag, err := checker.GetHealthDiagnostics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diag.ChromeCrashes != 0 {
		t.Errorf("expected ChromeCrashes=0, got %d", diag.ChromeCrashes)
	}
	if diag.Status != "healthy" {
		t.Errorf("expected status=healthy with no issues, got %s", diag.Status)
	}
}
