package main

import "testing"

func TestParseAllowedOrigins(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty defaults to star", raw: "", want: []string{"*"}},
		{name: "whitespace defaults to star", raw: "   ", want: []string{"*"}},
		{name: "single origin", raw: "http://example.com", want: []string{"http://example.com"}},
		{name: "comma separated with spaces", raw: "http://a.com, http://b.com ,http://c.com", want: []string{"http://a.com", "http://b.com", "http://c.com"}},
		{name: "trailing commas ignored", raw: "http://a.com, ,", want: []string{"http://a.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAllowedOrigins(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d origins, got %d (%v)", len(tt.want), len(got), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("origin[%d] expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}
