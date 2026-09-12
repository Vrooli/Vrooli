package handlers

import "testing"

// Path helper tests for the legacy `handlers` package. The proto-first
// project_files handler maintains its own copy of these helpers and tests
// them independently; the duplicates here cover the helpers as used by
// peer REST handlers (currently projects.go) and the RESTException
// ServeProjectFile streamer.

func TestNormalizeProjectRelPath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPath  string
		wantValid bool
	}{
		{"empty string", "", "", false},
		{"dot only", ".", "", false},
		{"valid path", "workflows/test.json", "workflows/test.json", true},
		{"leading slash", "/workflows/test.json", "workflows/test.json", true},
		{"trailing slash", "workflows/test/", "workflows/test", true},
		{"backslashes", "workflows\\test.json", "workflows/test.json", true},
		{"parent directory", "../outside", "", false},
		{"double parent", "../../outside", "", false},
		{"whitespace", "  workflows/test.json  ", "workflows/test.json", true},
		{"nested path", "workflows/folder/subfolder/test.json", "workflows/folder/subfolder/test.json", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeProjectRelPath(tt.input)
			if ok != tt.wantValid {
				t.Errorf("normalizeProjectRelPath(%q) valid = %v, want %v", tt.input, ok, tt.wantValid)
			}
			if ok && got != tt.wantPath {
				t.Errorf("normalizeProjectRelPath(%q) = %q, want %q", tt.input, got, tt.wantPath)
			}
		})
	}
}

func TestSafeJoinProjectPath(t *testing.T) {
	root := "/tmp/project"
	if abs, err := safeJoinProjectPath(root, "workflows/a.json"); err != nil || abs != "/tmp/project/workflows/a.json" {
		t.Fatalf("happy path failed: %q err=%v", abs, err)
	}
	if _, err := safeJoinProjectPath("", "x.json"); err == nil {
		t.Fatalf("empty root must error")
	}
	if _, err := safeJoinProjectPath(root, "../escape"); err == nil {
		t.Fatalf("traversal must error")
	}
}
