package viewer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGetContentReturnsMetadata(t *testing.T) {
	root, relPath := setupViewerFixture(t)
	svc, err := NewService(filepath.Join(root, "scenarios"))
	if err != nil {
		t.Fatalf("failed to create viewer service: %v", err)
	}

	result, err := svc.GetContent(context.Background(), DocContentRequest{Path: relPath, Format: "raw"})
	if err != nil {
		t.Fatalf("GetContent failed: %v", err)
	}
	if result.Path != relPath {
		t.Fatalf("expected path %q, got %q", relPath, result.Path)
	}
	if result.DocType == "" {
		t.Fatalf("expected doc type to be detected")
	}
	if !result.CanReset {
		t.Fatalf("expected reset to be enabled for problems doc")
	}
	if result.Size == 0 {
		t.Fatalf("expected size to be populated")
	}
}

func TestResetDocumentPreview(t *testing.T) {
	root, relPath := setupViewerFixture(t)
	svc, err := NewService(filepath.Join(root, "scenarios"))
	if err != nil {
		t.Fatalf("failed to create viewer service: %v", err)
	}

	result, err := svc.ResetDocument(context.Background(), DocResetRequest{
		Path:           relPath,
		MaxAgeDays:     30,
		KeepMinEntries: 1,
		PreviewOnly:    true,
	})
	if err != nil {
		t.Fatalf("ResetDocument failed: %v", err)
	}
	if result.RemovedCount == 0 {
		t.Fatalf("expected at least one removed entry")
	}
	if result.NewContent == "" {
		t.Fatalf("expected preview content")
	}
	if !result.PreviewOnly {
		t.Fatalf("expected preview flag")
	}
}

func TestResolveDocPathRejectsNonDocs(t *testing.T) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenariosRoot, 0o755); err != nil {
		t.Fatalf("failed to create scenarios root: %v", err)
	}
	outside := filepath.Join(root, "secrets.go")
	if err := os.WriteFile(outside, []byte("package main"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	svc, err := NewService(scenariosRoot)
	if err != nil {
		t.Fatalf("failed to create viewer service: %v", err)
	}

	_, err = svc.GetContent(context.Background(), DocContentRequest{Path: outside, Format: "raw"})
	if err == nil {
		t.Fatalf("expected error for non-doc file")
	}
}

func TestResolveDocPathRejectsOutsideRepo(t *testing.T) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenariosRoot, 0o755); err != nil {
		t.Fatalf("failed to create scenarios root: %v", err)
	}
	outsideRoot := t.TempDir()
	outsideFile := filepath.Join(outsideRoot, "README.md")
	if err := os.WriteFile(outsideFile, []byte("# outside"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	svc, err := NewService(scenariosRoot)
	if err != nil {
		t.Fatalf("failed to create viewer service: %v", err)
	}

	_, err = svc.GetContent(context.Background(), DocContentRequest{Path: outsideFile, Format: "raw"})
	if err == nil {
		t.Fatalf("expected error for outside path")
	}
}

func setupViewerFixture(t *testing.T) (string, string) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	scenario := filepath.Join(scenariosRoot, "alpha")
	docPath := filepath.Join(scenario, "docs", "internal")
	if err := os.MkdirAll(docPath, 0o755); err != nil {
		t.Fatalf("failed to create doc path: %v", err)
	}
	content := `# Problems

## 2000-01-01: Ancient issue
- Legacy note

## 2100-01-01: Recent issue
- Current note
`
	file := filepath.Join(docPath, "PROBLEMS.md")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write problems doc: %v", err)
	}
	return root, filepath.ToSlash(filepath.Join("scenarios", "alpha", "docs", "internal", "PROBLEMS.md"))
}
