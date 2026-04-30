package testing

import "testing"

func TestOllamaClientDisabledWhenBaseURLEmpty(t *testing.T) {
	client := NewOllamaClient("")
	if client.IsEnabled() {
		t.Fatal("expected empty base URL to disable Ollama client")
	}
	if _, _, err := client.Generate("model", "prompt", 10, 0.1); err == nil {
		t.Fatal("expected disabled client to reject generation")
	}
}

func TestOllamaClientEnabledWhenBaseURLPresent(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434")
	if !client.IsEnabled() {
		t.Fatal("expected configured client to be enabled")
	}
}
