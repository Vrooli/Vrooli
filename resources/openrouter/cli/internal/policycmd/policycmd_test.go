package policycmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resource-openrouter/cli/internal/policytest"
)

func testHandlers(t *testing.T) (*Handlers, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "model-policy.json")
	if err := os.WriteFile(path, []byte(policytest.FixturePolicyJSON), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	stdout := &bytes.Buffer{}
	h := &Handlers{
		GetEnv: func(k string) string {
			if k == "OPENROUTER_MODEL_POLICY_PATH" {
				return path
			}
			return ""
		},
		Stdout: stdout,
		Stderr: &bytes.Buffer{},
	}
	return h, stdout
}

func TestResolveRoleJSON(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Resolve([]string{"--role", "image.generate.logo", "--json"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	var got resolveReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", stdout.String(), err)
	}
	if got.Role != "image.generate.logo" || got.Model != "vendor/img-vec" || got.Endpoint != "images" {
		t.Fatalf("unexpected report: %+v", got.ResolvedPolicyModel)
	}
	if got.PolicyPath == "" {
		t.Fatal("policy_path empty")
	}
}

func TestResolveField(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Resolve([]string{"--role", "chat.default", "--field", "model"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "vendor/chat-a" {
		t.Fatalf("field model = %q", got)
	}
}

func TestResolveEndpointField(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Resolve([]string{"--role", "image.generate.logo", "--field", "endpoint"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "images" {
		t.Fatalf("field endpoint = %q", got)
	}
}

func TestResolveErrors(t *testing.T) {
	h, _ := testHandlers(t)
	if err := h.Resolve(nil); err == nil {
		t.Fatal("expected error with no selector")
	}
	if err := h.Resolve([]string{"--role", "a", "--model", "b"}); err == nil {
		t.Fatal("expected mutually-exclusive error")
	}
	if err := h.Resolve([]string{"--role", "missing.role"}); err == nil {
		t.Fatal("expected unknown role error")
	}
}

func TestRolesJSON(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Roles([]string{"--json"}); err != nil {
		t.Fatalf("Roles: %v", err)
	}
	var got rolesReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Roles) != 2 {
		t.Fatalf("roles = %d", len(got.Roles))
	}
}

func TestModelsAndConstraints(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Models([]string{"--json"}); err != nil {
		t.Fatalf("Models: %v", err)
	}
	var models modelsReport
	if err := json.Unmarshal(stdout.Bytes(), &models); err != nil {
		t.Fatalf("decode models: %v", err)
	}
	if len(models.Models) != 3 {
		t.Fatalf("models = %d", len(models.Models))
	}

	h2, stdout2 := testHandlers(t)
	if err := h2.Constraints([]string{"--json"}); err != nil {
		t.Fatalf("Constraints: %v", err)
	}
	var cons constraintsReport
	if err := json.Unmarshal(stdout2.Bytes(), &cons); err != nil {
		t.Fatalf("decode constraints: %v", err)
	}
	if len(cons.Constraints.Endpoints) != 2 {
		t.Fatalf("endpoints = %v", cons.Constraints.Endpoints)
	}
}
