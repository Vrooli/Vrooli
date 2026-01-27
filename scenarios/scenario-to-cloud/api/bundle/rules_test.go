package bundle

import "testing"

func TestIsExcluded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "no patterns - not excluded",
			path:     "src/main.go",
			patterns: nil,
			want:     false,
		},
		{
			name:     "empty patterns - not excluded",
			path:     "src/main.go",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "node_modules anywhere",
			path:     "foo/node_modules/bar/index.js",
			patterns: []string{"**/node_modules/**"},
			want:     true,
		},
		{
			name:     "dist folder anywhere",
			path:     "scenarios/foo/dist/bundle.js",
			patterns: []string{"**/dist/**"},
			want:     true,
		},
		{
			name:     "specific file at any depth",
			path:     "any/path/.DS_Store",
			patterns: []string{"**/.DS_Store"},
			want:     true,
		},
		{
			name:     "coverage folder prefix",
			path:     "coverage/reports/test.xml",
			patterns: []string{"coverage/**"},
			want:     true,
		},
		{
			name:     "no match - different path",
			path:     "src/main.go",
			patterns: []string{"**/node_modules/**", "coverage/**"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsExcluded(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("IsExcluded(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestMatchesAnywhereSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"foo/node_modules/bar", "**/node_modules/**", true},
		{"node_modules/foo", "**/node_modules/**", true},
		{"foo/bar/node_modules", "**/node_modules/**", true},
		{"foo/bar/baz", "**/node_modules/**", false},
		{"foo/nodemodules/bar", "**/node_modules/**", false}, // partial match
		{"dist/file.js", "**/dist/**", true},
		{"src/dist/bundle.js", "**/dist/**", true},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchesAnywhereSegment(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesAnywhereSegment(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchesAnywhereFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"any/path/.DS_Store", "**/.DS_Store", true},
		{".DS_Store", "**/.DS_Store", true},
		{"deep/nested/path/.DS_Store", "**/.DS_Store", true},
		{"file.go", "**/.DS_Store", false},
		{".DS_Store_backup", "**/.DS_Store", false}, // not exact match
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchesAnywhereFile(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesAnywhereFile(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchesPrefixGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"coverage/reports/test.xml", "coverage/**", true},
		{"coverage/lcov.info", "coverage/**", true},
		{"coverage", "coverage/**", true},
		{"coverage2/file", "coverage/**", false}, // different prefix
		{"src/coverage/file", "coverage/**", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchesPrefixGlob(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesPrefixGlob(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchesNestedSegmentGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"scenarios/foo/dist/file.js", "scenarios/**/dist/**", true},
		{"scenarios/foo/bar/dist/bundle.js", "scenarios/**/dist/**", true},
		{"scenarios/dist/file.js", "scenarios/**/dist/**", true},
		{"other/foo/dist/file.js", "scenarios/**/dist/**", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchesNestedSegmentGlob(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesNestedSegmentGlob(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchesSimpleGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"error.log", "*.log", true},
		{"debug.log", "*.log", true},
		{"error.txt", "*.log", false},
		{"main.go", "main.*", true},
		{"main.go", "main.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchesSimpleGlob(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesSimpleGlob(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestContainsPathSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		segment string
		want    bool
	}{
		{"foo/bar/baz", "bar", true},
		{"foo/bar/baz", "foo", true},
		{"foo/bar/baz", "baz", true},
		{"foo/bar/baz", "qux", false},
		{"foo/bar/baz", "ba", false},  // partial segment
		{"foo/bar/baz", "foobar", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.segment, func(t *testing.T) {
			got := containsPathSegment(tt.path, tt.segment)
			if got != tt.want {
				t.Errorf("containsPathSegment(%q, %q) = %v, want %v", tt.path, tt.segment, got, tt.want)
			}
		})
	}
}

func TestAdvancePastSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		segment string
		want    string
	}{
		{"foo/bar/baz", "bar", "baz"},
		{"foo/bar/baz/qux", "bar", "baz/qux"},
		{"foo/bar", "bar", ""},
		{"foo/bar/baz", "notfound", "foo/bar/baz"},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.segment, func(t *testing.T) {
			got := advancePastSegment(tt.path, tt.segment)
			if got != tt.want {
				t.Errorf("advancePastSegment(%q, %q) = %q, want %q", tt.path, tt.segment, got, tt.want)
			}
		})
	}
}
