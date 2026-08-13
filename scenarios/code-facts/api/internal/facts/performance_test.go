package facts

import (
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestAssertFamilyCostFailsWhenSyntheticBudgetIsLowered(t *testing.T) {
	cost := FamilyCost{Family: "file_domain", Target: "scenario:search-hub", ColdMS: 420, WarmMS: 95}
	if err := AssertFamilyCost(cost, 1000, 200); err != nil {
		t.Fatalf("measured cost rejected by normal budget: %v", err)
	}
	if err := AssertFamilyCost(cost, 100, 200); err == nil {
		t.Fatal("lowered synthetic cold budget did not fail")
	}
}

func TestUnitFingerprintInvalidatesOnlyTouchedParseUnit(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0o755); err != nil {
		t.Fatal(err)
	}
	firstFile := filepath.Join(first, "a.go")
	secondFile := filepath.Join(second, "b.go")
	if err := os.WriteFile(firstFile, []byte("package first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondFile, []byte("package second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstUnit := &factsv1.ParseUnit{RootPath: first}
	secondUnit := &factsv1.ParseUnit{RootPath: second}
	firstBefore, _ := sourceFingerprintForUnit(firstUnit)
	secondBefore, _ := sourceFingerprintForUnit(secondUnit)
	if err := os.WriteFile(firstFile, []byte("package first\n\nvar changed = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstAfter, _ := sourceFingerprintForUnit(firstUnit)
	secondAfter, _ := sourceFingerprintForUnit(secondUnit)
	if firstBefore == firstAfter {
		t.Fatal("touched parse unit retained its source fingerprint")
	}
	if secondBefore != secondAfter {
		t.Fatal("untouched parse unit fingerprint changed")
	}
}
