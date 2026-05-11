package testing

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestOllamaClientDisabledWhenNotEnabled(t *testing.T) {
	client := NewOllamaClient(false, "")
	if client.IsEnabled() {
		t.Fatal("expected disabled configuration to disable Ollama client")
	}
	if _, _, err := client.Generate("model", "prompt", 10, 0.1); err == nil {
		t.Fatal("expected disabled client to reject generation")
	}
}

func TestOllamaClientEnabledWhenConfigured(t *testing.T) {
	client := NewOllamaClient(true, "")
	if !client.IsEnabled() {
		t.Fatal("expected configured client to be enabled")
	}
}

func TestOllamaClientGenerateUsesGateway(t *testing.T) {
	var gotArgs []string
	var gotStdin string
	client := newOllamaClientWithRunner(true, "resource-ollama-test", func(_ context.Context, args []string, stdin []byte) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		gotStdin = string(stdin)
		return []byte(`{"response":"ok","eval_count":12}`), nil
	})

	resp, _, err := client.Generate("llama3.2:1b", "hello", 123, 0.25)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	wantArgs := []string{
		"resource-ollama-test",
		"gateway",
		"generate",
		"--model", "llama3.2:1b",
		"--json",
		"--prompt-stdin",
		"--max-tokens", "123",
		"--temperature", "0.25",
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	if gotStdin != "hello" {
		t.Fatalf("stdin = %q, want prompt", gotStdin)
	}
	if resp.Response != "ok" || resp.EvalCount != 12 {
		t.Fatalf("response = %+v", resp)
	}
}

func TestOllamaClientGenerateSurfacesGatewayError(t *testing.T) {
	client := newOllamaClientWithRunner(true, "", func(context.Context, []string, []byte) ([]byte, error) {
		return nil, errors.New("gateway failed")
	})

	if _, _, err := client.Generate("model", "prompt", 10, 0.1); err == nil {
		t.Fatal("expected gateway error")
	}
}
