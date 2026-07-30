package secrets

import (
	"testing"
)

func TestKeyUsable(t *testing.T) {
	good := "sk-or-" + makeString(40)
	cases := map[string]bool{
		good:                        true,
		"sk-or-short":               false,
		"sk-proj-" + makeString(40): false,
		"auto-null-123":             false,
		"":                          false,
		"[ERROR] nope":              false,
	}
	for k, want := range cases {
		if got := KeyUsable(k); got != want {
			t.Errorf("KeyUsable(%q)=%v want %v", k, got, want)
		}
	}
}

func TestResolveOpenRouterKey_EnvFirst(t *testing.T) {
	key := "sk-or-" + makeString(40)
	got := ResolveOpenRouterKey(Options{
		Getenv: func(k string) string {
			if k == "OPENROUTER_API_KEY" {
				return key
			}
			return ""
		},
	})
	if got != key {
		t.Errorf("env key not used: %q", got)
	}
}

func TestResolveOpenRouterKeyReturnsEmptyWithoutInjection(t *testing.T) {
	if got := ResolveOpenRouterKey(Options{Getenv: func(string) string { return "" }}); got != "" {
		t.Errorf("unconfigured key = %q", got)
	}
}

func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
