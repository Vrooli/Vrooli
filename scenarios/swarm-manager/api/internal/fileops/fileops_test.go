package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildFileTree_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	nodes, err := BuildFileTree(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestBuildFileTree_FilesAndDirs(t *testing.T) {
	dir := t.TempDir()

	// Create structure: sub/ (dir), a.txt, b.md
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}

	nodes, err := BuildFileTree(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directories first, then alphabetical.
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[0].Name != "sub" || nodes[0].Type != "directory" {
		t.Errorf("expected first node to be directory 'sub', got %s/%s", nodes[0].Type, nodes[0].Name)
	}
	if len(nodes[0].Children) != 1 {
		t.Errorf("expected 1 child in sub, got %d", len(nodes[0].Children))
	}
	if nodes[1].Name != "a.txt" || nodes[1].Type != "file" {
		t.Errorf("expected second node to be file 'a.txt', got %s/%s", nodes[1].Type, nodes[1].Name)
	}
	if nodes[2].Name != "b.md" || nodes[2].Type != "file" {
		t.Errorf("expected third node to be file 'b.md', got %s/%s", nodes[2].Type, nodes[2].Name)
	}
}

func TestBuildFileTree_NonexistentDir(t *testing.T) {
	_, err := BuildFileTree("/nonexistent/path", "")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestNormalizeRelativePath_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo/bar.txt", "foo/bar.txt"},
		{" notes.md ", "notes.md"},
		{"sub/dir/file.go", "sub/dir/file.go"},
		{"./sub/file.txt", "sub/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := NormalizeRelativePath(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNormalizeRelativePath_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"spaces-only", "   "},
		{"absolute", "/etc/passwd"},
		{"traversal-dotdot", "../secret"},
		{"traversal-deep", "../../etc/passwd"},
		{"dot-only", "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeRelativePath(tt.input)
			if err == nil {
				t.Error("expected error for invalid path")
			}
		})
	}
}

func TestIsProtectedPath(t *testing.T) {
	if !IsProtectedPath("initiative.json", "initiative.json") {
		t.Error("expected protected for exact match")
	}
	if !IsProtectedPath("sub/initiative.json", "initiative.json") {
		t.Error("expected protected for nested path")
	}
	if !IsProtectedPath("INITIATIVE.JSON", "initiative.json") {
		t.Error("expected protected for case-insensitive match")
	}
	if IsProtectedPath("other.json", "initiative.json") {
		t.Error("expected not protected for different file")
	}
	if IsProtectedPath("initiative.json.bak", "initiative.json") {
		t.Error("expected not protected for suffixed file")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "dest.txt")

	content := []byte("hello copy")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst, 0o644); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dest failed: %v", err)
	}
	if string(data) != "hello copy" {
		t.Errorf("expected 'hello copy', got %q", string(data))
	}
}

func TestCopyPath_Directory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	// Create source tree: src/a.txt, src/sub/b.txt
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyPath(src, dst); err != nil {
		t.Fatalf("CopyPath failed: %v", err)
	}

	// Verify destination.
	data, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil {
		t.Fatalf("read a.txt failed: %v", err)
	}
	if string(data) != "a" {
		t.Errorf("expected 'a', got %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	if err != nil {
		t.Fatalf("read sub/b.txt failed: %v", err)
	}
	if string(data) != "b" {
		t.Errorf("expected 'b', got %q", string(data))
	}
}

func TestCopyPath_SingleFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "file.txt")
	dst := filepath.Join(dir, "copy.txt")

	if err := os.WriteFile(src, []byte("single"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyPath(src, dst); err != nil {
		t.Fatalf("CopyPath failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read copy failed: %v", err)
	}
	if string(data) != "single" {
		t.Errorf("expected 'single', got %q", string(data))
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".md", "text/plain"},
		{".json", "application/json"},
		{".go", "text/x-go"},
		{".png", "image/png"},
		{".PDF", "application/pdf"},
		{".unknown", "text/plain"},
		{"", "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := GetContentType(tt.ext)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBuildFileNodeFromPath_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	node, err := BuildFileNodeFromPath(path, "test.txt", info, BuildFileTree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if node.Name != "test.txt" {
		t.Errorf("expected name 'test.txt', got %q", node.Name)
	}
	if node.Type != "file" {
		t.Errorf("expected type 'file', got %q", node.Type)
	}
	if node.Size != 7 {
		t.Errorf("expected size 7, got %d", node.Size)
	}
}

func TestBuildFileNodeFromPath_Directory(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "child.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatal(err)
	}

	node, err := BuildFileNodeFromPath(subDir, "sub", info, BuildFileTree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if node.Name != "sub" {
		t.Errorf("expected name 'sub', got %q", node.Name)
	}
	if node.Type != "directory" {
		t.Errorf("expected type 'directory', got %q", node.Type)
	}
	if len(node.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(node.Children))
	}
}
