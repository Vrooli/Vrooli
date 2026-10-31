package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPaidCodingSessionCredentialIsSessionBoundAndRevocable(t *testing.T) {
	registry := newPaidCodingSessionRegistry()
	token, err := registry.Bind("session-1", "account-1")
	if err != nil {
		t.Fatalf("Bind() error: %v", err)
	}
	if len(token) < 40 {
		t.Fatalf("token length = %d, want high-entropy opaque bearer", len(token))
	}
	if binding, ok := registry.Lookup(token); !ok || binding.SessionID != "session-1" || binding.BusinessAccountID != "account-1" {
		t.Fatalf("Lookup() = %#v, %v", binding, ok)
	}
	if _, ok := registry.Lookup(token + "x"); ok {
		t.Fatal("modified bearer token was accepted")
	}
	registry.RemoveSession("session-1")
	if _, ok := registry.Lookup(token); ok {
		t.Fatal("revoked session bearer remained valid")
	}
}

func TestPaidCodingSessionInjectsOpenCodeProviderConfigOnlyIntoPTYEnvironment(t *testing.T) {
	t.Setenv("API_PORT", "15005")
	registry := newPaidCodingSessionRegistry()
	token, err := registry.Bind("session-2", "account-2")
	if err != nil {
		t.Fatalf("Bind() error: %v", err)
	}
	env, err := registry.OpenCodeEnvironment("session-2")
	if err != nil {
		t.Fatalf("OpenCodeEnvironment() error: %v", err)
	}
	if len(env) != 1 || env["OPENCODE_CONFIG_CONTENT"] == "" {
		t.Fatalf("environment = %#v, want only runtime OpenCode config", env)
	}
	var config struct {
		Model    string `json:"model"`
		Provider map[string]struct {
			Options struct {
				BaseURL string `json:"baseURL"`
				APIKey  string `json:"apiKey"`
			} `json:"options"`
			Models map[string]struct {
				Limit struct {
					Context int `json:"context"`
					Output  int `json:"output"`
				} `json:"limit"`
			} `json:"models"`
		} `json:"provider"`
	}
	if err := json.Unmarshal([]byte(env["OPENCODE_CONFIG_CONTENT"]), &config); err != nil {
		t.Fatalf("decode runtime config: %v", err)
	}
	provider, ok := config.Provider["vrooli"]
	if !ok || config.Model != "vrooli/coding" {
		t.Fatalf("runtime model/provider = %q/%v", config.Model, config.Provider)
	}
	if !strings.HasPrefix(provider.Options.BaseURL, "http://127.0.0.1:15005/") || provider.Options.APIKey != token {
		t.Fatalf("provider endpoint/credential = %q/%t", provider.Options.BaseURL, provider.Options.APIKey == token)
	}
	if got := provider.Models["coding"].Limit; got.Context != 128000 || got.Output != 16384 {
		t.Fatalf("OpenCode token limits = %+v, want context=128000 output=16384", got)
	}
	if _, err := paidCodingBaseURL("evil.example"); err == nil {
		t.Fatal("untrusted host accepted as provider base URL")
	}
}
