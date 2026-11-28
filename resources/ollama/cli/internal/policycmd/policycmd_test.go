package policycmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestResolveRoleJSON(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Resolve([]string{"--role", "embedding.default", "--json"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	var got resolveReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if got.Role != "embedding.default" {
		t.Fatalf("role = %q", got.Role)
	}
	if got.Source != "role" {
		t.Fatalf("source = %q", got.Source)
	}
	if got.Model != "nomic-embed-text:latest" {
		t.Fatalf("model = %q", got.Model)
	}
	if got.EmbeddingDimensions != 768 {
		t.Fatalf("embedding_dimensions = %d", got.EmbeddingDimensions)
	}
	if got.ContextWindowTokens != 2048 {
		t.Fatalf("context_window_tokens = %d", got.ContextWindowTokens)
	}
	if got.PolicyPath == "" {
		t.Fatal("policy_path is empty")
	}
	policyBytes, err := os.ReadFile(got.PolicyPath)
	if err != nil {
		t.Fatalf("read resolved policy: %v", err)
	}
	wantDigest := sha256.Sum256(policyBytes)
	if got.PolicySHA256 != hex.EncodeToString(wantDigest[:]) {
		t.Fatalf("policy_sha256 = %q, want exact loaded bytes %x", got.PolicySHA256, wantDigest)
	}
}

func TestResolvePolicySHA256Field(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Resolve([]string{"--role", "embedding.default", "--field", "policy_sha256"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got := strings.TrimSpace(stdout.String())
	if len(got) != 64 {
		t.Fatalf("policy_sha256 field length = %d, want 64", len(got))
	}
	if _, err := hex.DecodeString(got); err != nil {
		t.Fatalf("policy_sha256 field = %q, not hex: %v", got, err)
	}
}

func TestResolveModelJSON(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Resolve([]string{"--model", "nomic-embed-text:latest", "--json"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	var got resolveReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if got.Role != "" {
		t.Fatalf("role = %q, want empty", got.Role)
	}
	if got.Source != "model" {
		t.Fatalf("source = %q", got.Source)
	}
	if got.Model != "nomic-embed-text:latest" {
		t.Fatalf("model = %q", got.Model)
	}
}

func TestResolveField(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Resolve([]string{"--role", "embedding.default", "--field", "embedding_dimensions"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "768" {
		t.Fatalf("stdout = %q, want 768", got)
	}
}

func TestResolveRejectsInvalidSelection(t *testing.T) {
	h, _ := testHandlers(t)
	err := h.Resolve([]string{"--role", "embedding.default", "--model", "nomic-embed-text:latest"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutual exclusion error, got %v", err)
	}
	err = h.Resolve([]string{"--role", "missing.role"})
	if err == nil || !strings.Contains(err.Error(), `unknown model role "missing.role"`) {
		t.Fatalf("expected unknown role error, got %v", err)
	}
	err = h.Resolve([]string{"--model", "missing:latest"})
	if err == nil || !strings.Contains(err.Error(), `unknown model "missing:latest"`) {
		t.Fatalf("expected unknown model error, got %v", err)
	}
}

func TestRolesAndModelsJSON(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Roles([]string{"--json"}); err != nil {
		t.Fatalf("Roles: %v", err)
	}
	var roles rolesReport
	if err := json.Unmarshal(stdout.Bytes(), &roles); err != nil {
		t.Fatalf("decode roles: %v", err)
	}
	if len(roles.Roles) == 0 {
		t.Fatal("roles list is empty")
	}
	policyBytes, err := os.ReadFile(roles.PolicyPath)
	if err != nil {
		t.Fatalf("read roles policy: %v", err)
	}
	wantDigest := sha256.Sum256(policyBytes)
	wantSHA := hex.EncodeToString(wantDigest[:])
	for _, role := range roles.Roles {
		if role.PolicySHA256 != wantSHA {
			t.Fatalf("role %q policy_sha256 = %q, want %q", role.Role, role.PolicySHA256, wantSHA)
		}
	}

	stdout.Reset()
	if err := h.Models([]string{"--json"}); err != nil {
		t.Fatalf("Models: %v", err)
	}
	var models modelsReport
	if err := json.Unmarshal(stdout.Bytes(), &models); err != nil {
		t.Fatalf("decode models: %v", err)
	}
	if len(models.Models) == 0 {
		t.Fatal("models list is empty")
	}
	for _, model := range models.Models {
		if model.PolicySHA256 != wantSHA {
			t.Fatalf("model %q policy_sha256 = %q, want %q", model.Model, model.PolicySHA256, wantSHA)
		}
	}
}

func TestRolesIncludeLocalInferenceRightsDecisions(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Roles([]string{"--require-local-inference", "--json"}); err != nil {
		t.Fatalf("Roles: %v", err)
	}
	var roles rolesReport
	if err := json.Unmarshal(stdout.Bytes(), &roles); err != nil {
		t.Fatal(err)
	}
	for _, role := range roles.Roles {
		if role.RightsDecision == nil || role.RightsDecision.Status == "" || role.RightsDecision.Use != "local_inference" {
			t.Fatalf("role %q has no local-inference rights decision", role.Role)
		}
	}
}

func TestConstraintsJSON(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Constraints([]string{"--json"}); err != nil {
		t.Fatalf("Constraints: %v", err)
	}
	var got constraintsReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode constraints: %v", err)
	}
	if got.Constraints.DefaultParallelRequests == 0 {
		t.Fatal("default_parallel_requests is zero")
	}
}

func TestModelsJSONIncludesCurrentLocalInferenceRights(t *testing.T) {
	h, stdout := testHandlers(t)
	if err := h.Models([]string{"--require-local-inference", "--json"}); err != nil {
		t.Fatalf("Models: %v", err)
	}
	var got modelsReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode models: %v", err)
	}
	if len(got.Models) == 0 {
		t.Fatal("policy model report is empty")
	}
	for _, model := range got.Models {
		if model.RightsDecision == nil || model.RightsDecision.Use != "local_inference" {
			t.Errorf("model %q is missing its local-inference decision: %+v", model.Model, model.RightsDecision)
		}
	}
}

func TestRetargetPlanJSONClassifiesSameDimensionModelChange(t *testing.T) {
	h, stdout := testHandlers(t)
	p, _, err := h.loadPolicy()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	resolved, err := p.ResolveRole("embedding.default")
	if err != nil {
		t.Fatalf("resolve role: %v", err)
	}
	if err := h.RetargetPlan([]string{
		"--role", "embedding.default",
		"--old-model", "old-fixture-embed:latest",
		"--old-dimensions", strconv.Itoa(resolved.EmbeddingDimensions),
		"--store", "qdrant:docs",
		"--json",
	}); err != nil {
		t.Fatalf("RetargetPlan: %v", err)
	}
	var got retargetPlanReport
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if got.Role != "embedding.default" {
		t.Fatalf("role = %q", got.Role)
	}
	if got.Compatibility != "compatible_reembed" {
		t.Fatalf("compatibility = %q", got.Compatibility)
	}
	if got.RequiredAction == "" || got.ApplySafety == "" {
		t.Fatalf("plan must include action and safety: %+v", got)
	}
	if len(got.AffectedStores) != 1 || got.AffectedStores[0] != "qdrant:docs" {
		t.Fatalf("affected_stores = %#v", got.AffectedStores)
	}
}

func TestRetargetPlanRejectsMissingOldDimensions(t *testing.T) {
	h, _ := testHandlers(t)
	err := h.RetargetPlan([]string{"--role", "embedding.default", "--old-model", "old-fixture-embed:latest"})
	if err == nil || !strings.Contains(err.Error(), "--old-dimensions") {
		t.Fatalf("expected old dimensions error, got %v", err)
	}
}

func testHandlers(t *testing.T) (*Handlers, *bytes.Buffer) {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("policy path: %v", err)
	}
	stdout := &bytes.Buffer{}
	return &Handlers{
		GetEnv: func(k string) string {
			if k == "OLLAMA_MODEL_POLICY_PATH" {
				return path
			}
			return ""
		},
		Stdout: stdout,
		Stderr: &bytes.Buffer{},
	}, stdout
}
