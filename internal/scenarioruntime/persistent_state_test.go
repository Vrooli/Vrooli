package scenarioruntime_test

import (
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
)

// The governed build applies these checks when it stamps the persistent-state
// contract; running them here surfaces a violation, or an unclassified
// constant, during ordinary development instead of as a refused activation.
func TestPersistentStateSourceChecks(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	findings, err := buildinfo.PersistentStateSourceFindings(root, "internal/scenarioruntime")
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
}

// Callers anywhere in the repository write process-ref statuses and demand
// audit operations; they must use this package's constants too.
func TestStoredValuesAreNeverWrittenAsLiterals(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	found, err := buildinfo.StoredValueLiteralCalls(root, "internal", "cmd")
	if err != nil {
		t.Fatal(err)
	}
	for _, location := range found {
		t.Errorf("%s writes a stored runtime-registry value as a string literal; use the scenarioruntime constant", location)
	}
}
