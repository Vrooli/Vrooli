package services

import (
	"encoding/json"
	"testing"
)

// TestInjectSkillsIntoArgs_NoSkills tests that args are unchanged when no skills are set.
func TestInjectSkillsIntoArgs_NoSkills(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	originalArgs := `{"task": "do something"}`
	result := svc.injectSkillsIntoArgs("any_tool", originalArgs)

	if result != originalArgs {
		t.Errorf("expected args unchanged, got %q", result)
	}
}

// TestInjectSkillsIntoArgs_AllToolsSkill tests skills with no TargetToolID (applies to all).
func TestInjectSkillsIntoArgs_AllToolsSkill(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	svc.SetSkills([]SkillPayload{
		{
			Key:          "global",
			Label:        "Global Skill",
			Content:      "Applies to all tools",
			TargetToolID: "", // Empty = applies to all
		},
	})

	result := svc.injectSkillsIntoArgs("any_tool", `{"task": "test"}`)

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(result), &args); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	attachments, ok := args["_context_attachments"].([]interface{})
	if !ok {
		t.Fatal("expected _context_attachments to be present")
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
}

// TestInjectSkillsIntoArgs_TargetedSkill tests skills with specific TargetToolID.
func TestInjectSkillsIntoArgs_TargetedSkill(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	svc.SetSkills([]SkillPayload{
		{
			Key:          "targeted",
			Label:        "Targeted Skill",
			Content:      "Only for target_tool",
			TargetToolID: "target_tool",
		},
	})

	// Should be included for matching tool
	result := svc.injectSkillsIntoArgs("target_tool", `{"task": "test"}`)
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(result), &args); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if _, ok := args["_context_attachments"]; !ok {
		t.Error("expected _context_attachments for matching tool")
	}

	// Should NOT be included for non-matching tool
	result2 := svc.injectSkillsIntoArgs("other_tool", `{"task": "test"}`)
	var args2 map[string]interface{}
	if err := json.Unmarshal([]byte(result2), &args2); err != nil {
		t.Fatalf("failed to parse result2: %v", err)
	}
	if _, ok := args2["_context_attachments"]; ok {
		t.Error("expected NO _context_attachments for non-matching tool")
	}
}

// TestInjectSkillsIntoArgs_MixedSkills tests a mix of global and targeted skills.
func TestInjectSkillsIntoArgs_MixedSkills(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	svc.SetSkills([]SkillPayload{
		{
			Key:          "global",
			Label:        "Global",
			Content:      "For all",
			TargetToolID: "",
		},
		{
			Key:          "targeted_a",
			Label:        "For Tool A",
			Content:      "Only for tool_a",
			TargetToolID: "tool_a",
		},
		{
			Key:          "targeted_b",
			Label:        "For Tool B",
			Content:      "Only for tool_b",
			TargetToolID: "tool_b",
		},
	})

	// tool_a should get global + targeted_a
	result := svc.injectSkillsIntoArgs("tool_a", `{}`)
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(result), &args); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	attachments := args["_context_attachments"].([]interface{})
	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments for tool_a, got %d", len(attachments))
	}

	// tool_b should get global + targeted_b
	result2 := svc.injectSkillsIntoArgs("tool_b", `{}`)
	var args2 map[string]interface{}
	json.Unmarshal([]byte(result2), &args2)
	attachments2 := args2["_context_attachments"].([]interface{})
	if len(attachments2) != 2 {
		t.Errorf("expected 2 attachments for tool_b, got %d", len(attachments2))
	}

	// tool_c should get only global
	result3 := svc.injectSkillsIntoArgs("tool_c", `{}`)
	var args3 map[string]interface{}
	json.Unmarshal([]byte(result3), &args3)
	attachments3 := args3["_context_attachments"].([]interface{})
	if len(attachments3) != 1 {
		t.Errorf("expected 1 attachment for tool_c, got %d", len(attachments3))
	}
}
