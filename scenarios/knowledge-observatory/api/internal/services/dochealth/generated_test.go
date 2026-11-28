package dochealth

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProgramRuntimeGuideGeneratedTableIsCurrent(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	guide := filepath.Join(root, "docs", "guides", "getting-started.md")
	if findings := generatedBlockFindings([]string{guide}, root); len(findings) != 0 {
		t.Fatalf("Program Runtime guide table must match its authoritative contracts: %+v", findings)
	}
}

func TestImageToolsBackendMatrixGeneratedBlockIsCurrent(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	imageRoot := filepath.Join(repo, "scenarios", "image-tools")
	doc := filepath.Join(imageRoot, "docs", "reference", "backends.md")
	if findings := generatedBlockFindings([]string{doc}, imageRoot); len(findings) != 0 {
		t.Fatalf("Image Tools backend matrix must match its renderer source and output hashes: %+v", findings)
	}
}

func TestBrowserAutomationStudioNodeIndexGeneratedBlockIsCurrent(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	basRoot := filepath.Join(repo, "scenarios", "browser-automation-studio")
	doc := filepath.Join(basRoot, "docs", "NODE_INDEX.md")
	if findings := generatedBlockFindings([]string{doc}, basRoot); len(findings) != 0 {
		t.Fatalf("Browser Automation Studio node index must match its UI registry and rendered content hashes: %+v", findings)
	}
}

func TestBrowserAutomationStudioActionTypesGeneratedBlockIsCurrent(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	basRoot := filepath.Join(repo, "scenarios", "browser-automation-studio")
	doc := filepath.Join(basRoot, "docs", "architecture", "recording.md")
	if findings := generatedBlockFindings([]string{doc}, basRoot); len(findings) != 0 {
		t.Fatalf("recording action-type table must match its API owner source and output hashes: %+v", findings)
	}
}

func TestGeneratedSourceDigestAcceptsCanonicalRootAlias(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, ".vrooli", "program-runtime")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "sample.json"), []byte(`{"name":"sample"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	block := "| program |\n|---|\n| sample |"
	marker := generatedTableMarker(t, "sample.json", block)
	writeAssertionFixture(t, root, "docs/generated.md", marker+"\n"+block+"\n"+generatedBlockEnd+"\n")
	alias := filepath.Join(t.TempDir(), "scenario")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	aliasedDoc := filepath.Join(alias, "docs", "generated.md")
	if findings := generatedBlockFindings([]string{aliasedDoc}, alias); len(findings) != 0 {
		t.Fatalf("a source directory inside the canonical scenario root should verify through its alias: %+v", findings)
	}
}

func TestGeneratedSourceDigestAcceptsExactOwnerFile(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "ui", "src", "constants")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "nodeCategories.ts"), []byte("export const registry = {};\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := generatedSourceDigest(root, "ui/src/constants/nodeCategories.ts"); err != nil {
		t.Fatalf("exact owner source file should be verifiable: %v", err)
	}
}

func TestGeneratedNodeIndexRejectsChangedExactOwnerFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "ui", "src", "constants"), 0o755); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "ui", "src", "constants", "nodeCategories.ts")
	if err := os.WriteFile(sourcePath, []byte("export const oldRegistry = {};\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceHash, err := generatedSourceDigest(root, "ui/src/constants/nodeCategories.ts")
	if err != nil {
		t.Fatal(err)
	}
	body := "| Node |\n| --- |\n| Navigate |"
	bodyHash := sha256.Sum256([]byte(body))
	marker := fmt.Sprintf("<!-- BEGIN GENERATED NODE INDEX source=ui/src/constants/nodeCategories.ts source_sha256=%s block_sha256=%x -->", sourceHash, bodyHash)
	doc := writeAssertionFixture(t, root, "docs/NODE_INDEX.md", marker+"\n"+body+"\n<!-- END GENERATED NODE INDEX -->\n")
	if err := os.WriteFile(sourcePath, []byte("export const currentRegistry = {};\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	findings := generatedBlockFindings([]string{doc}, root)
	if len(findings) != 1 || findings[0].Code != "stale_generated_block" || findings[0].Severity != SeverityFailure {
		t.Fatalf("a changed exact owner source file must fail its generated node index: %+v", findings)
	}
}
