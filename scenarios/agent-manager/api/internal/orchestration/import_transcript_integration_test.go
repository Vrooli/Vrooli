package orchestration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/structuredresult"
)

type labelExtractor struct {
	label string
	calls int
}

func (e *labelExtractor) Extract(context.Context, structuredresult.ExtractRequest) (structuredresult.ExtractResponse, error) {
	e.calls++
	value, _ := json.Marshal(e.label)
	return structuredresult.ExtractResponse{Candidate: value, Provider: "test"}, nil
}

// TestImportTranscriptPersistsAgainstSQLite exercises the real repositories,
// not a fake RunRepository. In particular, it proves imported runs satisfy the
// non-null runs.task_id foreign key through the durable synthetic task.
func TestImportTranscriptPersistsAgainstSQLite(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatalf("register codex runner: %v", err)
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1}),
		orchestration.WithInvocationReadModel(repos.InvocationReadModel),
	)
	path := filepath.Join(t.TempDir(), "codex.jsonl")
	body := "{\"type\":\"thread.started\",\"thread_id\":\"import-test\"}\n{\"type\":\"turn.completed\"}\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	request := orchestration.ImportTranscriptRequest{Path: path, RunnerType: domain.RunnerTypeCodex, Label: "sqlite", SourceHarness: "resource:codex/sessions", SourceSessionID: "import-test"}
	run, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("import transcript against SQLite: %v", err)
	}
	if run.TaskID.String() == "00000000-0000-0000-0000-000000000000" || run.ExecutionMode != domain.ExecutionModeImported {
		t.Fatalf("imported run = %+v", run)
	}
	task, err := svc.GetTask(context.Background(), run.TaskID)
	if err != nil || task.Title != "Imported external transcripts" {
		t.Fatalf("synthetic task = %+v, %v", task, err)
	}
	persisted, err := svc.GetRun(context.Background(), run.ID)
	if err != nil || persisted == nil || persisted.TaskID != run.TaskID {
		t.Fatalf("persisted run = %+v, %v", persisted, err)
	}
	if persisted.ImportSourceHarness != request.SourceHarness || persisted.ImportSourceSessionID != request.SourceSessionID || persisted.ImportedAt == nil {
		t.Fatalf("import provenance = %+v", persisted)
	}
	again, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("repeat import: %v", err)
	}
	if again.ID != run.ID {
		t.Fatalf("repeat import created %s, want existing %s", again.ID, run.ID)
	}
}

func TestImportTranscriptLabelsUseHarnessThenGeneratedPrecedence(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewClaudeForTest(), nil, nil)); err != nil {
		t.Fatalf("register claude runner: %v", err)
	}
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatalf("register codex runner: %v", err)
	}
	extractor := &labelExtractor{label: "Generated Codex Session"}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore), orchestration.WithRunners(registry),
		orchestration.WithLabelGenerator(extractor), orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1}),
		orchestration.WithInvocationReadModel(repos.InvocationReadModel),
	)

	claudePath := filepath.Join(t.TempDir(), "claude.jsonl")
	if err := os.WriteFile(claudePath, []byte("{\"type\":\"ai-title\",\"aiTitle\":\"Known Claude title\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	claudeRun, err := svc.ImportTranscript(context.Background(), orchestration.ImportTranscriptRequest{Path: claudePath, RunnerType: domain.RunnerTypeClaudeCode, SourceHarness: "claude", SourceSessionID: "claude-title"})
	if err != nil {
		t.Fatalf("import claude transcript: %v", err)
	}
	if claudeRun.Label != "Known Claude title" || claudeRun.LabelSource != domain.RunLabelSourceHarness {
		t.Fatalf("claude label=(%q,%q)", claudeRun.Label, claudeRun.LabelSource)
	}
	persistedClaude, err := svc.GetRun(context.Background(), claudeRun.ID)
	if err != nil {
		t.Fatalf("get claude run: %v", err)
	}
	if persistedClaude.Label != claudeRun.Label || persistedClaude.LabelSource != claudeRun.LabelSource {
		t.Fatalf("persisted claude label=(%q,%q)", persistedClaude.Label, persistedClaude.LabelSource)
	}

	codexPath := filepath.Join(t.TempDir(), "codex.jsonl")
	if err := os.WriteFile(codexPath, []byte("{\"type\":\"thread.started\",\"thread_id\":\"codex-title\"}\n{\"type\":\"turn.completed\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	codexRun, err := svc.ImportTranscript(context.Background(), orchestration.ImportTranscriptRequest{Path: codexPath, RunnerType: domain.RunnerTypeCodex, SourceHarness: "codex", SourceSessionID: "codex-title"})
	if err != nil {
		t.Fatalf("import codex transcript: %v", err)
	}
	if codexRun.Label != "Generated Codex Session" || codexRun.LabelSource != domain.RunLabelSourceGenerated {
		t.Fatalf("codex label=(%q,%q)", codexRun.Label, codexRun.LabelSource)
	}
	if extractor.calls != 1 {
		t.Fatalf("label generator calls=%d want 1", extractor.calls)
	}
	runs, err := svc.ListRuns(context.Background(), orchestration.RunListOptions{ListOptions: orchestration.ListOptions{Limit: 10}})
	if err != nil {
		t.Fatalf("list labeled runs: %v", err)
	}
	seenLabels := map[string]bool{}
	for _, item := range runs {
		seenLabels[item.Label] = true
	}
	if !seenLabels["Known Claude title"] || !seenLabels["Generated Codex Session"] {
		t.Fatalf("list labels=%v", seenLabels)
	}
}
