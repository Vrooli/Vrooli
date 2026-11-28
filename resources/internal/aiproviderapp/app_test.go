package aiproviderapp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testConfig(t *testing.T) (Config, func()) {
	t.Helper()
	dir := t.TempDir()
	policyBody := `{
      "schema_version": "1",
      "provider": {"id": "orcarouter", "family": "openai-compatible", "capabilities": ["chat"]},
      "roles": {"chat.default": {"model": "vendor/model", "endpoint": "chat", "required_capabilities": ["chat"], "description": "test", "preference": 1, "request_defaults": {"temperature": 0.4, "max_tokens": 64}, "provenance": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}},
      "models": {"vendor/model": {"provider": "vendor", "family": "test", "capabilities": ["chat"], "modalities": {"input": ["text"], "output": ["text"]}, "endpoints": ["chat"], "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}}},
      "constraints": {"endpoints": ["chat"], "capability_vocabulary": ["chat"], "role_preference_order": ["chat.default"], "provenance_source_kinds": ["manual_policy"], "modality_vocabulary": ["text"]},
      "provenance": {"policy": {"source_kind": "manual_policy", "confidence": "high", "source": "test", "observed_at": "2026-10-30"}}
    }`
	if err := os.WriteFile(filepath.Join(dir, "model-policy.json"), []byte(policyBody), 0o644); err != nil {
		t.Fatal(err)
	}
	oldSource := os.Getenv("ORCAROUTER_CLI_SOURCE_ROOT")
	oldBase := os.Getenv("ORCAROUTER_API_BASE")
	oldKey := os.Getenv("ORCAROUTER_API_KEY")
	os.Setenv("ORCAROUTER_CLI_SOURCE_ROOT", dir)
	os.Setenv("ORCAROUTER_API_KEY", "sk-test")
	cleanup := func() {
		os.Setenv("ORCAROUTER_CLI_SOURCE_ROOT", oldSource)
		os.Setenv("ORCAROUTER_API_BASE", oldBase)
		os.Setenv("ORCAROUTER_API_KEY", oldKey)
	}
	return Config{
		Name:               "orcarouter",
		DisplayName:        "Test",
		APIBaseEnv:         "ORCAROUTER_API_BASE",
		APIBaseDefault:     "https://api.invalid/v1",
		APIKeyEnv:          "ORCAROUTER_API_KEY",
		DefaultRoleEnv:     "ORCAROUTER_DEFAULT_ROLE",
		DefaultRole:        "chat.default",
		CredentialIdentity: "vrooli/orcarouter",
		CredentialField:    "api-key",
		PolicyEnv:          "ORCAROUTER_MODEL_POLICY_PATH",
		PolicySourceEnv:    "ORCAROUTER_CLI_SOURCE_ROOT",
	}, cleanup
}

func TestRunGenerateUsesRoleDefaultsAndOpenAIWireShape(t *testing.T) {
	config, cleanup := testConfig(t)
	defer cleanup()

	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello"}}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`))
	}))
	defer server.Close()
	os.Setenv("ORCAROUTER_API_BASE", server.URL+"/v1")

	var out bytes.Buffer
	if err := runGenerate(config, []string{"--role", "chat.default", "--prompt", "hi", "--json"}, &out, strings.NewReader("")); err != nil {
		t.Fatalf("runGenerate: %v", err)
	}
	if got["model"] != "vendor/model" || got["temperature"].(float64) != 0.4 || got["max_tokens"].(float64) != 64 {
		t.Fatalf("request = %+v", got)
	}
	if !strings.Contains(out.String(), `"hello"`) {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRunGenerateRequiresCredential(t *testing.T) {
	config, cleanup := testConfig(t)
	defer cleanup()
	os.Setenv("ORCAROUTER_API_KEY", "")
	if err := runGenerate(config, []string{"--role", "chat.default", "--prompt", "hi"}, &bytes.Buffer{}, strings.NewReader("")); err == nil {
		t.Fatal("expected unconfigured error")
	}
}

func TestRunEnsureSkipsCatalogCheckWithoutKey(t *testing.T) {
	config, cleanup := testConfig(t)
	defer cleanup()
	os.Setenv("ORCAROUTER_API_KEY", "")
	var out bytes.Buffer
	configB64 := "eyJtb2RlbF9yb2xlcyI6WyJjaGF0LmRlZmF1bHQiXX0=" // {"model_roles":["chat.default"]}
	if err := runEnsure(config, []string{"--config-base64", configB64}, &out); err != nil {
		t.Fatalf("runEnsure: %v", err)
	}
	if !strings.Contains(out.String(), "catalog availability check skipped") {
		t.Fatalf("output = %s", out.String())
	}
}
