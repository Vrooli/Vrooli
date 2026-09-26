package config

import "testing"

func TestFromEnvDefaultsToModelReadiness(t *testing.T) {
	cfg := FromEnv(func(key string) string {
		if key == "OLLAMA_HOST" {
			return "127.0.0.1:11434"
		}
		return ""
	})
	if cfg.BaseURL != "http://127.0.0.1:11434" || !cfg.RequireModel || cfg.ReadinessTimeout <= 0 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestFromEnvHonorsBaseURL(t *testing.T) {
	cfg := FromEnv(func(key string) string {
		if key == "OLLAMA_BASE_URL" {
			return "http://ollama.example/"
		}
		return ""
	})
	if cfg.BaseURL != "http://ollama.example" {
		t.Fatalf("base URL=%q", cfg.BaseURL)
	}
}
