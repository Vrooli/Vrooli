package main

import (
	"testing"
)

// --- SuggestionService tests using the EnvReader seam ---

// [REQ:P1-001] Test SuggestionService provider configuration
func TestSuggestionServiceProviders(t *testing.T) {
	svc := NewSuggestionService(nil)
	providers := svc.GetProviders()

	if len(providers) < 1 {
		t.Fatal("expected at least one provider")
	}
	if providers[0].Name != "ollama" {
		t.Errorf("expected first provider to be ollama, got %s", providers[0].Name)
	}
}

// [REQ:P2-001] Test provider fallback logic
func TestSuggestionServiceFallback(t *testing.T) {
	svc := NewSuggestionService(nil)

	provider, err := svc.GetActiveProvider()
	if err != nil {
		t.Fatalf("expected active provider, got error: %v", err)
	}
	if provider.Name != "ollama" {
		t.Errorf("expected ollama as primary, got %s", provider.Name)
	}
}

// [REQ:P2-003] Test provider list includes both primary and fallback
func TestProviderListStructure(t *testing.T) {
	svc := NewSuggestionService(nil)
	providers := svc.GetProviders()

	hasPrimary := false
	hasFallback := false
	for _, p := range providers {
		if !p.Fallback {
			hasPrimary = true
		}
		if p.Fallback {
			hasFallback = true
		}
	}
	if !hasPrimary {
		t.Error("expected at least one primary provider")
	}
	if !hasFallback {
		t.Error("expected at least one fallback provider")
	}
}

// [REQ:P2-003] Test GetActiveProvider returns primary when available
func TestGetActiveProviderPrimary(t *testing.T) {
	svc := NewSuggestionService(nil)
	provider, err := svc.GetActiveProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Fallback {
		t.Error("expected primary provider, got fallback")
	}
	if provider.Name != "ollama" {
		t.Errorf("expected ollama, got %s", provider.Name)
	}
}

// [REQ:P2-001] Test env seam: custom OLLAMA_URL via EnvReader
func TestSuggestionServiceWithEnv_CustomOllamaURL(t *testing.T) {
	env := func(key string) string {
		if key == "OLLAMA_URL" {
			return "http://custom-ollama:9999"
		}
		return ""
	}

	svc := NewSuggestionServiceWithEnv(nil, env)
	providers := svc.GetProviders()

	found := false
	for _, p := range providers {
		if p.Name == "ollama" {
			found = true
			if p.URL != "http://custom-ollama:9999" {
				t.Errorf("expected custom URL, got %s", p.URL)
			}
		}
	}
	if !found {
		t.Error("expected ollama provider")
	}
}

// [REQ:P2-001] Test env seam: OpenRouter activates when API key is set
func TestSuggestionServiceWithEnv_OpenRouterActive(t *testing.T) {
	env := func(key string) string {
		if key == "OPENROUTER_API_KEY" {
			return "sk-test-key"
		}
		return ""
	}

	svc := NewSuggestionServiceWithEnv(nil, env)
	providers := svc.GetProviders()

	for _, p := range providers {
		if p.Name == "openrouter" {
			if !p.Active {
				t.Error("expected openrouter to be active when API key is set")
			}
			return
		}
	}
	t.Error("openrouter provider not found")
}

// [REQ:P2-001] Test env seam: fallback to OpenRouter when no primary active
func TestSuggestionServiceWithEnv_FallbackToOpenRouter(t *testing.T) {
	env := func(key string) string {
		if key == "OPENROUTER_API_KEY" {
			return "sk-test-key"
		}
		return ""
	}

	svc := NewSuggestionServiceWithEnv(nil, env)
	// Deactivate primary provider to test fallback
	for i := range svc.providers {
		if !svc.providers[i].Fallback {
			svc.providers[i].Active = false
		}
	}

	provider, err := svc.GetActiveProvider()
	if err != nil {
		t.Fatalf("expected fallback provider, got error: %v", err)
	}
	if provider.Name != "openrouter" {
		t.Errorf("expected openrouter fallback, got %s", provider.Name)
	}
	if !provider.Fallback {
		t.Error("expected provider to be marked as fallback")
	}
}

// [REQ:P2-001] Test env seam: no providers available returns error
func TestSuggestionServiceWithEnv_NoProviders(t *testing.T) {
	env := func(key string) string { return "" }

	svc := NewSuggestionServiceWithEnv(nil, env)
	// Deactivate all providers
	for i := range svc.providers {
		svc.providers[i].Active = false
	}

	_, err := svc.GetActiveProvider()
	if err == nil {
		t.Error("expected error when no providers are active")
	}
}

// [REQ:P2-001] Test env seam: default Ollama URL when env var is empty
func TestSuggestionServiceWithEnv_DefaultOllamaURL(t *testing.T) {
	env := func(key string) string { return "" }

	svc := NewSuggestionServiceWithEnv(nil, env)
	for _, p := range svc.providers {
		if p.Name == "ollama" {
			if p.URL != "http://localhost:11434" {
				t.Errorf("expected default URL, got %s", p.URL)
			}
			return
		}
	}
	t.Error("ollama provider not found")
}
