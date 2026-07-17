package opsrunner

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentops"
)

// legacyImportFixture is a realistic Phase-8 legacy-execution-import document:
// a typed header plus the verbatim pre-cutover execution-runs.json entry.
const legacyImportFixture = `{
  "kind": "agentops-legacy-execution-import",
  "schema_version": "1.0.0",
  "operation": "execution-run",
  "execution_id": "exec-legacy-54af4aefa1199d31",
  "workflow_instance_id": "wf-plan-execution-abc",
  "imported_at": "2026-07-15T00:09:26Z",
  "legacy": {
    "execution_id": "54af4aefa1199d31",
    "backlog_kind": "chore",
    "backlog_name": "vrooli-emulator-documentation",
    "status": "canceled",
    "mode": "manual",
    "queued_at": "2026-07-15T00:09:26Z",
    "started_by": "swarm-manager",
    "operation": "generator"
  }
}`

func writeLegacyImport(t *testing.T, root string) (kind agentops.TargetKind, id, execID string) {
	t.Helper()
	kind, id = agentops.TargetKind("plan-execution"), "abc"
	execID = "exec-legacy-54af4aefa1199d31"
	loc := memLocator{root: root}
	dir, err := loc.AgentOpsDir(kind, id)
	if err != nil {
		t.Fatalf("agentops dir: %v", err)
	}
	execDir := filepath.Join(dir, executionsSubdir)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(execDir, execID+".json"), []byte(legacyImportFixture), 0o644); err != nil {
		t.Fatalf("write import doc: %v", err)
	}
	return kind, id, execID
}

// TestExecutionStoreDecodesLegacyImport proves Load and List recognize a
// Phase-8 legacy-execution-import document and surface ONLY the fields it
// really carries: operation + workflow id from the header, recorded-at from
// the deterministic imported_at, outcome from the legacy entry's own terminal
// status — with LegacyImport set so readers can label the row instead of
// rendering a hollow snapshot with blank provenance.
// [REQ:REQ-P0-011-MIGRATION-INTEGRITY]
func TestExecutionStoreDecodesLegacyImport(t *testing.T) {
	root := t.TempDir()
	kind, id, execID := writeLegacyImport(t, root)
	store := NewExecutionStore(memLocator{root: root})

	snap, found, err := store.Load(kind, id, execID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !found {
		t.Fatalf("import snapshot not found")
	}
	if snap.LegacyImport == nil {
		t.Fatalf("LegacyImport must be set for an import document")
	}
	if snap.Provenance.Operation != "execution-run" {
		t.Fatalf("operation = %q, want execution-run", snap.Provenance.Operation)
	}
	if snap.Provenance.WorkflowInstanceID != "wf-plan-execution-abc" {
		t.Fatalf("workflow id = %q", snap.Provenance.WorkflowInstanceID)
	}
	if snap.RecordedAt != "2026-07-15T00:09:26Z" {
		t.Fatalf("recorded_at = %q, want the deterministic imported_at", snap.RecordedAt)
	}
	if snap.Outcome != "canceled" {
		t.Fatalf("outcome = %q, want the legacy entry's own status", snap.Outcome)
	}
	// No provenance may be fabricated for an import.
	if snap.Provenance.Mode != "" || snap.Provenance.CompiledModeDigest != "" || snap.Provenance.PromptCatalogDigest != "" {
		t.Fatalf("import decoded with fabricated provenance: %+v", snap.Provenance)
	}
	if len(snap.LegacyImport.Legacy) == 0 {
		t.Fatalf("verbatim legacy entry must be carried")
	}

	stored, err := store.List(kind, id)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(stored) != 1 || stored[0].ID != execID {
		t.Fatalf("List = %+v", stored)
	}
	if stored[0].Snapshot.LegacyImport == nil {
		t.Fatalf("List must decode the import kind, not a hollow snapshot")
	}
}

// TestExecutionStoreReproduceRefusesLegacyImport proves reproducibility is
// honestly unavailable for imports (digests never existed) with a typed error,
// never a fabricated pass.
// [REQ:REQ-P0-011-MIGRATION-INTEGRITY]
func TestExecutionStoreReproduceRefusesLegacyImport(t *testing.T) {
	root := t.TempDir()
	kind, id, execID := writeLegacyImport(t, root)
	store := NewExecutionStore(memLocator{root: root})

	if _, err := store.Reproduce(kind, id, execID); !errors.Is(err, ErrLegacyImportNotReproducible) {
		t.Fatalf("Reproduce error = %v, want ErrLegacyImportNotReproducible", err)
	}
}
