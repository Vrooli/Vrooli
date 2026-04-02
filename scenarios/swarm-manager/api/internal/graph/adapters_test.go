package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/initiatives"
)

// --- captureAdapter tests ---

func TestCaptureAdapter_ListCaptures_Empty(t *testing.T) {
	rootDir := t.TempDir()
	adapter := NewCaptureAdapter(rootDir)

	caps, err := adapter.ListCaptures()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 0 {
		t.Errorf("expected 0 captures, got %d", len(caps))
	}
}

func TestCaptureAdapter_ListCaptures_ReadsCaptures(t *testing.T) {
	rootDir := t.TempDir()
	capturesDir := filepath.Join(rootDir, "captures")

	// Create two capture directories with capture.json files.
	for _, cap := range []struct {
		id, text, status string
	}{
		{"cap-1", "first capture", "pending"},
		{"cap-2", "second capture", "classified"},
	} {
		dir := filepath.Join(capturesDir, cap.id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(map[string]string{
			"id":     cap.id,
			"text":   cap.text,
			"status": cap.status,
		})
		if err := os.WriteFile(filepath.Join(dir, "capture.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	adapter := NewCaptureAdapter(rootDir)
	caps, err := adapter.ListCaptures()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(caps) != 2 {
		t.Fatalf("expected 2 captures, got %d", len(caps))
	}

	// Build a map for easier assertions (order is not guaranteed).
	byID := map[string]CaptureEntry{}
	for _, c := range caps {
		byID[c.ID] = c
	}

	c1, ok := byID["cap-1"]
	if !ok {
		t.Fatal("missing cap-1")
	}
	if c1.Text != "first capture" {
		t.Errorf("expected text 'first capture', got %q", c1.Text)
	}
	if c1.Status != "pending" {
		t.Errorf("expected status pending, got %q", c1.Status)
	}

	c2, ok := byID["cap-2"]
	if !ok {
		t.Fatal("missing cap-2")
	}
	if c2.Text != "second capture" {
		t.Errorf("expected text 'second capture', got %q", c2.Text)
	}
}

func TestCaptureAdapter_ListCaptures_WithClassification(t *testing.T) {
	rootDir := t.TempDir()
	capDir := filepath.Join(rootDir, "captures", "cap-cls")
	if err := os.MkdirAll(capDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write capture.json.
	capData, _ := json.Marshal(map[string]string{
		"id":     "cap-cls",
		"text":   "classified capture",
		"status": "classified",
	})
	if err := os.WriteFile(filepath.Join(capDir, "capture.json"), capData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write classification.json.
	clsData, _ := json.Marshal(map[string]any{
		"items": []map[string]string{
			{"kind": "execute", "title": "deploy task"},
			{"kind": "fix", "title": "fix login"},
		},
	})
	if err := os.WriteFile(filepath.Join(capDir, "classification.json"), clsData, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewCaptureAdapter(rootDir)
	caps, err := adapter.ListCaptures()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(caps) != 1 {
		t.Fatalf("expected 1 capture, got %d", len(caps))
	}

	cap := caps[0]
	if len(cap.Items) != 2 {
		t.Fatalf("expected 2 classification items, got %d", len(cap.Items))
	}
	if cap.Items[0].Kind != "execute" || cap.Items[0].Title != "deploy task" {
		t.Errorf("unexpected first item: %+v", cap.Items[0])
	}
	if cap.Items[1].Kind != "fix" || cap.Items[1].Title != "fix login" {
		t.Errorf("unexpected second item: %+v", cap.Items[1])
	}
}

func TestCaptureAdapter_ListCaptures_SkipsInvalidJSON(t *testing.T) {
	rootDir := t.TempDir()
	capturesDir := filepath.Join(rootDir, "captures")

	// Valid capture.
	validDir := filepath.Join(capturesDir, "cap-valid")
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatal(err)
	}
	validData, _ := json.Marshal(map[string]string{"id": "cap-valid", "text": "valid", "status": "pending"})
	if err := os.WriteFile(filepath.Join(validDir, "capture.json"), validData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Invalid capture (malformed JSON).
	invalidDir := filepath.Join(capturesDir, "cap-invalid")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invalidDir, "capture.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewCaptureAdapter(rootDir)
	caps, err := adapter.ListCaptures()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(caps) != 1 {
		t.Fatalf("expected 1 valid capture, got %d", len(caps))
	}
	if caps[0].ID != "cap-valid" {
		t.Errorf("expected cap-valid, got %q", caps[0].ID)
	}
}

func TestCaptureAdapter_ListCaptures_SkipsFiles(t *testing.T) {
	rootDir := t.TempDir()
	capturesDir := filepath.Join(rootDir, "captures")
	if err := os.MkdirAll(capturesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a regular file (not a directory) in captures dir.
	if err := os.WriteFile(filepath.Join(capturesDir, "stray.txt"), []byte("stray"), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewCaptureAdapter(rootDir)
	caps, err := adapter.ListCaptures()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(caps) != 0 {
		t.Errorf("expected 0 captures, got %d", len(caps))
	}
}

// --- initiativeAdapter tests ---

func TestInitiativeAdapter_List_Empty(t *testing.T) {
	store := initiatives.NewStore(t.TempDir())
	adapter := NewInitiativeAdapter(store)

	items, err := adapter.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 initiatives, got %d", len(items))
	}
}

func TestInitiativeAdapter_List_MapsFields(t *testing.T) {
	baseDir := t.TempDir()
	store := initiatives.NewStore(baseDir)

	// Save initiatives via the store directly.
	for _, init := range []struct {
		name, title, status string
		items               []string
	}{
		{"init-a", "Initiative A", "active", []string{"execute/task-1", "fix/bug-1"}},
		{"init-b", "Initiative B", "completed", nil},
	} {
		i := &initiatives.Initiative{
			Name:    init.name,
			Title:   init.title,
			Status:  init.status,
			Items:   init.items,
			Created: "2024-01-01T00:00:00Z",
			Updated: "2024-01-01T00:00:00Z",
		}
		if i.Items == nil {
			i.Items = []string{}
		}
		if err := store.Save(i); err != nil {
			t.Fatalf("Save %q: %v", init.name, err)
		}
	}

	adapter := NewInitiativeAdapter(store)
	entries, err := adapter.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Store.LoadAll returns sorted by name.
	if entries[0].Name != "init-a" {
		t.Errorf("expected first entry init-a, got %q", entries[0].Name)
	}
	if entries[0].Title != "Initiative A" {
		t.Errorf("expected title 'Initiative A', got %q", entries[0].Title)
	}
	if entries[0].Status != "active" {
		t.Errorf("expected status active, got %q", entries[0].Status)
	}
	if len(entries[0].Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(entries[0].Items))
	}
	if entries[0].Items[0] != "execute/task-1" || entries[0].Items[1] != "fix/bug-1" {
		t.Errorf("unexpected items: %v", entries[0].Items)
	}

	if entries[1].Name != "init-b" {
		t.Errorf("expected second entry init-b, got %q", entries[1].Name)
	}
	if entries[1].Status != "completed" {
		t.Errorf("expected status completed, got %q", entries[1].Status)
	}
}

func TestInitiativeAdapter_List_ItemsCopied(t *testing.T) {
	baseDir := t.TempDir()
	store := initiatives.NewStore(baseDir)

	init := &initiatives.Initiative{
		Name:    "copy-test",
		Title:   "Copy Test",
		Status:  "active",
		Items:   []string{"idea/a", "idea/b"},
		Created: "2024-01-01T00:00:00Z",
		Updated: "2024-01-01T00:00:00Z",
	}
	if err := store.Save(init); err != nil {
		t.Fatal(err)
	}

	adapter := NewInitiativeAdapter(store)
	entries, err := adapter.List()
	if err != nil {
		t.Fatal(err)
	}

	// Mutate the returned items slice.
	entries[0].Items[0] = "idea/mutated"

	// Reload and verify no mutation.
	entries2, err := adapter.List()
	if err != nil {
		t.Fatal(err)
	}
	if entries2[0].Items[0] != "idea/a" {
		t.Errorf("mutation leaked: expected idea/a, got %q", entries2[0].Items[0])
	}
}

// --- executionAdapter tests ---

// The executionAdapter is a thin pass-through to execution.Service.List().
// Since execution.Service requires complex dependencies (stores, agent spawners, etc.),
// we verify the adapter satisfies the ExecutionLister interface at compile time.
// Integration behavior is tested via the projection tests in projection_test.go.

var _ ExecutionLister = (*executionAdapter)(nil)
