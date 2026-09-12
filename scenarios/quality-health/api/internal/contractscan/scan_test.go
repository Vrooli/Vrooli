package contractscan

import (
	"os"
	"path/filepath"
	"testing"
)

func writeScanFixture(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasCode(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func TestContractScanPositiveFixture(t *testing.T) {
	root := t.TempDir()
	writeScanFixture(t, root, "internal/clean/clean.go", "package clean\n")
	findings, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("clean fixture findings = %+v", findings)
	}
}

func TestPersonalAbsolutePathNegativeFixture(t *testing.T) {
	root := t.TempDir()
	writeScanFixture(t, root, "internal/drift/drift.go", "package drift\nconst path = \"/home/alice/Vrooli\"\n")
	findings, _ := Scan(root)
	if !hasCode(findings, "QUALITY_PERSONAL_ABSOLUTE_PATHS") {
		t.Fatalf("expected personal path finding, got %+v", findings)
	}
}

func TestRuntimeHomeNegativeAndAnnotationFixture(t *testing.T) {
	root := t.TempDir()
	writeScanFixture(t, root, "internal/drift/drift.go", "package drift\nimport \"path/filepath\"\nfunc f(home string) string { return filepath.Join(home, \".vrooli\", \"logs\") }\n")
	findings, _ := Scan(root)
	if !hasCode(findings, "QUALITY_NO_RUNTIME_HOME_LITERALS") {
		t.Fatalf("expected runtime-home finding, got %+v", findings)
	}
	writeScanFixture(t, root, "internal/drift/ok.go", "package drift\nimport \"path/filepath\"\nfunc f(home string) string { return filepath.Join(home, \".vrooli\") // repo-contract:project-config\n}\n")
}

func TestAdoptionNegativeFixture(t *testing.T) {
	root := t.TempDir()
	writeScanFixture(t, root, "internal/drift/drift.go", "package drift\nfunc findRepoRoot(path string) string { return path }\n")
	findings, _ := Scan(root)
	if !hasCode(findings, "QUALITY_ADOPTION_RULES_ALIGNMENT") {
		t.Fatalf("expected adoption finding, got %+v", findings)
	}
}

func TestHostInventoryNegativeFixture(t *testing.T) {
	root := t.TempDir()
	writeScanFixture(t, root, "internal/drift/drift.go", "package drift\nimport \"os\"\nfunc f() []byte { data, _ := os.ReadFile(\"/proc/meminfo\"); return data }\n")
	findings, _ := Scan(root)
	if !hasCode(findings, "QUALITY_HOST_INVENTORY_AUTHORITY") {
		t.Fatalf("expected host inventory finding, got %+v", findings)
	}
}
