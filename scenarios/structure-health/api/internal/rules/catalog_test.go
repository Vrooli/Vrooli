package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogHasUniqueCompleteRules(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatal(err)
	}
	if got := len(Catalog()); got < 30 {
		t.Fatalf("catalog size = %d, want at least 30 structural rules", got)
	}
}

func TestCatalogCoversEveryDeclaredTargetKind(t *testing.T) {
	for _, row := range Coverage() {
		if !row.Reachable || row.CallerCount < 2 {
			t.Fatalf("unreachable target kind: %#v", row)
		}
		if row.RuleCount == 0 {
			t.Fatalf("target kind %q has no registered rules", row.TargetKind)
		}
	}
}

func TestGeneratedRuleDocumentationContainsCatalogClaims(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "reference", "structure-rules.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "GENERATED FILE") {
		t.Fatal("generated marker missing")
	}
	for _, entry := range Catalog() {
		if !strings.Contains(text, "| "+entry.Code+" |") {
			t.Errorf("generated catalog is missing %s", entry.Code)
		}
		if entry.Claim != "" && !strings.Contains(text, "| "+entry.Code+" | "+entry.TargetKind+" | "+entry.Severity+" | "+string(entry.Enforcement)+" | "+entry.Claim+" |") {
			t.Errorf("generated catalog claim is stale for %s", entry.Code)
		}
	}
}
