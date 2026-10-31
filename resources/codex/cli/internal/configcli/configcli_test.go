package configcli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEnsureResolvesResourceRoleWithoutPrivateDefaults(t *testing.T) {
	for role, model := range map[string]string{"code.default": "gpt-6.1-terra", "code.research": "gpt-6-luna", "code.fallback": "gpt-6.1-sol"} {
		var stdout, stderr bytes.Buffer
		h := &Handlers{CatalogPath: "../../../model-policy.json", GetEnv: func(string) string { return "" }, Stdout: &stdout, Stderr: &stderr}
		if err := h.Ensure([]string{"--format", "json", "--role", role}); err != nil {
			t.Fatal(err)
		}
		var result map[string]string
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result["model"] != model || result["role"] != role || result["effort"] == "" {
			t.Fatalf("role policy was ignored: %v", result)
		}
	}
}
func TestEnsureRefusesModelOverrideOutsideRole(t *testing.T) {
	for _, model := range []string{"gpt-6.1-sol", "gpt-6-astra", "gpt-5.6-terra"} {
		var stdout, stderr bytes.Buffer
		h := &Handlers{CatalogPath: "../../../model-policy.json", GetEnv: func(key string) string {
			if key == "CODEX_MODEL" {
				return model
			}
			return ""
		}, Stdout: &stdout, Stderr: &stderr}
		if err := h.Ensure([]string{"--format", "json"}); err == nil {
			t.Fatalf("ordinary role accepted override %s", model)
		}
	}
}
