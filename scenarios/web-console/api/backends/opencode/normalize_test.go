package opencode

import "testing"

func userMsg(id string, created int64, text string) MessageWithParts {
	m := MessageWithParts{Parts: []Part{{Type: "text", Text: text}}}
	m.Info.ID = id
	m.Info.Role = "user"
	m.Info.Time.Created = created
	return m
}

func assistantMsg(id string, created, completed int64, parts ...Part) MessageWithParts {
	m := MessageWithParts{Parts: parts}
	m.Info.ID = id
	m.Info.Role = "assistant"
	m.Info.Time.Created = created
	m.Info.Time.Completed = completed
	return m
}

func TestNormalize_UserAndCompletedAssistant(t *testing.T) {
	msgs := []MessageWithParts{
		userMsg("m1", 100, "hi"),
		assistantMsg("m2", 110, 200,
			Part{Type: "step-start"},
			Part{Type: "text", Text: "hello back"},
			Part{Type: "step-finish"},
		),
	}
	emissions, cur := Normalize(msgs, Cursor{})
	if len(emissions) != 2 {
		t.Fatalf("expected 2 emissions, got %d: %+v", len(emissions), emissions)
	}
	if emissions[0].Role != "user" || emissions[0].Text != "hi" {
		t.Fatalf("user emission wrong: %+v", emissions[0])
	}
	if emissions[1].Role != "assistant" || emissions[1].Text != "hello back" {
		t.Fatalf("assistant emission wrong: %+v", emissions[1])
	}
	if cur.LastUserCreated != 100 || cur.LastAssistantCompleted != 200 {
		t.Fatalf("cursor not advanced: %+v", cur)
	}
}

func TestNormalize_SkipsIncompleteAssistantAndEmptyText(t *testing.T) {
	msgs := []MessageWithParts{
		assistantMsg("m1", 110, 0, Part{Type: "text", Text: "streaming..."}), // not complete
		assistantMsg("m2", 120, 130, Part{Type: "text", Text: "   "}),        // empty after trim
		userMsg("m3", 140, ""), // empty user
	}
	emissions, cur := Normalize(msgs, Cursor{})
	if len(emissions) != 0 {
		t.Fatalf("expected no emissions, got %+v", emissions)
	}
	if cur != (Cursor{}) {
		t.Fatalf("cursor should not advance on skipped messages: %+v", cur)
	}
}

func TestNormalize_IdempotentAcrossReconciliation(t *testing.T) {
	msgs := []MessageWithParts{
		userMsg("m1", 100, "hi"),
		assistantMsg("m2", 110, 200, Part{Type: "text", Text: "hello"}),
	}
	first, cur := Normalize(msgs, Cursor{})
	if len(first) != 2 {
		t.Fatalf("first pass should emit 2, got %d", len(first))
	}
	// Re-running full history with the advanced cursor emits nothing.
	second, cur2 := Normalize(msgs, cur)
	if len(second) != 0 {
		t.Fatalf("second pass should emit 0, got %d: %+v", len(second), second)
	}
	if cur2 != cur {
		t.Fatalf("cursor changed on idempotent re-run: %+v -> %+v", cur, cur2)
	}

	// A new turn appended later emits only the new messages.
	msgs = append(msgs,
		userMsg("m3", 300, "again"),
		assistantMsg("m4", 310, 400, Part{Type: "text", Text: "sure"}),
	)
	third, _ := Normalize(msgs, cur)
	if len(third) != 2 || third[0].Text != "again" || third[1].Text != "sure" {
		t.Fatalf("third pass should emit only the new turn, got %+v", third)
	}
}
