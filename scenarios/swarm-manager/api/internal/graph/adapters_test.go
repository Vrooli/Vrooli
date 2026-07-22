package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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

// --- executionAdapter tests ---

// The executionAdapter is a thin pass-through to execution.Service.List().
// Since execution.Service requires complex dependencies (stores, agent spawners, etc.),
// we verify the adapter satisfies the ExecutionLister interface at compile time.
// Integration behavior is tested via the projection tests in projection_test.go.

var _ ExecutionLister = (*executionAdapter)(nil)
