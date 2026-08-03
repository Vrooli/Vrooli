package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	corestorage "github.com/vrooli/api-core/storage"
)

// descriptorFindingCodes reads the phase descriptor's declared finding catalog.
// It reuses findRepoRoot from hygiene_dogfood_test.go.
func descriptorFindingCodes(t *testing.T) map[string]struct{} {
	t.Helper()
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("repo root not found; descriptor parity is only checkable in-tree")
	}
	raw, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "storage-manager", ".vrooli", "test-genie.json"))
	if err != nil {
		t.Fatalf("read descriptor: %v", err)
	}
	var descriptor struct {
		Maturity struct {
			Findings map[string]json.RawMessage `json:"findings"`
		} `json:"maturity"`
	}
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		t.Fatalf("parse descriptor: %v", err)
	}
	codes := make(map[string]struct{}, len(descriptor.Maturity.Findings))
	for code := range descriptor.Maturity.Findings {
		codes[code] = struct{}{}
	}
	return codes
}

func accountabilityContext(t *testing.T, entries ...corestorage.StorageEntry) AnalyzerContext {
	t.Helper()
	root := t.TempDir()
	manifest := filepath.Join(root, "scenarios", "fixture", ".vrooli", "service.json")
	return AnalyzerContext{
		RepoRoot: root,
		Scenario: "fixture",
		Owner: &corestorage.OwnerManifest{
			Kind:           corestorage.OwnerScenario,
			ID:             "fixture",
			ManifestPath:   manifest,
			StorageEntries: entries,
		},
	}
}

func onlyCode(t *testing.T, got []Finding) string {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("findings = %#v, want exactly one rung marker", got)
	}
	return got[0].Code
}

// An owner that declares nothing must not inherit the top rung by silence.
// This is the inversion the marker exists to prevent: the maturity engine
// starts each capability at its maximum and lowers it only on a finding.
func TestAccountabilityUndeclaredOwnerBlocksL1(t *testing.T) {
	got := accountabilityFindings(accountabilityContext(t), nil)
	if code := onlyCode(t, got); code != "STORAGE_ACCOUNTABILITY_NOT_DECLARED" {
		t.Fatalf("code = %q, want STORAGE_ACCOUNTABILITY_NOT_DECLARED", code)
	}
	if got[0].Severity != SeverityInfo {
		t.Fatalf("severity = %v, want SeverityInfo so coverage gaps never fail the phase", got[0].Severity)
	}
}

func TestAccountabilityReconciliationDefectBlocksL2(t *testing.T) {
	ac := accountabilityContext(t, corestorage.StorageEntry{
		Name:   "data",
		Path:   corestorage.PortablePath{Value: "data"},
		Budget: &corestorage.BudgetDeclaration{MaxBytes: "1GiB"},
	})
	prior := []Finding{{Code: "STORAGE_ENTRY_NO_WRITER"}}
	got := accountabilityFindings(ac, prior)
	if code := onlyCode(t, got); code != "STORAGE_ACCOUNTABILITY_NOT_RECONCILED" {
		t.Fatalf("code = %q, want STORAGE_ACCOUNTABILITY_NOT_RECONCILED", code)
	}
	if got[0].Severity != SeverityWarning {
		t.Fatalf("severity = %v, want SeverityWarning", got[0].Severity)
	}
}

// A reconciled declaration with no ceiling and no reclaim command is not
// governed. Before the marker existed this scored L3 "governed end to end" on
// a fleet where only 6 of 208 owners had any budget at all.
func TestAccountabilityUnboundedEntryBlocksL3(t *testing.T) {
	ac := accountabilityContext(t, corestorage.StorageEntry{
		Name: "recordings",
		Path: corestorage.PortablePath{Value: "recordings"},
	})
	got := accountabilityFindings(ac, nil)
	if code := onlyCode(t, got); code != "STORAGE_ACCOUNTABILITY_NOT_GOVERNED" {
		t.Fatalf("code = %q, want STORAGE_ACCOUNTABILITY_NOT_GOVERNED", code)
	}
}

func TestAccountabilityBudgetOrReclaimSatisfiesGovernance(t *testing.T) {
	for name, entry := range map[string]corestorage.StorageEntry{
		"budget": {
			Name:   "data",
			Path:   corestorage.PortablePath{Value: "data"},
			Budget: &corestorage.BudgetDeclaration{MaxBytes: "20GiB"},
		},
		"reclaim": {
			Name:    "build-cache",
			Path:    corestorage.PortablePath{Value: "cache"},
			Reclaim: &corestorage.ReclaimDeclaration{Command: "go clean -cache"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := accountabilityFindings(accountabilityContext(t, entry), nil); len(got) != 0 {
				t.Fatalf("findings = %#v, want none (governed)", got)
			}
		})
	}
}

// SQLite sidecars are bounded by the database they belong to, never by their
// own policy, so they must not hold an owner below L3.
func TestAccountabilitySQLiteSidecarsAreExemptFromGovernance(t *testing.T) {
	entries := []corestorage.StorageEntry{
		{Name: "db", Path: corestorage.PortablePath{Value: "data/app.db"}, Format: "sqlite", Budget: &corestorage.BudgetDeclaration{MaxBytes: "2GiB"}},
		{Name: "db-wal", Path: corestorage.PortablePath{Value: "data/app.db-wal"}},
		{Name: "db-shm", Path: corestorage.PortablePath{Value: "data/app.db-shm"}},
	}
	if got := accountabilityFindings(accountabilityContext(t, entries...), nil); len(got) != 0 {
		t.Fatalf("findings = %#v, want none; sidecars are not independently bounded", got)
	}
}

// A verbose byOS map is not a correctness defect, so it must stay advisory
// rather than holding the ladder at L1.
func TestAccountabilityTokenSupersedableDoesNotBlockReconciliation(t *testing.T) {
	ac := accountabilityContext(t, corestorage.StorageEntry{
		Name:   "data",
		Path:   corestorage.PortablePath{Value: "data"},
		Budget: &corestorage.BudgetDeclaration{MaxBytes: "1GiB"},
	})
	prior := []Finding{{Code: "STORAGE_TOKEN_SUPERSEDABLE"}}
	if got := accountabilityFindings(ac, prior); len(got) != 0 {
		t.Fatalf("findings = %#v, want none", got)
	}
}

// The rung is a ladder: the lowest blocked rung is the only one reported, so
// an owner is never told to fix governance while reconciliation is still open.
func TestAccountabilityReportsLowestBlockedRungOnly(t *testing.T) {
	ac := accountabilityContext(t, corestorage.StorageEntry{
		Name: "data",
		Path: corestorage.PortablePath{Value: "data"},
	})
	prior := []Finding{{Code: "STORAGE_ENTRY_CLASS_CONFLICT"}}
	if code := onlyCode(t, accountabilityFindings(ac, prior)); code != "STORAGE_ACCOUNTABILITY_NOT_RECONCILED" {
		t.Fatalf("code = %q, want the reconciliation rung to win over governance", code)
	}
}

// Every marker code must exist in the phase descriptor, otherwise the maturity
// engine resolves it through the fallback mapping and the rung silently stops
// working — the exact failure mode being fixed.
func TestAccountabilityCodesAreDeclaredInDescriptor(t *testing.T) {
	declared := descriptorFindingCodes(t)
	for _, code := range []string{
		"STORAGE_ACCOUNTABILITY_NOT_DECLARED",
		"STORAGE_ACCOUNTABILITY_NOT_RECONCILED",
		"STORAGE_ACCOUNTABILITY_NOT_GOVERNED",
	} {
		if _, ok := declared[code]; !ok {
			t.Errorf("%s is emitted but not declared in .vrooli/test-genie.json", code)
		}
	}
}
