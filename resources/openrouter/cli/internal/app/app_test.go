package app

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/policy"
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

func TestResolveImageInputBuildsBoundedProviderNeutralPayload(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("image"))
	selected := policy.ResolvedPolicyModel{Model: "vision-model", Modalities: policy.Modalities{Input: []string{"text", "image"}}}
	prompt, images, err := resolveImageInput(strings.NewReader(`{"prompt":"describe","images":[{"media_type":"image/png","data_b64":"`+data+`"}]}`), selected)
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "describe" || len(images) != 1 || images[0].MediaType != "image/png" || images[0].DataB64 != data {
		t.Fatalf("unexpected decoded input: prompt=%q images=%+v", prompt, images)
	}
}

func TestResolveImageInputRejectsRoleWithoutImageModality(t *testing.T) {
	selected := policy.ResolvedPolicyModel{Model: "text-only", Modalities: policy.Modalities{Input: []string{"text"}}}
	_, _, err := resolveImageInput(strings.NewReader(`{"prompt":"describe","images":[{"media_type":"image/png","data_b64":"aGVsbG8="}]}`), selected)
	if err == nil || !strings.Contains(err.Error(), "image_input.capability_mismatch") {
		t.Fatalf("expected typed capability mismatch, got %v", err)
	}
}
