package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestPackagePlanUsesSharedCurrentRightsContextAndRestoresEligibility(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	policyBytes, err := os.ReadFile(filepath.Join(repoRoot, "resources", "ollama", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Models map[string]struct {
			Rights sharedpolicy.ArtifactRights `json:"rights"`
		} `json:"models"`
	}
	if err := json.Unmarshal(policyBytes, &registry); err != nil {
		t.Fatal(err)
	}
	model, ok := registry.Models["llama3.2:1b"]
	if !ok || model.Rights.ArtifactID != "registry.ollama.ai/library/llama3.2:1b" {
		t.Fatalf("source model rights facts missing or changed: %+v", model.Rights)
	}
	rightsFacts, err := json.Marshal(model.Rights)
	if err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "rights-demo")
	if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := `{"dependencies":{"resources":{"ollama":{"enabled":true,"required":true}}}}`
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(service), 0o644); err != nil {
		t.Fatal(err)
	}
	resourceDir := filepath.Join(root, "resources", "ollama")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resourceManifest := map[string]any{
		"bundling":        "bundled",
		"cli":             map[string]any{"distribution": map[string]any{"kind": "prebuilt_artifact", "artifact_name": "resource-ollama_${os}_${arch}"}},
		"deployment":      map[string]any{"profiles": map[string]any{"desktop": map[string]any{"linux": map[string]any{"support": "supported", "mode": "bundled-client", "architectures": []string{"amd64"}}}}},
		"model_artifacts": []BundledModelArtifact{{ModelID: "llama3.2:1b", Kind: "weights", Rights: rightsFacts}},
	}
	manifestBytes, err := json.Marshal(resourceManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(root, "release")
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	artifactNames, err := resourcedeployment.ArtifactFiles("resource-ollama_linux_amd64")
	if err != nil {
		t.Fatal(err)
	}
	releaseArtifacts := make([]resourcedeployment.ReleaseArtifact, 0, len(artifactNames))
	for _, name := range artifactNames {
		body := []byte("fixture:" + name)
		if err := os.WriteFile(filepath.Join(artifactRoot, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(body)
		releaseArtifacts = append(releaseArtifacts, resourcedeployment.ReleaseArtifact{
			Name: name, SHA256: hex.EncodeToString(digest[:]), Role: "resource", UpstreamProvenance: "desktop rights package-plan test fixture",
		})
	}
	releaseManifest, err := json.Marshal(resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: releaseArtifacts})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "release-manifest.json"), releaseManifest, 0o644); err != nil {
		t.Fatal(err)
	}

	currentContext := sharedpolicy.RightsContext{
		ID: "fixture-current-context", Version: "below-threshold", EntityScope: "operator-group", RevenueStatus: "pre_revenue",
		MonthlyActiveUsers:         map[string]string{"2024-08": "0"},
		MonthlyActiveUsersEvidence: map[string]string{"2024-08": "source Ollama model policy fact; test context value"},
		AsOf:                       "2026-11-14", Source: "test_fixture",
	}
	contextPath := filepath.Join(root, "model-rights-context.json")
	t.Setenv(sharedpolicy.RightsContextPathEnv, contextPath)
	writeContext := func() {
		t.Helper()
		data, err := json.Marshal(map[string]any{"schema_version": "1", "context": currentContext})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(contextPath, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeContext()
	evaluator := NewSharedModelRightsEvaluator()
	resolve := func() *ResourceDeploymentPlan {
		t.Helper()
		plan, err := resolveResourceDeploymentPlanWithRights(context.Background(), scenarioPath, artifactRoot, "", []string{"linux-amd64"}, resourcedeployment.ArtifactTrustDevelopmentLocal, evaluator)
		if err != nil {
			t.Fatalf("resolve package plan: %v", err)
		}
		if len(plan.Resources) != 1 {
			t.Fatalf("package plan has %d resources, want 1", len(plan.Resources))
		}
		return plan
	}

	plan := resolve()
	if item := plan.Resources[0]; item.Eligibility != "eligible" {
		t.Fatalf("operator's current below-threshold context did not allow declared weights: %+v", item)
	}
	currentContext.Version = "above-threshold"
	currentContext.MonthlyActiveUsers["2024-08"] = "700000001"
	currentContext.MonthlyActiveUsersEvidence["2024-08"] = "test threshold crossing"
	writeContext()
	plan = resolve()
	item := plan.Resources[0]
	if plan.Promotable || item.Eligibility != "ineligible" || !strings.Contains(item.EligibilityReason, "monthly_active_users_condition_not_satisfied") || !strings.Contains(strings.Join(item.Evidence, "\n"), "llama3.2:1b") {
		t.Fatalf("crossed trusted-context threshold did not refuse the package plan: %+v", item)
	}
	currentContext.Version = "restored-below-threshold"
	currentContext.MonthlyActiveUsers["2024-08"] = "0"
	currentContext.MonthlyActiveUsersEvidence["2024-08"] = "test restoration"
	writeContext()
	plan = resolve()
	if item := plan.Resources[0]; item.Eligibility != "eligible" {
		t.Fatalf("restored trusted context did not restore package eligibility: %+v", item)
	}
}
