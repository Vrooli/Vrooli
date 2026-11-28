package textutil

import "testing"

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{name: "no values"},
		{name: "all blank", values: []string{"", " \t\n"}},
		{name: "first non-empty", values: []string{" first ", "second"}, want: " first "},
		{name: "skip whitespace", values: []string{" \t", "value ", "later"}, want: "value "},
		{name: "unicode whitespace", values: []string{"\u00a0", "\u2003value\u2003"}, want: "\u2003value\u2003"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FirstNonEmpty(tt.values...); got != tt.want {
				t.Fatalf("FirstNonEmpty(%q) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}
