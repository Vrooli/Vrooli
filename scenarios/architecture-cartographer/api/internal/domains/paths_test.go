package domains

import "testing"

func TestPathMatches(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		{"recursive glob match prefix", "api/internal/graph/service.go", "api/internal/graph/**", true},
		{"recursive glob match self", "api/internal/graph", "api/internal/graph/**", true},
		{"recursive glob no match sibling", "api/internal/graphx/x.go", "api/internal/graph/**", false},
		{"trailing slash recursive", "api/internal/graph/sub/x.go", "api/internal/graph/", true},
		{"trailing slash self", "api/internal/graph", "api/internal/graph/", true},
		{"single-level glob match", "api/internal/graph/a.go", "api/internal/graph/*", true},
		{"single-level glob no nested", "api/internal/graph/sub/a.go", "api/internal/graph/*", false},
		{"exact file match", "api/main.go", "api/main.go", true},
		{"exact file mismatch", "api/main_test.go", "api/main.go", false},
		{"double-star matches all", "anything/at/all", "**", true},
		{"empty path", "", "api/**", false},
		{"empty pattern", "api/x", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PathMatches(tc.path, tc.pattern); got != tc.want {
				t.Fatalf("PathMatches(%q, %q) = %v, want %v", tc.path, tc.pattern, got, tc.want)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"`api/internal/graph/`": "api/internal/graph/",
		"  api/main.go  ":       "api/main.go",
		"./api/x":               "api/x",
		"`./api/y`":             "api/y",
	}
	for in, want := range cases {
		if got := NormalizePath(in); got != want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}
