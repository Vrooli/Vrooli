package transcriptmetrics

import "testing"

func TestNormalizeDefaultPolicy(t *testing.T) {
	got := Normalize("  HéLLo,  世界! $42—ok?\n", DefaultNormalizeOptions())
	if want := "héllo 世界 42ok"; got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestNormalizeOptionsAreIndependent(t *testing.T) {
	input := "A,  B!"
	cases := []struct {
		name string
		opts NormalizeOptions
		want string
	}{
		{"zero value preserves input", NormalizeOptions{}, input},
		{"lowercase only", NormalizeOptions{Lowercase: true}, "a,  b!"},
		{"punctuation only", NormalizeOptions{StripPunctuation: true}, "A  B"},
		{"whitespace only", NormalizeOptions{CollapseWhitespace: true}, "A, B!"},
		{"punctuation and whitespace", NormalizeOptions{StripPunctuation: true, CollapseWhitespace: true}, "A B"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Normalize(input, tc.opts); got != tc.want {
				t.Fatalf("Normalize() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTokenizeUsesSameNormalizationAndHandlesWhitespaceOnly(t *testing.T) {
	got := Tokenize(" The QUICK, brown!\tfox ", DefaultNormalizeOptions())
	want := []string{"the", "quick", "brown", "fox"}
	if len(got) != len(want) {
		t.Fatalf("Tokenize() = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if got := Tokenize(" \t\n", DefaultNormalizeOptions()); len(got) != 0 {
		t.Fatalf("whitespace-only tokens = %#v", got)
	}
}
