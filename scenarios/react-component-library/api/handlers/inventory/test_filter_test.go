package inventory

import "testing"

// TestIsTestFile locks in the filter that keeps test/spec/story files out of
// the production-surface inventory. ui-health's search index treats these as
// noise — they crowd the rankings without representing reusable UI.
func TestIsTestFile(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		want bool
	}{
		{"Badge.tsx", false},
		{"Badge.test.tsx", true},
		{"Badge.spec.tsx", true},
		{"Badge.stories.tsx", true},
		{"Component.test.jsx", true},
		{"Component.spec.jsx", true},
		{"renderWithProviders.tsx", false}, // filtered via test-utils/ dir skip, not name
		{"page.tsx", false},
		{"page.TEST.tsx", true}, // case-insensitive
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := isTestFile(c.name); got != c.want {
				t.Fatalf("isTestFile(%q) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}
