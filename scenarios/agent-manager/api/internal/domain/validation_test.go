package domain

import (
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// AGENT PROFILE VALIDATION TESTS
// =============================================================================

func TestAgentProfile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		profile *AgentProfile
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid profile",
			profile: &AgentProfile{
				Name:       "test-profile",
				RunnerType: RunnerTypeClaudeCode,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			profile: &AgentProfile{
				Name:       "",
				RunnerType: RunnerTypeClaudeCode,
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "whitespace only name",
			profile: &AgentProfile{
				Name:       "   ",
				RunnerType: RunnerTypeClaudeCode,
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "invalid runner type",
			profile: &AgentProfile{
				Name:       "test",
				RunnerType: "invalid-runner",
			},
			wantErr: true,
			errMsg:  "runnerType",
		},
		{
			name: "negative max turns",
			profile: &AgentProfile{
				Name:       "test",
				RunnerType: RunnerTypeClaudeCode,
				MaxTurns:   -1,
			},
			wantErr: true,
			errMsg:  "maxTurns",
		},
		{
			name: "negative timeout",
			profile: &AgentProfile{
				Name:       "test",
				RunnerType: RunnerTypeClaudeCode,
				Timeout:    -1,
			},
			wantErr: true,
			errMsg:  "timeout",
		},
		{
			name: "overlapping allowed/denied tools",
			profile: &AgentProfile{
				Name:         "test",
				RunnerType:   RunnerTypeClaudeCode,
				AllowedTools: []string{"Read", "Write"},
				DeniedTools:  []string{"Write", "Bash"},
			},
			wantErr: true,
			errMsg:  "Tools",
		},
		{
			name: "overlapping allowed/denied paths",
			profile: &AgentProfile{
				Name:         "test",
				RunnerType:   RunnerTypeClaudeCode,
				AllowedPaths: []string{"src/", "tests/"},
				DeniedPaths:  []string{"tests/", "docs/"},
			},
			wantErr: true,
			errMsg:  "Paths",
		},
		{
			name: "zero values are valid defaults",
			profile: &AgentProfile{
				Name:       "test",
				RunnerType: RunnerTypeCodex,
				MaxTurns:   0, // 0 = unlimited
				Timeout:    0, // 0 = use default
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if ve, ok := err.(*ValidationError); ok {
					if tt.errMsg != "" && ve.Field != tt.errMsg && !containsSubstring(ve.Field, tt.errMsg) {
						t.Errorf("Validate() error field = %v, want containing %v", ve.Field, tt.errMsg)
					}
				}
			}
		})
	}
}

// =============================================================================
// TASK VALIDATION TESTS
// =============================================================================

func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    *Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid task",
			task: &Task{
				Title:     "Fix bug in login",
				ScopePath: "src/auth",
			},
			wantErr: false,
		},
		{
			name: "empty title",
			task: &Task{
				Title:     "",
				ScopePath: "src/",
			},
			wantErr: true,
			errMsg:  "title",
		},
		{
			name: "empty scope path",
			task: &Task{
				Title:     "Fix bug",
				ScopePath: "",
			},
			wantErr: true,
			errMsg:  "scopePath",
		},
		{
			name: "path traversal in scope",
			task: &Task{
				Title:     "Fix bug",
				ScopePath: "../outside",
			},
			wantErr: true,
			errMsg:  "scopePath",
		},
		{
			name: "root scope is valid",
			task: &Task{
				Title:     "Full repo task",
				ScopePath: "/",
			},
			wantErr: false,
		},
		{
			name: "dot scope is valid",
			task: &Task{
				Title:     "Current dir task",
				ScopePath: ".",
			},
			wantErr: false,
		},
		{
			name: "valid context attachment",
			task: &Task{
				Title:     "Task with context",
				ScopePath: "src/",
				ContextAttachments: []ContextAttachment{
					{Type: "file", Path: "/path/to/file.txt"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid context attachment type",
			task: &Task{
				Title:     "Task with bad context",
				ScopePath: "src/",
				ContextAttachments: []ContextAttachment{
					{Type: "invalid", Path: "/path"},
				},
			},
			wantErr: true,
			errMsg:  "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// RUN VALIDATION TESTS
// =============================================================================

func TestRun_Validate(t *testing.T) {
	validTaskID := uuid.New()
	validProfileID := uuid.New()

	tests := []struct {
		name    string
		run     *Run
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid run",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: &validProfileID,
				RunMode:        RunModeSandboxed,
			},
			wantErr: false,
		},
		{
			name: "nil task ID",
			run: &Run{
				TaskID:         uuid.Nil,
				AgentProfileID: &validProfileID,
				RunMode:        RunModeSandboxed,
			},
			wantErr: true,
			errMsg:  "taskId",
		},
		{
			name: "nil profile ID and no config",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: nil, // No profile
				ResolvedConfig: nil, // No inline config either
				RunMode:        RunModeSandboxed,
			},
			wantErr: true,
			errMsg:  "agentProfileId",
		},
		{
			name: "valid with resolved config instead of profile",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: nil,                // No profile
				ResolvedConfig: DefaultRunConfig(), // Inline config provided
				RunMode:        RunModeSandboxed,
			},
			wantErr: false,
		},
		{
			name: "invalid run mode",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: &validProfileID,
				RunMode:        "invalid",
			},
			wantErr: true,
			errMsg:  "runMode",
		},
		{
			name: "in_place mode is valid",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: &validProfileID,
				RunMode:        RunModeInPlace,
			},
			wantErr: false,
		},
		{
			name: "invalid status",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: &validProfileID,
				RunMode:        RunModeSandboxed,
				Status:         "invalid_status",
			},
			wantErr: true,
			errMsg:  "status",
		},
		{
			name: "valid status",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: &validProfileID,
				RunMode:        RunModeSandboxed,
				Status:         RunStatusPending,
			},
			wantErr: false,
		},
		{
			name: "progress percent out of range",
			run: &Run{
				TaskID:          validTaskID,
				AgentProfileID:  &validProfileID,
				RunMode:         RunModeSandboxed,
				ProgressPercent: 150,
			},
			wantErr: true,
			errMsg:  "progressPercent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRun_ValidateForCreation(t *testing.T) {
	validTaskID := uuid.New()
	validProfileID := uuid.New()

	tests := []struct {
		name    string
		run     *Run
		wantErr bool
	}{
		{
			name: "valid new run",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: &validProfileID,
				RunMode:        RunModeSandboxed,
				Status:         RunStatusPending,
				Phase:          RunPhaseQueued,
			},
			wantErr: false,
		},
		{
			name: "non-pending status",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: &validProfileID,
				RunMode:        RunModeSandboxed,
				Status:         RunStatusRunning,
			},
			wantErr: true,
		},
		{
			name: "non-queued phase",
			run: &Run{
				TaskID:         validTaskID,
				AgentProfileID: &validProfileID,
				RunMode:        RunModeSandboxed,
				Phase:          RunPhaseExecuting,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run.ValidateForCreation()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateForCreation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// POLICY VALIDATION TESTS
// =============================================================================

func TestPolicy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		policy  *Policy
		wantErr bool
	}{
		{
			name: "valid policy",
			policy: &Policy{
				Name:     "default-policy",
				Priority: 100,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			policy: &Policy{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "negative priority",
			policy: &Policy{
				Name:     "test",
				Priority: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPolicyRules_Validate(t *testing.T) {
	intPtr := func(i int) *int { return &i }
	int64Ptr := func(i int64) *int64 { return &i }

	tests := []struct {
		name    string
		rules   *PolicyRules
		wantErr bool
	}{
		{
			name:    "empty rules are valid",
			rules:   &PolicyRules{},
			wantErr: false,
		},
		{
			name: "zero max concurrent runs",
			rules: &PolicyRules{
				MaxConcurrentRuns: intPtr(0),
			},
			wantErr: true,
		},
		{
			name: "zero max files changed",
			rules: &PolicyRules{
				MaxFilesChanged: intPtr(0),
			},
			wantErr: true,
		},
		{
			name: "execution time too short",
			rules: &PolicyRules{
				MaxExecutionTimeMs: int64Ptr(500), // Less than 1 second
			},
			wantErr: true,
		},
		{
			name: "conflicting runner lists",
			rules: &PolicyRules{
				AllowedRunners: []RunnerType{RunnerTypeClaudeCode, RunnerTypeCodex},
				DeniedRunners:  []RunnerType{RunnerTypeCodex, RunnerTypeOpenCode},
			},
			wantErr: true,
		},
		{
			name: "valid rules",
			rules: &PolicyRules{
				MaxConcurrentRuns:  intPtr(5),
				MaxFilesChanged:    intPtr(100),
				MaxExecutionTimeMs: int64Ptr(60000), // 1 minute
				AllowedRunners:     []RunnerType{RunnerTypeClaudeCode},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rules.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// STATUS VALIDITY TESTS
// =============================================================================

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   bool
	}{
		{TaskStatusQueued, true},
		{TaskStatusRunning, true},
		{TaskStatusNeedsReview, true},
		{TaskStatusApproved, true},
		{TaskStatusRejected, true},
		{TaskStatusFailed, true},
		{TaskStatusCancelled, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunStatus_IsValid(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   bool
	}{
		{RunStatusPending, true},
		{RunStatusStarting, true},
		{RunStatusRunning, true},
		{RunStatusNeedsReview, true},
		{RunStatusComplete, true},
		{RunStatusFailed, true},
		{RunStatusCancelled, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   bool
	}{
		{RunStatusPending, false},
		{RunStatusStarting, false},
		{RunStatusRunning, false},
		{RunStatusNeedsReview, false},
		{RunStatusComplete, true},
		{RunStatusFailed, true},
		{RunStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunStatus_IsActive(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   bool
	}{
		{RunStatusPending, true},
		{RunStatusStarting, true},
		{RunStatusRunning, true},
		{RunStatusNeedsReview, false},
		{RunStatusComplete, false},
		{RunStatusFailed, false},
		{RunStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApprovalState_IsValid(t *testing.T) {
	tests := []struct {
		state ApprovalState
		want  bool
	}{
		{ApprovalStateNone, true},
		{ApprovalStatePending, true},
		{ApprovalStatePartiallyApproved, true},
		{ApprovalStateApproved, true},
		{ApprovalStateRejected, true},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunPhase_IsValid(t *testing.T) {
	tests := []struct {
		phase RunPhase
		want  bool
	}{
		{RunPhaseQueued, true},
		{RunPhaseInitializing, true},
		{RunPhaseSandboxCreating, true},
		{RunPhaseRunnerAcquiring, true},
		{RunPhaseExecuting, true},
		{RunPhaseCollectingResults, true},
		{RunPhaseAwaitingReview, true},
		{RunPhaseApplying, true},
		{RunPhaseCleaningUp, true},
		{RunPhaseCompleted, true},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			if got := tt.phase.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// CONTEXT ATTACHMENT VALIDATION TESTS
// =============================================================================

func TestContextAttachment_Validate(t *testing.T) {
	tests := []struct {
		name       string
		attachment *ContextAttachment
		wantErr    bool
	}{
		{
			name:       "valid file attachment",
			attachment: &ContextAttachment{Type: "file", Path: "/path/to/file.txt"},
			wantErr:    false,
		},
		{
			name:       "valid link attachment",
			attachment: &ContextAttachment{Type: "link", URL: "https://example.com"},
			wantErr:    false,
		},
		{
			name:       "valid note attachment",
			attachment: &ContextAttachment{Type: "note", Content: "Some notes here"},
			wantErr:    false,
		},
		{
			name:       "invalid type",
			attachment: &ContextAttachment{Type: "unknown"},
			wantErr:    true,
		},
		{
			name:       "file without path",
			attachment: &ContextAttachment{Type: "file"},
			wantErr:    true,
		},
		{
			name:       "link without url",
			attachment: &ContextAttachment{Type: "link"},
			wantErr:    true,
		},
		{
			name:       "note without content",
			attachment: &ContextAttachment{Type: "note"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.attachment.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// HELPER TESTS
// =============================================================================

func Test_hasStringOverlap(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"no overlap", []string{"a", "b"}, []string{"c", "d"}, false},
		{"has overlap", []string{"a", "b"}, []string{"b", "c"}, true},
		{"empty a", []string{}, []string{"a"}, false},
		{"empty b", []string{"a"}, []string{}, false},
		{"both empty", []string{}, []string{}, false},
		{"nil slices", nil, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasStringOverlap(tt.a, tt.b); got != tt.want {
				t.Errorf("hasStringOverlap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasRunnerOverlap(t *testing.T) {
	tests := []struct {
		name string
		a    []RunnerType
		b    []RunnerType
		want bool
	}{
		{"no overlap", []RunnerType{RunnerTypeClaudeCode}, []RunnerType{RunnerTypeCodex}, false},
		{"has overlap", []RunnerType{RunnerTypeClaudeCode, RunnerTypeCodex}, []RunnerType{RunnerTypeCodex}, true},
		{"empty", []RunnerType{}, []RunnerType{RunnerTypeClaudeCode}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasRunnerOverlap(tt.a, tt.b); got != tt.want {
				t.Errorf("hasRunnerOverlap() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for substring matching
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsSubstring(s[1:], substr)))
}
