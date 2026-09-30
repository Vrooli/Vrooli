package intent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractorsAndChecks(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "requirements", "core"))
	mkdir(t, filepath.Join(root, "api", "internal", "core"))
	write(t, filepath.Join(root, "PRD.md"), `# Product
## Operational Targets
- [ ] OT-P0-001 | Core | Works
- [ ] OT-P1-001 | Extra | Later
`)
	write(t, filepath.Join(root, "api", "internal", "core", "core_test.go"), "package core\n")
	write(t, filepath.Join(root, "requirements", "core", "module.json"), `{
  "requirements": [
    {
      "id": "REQ-1",
      "title": "Core",
      "description": "Core works",
      "prd_ref": "OT-P0-001",
      "validation": [
        {"type": "test", "ref": "api/internal/core/core_test.go::TestCore"},
        {"type": "test", "ref": "api/internal/core/*_missing.go"},
        {"type": "manual", "ref": "Attended review"}
      ]
    },
    {
      "id": "REQ-2",
      "title": "Broken",
      "prd_ref": "OT-P0-999",
      "validation": [{"type": "test", "ref": "api/internal/core/missing_test.go#case"}]
    }
  ]
}`)

	outcomes, err := (FilePRDExtractor{}).ExtractPRDClaims(root)
	if err != nil {
		t.Fatal(err)
	}
	requirements, err := (FileRequirementsExtractor{}).ExtractRequirementClaims(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(outcomes), 2; got != want {
		t.Fatalf("outcome count = %d, want %d", got, want)
	}
	if got, want := len(requirements), 2; got != want {
		t.Fatalf("requirement count = %d, want %d", got, want)
	}
	if got := requirements[0].Refs[1]; got.Path != "api/internal/core/core_test.go" || got.Member != "TestCore" || got.Kind != RefCode {
		t.Fatalf("normalized member ref = %+v", got)
	}
	if got := requirements[0].Refs[2]; got.Path != "api/internal/core" || !got.Glob {
		t.Fatalf("normalized glob ref = %+v", got)
	}
	if got := requirements[0].Refs[3]; got.Kind != RefManual {
		t.Fatalf("manual ref kind = %+v", got)
	}

	if got := CheckPRDRefResolves(outcomes, requirements); len(got) != 1 || got[0].Code != CodePRDRefUnmatched {
		t.Fatalf("prd ref findings = %+v", got)
	}
	if got := CheckOrphanOutcome(outcomes, requirements); len(got) != 1 || got[0].ClaimID != "OT-P1-001" {
		t.Fatalf("orphan findings = %+v", got)
	}
	if got := CheckRefExists(root, requirements); len(got) != 1 {
		t.Fatalf("ref missing findings = %+v", got)
	}
}

func TestNormalizeRefLegacyColonAndDocs(t *testing.T) {
	ref := NormalizeRef("docs/concepts/DOMAINS.md#inventory", "test")
	if ref.Path != "docs/concepts/DOMAINS.md" || ref.Kind != RefDoc {
		t.Fatalf("doc ref = %+v", ref)
	}
	ref = NormalizeRef("api/foo_test.go:TestFoo", "test")
	if ref.Path != "api/foo_test.go" || ref.Member != "TestFoo" {
		t.Fatalf("legacy colon ref = %+v", ref)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
