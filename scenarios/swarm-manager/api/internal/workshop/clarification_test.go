package workshop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveClarification(t *testing.T) {
	dir := t.TempDir()
	thread := &ClarificationThread{
		ID:          "thread-1",
		RoundNumber: 1,
		ItemID:      "d1",
		RunID:       "run-abc",
		Messages: []ClarificationMessage{
			{Role: "user", Content: "What does this mean?", CreatedAt: "2026-04-01T00:00:00Z"},
		},
		Status:    "active",
		CreatedAt: "2026-04-01T00:00:00Z",
		UpdatedAt: "2026-04-01T00:00:00Z",
	}

	if err := SaveClarification(dir, thread); err != nil {
		t.Fatalf("SaveClarification: %v", err)
	}

	loaded, err := LoadClarification(dir, 1, "d1")
	if err != nil {
		t.Fatalf("LoadClarification: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected thread, got nil")
	}
	if loaded.ID != "thread-1" {
		t.Errorf("ID = %q, want %q", loaded.ID, "thread-1")
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("Messages len = %d, want 1", len(loaded.Messages))
	}
}

func TestLoadClarification_NotFound(t *testing.T) {
	dir := t.TempDir()
	loaded, err := LoadClarification(dir, 1, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Fatal("expected nil, got thread")
	}
}

func TestLoadClarificationByID(t *testing.T) {
	dir := t.TempDir()
	thread := &ClarificationThread{
		ID:          "find-me",
		RoundNumber: 2,
		ItemID:      "d5",
		Status:      "active",
		CreatedAt:   "2026-04-01T00:00:00Z",
		UpdatedAt:   "2026-04-01T00:00:00Z",
	}
	if err := SaveClarification(dir, thread); err != nil {
		t.Fatalf("save: %v", err)
	}

	found, err := LoadClarificationByID(dir, "find-me")
	if err != nil {
		t.Fatalf("LoadClarificationByID: %v", err)
	}
	if found == nil || found.ID != "find-me" {
		t.Errorf("expected to find thread 'find-me', got %v", found)
	}

	notFound, err := LoadClarificationByID(dir, "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestDeleteClarification(t *testing.T) {
	dir := t.TempDir()
	thread := &ClarificationThread{
		ID: "del-me", RoundNumber: 1, ItemID: "d1",
		Status: "active", CreatedAt: "2026-04-01T00:00:00Z", UpdatedAt: "2026-04-01T00:00:00Z",
	}
	if err := SaveClarification(dir, thread); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := DeleteClarification(dir, 1, "d1"); err != nil {
		t.Fatalf("DeleteClarification: %v", err)
	}

	loaded, _ := LoadClarification(dir, 1, "d1")
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestDeleteClarification_NotFound(t *testing.T) {
	dir := t.TempDir()
	if err := DeleteClarification(dir, 1, "nonexistent"); err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
}

func TestLoadAllClarifications(t *testing.T) {
	dir := t.TempDir()
	threads := []*ClarificationThread{
		{ID: "t1", RoundNumber: 1, ItemID: "d1", Status: "resolved", CreatedAt: "2026-04-01T00:00:00Z", UpdatedAt: "2026-04-01T00:00:00Z"},
		{ID: "t2", RoundNumber: 2, ItemID: "d3", Status: "active", CreatedAt: "2026-04-01T00:00:00Z", UpdatedAt: "2026-04-01T00:00:00Z"},
		{ID: "t3", RoundNumber: 1, ItemID: "d2", Status: "resolved", CreatedAt: "2026-04-01T00:00:00Z", UpdatedAt: "2026-04-01T00:00:00Z"},
	}
	for _, th := range threads {
		if err := SaveClarification(dir, th); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	all, err := LoadAllClarifications(dir)
	if err != nil {
		t.Fatalf("LoadAllClarifications: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len = %d, want 3", len(all))
	}
	// Should be sorted by round then item ID.
	if all[0].ItemID != "d1" || all[1].ItemID != "d2" || all[2].ItemID != "d3" {
		t.Errorf("unexpected sort order: %v, %v, %v", all[0].ItemID, all[1].ItemID, all[2].ItemID)
	}
}

func TestDeleteClarificationsForRound(t *testing.T) {
	dir := t.TempDir()
	threads := []*ClarificationThread{
		{ID: "t1", RoundNumber: 1, ItemID: "d1", Status: "active", CreatedAt: "2026-04-01T00:00:00Z", UpdatedAt: "2026-04-01T00:00:00Z"},
		{ID: "t2", RoundNumber: 1, ItemID: "d2", Status: "active", CreatedAt: "2026-04-01T00:00:00Z", UpdatedAt: "2026-04-01T00:00:00Z"},
		{ID: "t3", RoundNumber: 2, ItemID: "d3", Status: "active", CreatedAt: "2026-04-01T00:00:00Z", UpdatedAt: "2026-04-01T00:00:00Z"},
	}
	for _, th := range threads {
		if err := SaveClarification(dir, th); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	if err := DeleteClarificationsForRound(dir, 1); err != nil {
		t.Fatalf("DeleteClarificationsForRound: %v", err)
	}

	all, _ := LoadAllClarifications(dir)
	if len(all) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(all))
	}
	if all[0].ID != "t3" {
		t.Errorf("expected thread t3, got %s", all[0].ID)
	}
}

func TestRenumberClarifications(t *testing.T) {
	dir := t.TempDir()
	thread := &ClarificationThread{
		ID: "t1", RoundNumber: 3, ItemID: "d1",
		Status: "active", CreatedAt: "2026-04-01T00:00:00Z", UpdatedAt: "2026-04-01T00:00:00Z",
	}
	if err := SaveClarification(dir, thread); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := RenumberClarifications(dir, 3, 2); err != nil {
		t.Fatalf("RenumberClarifications: %v", err)
	}

	// Old file should be gone.
	oldPath := filepath.Join(dir, "workshop", "clarifications", "round-003-item-d1.json")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("expected old file to be removed")
	}

	// New file should exist with updated round number.
	loaded, err := LoadClarification(dir, 2, "d1")
	if err != nil {
		t.Fatalf("LoadClarification: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected renumbered thread, got nil")
	}
	if loaded.RoundNumber != 2 {
		t.Errorf("RoundNumber = %d, want 2", loaded.RoundNumber)
	}
}

func TestParseImpactXML_Valid(t *testing.T) {
	content := `Here's the explanation.

<impact level="decision">
  <reasoning>The question is framed incorrectly.</reasoning>
  <context_note>User wants identity to remain queryable after run completion.</context_note>
  <suggested_update>Revocation means marking identity as inactive, not deleting data.</suggested_update>
</impact>`

	impact := ParseImpactXML(content)
	if impact == nil {
		t.Fatal("expected impact, got nil")
	}
	if impact.Level != "decision" {
		t.Errorf("Level = %q, want %q", impact.Level, "decision")
	}
	if impact.Reasoning != "The question is framed incorrectly." {
		t.Errorf("Reasoning = %q", impact.Reasoning)
	}
	if impact.ContextNote != "User wants identity to remain queryable after run completion." {
		t.Errorf("ContextNote = %q", impact.ContextNote)
	}
	if impact.SuggestedUpdate != "Revocation means marking identity as inactive, not deleting data." {
		t.Errorf("SuggestedUpdate = %q", impact.SuggestedUpdate)
	}
}

func TestParseImpactXML_None(t *testing.T) {
	content := `<impact level="none">
  <reasoning>Just a clarification question.</reasoning>
  <context_note>No changes needed.</context_note>
  <suggested_update></suggested_update>
</impact>`

	impact := ParseImpactXML(content)
	if impact == nil {
		t.Fatal("expected impact, got nil")
	}
	if impact.Level != "none" {
		t.Errorf("Level = %q, want %q", impact.Level, "none")
	}
	if impact.SuggestedUpdate != "" {
		t.Errorf("SuggestedUpdate should be empty, got %q", impact.SuggestedUpdate)
	}
}

func TestParseImpactXML_Missing(t *testing.T) {
	content := "Just a regular response without any XML tags."
	impact := ParseImpactXML(content)
	if impact != nil {
		t.Errorf("expected nil, got %+v", impact)
	}
}

func TestParseImpactXML_Malformed(t *testing.T) {
	content := `<impact level="invalid">broken</impact>`
	impact := ParseImpactXML(content)
	if impact != nil {
		t.Errorf("expected nil for invalid level, got %+v", impact)
	}
}

func TestParseImpactXML_PartialTags(t *testing.T) {
	content := `<impact level="round">
  <reasoning>Big change.</reasoning>
</impact>`

	impact := ParseImpactXML(content)
	if impact == nil {
		t.Fatal("expected impact, got nil")
	}
	if impact.Level != "round" {
		t.Errorf("Level = %q, want %q", impact.Level, "round")
	}
	if impact.Reasoning != "Big change." {
		t.Errorf("Reasoning = %q", impact.Reasoning)
	}
	if impact.ContextNote != "" {
		t.Errorf("ContextNote should be empty, got %q", impact.ContextNote)
	}
}

func TestFormatOptionsForPrompt(t *testing.T) {
	options := []Option{
		{Key: "A", Label: "In-memory map", Rationale: "Simple and fast", Recommended: true},
		{Key: "B", Label: "DB column", Rationale: "Persistent"},
	}
	result := FormatOptionsForPrompt(options)
	if result == "" {
		t.Error("expected non-empty result")
	}
	if !contains(result, "*(recommended)*") {
		t.Error("expected recommended marker")
	}
	if !contains(result, "In-memory map") {
		t.Error("expected option A label")
	}
}

func TestFormatOptionsForPrompt_Empty(t *testing.T) {
	result := FormatOptionsForPrompt(nil)
	if result != "(no options defined)" {
		t.Errorf("expected empty marker, got %q", result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
