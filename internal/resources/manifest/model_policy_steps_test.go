package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	capacitypkg "github.com/vrooli/vrooli/internal/capacity"
)

func TestPolicyDerivedStepsRejectRungThatServesNoRole(t *testing.T) {
	resourceManifest, path := policyStepFixture(t, []capacitypkg.DegradeStep{
		{Label: "model-a", AmountBytes: 100},
		{Label: "model-b", AmountBytes: 50},
	})
	err := validateMaterializedPolicySteps(path, resourceManifest)
	if err == nil || !strings.Contains(err.Error(), `rung "model-b" serves no role`) {
		t.Fatalf("validation error = %v, want orphan rung by name", err)
	}
}

func TestPolicyDerivedStepsRejectServedModelWithoutRung(t *testing.T) {
	resourceManifest, path := policyStepFixture(t, nil)
	err := validateMaterializedPolicySteps(path, resourceManifest)
	if err == nil || !strings.Contains(err.Error(), `policy-served model "model-a"`) {
		t.Fatalf("validation error = %v, want missing policy model by name", err)
	}
}

func TestRealPolicyDerivedProfilesAgree(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ollama", "reranker"} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(filepath.Join(root, "resources", name, "resource.json")); err != nil {
				t.Fatalf("Load(%s) error = %v", name, err)
			}
		})
	}
}

func TestPolicyDerivedStepsAllowTerminalCPURung(t *testing.T) {
	resourceManifest, path := policyStepFixture(t, []capacitypkg.DegradeStep{
		{Label: "model-a", AmountBytes: 100},
		{Label: "cpu", AmountBytes: 0},
	})
	resourceManifest.Acceleration.Claim.FloorBytes = 0
	if err := validateMaterializedPolicySteps(path, resourceManifest); err != nil {
		t.Fatalf("validation error = %v, want terminal cpu rung accepted", err)
	}
}

func policyStepFixture(t *testing.T, steps []capacitypkg.DegradeStep) (ResourceManifest, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "resource.json")
	if err := os.WriteFile(filepath.Join(dir, "model-policy.json"), []byte(`{
	  "roles":{"role.default":{"model":"model-a"}},
	  "models":{"model-a":{"resident_bytes":100}}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := &capacitypkg.DegradeProfile{StepsSource: "model-policy.json", Steps: steps}
	return ResourceManifest{Acceleration: &AccelerationSpec{Claim: &capacitypkg.ResourceClaimSpec{
		PreferredBytes: 100, FloorBytes: 100, Confidence: "measured", Profile: profile,
	}}}, path
}
