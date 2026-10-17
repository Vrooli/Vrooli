package configcli

import (
	"bytes"
	"testing"
)

func TestEnsureDefaultsToCurrentDeliveryModel(t *testing.T) {
	var stdout, stderr bytes.Buffer
	h := &Handlers{
		GetEnv: func(string) string { return "" },
		Stdout: &stdout,
		Stderr: &stderr,
	}

	if err := h.Ensure([]string{"--format", "json"}); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if got, want := stdout.String(), `{"active_source":"","role":"code.default","model":"gpt-6-luna","effort":"medium"}`; got != want {
		t.Fatalf("Ensure() output = %s, want %s", got, want)
	}
}

func TestEnsureHonorsExplicitModel(t *testing.T) {
	var stdout, stderr bytes.Buffer
	h := &Handlers{
		GetEnv: func(key string) string {
			if key == "CODEX_MODEL" {
				return "gpt-6-sol"
			}
			return ""
		},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	if err := h.Ensure([]string{"--format", "json", "--role", "judgment.supervision"}); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if got, want := stdout.String(), `{"active_source":"","role":"judgment.supervision","model":"gpt-6-sol","effort":"medium"}`; got != want {
		t.Fatalf("Ensure() output = %s, want %s", got, want)
	}
}
