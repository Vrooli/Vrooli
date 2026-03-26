package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

// Unit tests for scanner pure functions - no HTTP infrastructure needed.
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-SCAN-PARTIAL]

func TestCSSMarkerRegex(t *testing.T) {
	tests := []struct {
		input   string
		matches int
		element string
	}{
		{"/* brand-manager:primary */", 1, "primary"},
		{"/* brand-manager:secondary */", 1, "secondary"},
		{"--brand-primary: #ff0000; /* brand-manager:primary */", 1, "primary"},
		{"no markers here", 0, ""},
		{"/* brand-manager:colors */ /* brand-manager:text */", 2, "colors"}, // two markers on one line
		{"/* other-tool:primary */", 0, ""},
	}

	for _, tt := range tests {
		matches := cssMarkerRe.FindAllStringSubmatch(tt.input, -1)
		if len(matches) != tt.matches {
			t.Errorf("input=%q: got %d matches, want %d", tt.input, len(matches), tt.matches)
		}
		if tt.matches > 0 && matches[0][1] != tt.element {
			t.Errorf("input=%q: element=%q, want %q", tt.input, matches[0][1], tt.element)
		}
	}
}

func TestJSONBrandKeyRegex(t *testing.T) {
	tests := []struct {
		input   string
		matches int
		key     string
	}{
		{`"_brand_id": "b1"`, 1, "_brand_id"},
		{`"_brand_version": 2`, 1, "_brand_version"},
		{`"name": "Test"`, 0, ""},
		{`"_brand_display_name": "X", "_brand_id": "b1"`, 2, "_brand_display_name"},
	}

	for _, tt := range tests {
		matches := jsonBrandKeyRe.FindAllStringSubmatch(tt.input, -1)
		if len(matches) != tt.matches {
			t.Errorf("input=%q: got %d matches, want %d", tt.input, len(matches), tt.matches)
		}
		if tt.matches > 0 && matches[0][1] != tt.key {
			t.Errorf("input=%q: key=%q, want %q", tt.input, matches[0][1], tt.key)
		}
	}
}

func TestScanFileForCSS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "brand.css")
	os.WriteFile(path, []byte(`/* brand-manager:colors */
:root {
  --brand-primary: #ff0000; /* brand-manager:primary */
  --brand-text: #333; /* brand-manager:text */
}
`), 0o644)

	results := scanFileForCSS(path, "brand.css")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Type != "css" {
		t.Errorf("type = %q, want css", results[0].Type)
	}
	if results[0].Element != "colors" {
		t.Errorf("first element = %q, want colors", results[0].Element)
	}
	if results[0].Line != 1 {
		t.Errorf("first line = %d, want 1", results[0].Line)
	}
}

func TestScanFileForJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	os.WriteFile(path, []byte(`{
  "name": "Test",
  "_brand_id": "b1",
  "_brand_version": 2
}`), 0o644)

	results := scanFileForJSON(path, "manifest.json")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Element != "_brand_id" {
		t.Errorf("first element = %q, want _brand_id", results[0].Element)
	}
}

func TestScanFileWithRegex_MissingFile(t *testing.T) {
	results := scanFileWithRegex("/nonexistent/path", "rel", "css", cssMarkerRe)
	if results != nil {
		t.Errorf("expected nil for missing file, got %d results", len(results))
	}
}

func TestWalkScenarioDir_SkipsDirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0o755)

	os.WriteFile(filepath.Join(dir, "src", "file.css"), []byte("ok"), 0o644)
	os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("skip"), 0o644)
	os.WriteFile(filepath.Join(dir, ".git", "objects", "abc"), []byte("skip"), 0o644)

	var visited []string
	walkScenarioDir(dir, func(path, relPath, ext string) {
		visited = append(visited, relPath)
	})

	if len(visited) != 1 {
		t.Errorf("expected 1 visited file, got %d: %v", len(visited), visited)
	}
	if visited[0] != filepath.Join("src", "file.css") {
		t.Errorf("visited = %q, want src/file.css", visited[0])
	}
}
