package manifestschema

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScenarioEnvironmentRulesMotivatingManifestsPass(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../../../.."))
	for _, name := range []string{"calendar", "api-library"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(repoRoot, "scenarios", name, ".vrooli", "service.json")
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
