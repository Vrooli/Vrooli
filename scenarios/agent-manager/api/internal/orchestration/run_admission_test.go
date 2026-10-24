package orchestration

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/orchestration/testutil"
	"github.com/google/uuid"
)

// launchInfoStub satisfies runner.AgentLaunchInfo and
// runner.RuntimeVersionReporter on top of MockRunner so admission capture can
// be exercised without a real codec.
type launchInfoStub struct {
	*runner.MockRunner
	args            []string
	err             error
	runtimeVersion  string
	runtimeVersionE error
	controlArgs     func(*domain.RunConfig) ([]string, error)
}

func (s *launchInfoStub) TagEnvKey() string { return "STUB_TAG" }

func (s *launchInfoStub) BinaryPath() string { return "/usr/bin/stub" }

func (s *launchInfoStub) ControlArgs(cfg *domain.RunConfig) ([]string, error) {
	if s.controlArgs != nil {
		return s.controlArgs(cfg)
	}
	return s.args, s.err
}

func (s *launchInfoStub) RuntimeVersion(context.Context) (string, error) {
	return s.runtimeVersion, s.runtimeVersionE
}

type closeDispatcherAfterAppendStore struct {
	event.Store
	after func()
	once  sync.Once
}

func (s *closeDispatcherAfterAppendStore) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	if err := s.Store.Append(ctx, runID, events...); err != nil {
		return err
	}
	s.once.Do(s.after)
	return nil
}

func TestCreateRunAdmissionUsesFinalSandboxConfig(t *testing.T) {
	ctx := context.Background()
	o := newDeclarationOrchestrator(t)
	newCurrentModelPolicyFixtureOption(t)(o)
	codec := codecs.NewCodexForTest()
	o.runners = runner.NewRegistry()
	if err := o.runners.Register(&launchInfoStub{
		MockRunner:  runner.NewMockRunner(domain.RunnerTypeCodex),
		controlArgs: codec.ControlArgs,
	}); err != nil {
		t.Fatal(err)
	}
	// Persist the real admission, then close dispatch immediately after the
	// first durable prompt event. This exercises a late enqueue failure without
	// modeling a run that was never admitted.
	o.events = &closeDispatcherAfterAppendStore{Store: o.events, after: o.dispatcher.Close}
	task, err := o.CreateTask(ctx, &domain.Task{
		Title: "sandbox admission", Description: "persisted prompt for post-admission close", ScopePath: ".", ProjectRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	role, network := "code.default", domain.NetworkAccessNone
	const key = "final-sandbox-admission"
	_, err = o.CreateRun(ctx, authenticatedInternalCreateRunFixture(o, CreateRunRequest{
		TaskID: task.ID, RoleRef: &role, NetworkAccess: &network, IdempotencyKey: key,
		SandboxConfig: &domain.SandboxConfig{
			Mode: domain.SandboxModeProtected, ManualReview: true,
			WritePolicy: &domain.WorkspaceWritePolicy{Paths: []string{"src"}},
		},
	}))
	if !errors.Is(err, spawn.ErrDispatcherClosed) {
		t.Fatalf("expected dispatch refusal after persistence, got %v", err)
	}
	saved, err := o.runs.GetByIdempotencyKey(ctx, key)
	if err != nil || saved == nil || saved.ResolvedConfig == nil || saved.ResolvedConfig.Admission == nil {
		t.Fatalf("missing persisted admission: run=%v, err=%v", saved, err)
	}
	want, err := codec.ControlArgs(saved.ResolvedConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(want, " "), "vrooli-network-only") {
		t.Fatalf("expected owner write-policy launch, got %v", want)
	}
	if got := saved.ResolvedConfig.Admission.PassedControlArgs; !reflect.DeepEqual(got, want) {
		t.Fatalf("admission recorded stale controls: got %v, final launch %v", got, want)
	}
}

func TestCreateRunRejectsMissingEffectiveInputBeforeAdmission(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	registry := runner.NewRegistry()
	codec := codecs.NewCodexForTest()
	if err := registry.Register(&launchInfoStub{
		MockRunner:  runner.NewMockRunner(domain.RunnerTypeCodex),
		controlArgs: codec.ControlArgs,
	}); err != nil {
		t.Fatal(err)
	}
	o := New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		WithEvents(eventStore),
		WithRunners(registry),
		WithIdempotency(repos.Idempotency),
	)
	t.Cleanup(o.dispatcher.Close)
	newCurrentModelPolicyFixtureOption(t)(o)

	task, err := o.CreateTask(ctx, &domain.Task{
		Title:       "empty run input",
		ScopePath:   ".",
		ProjectRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	role, network := "code.default", domain.NetworkAccessNone
	const key = "empty-effective-input"
	for attempt := 0; attempt < 2; attempt++ {
		_, err := o.CreateRun(ctx, authenticatedInternalCreateRunFixture(o, CreateRunRequest{
			TaskID: task.ID, RoleRef: &role, NetworkAccess: &network, IdempotencyKey: key,
			ExecutionMode: domain.ExecutionModeInteractive,
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff},
		}))
		var validation *domain.ValidationError
		if !errors.As(err, &validation) || validation.Field != "prompt" {
			runs, _ := o.ListRuns(ctx, RunListOptions{TaskID: &task.ID})
			currentTask, _ := o.GetTask(ctx, task.ID)
			byKey, _ := o.runs.GetByIdempotencyKey(ctx, key)
			t.Fatalf("attempt %d: expected prompt validation refusal, got %v (runs=%d currentRun=%v idempotencyRun=%v dispatcher=%+v)", attempt+1, err, len(runs), currentTask.CurrentRunID, byKey, o.dispatcher.Stats())
		}
	}

	updatedTask, err := o.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedTask.CurrentRunID != nil {
		t.Fatalf("empty-input refusal claimed task for run %s", updatedTask.CurrentRunID)
	}
	if updatedTask.Status != domain.TaskStatusQueued {
		t.Fatalf("empty-input refusal changed task status to %q", updatedTask.Status)
	}
	runs, err := o.ListRuns(ctx, RunListOptions{TaskID: &task.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("empty-input refusal persisted %d run(s)", len(runs))
	}
	byKey, err := o.runs.GetByIdempotencyKey(ctx, key)
	if err != nil || byKey != nil {
		t.Fatalf("empty-input refusal persisted idempotency owner: run=%v err=%v", byKey, err)
	}
	idempotency, err := repos.Idempotency.Check(ctx, key)
	if err != nil || idempotency != nil {
		t.Fatalf("empty-input refusal reserved idempotency key: record=%v err=%v", idempotency, err)
	}
	if stats := o.dispatcher.Stats(); stats.QueueDepth != 0 || stats.ActiveCount != 0 || stats.StartingCount != 0 {
		t.Fatalf("empty-input refusal dispatched work: %+v", stats)
	}
}

func TestEffectiveRunInputValidationPreservesUsableInputs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		task      *domain.Task
		prompt    string
		wantError bool
	}{
		{name: "inherited description", task: &domain.Task{Description: "  do the task  "}},
		{name: "explicit prompt", task: &domain.Task{}, prompt: "  do the task  "},
		{name: "whitespace override does not fall back to description", task: &domain.Task{Description: "do the task"}, prompt: "  ", wantError: true},
		{name: "text attachment only", task: &domain.Task{ContextAttachments: []domain.ContextAttachment{{Type: "note", Content: "reference material"}}}},
		{name: "image attachment only", task: &domain.Task{ContextAttachments: []domain.ContextAttachment{{Type: "image", AttachmentID: "image-id"}}}},
		{name: "empty input", task: &domain.Task{}, wantError: true},
		{name: "invalid attachment is not usable context", task: &domain.Task{ContextAttachments: []domain.ContextAttachment{{Type: "image"}}}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEffectiveRunInput(tc.task, tc.prompt)
			if (err != nil) != tc.wantError {
				t.Fatalf("validation error = %v, wantError %v", err, tc.wantError)
			}
		})
	}
}

func TestBuildRunAdmissionCapturesRequestedAndEffective(t *testing.T) {
	effort := domain.EffortHigh
	model := "gpt-6-luna"
	roleRef := "code.default"
	timeout := 45 * time.Minute
	maxTurns := 12
	req := CreateRunRequest{
		PreferredRunner: "codex",
		Model:           &model,
		RoleRef:         &roleRef,
		Effort:          &effort,
		Timeout:         &timeout,
		MaxTurns:        &maxTurns,
		Until:           "all requested outcomes have evidence",
	}
	cfg := &domain.RunConfig{
		RunnerType: domain.RunnerTypeCodex,
		Model:      "gpt-6-luna",
		Effort:     domain.EffortMedium,
		Timeout:    time.Hour,
		MaxTurns:   20,
		Until:      "all requested outcomes have evidence",
		PolicySnapshot: &domain.ExecutionPolicySnapshot{
			CatalogDigest: "catalog-abc",
			SelectedCandidate: domain.ExecutionCandidate{
				PolicyDigest: "policy-def",
				PolicyPath:   "resources/codex/model-policy.json",
			},
			SelectionReason: "role_policy",
		},
	}

	got := buildRunAdmission(req, cfg)
	if got == nil {
		t.Fatal("admission is nil")
	}
	if got.RequestedRunner != "codex" || got.RequestedModel != "gpt-6-luna" ||
		got.RequestedRoleRef != "code.default" || got.RequestedEffort != "high" ||
		got.RequestedTimeout != timeout || got.RequestedMaxTurns != maxTurns ||
		got.RequestedGoalMode != "until" {
		t.Fatalf("requested side mismatch: %#v", got)
	}
	if got.EffectiveRunner != string(domain.RunnerTypeCodex) || got.EffectiveModel != "gpt-6-luna" ||
		got.EffectiveEffort != string(domain.EffortMedium) || got.EffectiveTimeout != time.Hour ||
		got.EffectiveMaxTurns != 20 || got.EffectiveUntil != "all requested outcomes have evidence" {
		t.Fatalf("effective side mismatch: %#v", got)
	}
	if got.CatalogDigest != "catalog-abc" || got.PolicyDigest != "policy-def" ||
		got.PolicyPath != "resources/codex/model-policy.json" || got.SelectionReason != "role_policy" {
		t.Fatalf("policy provenance mismatch: %#v", got)
	}
}

func TestBuildRunAdmissionLeavesUnrequestedFieldsEmpty(t *testing.T) {
	cfg := &domain.RunConfig{
		RunnerType: domain.RunnerTypeCodex,
		Model:      "gpt-6-luna",
		Effort:     domain.EffortMedium,
		Timeout:    time.Hour,
		MaxTurns:   20,
		PolicySnapshot: &domain.ExecutionPolicySnapshot{
			SelectedCandidate: domain.ExecutionCandidate{PolicyDigest: "policy-def"},
		},
	}

	got := buildRunAdmission(CreateRunRequest{}, cfg)
	if got == nil {
		t.Fatal("admission is nil")
	}
	if got.RequestedRunner != "" || got.RequestedModel != "" || got.RequestedRoleRef != "" ||
		got.RequestedEffort != "" || got.RequestedTimeout != 0 || got.RequestedMaxTurns != 0 ||
		got.RequestedGoalMode != "" {
		t.Fatalf("unrequested side should be empty: %#v", got)
	}
	if got.EffectiveRunner != string(domain.RunnerTypeCodex) || got.EffectiveTimeout != time.Hour {
		t.Fatalf("effective side mismatch: %#v", got)
	}
}

func TestRecordPassedInvocationCapturesControlArgs(t *testing.T) {
	registry := runner.NewRegistry()
	want := []string{"-m", "gpt-6-luna", "-c", "model_reasoning_effort=medium"}
	if err := registry.Register(&launchInfoStub{
		MockRunner:     runner.NewMockRunner(domain.RunnerTypeCodex),
		args:           want,
		runtimeVersion: "codex-cli 0.153.4",
	}); err != nil {
		t.Fatalf("register stub: %v", err)
	}
	o := &Orchestrator{runners: registry}
	admission := &domain.RunAdmission{}

	o.recordPassedInvocation(context.Background(), admission, &domain.RunConfig{RunnerType: domain.RunnerTypeCodex})

	if !reflect.DeepEqual(admission.PassedControlArgs, want) {
		t.Fatalf("passed control args = %#v, want %#v", admission.PassedControlArgs, want)
	}
	if admission.RuntimeVersion != "codex-cli 0.153.4" {
		t.Fatalf("runtime version = %q, want codex-cli 0.153.4", admission.RuntimeVersion)
	}
	if len(admission.TranslationDiagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", admission.TranslationDiagnostics)
	}
}

func TestRecordPassedInvocationRecordsRuntimeVersionDiagnostic(t *testing.T) {
	registry := runner.NewRegistry()
	if err := registry.Register(&launchInfoStub{
		MockRunner:      runner.NewMockRunner(domain.RunnerTypeCodex),
		runtimeVersionE: errors.New("binary missing"),
	}); err != nil {
		t.Fatalf("register stub: %v", err)
	}
	o := &Orchestrator{runners: registry}
	admission := &domain.RunAdmission{}

	o.recordPassedInvocation(context.Background(), admission, &domain.RunConfig{RunnerType: domain.RunnerTypeCodex})

	if admission.RuntimeVersion != "" {
		t.Fatalf("unobserved runtime version must stay empty, got %q", admission.RuntimeVersion)
	}
	found := false
	for _, d := range admission.TranslationDiagnostics {
		if strings.Contains(d, "runtime version not observed") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a runtime-version diagnostic, got %#v", admission.TranslationDiagnostics)
	}
}

func TestRecordPassedInvocationRecordsRefusal(t *testing.T) {
	registry := runner.NewRegistry()
	if err := registry.Register(&launchInfoStub{
		MockRunner: runner.NewMockRunner(domain.RunnerTypeCodex),
		err:        errors.New("codex has no native reasoning effort for \"max\""),
	}); err != nil {
		t.Fatalf("register stub: %v", err)
	}
	o := &Orchestrator{runners: registry}
	admission := &domain.RunAdmission{}

	o.recordPassedInvocation(context.Background(), admission, &domain.RunConfig{RunnerType: domain.RunnerTypeCodex})

	if len(admission.PassedControlArgs) != 0 {
		t.Fatalf("refused translation must not record passed args: %#v", admission.PassedControlArgs)
	}
	if len(admission.TranslationDiagnostics) != 1 {
		t.Fatalf("want one refusal diagnostic, got %#v", admission.TranslationDiagnostics)
	}
}

func TestRecordPassedInvocationWithoutRegistryRecordsDiagnostic(t *testing.T) {
	admission := &domain.RunAdmission{}
	(&Orchestrator{}).recordPassedInvocation(context.Background(), admission, &domain.RunConfig{RunnerType: domain.RunnerTypeCodex})
	if len(admission.TranslationDiagnostics) != 1 {
		t.Fatalf("want one diagnostic when the registry is unavailable, got %#v", admission.TranslationDiagnostics)
	}
}
