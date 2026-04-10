package pipeline

import (
	"encoding/json"
	"testing"
)

func TestLifecycleStateDescription(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		wantDesc string // Substring that should be in the description
	}{
		// Basic lifecycle states
		{
			name:     "empty state (pre-init crash)",
			state:    "",
			wantDesc: "crashed before smoke test initialization",
		},
		{
			name:     "init state",
			state:    "init",
			wantDesc: "crashed during initialization",
		},
		{
			name:     "ready state",
			state:    "ready",
			wantDesc: "server connectivity check",
		},
		{
			name:     "result state",
			state:    "result",
			wantDesc: "crashed during cleanup",
		},
		{
			name:     "exit state",
			state:    "exit",
			wantDesc: "completed the smoke test lifecycle",
		},
		// Granular bundled-mode states
		{
			name:     "bundle_resolving state",
			state:    "bundle_resolving",
			wantDesc: "locating the bundle directory",
		},
		{
			name:     "runtime_starting state",
			state:    "runtime_starting",
			wantDesc: "spawning the bundled runtime process",
		},
		{
			name:     "runtime_healthz state",
			state:    "runtime_healthz",
			wantDesc: "/healthz endpoint",
		},
		{
			name:     "runtime_readyz state",
			state:    "runtime_readyz",
			wantDesc: "/readyz endpoint",
		},
		{
			name:     "runtime_ports state",
			state:    "runtime_ports",
			wantDesc: "/ports endpoint",
		},
		// Unknown state
		{
			name:     "unknown state",
			state:    "unknown_state",
			wantDesc: "reached state 'unknown_state'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lifecycleStateDescription(tt.state)
			if got == "" {
				t.Error("Expected non-empty description")
			}
			if !containsSubstring(got, tt.wantDesc) {
				t.Errorf("lifecycleStateDescription(%q) = %q, want substring %q", tt.state, got, tt.wantDesc)
			}
		})
	}
}

func TestSmokeTestDetails_GetAppReportedError(t *testing.T) {
	tests := []struct {
		name    string
		details smokeTestDetails
		want    string
	}{
		{
			name:    "nil app reported error",
			details: smokeTestDetails{},
			want:    "",
		},
		{
			name: "empty message",
			details: smokeTestDetails{
				AppReportedError: &appReportedErrorDTO{
					Event: "smoke_test_failed",
				},
			},
			want: "",
		},
		{
			name: "with message",
			details: smokeTestDetails{
				AppReportedError: &appReportedErrorDTO{
					Event:   "smoke_test_failed",
					Message: "Bundled payload is missing",
				},
			},
			want: "Bundled payload is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.details.getAppReportedError()
			if got != tt.want {
				t.Errorf("getAppReportedError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSmokeTestDetails_GetAppReportedErrorContext(t *testing.T) {
	tests := []struct {
		name    string
		details smokeTestDetails
		want    string
	}{
		{
			name:    "nil app reported error",
			details: smokeTestDetails{},
			want:    "",
		},
		{
			name: "no context fields",
			details: smokeTestDetails{
				AppReportedError: &appReportedErrorDTO{
					Message: "Error message",
				},
			},
			want: "",
		},
		{
			name: "with deployment mode",
			details: smokeTestDetails{
				AppReportedError: &appReportedErrorDTO{
					Message:        "Error message",
					DeploymentMode: "bundled",
				},
			},
			want: "deployment_mode=bundled",
		},
		{
			name: "with event",
			details: smokeTestDetails{
				AppReportedError: &appReportedErrorDTO{
					Message: "Error message",
					Event:   "smoke_test_failed",
				},
			},
			want: "event=smoke_test_failed",
		},
		{
			name: "with both",
			details: smokeTestDetails{
				AppReportedError: &appReportedErrorDTO{
					Message:        "Error message",
					DeploymentMode: "bundled",
					Event:          "smoke_test_failed",
				},
			},
			want: "deployment_mode=bundled, event=smoke_test_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.details.getAppReportedErrorContext()
			if got != tt.want {
				t.Errorf("getAppReportedErrorContext() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSmokeTestDetails_GetLifecycleState(t *testing.T) {
	tests := []struct {
		name    string
		details smokeTestDetails
		want    string
	}{
		{
			name:    "nil error context",
			details: smokeTestDetails{},
			want:    "",
		},
		{
			name: "empty error context",
			details: smokeTestDetails{
				ErrorContext: map[string]string{},
			},
			want: "",
		},
		{
			name: "with lifecycle state",
			details: smokeTestDetails{
				ErrorContext: map[string]string{
					"last_lifecycle_state": "init",
				},
			},
			want: "init",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.details.getLifecycleState()
			if got != tt.want {
				t.Errorf("getLifecycleState() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSmokeTestDetails_GetStderr(t *testing.T) {
	tests := []struct {
		name    string
		details smokeTestDetails
		want    string
	}{
		{
			name:    "no stderr",
			details: smokeTestDetails{},
			want:    "",
		},
		{
			name: "LastStderr present",
			details: smokeTestDetails{
				LastStderr: "some error output",
			},
			want: "some error output",
		},
		{
			name: "ErrorContext stderr",
			details: smokeTestDetails{
				ErrorContext: map[string]string{
					"stderr": "context error output",
				},
			},
			want: "context error output",
		},
		{
			name: "LastStderr takes priority over ErrorContext",
			details: smokeTestDetails{
				LastStderr: "last stderr",
				ErrorContext: map[string]string{
					"stderr": "context stderr",
				},
			},
			want: "last stderr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.details.getStderr()
			if got != tt.want {
				t.Errorf("getStderr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStageResult_GetSmokeTestDetails(t *testing.T) {
	tests := []struct {
		name    string
		result  stageResult
		wantNil bool
	}{
		{
			name:    "nil details",
			result:  stageResult{},
			wantNil: true,
		},
		{
			name: "empty details",
			result: stageResult{
				Details: json.RawMessage(`{}`),
			},
			wantNil: true,
		},
		{
			name: "with stdout",
			result: stageResult{
				Details: json.RawMessage(`{"last_stdout": "some output"}`),
			},
			wantNil: false,
		},
		{
			name: "with stderr",
			result: stageResult{
				Details: json.RawMessage(`{"last_stderr": "some error"}`),
			},
			wantNil: false,
		},
		{
			name: "with error_context",
			result: stageResult{
				Details: json.RawMessage(`{"error_context": {"stderr": "context error"}}`),
			},
			wantNil: false,
		},
		{
			name: "with app_reported_error",
			result: stageResult{
				Details: json.RawMessage(`{"error_context": {"stderr": "test"}, "app_reported_error": {"message": "Bundled payload is missing"}}`),
			},
			wantNil: false,
		},
		{
			name: "invalid JSON",
			result: stageResult{
				Details: json.RawMessage(`invalid`),
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.getSmokeTestDetails()
			if tt.wantNil && got != nil {
				t.Error("Expected nil, got non-nil")
			}
			if !tt.wantNil && got == nil {
				t.Error("Expected non-nil, got nil")
			}
		})
	}
}

func TestAnalyzeStderr(t *testing.T) {
	tests := []struct {
		name       string
		stderr     string
		wantHint   bool
		wantSubstr string
	}{
		{
			name:     "empty stderr",
			stderr:   "",
			wantHint: false,
		},
		{
			name:       "go module error",
			stderr:     "unable to resolve paths for staleness",
			wantHint:   true,
			wantSubstr: "VROOLI_API_SKIP_STALE_CHECK",
		},
		{
			name:       "permission denied",
			stderr:     "Error: permission denied for file",
			wantHint:   true,
			wantSubstr: "permissions",
		},
		{
			name:       "GLIBC error",
			stderr:     "/lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.29' not found",
			wantHint:   true,
			wantSubstr: "GLIBC",
		},
		{
			name:       "missing file",
			stderr:     "Error: ENOENT: no such file or directory",
			wantHint:   true,
			wantSubstr: "Required file",
		},
		{
			name:       "shared library",
			stderr:     "error while loading shared libraries: libgtk-3.so.0: cannot open shared object file",
			wantHint:   true,
			wantSubstr: "library",
		},
		{
			name:       "connection refused",
			stderr:     "Error: connect ECONNREFUSED 127.0.0.1:8080",
			wantHint:   true,
			wantSubstr: "connection refused",
		},
		{
			name:       "timeout",
			stderr:     "Error: operation timed out",
			wantHint:   true,
			wantSubstr: "timeout",
		},
		{
			name:       "out of memory",
			stderr:     "Fatal error: out of memory",
			wantHint:   true,
			wantSubstr: "memory",
		},
		{
			name:       "segmentation fault",
			stderr:     "Segmentation fault (core dumped)",
			wantHint:   true,
			wantSubstr: "crash",
		},
		{
			name:     "unknown error",
			stderr:   "Some unrecognized error message",
			wantHint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzeStderr(tt.stderr)
			if tt.wantHint && got == "" {
				t.Error("Expected a hint, got empty string")
			}
			if !tt.wantHint && got != "" {
				t.Errorf("Expected no hint, got %q", got)
			}
			if tt.wantSubstr != "" && !containsSubstring(got, tt.wantSubstr) {
				t.Errorf("analyzeStderr() = %q, want substring %q", got, tt.wantSubstr)
			}
		})
	}
}

func TestExtractSmokeTestErrorHint(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		stderr string
		want   string
	}{
		{
			name:   "no error markers",
			stdout: "App started normally",
			stderr: "",
			want:   "",
		},
		{
			name:   "config error in stdout",
			stdout: `SMOKE_TEST_ERROR kind=config msg="Missing API key"`,
			stderr: "",
			want:   "Missing API key",
		},
		{
			name:   "validation error in stdout",
			stdout: `SMOKE_TEST_ERROR kind=validation msg="Invalid port number"`,
			stderr: "",
			want:   "Invalid port number",
		},
		{
			name:   "runtime error in stderr",
			stdout: "",
			stderr: `SMOKE_TEST_ERROR kind=runtime msg="Connection failed"`,
			want:   "Connection failed",
		},
		{
			name:   "config error takes priority",
			stdout: `SMOKE_TEST_ERROR kind=config msg="Config issue"`,
			stderr: `SMOKE_TEST_ERROR kind=runtime msg="Runtime issue"`,
			want:   "Config issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSmokeTestErrorHint(tt.stdout, tt.stderr)
			if got != tt.want {
				t.Errorf("extractSmokeTestErrorHint() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper functions
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

