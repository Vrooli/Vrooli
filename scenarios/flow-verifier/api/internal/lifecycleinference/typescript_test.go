package lifecycleinference

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTypeScriptLifecycleInferenceKeepsEvidenceAndRejectsWeakInputs(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "sample")
	for _, dir := range []string{"api/domain", "ui/src", "cli"} {
		if err := os.MkdirAll(filepath.Join(scenario, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	source := `type JobState = "queued" | "running" | "failed";
type PlainStatus = string;
interface Job {
  status: JobState;
  updatedAt: Date;
  retryable: boolean;
  lastEvent: string;
}

interface Weak { status: PlainStatus; }
`
	if err := os.WriteFile(filepath.Join(scenario, "api/domain/job.ts"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenario, "ui/src/broken.tsx"), []byte("interface Broken { status: State;"), 0o644); err != nil {
		t.Fatal(err)
	}
	js := `/** @typedef {'open'|'closed'} PanelState */
/** @typedef {Object} Panel
 * @property {PanelState} state
 * @property {Date} changedAt
 */
`
	if err := os.WriteFile(filepath.Join(scenario, "ui/src/panel.js"), []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, files, errors := analyzeTypeScriptScenario(scenario, "sample")
	if files != 3 || errors != 0 {
		t.Fatalf("files=%d errors=%d", files, errors)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected only finite lifecycle vocabulary candidates, got %#v", candidates)
	}
	var ts, jsCandidate *Candidate
	for i := range candidates {
		if candidates[i].FieldName == "status" {
			ts = &candidates[i]
		}
		if candidates[i].FieldName == "state" {
			jsCandidate = &candidates[i]
		}
	}
	if ts == nil || len(ts.States) != 3 || len(ts.TimeFields) == 0 || len(ts.RelatedFlags) == 0 {
		t.Fatalf("expected typed state, timestamp, retry, and event context, got %#v", candidates)
	}
	if jsCandidate == nil || len(jsCandidate.States) != 2 {
		t.Fatalf("expected JSDoc finite state candidate, got %#v", candidates)
	}
	if ts.Persistence.Class != ClassUnknown || !ts.Persistence.ScanComplete {
		t.Fatalf("TypeScript must remain persistence-unknown: %#v", ts.Persistence)
	}
}
