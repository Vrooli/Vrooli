package orchestration

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/fixtures"

	"github.com/google/uuid"
)

const tokenCapProfile = "test/capped-worker"

// tokenCapFixture seeds a running codec-pipe run of model whose profile carries
// a 1,000-weighted-token cap. Accounting weighs a run by its resolved model.
func tokenCapFixture(t *testing.T, model string) (*database.Repositories, event.Store, *domain.Run) {
	t.Helper()
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	profile := fixtures.NewAgentProfile(t, fixtures.WithAgentProfileName(tokenCapProfile))
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatal(err)
	}
	task := fixtures.NewTask(t)
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType, cfg.Model = domain.RunnerTypeCodex, model
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, AgentProfileID: &profile.ID, Status: domain.RunStatusRunning, ResolvedConfig: cfg,
		RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeCodecPipe, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	return repos, events, run
}

func tokenCapOrchestrator(repos *database.Repositories, events event.Store, mode TokenCapMode) *Orchestrator {
	policy := DefaultTokenCapPolicy()
	policy.Mode = mode
	policy.Caps = map[string]int64{tokenCapProfile: 1000}
	return New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithTokenCapPolicy(policy))
}

// usageEvent is one provider usage report; turn keeps reports distinct, as
// accounting deduplicates identical reports within an invocation.
func usageEvent(runID uuid.UUID, model string, nonCache, turn int) *domain.RunEvent {
	return &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Timestamp: time.Now(), Data: &domain.UsageEventData{
		PayloadKind: domain.PayloadKindUsage, InputTokens: nonCache, CacheReadTokens: 5000, TurnIndex: turn, Model: model, RunnerType: string(domain.RunnerTypeCodex),
	}}
}

func tokenCapMessages(t *testing.T, events event.Store, runID uuid.UUID) []string {
	t.Helper()
	all, err := events.Get(context.Background(), runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	var messages []string
	for _, evt := range all {
		if data, ok := evt.Data.(*domain.LogEventData); ok && strings.Contains(data.Message, "token cap") {
			messages = append(messages, data.Message)
		}
	}
	return messages
}

func eventually(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Token caps run on every live sink path: the fresh headless executor, the
// interactive tail and recovery (D-1), not only continuations.
func TestTokenCapNudgesOnceAt85PercentThenStopsWithTokenCapReason(t *testing.T) {
	for _, path := range []string{"headless executor", "interactive tail", "recovery"} {
		t.Run(path, func(t *testing.T) { testTokenCapNudgeThenStop(t, path) })
	}
}

func testTokenCapNudgeThenStop(t *testing.T, path string) {
	repos, events, run := tokenCapFixture(t, "gpt-6-luna")
	o := tokenCapOrchestrator(repos, events, TokenCapEnforce)
	sink := runEventSinkPaths(t, o)[path](run)
	ctx := context.Background()

	if err := sink.Emit(usageEvent(run.ID, "gpt-6-luna", 860, 1)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the 85% nudge", func() bool { return len(tokenCapMessages(t, events, run.ID)) == 1 })
	if msg := tokenCapMessages(t, events, run.ID)[0]; !strings.Contains(msg, "860 of 1000") || !strings.Contains(msg, "write the handoff") {
		t.Fatalf("nudge = %q", msg)
	}
	if err := sink.Emit(usageEvent(run.ID, "gpt-6-luna", 50, 2)); err != nil {
		t.Fatal(err)
	}
	if err := sink.Emit(usageEvent(run.ID, "gpt-6-luna", 100, 3)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the token-cap stop", func() bool {
		current, err := repos.Runs.Get(ctx, run.ID)
		return err == nil && current.Status == domain.RunStatusCancelled && current.StopReason == domain.RunStopReasonTokenCap
	})
	stopped, _ := repos.Runs.Get(ctx, run.ID)
	if stopped.TerminalClass != domain.RunTerminalClassInterruption {
		t.Fatalf("terminal class = %q, want interruption", stopped.TerminalClass)
	}
	messages := tokenCapMessages(t, events, run.ID)
	if len(messages) != 2 || !strings.Contains(messages[1], "1010 weighted tokens reached the cap of 1000") {
		t.Fatalf("want one nudge and one stop record, got %q", messages)
	}
}

func TestTokenCapReportOnlyWeighsSolAndNeverStops(t *testing.T) {
	repos, events, run := tokenCapFixture(t, "gpt-6-sol")
	o := tokenCapOrchestrator(repos, events, TokenCapReport)

	// 110 non-cache Sol tokens weigh 1,100; cache reads never count.
	if err := o.runEventSink(run.ID).Emit(usageEvent(run.ID, "gpt-6-sol", 110, 1)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the report-only cap record", func() bool {
		for _, msg := range tokenCapMessages(t, events, run.ID) {
			if strings.Contains(msg, "report-only") && strings.Contains(msg, "1100 weighted tokens reached the cap of 1000") {
				return true
			}
		}
		return false
	})
	current, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil || current.Status != domain.RunStatusRunning || current.StopReason != "" {
		t.Fatalf("report-only mode changed the run: %+v err=%v", current, err)
	}
}

func TestRunAccountingAndTokenCapPreserveIdenticalReceiptsAcrossContinuations(t *testing.T) {
	repos, events, run := tokenCapFixture(t, "gpt-6-luna")
	o := tokenCapOrchestrator(repos, events, TokenCapReport)
	ctx := context.Background()
	usage := func() *domain.RunEvent {
		return &domain.RunEvent{RunID: run.ID, EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{
			PayloadKind: domain.PayloadKindUsage, InputTokens: 100, CacheReadTokens: 5000,
			TurnIndex: 1, Model: "gpt-6-luna", RunnerType: string(domain.RunnerTypeCodex), ReconciliationAuthority: true,
		}}
	}
	if err := events.Append(ctx, run.ID,
		domain.NewStatusEvent(run.ID, "starting", "running", "first invocation"),
		usage(),
		domain.NewStatusEvent(run.ID, "complete", "running", "continuation"),
		usage(),
	); err != nil {
		t.Fatal(err)
	}

	accounting, err := o.RunAccounting(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if accounting.Tokens != 10200 || accounting.CacheReadTokens != 10000 || accounting.NonCacheTokens != 200 {
		t.Errorf("full-stream accounting lost a continuation: %+v", accounting)
	}
	weighted, err := o.weightedRunTokens(ctx, run, o.capPolicy())
	if err != nil || weighted == nil || *weighted != 200 {
		t.Errorf("token-cap weight disagreed with full run accounting: got %v, err=%v", weighted, err)
	}
}

func TestTokenCapSeedsFromPersistedUsageAfterRestart(t *testing.T) {
	repos, events, run := tokenCapFixture(t, "gpt-6-luna")
	// Usage persisted before this owner instance existed.
	if err := events.Append(context.Background(), run.ID, usageEvent(run.ID, "gpt-6-luna", 950, 1)); err != nil {
		t.Fatal(err)
	}
	o := tokenCapOrchestrator(repos, events, TokenCapEnforce)
	if err := o.runEventSink(run.ID).Emit(usageEvent(run.ID, "gpt-6-luna", 60, 2)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the stop counted from persisted usage", func() bool {
		current, err := repos.Runs.Get(context.Background(), run.ID)
		return err == nil && current.StopReason == domain.RunStopReasonTokenCap
	})
}

func TestTokenCapUsageDistinguishesMissingAndZeroReceipts(t *testing.T) {
	repos, events, run := tokenCapFixture(t, "gpt-6-luna")
	o := tokenCapOrchestrator(repos, events, TokenCapReport)
	ctx := context.Background()
	weighted, limit, mode := o.TokenCapUsage(ctx, run.ID)
	if weighted != nil || limit != 1000 || mode != string(TokenCapReport) {
		t.Fatalf("missing usage = %v, %d, %q; want unknown with configured cap", weighted, limit, mode)
	}
	for _, tokens := range []int{0, 25} {
		if err := events.Append(ctx, run.ID, usageEvent(run.ID, "gpt-6-luna", tokens, tokens+1)); err != nil {
			t.Fatal(err)
		}
		weighted, _, _ = o.TokenCapUsage(ctx, run.ID)
		if weighted == nil || *weighted != float64(tokens) {
			t.Fatalf("after %d-token receipt, usage = %v", tokens, weighted)
		}
	}
}

func TestTokenCapPolicyFromEnvDefaultsOff(t *testing.T) {
	t.Setenv(tokenCapModeEnv, "")
	if policy := TokenCapPolicyFromEnv(); policy.Mode != TokenCapOff || policy.weight("gpt-6-sol") != 10 || policy.weight("gpt-6-luna") != 1 || policy.weight("other") != 1 {
		t.Fatalf("default policy = %+v", policy)
	}
	t.Setenv(tokenCapModeEnv, "bogus")
	if policy := TokenCapPolicyFromEnv(); policy.Mode != TokenCapOff {
		t.Fatalf("an unknown mode enabled caps: %+v", policy)
	}
	t.Setenv(tokenCapModeEnv, "Enforce")
	t.Setenv(tokenCapsEnv, `{"prompt-manager/delivery-orchestrator":150000}`)
	t.Setenv(tokenCapWeightsEnv, "sol=12,luna=1")
	policy := TokenCapPolicyFromEnv()
	if policy.Mode != TokenCapEnforce || policy.Caps["prompt-manager/delivery-orchestrator"] != 150000 || policy.weight("gpt-6-sol") != 12 {
		t.Fatalf("configured policy = %+v", policy)
	}
	t.Setenv(tokenCapsEnv, "not json")
	if policy := TokenCapPolicyFromEnv(); len(policy.Caps) != 0 {
		t.Fatalf("invalid caps were applied: %+v", policy.Caps)
	}
}

func TestCreateRunCanOnlyLowerTheProfileTokenCap(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	profile := fixtures.NewAgentProfile(t, fixtures.WithAgentProfileName("test/capped-profile"))
	profile.MaxWeightedTokens = 1000
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatal(err)
	}
	stored, err := repos.Profiles.Get(ctx, profile.ID)
	if err != nil || stored.MaxWeightedTokens != 1000 {
		t.Fatalf("the profile cap did not persist: %+v %v", stored, err)
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, newCurrentModelPolicyFixtureOption(t))
	for _, tc := range []struct {
		override *int64
		want     int64
	}{{nil, 1000}, {ptrInt64(400), 400}, {ptrInt64(5000), 1000}} {
		cfg, _, err := o.resolveRunConfig(ctx, CreateRunRequest{AgentProfileID: &profile.ID, MaxWeightedTokens: tc.override})
		if err != nil || cfg.MaxWeightedTokens != tc.want {
			t.Fatalf("override %v: resolved %+v, want cap %d (err %v)", tc.override, cfg, tc.want, err)
		}
	}
	if _, _, err := o.resolveRunConfig(ctx, CreateRunRequest{AgentProfileID: &profile.ID, MaxWeightedTokens: ptrInt64(0)}); err == nil {
		t.Fatal("a zero run cap must be refused, not read as no cap")
	}
}

func TestTokenCapPrefersTheRunCapOverTheEnvironmentFallback(t *testing.T) {
	repos, events, run := tokenCapFixture(t, "gpt-6-luna")
	run.ResolvedConfig.MaxWeightedTokens = 500
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	o := tokenCapOrchestrator(repos, events, TokenCapEnforce) // environment cap 1,000
	if err := o.runEventSink(run.ID).Emit(usageEvent(run.ID, "gpt-6-luna", 520, 1)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the stop at the run cap", func() bool {
		current, err := repos.Runs.Get(context.Background(), run.ID)
		return err == nil && current.Status == domain.RunStatusCancelled && current.StopReason == domain.RunStopReasonTokenCap
	})
	if messages := tokenCapMessages(t, events, run.ID); len(messages) == 0 || !strings.Contains(messages[len(messages)-1], "reached the cap of 500") {
		t.Fatalf("the run cap of 500 was not the enforced cap: %q", messages)
	}
}

func ptrInt64(v int64) *int64 { return &v }
