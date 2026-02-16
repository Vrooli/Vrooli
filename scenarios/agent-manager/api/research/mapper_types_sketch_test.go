package research

import "testing"

func TestClassifyToolAccess(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     string
	}{
		{name: "read", toolName: "Read", want: "read"},
		{name: "search_glob", toolName: "Glob", want: "search"},
		{name: "search_grep", toolName: "Grep", want: "search"},
		{name: "write", toolName: "Write", want: "write"},
		{name: "edit", toolName: "Edit", want: "write"},
		{name: "unknown", toolName: "Other", want: "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyToolAccess(tc.toolName)
			if got != tc.want {
				t.Fatalf("ClassifyToolAccess(%q) = %q, want %q", tc.toolName, got, tc.want)
			}
		})
	}
}

func TestExtractPathsFromToolCall(t *testing.T) {
	tc := ToolCallEventData{
		ToolName: "Read",
		Input: map[string]interface{}{
			"file_path": "README.md",
		},
	}

	paths := ExtractPathsFromToolCall(tc)
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if paths[0] != "README.md" {
		t.Fatalf("expected README.md, got %q", paths[0])
	}
}
