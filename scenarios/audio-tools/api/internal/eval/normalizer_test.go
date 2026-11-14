package eval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize_DefaultPolicy(t *testing.T) {
	opts := DefaultNormalizeOptions()
	cases := []struct {
		in   string
		want string
	}{
		{"Hello, World!", "hello world"},
		{"  multiple   spaces\tand\ntabs ", "multiple spaces and tabs"},
		{"UPPER lower MiXeD", "upper lower mixed"},
		{"don't well-known", "dont wellknown"}, // intra-word punctuation folded (documented v1)
		{"100% sure — really?", "100 sure really"},
		{"", ""},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, Normalize(tc.in, opts), "input=%q", tc.in)
	}
}

func TestNormalize_SelectivePasses(t *testing.T) {
	// Only lowercase, keep punctuation + spacing.
	got := Normalize("Hello, World!", NormalizeOptions{Lowercase: true})
	require.Equal(t, "hello, world!", got)
	// Zero options = identity.
	require.Equal(t, "Hello, World!", Normalize("Hello, World!", NormalizeOptions{}))
}

func TestTokenize(t *testing.T) {
	require.Equal(t, []string{"the", "quick", "brown", "fox"},
		Tokenize("The Quick, Brown!  fox", DefaultNormalizeOptions()))
	require.Empty(t, Tokenize("   ", DefaultNormalizeOptions()))
}
