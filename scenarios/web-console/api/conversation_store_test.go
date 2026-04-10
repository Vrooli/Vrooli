package main

import "testing"

func TestAppendAssistantEvent_Basic(t *testing.T) {
	store := NewConversationStore()
	event, result := store.AppendAssistantEvent("sess1", "test", "Hello world")
	if !result.Appended {
		t.Fatalf("expected appended, got code=%s reason=%s", result.Code, result.Reason)
	}
	if event.Role != ConversationRoleAssistant {
		t.Errorf("expected role assistant, got %s", event.Role)
	}
	if event.Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got %q", event.Text)
	}
	if len(event.SpeechParagraphs) == 0 {
		t.Error("expected speechParagraphs to be populated for assistant events")
	}
	if event.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", event.Sequence)
	}
}

func TestAppendAssistantEvent_EmptySession(t *testing.T) {
	store := NewConversationStore()
	_, result := store.AppendAssistantEvent("", "test", "text")
	if result.Appended {
		t.Error("expected failure for empty session ID")
	}
	if result.Code != "conversation_target_missing" {
		t.Errorf("expected code conversation_target_missing, got %s", result.Code)
	}
}

func TestAppendAssistantEvent_EmptyText(t *testing.T) {
	store := NewConversationStore()
	_, result := store.AppendAssistantEvent("sess1", "test", "")
	if result.Appended {
		t.Error("expected failure for empty text")
	}
	if result.Code != "conversation_input_required" {
		t.Errorf("expected code conversation_input_required, got %s", result.Code)
	}
}

func TestAppendAssistantEvent_Dedup(t *testing.T) {
	store := NewConversationStore()
	_, r1 := store.AppendAssistantEvent("sess1", "test", "Hello")
	if !r1.Appended || r1.Duplicate {
		t.Fatal("first append should succeed and not be duplicate")
	}
	_, r2 := store.AppendAssistantEvent("sess1", "test", "Hello")
	if !r2.Appended || !r2.Duplicate {
		t.Fatal("second append of same text should be marked duplicate")
	}
}

func TestAppendUserEvent_Basic(t *testing.T) {
	store := NewConversationStore()
	event, result := store.AppendUserEvent("sess1", "test", "What is 2+2?")
	if !result.Appended {
		t.Fatalf("expected appended, got code=%s reason=%s", result.Code, result.Reason)
	}
	if event.Role != ConversationRoleUser {
		t.Errorf("expected role user, got %s", event.Role)
	}
	if event.Text != "What is 2+2?" {
		t.Errorf("expected text 'What is 2+2?', got %q", event.Text)
	}
	if event.SpeechParagraphs != nil {
		t.Error("expected nil speechParagraphs for user events")
	}
	if event.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", event.Sequence)
	}
}

func TestAppendUserEvent_EmptySession(t *testing.T) {
	store := NewConversationStore()
	_, result := store.AppendUserEvent("", "test", "text")
	if result.Appended {
		t.Error("expected failure for empty session ID")
	}
}

func TestAppendUserEvent_EmptyText(t *testing.T) {
	store := NewConversationStore()
	_, result := store.AppendUserEvent("sess1", "test", "   ")
	if result.Appended {
		t.Error("expected failure for whitespace-only text")
	}
}

func TestAppendUserEvent_Dedup(t *testing.T) {
	store := NewConversationStore()
	_, r1 := store.AppendUserEvent("sess1", "test", "Hello")
	if !r1.Appended || r1.Duplicate {
		t.Fatal("first append should succeed")
	}
	_, r2 := store.AppendUserEvent("sess1", "test", "Hello")
	if !r2.Appended || !r2.Duplicate {
		t.Fatal("second append should be duplicate")
	}
}

func TestAppendUserEvent_SequenceIncrements(t *testing.T) {
	store := NewConversationStore()
	e1, _ := store.AppendAssistantEvent("sess1", "test", "assistant msg")
	e2, _ := store.AppendUserEvent("sess1", "test", "user msg")
	if e2.Sequence != e1.Sequence+1 {
		t.Errorf("expected sequence %d, got %d", e1.Sequence+1, e2.Sequence)
	}
}

func TestUpdateSpeechParagraphs(t *testing.T) {
	store := NewConversationStore()
	event, _ := store.AppendAssistantEvent("sess1", "test", "Hello world")

	newParagraphs := []string{"summarized text"}
	store.UpdateSpeechParagraphs("sess1", event.ID, newParagraphs)

	state := store.ListSession("sess1")
	if len(state.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(state.Events))
	}
	updated := state.Events[0]
	if len(updated.SpeechParagraphs) != 1 || updated.SpeechParagraphs[0] != "summarized text" {
		t.Errorf("expected summarized paragraphs, got %v", updated.SpeechParagraphs)
	}
	if !updated.Summarized {
		t.Error("expected Summarized to be true after UpdateSpeechParagraphs")
	}
	if len(updated.OriginalSpeechParagraphs) == 0 {
		t.Error("expected OriginalSpeechParagraphs to be preserved")
	}
	if updated.OriginalSpeechParagraphs[0] != event.SpeechParagraphs[0] {
		t.Errorf("expected OriginalSpeechParagraphs to match original, got %v", updated.OriginalSpeechParagraphs)
	}
}

func TestUpdateSpeechParagraphs_NonExistent(t *testing.T) {
	store := NewConversationStore()
	store.AppendAssistantEvent("sess1", "test", "Hello")
	// Should be a no-op, no panic
	store.UpdateSpeechParagraphs("sess1", "nonexistent-id", []string{"text"})
	store.UpdateSpeechParagraphs("nonexistent-session", "id", []string{"text"})
}

func TestListSession_Empty(t *testing.T) {
	store := NewConversationStore()
	state := store.ListSession("nonexistent")
	if len(state.Events) != 0 {
		t.Errorf("expected empty events, got %d", len(state.Events))
	}
}

func TestListSession_MixedRoles(t *testing.T) {
	store := NewConversationStore()
	store.AppendUserEvent("sess1", "test", "user question")
	store.AppendAssistantEvent("sess1", "test", "assistant answer")

	state := store.ListSession("sess1")
	if len(state.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(state.Events))
	}
	if state.Events[0].Role != ConversationRoleUser {
		t.Errorf("expected first event user, got %s", state.Events[0].Role)
	}
	if state.Events[1].Role != ConversationRoleAssistant {
		t.Errorf("expected second event assistant, got %s", state.Events[1].Role)
	}
}
