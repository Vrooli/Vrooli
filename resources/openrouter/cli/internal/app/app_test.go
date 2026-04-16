package app

import (
	"os"
	"strings"
	"testing"
)

func TestResolvePromptPrefersFlag(t *testing.T) {
	t.Parallel()

	got, err := resolvePrompt("hello", "", []string{"ignored"}, strings.NewReader("stdin"))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestResolvePromptUsesTrailingArgs(t *testing.T) {
	t.Parallel()

	got, err := resolvePrompt("", "", []string{"hello", "world"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "hello world" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestResolvePromptUsesFile(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/prompt.txt"
	if err := os.WriteFile(path, []byte("from file\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := resolvePrompt("", path, nil, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "from file" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestExtractPrimaryText(t *testing.T) {
	t.Parallel()

	got, err := extractPrimaryText([]byte(`{"choices":[{"message":{"content":"hello world"}}]}`))
	if err != nil {
		t.Fatalf("extractPrimaryText() error = %v", err)
	}
	if got != "hello world" {
		t.Fatalf("extractPrimaryText() = %q", got)
	}
}
