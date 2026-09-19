package services

import (
	"context"
	"encoding/json"
	"testing"

	"agent-inbox/domain"
)

// TestInjectSkillsIntoArgs_InvalidJSON tests behavior with invalid JSON arguments.
func TestInjectSkillsIntoArgs_InvalidJSON(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	svc.SetSkills([]SkillPayload{
		{Key: "test", Content: "test"},
	})

	// Invalid JSON should return original args unchanged
	invalidArgs := `{invalid json`
	result := svc.injectSkillsIntoArgs("any_tool", invalidArgs)

	if result != invalidArgs {
		t.Error("expected invalid JSON args to be returned unchanged")
	}
}

// TestInjectSkillsIntoArgs_PreservesExistingFields tests that existing args are preserved.
func TestInjectSkillsIntoArgs_PreservesExistingFields(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	svc.SetSkills([]SkillPayload{
		{Key: "test", Label: "Test", Content: "test content"},
	})

	originalArgs := `{"task": "build app", "priority": "high", "nested": {"key": "value"}}`
	result := svc.injectSkillsIntoArgs("any_tool", originalArgs)

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(result), &args); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Verify original fields preserved
	if args["task"] != "build app" {
		t.Error("expected 'task' field to be preserved")
	}
	if args["priority"] != "high" {
		t.Error("expected 'priority' field to be preserved")
	}
	nested, ok := args["nested"].(map[string]interface{})
	if !ok || nested["key"] != "value" {
		t.Error("expected nested field to be preserved")
	}

	// Verify attachments added
	if _, ok := args["_context_attachments"]; !ok {
		t.Error("expected _context_attachments to be added")
	}
}

// TestExecuteToolCalls_SkillsInjectionWithTargeting tests the full flow with targeted skills.
func TestExecuteToolCalls_SkillsInjectionWithTargeting(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
	})

	// Set skills: one global, one targeted to tool_a
	svc.SetSkills([]SkillPayload{
		{
			Key:          "global",
			Label:        "Global",
			Content:      "Global content",
			TargetToolID: "",
		},
		{
			Key:          "tool_a_only",
			Label:        "Tool A Only",
			Content:      "Only for tool_a",
			TargetToolID: "tool_a",
		},
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "tool_a", `{"input": "test"}`),
		makeToolCall("tc-2", "tool_b", `{"input": "test"}`),
	}

	_, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := executor.GetExecuteCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}

	// tool_a should have 2 attachments (global + tool_a_only)
	var args1 map[string]interface{}
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args1); err != nil {
		t.Fatalf("failed to parse tool_a args: %v", err)
	}
	attachments1, ok := args1["_context_attachments"].([]interface{})
	if !ok {
		t.Fatal("expected _context_attachments for tool_a")
	}
	if len(attachments1) != 2 {
		t.Errorf("expected 2 attachments for tool_a, got %d", len(attachments1))
	}

	// tool_b should have 1 attachment (global only)
	var args2 map[string]interface{}
	if err := json.Unmarshal([]byte(calls[1].Arguments), &args2); err != nil {
		t.Fatalf("failed to parse tool_b args: %v", err)
	}
	attachments2, ok := args2["_context_attachments"].([]interface{})
	if !ok {
		t.Fatal("expected _context_attachments for tool_b")
	}
	if len(attachments2) != 1 {
		t.Errorf("expected 1 attachment for tool_b, got %d", len(attachments2))
	}
}

// TestSetSkills_JSONRoundTrip_FieldMapping verifies all fields survive the JSON round-trip.
func TestSetSkills_JSONRoundTrip_FieldMapping(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	// Use a map to simulate data coming from JSON decode
	skills := []map[string]interface{}{
		{
			"id":           "skill-123",
			"name":         "Test Skill Name",
			"content":      "Test content here",
			"key":          "test_key",
			"label":        "Test Label",
			"tags":         []interface{}{"tag1", "tag2"},
			"targetToolId": "specific_tool",
		},
	}

	svc.SetSkills(skills)

	if len(svc.skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(svc.skills))
	}

	skill := svc.skills[0]

	// Verify all fields mapped correctly
	if skill.ID != "skill-123" {
		t.Errorf("ID: expected 'skill-123', got %q", skill.ID)
	}
	if skill.Name != "Test Skill Name" {
		t.Errorf("Name: expected 'Test Skill Name', got %q", skill.Name)
	}
	if skill.Content != "Test content here" {
		t.Errorf("Content: expected 'Test content here', got %q", skill.Content)
	}
	if skill.Key != "test_key" {
		t.Errorf("Key: expected 'test_key', got %q", skill.Key)
	}
	if skill.Label != "Test Label" {
		t.Errorf("Label: expected 'Test Label', got %q", skill.Label)
	}
	if skill.TargetToolID != "specific_tool" {
		t.Errorf("TargetToolID: expected 'specific_tool', got %q", skill.TargetToolID)
	}
	if len(skill.Tags) != 2 {
		t.Errorf("Tags: expected 2 tags, got %d", len(skill.Tags))
	}
}

// TestSetSkills_NullFieldsInJSON tests handling of null values in JSON input.
func TestSetSkills_NullFieldsInJSON(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	// Simulate JSON with null values (common from JS frontend)
	jsonData := `[{
		"id": "skill-1",
		"name": null,
		"content": "Content here",
		"key": "test",
		"label": "Test",
		"tags": null,
		"targetToolId": null
	}]`

	var skills interface{}
	if err := json.Unmarshal([]byte(jsonData), &skills); err != nil {
		t.Fatalf("failed to unmarshal test data: %v", err)
	}

	svc.SetSkills(skills)

	if len(svc.skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(svc.skills))
	}

	skill := svc.skills[0]
	if skill.Key != "test" {
		t.Errorf("Key: expected 'test', got %q", skill.Key)
	}
	if skill.Content != "Content here" {
		t.Errorf("Content: expected 'Content here', got %q", skill.Content)
	}
	// Null fields should be empty strings/nil
	if skill.Name != "" {
		t.Errorf("Name: expected empty string for null, got %q", skill.Name)
	}
	if skill.TargetToolID != "" {
		t.Errorf("TargetToolID: expected empty string for null, got %q", skill.TargetToolID)
	}
}
