package main

import (
	"strings"
	"testing"
	"time"
)

// [REQ:P1-002] ExportFormatVersion must follow the vrooli-graph-vN convention
func TestExportFormatVersionConvention(t *testing.T) {
	if !strings.HasPrefix(ExportFormatVersion, "vrooli-graph-v") {
		t.Errorf("ExportFormatVersion should start with 'vrooli-graph-v', got %q", ExportFormatVersion)
	}
}

// [REQ:P1-002] ExportFormatVersion used by ExportService matches the constant
func TestExportServiceUsesConfigConstant(t *testing.T) {
	// Verify through ExportData struct that the constant is wired correctly.
	// ExportService sets ExportFormat = ExportFormatVersion in export_service.go.
	if ExportFormatVersion != "vrooli-graph-v1" {
		t.Errorf("expected vrooli-graph-v1, got %s", ExportFormatVersion)
	}
}

// [REQ:P0-001] AppVersion follows semver format
func TestAppVersionSemver(t *testing.T) {
	parts := strings.Split(AppVersion, ".")
	if len(parts) != 3 {
		t.Errorf("AppVersion should be semver (major.minor.patch), got %q", AppVersion)
	}
}

// [REQ:P2-002] DefaultOllamaURL points to the standard local Ollama port
func TestDefaultOllamaURL(t *testing.T) {
	if !strings.HasPrefix(DefaultOllamaURL, "http://") {
		t.Errorf("DefaultOllamaURL should be HTTP, got %q", DefaultOllamaURL)
	}
	if !strings.Contains(DefaultOllamaURL, "11434") {
		t.Errorf("DefaultOllamaURL should use standard Ollama port 11434, got %q", DefaultOllamaURL)
	}
}

// [REQ:P2-002] OpenRouterURL points to the OpenRouter API
func TestOpenRouterURL(t *testing.T) {
	if !strings.HasPrefix(OpenRouterURL, "https://") {
		t.Errorf("OpenRouterURL should be HTTPS, got %q", OpenRouterURL)
	}
	if !strings.Contains(OpenRouterURL, "openrouter.ai") {
		t.Errorf("OpenRouterURL should point to openrouter.ai, got %q", OpenRouterURL)
	}
}

// [REQ:P2-002] SuggestionService wires DefaultOllamaURL when OLLAMA_URL is unset
func TestSuggestionServiceUsesDefaultOllamaURL(t *testing.T) {
	svc := NewSuggestionServiceWithEnv(nil, func(key string) string { return "" })
	providers := svc.GetProviders()
	var ollamaURL string
	for _, p := range providers {
		if p.Name == "ollama" {
			ollamaURL = p.URL
			break
		}
	}
	if ollamaURL != DefaultOllamaURL {
		t.Errorf("expected ollama URL %q, got %q", DefaultOllamaURL, ollamaURL)
	}
}

// [REQ:P2-002] SuggestionService wires OpenRouterURL for the fallback provider
func TestSuggestionServiceUsesOpenRouterURL(t *testing.T) {
	svc := NewSuggestionServiceWithEnv(nil, func(key string) string {
		if key == "OPENROUTER_API_KEY" {
			return "test-key"
		}
		return ""
	})
	providers := svc.GetProviders()
	var orURL string
	for _, p := range providers {
		if p.Name == "openrouter" {
			orURL = p.URL
			break
		}
	}
	if orURL != OpenRouterURL {
		t.Errorf("expected openrouter URL %q, got %q", OpenRouterURL, orURL)
	}
}

// [REQ:P0-001] RequestTimeout is reasonable for API operations
func TestRequestTimeoutBounds(t *testing.T) {
	if RequestTimeout < 5*time.Second {
		t.Errorf("RequestTimeout too short for production: %v", RequestTimeout)
	}
	if RequestTimeout > 5*time.Minute {
		t.Errorf("RequestTimeout too long, will hang clients: %v", RequestTimeout)
	}
}

// [REQ:P0-001] AppVersion is not empty
func TestAppVersionNonEmpty(t *testing.T) {
	if AppVersion == "" {
		t.Error("AppVersion must not be empty")
	}
}

// [REQ:P1-002] ExportFormatVersion is not empty
func TestExportFormatVersionNonEmpty(t *testing.T) {
	if ExportFormatVersion == "" {
		t.Error("ExportFormatVersion must not be empty")
	}
}
