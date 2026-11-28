package experience

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReaderJoinsContractAndEvidence(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "DrawerShell", "versions", "1.0.0")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := `{"component":{"id":"drawer-shell","title":"DrawerShell","purpose":"Named dialog surface."},"states":[{"id":"full-open","example":"full-open","description":"Open."}],"claims":[{"id":"dialog-present","type":"element-present","statement":"A dialog is present.","tier":"machine","states":["full-open"]}]}`
	if err := os.WriteFile(filepath.Join(path, "experience-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &reader{repoRoot: root, listEvidence: func(_ context.Context, componentID string) ([]Evidence, error) {
		if componentID != "drawer-shell" {
			t.Fatalf("component id = %q", componentID)
		}
		return []Evidence{{ClaimID: "dialog-present", Verdict: "passed", StateID: "full-open", ExampleName: "full-open"}}, nil
	}}
	got, err := r.Get(context.Background(), Component{ID: "cmp-1", LibraryID: "react-component-library:DrawerShell", Slug: "DrawerShell", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ContractID != "drawer-shell" || got.Title != "DrawerShell" || len(got.States) != 1 || len(got.Claims) != 1 || len(got.Evidence) != 1 || got.EvidenceStatus != "covered" {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestCoverageStatusUsesDeclaredStatesNotTransportReachability(t *testing.T) {
	states := []State{{ID: "default"}, {ID: "ready"}}
	cases := []struct {
		name     string
		evidence []Evidence
		want     string
	}{
		{name: "no evidence", want: "unmeasured"},
		{name: "skipped capture", evidence: []Evidence{{StateID: "default", Verdict: "skipped"}}, want: "unmeasured"},
		{name: "one passing state", evidence: []Evidence{{StateID: "default", Verdict: "passed"}}, want: "partial"},
		{name: "all passing states", evidence: []Evidence{{StateID: "default", Verdict: "passed"}, {StateID: "ready", Verdict: "passed"}}, want: "covered"},
		{name: "renamed state is ignored", evidence: []Evidence{{StateID: "old", Verdict: "passed"}}, want: "unmeasured"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := coverageStatus(states, tc.evidence); got != tc.want {
				t.Fatalf("coverageStatus() = %q, want %q", got, tc.want)
			}
		})
	}
	if got := coverageStatus(nil, nil); got != "unmeasured" {
		t.Fatalf("contract without states = %q, want unmeasured", got)
	}
}

func TestReaderUsesCanonicalScenarioLevelContract(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "scenarios", "react-component-library", "experience", "components")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "drawer-shell.json"), []byte(`{"component":{"id":"drawer-shell"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &reader{repoRoot: root, listEvidence: func(context.Context, string) ([]Evidence, error) { return nil, nil }}
	got, err := r.Get(context.Background(), Component{ID: "cmp-1", Slug: "DrawerShell", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ContractID != "drawer-shell" || got.Title != "" {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestReaderReportsNoContractWithoutCallingEvidence(t *testing.T) {
	called := false
	r := &reader{repoRoot: t.TempDir(), listEvidence: func(context.Context, string) ([]Evidence, error) { called = true; return nil, nil }}
	got, err := r.Get(context.Background(), Component{ID: "cmp-1", LibraryID: "react-component-library:Unknown", Slug: "Unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if got.EvidenceStatus != "not-configured" || called {
		t.Fatalf("snapshot = %+v called=%t", got, called)
	}
}

func TestNormalizeEvidenceKeepsOneLatestVerdictPerDeclaredClaimAndState(t *testing.T) {
	claims := []Claim{{ID: "card-grid-present"}, {ID: "layout-stable"}}
	raw := []Evidence{
		{ClaimID: "card-grid-present", StateID: "default", Verdict: "failed", CheckedAt: "2026-11-20T10:00:00Z"},
		{ClaimID: "card-grid-present", StateID: "default", Verdict: "passed", CheckedAt: "2026-11-22T10:00:00Z"},
		{ClaimID: "card-grid-present", StateID: "default", Verdict: "skipped", CheckedAt: "2026-11-22T10:00:00Z"},
		{ClaimID: "card-grid-present", StateID: "default", Verdict: "passed", CheckedAt: "2026-11-22T10:00:00Z"},
		{ClaimID: "layout-stable", StateID: "default", Verdict: "skipped", CheckedAt: "2026-11-21T10:00:00Z"},
		{ClaimID: "floor-no-component-overflow", StateID: "default", Verdict: "passed", CheckedAt: "2026-11-22T10:00:00Z"},
	}
	got := normalizeEvidence(raw, claims)

	if len(got) != 2 {
		t.Fatalf("evidence = %+v, want two rows", got)
	}
	if got[0].ClaimID != "card-grid-present" || got[0].Verdict != "pass" {
		t.Fatalf("first row = %+v, want the latest decisive pass", got[0])
	}
	if got[1].ClaimID != "layout-stable" || got[1].Verdict != "unmeasured" {
		t.Fatalf("second row = %+v, want skipped normalized to unmeasured", got[1])
	}
	for _, row := range got {
		if row.ClaimID == "floor-no-component-overflow" {
			t.Fatalf("undeclared floor claim leaked into the contract evidence: %+v", row)
		}
	}
}

func TestNormalizeEvidenceSortsNewestCaptureFirst(t *testing.T) {
	got := normalizeEvidence([]Evidence{
		{ClaimID: "a", StateID: "default", Verdict: "passed", CheckedAt: "2026-11-20T00:00:00Z"},
		{ClaimID: "b", StateID: "default", Verdict: "failed", CheckedAt: "2026-11-23T00:00:00Z"},
	}, nil)
	if len(got) != 2 || got[0].ClaimID != "b" {
		t.Fatalf("evidence = %+v, want newest first", got)
	}
}

func TestVerdictMapsEveryRecordedVocabulary(t *testing.T) {
	cases := map[string]string{
		"passed": "pass", "pass": "pass", "ok": "pass",
		"failed": "fail", "fail": "fail", "error": "fail",
		"warned": "warn", "warning": "warn",
		"skipped": "unmeasured", "unknown": "unmeasured", "": "unmeasured",
	}
	for raw, want := range cases {
		if got := Verdict(raw); got != want {
			t.Errorf("Verdict(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeEvidenceDropsStatesTheContractNoLongerDeclares(t *testing.T) {
	claims := []Claim{{ID: "card-grid-present", States: []string{"card-grid-raised"}}}
	raw := []Evidence{
		{ClaimID: "card-grid-present", StateID: "card-grid", Verdict: "passed", CheckedAt: "2026-11-22T10:00:00Z"},
		{ClaimID: "card-grid-present", StateID: "card-grid-raised", Verdict: "skipped", CheckedAt: "2026-11-22T10:00:00Z"},
	}
	got := normalizeEvidence(raw, claims)
	if len(got) != 1 || got[0].StateID != "card-grid-raised" {
		t.Fatalf("evidence = %+v, want only the declared state", got)
	}
}
