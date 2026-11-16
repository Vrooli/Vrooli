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

func TestAnalyzeScenarioIDScopesSourceWorkToOneOwner(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/alpha/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/alpha/api/job.go", `package alpha
type JobState string
const ( Pending JobState = "pending"; Done JobState = "done" )
type Job struct { State JobState }
func finish(job *Job) { job.State = Done }
`)
	write("scenarios/beta/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/beta/api/broken.go", "package beta\nfunc (\n")

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{ScenarioID: "alpha"})
	require.NoError(t, err)
	require.Equal(t, 1, got.ScenarioCount)
	require.Equal(t, "alpha", got.Scenarios[0].ID)
	require.Equal(t, 0, got.Scenarios[0].ParseErrors)
	require.NotEmpty(t, allInventoryCandidates(got))
}

func TestImplicitSignalIsTypedAndNeverRankedAsPersistedLifecycle(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/alpha/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/alpha/api/work.go", `package alpha
type Worker struct { started bool; stopped bool; done chan struct{}; attempts int }
func (w *Worker) Start() { w.started = true }
func (w *Worker) Stop() { w.stopped = true; close(w.done) }
type RetryConfig struct { Retry bool; Attempts int }
func configure(c RetryConfig) {}
`)
	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{})
	require.NoError(t, err)
	require.Empty(t, got.Candidates, "non-ranked implicit signals must not enter the persisted-lifecycle rank")
	require.Len(t, got.UnrankedCandidates, 1, "channel and retry configuration without coupled flags on a behavior owner is a counterexample")
	require.Equal(t, lifecycleinference.CandidateKindImplicitSignal, got.UnrankedCandidates[0].Kind)
	require.Equal(t, "unknown", string(got.UnrankedCandidates[0].Persistence.Class))
	require.Equal(t, "implicit_signal_non_ranked", got.UnrankedCandidates[0].PreselectionStatus)
	require.Empty(t, got.PreselectionCandidates)
}

func TestAgentManagerRecoveryIsVisibleAsNonPersistentImplicitSignal(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	require.NoError(t, err)
	got, err := lifecycleinference.Analyze(repoRoot, lifecycleinference.Options{ScenarioID: "agent-manager"})
	require.NoError(t, err)
	var recovery *lifecycleinference.Candidate
	implicitCount := 0
	for i := range got.UnrankedCandidates {
		if got.UnrankedCandidates[i].Kind == "implicit_signal" {
			implicitCount++
		}
		if got.UnrankedCandidates[i].TypeName == "Recovery" && got.UnrankedCandidates[i].Kind == "implicit_signal" {
			recovery = &got.UnrankedCandidates[i]
		}
	}
	require.NotNil(t, recovery, "Recovery couples startupFinished/stopped with done-channel coordination and receiver behavior")
	require.Equal(t, "unknown", string(recovery.Persistence.Class), "volatile in-memory coordination must not inflate durable lifecycle totals")
	require.Equal(t, "implicit_signal_non_ranked", recovery.PreselectionStatus)
	t.Logf("real owner scan: %d standard candidates + %d explicit non-ranked implicit signal(s); persisted rank unchanged", got.CandidateCount-implicitCount, implicitCount)
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

func TestAnalyzeRecognizesTransitionGraphOnlyWithInvalidStateTest(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/queue/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/queue/api/job.go", `package queue
import "fmt"
type JobState string
const ( JobPending JobState = "pending"; JobRunning JobState = "running" )
var jobTransitions = map[JobState]map[JobState]bool{JobPending: {JobRunning: true}}
type Job struct { Status JobState }
func (job *Job) TransitionStatus(next JobState) error {
  if !jobTransitions[job.Status][next] { return fmt.Errorf("invalid transition: %s -> %s", job.Status, next) }
  job.Status = next
  return nil
}


`)
	write("scenarios/queue/api/job_test.go", `package queue
import "testing"
func TestTransitionStatusRejectsUnknownState(t *testing.T) {
  job := Job{Status: JobPending}
  if err := job.TransitionStatus("unknown"); err == nil { t.Fatal("expected invalid state rejection") }
}
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(`{"scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[]}`)})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 1)
	candidate := allInventoryCandidates(got)[0]
	require.True(t, candidate.TransitionGraphVerified)
	require.True(t, candidate.InvalidTransitionTested)
	require.Equal(t, "already_explicitly_modeled", candidate.PreselectionStatus)
	require.NotEqual(t, lifecycleinference.ClassPersistedLifecycle, candidate.Persistence.Class, "explicit modeling is independent from the persistence classification")
}

func TestAnalyzeRecognizesFieldGraphDelegatedToTransitionHelper(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/queue/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/queue/api/steps.go", `package queue
import "fmt"
type StepStatus string
const (
  StepPending StepStatus = "pending"
  StepRunning StepStatus = "running"
  StepDone StepStatus = "done"
)
var stepTransitions = map[StepStatus]map[StepStatus]struct{}{
  StepPending: {StepRunning: {}},
  StepRunning: {StepDone: {}},
}
type ReviewStep struct { Status StepStatus }
func transitionStepStatus(current *StepStatus, next StepStatus, allowed map[StepStatus]map[StepStatus]struct{}) error {
  if current == nil { return fmt.Errorf("missing current state") }
  if _, ok := allowed[*current][next]; !ok { return fmt.Errorf("invalid transition: %s -> %s", *current, next) }
  *current = next
  return nil
}
func (review *ReviewStep) TransitionStatus(next StepStatus) error {
  return transitionStepStatus(&review.Status, next, stepTransitions)
}
`)
	write("scenarios/queue/api/steps_test.go", `package queue
import "testing"
func TestReviewStepTransitionRejectsUnknownState(t *testing.T) {
  step := ReviewStep{Status: StepPending}
  if err := step.TransitionStatus("unknown"); err == nil { t.Fatal("expected invalid state rejection") }
}
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(`{"scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[]}`)})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 1)
	candidate := allInventoryCandidates(got)[0]
	require.Equal(t, "ReviewStep.Status", candidate.TypeName+"."+candidate.FieldName)
	require.True(t, candidate.TransitionGraphVerified)
	require.True(t, candidate.InvalidTransitionTested)
	require.Equal(t, "already_explicitly_modeled", candidate.PreselectionStatus)
	require.Len(t, candidate.TransitionOwners, 1)
	require.Contains(t, candidate.TransitionOwners[0].Text, "TransitionStatus")
}

func TestAnalyzeRecognizesRecordOwnedTypedStatusPolicy(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/queue/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/queue/api/file.go", `package queue
const (
  StatusPending = "pending"
  StatusIndexed = "indexed"
)
type FileRow struct { Status string }
func (row FileRow) CanTransitionTo(next string) bool {
  if !knownStatus(next) || !knownStatus(row.Status) { return false }
  if row.Status == next { return true }
  switch row.Status {
  case StatusPending: return next == StatusIndexed
  default: return false
  }
}
func knownStatus(status string) bool {
  switch status { case StatusPending, StatusIndexed: return true; default: return false }
}
`)
	write("scenarios/queue/api/file_test.go", `package queue
import "testing"
func TestFileRowRejectsUnknownTransition(t *testing.T) {
  if (FileRow{Status: StatusPending}).CanTransitionTo("unknown") { t.Fatal("unknown transition allowed") }
}
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(`{"scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[]}`)})
	require.NoError(t, err)
	require.Len(t, allInventoryCandidates(got), 1)
	candidate := allInventoryCandidates(got)[0]
	require.Equal(t, "FileRow.Status", candidate.TypeName+"."+candidate.FieldName)
	require.True(t, candidate.TransitionGraphVerified)
	require.True(t, candidate.InvalidTransitionTested)
	require.Equal(t, "already_explicitly_modeled", candidate.PreselectionStatus)
	require.Len(t, candidate.TransitionOwners, 1)
	require.Contains(t, candidate.TransitionOwners[0].Text, "CanTransitionTo")
}

func TestAnalyzeRecognizesPureTransitionFunctionAndMatrixRejectionTests(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/content-desk/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/content-desk/api/draft.go", `package artifacts
import "fmt"
type DraftStatus string
type DraftEvent string
const ( DraftRequested DraftStatus = "requested"; DraftDrafting DraftStatus = "drafting" )
type Draft struct { Status DraftStatus }
type DraftState struct { Status DraftStatus }
func TransitionDraft(state DraftState, event DraftEvent) (DraftState, error) {
  switch state.Status {
  case DraftRequested:
    if event == "begin" { return DraftState{Status: DraftDrafting}, nil }
  }
  return state, fmt.Errorf("invalid draft transition")
}


`)
	write("scenarios/content-desk/api/draft_test.go", `package artifacts
import "testing"
func TestDraftTransitionMatrix(t *testing.T) {
  valid := map[string]bool{"requested/begin": true}
  for edge, isValid := range valid {
    _, err := TransitionDraft(DraftState{Status: DraftRequested}, DraftEvent("begin"))
    if isValid && err == nil { continue }
    if !isValid && err == nil { t.Fatal("illegal transition accepted", edge) }
  }
  _, err := TransitionDraft(DraftState{Status: DraftDrafting}, "begin")
  if err == nil { t.Fatal("invalid transition accepted") }
}
`)
	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(`{"scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[]}`)})
	require.NoError(t, err)
	found := false
	for _, candidate := range allInventoryCandidates(got) {
		if candidate.TypeName == "Draft" && candidate.FieldName == "Status" {
			found = true
			require.True(t, candidate.TransitionGraphVerified)
			require.True(t, candidate.InvalidTransitionTested)
		}
	}
	require.True(t, found)
}

func TestAnalyzeRecognizesSwitchGuardBeforeFieldMutation(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/models/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/models/api/model.go", `package models
import "fmt"
type State string
const ( Ready State = "ready"; Failed State = "failed" )
type ModelLifecycle struct { State State }
func (s ModelLifecycle) Transition(next State) (ModelLifecycle, error) {
  valid := false
  switch next { case Ready: valid = s.State == Failed; case Failed: valid = s.State == Ready }
  if !valid { return s, fmt.Errorf("invalid transition") }
  s.State = next
  return s, nil
}`)
	write("scenarios/models/api/model_test.go", `package models
import "testing"
func TestTransitionRejectsUnknown(t *testing.T) {
  _, err := (ModelLifecycle{State: Ready}).Transition("unknown")
  if err == nil { t.Fatal("invalid transition accepted") }
}`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(`{"scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[]}`)})
	require.NoError(t, err)
	for _, candidate := range allInventoryCandidates(got) {
		if candidate.TypeName == "ModelLifecycle" && candidate.FieldName == "State" {
			require.True(t, candidate.TransitionGraphVerified)
			require.True(t, candidate.InvalidTransitionTested)
			return
		}
	}
	t.Fatal("ModelLifecycle.State candidate not found")
}

func TestAnalyzeRecognizesSQLCompareAndSwapLifecycle(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/arms/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/arms/api/registry.go", `package arms
import ("context"; "database/sql"; "fmt")
type State string
const ( StateNew State = "new"; StateReady State = "ready" )
type Arm struct { State State }
type TransitionRequest struct { From, To State }
type Registry struct { db *sql.DB }
func (r *Registry) Transition(ctx context.Context, request TransitionRequest) (Arm, error) {
  if err := validateTransitionRequest(request); err != nil { return Arm{}, err }
  _, err := r.db.ExecContext(ctx, "UPDATE arms SET state = ? WHERE arm_id = ? AND state = ?", request.To, request.From, request.From)
  if err != nil { return Arm{}, err }
  return Arm{State: request.To}, nil
}


func validateTransitionRequest(request TransitionRequest) error {
  if !legalTransition(request.From, request.To) { return fmt.Errorf("%w: illegal", ErrIllegalTransition) }
  return nil
}
var ErrIllegalTransition = fmt.Errorf("illegal transition")
func legalTransition(from, to State) bool { return from == StateNew && to == StateReady }
`)
	write("scenarios/arms/api/registry_test.go", `package arms
import ("errors"; "testing")
func TestTransitionRejectsIllegal(t *testing.T) {
  _, err := new(Registry).Transition(nil, TransitionRequest{From: StateReady, To: StateNew})
  if !errors.Is(err, ErrIllegalTransition) { t.Fatalf("expected illegal transition: %v", err) }
}
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(`{"scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[]}`)})
	require.NoError(t, err)
	for _, candidate := range allInventoryCandidates(got) {
		if candidate.TypeName == "Arm" && candidate.FieldName == "State" {
			require.True(t, candidate.TransitionGraphVerified)
			require.True(t, candidate.InvalidTransitionTested)
			require.NotEmpty(t, candidate.TransitionOwners)
			return
		}
	}
	t.Fatal("Arm.State candidate not found")
}

func TestAnalyzeRecognizesNamedCanTransitionPolicy(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/execution/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/execution/api/state.go", `package execution
import "fmt"
type Status string
const ( Required Status = "required"; Complete Status = "complete" )
type BaselineSetState struct { Status Status }
func (s BaselineSetState) CanTransitionTo(next Status) bool {
  switch s.Status { case Required: return next == Complete; case Complete: return next == Complete }
  return false
}

func (s *BaselineSetState) TransitionTo(next Status) error {
  if !s.CanTransitionTo(next) { return fmt.Errorf("invalid transition") }
  s.Status = next
  return nil
}
`)
	write("scenarios/execution/api/state_test.go", `package execution
import "testing"
func TestTransitionToRejectsUnknown(t *testing.T) {
  state := BaselineSetState{Status: Complete}
  if err := state.TransitionTo("unknown"); err == nil { t.Fatal("unknown transition accepted") }
}
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(`{"scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[]}`)})
	require.NoError(t, err)
	for _, candidate := range allInventoryCandidates(got) {
		if candidate.TypeName == "BaselineSetState" && candidate.FieldName == "Status" {
			require.True(t, candidate.TransitionGraphVerified)
			require.True(t, candidate.InvalidTransitionTested)
			return
		}
	}
	t.Fatal("BaselineSetState.Status candidate not found")
}

func TestAnalyzeRecognizesTypedStatePolicyAtPersistedBoundary(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
	write("scenarios/indexer/.vrooli/service.json", `{"service":{}}`)
	write("scenarios/indexer/api/jobs.go", `package indexer
import "fmt"
type JobState string
const ( Queued JobState = "queued"; Running JobState = "running"; Done JobState = "done" )
func (s JobState) CanTransitionTo(next JobState) bool {
 switch s { case Queued: return next == Running; case Running: return next == Done }
 return false
}
type Job struct { State JobState }
func (store *Store) Update(job *Job, next JobState) error { if !job.State.CanTransitionTo(next) { return fmt.Errorf("invalid transition") }; job.State = next; return nil }
type Store struct{}
`)
	write("scenarios/indexer/api/jobs_test.go", `package indexer
import "testing"
func TestStoreRejectsUnknownState(t *testing.T) { var store Store; job := Job{State: JobState("unknown")}; if err := store.Update(&job, Running); err == nil { t.Fatal("unknown state accepted") } }
`)

	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: []byte(`{"scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[]}`)})
	require.NoError(t, err)
	for _, candidate := range allInventoryCandidates(got) {
		if candidate.TypeName == "Job" && candidate.FieldName == "State" {
			require.True(t, candidate.TransitionGraphVerified)
			require.True(t, candidate.InvalidTransitionTested)
			require.NotEmpty(t, candidate.TransitionOwners)
			return
		}
	}
	t.Fatal("Job.State candidate not found")
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

	complete := []byte(`{"source":"source-ledger.journal.export","scope":"ecosystem-hunt","complete":true,"entry_count":110,"degraded":false,"entries":[{"id":"record-1","body":"Ecosystem Hunt Round #7: Alpha Order workflow status lifecycle"}]}`)
	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: complete})
	require.NoError(t, err)
	require.True(t, got.HuntCoverageComplete)

	degraded := []byte(`{"source":"source-ledger.journal.export","scope":"ecosystem-hunt","complete":true,"entry_count":110,"degraded":true,"entries":[]}`)
	got, err = lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: degraded})
	require.NoError(t, err)
	require.False(t, got.HuntCoverageComplete, "a degraded authoritative export is not complete coverage")

	partial := []byte(`{"source":"source-ledger.journal.list","scope":"ecosystem-hunt","complete":false,"entry_count":40,"degraded":["provider timeout"],"entries":[]}`)
	got, err = lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: partial})
	require.NoError(t, err)
	require.False(t, got.HuntCoverageComplete)
}

func TestAnalyzeSourceLedgerExportCrossReferencesEntryIDAndBody(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(`{"service":{}}`), 0o644))
	source := `package domain
 type OrderStatus string
 const (OrderPending OrderStatus = "pending"; OrderDone OrderStatus = "done")
 type Order struct { Status OrderStatus }
 func finish(order *Order) { run.Status = OrderDone }
 `
	path := filepath.Join(root, "scenarios", "alpha", "api", "domain", "run.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(source), 0o644))
	export := []byte(`{"source":"source-ledger.journal.export","scope":"ecosystem-hunt","complete":true,"degraded":false,"entry_count":1,"entries":[{"id":"entry-1","body":"Ecosystem Hunt Round #7: Alpha Order lifecycle"}]}`)
	got, err := lifecycleinference.Analyze(root, lifecycleinference.Options{HuntIndexJSON: export})
	require.NoError(t, err)
	require.True(t, got.HuntCoverageComplete)
	require.Equal(t, "ecosystem-hunt", got.HuntCoverageScope)
	var candidate *lifecycleinference.Candidate
	for _, item := range allInventoryCandidates(got) {
		if item.TypeName == "Order" {
			candidate = &item
			break
		}
	}
	require.NotNil(t, candidate)
	require.Len(t, candidate.HuntRecords, 1)
	require.Equal(t, "entry-1", candidate.HuntRecords[0].EntryID)
	require.Equal(t, "review", candidate.HuntRecords[0].Status)
	require.Equal(t, "prior_hunt_review_required", candidate.PreselectionStatus)
}

func allInventoryCandidates(inventory lifecycleinference.Inventory) []lifecycleinference.Candidate {
	return append(append([]lifecycleinference.Candidate(nil), inventory.Candidates...), inventory.UnrankedCandidates...)
}
