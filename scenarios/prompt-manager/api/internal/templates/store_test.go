package templates

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestListAgentFileTemplatesLoadsAndSortsTemplates(t *testing.T) {
	root := t.TempDir()
	writeTemplate(t, root, "second", `{"id":"second","name":"Second","fileName":"B.md"}`, "TEMPLATE.md", "second content")
	writeTemplate(t, root, "first", `{"id":"first","name":"First","fileName":"A.md"}`, "TEMPLATE.md", "first content")

	templates, err := NewStore(root).ListAgentFileTemplates(context.Background())
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0].ID != "first" || templates[1].ID != "second" {
		t.Fatalf("templates not sorted by filename: %+v", templates)
	}
	if templates[0].Content != "first content" {
		t.Fatalf("unexpected template content: %q", templates[0].Content)
	}
}

func writeTemplate(t *testing.T, root, id, meta, entry, content string) {
	t.Helper()
	dir := filepath.Join(root, "templates", "agent-files", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "template.json"), []byte(meta), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, entry), []byte(content), 0o644); err != nil {
		t.Fatalf("write content: %v", err)
	}
}
