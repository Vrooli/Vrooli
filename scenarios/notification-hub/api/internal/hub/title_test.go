package hub

import "testing"

func TestTitleWithSource(t *testing.T) {
	cases := []struct {
		name, display, title, want string
	}{
		{"prefixes source", "Web Console", "Session needs your input", "Web Console — Session needs your input"},
		{"no display is passthrough", "", "Session needs your input", "Session needs your input"},
		{"already named is not doubled", "Web Console", "Web Console — delivery test", "Web Console — delivery test"},
		{"case-insensitive dedupe", "Web Console", "web console update", "web console update"},
		{"empty title falls back to name", "Web Console", "", "Web Console"},
		{"trims whitespace", "  Web Console  ", "  Session finished its turn  ", "Web Console — Session finished its turn"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := titleWithSource(tc.display, tc.title); got != tc.want {
				t.Fatalf("titleWithSource(%q, %q) = %q, want %q", tc.display, tc.title, got, tc.want)
			}
		})
	}
}
