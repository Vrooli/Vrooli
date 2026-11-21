package signals

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeCollector struct {
	name string
	fn   func(snap *Snapshot) error
}

func (f fakeCollector) Name() string                 { return f.name }
func (f fakeCollector) Collect(snap *Snapshot) error { return f.fn(snap) }

func TestCollectPanicYieldsDegradationNotCrash(t *testing.T) {
	svc := newService(
		fakeCollector{name: "boom", fn: func(*Snapshot) error { panic("malformed input") }},
		fakeCollector{name: "ok", fn: func(snap *Snapshot) error { snap.Category = "ai_tools"; return nil }},
	)

	snap := svc.Collect("demo", t.TempDir())

	if len(snap.Degradations) != 1 {
		t.Fatalf("degradations = %+v, want exactly one", snap.Degradations)
	}
	d := snap.Degradations[0]
	if d.Collector != "boom" || d.State != "failed" || !strings.Contains(d.Reason, "malformed input") {
		t.Fatalf("degradation = %+v, want failed boom with panic reason", d)
	}
	if snap.Category != "ai_tools" {
		t.Fatal("collectors after a panicking one must still run")
	}
}

func TestCollectBreakerOpensAfterRepeatedFailures(t *testing.T) {
	svc := newService(fakeCollector{
		name: "flaky",
		fn:   func(*Snapshot) error { return errors.New("boom") },
	})

	root := t.TempDir()
	for i := 0; i < failureThreshold; i++ {
		snap := svc.Collect("demo", root)
		if got := snap.Degradations[0].State; got != "failed" {
			t.Fatalf("collection %d degradation state = %q, want failed", i+1, got)
		}
	}

	snap := svc.Collect("demo", root)
	if got := snap.Degradations[0].State; got != "open" {
		t.Fatalf("degradation state after breaker trip = %q, want open", got)
	}
}

func TestCollectErrorNeverFailsSnapshot(t *testing.T) {
	svc := newService(
		fakeCollector{name: "bad", fn: func(*Snapshot) error { return errors.New("nope") }},
		fakeCollector{name: "good", fn: func(snap *Snapshot) error {
			snap.UI = UISignals{Collected: true, FileCount: 7}
			return nil
		}},
	)

	snap := svc.Collect("demo", t.TempDir())
	if !snap.UI.Collected || snap.UI.FileCount != 7 {
		t.Fatalf("UI section = %+v, want collected despite earlier failure", snap.UI)
	}
}

// End-to-end over a realistic fixture tree with the standard collector set.
func TestServiceCollectFullFixture(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".vrooli/service.json", `{"category":"developer_tools"}`)
	writeFile(t, root, "requirements/index.json", `{"imports":["01-core/module.json"]}`)
	writeFile(t, root, "requirements/01-core/module.json", `{
		"module_id":"MOD-P0-001","prd_ref":"OT-P0-001","priority":"P0",
		"requirements":[
			{"id":"REQ-P0-001","prd_ref":"OT-P0-001","status":"passed",
			 "validation":[{"type":"test","phase":"unit"}]},
			{"id":"REQ-P0-002","prd_ref":"OT-P0-001","status":"draft"}
		]}`)
	writeFile(t, root, "coverage/runs/20260610-120000-aaaa/phase-results/unit.json",
		`{"phase":"unit","status":"passed","updated_at":"2026-06-10T12:00:00Z","findings":[]}`)
	writeFile(t, root, "ui/src/App.tsx", "export const App = () => fetch('/api/v1/items');\n")

	snap := newService(
		serviceCollector{},
		requirementsCollector{syncSource: fileSyncMetadataSource{}},
		phasesCollector{source: filesystemPhaseSource{root: root}},
		uiCollector{},
	).Collect("demo", root)

	if len(snap.Degradations) != 0 {
		t.Fatalf("degradations = %+v, want none", snap.Degradations)
	}
	if snap.Category != "developer_tools" {
		t.Fatalf("category = %q", snap.Category)
	}
	if !snap.Requirements.Collected || snap.Requirements.Total != 2 || snap.Requirements.Passing != 1 {
		t.Fatalf("requirements = %+v", snap.Requirements)
	}
	if !snap.Phases.Collected || snap.Phases.Phases["unit"].Status != "passed" {
		t.Fatalf("phases = %+v", snap.Phases)
	}
	if !snap.UI.Collected || snap.UI.APIEndpoints != 1 {
		t.Fatalf("ui = %+v", snap.UI)
	}
}

// writeFile creates path (and parents) under root with content.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
