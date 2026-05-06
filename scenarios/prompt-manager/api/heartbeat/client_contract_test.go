package heartbeat

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestCreateTaskRequest_ProtojsonCompatibility verifies that the CreateTaskRequest
// JSON envelope matches what agent-manager's protojson unmarshaller expects.
func TestCreateTaskRequest_ProtojsonCompatibility(t *testing.T) {
	req := CreateTaskRequest{
		Task: &Task{
			Title:       "Heartbeat: team-1/agent-1",
			Description: "Execute heartbeat prompt",
			ScopePath:   "/home/user/vrooli",
			ProjectRoot: "/home/user/vrooli",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Parse into raw map to check field names
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Must have "task" envelope
	taskRaw, ok := fields["task"]
	if !ok {
		t.Fatal("expected top-level 'task' field in CreateTaskRequest JSON")
	}

	// Verify task fields are snake_case
	var taskFields map[string]json.RawMessage
	if err := json.Unmarshal(taskRaw, &taskFields); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}

	allowed := map[string]bool{
		"id": true, "title": true, "description": true,
		"scope_path": true, "project_root": true,
	}
	for key := range taskFields {
		if !allowed[key] {
			t.Errorf("unexpected field %q in Task JSON — protojson with DiscardUnknown=false will reject this", key)
		}
	}

	// Required fields must be present
	for _, required := range []string{"title", "description", "scope_path"} {
		if _, ok := taskFields[required]; !ok {
			t.Errorf("required field %q missing from Task JSON", required)
		}
	}

	// Verify no unknown top-level fields
	for key := range fields {
		if key != "task" {
			t.Errorf("unexpected top-level field %q — agent-manager expects only 'task'", key)
		}
	}
}

// TestCreateRunRequest_ProtojsonCompatibility verifies that the CreateRunRequest
// JSON matches the agent-manager proto schema.
func TestCreateRunRequest_ProtojsonCompatibility(t *testing.T) {
	tag := "heartbeat-team-1-agent-1-2025-01-01T00-00-00Z"
	req := CreateRunRequest{
		TaskID: "task-100",
		ProfileRef: &ProfileRef{
			ProfileKey: "prompt-manager-heartbeat",
		},
		Tag: &tag,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	allowed := map[string]bool{
		"task_id": true, "profile_ref": true, "tag": true,
	}
	for key := range fields {
		if !allowed[key] {
			t.Errorf("unexpected field %q in CreateRunRequest JSON", key)
		}
	}

	// Verify profile_ref structure
	refRaw, ok := fields["profile_ref"]
	if !ok {
		t.Fatal("expected 'profile_ref' field")
	}
	var refFields map[string]json.RawMessage
	if err := json.Unmarshal(refRaw, &refFields); err != nil {
		t.Fatalf("unmarshal profile_ref: %v", err)
	}
	allowedRef := map[string]bool{"profile_key": true, "defaults": true}
	for key := range refFields {
		if !allowedRef[key] {
			t.Errorf("unexpected field %q in ProfileRef JSON", key)
		}
	}
}

func TestInvestigateRunRequest_JSONFieldNames(t *testing.T) {
	req := InvestigateRunRequest{
		RunIDs:        []string{"11111111-1111-1111-1111-111111111111"},
		Depth:         "standard",
		CustomContext: "check",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	allowed := map[string]bool{
		"runIds":        true,
		"depth":         true,
		"customContext": true,
	}
	for key := range fields {
		if !allowed[key] {
			t.Errorf("unexpected field %q in InvestigateRunRequest JSON", key)
		}
	}
	for key := range allowed {
		if _, ok := fields[key]; !ok {
			t.Errorf("expected field %q missing from InvestigateRunRequest JSON", key)
		}
	}
}

func TestInvestigationApplyRequest_JSONFieldNames(t *testing.T) {
	req := InvestigationApplyRequest{
		InvestigationRunID: "11111111-1111-1111-1111-111111111111",
		CustomContext:      "apply",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	allowed := map[string]bool{
		"investigationRunId": true,
		"customContext":      true,
	}
	for key := range fields {
		if !allowed[key] {
			t.Errorf("unexpected field %q in InvestigationApplyRequest JSON", key)
		}
	}
	for key := range allowed {
		if _, ok := fields[key]; !ok {
			t.Errorf("expected field %q missing from InvestigationApplyRequest JSON", key)
		}
	}
}

// TestCreateRunRequest_WithProfileRefDefaults verifies ProfileRef with Defaults
// serializes correctly.
func TestCreateRunRequest_WithProfileRefDefaults(t *testing.T) {
	req := CreateRunRequest{
		TaskID: "task-1",
		ProfileRef: &ProfileRef{
			ProfileKey: "custom",
			Defaults: &AgentProfile{
				Name:       "Custom Agent",
				ProfileKey: "custom",
				RunnerType: "RUNNER_TYPE_CODEX",
				Model:      "opus",
				MaxTurns:   50,
				Timeout:    DurationToProtojson(5 * time.Minute),
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Round-trip: unmarshal back and verify
	var decoded CreateRunRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ProfileRef == nil || decoded.ProfileRef.Defaults == nil {
		t.Fatal("expected profile_ref with defaults after round-trip")
	}
	d := decoded.ProfileRef.Defaults
	if d.Name != "Custom Agent" {
		t.Errorf("expected name 'Custom Agent', got %q", d.Name)
	}
	if d.RunnerType != "RUNNER_TYPE_CODEX" {
		t.Errorf("expected runner_type 'RUNNER_TYPE_CODEX', got %q", d.RunnerType)
	}
	if d.MaxTurns != 50 {
		t.Errorf("expected max_turns 50, got %d", d.MaxTurns)
	}
	if d.Timeout != "300s" {
		t.Errorf("expected timeout '300s', got %q", d.Timeout)
	}

	// Verify timeout is a Duration string in the raw JSON, not a number
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawFields); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	var refFields map[string]json.RawMessage
	if err := json.Unmarshal(rawFields["profile_ref"], &refFields); err != nil {
		t.Fatalf("unmarshal profile_ref: %v", err)
	}
	var defaultsFields map[string]json.RawMessage
	if err := json.Unmarshal(refFields["defaults"], &defaultsFields); err != nil {
		t.Fatalf("unmarshal defaults: %v", err)
	}
	timeoutRaw := string(defaultsFields["timeout"])
	if !strings.HasPrefix(timeoutRaw, `"`) {
		t.Errorf("timeout must be a JSON string (protojson Duration), got raw: %s", timeoutRaw)
	}
}

func TestDefaultHeartbeatProfilesUseSandboxProtoEnums(t *testing.T) {
	for _, tt := range []struct {
		name        string
		profileKey  string
		runtimeMode string
	}{
		{"codex", DefaultProfileKeyCodex, "multi-process"},
		{"claude-code", DefaultProfileKeyClaudeCode, "single-process"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			profile := BuildDefaultProfileForRuntimeMode(tt.profileKey, tt.runtimeMode)
			if profile.SandboxConfig == nil {
				t.Fatal("expected sandbox config")
			}
			if profile.SandboxConfig.Mode != "SANDBOX_MODE_PROTECTED" {
				t.Fatalf("sandbox mode = %q, want SANDBOX_MODE_PROTECTED", profile.SandboxConfig.Mode)
			}
		})
	}
}

// TestRunResponse_ErrorFieldDeserialization verifies that the error_msg field
// from agent-manager responses is correctly mapped to the Run.Error field.
func TestRunResponse_ErrorFieldDeserialization(t *testing.T) {
	// Simulate a response from agent-manager with error_msg field
	responseJSON := `{
		"run": {
			"id": "run-123",
			"task_id": "task-456",
			"status": "RUN_STATUS_FAILED",
			"error_msg": "agent crashed: out of memory"
		}
	}`

	var resp GetRunResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Run == nil {
		t.Fatal("expected run in response")
	}
	if resp.Run.Error != "agent crashed: out of memory" {
		t.Errorf("expected error_msg to map to Error field, got %q", resp.Run.Error)
	}
	if resp.Run.Status != "RUN_STATUS_FAILED" {
		t.Errorf("expected RUN_STATUS_FAILED, got %q", resp.Run.Status)
	}
}

// TestGetRunResponse_TerminalStatusMapping verifies all terminal statuses from
// proto are recognized by IsTerminalStatus.
func TestGetRunResponse_TerminalStatusMapping(t *testing.T) {
	terminal := []string{
		"RUN_STATUS_COMPLETE",
		"RUN_STATUS_FAILED",
		"RUN_STATUS_CANCELLED",
		"complete",
		"failed",
		"cancelled",
	}
	for _, s := range terminal {
		if !IsTerminalStatus(s) {
			t.Errorf("expected %q to be terminal", s)
		}
	}

	nonTerminal := []string{
		"RUN_STATUS_RUNNING",
		"RUN_STATUS_PENDING",
		"running",
		"pending",
		"",
	}
	for _, s := range nonTerminal {
		if IsTerminalStatus(s) {
			t.Errorf("expected %q to NOT be terminal", s)
		}
	}
}

// TestEnsureProfileRequest_ProtojsonCompatibility verifies that the EnsureProfileRequest
// JSON matches what agent-manager's protojson unmarshaller expects, particularly that
// the Timeout field uses protojson Duration string format, not Go nanosecond integers.
func TestEnsureProfileRequest_ProtojsonCompatibility(t *testing.T) {
	req := EnsureProfileRequest{
		ProfileKey: "prompt-manager-heartbeat",
		Defaults: &AgentProfile{
			Name:                 "Prompt Manager Heartbeat",
			ProfileKey:           "prompt-manager-heartbeat",
			Description:          "Profile for team member heartbeat execution",
			RunnerType:           "RUNNER_TYPE_CODEX",
			ModelPreset:          "MODEL_PRESET_SMART",
			MaxTurns:             50,
			Timeout:              DurationToProtojson(10 * time.Minute),
			AllowedTools:         []string{"read_file", "write_file", "execute_command"},
			SkipPermissionPrompt: true,
			CreatedBy:            "prompt-manager",
		},
		UpdateExisting: false,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Parse into raw map to check field names
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify top-level fields are snake_case and expected
	allowedTopLevel := map[string]bool{
		"profile_key": true, "defaults": true, "update_existing": true,
	}
	for key := range fields {
		if !allowedTopLevel[key] {
			t.Errorf("unexpected top-level field %q in EnsureProfileRequest JSON — protojson with DiscardUnknown=false will reject this", key)
		}
	}

	// Verify defaults structure
	defaultsRaw, ok := fields["defaults"]
	if !ok {
		t.Fatal("expected 'defaults' field in EnsureProfileRequest")
	}

	var defaultsFields map[string]json.RawMessage
	if err := json.Unmarshal(defaultsRaw, &defaultsFields); err != nil {
		t.Fatalf("unmarshal defaults: %v", err)
	}

	// Verify all field names are snake_case
	allowedDefaults := map[string]bool{
		"name": true, "profile_key": true, "description": true,
		"runner_type": true, "model": true, "model_preset": true,
		"max_turns": true, "timeout": true, "allowed_tools": true,
		"skip_permission_prompt": true, "sandbox_config": true,
		"created_by": true,
	}
	for key := range defaultsFields {
		if !allowedDefaults[key] {
			t.Errorf("unexpected field %q in AgentProfile JSON — protojson will reject this", key)
		}
	}

	// CRITICAL: Verify timeout is a Duration string, not a nanosecond integer.
	// Before the fix, time.Duration serialized as nanoseconds (e.g. 600000000000),
	// which agent-manager's protojson unmarshaller rejects for google.protobuf.Duration.
	timeoutRaw := string(defaultsFields["timeout"])
	if !strings.HasPrefix(timeoutRaw, `"`) {
		t.Fatalf("timeout must be a JSON string (protojson Duration format), got raw: %s — this was the original serialization bug", timeoutRaw)
	}

	// Verify it matches the expected Duration string pattern (e.g. "600s", "1.5s")
	var timeoutStr string
	if err := json.Unmarshal(defaultsFields["timeout"], &timeoutStr); err != nil {
		t.Fatalf("unmarshal timeout string: %v", err)
	}
	durationPattern := regexp.MustCompile(`^\d+(\.\d+)?s$`)
	if !durationPattern.MatchString(timeoutStr) {
		t.Errorf("timeout %q does not match protojson Duration format (expected e.g. '600s')", timeoutStr)
	}
	if timeoutStr != "600s" {
		t.Errorf("expected timeout '600s' for 10 minutes, got %q", timeoutStr)
	}

	// Verify runner_type and model_preset are valid proto enum name strings
	var runnerType string
	if err := json.Unmarshal(defaultsFields["runner_type"], &runnerType); err != nil {
		t.Fatalf("unmarshal runner_type: %v", err)
	}
	if !strings.HasPrefix(runnerType, "RUNNER_TYPE_") {
		t.Errorf("runner_type %q should use proto enum name format (RUNNER_TYPE_*)", runnerType)
	}

	var modelPreset string
	if err := json.Unmarshal(defaultsFields["model_preset"], &modelPreset); err != nil {
		t.Fatalf("unmarshal model_preset: %v", err)
	}
	if !strings.HasPrefix(modelPreset, "MODEL_PRESET_") {
		t.Errorf("model_preset %q should use proto enum name format (MODEL_PRESET_*)", modelPreset)
	}
}

// TestEnsureProfileRequest_ProtojsonRoundTrip verifies that the JSON produced by
// prompt-manager is structurally compatible with protojson for google.protobuf.Duration
// and proto enum types. Since we can't import agent-manager's proto types directly
// (different Go module), we validate the JSON structure matches protojson expectations.
func TestEnsureProfileRequest_ProtojsonRoundTrip(t *testing.T) {
	req := EnsureProfileRequest{
		ProfileKey: "prompt-manager-heartbeat",
		Defaults: &AgentProfile{
			Name:                 "Test Profile",
			ProfileKey:           "prompt-manager-heartbeat",
			RunnerType:           "RUNNER_TYPE_CODEX",
			ModelPreset:          "MODEL_PRESET_SMART",
			MaxTurns:             50,
			Timeout:              DurationToProtojson(10 * time.Minute),
			AllowedTools:         []string{"read_file"},
			SkipPermissionPrompt: true,
			CreatedBy:            "prompt-manager",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify the raw JSON can be consumed by a strict protojson-like parser:
	// 1. All field names must be snake_case (protojson UseProtoNames)
	// 2. Duration must be a string like "600s"
	// 3. Enums must be proto enum name strings
	// 4. No unknown fields

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var defaults map[string]json.RawMessage
	if err := json.Unmarshal(raw["defaults"], &defaults); err != nil {
		t.Fatalf("unmarshal defaults: %v", err)
	}

	// Duration field must be a quoted string
	if timeout, ok := defaults["timeout"]; ok {
		s := strings.TrimSpace(string(timeout))
		if s[0] != '"' {
			t.Errorf("protojson requires Duration as string, got: %s", s)
		}
	}

	// Enum fields must be quoted strings (not numbers)
	for _, enumField := range []string{"runner_type", "model_preset"} {
		if val, ok := defaults[enumField]; ok {
			s := strings.TrimSpace(string(val))
			if s[0] != '"' {
				t.Errorf("protojson requires enum %s as string name, got: %s", enumField, s)
			}
		}
	}

	// Round-trip back through our own types to verify no data loss
	var decoded EnsureProfileRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if decoded.Defaults.Timeout != "600s" {
		t.Errorf("round-trip: expected timeout '600s', got %q", decoded.Defaults.Timeout)
	}
	if decoded.Defaults.RunnerType != "RUNNER_TYPE_CODEX" {
		t.Errorf("round-trip: expected RUNNER_TYPE_CODEX, got %q", decoded.Defaults.RunnerType)
	}
	if decoded.Defaults.MaxTurns != 50 {
		t.Errorf("round-trip: expected max_turns 50, got %d", decoded.Defaults.MaxTurns)
	}
}

// TestDurationToProtojson verifies the helper produces correct protojson Duration strings.
func TestDurationToProtojson(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, ""},
		{10 * time.Minute, "600s"},
		{1 * time.Hour, "3600s"},
		{30 * time.Second, "30s"},
		{1500 * time.Millisecond, "1.5s"},
		{5 * time.Minute, "300s"},
	}

	for _, tt := range tests {
		got := DurationToProtojson(tt.input)
		if got != tt.expected {
			t.Errorf("DurationToProtojson(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
