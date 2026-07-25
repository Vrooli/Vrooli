// Package protoconv provides conversions between internal domain types and proto types.
package protoconv

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"

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
		{"unspecified", pb.RunnerType_RUNNER_TYPE_UNSPECIFIED, domain.RunnerType("")},
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

func TestExecutionModeToProto(t *testing.T) {
	tests := []struct {
		name     string
		input    domain.ExecutionMode
		expected pb.ExecutionMode
	}{
		{"codec_pipe", domain.ExecutionModeCodecPipe, pb.ExecutionMode_EXECUTION_MODE_CODEC_PIPE},
		{"interactive", domain.ExecutionModeInteractive, pb.ExecutionMode_EXECUTION_MODE_INTERACTIVE},
		{"empty normalizes to codec_pipe", domain.ExecutionMode(""), pb.ExecutionMode_EXECUTION_MODE_CODEC_PIPE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExecutionModeToProto(tt.input); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestExecutionModeFromProto(t *testing.T) {
	tests := []struct {
		name     string
		input    pb.ExecutionMode
		expected domain.ExecutionMode
	}{
		{"codec_pipe", pb.ExecutionMode_EXECUTION_MODE_CODEC_PIPE, domain.ExecutionModeCodecPipe},
		{"interactive", pb.ExecutionMode_EXECUTION_MODE_INTERACTIVE, domain.ExecutionModeInteractive},
		{"unspecified maps to empty", pb.ExecutionMode_EXECUTION_MODE_UNSPECIFIED, domain.ExecutionMode("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExecutionModeFromProto(tt.input); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// TestRunToProtoInteractiveFields verifies the interactive execution-mode
// surface (execution_mode, web_console_session_id, computed web_console_session_url)
// round-trips through the Run converters.
func TestRunToProtoInteractiveFields(t *testing.T) {
	r := &domain.Run{
		ID:                   uuid.New(),
		TaskID:               uuid.New(),
		ExecutionMode:        domain.ExecutionModeInteractive,
		WebConsoleSessionID:  "sess-123",
		WebConsoleSessionURL: "http://localhost:21233/?session=sess-123",
	}
	pbRun := RunToProto(r)
	if pbRun.ExecutionMode != pb.ExecutionMode_EXECUTION_MODE_INTERACTIVE {
		t.Errorf("execution_mode: expected INTERACTIVE, got %v", pbRun.ExecutionMode)
	}
	if pbRun.WebConsoleSessionId != "sess-123" {
		t.Errorf("web_console_session_id: expected sess-123, got %q", pbRun.WebConsoleSessionId)
	}
	if pbRun.WebConsoleSessionUrl != "http://localhost:21233/?session=sess-123" {
		t.Errorf("web_console_session_url: unexpected %q", pbRun.WebConsoleSessionUrl)
	}

	back := RunFromProto(pbRun)
	if back.ExecutionMode != domain.ExecutionModeInteractive {
		t.Errorf("round-trip execution_mode: expected interactive, got %q", back.ExecutionMode)
	}
	if back.WebConsoleSessionID != "sess-123" {
		t.Errorf("round-trip session id: got %q", back.WebConsoleSessionID)
	}
}

func TestRunResultProtoRoundTrip(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), Result: &domain.RunResult{
		FinalOutput: "canonical handoff",
		Selection: domain.FinalOutputSelection{
			Status: domain.FinalOutputSelectionSelected, SelectedCandidateID: "candidate-1",
			Rule: "unique_terminal_main_assistant", AlgorithmVersion: domain.FinalOutputResolverVersion,
			Evidence: []string{"event=result"},
		},
		Candidates: []domain.FinalOutputCandidate{{ID: "candidate-1", Content: "canonical handoff", MessageID: "msg-1", Terminal: true, EvidenceTier: 3}},
		Success:    true,
		Structured: &domain.StructuredResult{
			Status: domain.StructuredResultSuccess, SpecKind: domain.ResultSpecKindJSONSchema,
			SchemaDigest: "sha256:test", Value: json.RawMessage(`{"answer":"yes"}`), Method: "whole_document",
			Diagnostics: []domain.StructuredDiagnostic{{Code: "note", Path: "/answer", Message: "safe"}},
		},
	}}
	converted := RunFromProto(RunToProto(run))
	if converted.Result == nil || converted.Result.FinalOutput != "canonical handoff" {
		t.Fatalf("result round trip = %#v", converted.Result)
	}
	if converted.Result.Selection.Status != domain.FinalOutputSelectionSelected || len(converted.Result.Candidates) != 1 {
		t.Fatalf("selection round trip = %#v", converted.Result)
	}
	if converted.Result.Structured == nil || converted.Result.Structured.Status != domain.StructuredResultSuccess || string(converted.Result.Structured.Value) != `{"answer":"yes"}` {
		t.Fatalf("structured result round trip = %#v", converted.Result.Structured)
	}
}

func TestExecutionPolicySnapshotRoundTrip(t *testing.T) {
	original := &domain.RunConfig{
		RunnerType: domain.RunnerTypeCodex,
		Model:      "gpt-primary",
		ResultSpec: &domain.ResultSpec{Version: "result-spec/v1", Kind: domain.ResultSpecKindClassification, Schema: json.RawMessage(`{"enum":["yes","no"],"type":"string"}`), SchemaDigest: "sha256:spec", ExtractionMode: domain.StructuredExtractionConstrained, ExtractionRole: "extract.structured"},
		PolicySnapshot: &domain.ExecutionPolicySnapshot{
			CatalogDigest: "sha256:catalog-revision",
			RoleRef:       "code.smart",
			Candidates: []domain.ExecutionCandidate{
				{RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeModel, Model: "gpt-primary"},
				{RunnerType: domain.RunnerTypeClaudeCode, SelectionType: domain.ModelSelectionTypeRunnerDefault},
			},
			SelectedIndex: 1,
			SelectedCandidate: domain.ExecutionCandidate{
				RunnerType:    domain.RunnerTypeClaudeCode,
				SelectionType: domain.ModelSelectionTypeRunnerDefault,
			},
			Explanation: domain.PolicyResolutionExplanation{
				Source:           "portable_role",
				Summary:          "resolved before run creation",
				RequestedRoleRef: "code.smart",
				Preflight: []domain.CandidatePreflight{
					{
						Index:     0,
						Candidate: domain.ExecutionCandidate{RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeModel, Model: "gpt-primary"},
						Reason:    "model unavailable",
					},
					{
						Index:     1,
						Candidate: domain.ExecutionCandidate{RunnerType: domain.RunnerTypeClaudeCode, SelectionType: domain.ModelSelectionTypeRunnerDefault},
						Available: true,
					},
				},
			},
		},
	}

	protoConfig := RunConfigToProto(original)
	if protoConfig.PolicySnapshot == nil {
		t.Fatal("policy snapshot was dropped from proto config")
	}
	roundTrip := RunConfigFromProto(protoConfig)
	if !reflect.DeepEqual(roundTrip.PolicySnapshot, original.PolicySnapshot) {
		t.Fatalf("policy snapshot round trip mismatch\n got: %#v\nwant: %#v", roundTrip.PolicySnapshot, original.PolicySnapshot)
	}
	if !reflect.DeepEqual(roundTrip.ResultSpec, original.ResultSpec) {
		t.Fatalf("result spec round trip mismatch\n got: %#v\nwant: %#v", roundTrip.ResultSpec, original.ResultSpec)
	}
}

func TestSandboxConfigRoundTripPreservesLifecycleAcceptanceAndNilCriteria(t *testing.T) {
	trueValue := true
	original := &domain.SandboxConfig{
		Mode:           domain.SandboxModeProtected,
		ManualReview:   true,
		AutoApply:      &trueValue,
		ApplyOnFailure: &trueValue,
		NetworkMode:    domain.NetworkAccessLocalhost,
		NoLock:         true,
		Acceptance: domain.SandboxAcceptanceConfig{
			Mode:         "allowlist",
			Allow:        domain.SandboxFileCriteria{PathGlobs: []string{"src/**"}, Extensions: []string{".go"}},
			IgnoreBinary: true,
		},
		Lifecycle: domain.SandboxLifecycleConfig{
			CheckpointOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTurnCompleted, domain.SandboxLifecycleTurnFailed},
			StopOn:       []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRejected},
			DeleteOn:     []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal},
			TTL:          30 * time.Minute,
			IdleTimeout:  5 * time.Minute,
		},
	}
	converted := SandboxConfigToProto(original)
	if converted.GetAcceptance().GetDeny() != nil {
		t.Fatal("empty deny criteria must be omitted so it cannot mean deny everything")
	}
	if got := SandboxConfigFromProto(converted); got == nil || !reflect.DeepEqual(*got, *original) {
		t.Fatalf("sandbox config round trip mismatch\n got: %#v\nwant: %#v", got, *original)
	}
	if SandboxConfigToProto(nil) != nil || SandboxConfigFromProto(nil) != nil {
		t.Fatal("nil sandbox configs are not safely represented")
	}
}

func TestSandboxLifecycleEventsModesAndAcceptanceConvertersFailClosed(t *testing.T) {
	events := []domain.SandboxLifecycleEvent{
		domain.SandboxLifecycleTurnCompleted, domain.SandboxLifecycleTurnFailed, domain.SandboxLifecycleTurnCancelled,
		domain.SandboxLifecycleTerminal, domain.SandboxLifecycleRunCompleted, domain.SandboxLifecycleRunFailed,
		domain.SandboxLifecycleRunCancelled, domain.SandboxLifecycleApproved, domain.SandboxLifecycleRejected, domain.SandboxLifecycleTerminal,
	}
	for _, event := range events {
		if got := SandboxLifecycleEventFromProto(SandboxLifecycleEventToProto(event)); got != event {
			t.Fatalf("event %q round trip=%q", event, got)
		}
	}
	if SandboxLifecycleEventToProto("unknown") != pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_UNSPECIFIED || SandboxLifecycleEventFromProto(pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_UNSPECIFIED) != "" {
		t.Fatal("unknown lifecycle events must fail closed")
	}
	for _, mode := range []domain.SandboxMode{domain.SandboxModeOff, domain.SandboxModeTracking, domain.SandboxModeProtected} {
		if got := SandboxModeFromProto(SandboxModeToProto(mode)); got != mode {
			t.Fatalf("sandbox mode %q round trip=%q", mode, got)
		}
	}
	if SandboxModeFromProto(pb.SandboxMode_SANDBOX_MODE_UNSPECIFIED) != domain.SandboxModeUnspecified || SandboxAcceptanceModeFromProto(pb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_UNSPECIFIED) != "" || SandboxAcceptanceModeToProto("unknown") != pb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_UNSPECIFIED {
		t.Fatal("unknown sandbox policy values must not receive a permissive default")
	}
}

func TestRunEventTypeConversionsPreserveKnownEventsAndDefaultSafely(t *testing.T) {
	events := []domain.RunEventType{
		domain.EventTypeLog, domain.EventTypeMessage, domain.EventTypeMessageDeleted, domain.EventTypeToolCall,
		domain.EventTypeToolResult, domain.EventTypeStatus, domain.EventTypeMetric, domain.EventTypeArtifact,
		domain.EventTypeError, domain.EventTypeCompaction, domain.EventTypeLifecycle,
	}
	for _, event := range events {
		if got := RunEventTypeFromProto(RunEventTypeToProto(event)); got != event {
			t.Fatalf("event type %q round trip=%q", event, got)
		}
	}
	if RunEventTypeToProto(domain.RunEventType("unknown")) != pb.RunEventType_RUN_EVENT_TYPE_UNSPECIFIED {
		t.Fatal("unknown event type must not be exposed as a known API event")
	}
	if RunEventTypeFromProto(pb.RunEventType_RUN_EVENT_TYPE_UNSPECIFIED) != domain.EventTypeLog {
		t.Fatal("unspecified event type must retain the log-compatible default")
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

func TestTaskStatusFromProtoUnspecified(t *testing.T) {
	if got := TaskStatusFromProto(pb.TaskStatus_TASK_STATUS_UNSPECIFIED); got != "" {
		t.Fatalf("expected empty status for unspecified, got %q", got)
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
		domain.RunStatusParked,
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

func TestRunStatusFromProtoUnspecified(t *testing.T) {
	if got := RunStatusFromProto(pb.RunStatus_RUN_STATUS_UNSPECIFIED); got != "" {
		t.Fatalf("expected empty status for unspecified, got %q", got)
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
		ID:          uuid.New(),
		Name:        "test-profile",
		ProfileKey:  "test-profile-key",
		Description: "A test profile",

		MaxTurns:              100,
		Timeout:               10 * time.Minute,
		AllowedTools:          []string{"read", "write"},
		DeniedTools:           []string{"bash"},
		ToolRestrictionPolicy: domain.ToolRestrictionPolicyAdvisory,
		SkipPermissionPrompt:  true,
		AllowedPaths:          []string{"/src"},
		DeniedPaths:           []string{"/secrets"},
		CreatedBy:             "test-user",
		CreatedAt:             time.Now().Truncate(time.Second),
		UpdatedAt:             time.Now().Truncate(time.Second), RoleRef: "code.default",
	}

	proto := AgentProfileToProto(original)
	result := AgentProfileFromProto(proto)

	if result.ID != original.ID {
		t.Errorf("ID: expected %v, got %v", original.ID, result.ID)
	}
	if result.Name != original.Name {
		t.Errorf("Name: expected %v, got %v", original.Name, result.Name)
	}
	if result.ToolRestrictionPolicy != domain.ToolRestrictionPolicyAdvisory {
		t.Errorf("ToolRestrictionPolicy = %q", result.ToolRestrictionPolicy)
	}
	if result.ProfileKey != original.ProfileKey {
		t.Errorf("ProfileKey: expected %v, got %v", original.ProfileKey, result.ProfileKey)
	}
	if result.RoleRef != original.RoleRef {
		t.Errorf("RoleRef: expected %q, got %q", original.RoleRef, result.RoleRef)
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
		Actions: &domain.RunActions{
			CanInvestigate:        true,
			CanDelete:             true,
			CanContinue:           true,
			CanApplyInvestigation: false,
		},
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
	if result.Actions == nil || result.Actions.CanDelete != original.Actions.CanDelete {
		t.Errorf("Actions.CanDelete: expected %v, got %v", original.Actions.CanDelete, result.Actions)
	}
}

func TestRunAwaitHandleRoundTrip(t *testing.T) {
	registeredAt := time.Now().Add(-5 * time.Minute).Truncate(time.Second)
	deadline := time.Now().Add(25 * time.Minute).Truncate(time.Second)

	original := &domain.Run{
		ID:        uuid.New(),
		TaskID:    uuid.New(),
		Status:    domain.RunStatusParked,
		Phase:     domain.RunPhaseExecuting,
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		AwaitHandle: &domain.AwaitHandle{
			Producer:     "test-genie",
			Key:          "agent-manager/20260625-1",
			Deadline:     &deadline,
			RegisteredAt: registeredAt,
		},
	}

	result := RunFromProto(RunToProto(original))

	if result.AwaitHandle == nil {
		t.Fatalf("AwaitHandle: expected non-nil after round-trip")
	}
	if result.AwaitHandle.Producer != original.AwaitHandle.Producer {
		t.Errorf("Producer: expected %q, got %q", original.AwaitHandle.Producer, result.AwaitHandle.Producer)
	}
	if result.AwaitHandle.Key != original.AwaitHandle.Key {
		t.Errorf("Key: expected %q, got %q", original.AwaitHandle.Key, result.AwaitHandle.Key)
	}
	if result.AwaitHandle.Deadline == nil || !result.AwaitHandle.Deadline.Equal(*original.AwaitHandle.Deadline) {
		t.Errorf("Deadline: expected %v, got %v", original.AwaitHandle.Deadline, result.AwaitHandle.Deadline)
	}
	if !result.AwaitHandle.RegisteredAt.Equal(original.AwaitHandle.RegisteredAt) {
		t.Errorf("RegisteredAt: expected %v, got %v", original.AwaitHandle.RegisteredAt, result.AwaitHandle.RegisteredAt)
	}

	// A non-parked run carries no handle.
	if RunToProto(&domain.Run{ID: uuid.New()}).AwaitHandle != nil {
		t.Errorf("expected nil await_handle for a handle-less run")
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

func TestRunEventToProtoPayloads(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := uuid.New()

	t.Run("message provider evidence", func(t *testing.T) {
		event := domain.NewProviderMessageEvent(runID, "assistant", "done", domain.MessageEventData{
			MessageID: "msg-1", ConversationID: "session-1", TurnID: "turn-1",
			ProviderOrigin: "codex", CompletionReason: "turn_completed", Terminal: true,
			ParentMessageID: "", ProviderEventType: "turn.completed", RawEvidenceRef: "codex:turn.completed",
			EvidenceOnly: true, EvidenceForEventID: "event-1",
		})
		message := RunEventToProto(event).GetMessage()
		if message == nil || !message.Terminal || message.MessageId != "msg-1" || message.CompletionReason != "turn_completed" || !message.EvidenceOnly || message.EvidenceForEventId != "event-1" {
			t.Fatalf("message evidence proto = %#v", message)
		}
	})

	t.Run("tool call", func(t *testing.T) {
		event := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeToolCall,
			Timestamp: now,
			Sequence:  1,
			Data: &domain.ToolCallEventData{
				ToolName:   "Read",
				ToolCallID: "toolu_test_read",
				Input:      map[string]interface{}{"path": "README.md"},
			},
		}
		proto := RunEventToProto(event)
		payload := proto.GetToolCall()
		if payload == nil {
			t.Fatalf("expected tool_call payload, got nil")
		}
		if payload.ToolName != "Read" {
			t.Errorf("ToolName: expected Read, got %s", payload.ToolName)
		}
		if payload.ToolCallId != "toolu_test_read" {
			t.Errorf("ToolCallId: expected toolu_test_read, got %s", payload.ToolCallId)
		}
		if payload.Input == nil || payload.Input.AsMap()["path"] != "README.md" {
			t.Errorf("Input: expected path README.md, got %#v", payload.Input)
		}
	})

	t.Run("tool result", func(t *testing.T) {
		event := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeToolResult,
			Timestamp: now,
			Sequence:  2,
			Data: &domain.ToolResultEventData{
				ToolName:   "Write",
				ToolCallID: "toolu_123",
				Output:     "ok",
				Success:    true,
			},
		}
		proto := RunEventToProto(event)
		payload := proto.GetToolResult()
		if payload == nil {
			t.Fatalf("expected tool_result payload, got nil")
		}
		if payload.ToolCallId != "toolu_123" {
			t.Errorf("ToolCallId: expected toolu_123, got %s", payload.ToolCallId)
		}
		if !payload.Success {
			t.Errorf("Success: expected true, got false")
		}
	})

	t.Run("metric", func(t *testing.T) {
		event := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeMetric,
			Timestamp: now,
			Sequence:  3,
			Data: &domain.MetricEventData{
				Name:  "tokens_used",
				Value: 42,
				Unit:  "tokens",
				Tags:  map[string]string{"scope": "unit"},
			},
		}
		proto := RunEventToProto(event)
		payload := proto.GetMetric()
		if payload == nil {
			t.Fatalf("expected metric payload, got nil")
		}
		if payload.Name != "tokens_used" || payload.Unit != "tokens" {
			t.Errorf("Metric: expected tokens_used/tokens, got %s/%s", payload.Name, payload.Unit)
		}
		if !reflect.DeepEqual(payload.Tags, map[string]string{"scope": "unit"}) {
			t.Errorf("Tags: unexpected value %#v", payload.Tags)
		}
	})

	t.Run("artifact", func(t *testing.T) {
		event := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeArtifact,
			Timestamp: now,
			Sequence:  4,
			Data: &domain.ArtifactEventData{
				Type:     "diff",
				Path:     "/tmp/diff",
				Size:     123,
				MimeType: "text/plain",
			},
		}
		proto := RunEventToProto(event)
		payload := proto.GetArtifact()
		if payload == nil {
			t.Fatalf("expected artifact payload, got nil")
		}
		if payload.MimeType != "text/plain" {
			t.Errorf("MimeType: expected text/plain, got %s", payload.MimeType)
		}
	})

	t.Run("progress", func(t *testing.T) {
		event := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeStatus,
			Timestamp: now,
			Sequence:  5,
			Data: &domain.ProgressEventData{
				Phase:              domain.RunPhaseExecuting,
				PercentComplete:    60,
				CurrentAction:      "executing",
				TurnsCompleted:     2,
				TurnsTotal:         4,
				TokensUsed:         120,
				ElapsedSeconds:     10.5,
				EstimatedRemaining: 5.5,
			},
		}
		proto := RunEventToProto(event)
		payload := proto.GetProgress()
		if payload == nil {
			t.Fatalf("expected progress payload, got nil")
		}
		if payload.Phase != pb.RunPhase_RUN_PHASE_EXECUTING {
			t.Errorf("Phase: expected EXECUTING, got %v", payload.Phase)
		}
		if payload.PercentComplete != 60 {
			t.Errorf("PercentComplete: expected 60, got %d", payload.PercentComplete)
		}
	})

	t.Run("rate limit", func(t *testing.T) {
		reset := now.Add(2 * time.Minute)
		event := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeError,
			Timestamp: now,
			Sequence:  6,
			Data: &domain.RateLimitEventData{
				LimitType:   "daily",
				ResetTime:   &reset,
				RetryAfter:  120,
				CurrentUsed: 10,
				Limit:       12,
				Message:     "rate limited",
			},
		}
		proto := RunEventToProto(event)
		payload := proto.GetRateLimit()
		if payload == nil {
			t.Fatalf("expected rate_limit payload, got nil")
		}
		if payload.ResetTime == nil || !payload.ResetTime.AsTime().Equal(reset) {
			t.Errorf("ResetTime: expected %v, got %v", reset, payload.ResetTime)
		}
	})

	t.Run("cost", func(t *testing.T) {
		event := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeMetric,
			Timestamp: now,
			Sequence:  7,
			Data: &domain.CostEventData{
				InputTokens:           10,
				OutputTokens:          20,
				CacheCreationTokens:   1,
				CacheReadTokens:       2,
				TotalCostUSD:          0.12,
				ServiceTier:           "standard",
				Model:                 "o4-mini",
				WebSearchRequests:     3,
				ServerToolUseRequests: 4,
			},
		}
		proto := RunEventToProto(event)
		payload := proto.GetCost()
		if payload == nil {
			t.Fatalf("expected cost payload, got nil")
		}
		if payload.ServiceTier != "standard" || payload.Model != "o4-mini" {
			t.Errorf("Cost: expected standard/o4-mini, got %s/%s", payload.ServiceTier, payload.Model)
		}
	})

	t.Run("error", func(t *testing.T) {
		event := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeError,
			Timestamp: now,
			Sequence:  8,
			Data: &domain.ErrorEventData{
				Code:       "E_TEST",
				Message:    "failure",
				Retryable:  true,
				Recovery:   domain.RecoveryFixInput,
				StackTrace: "stack",
				Details:    map[string]interface{}{"field": "value"},
			},
		}
		proto := RunEventToProto(event)
		payload := proto.GetError()
		if payload == nil {
			t.Fatalf("expected error payload, got nil")
		}
		if payload.Recovery != pb.RecoveryAction_RECOVERY_ACTION_FIX_INPUT {
			t.Errorf("Recovery: expected FIX_INPUT, got %v", payload.Recovery)
		}
		if payload.Details == nil || payload.Details.AsMap()["field"] != "value" {
			t.Errorf("Details: expected field=value, got %#v", payload.Details)
		}
	})
}

// =============================================================================
// FEATURE FLAGS PROTO ROUND-TRIP TESTS
// =============================================================================

func TestFeatureFlagsRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		flags domain.FeatureFlags
	}{
		{
			name:  "zero value",
			flags: domain.FeatureFlags{},
		},
		{
			name:  "EnableBrowser true",
			flags: domain.FeatureFlags{EnableBrowser: true},
		},
		{
			name:  "EnableBrowser false",
			flags: domain.FeatureFlags{EnableBrowser: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := FeatureFlagsToProto(tt.flags)
			result := FeatureFlagsFromProto(proto)
			if result != tt.flags {
				t.Errorf("round-trip failed: expected %+v, got %+v", tt.flags, result)
			}
		})
	}
}

func TestFeatureFlagsToProto_ZeroReturnsNil(t *testing.T) {
	result := FeatureFlagsToProto(domain.FeatureFlags{})
	if result != nil {
		t.Errorf("expected nil for zero FeatureFlags, got %+v", result)
	}
}

func TestFeatureFlagsFromProto_NilReturnsZero(t *testing.T) {
	result := FeatureFlagsFromProto(nil)
	if result != (domain.FeatureFlags{}) {
		t.Errorf("expected zero FeatureFlags for nil, got %+v", result)
	}
}

// =============================================================================
// RUNNER EXTRA FLAGS PROTO ROUND-TRIP TESTS
// =============================================================================

func TestRunnerExtraFlagsRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		flags domain.RunnerExtraFlags
	}{
		{
			name:  "nil flags",
			flags: nil,
		},
		{
			name: "single runner single flag",
			flags: domain.RunnerExtraFlags{
				domain.RunnerTypeClaudeCode: []string{"--verbose"},
			},
		},
		{
			name: "single runner multiple flags",
			flags: domain.RunnerExtraFlags{
				domain.RunnerTypeClaudeCode: []string{"--verbose", "--allowedTools=Read,Write"},
			},
		},
		{
			name: "multiple runners",
			flags: domain.RunnerExtraFlags{
				domain.RunnerTypeClaudeCode: []string{"--verbose"},
				domain.RunnerTypeCodex:      []string{"--verbose"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := RunnerExtraFlagsToProto(tt.flags)
			result := RunnerExtraFlagsFromProto(proto)

			// Both nil and empty should be treated equivalently
			if len(tt.flags) == 0 && len(result) == 0 {
				return // Both nil/empty - OK
			}

			if len(result) != len(tt.flags) {
				t.Errorf("round-trip failed: expected %d runner types, got %d", len(tt.flags), len(result))
				return
			}

			for rt, expectedFlags := range tt.flags {
				gotFlags, ok := result[rt]
				if !ok {
					t.Errorf("round-trip failed: missing runner type %s", rt)
					continue
				}
				if len(gotFlags) != len(expectedFlags) {
					t.Errorf("round-trip failed for %s: expected %d flags, got %d", rt, len(expectedFlags), len(gotFlags))
					continue
				}
				for i, flag := range expectedFlags {
					if gotFlags[i] != flag {
						t.Errorf("round-trip failed for %s[%d]: expected %q, got %q", rt, i, flag, gotFlags[i])
					}
				}
			}
		})
	}
}

func TestRunnerExtraFlagsToProto_NilReturnsNil(t *testing.T) {
	result := RunnerExtraFlagsToProto(nil)
	if result != nil {
		t.Errorf("expected nil for nil RunnerExtraFlags, got %+v", result)
	}
}

func TestRunnerExtraFlagsToProto_EmptyReturnsNil(t *testing.T) {
	result := RunnerExtraFlagsToProto(domain.RunnerExtraFlags{})
	if result != nil {
		t.Errorf("expected nil for empty RunnerExtraFlags, got %+v", result)
	}
}

func TestRunnerExtraFlagsFromProto_NilReturnsNil(t *testing.T) {
	result := RunnerExtraFlagsFromProto(nil)
	if result != nil {
		t.Errorf("expected nil for nil proto, got %+v", result)
	}
}

// =============================================================================
// AGENT PROFILE WITH FEATURES ROUND-TRIP TEST
// =============================================================================

func TestAgentProfileWithFeaturesRoundTrip(t *testing.T) {
	original := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "features-profile",
		ProfileKey: "features-key",

		Features: domain.FeatureFlags{EnableBrowser: true},
		ExtraFlags: domain.RunnerExtraFlags{
			domain.RunnerTypeClaudeCode: []string{"--verbose", "--allowedTools"},
			domain.RunnerTypeCodex:      []string{"--verbose"},
		},
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second), RoleRef: "code.default",
	}

	proto := AgentProfileToProto(original)
	result := AgentProfileFromProto(proto)

	// Verify features round-trip
	if result.Features.EnableBrowser != original.Features.EnableBrowser {
		t.Errorf("Features.EnableBrowser: expected %v, got %v", original.Features.EnableBrowser, result.Features.EnableBrowser)
	}

	// Verify extra flags round-trip
	if len(result.ExtraFlags) != len(original.ExtraFlags) {
		t.Errorf("ExtraFlags: expected %d runner types, got %d", len(original.ExtraFlags), len(result.ExtraFlags))
	}

	for rt, expectedFlags := range original.ExtraFlags {
		gotFlags, ok := result.ExtraFlags[rt]
		if !ok {
			t.Errorf("ExtraFlags: missing runner type %s", rt)
			continue
		}
		if !reflect.DeepEqual(gotFlags, expectedFlags) {
			t.Errorf("ExtraFlags[%s]: expected %v, got %v", rt, expectedFlags, gotFlags)
		}
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

		MaxTurns: 100, RoleRef: "code.default",
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
	if result.RoleRef != profile.RoleRef {
		t.Errorf("RoleRef: expected %q, got %q", profile.RoleRef, result.RoleRef)
	}
}

// =============================================================================
// DIFF CONVERSION TESTS
// =============================================================================

func TestDiffResultToProto_PatchSuffixMatch(t *testing.T) {
	// Regression test: workspace-sandbox returns scope-relative file paths
	// (e.g. "api/main.go") but the unified diff uses project-root-relative
	// paths (e.g. "scenarios/foo/api/main.go"). DiffResultToProto must match
	// them via suffix so that per-file patches are populated.
	runID := uuid.New()
	fileID := uuid.New()

	unified := `diff --git a/scenarios/my-app/api/main.go b/scenarios/my-app/api/main.go
--- a/scenarios/my-app/api/main.go
+++ b/scenarios/my-app/api/main.go
@@ -1,3 +1,3 @@
 package main
-// old
+// new
 func main() {}
`
	dr := &DiffResult{
		SandboxID:   uuid.New(),
		UnifiedDiff: unified,
		Files: []FileChange{
			{
				ID:         fileID,
				FilePath:   "api/main.go", // scope-relative
				ChangeType: "modified",
				LinesAdded: 1,
			},
		},
	}

	proto := DiffResultToProto(runID, dr)
	if proto == nil {
		t.Fatal("expected non-nil RunDiff")
	}
	if len(proto.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(proto.Files))
	}
	if proto.Files[0].Patch == "" {
		t.Error("expected non-empty patch for suffix-matched file, got empty string")
	}
}

func TestDiffResultToProto_ExactPathMatch(t *testing.T) {
	runID := uuid.New()
	unified := `diff --git a/api/main.go b/api/main.go
--- a/api/main.go
+++ b/api/main.go
@@ -1 +1 @@
-old
+new
`
	dr := &DiffResult{
		UnifiedDiff: unified,
		Files: []FileChange{
			{FilePath: "api/main.go", ChangeType: "modified"},
		},
	}

	proto := DiffResultToProto(runID, dr)
	if proto.Files[0].Patch == "" {
		t.Error("expected non-empty patch for exact-matched file")
	}
}

func TestDiffResultToProto_NilResult(t *testing.T) {
	if DiffResultToProto(uuid.New(), nil) != nil {
		t.Error("expected nil for nil DiffResult")
	}
}

func TestDiffResultToProto_EmptyDiff(t *testing.T) {
	dr := &DiffResult{
		UnifiedDiff: "",
		Files: []FileChange{
			{FilePath: "api/main.go", ChangeType: "modified"},
		},
	}
	proto := DiffResultToProto(uuid.New(), dr)
	if proto.Files[0].Patch != "" {
		t.Error("expected empty patch when unified diff is empty")
	}
}

func TestLookupPatch(t *testing.T) {
	m := map[string]string{
		"scenarios/foo/api/main.go":     "patch-a",
		"scenarios/foo/api/handlers.go": "patch-b",
	}

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{"suffix match", "api/main.go", "patch-a"},
		{"exact match", "scenarios/foo/api/main.go", "patch-a"},
		{"no match", "api/other.go", ""},
		{"root-level file suffix match", "main.go", "patch-a"}, // /main.go suffix matches
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lookupPatch(m, tt.filePath)
			if got != tt.want {
				t.Errorf("lookupPatch(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}
