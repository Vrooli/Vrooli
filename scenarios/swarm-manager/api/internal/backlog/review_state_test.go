package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadReviewState_EmptyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	state, err := ReadReviewState(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(state))
	}
}

func TestWriteAndReadReviewState_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	// Create archive dir so WriteReviewState can write the file.
	if err := os.MkdirAll(filepath.Join(dir, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}

	state := map[string]ReviewState{
		"OT-P0-001": {
			ReviewedAt:    "2026-03-23T12:00:00Z",
			ReviewComment: "Looks good",
			ReviewStatus:  "approved",
		},
		"OT-P1-002": {
			ReviewedAt:    "2026-03-23T12:05:00Z",
			ReviewComment: "Needs more detail",
			ReviewStatus:  "flagged",
		},
	}

	if err := WriteReviewState(dir, state); err != nil {
		t.Fatalf("write error: %v", err)
	}

	got, err := ReadReviewState(dir)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}

	if got["OT-P0-001"].ReviewStatus != "approved" {
		t.Errorf("expected approved, got %s", got["OT-P0-001"].ReviewStatus)
	}
	if got["OT-P1-002"].ReviewComment != "Needs more detail" {
		t.Errorf("expected 'Needs more detail', got %s", got["OT-P1-002"].ReviewComment)
	}
}

func TestPruneReviewState(t *testing.T) {
	state := map[string]ReviewState{
		"OT-P0-001": {ReviewStatus: "approved"},
		"OT-P1-002": {ReviewStatus: "flagged"},
		"OT-REMOVED": {ReviewStatus: "approved"},
	}

	targetIDs := map[string]bool{
		"OT-P0-001": true,
		"OT-P1-002": true,
	}

	PruneReviewState(state, targetIDs)

	if len(state) != 2 {
		t.Fatalf("expected 2 entries after prune, got %d", len(state))
	}
	if _, ok := state["OT-REMOVED"]; ok {
		t.Error("expected OT-REMOVED to be pruned")
	}
}

func TestPatchModuleReviewState(t *testing.T) {
	// Set up a module fixture.
	itemDir := t.TempDir()
	reqDir := filepath.Join(itemDir, "requirements")
	modDir := filepath.Join(reqDir, "01-core")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write index.json.
	idx := map[string]any{
		"imports":      []string{"01-core/module.json"},
		"requirements": []any{},
	}
	b, _ := json.MarshalIndent(idx, "", "  ")
	os.WriteFile(filepath.Join(reqDir, "index.json"), b, 0o644)

	// Write module.json with requirements.
	mod := map[string]any{
		"module":      "core",
		"description": "Core requirements",
		"requirements": []map[string]any{
			{"id": "REQ-001", "title": "First", "status": "pending"},
			{"id": "REQ-002", "title": "Second", "status": "pending"},
		},
	}
	b, _ = json.MarshalIndent(mod, "", "  ")
	os.WriteFile(filepath.Join(modDir, "module.json"), b, 0o644)

	// Patch review state.
	updates := map[string]RequirementReviewUpdate{
		"REQ-001": {
			ReviewStatus:  "approved",
			ReviewComment: "",
			ReviewedAt:    "2026-03-23T12:00:00Z",
		},
		"REQ-002": {
			ReviewStatus:  "flagged",
			ReviewComment: "Needs work",
			ReviewedAt:    "2026-03-23T12:05:00Z",
		},
	}

	if err := PatchModuleReviewState(itemDir, "core", updates); err != nil {
		t.Fatalf("patch error: %v", err)
	}

	// Read back and verify.
	data, err := os.ReadFile(filepath.Join(modDir, "module.json"))
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}

	reqs := result["requirements"].([]any)
	req1 := reqs[0].(map[string]any)
	req2 := reqs[1].(map[string]any)

	if req1["review_status"] != "approved" {
		t.Errorf("REQ-001: expected review_status=approved, got %v", req1["review_status"])
	}
	if req2["review_status"] != "flagged" {
		t.Errorf("REQ-002: expected review_status=flagged, got %v", req2["review_status"])
	}
	if req2["review_comment"] != "Needs work" {
		t.Errorf("REQ-002: expected review_comment='Needs work', got %v", req2["review_comment"])
	}
}

func TestPatchModuleReviewState_Unreviewed_RemovesFields(t *testing.T) {
	itemDir := t.TempDir()
	reqDir := filepath.Join(itemDir, "requirements")
	modDir := filepath.Join(reqDir, "01-core")
	os.MkdirAll(modDir, 0o755)

	idx := map[string]any{
		"imports":      []string{"01-core/module.json"},
		"requirements": []any{},
	}
	b, _ := json.MarshalIndent(idx, "", "  ")
	os.WriteFile(filepath.Join(reqDir, "index.json"), b, 0o644)

	mod := map[string]any{
		"module": "core",
		"requirements": []map[string]any{
			{
				"id":             "REQ-001",
				"title":          "First",
				"review_status":  "approved",
				"review_comment": "ok",
				"reviewed_at":    "2026-03-23T12:00:00Z",
			},
		},
	}
	b, _ = json.MarshalIndent(mod, "", "  ")
	os.WriteFile(filepath.Join(modDir, "module.json"), b, 0o644)

	updates := map[string]RequirementReviewUpdate{
		"REQ-001": {ReviewStatus: "unreviewed"},
	}

	if err := PatchModuleReviewState(itemDir, "core", updates); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(modDir, "module.json"))
	var result map[string]any
	json.Unmarshal(data, &result)

	req := result["requirements"].([]any)[0].(map[string]any)
	if _, ok := req["review_status"]; ok {
		t.Error("expected review_status to be removed")
	}
	if _, ok := req["review_comment"]; ok {
		t.Error("expected review_comment to be removed")
	}
	if _, ok := req["reviewed_at"]; ok {
		t.Error("expected reviewed_at to be removed")
	}
}

func TestPatchModuleReviewState_RootIndex(t *testing.T) {
	// Requirements stored directly in index.json (moduleID = "index")
	// should be patched in the root file, not searched in imports.
	itemDir := t.TempDir()
	reqDir := filepath.Join(itemDir, "requirements")
	os.MkdirAll(reqDir, 0o755)

	idx := map[string]any{
		"imports": []string{},
		"requirements": []map[string]any{
			{"id": "LD-FUNC-001", "title": "Shared event schema", "status": "pending"},
			{"id": "LD-FUNC-002", "title": "Domain registration", "status": "pending"},
		},
	}
	b, _ := json.MarshalIndent(idx, "", "  ")
	os.WriteFile(filepath.Join(reqDir, "index.json"), b, 0o644)

	updates := map[string]RequirementReviewUpdate{
		"LD-FUNC-001": {
			ReviewStatus: "approved",
			ReviewedAt:   "2026-03-23T12:00:00Z",
		},
	}

	if err := PatchModuleReviewState(itemDir, "index", updates); err != nil {
		t.Fatalf("expected no error for root index module, got: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(reqDir, "index.json"))
	var result map[string]any
	json.Unmarshal(data, &result)

	reqs := result["requirements"].([]any)
	req1 := reqs[0].(map[string]any)
	req2 := reqs[1].(map[string]any)

	if req1["review_status"] != "approved" {
		t.Errorf("LD-FUNC-001: expected review_status=approved, got %v", req1["review_status"])
	}
	if req1["reviewed_at"] != "2026-03-23T12:00:00Z" {
		t.Errorf("LD-FUNC-001: expected reviewed_at set, got %v", req1["reviewed_at"])
	}
	// LD-FUNC-002 should be untouched
	if _, ok := req2["review_status"]; ok {
		t.Error("LD-FUNC-002: expected no review_status (not in updates)")
	}
}
