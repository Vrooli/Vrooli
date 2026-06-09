package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/measures-go/manifestscan"
)

// writeFile writes p (relative to root) with contents, creating parent dirs.
func writeFile(t *testing.T, root, p, contents string) {
	t.Helper()
	full := filepath.Join(root, p)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFilesystemSubstrateDetector(t *testing.T) {
	root := t.TempDir()
	// A genuine countable entity: notes with created_at.
	writeFile(t, root, "scenarios/demo/api/internal/notes/schema.sql", `
CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL
);`)
	// A singleton state row (CHECK id=1) — not countable even with a *_at column.
	writeFile(t, root, "scenarios/demo/api/internal/state/schema.sql", `
CREATE TABLE IF NOT EXISTS reconcile_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    created_at TEXT,
    last_reconcile_at TEXT
);`)
	// A mutation-only table (no created_at) — not detected.
	writeFile(t, root, "scenarios/demo/api/internal/cfg/schema.sql", `
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    updated_at TEXT
);`)
	// A CQRS substrate table (events) — denylisted even with created_at.
	writeFile(t, root, "scenarios/demo/api/internal/eventlog/repo.go", "package eventlog\nconst s = `CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY, created_at TEXT, payload TEXT);`")
	// A Go-embedded countable table with a nested CHECK/ DEFAULT to exercise the
	// balanced-paren scan.
	writeFile(t, root, "scenarios/demo/api/internal/orders/repo.go", "package orders\nconst s = `CREATE TABLE orders (id TEXT PRIMARY KEY, total INTEGER DEFAULT (0), status TEXT CHECK (status IN ('a','b')), created_at TEXT NOT NULL);`")

	got, err := FilesystemSubstrateDetector{RepoRoot: root}.DetectedEntities("demo")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range got {
		names[e.Name] = true
	}
	if !names["notes"] {
		t.Errorf("expected notes detected, got %+v", got)
	}
	if !names["orders"] {
		t.Errorf("expected orders detected (balanced-paren body), got %+v", got)
	}
	if names["reconcile_state"] {
		t.Errorf("singleton state table must not be detected, got %+v", got)
	}
	if names["settings"] {
		t.Errorf("mutation-only table (no created_at) must not be detected, got %+v", got)
	}
	if names["events"] {
		t.Errorf("CQRS substrate table must be denylisted, got %+v", got)
	}
}

func TestFilesystemSubstrateDetector_NoApiDir(t *testing.T) {
	got, err := FilesystemSubstrateDetector{RepoRoot: t.TempDir()}.DetectedEntities("nope")
	if err != nil {
		t.Fatalf("missing api dir must not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want no detections, got %+v", got)
	}
}

func TestClassify_UndeclaredSubstrateIsWarning(t *testing.T) {
	// Fallback scenario, nothing declared, but a created_at table exists.
	rep := Classify(Inputs{
		Scenario: "demo",
		Mode:     ModeFallback,
		Detected: []DetectedEntity{{Name: "notes", Evidence: "table `notes` (created_at) in scenarios/demo/api/internal/notes/schema.sql"}},
	})
	if !rep.Passed {
		t.Fatalf("undeclared substrate is WARNING, must not fail; findings=%+v", rep.Findings)
	}
	f := findingBy(rep, "measures.undeclared-substrate")
	if f == nil || f.Severity != SeverityWarning {
		t.Fatalf("want undeclared-substrate WARNING, findings=%+v", rep.Findings)
	}
}

func TestClassify_SubstrateWaivedIsSilent(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "demo",
		Mode:     ModeFallback,
		Omitted:  []manifestscan.Omission{{Domain: "notes", Reason: "ephemeral scratch"}},
		Detected: []DetectedEntity{{Name: "notes", Evidence: "table `notes` (created_at)"}},
	})
	if f := findingBy(rep, "measures.undeclared-substrate"); f != nil {
		t.Fatalf("a waived entity must be silent, got %+v", f)
	}
}

func TestClassify_SubstrateCoveredIsSilent(t *testing.T) {
	// Conformant, covered domain. The plural table maps to the singular domain.
	rep := Classify(Inputs{
		Scenario: "demo",
		Mode:     ModeConformant,
		Domains:  []DerivedDomain{{Name: "note", Stateful: true}},
		Measures: []HarvestedMeasure{{Name: "note.count", Domain: "note", Tier: manifestscan.TierFull}},
		Detected: []DetectedEntity{{Name: "notes", Evidence: "table `notes` (created_at)"}},
	})
	if !rep.Passed {
		t.Fatalf("covered domain must pass; findings=%+v", rep.Findings)
	}
	if f := findingBy(rep, "measures.undeclared-substrate"); f != nil {
		t.Fatalf("a covered entity (plural table -> singular domain) must be silent, got %+v", f)
	}
}

func TestClassify_SubstrateMatchesUncoveredDomainNoDoubleFlag(t *testing.T) {
	// The detected entity maps to a known (uncovered) domain: the uncovered-domain
	// ERROR already covers it — no additional substrate WARNING.
	rep := Classify(Inputs{
		Scenario: "demo",
		Mode:     ModeConformant,
		Domains:  []DerivedDomain{{Name: "notes", Stateful: true}},
		Detected: []DetectedEntity{{Name: "notes", Evidence: "table `notes` (created_at)"}},
	})
	if findingBy(rep, "measures.uncovered-domain") == nil {
		t.Fatalf("want uncovered-domain ERROR, findings=%+v", rep.Findings)
	}
	if f := findingBy(rep, "measures.undeclared-substrate"); f != nil {
		t.Fatalf("known domain must not also raise a substrate WARNING, got %+v", f)
	}
}
