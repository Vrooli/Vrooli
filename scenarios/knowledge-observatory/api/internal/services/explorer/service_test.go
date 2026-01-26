package explorer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGetDocTreeAnnotations(t *testing.T) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	scenarioPath := filepath.Join(scenariosRoot, "alpha")
	if err := os.MkdirAll(filepath.Join(scenarioPath, "docs", "misc"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "README.md"), []byte("# Alpha"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "docs", "PROGRESS.md"), []byte("# Progress"), 0o644); err != nil {
		t.Fatalf("write progress: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "docs", "misc", "NOTE.md"), []byte("note"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	svc, err := NewService(scenariosRoot, nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	tree, err := svc.GetDocTree(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("GetDocTree: %v", err)
	}

	pathIndex := map[string]DocTreeNode{}
	flattenTree(*tree, pathIndex)

	rootPath := filepath.ToSlash(filepath.Join("scenarios", "alpha"))
	if _, ok := pathIndex[rootPath]; !ok {
		t.Fatalf("expected root node path %s", rootPath)
	}
	readmePath := filepath.ToSlash(filepath.Join("scenarios", "alpha", "README.md"))
	readmeNode, ok := pathIndex[readmePath]
	if !ok {
		t.Fatalf("expected README node")
	}
	if readmeNode.DocType != "readme" {
		t.Fatalf("expected readme doc type, got %s", readmeNode.DocType)
	}

	misplacedPath := filepath.ToSlash(filepath.Join("scenarios", "alpha", "docs", "PROGRESS.md"))
	misplacedNode, ok := pathIndex[misplacedPath]
	if !ok {
		t.Fatalf("expected misplaced progress node")
	}
	if misplacedNode.Warning == nil || misplacedNode.Warning.Type != "misplaced" {
		t.Fatalf("expected misplaced warning")
	}

	extraPath := filepath.ToSlash(filepath.Join("scenarios", "alpha", "docs", "misc", "NOTE.md"))
	extraNode, ok := pathIndex[extraPath]
	if !ok {
		t.Fatalf("expected extra note node")
	}
	if extraNode.Warning == nil || extraNode.Warning.Type != "extra" {
		t.Fatalf("expected extra warning")
	}
}

func flattenTree(node DocTreeNode, index map[string]DocTreeNode) {
	index[node.Path] = node
	for _, child := range node.Children {
		flattenTree(child, index)
	}
}
