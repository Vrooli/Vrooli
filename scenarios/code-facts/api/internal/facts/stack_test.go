package facts

import (
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestDescribeStackFactsPreservesDeclarationsAndUnknowns(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"components":{"api":{"role":"api","build":{"kind":"go_module","dir":"api"}}},"dependencies":{"resources":{"postgres":{"enabled":true}}}}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	facts := describeStackFacts(root)
	if len(facts) != 2 {
		t.Fatalf("facts=%+v", facts)
	}
	component := facts[0]
	if component.GetFamily() != factsv1.FactFamily_FACT_FAMILY_STACK || component.GetAttributes()["build.kind"] != "go_module" || component.GetAttributes()["language"] != "go" {
		t.Fatalf("component=%+v", component)
	}
	if component.GetAttributes()["ui_framework"] != "" {
		t.Fatalf("unknown framework was guessed: %+v", component.GetAttributes())
	}
	foundUnknown := false
	for _, ev := range component.GetEvidence() {
		if ev.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN {
			foundUnknown = true
		}
	}
	if !foundUnknown {
		t.Fatalf("unknown framework/bundler lacked unknown evidence: %+v", component.GetEvidence())
	}
	if facts[1].GetKind() != "resource" || facts[1].GetSubject() != "postgres" {
		t.Fatalf("resource=%+v", facts[1])
	}
}
