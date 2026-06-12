package queue

import (
	"context"
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// fakeBaselineRunner records calls instead of shelling git-control-tower.
type fakeBaselineRunner struct {
	startEng   BaselineEngagement
	startErr   error
	promoteErr error
	abandonErr error

	started   []string
	promoted  []BaselinePromoteParams
	abandoned []string
}

func (f *fakeBaselineRunner) Start(_ context.Context, scenario string) (BaselineEngagement, error) {
	f.started = append(f.started, scenario)
	if f.startErr != nil {
		return BaselineEngagement{}, f.startErr
	}
	eng := f.startEng
	if eng.Scenario == "" {
		eng.Scenario = scenario
	}
	if eng.Mode == "" {
		eng.Mode = "shadow"
	}
	return eng, nil
}

func (f *fakeBaselineRunner) Promote(_ context.Context, p BaselinePromoteParams) error {
	f.promoted = append(f.promoted, p)
	return f.promoteErr
}

func (f *fakeBaselineRunner) Abandon(_ context.Context, scenario string) error {
	f.abandoned = append(f.abandoned, scenario)
	return f.abandonErr
}

// ---- runner JSON parsing -------------------------------------------------

func TestParseStartJSON(t *testing.T) {
	cases := []struct {
		name        string
		out         string
		wantMode    string
		wantAmbient string
		wantReflex  bool
		wantErr     bool
	}{
		{
			name:        "shadow variant with ambient",
			out:         `{"scenario":"foo","variant":"shadow","ambientVar":"foo","decision":{"mode":"shadow","reflexive":false}}`,
			wantMode:    "shadow",
			wantAmbient: "foo",
		},
		{
			name:       "live variant, reflexive",
			out:        `{"scenario":"test-genie","variant":"live","decision":{"mode":"live","reflexive":true}}`,
			wantMode:   "live",
			wantReflex: true,
		},
		{
			name:     "empty variant falls back to decision mode",
			out:      `{"scenario":"foo","variant":"","decision":{"mode":"shadow"}}`,
			wantMode: "shadow",
		},
		{name: "malformed json errors", out: `{not json`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eng, err := parseStartJSON("foo", tc.out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if eng.Mode != tc.wantMode {
				t.Fatalf("Mode=%q, want %q", eng.Mode, tc.wantMode)
			}
			if eng.AmbientVar != tc.wantAmbient {
				t.Fatalf("AmbientVar=%q, want %q", eng.AmbientVar, tc.wantAmbient)
			}
			if eng.Reflexive != tc.wantReflex {
				t.Fatalf("Reflexive=%v, want %v", eng.Reflexive, tc.wantReflex)
			}
		})
	}
}

// ---- maybeStartEngagement (needs the orchestrator for ProfileForTask) ----

// enabledIntegration wires a real orchestrator (canned audit for "test-scenario")
// + a fake baseline runner, with an initialized task whose profile optionally
// enables BaselinePromote. Returns the integration, the fake, and the task.
func enabledIntegration(t *testing.T, bp *autosteer.BaselinePromoteObjective) (*AutoSteerIntegration, *fakeBaselineRunner, *tasks.TaskItem) {
	t.Helper()
	profileRepo := autosteer.NewMockProfileRepository()
	profile := objectiveProfile("p-bp", "Baseline Profile", "progress")
	profile.BaselinePromote = bp
	if err := profileRepo.CreateProfile(profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	orchestrator, _ := newTestOrchestrator(profileRepo)
	fake := &fakeBaselineRunner{}
	integration := NewAutoSteerIntegration(orchestrator, "").SetBaselineRunner(fake)

	task := &tasks.TaskItem{ID: "task-bp", Type: "scenario", Operation: "improver", AutoSteerProfileID: "p-bp"}
	if err := integration.InitializeAutoSteer(task, "test-scenario"); err != nil {
		t.Fatalf("init auto steer: %v", err)
	}
	return integration, fake, task
}

func TestMaybeStartEngagement_DisabledProfile_NoEngagement(t *testing.T) {
	integration, fake, task := enabledIntegration(t, nil) // no BaselinePromote block
	integration.maybeStartEngagement(task, "test-scenario")
	if len(fake.started) != 0 {
		t.Fatalf("expected no Start calls, got %v", fake.started)
	}
	if got := integration.ShadowScenarioForTask(task.ID); got != "" {
		t.Fatalf("expected no shadow routing, got %q", got)
	}
}

func TestMaybeStartEngagement_Enabled_Shadow(t *testing.T) { // [REQ:EM-BASE-001]
	integration, fake, task := enabledIntegration(t, &autosteer.BaselinePromoteObjective{Enabled: true})
	integration.maybeStartEngagement(task, "test-scenario")
	if len(fake.started) != 1 || fake.started[0] != "test-scenario" {
		t.Fatalf("expected one Start(test-scenario), got %v", fake.started)
	}
	if got := integration.ShadowScenarioForTask(task.ID); got != "test-scenario" {
		t.Fatalf("shadow routing=%q, want test-scenario", got)
	}
	// Idempotent: a second call must not re-start.
	integration.maybeStartEngagement(task, "test-scenario")
	if len(fake.started) != 1 {
		t.Fatalf("expected Start to remain called once, got %v", fake.started)
	}
}

func TestMaybeStartEngagement_SelfScenario_Externalized(t *testing.T) { // [REQ:EM-BASE-003]
	integration, fake, task := enabledIntegration(t, &autosteer.BaselinePromoteObjective{Enabled: true})
	// Treat the canned scenario as "self" so the externalize guard fires.
	integration.selfScenario = "test-scenario"
	integration.maybeStartEngagement(task, "test-scenario")
	if len(fake.started) != 0 {
		t.Fatalf("self scenario must not engage, got Start %v", fake.started)
	}
	if got := integration.ShadowScenarioForTask(task.ID); got != "" {
		t.Fatalf("self scenario must not route to shadow, got %q", got)
	}
}

func TestMaybeStartEngagement_RunnerError_DegradesInPlace(t *testing.T) {
	integration, fake, task := enabledIntegration(t, &autosteer.BaselinePromoteObjective{Enabled: true})
	fake.startErr = context.DeadlineExceeded
	integration.maybeStartEngagement(task, "test-scenario")
	if len(fake.started) != 1 {
		t.Fatalf("expected one Start attempt, got %v", fake.started)
	}
	if got := integration.ShadowScenarioForTask(task.ID); got != "" {
		t.Fatalf("failed start must not route to shadow, got %q", got)
	}
	// A failed start records a sentinel so it isn't retried every iteration.
	integration.maybeStartEngagement(task, "test-scenario")
	if len(fake.started) != 1 {
		t.Fatalf("failed start must not retry, got %v", fake.started)
	}
}

func TestMaybeStartEngagement_LiveMode_NoShadowRouting(t *testing.T) { // [REQ:EM-BASE-002]
	integration, fake, task := enabledIntegration(t, &autosteer.BaselinePromoteObjective{Enabled: true})
	fake.startEng = BaselineEngagement{Mode: "live"}
	integration.maybeStartEngagement(task, "test-scenario")
	if len(fake.started) != 1 {
		t.Fatalf("expected one Start, got %v", fake.started)
	}
	if got := integration.ShadowScenarioForTask(task.ID); got != "" {
		t.Fatalf("live engagement must not route to shadow, got %q", got)
	}
}

// ---- terminal promote / abandon (engagement state seeded directly) -------

// seededIntegration returns an integration with a fake runner and a single
// active engagement, bypassing the orchestrator (these methods operate purely on
// the engagement map).
func seededIntegration(te *taskEngagement) (*AutoSteerIntegration, *fakeBaselineRunner) {
	fake := &fakeBaselineRunner{}
	a := &AutoSteerIntegration{
		baselineRunner: fake,
		selfScenario:   defaultSelfScenario,
		engagements:    map[string]*taskEngagement{"t": te},
	}
	return a, fake
}

func activeShadowEngagement(mode string, cadence int) *taskEngagement {
	return &taskEngagement{
		scenario: "foo", mode: "shadow", ambientVar: "foo",
		promoteMode: mode, cadence: cadence, tagPrefix: "ecosystem-t", runID: "run-1", active: true,
	}
}

func TestMaybeFinishEngagement_ObjectiveMet_Promotes(t *testing.T) { // [REQ:EM-BASE-001]
	a, fake := seededIntegration(activeShadowEngagement(autosteer.BaselinePromoteEndOfEngagement, 0))
	a.maybeFinishEngagement("t", autosteer.StopObjectiveMet)
	if len(fake.promoted) != 1 {
		t.Fatalf("expected one Promote, got %v", fake.promoted)
	}
	p := fake.promoted[0]
	if p.Scenario != "foo" || p.ExcludeRun != "run-1" || p.TagPrefix != "ecosystem-t" {
		t.Fatalf("promote params=%+v, want scenario=foo exclude=run-1 tag=ecosystem-t", p)
	}
	if len(fake.abandoned) != 0 {
		t.Fatalf("must not abandon on objective met")
	}
	// Engagement removed.
	if a.ShadowScenarioForTask("t") != "" {
		t.Fatalf("engagement should be cleared after finish")
	}
}

func TestMaybeFinishEngagement_NotMet_Abandons(t *testing.T) {
	a, fake := seededIntegration(activeShadowEngagement(autosteer.BaselinePromoteEndOfEngagement, 0))
	a.maybeFinishEngagement("t", autosteer.StopBudgetExhausted)
	if len(fake.abandoned) != 1 || fake.abandoned[0] != "foo" {
		t.Fatalf("expected Abandon(foo), got %v", fake.abandoned)
	}
	if len(fake.promoted) != 0 {
		t.Fatalf("must not promote on a non-objective-met stop")
	}
}

func TestMaybeFinishEngagement_NoEngagement_Noop(t *testing.T) {
	fake := &fakeBaselineRunner{}
	a := &AutoSteerIntegration{baselineRunner: fake, selfScenario: defaultSelfScenario, engagements: map[string]*taskEngagement{}}
	a.maybeFinishEngagement("absent", autosteer.StopObjectiveMet)
	if len(fake.promoted) != 0 || len(fake.abandoned) != 0 {
		t.Fatalf("no engagement must be a no-op")
	}
}

// ---- checkpoint_on_green cadence -----------------------------------------

func TestMaybeCheckpointPromote_EndOfEngagement_Noop(t *testing.T) {
	a, fake := seededIntegration(activeShadowEngagement(autosteer.BaselinePromoteEndOfEngagement, 0))
	got := a.maybeCheckpointPromote("t", &autosteer.IterationEvaluation{ObjectiveMet: true, Iteration: 2})
	if got {
		t.Fatalf("end_of_engagement must not checkpoint-promote")
	}
	if len(fake.promoted) != 0 {
		t.Fatalf("no promote expected, got %v", fake.promoted)
	}
	// Engagement still active (finish would still promote it later).
	if a.ShadowScenarioForTask("t") != "foo" {
		t.Fatalf("engagement should remain active")
	}
}

func TestMaybeCheckpointPromote_Green_Promotes(t *testing.T) { // [REQ:EM-BASE-001]
	a, fake := seededIntegration(activeShadowEngagement(autosteer.BaselinePromoteCheckpointOnGreen, 0))
	got := a.maybeCheckpointPromote("t", &autosteer.IterationEvaluation{ObjectiveMet: true, Iteration: 1})
	if !got {
		t.Fatalf("checkpoint_on_green + objective met should promote")
	}
	if len(fake.promoted) != 1 || fake.promoted[0].Scenario != "foo" {
		t.Fatalf("expected Promote(foo), got %v", fake.promoted)
	}
	if a.ShadowScenarioForTask("t") != "" {
		t.Fatalf("engagement should be cleared after checkpoint promote")
	}
}

func TestMaybeCheckpointPromote_NotGreen_Noop(t *testing.T) {
	a, fake := seededIntegration(activeShadowEngagement(autosteer.BaselinePromoteCheckpointOnGreen, 0))
	got := a.maybeCheckpointPromote("t", &autosteer.IterationEvaluation{ObjectiveMet: false, Iteration: 1})
	if got || len(fake.promoted) != 0 {
		t.Fatalf("not-green checkpoint must be a no-op, got promoted=%v", fake.promoted)
	}
}

func TestMaybeCheckpointPromote_CadenceNotDue_Noop(t *testing.T) {
	a, fake := seededIntegration(activeShadowEngagement(autosteer.BaselinePromoteCheckpointOnGreen, 3))
	// Iteration 2 with cadence 3 ⇒ not a checkpoint boundary.
	if a.maybeCheckpointPromote("t", &autosteer.IterationEvaluation{ObjectiveMet: true, Iteration: 2}) {
		t.Fatalf("cadence not due must not promote")
	}
	if len(fake.promoted) != 0 {
		t.Fatalf("no promote expected at iteration 2, got %v", fake.promoted)
	}
	// Iteration 3 is due.
	if !a.maybeCheckpointPromote("t", &autosteer.IterationEvaluation{ObjectiveMet: true, Iteration: 3}) {
		t.Fatalf("cadence due (iter 3, cadence 3) should promote")
	}
	if len(fake.promoted) != 1 {
		t.Fatalf("expected one promote at the cadence boundary, got %v", fake.promoted)
	}
}

// ---- SetEngagementRunID --------------------------------------------------

func TestSetEngagementRunID(t *testing.T) {
	te := activeShadowEngagement(autosteer.BaselinePromoteEndOfEngagement, 0)
	te.runID = ""
	a, fake := seededIntegration(te)
	a.SetEngagementRunID("t", "run-99")
	a.maybeFinishEngagement("t", autosteer.StopObjectiveMet)
	if len(fake.promoted) != 1 || fake.promoted[0].ExcludeRun != "run-99" {
		t.Fatalf("expected promote exclude-run=run-99, got %v", fake.promoted)
	}
}
