package rules

import (
	"os"
	"path/filepath"
	"testing"

	"structure-health/internal/reconcile"
)

func TestDeployabilityInstanceRuleDetectsAndClearsManifestLiteral(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "deployability"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "declared-resource"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "declared-resource", "resource.json"), []byte(`{"name":"declared-resource"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "internal", "deployability", "resolver.go")
	if err := os.WriteFile(sourcePath, []byte("package deployability\nconst name = \"declared-resource\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := Input{Model: reconcile.Model{RootPath: root}}
	findings := deployabilityInstanceRules(input)
	if len(findings) != 1 || findings[0].Code != deployabilityInstanceRule {
		t.Fatalf("findings = %+v, want one instance-identifier finding", findings)
	}

	if err := os.WriteFile(sourcePath, []byte("package deployability\nconst name = \"manifest-derived\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := deployabilityInstanceRules(input); len(findings) != 0 {
		t.Fatalf("findings after removing literal = %+v, want none", findings)
	}
}
