package lifecycleinference_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"flow-verifier/internal/lifecycleinference"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeContextStopsWhenCallerIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := lifecycleinference.AnalyzeContext(ctx, t.TempDir(), lifecycleinference.Options{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestAnalyzeRanksLifecycleEvidenceAndMatchesHuntLeads(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/alpha/.vrooli/service.json", `{"service":{"displayName":"Alpha"}}`)
	write("scenarios/alpha/api/internal/workflow/workflow.go", `package workflow

import "time"

type Stage string
const (
  StageNew Stage = "new"
  StageDone Stage = "done"
)

type OrderWorkflow struct {
  Status Stage
  Complete bool
  StartedAt time.Time
}

func (w *OrderWorkflow) Finish() { w.Status = StageDone }
func (w OrderWorkflow) IsDone() bool { return w.Status == StageDone }
`)
	write("scenarios/alpha/api/internal/workflow/workflow_test.go", `package workflow
func TestWorkflowStatus(t *testing.T) { _ = OrderWorkflow{Status: StageNew} }
`)
	write("scenarios/alpha/api/internal/workflow/flow/flow.json", `{"flowId":"alpha.order-workflow","runtime":{"go":{"statusType":"Stage"}},"states":[{"id":"new"},{"id":"done"}]}`)
	huntPath := filepath.Join(root, "hunt.json")
	require.NoError(t, os.WriteFile(huntPath, []byte(`{"hits":[{"entry_id":"record-1","text":"Ecosystem Hunt Round #7: Alpha Order workflow status lifecycle"}]}`), 0o644))

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndex: huntPath})
	require.NoError(t, err)
	require.Equal(t, 1, got.ScenarioCount)
	require.Equal(t, 1, got.CandidateCount)
	require.Empty(t, got.Candidates, "UNKNOWN or prior-hunt candidates are never in the ranked surface")
	require.Len(t, got.UnrankedCandidates, 1)
	require.Equal(t, "unknown_hunt_coverage", got.UnrankedCandidates[0].PreselectionStatus)
	require.Len(t, got.Scenarios, 1)
	require.Equal(t, 1, got.Scenarios[0].ExplicitFlows)
	require.Len(t, allInventoryCandidates(got), 1)
	if len(allInventoryCandidates(got)) == 1 {
		candidate := allInventoryCandidates(got)[0]
		require.Equal(t, "OrderWorkflow", candidate.TypeName)
		require.Equal(t, "Status", candidate.FieldName)
		require.ElementsMatch(t, []string{"new", "done"}, []string{candidate.States[0].Value, candidate.States[1].Value})
		require.Len(t, candidate.RelatedFlags, 1)
		require.Len(t, candidate.TimeFields, 1)
		require.Len(t, candidate.WriteSites, 1)
		require.Len(t, candidate.GuardSites, 1)
		require.Len(t, candidate.TestFiles, 1)
		require.Len(t, candidate.HuntRecords, 1)
		require.Equal(t, "review", candidate.HuntRecords[0].Status)
		require.True(t, candidate.ExplicitlyModeled)
		require.Len(t, candidate.ExplicitFlows, 1)
		require.True(t, candidate.ExplicitFlows[0].ExactLifecycleMatch)
		require.Equal(t, "alpha.order-workflow", candidate.ExplicitFlows[0].FlowID)
		require.Equal(t, "strong-candidate", candidate.Confidence)
		require.Greater(t, candidate.BurdenScore, 1)
		require.NotContains(t, candidate.Declaration.Path, root, "evidence paths are stable relative paths")
	}
}

func TestAnalyzeReportsEveryScenarioAndParseFailures(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/empty/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/broken/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/broken/api/broken.go", "package broken\nfunc (\n")
	write("scenarios/broken/api/internal/flow/flow.json", "{invalid json")
	write("scenarios/not-a-scenario/api/domain.go", "package domain\ntype State struct{}\n")

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{})
	require.NoError(t, err)
	require.Equal(t, 2, got.ScenarioCount)
	require.Equal(t, []string{"broken", "empty"}, []string{got.Scenarios[0].ID, got.Scenarios[1].ID})
	require.Equal(t, 1, got.Scenarios[0].ParseErrors)
	require.Equal(t, 1, got.Scenarios[0].FlowParseErrors)
	require.Empty(t, got.Scenarios[1].CandidateCount)
}

func TestAnalyzeUsesExactCandidateBindingsForGenericHuntTerms(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/agent-manager/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/agent-manager/api/internal/domain/run.go", `package domain

type RunStatus string
const (
  RunPending RunStatus = "pending"
  RunDone RunStatus = "done"
)
type Run struct { Status RunStatus }
func finish(r *Run) { r.Status = RunDone }
type SandboxStatus string
const SandboxReady SandboxStatus = "ready"
type Sandbox struct { Status SandboxStatus }
`)
	index := `{"hits":[
  {"entry_id":"round-32","text":"Round 32 fixed Agent Manager terminal-result persistence and recovery.","candidate_ids":["agent-manager/api/internal/domain#Run.Status"]},
  {"entry_id":"generic-no-bind","text":"Agent Manager Run recovery was discussed."}
]}`

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(index)})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 2)
	var run, sandbox *lifecycleinference.Candidate
	for i := range allInventoryCandidates(got) {
		switch allInventoryCandidates(got)[i].TypeName {
		case "Run":
			run = &allInventoryCandidates(got)[i]
		case "Sandbox":
			sandbox = &allInventoryCandidates(got)[i]
		}
	}
	require.NotNil(t, run)
	require.NotNil(t, sandbox)
	require.Len(t, run.HuntRecords, 1)
	require.Equal(t, "round-32", run.HuntRecords[0].EntryID)
	require.Equal(t, "review", run.HuntRecords[0].Status)
	require.Empty(t, sandbox.HuntRecords, "exact source bindings must not spread to candidates sharing nearby prose")
}

func TestAnalyzeDetectsATypeOwnedTransitionMethod(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/queue/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/queue/api/job.go", `package queue

type JobState string
const (
  JobPending JobState = "pending"
  JobRunning JobState = "running"
)
type Job struct { Status JobState }
func (job *Job) TransitionStatus(next JobState) error {
  if job.Status == next { return nil }
  job.Status = next
  return nil
}

func (job *Job) SetStatus(next JobState) { job.Status = next }
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 1)
	candidate := allInventoryCandidates(got)[0]
	require.Equal(t, "Job.Status", candidate.TypeName+"."+candidate.FieldName)
	require.False(t, candidate.ExplicitlyModeled, "a named owner method is evidence to audit, not proof of a complete transition model")
	require.Len(t, candidate.TransitionOwners, 1, "only the named transition method that owns the field write is authoritative")
	require.Equal(t, "scenarios/queue/api/job.go", candidate.TransitionOwners[0].Path)
	require.Contains(t, candidate.TransitionOwners[0].Text, "TransitionStatus")
}

func TestAnalyzeDetectsTypedOutcomeLifecycleAndRanksPlainOutcomeAsWeak(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/onboarding/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/onboarding/api/apply.go", `package onboarding

type ItemOutcome string
const (
  ItemPending ItemOutcome = "pending"
  ItemApplying ItemOutcome = "applying"
  ItemApplied ItemOutcome = "applied"
)
type ApplyItemResult struct { Outcome ItemOutcome }
func (result *ApplyItemResult) TransitionOutcome(next ItemOutcome) error {
  if result.Outcome == next { return nil }
  if result.Outcome == ItemPending && next == ItemApplying { result.Outcome = next; return nil }
  if result.Outcome == ItemApplying && next == ItemApplied { result.Outcome = next; return nil }
  return nil
}
type AuditResult struct { Outcome string }
`)
	write("scenarios/onboarding/api/apply_test.go", `package onboarding
func TestOutcomeTransition(t *testing.T) { _ = ApplyItemResult{Outcome: ItemPending} }
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 2)
	var applyItem, audit *lifecycleinference.Candidate
	for i := range allInventoryCandidates(got) {
		switch allInventoryCandidates(got)[i].TypeName {
		case "ApplyItemResult":
			applyItem = &allInventoryCandidates(got)[i]
		case "AuditResult":
			audit = &allInventoryCandidates(got)[i]
		}
	}
	require.NotNil(t, applyItem)
	require.Equal(t, "Outcome", applyItem.FieldName)
	require.ElementsMatch(t, []string{"pending", "applying", "applied"}, []string{applyItem.States[0].Value, applyItem.States[1].Value, applyItem.States[2].Value})
	require.Len(t, applyItem.WriteSites, 2)
	require.NotEmpty(t, applyItem.GuardSites)
	require.Len(t, applyItem.TransitionOwners, 1)
	require.Contains(t, applyItem.TransitionOwners[0].Text, "TransitionOutcome")
	require.Equal(t, []string{"scenarios/onboarding/api/apply_test.go"}, applyItem.TestFiles)
	require.Equal(t, "strong-candidate", applyItem.Confidence)
	require.NotNil(t, audit)
	require.Equal(t, "field-only", audit.Confidence, "an untyped outcome with no state evidence or writes is only a weak lead")
}

func TestAnalyzeDetectsTypedFunctionalTransitionOwner(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/content-desk/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/content-desk/api/artifacts.go", `package artifacts

type DraftStatus string
const (
  DraftRequested DraftStatus = "requested"
  DraftApproved DraftStatus = "approved"
)
type Draft struct { Status DraftStatus; Complete bool }
type DraftState struct { Status DraftStatus }
type DraftEvent string
func TransitionDraft(state DraftState, event DraftEvent) (DraftState, error) {
  if state.Status == DraftRequested { return DraftState{Status: DraftApproved}, nil }
  return state, nil
}

func TransitionDraftLifecycle(status DraftStatus, event DraftEvent) (DraftStatus, error) {
  return status, nil
}

func setDraftStatus(draft *Draft, next DraftStatus) { draft.Status = next }
`)
	write("scenarios/content-desk/api/artifacts_test.go", `package artifacts
func TestTransitionDraft(state DraftState) {
  if state.Status == DraftRequested { _ = TransitionDraft(state, "approve") }
}
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 2)
	var candidate *lifecycleinference.Candidate
	for i := range allInventoryCandidates(got) {
		if allInventoryCandidates(got)[i].TypeName == "Draft" {
			candidate = &allInventoryCandidates(got)[i]
		}
	}
	require.NotNil(t, candidate)
	require.Equal(t, "Draft.Status", candidate.TypeName+"."+candidate.FieldName)
	require.Len(t, candidate.TransitionOwners, 1, "only the typed state transformer is an owner; wrappers and ordinary setters are not")
	require.Equal(t, "scenarios/content-desk/api/artifacts.go", candidate.TransitionOwners[0].Path)
	require.Contains(t, candidate.TransitionOwners[0].Text, "TransitionDraft(state DraftState")
	require.Equal(t, []string{"scenarios/content-desk/api/artifacts_test.go"}, candidate.TestFiles, "tests that exercise the typed transition value also cover its persisted domain owner")
}

func TestAnalyzeAssociatesExternalTestsWithImportedLocalTypesOnly(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/device-studio/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/device-studio/api/go.mod", "module device-studio\n")
	write("scenarios/device-studio/api/internal/jobs/types.go", `package jobs

type AttemptState string
const (
  AttemptQueued AttemptState = "queued"
  AttemptRunning AttemptState = "running"
)
type Attempt struct { State AttemptState }
func (attempt *Attempt) Transition() {
  if attempt.State == AttemptQueued { attempt.State = AttemptRunning }
}
`)
	write("scenarios/device-studio/api/internal/jobs/attempt_test.go", `package jobs_test

import jobs "device-studio/internal/jobs"

func TestAttemptTransition(t *testing.T) {
  attempt := jobs.Attempt{State: jobs.AttemptQueued}
  attempt.Transition()
  if attempt.State != jobs.AttemptRunning { t.Fatal("attempt did not start") }
}
`)
	write("scenarios/device-studio/api/internal/jobs/unrelated_test.go", `package jobs_test

import otherjobs "third.party/example/jobs"

func TestUnrelatedAttempt(t *testing.T) {
  other := otherjobs.Attempt{State: "pending"}
  _ = other.State
}
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 1)
	candidate := allInventoryCandidates(got)[0]
	require.Equal(t, "device-studio/api/internal/jobs#Attempt.State", candidate.ID)
	require.Len(t, candidate.TransitionOwners, 1)
	require.Contains(t, candidate.TransitionOwners[0].Text, "attempt.State = AttemptRunning")
	require.Equal(t, []string{"scenarios/device-studio/api/internal/jobs/attempt_test.go"}, candidate.TestFiles,
		"external tests importing this exact package count as coverage; a same-directory import with a different module path does not")
}

func TestAnalyzeImportsSourceLedgerEntriesAndExactBindings(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/agent-manager/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/agent-manager/api/internal/domain/run.go", `package domain

type RunStatus string
const (
  RunPending RunStatus = "pending"
  RunDone RunStatus = "done"
)
type Run struct { Status RunStatus }
type Sandbox struct { Status RunStatus }
func finish(run *Run) { run.Status = RunDone }
`)
	export := `{
  "source":"source-ledger.journal.list",
  "scope":"ecosystem-hunt",
  "coverage":{"complete":true,"entry_count":1,"next_cursor":null},
  "entries":[{"id":"ledger-round-32","kind":"task-record","body":"Round 32 fixed Agent Manager durable terminal result persistence."}],
  "candidate_bindings":[{"entry_id":"ledger-round-32","candidate_ids":["agent-manager/api/internal/domain#Run.Status"]}]
}`
	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(export)})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 2)
	var run, sandbox *lifecycleinference.Candidate
	for i := range allInventoryCandidates(got) {
		switch allInventoryCandidates(got)[i].TypeName {
		case "Run":
			run = &allInventoryCandidates(got)[i]
		case "Sandbox":
			sandbox = &allInventoryCandidates(got)[i]
		}
	}
	require.NotNil(t, run)
	require.NotNil(t, sandbox)
	require.Len(t, run.HuntRecords, 1)
	require.Equal(t, "ledger-round-32", run.HuntRecords[0].EntryID)
	require.Equal(t, "review", run.HuntRecords[0].Status)
	require.Empty(t, sandbox.HuntRecords, "reviewed exact bindings must not spread to another candidate in the same package")
	require.True(t, got.HuntCoverageComplete)
	require.Equal(t, "ecosystem-hunt", got.HuntCoverageScope)
	require.Equal(t, "prior_hunt_review_required", run.PreselectionStatus)
}

func TestAnalyzeAcceptsSourceLedgerTopLevelCompleteExportAndRejectsPartial(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(`{"service":{}}`), 0o644))

	complete := []byte(`{"source":"source-ledger.journal.list","scope":"ecosystem-hunt","complete":true,"entry_count":110,"degraded":[],"entries":[]}`)
	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: complete})
	require.NoError(t, err)
	require.True(t, got.HuntCoverageComplete)

	partial := []byte(`{"source":"source-ledger.journal.list","scope":"ecosystem-hunt","complete":false,"entry_count":40,"degraded":["provider timeout"],"entries":[]}`)
	got, err = lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: partial})
	require.NoError(t, err)
	require.False(t, got.HuntCoverageComplete)
}

func allInventoryCandidates(inventory lifecycleinference.Inventory) []lifecycleinference.Candidate {
	return append(append([]lifecycleinference.Candidate(nil), inventory.Candidates...), inventory.UnrankedCandidates...)
}
