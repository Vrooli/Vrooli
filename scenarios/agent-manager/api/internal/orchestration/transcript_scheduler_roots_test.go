package orchestration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
)

// newImportOrchestrator builds a real-repository orchestrator that can parse
// codex transcripts, matching the import integration tests.
func newImportOrchestrator(t *testing.T) *orchestration.Orchestrator {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatalf("register codex runner: %v", err)
	}
	return orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1}),
		orchestration.WithInvocationReadModel(repos.InvocationReadModel),
	)
}

// A configured extra scan root makes a transcript that lives outside the
// built-in harness roots importable, and the per-root summary reports that
// contribution. A repeat sweep imports nothing new (idempotent) and the same
// transcript is reported as existing, so externally-run agents stay complete
// without duplicating runs.
func TestRunOnceImportsFromConfiguredScanRoot(t *testing.T) {
	// Isolate from the operator's real harness roots.
	t.Setenv("HOME", t.TempDir())

	extraRoot := t.TempDir()
	// A codex rollout the built-in roots would never reach.
	path := filepath.Join(extraRoot, "remote-session.jsonl")
	body := "{\"type\":\"thread.started\",\"thread_id\":\"remote-session\"}\n{\"type\":\"turn.completed\"}\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	orch := newImportOrchestrator(t)
	scheduler := orchestration.NewTranscriptImportScheduler(orch, time.Hour, extraRoot)

	first, err := scheduler.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	if first.Imported != 1 {
		t.Fatalf("first sweep imported = %d, want 1 (summary %+v)", first.Imported, first)
	}
	configured := findRootSummary(t, first, extraRoot)
	if !configured.Present || configured.Kind != "configured" || configured.Imported != 1 || configured.Scanned != 1 {
		t.Fatalf("configured root summary = %+v", configured)
	}

	// Idempotency: a repeat sweep must import nothing new and see the transcript
	// as already adopted.
	second, err := scheduler.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if second.Imported != 0 || second.Existing != 1 {
		t.Fatalf("second sweep imported=%d existing=%d, want 0/1 (summary %+v)", second.Imported, second.Existing, second)
	}
	configuredAgain := findRootSummary(t, second, extraRoot)
	if configuredAgain.Imported != 0 || configuredAgain.Existing != 1 {
		t.Fatalf("configured root summary on repeat = %+v", configuredAgain)
	}
}

// A configured scan root pointing at a built-in harness root must NOT cause a
// double import. Provenance keys on (harness, sessionID); without dedup the same
// transcript would be walked once as harness "codex" and once as harness
// "configured", producing two runs for one conversation. Pointing ScanRoots at
// ~/.codex/sessions is the most natural operator reaction to a missing chat, so
// this guards that exact foot-gun.
func TestRunOnceDedupsConfiguredRootOverlappingBuiltin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Populate the built-in codex root with one rollout.
	codexSessions := filepath.Join(home, ".codex", "sessions")
	if err := os.MkdirAll(codexSessions, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(codexSessions, "overlap-session.jsonl")
	body := "{\"type\":\"thread.started\",\"thread_id\":\"overlap-session\"}\n{\"type\":\"turn.completed\"}\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	orch := newImportOrchestrator(t)
	// Configure a scan root equal to the built-in codex sessions dir.
	scheduler := orchestration.NewTranscriptImportScheduler(orch, time.Hour, codexSessions)

	summary, err := scheduler.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	// Exactly one import: the built-in codex root. The configured duplicate is
	// deduped before the walk, so it never re-imports under harness "configured".
	if summary.Imported != 1 {
		t.Fatalf("imported = %d, want 1 (configured root equal to built-in must not double-import): %+v", summary.Imported, summary)
	}
	// The configured root must not appear as its own per-root summary entry,
	// because it collapsed into the built-in codex root.
	for _, r := range summary.Roots {
		if r.Kind == "configured" {
			t.Fatalf("configured root overlapping a built-in should be deduped, but it was swept: %+v", r)
		}
	}

	// A repeat sweep is idempotent: still only the already-adopted run, nothing
	// new imported.
	second, err := scheduler.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if second.Imported != 0 || second.Existing != 1 {
		t.Fatalf("second sweep imported=%d existing=%d, want 0/1: %+v", second.Imported, second.Existing, second)
	}
}

// Two configured roots where one is nested under the other must dedup so the
// nested subtree is not walked twice under harness "configured".
func TestRunOnceDedupsNestedConfiguredRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	parent := t.TempDir()
	child := filepath.Join(parent, "nested")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(child, "nested-session.jsonl")
	body := "{\"type\":\"thread.started\",\"thread_id\":\"nested-session\"}\n{\"type\":\"turn.completed\"}\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	orch := newImportOrchestrator(t)
	// parent first, then child (nested under parent) — child must be dropped.
	scheduler := orchestration.NewTranscriptImportScheduler(orch, time.Hour, parent, child)

	summary, err := scheduler.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if summary.Imported != 1 {
		t.Fatalf("imported = %d, want 1 (nested configured root must not double-import): %+v", summary.Imported, summary)
	}
}

// A configured root that does not exist is reported as a present=false root, so
// "why isn't my transcript here" is answerable from the sweep output rather than
// being silently folded into the aggregate counts.
func TestRunOnceReportsMissingConfiguredRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	orch := newImportOrchestrator(t)
	scheduler := orchestration.NewTranscriptImportScheduler(orch, time.Hour, missing)

	summary, err := scheduler.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	root := findRootSummary(t, summary, missing)
	if root.Present {
		t.Fatalf("missing root reported present: %+v", root)
	}
	if root.Kind != "configured" {
		t.Fatalf("missing root kind = %q, want configured", root.Kind)
	}
}

// The built-in harness roots are always reported in the per-root summary even
// when empty, so a sweep that found nothing is distinguishable from a sweep that
// never scanned the roots.
func TestRunOnceAlwaysReportsBuiltinRoots(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	orch := newImportOrchestrator(t)
	scheduler := orchestration.NewTranscriptImportScheduler(orch, time.Hour)

	summary, err := scheduler.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	kinds := map[string]bool{}
	for _, r := range summary.Roots {
		kinds[r.Kind] = true
	}
	if !kinds["codex-home"] || !kinds["claude-home"] {
		t.Fatalf("built-in roots missing from summary: %+v", summary.Roots)
	}
}

func findRootSummary(t *testing.T, summary orchestration.TranscriptImportSummary, path string) orchestration.TranscriptRootSummary {
	t.Helper()
	for _, r := range summary.Roots {
		if r.Path == path {
			return r
		}
	}
	t.Fatalf("root %q not found in summary roots %+v", path, summary.Roots)
	return orchestration.TranscriptRootSummary{}
}
