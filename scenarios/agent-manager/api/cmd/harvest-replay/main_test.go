package main

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	got := redact(`{"api_key":"abc", "input_tokens":5, "path":"/home/alice/.codex", "token=xyz"}`)
	for _, forbidden := range []string{"abc", "xyz", "/home/alice"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("redaction retained %q: %s", forbidden, got)
		}
	}
	if !strings.Contains(got, `"input_tokens":5`) {
		t.Fatalf("redaction corrupted telemetry: %s", got)
	}
}
