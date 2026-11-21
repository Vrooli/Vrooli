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

func TestCompareVersionCoreLabelsPreservesAdoptionOrdering(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "suffix ignored", a: "1.2.0-rc.1", b: "1.2.0", want: 0},
		{name: "double digit minor", a: "1.10.0", b: "1.9.0", want: 1},
		{name: "partial core", a: "1.2", b: "1.1.9", want: 1},
		{name: "wildcard core defaults to zero", a: "1.x.2", b: "1.0.1", want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := CompareVersionCoreLabels(test.a, test.b)
			if !ok || got != test.want {
				t.Fatalf("CompareVersionCoreLabels(%q, %q) = %d, %t, want %d, true", test.a, test.b, got, ok, test.want)
			}
		})
	}
}

func TestCompareVersionCoreLabelsRejectsEmptyAndInvalidLabels(t *testing.T) {
	for _, label := range []string{"", "latest", "1.2.bad"} {
		if _, ok := CompareVersionCoreLabels(label, "1.2.0"); ok {
			t.Errorf("CompareVersionCoreLabels(%q, valid) accepted invalid input", label)
		}
	}
}
