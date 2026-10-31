package models

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newPolicyHandlers(t *testing.T, policyBody string) (*PolicyHandlers, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "model-policy.json")
	if err := os.WriteFile(path, []byte(policyBody), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	var out bytes.Buffer
	return &PolicyHandlers{
		GetEnv: func(k string) string {
			if k == "PII_DETECTOR_MODEL_POLICY_PATH" {
				return path
			}
			return ""
		},
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	}, &out
}

func TestPolicyRolesJSON(t *testing.T) {
	h, out := newPolicyHandlers(t, validPolicy)
	if err := h.Roles([]string{"--json"}); err != nil {
		t.Fatalf("Roles() error = %v", err)
	}
	var payload struct {
		Roles []struct {
			Role                 string   `json:"role"`
			RequiredCapabilities []string `json:"required_capabilities"`
		} `json:"roles"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("roles output not json: %v\n%s", err, out.String())
	}
	if len(payload.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(payload.Roles))
	}
	// Roles are sorted; pii.default then pii.small.
	found := map[string]bool{}
	for _, r := range payload.Roles {
		found[r.Role] = true
		if len(r.RequiredCapabilities) == 0 {
			t.Errorf("role %q has no capabilities", r.Role)
		}
	}
	if !found["pii.small"] || !found["pii.default"] {
		t.Errorf("missing expected roles: %+v", found)
	}
}

func TestPolicyResolveJSON(t *testing.T) {
	h, out := newPolicyHandlers(t, validPolicy)
	if err := h.Resolve([]string{"--role", "pii.default", "--json"}); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	var payload struct {
		Role         string   `json:"role"`
		Model        string   `json:"model"`
		Canonical    string   `json:"canonical_model"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("resolve output not json: %v\n%s", err, out.String())
	}
	if payload.Role != "pii.default" || payload.Model != "mid" || payload.Canonical != "mid" {
		t.Errorf("unexpected resolve payload: %+v", payload)
	}
	if len(payload.Capabilities) == 0 {
		t.Error("expected capabilities in resolve output")
	}
}

func TestPolicyResolveUnknownRole(t *testing.T) {
	h, _ := newPolicyHandlers(t, validPolicy)
	if err := h.Resolve([]string{"--role", "nope", "--json"}); err == nil {
		t.Error("expected error for unknown role")
	}
}
