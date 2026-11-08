package manifestschema

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Every fleet manifest, including the ones that motivated these rules, passes
// the environment contracts; a new violation names its manifest.
func TestScenarioEnvironmentRulesFleetManifestsPass(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../../../.."))
	paths, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no scenario manifests found")
	}
	for _, path := range paths {
		name := filepath.Base(filepath.Dir(filepath.Dir(path)))
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := CheckScenarioHardcodedPeerAddress(content, path); len(got) != 0 {
				t.Fatalf("hardcoded peer findings = %#v", got)
			}
			if got := CheckScenarioSecretLiteral(content, path); len(got) != 0 {
				t.Fatalf("secret findings = %#v", got)
			}
			if got := CheckScenarioRedeclaresResourceEnv(content, path); len(got) != 0 {
				t.Fatalf("resource redeclaration findings = %#v", got)
			}
		})
	}
}

func TestScenarioEnvironmentRulesRejectForbiddenValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenarios", "calendar", ".vrooli", "service.json")
	content := []byte(`{
  "dependencies": {"scenarios": {"auth": {}}, "resources": {}},
  "components": {"api": {"run": {"env": {
    "AUTH_URL": "http://localhost:15785",
    "JWT_SECRET": "development-secret",
    "OPAQUE": "aB9sK2mN4pQ7rT8vW3xY6zC1"
  }}}}
}`)
	if got := CheckScenarioHardcodedPeerAddress(content, path); len(got) != 1 {
		t.Fatalf("peer findings = %#v", got)
	}
	if got := CheckScenarioSecretLiteral(content, path); len(got) != 2 {
		t.Fatalf("secret findings = %#v", got)
	}
}
