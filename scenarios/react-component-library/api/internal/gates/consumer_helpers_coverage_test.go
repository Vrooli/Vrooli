package gates

import "testing"

func TestSemverPartsOrdersThreeNumericComponents(t *testing.T) {
	cases := []struct {
		left, right string
	}{
		{"1.9.0", "1.10.0"},
		{"1.10.0", "2.0.0"},
		{"2.0", "2.0.1"},
	}
	for _, tc := range cases {
		t.Run(tc.left+"<"+tc.right, func(t *testing.T) {
			if left, right := semverParts(tc.left), semverParts(tc.right); left >= right {
				t.Fatalf("semverParts(%q) = %d, semverParts(%q) = %d; want left < right", tc.left, left, tc.right, right)
			}
		})
	}
}
