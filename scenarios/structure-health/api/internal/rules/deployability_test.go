package rules

import (
	"os"
	"path/filepath"
	"strings"
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

// TestEvaluateFindingsStayInsideTarget is the invariant that the control-plane
// scope leak violated: a scenario cannot act on a finding located outside its
// own tree, so Evaluate must never return one. This guards the whole scenario
// rule set, not just the rule that broke it.
func TestEvaluateFindingsStayInsideTarget(t *testing.T) {
	repo := t.TempDir()
	// A control-plane file that would trip the deployability rule if it ran.
	if err := os.MkdirAll(filepath.Join(repo, "internal", "deployability"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "resources", "declared-resource"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "resources", "declared-resource", "resource.json"), []byte(`{"name":"declared-resource"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "internal", "deployability", "resolver.go"),
		[]byte("package deployability\nconst name = \"declared-resource\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	scenarioRoot := filepath.Join(repo, "scenarios", "example")
	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}

	findings := Evaluate(Input{Model: reconcile.Model{Scenario: "example", RootPath: scenarioRoot}})
	for _, f := range findings {
		if filepath.IsAbs(f.Location) && !strings.HasPrefix(filepath.Clean(f.Location), filepath.Clean(scenarioRoot)) {
			t.Errorf("finding %s is located outside the validated target: %s", f.Code, f.Location)
		}
		if f.Code == deployabilityInstanceRule {
			t.Errorf("control-plane rule %s ran on a scenario target", f.Code)
		}
	}
}

// TestDeployabilityInstanceRuleSkipsTestFiles keeps fixtures out of the report.
// A test must name a concrete object to exercise the resolver; that is not the
// defect the rule exists to find.
func TestDeployabilityInstanceRuleSkipsTestFiles(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(root, "internal", "deployability", "resolver_test.go"),
		[]byte("package deployability\nconst fixture = \"declared-resource\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := deployabilityInstanceRules(Input{Model: reconcile.Model{RootPath: root}}); len(findings) != 0 {
		t.Fatalf("findings from a _test.go fixture = %+v, want none", findings)
	}
}

// TestDeployabilityInstanceVocabularyExcludesTools pins the vocabulary
// narrowing: a tool is an ambient binary this code invokes, not an object it
// deploys, so exec("go", ...) is not a violation.
func TestDeployabilityInstanceVocabularyExcludesTools(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "deployability"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "tools", "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "tools", "go", "tool.json"), []byte(`{"name":"go"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "deployability", "run.go"),
		[]byte("package deployability\nfunc run() []string { return []string{\"go\", \"vet\"} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := deployabilityInstanceRules(Input{Model: reconcile.Model{RootPath: root}}); len(findings) != 0 {
		t.Fatalf("findings for a tool literal = %+v, want none", findings)
	}
}

// TestDeployabilityInstanceVocabularyUsesScenarioManifest pins that a scenario
// is named by its manifest, not its directory, so a non-scenario directory
// under scenarios/ never enters the vocabulary.
func TestDeployabilityInstanceVocabularyUsesScenarioManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "deployability"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A directory under scenarios/ that is not a scenario: no service.json.
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A real scenario, named by its manifest.
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "real-one", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "real-one", ".vrooli", "service.json"), []byte(`{"service":{"name":"real-one"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "deployability", "a.go"),
		[]byte("package deployability\nconst a = \"scenarios\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := deployabilityInstanceRules(Input{Model: reconcile.Model{RootPath: root}}); len(findings) != 0 {
		t.Fatalf("findings for a non-scenario directory name = %+v, want none", findings)
	}

	if err := os.WriteFile(filepath.Join(root, "internal", "deployability", "a.go"),
		[]byte("package deployability\nconst a = \"real-one\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := deployabilityInstanceRules(Input{Model: reconcile.Model{RootPath: root}}); len(findings) != 1 {
		t.Fatalf("findings for a manifest-declared scenario name = %+v, want one", findings)
	}
}

// TestDeployabilityInstanceRuleSkipsCommandWiring pins the cmd/ exclusion: a
// command that dials a service by name is the wiring boundary, which is where
// names are supposed to enter. Decision code outside cmd/ is still scanned.
func TestDeployabilityInstanceRuleSkipsCommandWiring(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "internal", "deployability", "cmd", "some-tool"),
		filepath.Join(root, "resources", "declared-resource"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "declared-resource", "resource.json"), []byte(`{"name":"declared-resource"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "deployability", "cmd", "some-tool", "main.go"),
		[]byte("package main\nconst target = \"declared-resource\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := deployabilityInstanceRules(Input{Model: reconcile.Model{RootPath: root}}); len(findings) != 0 {
		t.Fatalf("findings from cmd/ wiring = %+v, want none", findings)
	}

	// The same literal in decision code is still a violation.
	if err := os.WriteFile(filepath.Join(root, "internal", "deployability", "resolver.go"),
		[]byte("package deployability\nconst target = \"declared-resource\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := deployabilityInstanceRules(Input{Model: reconcile.Model{RootPath: root}}); len(findings) != 1 {
		t.Fatalf("findings from decision code = %+v, want one", findings)
	}
}
