package versions

import "testing"

func TestCompareVersionLabelsUsesSemanticPrecedence(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "double digit minor", a: "1.10.0", b: "1.9.0", want: 1},
		{name: "release above draft", a: "1.0.0", b: "1.0.0-draft.1", want: 1},
		{name: "draft below release", a: "1.0.0-draft.1", b: "1.0.0", want: -1},
		{name: "numeric prerelease identifiers", a: "1.0.0-rc.10", b: "1.0.0-rc.2", want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := compareVersionLabels(test.a, test.b); got != test.want {
				t.Fatalf("compareVersionLabels(%q, %q) = %d, want %d", test.a, test.b, got, test.want)
			}
		})
	}
}
