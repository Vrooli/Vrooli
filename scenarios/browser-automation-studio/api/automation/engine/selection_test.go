package engine

import "testing"

func TestResolvePrefersOverride(t *testing.T) {
	cfg := SelectionConfig{DefaultEngine: "browserless", Override: "desktop"}
	if got := cfg.Resolve("request"); got != "desktop" {
		t.Fatalf("expected override to win, got %q", got)
	}
}

func TestResolveUsesRequestedWhenNoOverride(t *testing.T) {
	cfg := SelectionConfig{DefaultEngine: "browserless"}
	if got := cfg.Resolve("desktop"); got != "desktop" {
		t.Fatalf("expected requested engine to be used, got %q", got)
	}
}

func TestResolveFallsBackToDefault(t *testing.T) {
	cfg := SelectionConfig{DefaultEngine: "browserless"}
	if got := cfg.Resolve(""); got != "browserless" {
		t.Fatalf("expected default engine, got %q", got)
	}
}
