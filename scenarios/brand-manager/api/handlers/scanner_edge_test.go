package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"brand-manager/domain"
)

// Unit tests for scanner internals: walkScenarioDir, scanFileWithRegex,
// scanFileForCSS, scanFileForJSON, and invertForDarkMode.
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-UI-THEME]

func TestScanFileForCSS_MultipleMarkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "brand.css")
	os.WriteFile(path, []byte(`:root {
  --brand-primary: #ff0000; /* brand-manager:primary */
  --brand-secondary: #00ff00; /* brand-manager:secondary */
  --brand-text: #333; /* brand-manager:text */
}
`), 0o644)

	results := scanFileForCSS(path, "brand.css")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	elements := map[string]bool{}
	for _, r := range results {
		elements[r.Element] = true
		if r.Type != "css" {
			t.Errorf("type = %q, want css", r.Type)
		}
		if r.File != "brand.css" {
			t.Errorf("file = %q, want brand.css", r.File)
		}
	}
	for _, want := range []string{"primary", "secondary", "text"} {
		if !elements[want] {
			t.Errorf("missing element: %s", want)
		}
	}
}

func TestScanFileForCSS_NoMarkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.css")
	os.WriteFile(path, []byte("body { color: red; }"), 0o644)

	results := scanFileForCSS(path, "plain.css")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestScanFileForCSS_MissingFile(t *testing.T) {
	results := scanFileForCSS("/nonexistent/file.css", "file.css")
	if results != nil {
		t.Errorf("expected nil for missing file, got %v", results)
	}
}

func TestScanFileForJSON_MultipleBrandKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	os.WriteFile(path, []byte(`{
  "_brand_name": "Test",
  "_brand_version": 1,
  "other": "value"
}`), 0o644)

	results := scanFileForJSON(path, "manifest.json")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Type != "json" {
			t.Errorf("type = %q, want json", r.Type)
		}
	}
}

func TestScanFileForJSON_NoBrandKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`{"name": "test", "value": 42}`), 0o644)

	results := scanFileForJSON(path, "data.json")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestScanFileWithRegex_LineNumbers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.css")
	os.WriteFile(path, []byte("line1\n/* brand-manager:primary */\nline3\n/* brand-manager:text */\n"), 0o644)

	results := scanFileWithRegex(path, "test.css", "css", cssMarkerRe)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Line != 2 {
		t.Errorf("first result line = %d, want 2", results[0].Line)
	}
	if results[1].Line != 4 {
		t.Errorf("second result line = %d, want 4", results[1].Line)
	}
}

func TestWalkScenarioDir_SkipsIgnoredDirs(t *testing.T) {
	dir := t.TempDir()

	// Create files in normal and ignored directories
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(dir, "dist"), 0o755)

	os.WriteFile(filepath.Join(dir, "src", "app.css"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(dir, "node_modules", "lib.css"), []byte("b"), 0o644)
	os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("c"), 0o644)
	os.WriteFile(filepath.Join(dir, "dist", "bundle.js"), []byte("d"), 0o644)

	var visited []string
	walkScenarioDir(dir, func(path, relPath, ext string) {
		visited = append(visited, relPath)
	})

	if len(visited) != 1 {
		t.Errorf("expected 1 visited file, got %d: %v", len(visited), visited)
	}
	if len(visited) > 0 && visited[0] != filepath.Join("src", "app.css") {
		t.Errorf("visited[0] = %q, want src/app.css", visited[0])
	}
}

func TestWalkScenarioDir_ReturnsExtension(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "style.CSS"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(dir, "data.json"), []byte("b"), 0o644)

	exts := map[string]bool{}
	walkScenarioDir(dir, func(path, relPath, ext string) {
		exts[ext] = true
	})

	if !exts[".css"] {
		t.Error("expected .css extension (lowercased)")
	}
	if !exts[".json"] {
		t.Error("expected .json extension")
	}
}

func TestInvertForDarkMode(t *testing.T) {
	tests := []struct {
		name, value, want string
	}{
		{"background", "#ffffff", "#1a1a2e"},
		{"surface", "#f5f5f5", "#16213e"},
		{"text", "#333333", "#eaeaea"},
		{"primary", "#3498db", "#3498db"},   // stays same
		{"secondary", "#2ecc71", "#2ecc71"}, // stays same
		{"accent", "#ff6600", "#ff6600"},    // stays same
	}

	for _, tt := range tests {
		got := invertForDarkMode(tt.name, tt.value)
		if got != tt.want {
			t.Errorf("invertForDarkMode(%q, %q) = %q, want %q", tt.name, tt.value, got, tt.want)
		}
	}
}

func TestNewScannerRegistry_DefaultPlugins(t *testing.T) {
	reg := NewScannerRegistry()

	// Should have css and json by default
	cssPlugin := reg.PluginForExt(".css")
	if cssPlugin == nil || cssPlugin.Name() != "css" {
		t.Error("missing css plugin")
	}
	jsonPlugin := reg.PluginForExt(".json")
	if jsonPlugin == nil || jsonPlugin.Name() != "json" {
		t.Error("missing json plugin")
	}

	// SCSS and LESS should also map to CSS plugin
	if p := reg.PluginForExt(".scss"); p == nil || p.Name() != "css" {
		t.Error("expected .scss to map to css plugin")
	}
	if p := reg.PluginForExt(".less"); p == nil || p.Name() != "css" {
		t.Error("expected .less to map to css plugin")
	}
}

func TestScannerRegistry_UnregisteredExtension(t *testing.T) {
	reg := NewScannerRegistry()
	if p := reg.PluginForExt(".xyz"); p != nil {
		t.Error("expected nil for unregistered extension")
	}
}

func TestScannerRegistry_PluginsReturnsAll(t *testing.T) {
	reg := NewScannerRegistry()
	initial := len(reg.Plugins())
	reg.Register(&YAMLPlugin{})
	if len(reg.Plugins()) != initial+1 {
		t.Errorf("expected %d plugins after adding yaml, got %d", initial+1, len(reg.Plugins()))
	}
}

func TestYAMLPlugin_ScanFile_NoMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("name: test\nport: 8080\n"), 0o644)

	plugin := &YAMLPlugin{}
	results := plugin.ScanFile(path, "config.yaml")
	if len(results) != 0 {
		t.Errorf("expected 0 results for non-brand yaml, got %d", len(results))
	}
}

func TestHTMLPlugin_ScanFile_NoMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.html")
	os.WriteFile(path, []byte("<html><body>Hello</body></html>"), 0o644)

	plugin := &HTMLPlugin{}
	results := plugin.ScanFile(path, "index.html")
	if len(results) != 0 {
		t.Errorf("expected 0 results for plain html, got %d", len(results))
	}
}

func TestHTMLPlugin_Extensions(t *testing.T) {
	plugin := &HTMLPlugin{}
	exts := plugin.Extensions()
	extMap := map[string]bool{}
	for _, e := range exts {
		extMap[e] = true
	}
	if !extMap[".html"] || !extMap[".htm"] {
		t.Errorf("expected .html and .htm, got %v", exts)
	}
}

func TestYAMLPlugin_Extensions(t *testing.T) {
	plugin := &YAMLPlugin{}
	exts := plugin.Extensions()
	extMap := map[string]bool{}
	for _, e := range exts {
		extMap[e] = true
	}
	if !extMap[".yaml"] || !extMap[".yml"] {
		t.Errorf("expected .yaml and .yml, got %v", exts)
	}
}

func TestScanReport_EmptyScenario(t *testing.T) {
	report := domain.ScanReport{Scenario: "empty"}
	if report.Total != 0 {
		t.Errorf("expected 0 total for empty report")
	}
	if report.Results != nil {
		t.Error("expected nil results for empty report")
	}
}
