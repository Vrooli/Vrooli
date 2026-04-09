package heartbeat

import (
	"context"
	"encoding/json"
	"testing"

	"prompt-manager/store"
)

func TestExecutorExecuteFailsWhenConfigMissing(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	result, err := executor.Execute(ctx, "team-1", "agent-1", "profile-key")

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if result == nil || result.Status != store.HeartbeatStatusFailed {
		t.Fatalf("expected failed status, got %+v", result)
	}
}

// TestCreateTaskRequestMatchesProtoSchema verifies the JSON produced by
// CreateTaskRequest uses field names compatible with the agent-manager proto
// schema. The agent-manager uses protojson with DiscardUnknown=false, so any
// unrecognised field (e.g. "prompt", "workingDir") causes an unmarshal error.
//
// Regression test for: "creating task: agent-manager error: validation error
// on body: invalid JSON request body"
func TestCreateTaskRequestMatchesProtoSchema(t *testing.T) {
	task := &Task{
		Title:       "Heartbeat: team-1/agent-1",
		Description: "Do the thing",
		ScopePath:   "/home/user/project",
		ProjectRoot: "/home/user/project",
	}
	req := CreateTaskRequest{Task: task}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Parse back to a generic map to inspect field names.
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	taskRaw, ok := envelope["task"]
	if !ok {
		t.Fatal("expected top-level 'task' field in request JSON")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(taskRaw, &fields); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}

	// Allowed proto field names (snake_case as protojson expects).
	allowed := map[string]bool{
		"id": true, "title": true, "description": true,
		"scope_path": true, "project_root": true,
		"phase_prompt_ids": true, "context_attachments": true,
		"status": true, "created_by": true,
		"created_at": true, "updated_at": true,
	}

	for key := range fields {
		if !allowed[key] {
			t.Errorf("unexpected field %q in task JSON — agent-manager proto will reject this (DiscardUnknown=false)", key)
		}
	}

	// Required proto fields must be present and non-empty.
	for _, required := range []string{"title", "description", "scope_path"} {
		raw, ok := fields[required]
		if !ok {
			t.Errorf("required proto field %q missing from task JSON", required)
			continue
		}
		var val string
		if err := json.Unmarshal(raw, &val); err != nil {
			t.Errorf("field %q is not a string: %v", required, err)
		} else if val == "" {
			t.Errorf("required proto field %q is empty", required)
		}
	}
}
