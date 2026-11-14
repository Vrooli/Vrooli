package hygiene

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// pathAuthorityFixture is a contract-bearing repository whose source roots
// hold files (repo-relative path -> Go source).
func pathAuthorityFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	fixture := repocontracttest.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "demo")
	root := fixture.Root
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const homeRuntimeJoin = "package bad\nvar _ = filepath.Join(home, \".vrooli\")\n"

func findingCodes(report Report) map[string]Finding {
	codes := map[string]Finding{}
	for _, finding := range report.Findings {
		codes[finding.Code] = finding
	}
	return codes
}

// The scan covers every source root, not only scenarios, and skips owner docs
// trees and test files.
func TestPathAuthorityProviderRejectsHomeRuntimeJoinInAnySourceRoot(t *testing.T) {
	root := pathAuthorityFixture(t, map[string]string{
		"scenarios/demo/api/bad.go":                homeRuntimeJoin,
		"internal/thing/bad.go":                    homeRuntimeJoin,
		"packages/lib/bad.go":                      homeRuntimeJoin,
		"scenarios/demo/docs/evidence/snapshot.go": homeRuntimeJoin,
		"scenarios/demo/api/bad_test.go":           homeRuntimeJoin,
	})
	var report Report
	if err := (pathAuthorityProvider{root: root, debt: map[string]int{}}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	finding, ok := findingCodes(report)["runtime_path_authority_bypass"]
	want := []string{"internal/thing/bad.go:2", "packages/lib/bad.go:2", "scenarios/demo/api/bad.go:2"}
	if !ok || strings.Join(finding.Locations, ",") != strings.Join(want, ",") {
		t.Fatalf("bypass finding = %+v, want locations %v", report.Findings, want)
	}
}

// The ledger tolerates exactly its recorded sites as a warning, rejects any
// growth, and reports an entry that allows more than its file still holds.
func TestPathAuthorityProviderLedgerOnlyShrinks(t *testing.T) {
	root := pathAuthorityFixture(t, map[string]string{
		"scenarios/demo/api/known.go": homeRuntimeJoin,
		"scenarios/demo/api/grown.go": homeRuntimeJoin + "var _ = filepath.Join(home, \".vrooli\", \"bin\")\n",
	})
	debt := map[string]int{"scenarios/demo/api/known.go": 1, "scenarios/demo/api/grown.go": 1, "scenarios/demo/api/fixed.go": 1}
	var report Report
	if err := (pathAuthorityProvider{root: root, debt: debt}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	codes := findingCodes(report)
	if got := codes["runtime_path_authority_debt"]; got.Severity != SeverityWarning || strings.Join(got.Locations, ",") != "scenarios/demo/api/known.go:2" {
		t.Errorf("ledgered finding = %+v", got)
	}
	if got := codes["runtime_path_authority_bypass"]; got.Severity != SeverityError || strings.Join(got.Locations, ",") != "scenarios/demo/api/grown.go:2,scenarios/demo/api/grown.go:3" {
		t.Errorf("growth finding = %+v", got)
	}
	if got := codes["runtime_path_authority_debt_stale"]; len(got.Locations) != 1 || !strings.HasPrefix(got.Locations[0], "scenarios/demo/api/fixed.go") {
		t.Errorf("stale finding = %+v", got)
	}
}

// Declaring the runtime-home-relative path as a constant and joining it to
// the home later is the same bypass; a repository-relative ".vrooli" join and
// a constant never joined to a home are not.
func TestRuntimeHomeBypassLinesCatchDeclaredRelativePaths(t *testing.T) {
	source := strings.Join([]string{
		"package p",
		"const statusRelPath = \".vrooli/state/x/status.json\"",
		"func a(home string) string { return filepath.Join(home, filepath.FromSlash(statusRelPath)) }",
		"func b(root string) string { return filepath.Join(root, \".vrooli\", \"service.json\") }",
		"var marker = []byte(\".vrooli/evidence/\")",
		"func c(home string) string { return filepath.Join(home, \".codex\", \"sessions\") }",
		"func d(home string) string { return filepath.Join(home, \".vrooli\", \"bin\") }",
	}, "\n")
	got := runtimeHomeBypassLines(source)
	if len(got) != 2 || got[0] != 3 || got[1] != 7 {
		t.Fatalf("bypass lines = %v, want [3 7]", got)
	}
}

func TestPathAuthorityProviderPassesMigratedTree(t *testing.T) {
	root := pathAuthorityFixture(t, map[string]string{
		"scenarios/demo/api/good.go": "package good\nvar _ = filepath.Join(stateRoot, \"locks\")\n",
	})
	var report Report
	if err := (pathAuthorityProvider{root: root, debt: map[string]int{}}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("migrated tree findings = %+v", report.Findings)
	}
}

func TestPathAuthorityProviderCurrentRepositoryIsClean(t *testing.T) {
	var report Report
	if err := (pathAuthorityProvider{root: filepath.Clean(filepath.Join("..", "..", ".."))}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	for _, finding := range report.Findings {
		if finding.Code != "runtime_path_authority_debt" {
			t.Errorf("repository finding %s: %v", finding.Code, finding.Locations)
		}
	}
}
