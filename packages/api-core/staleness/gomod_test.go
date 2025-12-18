package staleness

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseReplaceDirectives(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "single line replace",
			content: `module test
go 1.21

replace github.com/foo/bar => ../packages/bar
`,
			expected: []string{"../packages/bar"},
		},
		{
			name: "multiple single line replaces",
			content: `module test
go 1.21

replace github.com/foo/bar => ../packages/bar
replace github.com/baz/qux => ../../shared/qux
`,
			expected: []string{"../packages/bar", "../../shared/qux"},
		},
		{
			name: "replace block",
			content: `module test
go 1.21

replace (
	github.com/foo/bar => ../packages/bar
	github.com/baz/qux => ../../shared/qux
)
`,
			expected: []string{"../packages/bar", "../../shared/qux"},
		},
		{
			name: "mixed single and block",
			content: `module test
go 1.21

replace github.com/single => ./local/single

replace (
	github.com/block1 => ../block1
	github.com/block2 => ../block2
)
`,
			expected: []string{"./local/single", "../block1", "../block2"},
		},
		{
			name: "ignore non-relative paths",
			content: `module test
go 1.21

replace github.com/foo/bar => ../packages/bar
replace github.com/old/version v1.0.0 => github.com/new/version v2.0.0
replace github.com/absolute => /absolute/path
`,
			expected: []string{"../packages/bar"},
		},
		{
			name: "no replaces",
			content: `module test
go 1.21

require github.com/foo/bar v1.0.0
`,
			expected: nil,
		},
		{
			name: "empty file",
			content: ``,
			expected: nil,
		},
		{
			name: "real world example",
			content: `module test-genie/cli

go 1.24.0

require (
	github.com/vrooli/cli-core v0.0.0
	golang.org/x/term v0.37.0
	test-genie v0.0.0
)

replace github.com/vrooli/cli-core => ../../../packages/cli-core

replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto

replace test-genie => ../api
`,
			expected: []string{
				"../../../packages/cli-core",
				"../../../packages/proto",
				"../api",
			},
		},
		{
			name: "replace block with whitespace variations",
			content: `module test
go 1.21

replace  (
	github.com/foo/bar => ../packages/bar
)
`,
			expected: []string{"../packages/bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			goModPath := filepath.Join(tmpDir, "go.mod")
			if err := os.WriteFile(goModPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write go.mod: %v", err)
			}

			got, err := ParseReplaceDirectives(goModPath)
			if err != nil {
				t.Fatalf("ParseReplaceDirectives() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseReplaceDirectives() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseReplaceDirectives_FileNotFound(t *testing.T) {
	_, err := ParseReplaceDirectives("/nonexistent/go.mod")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
