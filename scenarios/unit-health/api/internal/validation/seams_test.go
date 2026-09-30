package validation

import (
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestMethodSetsSubsetIgnoresInterfaceNameAndMatchesSubset(t *testing.T) {
	smaller := seamMethodSet{"Now": "() time.Time"}
	larger := seamMethodSet{"Now": "() time.Time", "NewTimer": "(time.Duration) Timer"}

	if !methodSetsSubset(smaller, larger) {
		t.Fatal("Now-only seam should be a subset of Now+NewTimer")
	}
	if methodSetsSubset(larger, smaller) {
		t.Fatal("larger seam must not be a subset of Now-only seam")
	}
}

func TestFindDuplicatedPackageSeamsReportsSubsetInterfacesAndSleeper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/api-core\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "clock.go"), `package api

import "time"

type DatabaseClock interface { Now() time.Time }
type RetentionClock interface { Now() time.Time; NewTimer(time.Duration) Timer }
type FileClock interface { Now() time.Time }
type Timer interface { Stop() bool }
`)
	writeFile(t, filepath.Join(root, "retry.go"), `package api

import "time"

type Config struct { Sleeper func(time.Duration) }
`)

	findings := findDuplicatedPackageSeams("api-core", Workspace{ID: "api-core", Language: "go", RootPath: root}, fixedNowStr)
	if len(findings) != 4 {
		t.Fatalf("findings = %d, want four duplicate declarations: %+v", len(findings), findings)
	}
	for _, finding := range findings {
		if finding.Code != codeSeamDuplicatedInPackage || finding.Severity != "error" {
			t.Fatalf("unexpected finding: %+v", finding)
		}
		if !strings.Contains(finding.Remediation, "api-core/schedule.Clock") {
			t.Errorf("remediation does not name shared symbol: %q", finding.Remediation)
		}
	}
}

func TestFindDuplicatedPackageSeamsAllowsSingleSharedInterface(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/api-core\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "clock.go"), `package api

import "github.com/vrooli/api-core/schedule"

type Service struct { Clock schedule.Clock }
`)

	findings := findDuplicatedPackageSeams("api-core", Workspace{ID: "api-core", Language: "go", RootPath: root}, fixedNowStr)
	if len(findings) != 0 {
		t.Fatalf("adopted single seam must be clean, got %+v", findings)
	}
}

func TestSeamDuplicateWaiverSuppressesFindingAfterAllAnalyzers(t *testing.T) {
	findings := []Finding{{Code: codeSeamDuplicatedInPackage, Severity: "error"}}
	waivers := []unitPolicyWaiver{{
		Finding:  codeSeamDuplicatedInPackage,
		Reason:   "migration window",
		Owner:    "architecture-team",
		Evidence: "ticket-123",
		Revisit:  "after schedule migration",
	}}

	marked := applyUnitPolicyWaivers(findings, waivers, "api-core", "/tmp/testing.json", fixedNowStr)
	if len(marked) != 1 || !marked[0].Suppressed {
		t.Fatalf("valid waiver must suppress seam finding, got %+v", marked)
	}
	active, suppressed := splitSuppressedFindings(marked)
	if len(active) != 0 || len(suppressed) != 1 || suppressed[0].Code != codeSeamDuplicatedInPackage {
		t.Fatalf("suppressed seam finding was not separated: active=%+v suppressed=%+v", active, suppressed)
	}
}

func TestCurrentAPICoreHasNoDuplicatedProductionSeams(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	findings := findDuplicatedPackageSeams("api-core", Workspace{
		ID: "api-core", Language: "go", RootPath: filepath.Join(root, "packages", "api-core"),
	}, fixedNowStr)
	if len(findings) != 0 {
		t.Fatalf("current api-core must have no duplicated production seams: %+v", findings)
	}
}
