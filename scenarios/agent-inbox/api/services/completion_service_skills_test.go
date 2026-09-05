package services

import (
	"context"
	"testing"

	"agent-inbox/domain"
)

func TestRejectToolCall_DefaultReason(t *testing.T) {
	repo := newMockCompletionRepository()
	repo.AddToolCallRecord(&domain.ToolCallRecord{
		ID:        "tc-1",
		ChatID:    "chat-1",
		MessageID: "msg-1",
		Status:    domain.StatusPendingApproval,
	})

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	err := svc.RejectToolCall(context.Background(), "chat-1", "tc-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify default error message
	if repo.updateToolCallStatusCalls[0].ErrorMessage != "Rejected by user" {
		t.Errorf("expected default rejection message, got %q", repo.updateToolCallStatusCalls[0].ErrorMessage)
	}
}

// =============================================================================
// Tests for Skill Injection - SetSkills
// =============================================================================

// TestSetSkills_DirectType tests SetSkills with the expected SkillPayload type.
func TestSetSkills_DirectType(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	skills := []SkillPayload{
		{
			ID:      "skill-1",
			Key:     "security",
			Label:   "Security",
			Content: "Security content",
			Tags:    []string{"security", "best-practices"},
		},
	}

	svc.SetSkills(skills)

	if len(svc.skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(svc.skills))
	}
	if svc.skills[0].Key != "security" {
		t.Errorf("expected key 'security', got %q", svc.skills[0].Key)
	}
	if svc.skills[0].Content != "Security content" {
		t.Errorf("expected content 'Security content', got %q", svc.skills[0].Content)
	}
}

// TestSetSkills_InterfaceType tests SetSkills with interface{} (simulating handler passing).
// This is the critical path that happens in production.
func TestSetSkills_InterfaceType(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	// Simulate what the handler does - pass a slice through interface{}
	skills := []map[string]interface{}{
		{
			"id":           "skill-1",
			"key":          "security",
			"label":        "Security",
			"content":      "Security content",
			"tags":         []string{"security"},
			"targetToolId": "specific_tool",
		},
	}

	var skillsInterface interface{} = skills
	svc.SetSkills(skillsInterface)

	if len(svc.skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(svc.skills))
	}
	if svc.skills[0].Key != "security" {
		t.Errorf("expected key 'security', got %q", svc.skills[0].Key)
	}
	if svc.skills[0].TargetToolID != "specific_tool" {
		t.Errorf("expected targetToolId 'specific_tool', got %q", svc.skills[0].TargetToolID)
	}
}

// TestSetSkills_HandlerPayloadType simulates the exact handler struct being passed.
func TestSetSkills_HandlerPayloadType(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	// Create a type that mirrors handlers.SkillPayload
	type HandlerSkillPayload struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Content      string   `json:"content"`
		Key          string   `json:"key"`
		Label        string   `json:"label"`
		Tags         []string `json:"tags,omitempty"`
		TargetToolID string   `json:"targetToolId,omitempty"`
	}

	handlerSkills := []HandlerSkillPayload{
		{
			ID:           "skill-1",
			Name:         "Security Skill",
			Key:          "security",
			Label:        "Security",
			Content:      "Security content",
			Tags:         []string{"security"},
			TargetToolID: "",
		},
	}

	// Pass through interface{} like the handler does
	var skillsInterface interface{} = handlerSkills
	svc.SetSkills(skillsInterface)

	if len(svc.skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(svc.skills))
	}
	if svc.skills[0].Key != "security" {
		t.Errorf("expected key 'security', got %q", svc.skills[0].Key)
	}
}

// TestSetSkills_Nil tests SetSkills with nil input.
func TestSetSkills_Nil(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	// First set some skills
	svc.SetSkills([]SkillPayload{{Key: "test", Content: "test"}})
	if len(svc.skills) != 1 {
		t.Fatal("expected skill to be set")
	}

	// Now set nil - should clear skills
	svc.SetSkills(nil)

	if svc.skills != nil {
		t.Error("expected skills to be nil after SetSkills(nil)")
	}
}

// TestSetSkills_EmptySlice tests SetSkills with empty slice.
func TestSetSkills_EmptySlice(t *testing.T) {
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: newMockCompletionRepository(),
	})

	svc.SetSkills([]SkillPayload{})

	if len(svc.skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(svc.skills))
	}
}
