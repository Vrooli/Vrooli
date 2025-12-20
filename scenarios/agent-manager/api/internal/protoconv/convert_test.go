// Package protoconv provides conversions between internal domain types and proto types.
package protoconv

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// =============================================================================
// RUNNER TYPE TESTS
// =============================================================================

func TestRunnerTypeToProto(t *testing.T) {
	tests := []struct {
		name     string
		input    domain.RunnerType
		expected pb.RunnerType
	}{
		{"claude-code", domain.RunnerTypeClaudeCode, pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE},
		{"codex", domain.RunnerTypeCodex, pb.RunnerType_RUNNER_TYPE_CODEX},
		{"opencode", domain.RunnerTypeOpenCode, pb.RunnerType_RUNNER_TYPE_OPENCODE},
		{"unknown", domain.RunnerType("unknown"), pb.RunnerType_RUNNER_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RunnerTypeToProto(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRunnerTypeFromProto(t *testing.T) {
	tests := []struct {
		name     string
		input    pb.RunnerType
		expected domain.RunnerType
	}{
		{"claude-code", pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE, domain.RunnerTypeClaudeCode},
		{"codex", pb.RunnerType_RUNNER_TYPE_CODEX, domain.RunnerTypeCodex},
		{"opencode", pb.RunnerType_RUNNER_TYPE_OPENCODE, domain.RunnerTypeOpenCode},
		{"unspecified", pb.RunnerType_RUNNER_TYPE_UNSPECIFIED, domain.RunnerTypeClaudeCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RunnerTypeFromProto(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// TASK STATUS TESTS
// =============================================================================

func TestTaskStatusRoundTrip(t *testing.T) {
	statuses := []domain.TaskStatus{
		domain.TaskStatusQueued,
		domain.TaskStatusRunning,
		domain.TaskStatusNeedsReview,
		domain.TaskStatusApproved,
		domain.TaskStatusRejected,
		domain.TaskStatusFailed,
		domain.TaskStatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			proto := TaskStatusToProto(status)
			result := TaskStatusFromProto(proto)
			if result != status {
				t.Errorf("round-trip failed: expected %v, got %v", status, result)
			}
		})
	}
}

// =============================================================================
// RUN STATUS TESTS
// =============================================================================

func TestRunStatusRoundTrip(t *testing.T) {
	statuses := []domain.RunStatus{
		domain.RunStatusPending,
		domain.RunStatusStarting,
		domain.RunStatusRunning,
		domain.RunStatusNeedsReview,
		domain.RunStatusComplete,
		domain.RunStatusFailed,
		domain.RunStatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			proto := RunStatusToProto(status)
			result := RunStatusFromProto(proto)
			if result != status {
				t.Errorf("round-trip failed: expected %v, got %v", status, result)
			}
		})
	}
}

// =============================================================================
// RUN PHASE TESTS
// =============================================================================

func TestRunPhaseRoundTrip(t *testing.T) {
	phases := []domain.RunPhase{
		domain.RunPhaseQueued,
		domain.RunPhaseInitializing,
		domain.RunPhaseSandboxCreating,
		domain.RunPhaseRunnerAcquiring,
		domain.RunPhaseExecuting,
		domain.RunPhaseCollectingResults,
		domain.RunPhaseAwaitingReview,
		domain.RunPhaseApplying,
		domain.RunPhaseCleaningUp,
		domain.RunPhaseCompleted,
	}

	for _, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			proto := RunPhaseToProto(phase)
			result := RunPhaseFromProto(proto)
			if result != phase {
				t.Errorf("round-trip failed: expected %v, got %v", phase, result)
			}
		})
	}
}

// =============================================================================
// HELPER FUNCTION TESTS
// =============================================================================

func TestUUIDConversions(t *testing.T) {
	t.Run("valid UUID round trip", func(t *testing.T) {
		original := uuid.New()
		str := UUIDToString(original)
		result := UUIDFromString(str)
		if result != original {
			t.Errorf("expected %v, got %v", original, result)
		}
	})

	t.Run("nil UUID returns empty string", func(t *testing.T) {
		str := UUIDToString(uuid.Nil)
		if str != "" {
			t.Errorf("expected empty string for nil UUID, got %q", str)
		}
	})

	t.Run("empty string returns nil UUID", func(t *testing.T) {
		result := UUIDFromString("")
		if result != uuid.Nil {
			t.Errorf("expected nil UUID, got %v", result)
		}
	})

	t.Run("invalid string returns nil UUID", func(t *testing.T) {
		result := UUIDFromString("not-a-uuid")
		if result != uuid.Nil {
			t.Errorf("expected nil UUID for invalid input, got %v", result)
		}
	})
}

func TestOptionalUUIDConversions(t *testing.T) {
	t.Run("valid pointer round trip", func(t *testing.T) {
		original := uuid.New()
		str := OptionalUUIDToString(&original)
		if str == nil {
			t.Fatal("expected non-nil string pointer")
		}
		result := OptionalStringToUUID(str)
		if result == nil || *result != original {
			t.Errorf("expected %v, got %v", original, result)
		}
	})

	t.Run("nil pointer returns nil", func(t *testing.T) {
		str := OptionalUUIDToString(nil)
		if str != nil {
			t.Errorf("expected nil for nil input, got %v", str)
		}
	})

	t.Run("nil UUID pointer returns nil", func(t *testing.T) {
		nilUUID := uuid.Nil
		str := OptionalUUIDToString(&nilUUID)
		if str != nil {
			t.Errorf("expected nil for nil UUID, got %v", str)
		}
	})
}

func TestTimestampConversions(t *testing.T) {
	t.Run("valid time round trip", func(t *testing.T) {
		original := time.Now().Truncate(time.Second)
		proto := TimestampToProto(original)
		result := TimestampFromProto(proto)
		if !result.Equal(original) {
			t.Errorf("expected %v, got %v", original, result)
		}
	})

	t.Run("zero time returns nil", func(t *testing.T) {
		proto := TimestampToProto(time.Time{})
		if proto != nil {
			t.Errorf("expected nil for zero time, got %v", proto)
		}
	})

	t.Run("nil proto returns zero time", func(t *testing.T) {
		result := TimestampFromProto(nil)
		if !result.IsZero() {
			t.Errorf("expected zero time for nil proto, got %v", result)
		}
	})
}

func TestDurationConversions(t *testing.T) {
	t.Run("valid duration round trip", func(t *testing.T) {
		original := 5 * time.Minute
		proto := DurationToProto(original)
		result := DurationFromProto(proto)
		if result != original {
			t.Errorf("expected %v, got %v", original, result)
		}
	})

	t.Run("zero duration returns nil", func(t *testing.T) {
		proto := DurationToProto(0)
		if proto != nil {
			t.Errorf("expected nil for zero duration, got %v", proto)
		}
	})

	t.Run("nil proto returns zero duration", func(t *testing.T) {
		result := DurationFromProto(nil)
		if result != 0 {
			t.Errorf("expected zero duration for nil proto, got %v", result)
		}
	})
}

// =============================================================================
// ENTITY CONVERSION TESTS
// =============================================================================

func TestAgentProfileRoundTrip(t *testing.T) {
	original := &domain.AgentProfile{
		ID:                   uuid.New(),
		Name:                 "test-profile",
		Description:          "A test profile",
		RunnerType:           domain.RunnerTypeClaudeCode,
		Model:                "claude-3-opus",
		MaxTurns:             100,
		Timeout:              10 * time.Minute,
		AllowedTools:         []string{"read", "write"},
		DeniedTools:          []string{"bash"},
		SkipPermissionPrompt: true,
		RequiresSandbox:      true,
		RequiresApproval:     false,
		AllowedPaths:         []string{"/src"},
		DeniedPaths:          []string{"/secrets"},
		CreatedBy:            "test-user",
		CreatedAt:            time.Now().Truncate(time.Second),
		UpdatedAt:            time.Now().Truncate(time.Second),
	}

	proto := AgentProfileToProto(original)
	result := AgentProfileFromProto(proto)

	if result.ID != original.ID {
		t.Errorf("ID: expected %v, got %v", original.ID, result.ID)
	}
	if result.Name != original.Name {
		t.Errorf("Name: expected %v, got %v", original.Name, result.Name)
	}
	if result.RunnerType != original.RunnerType {
		t.Errorf("RunnerType: expected %v, got %v", original.RunnerType, result.RunnerType)
	}
	if result.MaxTurns != original.MaxTurns {
		t.Errorf("MaxTurns: expected %v, got %v", original.MaxTurns, result.MaxTurns)
	}
	if result.SkipPermissionPrompt != original.SkipPermissionPrompt {
		t.Errorf("SkipPermissionPrompt: expected %v, got %v", original.SkipPermissionPrompt, result.SkipPermissionPrompt)
	}
}

func TestAgentProfileNilHandling(t *testing.T) {
	if AgentProfileToProto(nil) != nil {
		t.Error("expected nil for nil input")
	}
	if AgentProfileFromProto(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestTaskRoundTrip(t *testing.T) {
	original := &domain.Task{
		ID:          uuid.New(),
		Title:       "Test Task",
		Description: "A test task",
		ScopePath:   "src/",
		ProjectRoot: "/project",
		Status:      domain.TaskStatusQueued,
		CreatedBy:   "test-user",
		CreatedAt:   time.Now().Truncate(time.Second),
		UpdatedAt:   time.Now().Truncate(time.Second),
	}

	proto := TaskToProto(original)
	result := TaskFromProto(proto)

	if result.ID != original.ID {
		t.Errorf("ID: expected %v, got %v", original.ID, result.ID)
	}
	if result.Title != original.Title {
		t.Errorf("Title: expected %v, got %v", original.Title, result.Title)
	}
	if result.Status != original.Status {
		t.Errorf("Status: expected %v, got %v", original.Status, result.Status)
	}
}

func TestTaskNilHandling(t *testing.T) {
	if TaskToProto(nil) != nil {
		t.Error("expected nil for nil input")
	}
	if TaskFromProto(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestRunRoundTrip(t *testing.T) {
	profileID := uuid.New()
	startedAt := time.Now().Add(-time.Hour).Truncate(time.Second)

	original := &domain.Run{
		ID:              uuid.New(),
		TaskID:          uuid.New(),
		AgentProfileID:  &profileID,
		Tag:             "test-run",
		RunMode:         domain.RunModeSandboxed,
		Status:          domain.RunStatusRunning,
		Phase:           domain.RunPhaseExecuting,
		ProgressPercent: 50,
		IdempotencyKey:  "idem-123",
		ApprovalState:   domain.ApprovalStatePending,
		StartedAt:       &startedAt,
		CreatedAt:       time.Now().Truncate(time.Second),
		UpdatedAt:       time.Now().Truncate(time.Second),
	}

	proto := RunToProto(original)
	result := RunFromProto(proto)

	if result.ID != original.ID {
		t.Errorf("ID: expected %v, got %v", original.ID, result.ID)
	}
	if result.TaskID != original.TaskID {
		t.Errorf("TaskID: expected %v, got %v", original.TaskID, result.TaskID)
	}
	if result.AgentProfileID == nil || *result.AgentProfileID != *original.AgentProfileID {
		t.Errorf("AgentProfileID: expected %v, got %v", original.AgentProfileID, result.AgentProfileID)
	}
	if result.Status != original.Status {
		t.Errorf("Status: expected %v, got %v", original.Status, result.Status)
	}
	if result.Phase != original.Phase {
		t.Errorf("Phase: expected %v, got %v", original.Phase, result.Phase)
	}
	if result.ProgressPercent != original.ProgressPercent {
		t.Errorf("ProgressPercent: expected %v, got %v", original.ProgressPercent, result.ProgressPercent)
	}
}

func TestRunNilHandling(t *testing.T) {
	if RunToProto(nil) != nil {
		t.Error("expected nil for nil input")
	}
	if RunFromProto(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

// =============================================================================
// JSON SERIALIZATION TESTS
// =============================================================================

func TestMarshalUnmarshalJSON(t *testing.T) {
	profile := &pb.AgentProfile{
		Id:          uuid.New().String(),
		Name:        "test",
		Description: "test profile",
		RunnerType:  pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		MaxTurns:    100,
	}

	data, err := MarshalJSON(profile)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var result pb.AgentProfile
	if err := UnmarshalJSON(data, &result); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if result.Name != profile.Name {
		t.Errorf("Name: expected %v, got %v", profile.Name, result.Name)
	}
	if result.RunnerType != profile.RunnerType {
		t.Errorf("RunnerType: expected %v, got %v", profile.RunnerType, result.RunnerType)
	}
}
