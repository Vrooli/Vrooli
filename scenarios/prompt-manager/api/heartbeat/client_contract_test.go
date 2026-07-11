package heartbeat

import (
	"encoding/json"
	"testing"
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

func TestCreateRunRequest_ProfileRefIsKeyOnly(t *testing.T) {
	req := CreateRunRequest{
		TaskID:     "task-1",
		ProfileRef: &ProfileRef{ProfileKey: "prompt-manager/heartbeat"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded CreateRunRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ProfileRef == nil || decoded.ProfileRef.ProfileKey != "prompt-manager/heartbeat" {
		t.Fatalf("profile_ref = %#v, want only the declared profile key", decoded.ProfileRef)
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

func TestEnsureProfileRequest_IsKeyOnly(t *testing.T) {
	data, err := json.Marshal(EnsureProfileRequest{ProfileKey: "prompt-manager/heartbeat"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `{"profile_key":"prompt-manager/heartbeat"}` {
		t.Fatalf("EnsureProfileRequest JSON = %s", data)
	}
}
