package signals

import (
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func collectPhases(t *testing.T, root string) (PhaseSignals, error) {
	t.Helper()
	snap := Snapshot{Root: root}
	err := phasesCollector{}.Collect(&snap)
	return snap.Phases, err
}

func TestPhasesMissingCoverageNotCollected(t *testing.T) {
	sig, err := collectPhases(t, t.TempDir())
	if err != nil {
		t.Fatalf("missing coverage/ must not error, got %v", err)
	}
	if sig.Collected {
		t.Fatal("missing coverage/ must report Collected=false")
	}
}

func TestPhasesNewestWins(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		wantStatus string
	}{
		{
			name: "updated_at wins even against a later run dir",
			files: map[string]string{
				"coverage/runs/20260601-000000-aaaa/phase-results/unit.json": `{"phase":"unit","status":"passed","updated_at":"2026-06-10T12:00:00Z"}`,
				"coverage/runs/20260609-000000-bbbb/phase-results/unit.json": `{"phase":"unit","status":"failed","updated_at":"2026-06-09T12:00:00Z"}`,
			},
			wantStatus: "passed",
		},
		{
			name: "run-dir name ordering when updated_at missing",
			files: map[string]string{
				"coverage/runs/20260601-000000-aaaa/phase-results/unit.json": `{"phase":"unit","status":"failed"}`,
				"coverage/runs/20260609-000000-bbbb/phase-results/unit.json": `{"phase":"unit","status":"passed"}`,
			},
			wantStatus: "passed",
		},
		{
			name: "run-dir name ordering when updated_at unparseable",
			files: map[string]string{
				"coverage/runs/20260601-000000-aaaa/phase-results/unit.json": `{"phase":"unit","status":"failed","updated_at":"yesterday"}`,
				"coverage/runs/20260609-000000-bbbb/phase-results/unit.json": `{"phase":"unit","status":"passed","updated_at":"not-a-time"}`,
			},
			wantStatus: "passed",
		},
		{
			name: "legacy file loses ties to run-dir files",
			files: map[string]string{
				"coverage/phase-results/unit.json":                           `{"phase":"unit","status":"failed","updated_at":"2026-06-10T12:00:00Z"}`,
				"coverage/runs/20260601-000000-aaaa/phase-results/unit.json": `{"phase":"unit","status":"passed","updated_at":"2026-06-10T12:00:00Z"}`,
			},
			wantStatus: "passed",
		},
		{
			name: "legacy file wins on strictly newer updated_at",
			files: map[string]string{
				"coverage/phase-results/unit.json":                           `{"phase":"unit","status":"passed","updated_at":"2026-06-11T12:00:00Z"}`,
				"coverage/runs/20260601-000000-aaaa/phase-results/unit.json": `{"phase":"unit","status":"failed","updated_at":"2026-06-10T12:00:00Z"}`,
			},
			wantStatus: "passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tt.files {
				writeFile(t, root, rel, content)
			}

			sig, err := collectPhases(t, root)
			if err != nil {
				t.Fatal(err)
			}
			if !sig.Collected {
				t.Fatal("want Collected=true")
			}
			got, ok := sig.Phases["unit"]
			if !ok || got.Status != tt.wantStatus {
				t.Fatalf("unit = %+v (present=%v), want status %q", got, ok, tt.wantStatus)
			}
		})
	}
}

func TestPhasesLegacyOnlyFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "coverage/phase-results/smoke.json",
		`{"phase":"smoke","status":"passed","updated_at":"2026-06-01T00:00:00Z"}`)

	sig, err := collectPhases(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if !sig.Collected || sig.Phases["smoke"].Status != "passed" {
		t.Fatalf("phases = %+v", sig)
	}
}

func TestPhasesFindingsDecoding(t *testing.T) {
	root := t.TempDir()
	// Enums persist as proto integer values: severity 3 = ERROR,
	// source 5 = STANDARDS.
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/standards.json",
		`{"phase":"standards","status":"failed","updated_at":"2026-06-10T00:00:00Z",
		  "findings":[{"severity":3,"source":5,"message":"x","code":"rule/x"}]}`)
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/unit.json",
		`{"phase":"unit","status":"passed","updated_at":"2026-06-10T00:00:00Z","findings":[]}`)
	// Older writer: no findings key at all.
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/smoke.json",
		`{"phase":"smoke","status":"passed","updated_at":"2026-06-10T00:00:00Z"}`)
	// Undecodable findings: status kept, findings dropped.
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/lint.json",
		`{"phase":"lint","status":"failed","updated_at":"2026-06-10T00:00:00Z",
		  "findings":[{"severity":"high"}]}`)
	// Explicit null counts as absent.
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/docs.json",
		`{"phase":"docs","status":"passed","updated_at":"2026-06-10T00:00:00Z","findings":null}`)

	sig, err := collectPhases(t, root)
	if err != nil {
		t.Fatal(err)
	}

	standards := sig.Phases["standards"]
	if !standards.HasFindings || len(standards.Findings) != 1 {
		t.Fatalf("standards = %+v, want one finding", standards)
	}
	f := standards.Findings[0]
	if f.Severity != architecturev1.FindingSeverity(3) || f.Source != architecturev1.FindingSource(5) || f.Message != "x" {
		t.Fatalf("finding = %+v, want severity 3 / source 5 / message x", f)
	}

	unit := sig.Phases["unit"]
	if !unit.HasFindings || len(unit.Findings) != 0 {
		t.Fatalf("unit = %+v, want HasFindings with zero findings", unit)
	}

	smoke := sig.Phases["smoke"]
	if smoke.HasFindings || smoke.Findings != nil {
		t.Fatalf("smoke = %+v, want no findings signal for older shape", smoke)
	}

	lint := sig.Phases["lint"]
	if lint.Status != "failed" || lint.HasFindings || lint.Findings != nil {
		t.Fatalf("lint = %+v, want status kept and findings dropped", lint)
	}

	docs := sig.Phases["docs"]
	if docs.HasFindings {
		t.Fatalf("docs = %+v, want findings:null treated as absent", docs)
	}
}

func TestPhasesMalformedFileSkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/broken.json", `{`)
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/unit.json",
		`{"phase":"unit","status":"passed","updated_at":"2026-06-10T00:00:00Z"}`)
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/notes.txt", "not json")

	sig, err := collectPhases(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig.Phases) != 1 || sig.Phases["unit"].Status != "passed" {
		t.Fatalf("phases = %+v, want only unit", sig.Phases)
	}
}

func TestPhasesNameFallsBackToFilename(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "coverage/runs/20260610-000000-aaaa/phase-results/integration.json",
		`{"status":"skipped","updated_at":"2026-06-10T00:00:00Z"}`)

	sig, err := collectPhases(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if sig.Phases["integration"].Status != "skipped" {
		t.Fatalf("phases = %+v, want filename-derived phase name", sig.Phases)
	}
}

func TestPhasesEmptyRunsTreeStillCollected(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "coverage/runs/.keep", "")

	sig, err := collectPhases(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if !sig.Collected || len(sig.Phases) != 0 {
		t.Fatalf("phases = %+v, want collected with empty map", sig)
	}
}
