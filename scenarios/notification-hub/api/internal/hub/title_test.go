package hub

import "testing"

func TestPushTitleIsTheProducerHeadline(t *testing.T) {
	cases := []struct {
		name, display, title, want string
	}{
		{"no source prefix", "Web Console", "Agent turn finished", "Agent turn finished"},
		{"no display is passthrough", "", "Agent needs your input", "Agent needs your input"},
		{"empty title falls back to name", "Web Console", "", "Web Console"},
		{"trims whitespace", "  Web Console  ", "  Agent turn finished  ", "Agent turn finished"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pushTitle(tc.display, tc.title); got != tc.want {
				t.Fatalf("pushTitle(%q, %q) = %q, want %q", tc.display, tc.title, got, tc.want)
			}
		})
	}
}
