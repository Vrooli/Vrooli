package main

import "testing"

func TestNotesSemanticOperationIDIsStablePerDurableSourceEvent(t *testing.T) {
	first := notesSemanticOperationID("upsert", 41)
	retry := notesSemanticOperationID("upsert", 41)
	if first == "" || first != retry {
		t.Fatalf("retry key = %q, first key = %q; want stable non-empty key", retry, first)
	}
	if notesSemanticOperationID("upsert", 42) == first {
		t.Fatal("distinct durable source events reused one operation key")
	}
	if notesSemanticOperationID("delete", 41) == first {
		t.Fatal("different mutation classes reused one operation key")
	}
	if notesSemanticOperationID("upsert", 0) != "" || notesSemanticOperationID("unknown", 41) != "" {
		t.Fatal("invalid source event produced an operation key")
	}
}
