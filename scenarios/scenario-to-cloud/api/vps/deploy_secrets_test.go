package vps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"scenario-to-cloud/domain"

	repocontract "github.com/vrooli/repo-contract-go"
)

// buildUserSecretMap must not read a plaintext file. Both stores it used to
// consult are gone, and the point of removing them is that a cloud deploy can
// no longer depend on — or silently recreate — a credential sitting
// unencrypted on the operator's disk. A file placed where the old code looked
// must therefore have no effect whatsoever.
func TestBuildUserSecretMapNeverReadsAPlaintextFile(t *testing.T) {
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", root)
	home := t.TempDir()
	t.Setenv("HOME", home)

	workspacePath, err := repocontract.UserPlaintextSecretsPath(home)
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	scenarioPath, err := repocontract.UserScenarioPlaintextSecretsPath(home, "demo")
	if err != nil {
		t.Fatalf("UserScenarioPlaintextSecretsPath: %v", err)
	}
	// Bait: values in exactly the two places the old implementation merged from.
	writeJSONFile(t, workspacePath, map[string]interface{}{"API_KEY": "from-workspace-plaintext"})
	writeJSONFile(t, scenarioPath, map[string]interface{}{"API_KEY": "from-scenario-plaintext"})

	manifest := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "demo"},
		Secrets: &domain.ManifestSecrets{
			BundleSecrets: []domain.BundleSecretPlan{{
				ID:       "api-key",
				Class:    "user_prompt",
				Required: true,
				Target:   domain.BundleSecretTarget{Name: "API_KEY"},
			}},
		},
	}

	got := buildUserSecretMap(manifest, nil)
	if value, ok := got["API_KEY"]; ok && strings.Contains(value, "plaintext") {
		t.Fatalf("buildUserSecretMap read a plaintext file: API_KEY = %q", value)
	}

	// An explicitly supplied value is an instruction for this deploy and still
	// takes precedence over anything the store holds.
	got = buildUserSecretMap(manifest, map[string]string{"API_KEY": "provided"})
	if got["API_KEY"] != "provided" {
		t.Fatalf("buildUserSecretMap explicit = %q, want %q", got["API_KEY"], "provided")
	}
}

// The field name a bundle secret resolves to must match the one the remote
// provisioning path writes, or a value lands under a key nothing reads back.
func TestCredentialFieldForMatchesTheRemoteProvisioningNormalization(t *testing.T) {
	for _, testCase := range []struct{ id, target, want string }{
		{id: "api-key", target: "API_KEY", want: "api-key"},
		{id: "", target: "SESSION_SECRET", want: "session-secret"},
		{id: "cloudflare.api_token", target: "", want: "cloudflare-api-token"},
		{id: "", target: "", want: ""},
	} {
		plan := domain.BundleSecretPlan{ID: testCase.id, Target: domain.BundleSecretTarget{Name: testCase.target}}
		if got := credentialFieldFor(plan); got != testCase.want {
			t.Fatalf("credentialFieldFor(%q,%q) = %q, want %q", testCase.id, testCase.target, got, testCase.want)
		}
	}
}

func TestRequiredResourcesForScenarioUsesContractResolvedServicePath(t *testing.T) {
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", root)

	writeJSONFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]interface{}{
		"service": map[string]interface{}{"name": "demo"},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{"enabled": true},
				"redis":    map[string]interface{}{"enabled": true},
				"vault":    map[string]interface{}{"enabled": false},
			},
		},
	})

	got, err := RequiredResourcesForScenario("demo")
	if err != nil {
		t.Fatalf("RequiredResourcesForScenario: %v", err)
	}
	if len(got) != 2 || got[0] != "postgres" || got[1] != "redis" {
		t.Fatalf("RequiredResourcesForScenario = %#v, want [postgres redis]", got)
	}
}

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()

	for _, dir := range []string{".vrooli", "scenarios", "resources", "templates", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Copy the live repo's .vrooli/repo-contract.json verbatim rather than
	// hand-typing a literal. This keeps the single source of truth authoritative
	// and prevents the fixture from drifting when the contract schema gains a
	// required field (e.g. runtime_home).
	contract := liveRepoContract(t)
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
}

// liveRepoContract reads the repository's authoritative
// .vrooli/repo-contract.json by walking up from this source file until the
// contract is found, returning the raw bytes for verbatim copy into a fixture.
func liveRepoContract(t *testing.T) []byte {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed; cannot locate live repo contract")
	}
	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, ".vrooli", "repo-contract.json")
		if data, err := os.ReadFile(candidate); err == nil {
			return data
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate .vrooli/repo-contract.json above test package")
		}
		dir = parent
	}
}

func writeJSONFile(t *testing.T, path string, payload map[string]interface{}) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
