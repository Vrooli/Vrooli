package database

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
)

// usageFixture seeds runs and the requests they made, so windowed totals can
// be checked against what a provider or the transcript files would report.
type usageFixture struct {
	t    *testing.T
	repo *invocationReadModelRepository
	db   *DB
	seq  int
}

func newUsageFixture(t *testing.T) *usageFixture {
	t.Helper()
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	return &usageFixture{t: t, repo: &invocationReadModelRepository{db: db}, db: db}
}

func (f *usageFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.ExecContext(context.Background(), query, args...); err != nil {
		f.t.Fatalf("%s: %v", query, err)
	}
}

type fixtureRun struct {
	id, mode, status, billing, runner, model string
	started, ended                           time.Time
	lifetimeTokens                           int64
	lifetimeCostUSD                          float64
}

// run seeds the run row and its read-model summary; the summary carries the
// lifetime counters the windowed readers must not use.
func (f *usageFixture) run(r fixtureRun) {
	f.t.Helper()
	if r.status == "" {
		r.status = "complete"
	}
	if r.runner == "" {
		r.runner = "codex"
	}
	if r.model == "" {
		r.model = "gpt-test"
	}
	f.exec(`INSERT INTO runs (id, execution_mode, status, billing_snapshot, resolved_config) VALUES (?, ?, ?, ?, '{}')`, r.id, r.mode, r.status, `{"mode":"`+r.billing+`"}`)
	started, ended := r.started, r.ended
	fact := invocationreadmodel.RunFact{
		RunID: r.id, OccurredAt: ended, CreatedAt: started, StartedAt: &started, EndedAt: &ended, DurationMS: ended.Sub(started).Milliseconds(),
		Status: r.status, ProfileID: "profile", RunnerType: r.runner, Model: r.model, Tag: "t", WorkloadKind: "adhoc",
		TotalTokens: r.lifetimeTokens, InputTokens: r.lifetimeTokens, TotalCostUSD: r.lifetimeCostUSD, TotalChargeMicroUSD: int64(r.lifetimeCostUSD * 1e6),
		CostTimeBasis: "terminal_projection", TimeBasis: "ingestion", ProjectedAt: ended,
	}
	if err := f.repo.ReplaceRun(context.Background(), fact); err != nil {
		f.t.Fatalf("seed run %s: %v", r.id, err)
	}
}

// request seeds one transcript request of tokens input tokens at observed.
func (f *usageFixture) request(runID, fileKey, session string, observed time.Time, tokens int) {
	f.t.Helper()
	f.seq++
	f.exec(`INSERT OR IGNORE INTO transcript_files (file_key, path, root_kind, root_path, harness, session_id, run_id, first_seen_at) VALUES (?, ?, 'codex', '/r', 'codex', ?, ?, '2026-10-01T00:00:00Z')`, fileKey, "/p/"+fileKey, session, runID)
	f.exec(`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at, model, turn_index, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens) VALUES (?, ?, 0, ?, ?, 'gpt-test', ?, ?, 0, 0, 0)`,
		fileKey, f.seq, runID, observed.UTC().Format(time.RFC3339Nano), f.seq, tokens)
}

var (
	usageNow    = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	usageWeek   = usageNow.Add(-7 * 24 * time.Hour)
	usageFilter = invocationreadmodel.Filter{From: &usageWeek, To: &usageNow}
)

// A run that started 30 days ago and ended today contributes only the requests
// of the last seven days to a seven-day total.
func TestWindowedUsageCountsOnlyRequestsInsideTheWindow(t *testing.T) {
	f := newUsageFixture(t)
	f.run(fixtureRun{id: "long", mode: "codec_pipe", started: usageNow.Add(-30 * 24 * time.Hour), ended: usageNow.Add(-time.Hour), lifetimeTokens: 1_000_000})
	f.request("long", "k-long", "s-long", usageNow.Add(-29*24*time.Hour), 600_000)
	f.request("long", "k-long", "s-long", usageNow.Add(-2*24*time.Hour), 300_000)
	f.request("long", "k-long", "s-long", usageNow.Add(-3*time.Hour), 100_000)

	got, err := f.repo.WindowedUsage(context.Background(), usageFilter)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalTokens != 400_000 || got.Requests != 2 || got.Runs != 1 {
		t.Fatalf("7-day usage = %+v, want 400000 tokens in 2 requests of 1 run (lifetime counter is 1000000)", got)
	}
}

// Token totals count every request, including operator sessions; run
// statistics count only runs Agent Manager executed unless a scope is chosen.
func TestRunScopeSeparatesRunStatisticsFromTokenTotals(t *testing.T) {
	f := newUsageFixture(t)
	f.run(fixtureRun{id: "am", mode: "codec_pipe", started: usageNow.Add(-2 * time.Hour), ended: usageNow.Add(-time.Hour)})
	f.run(fixtureRun{id: "imported", mode: "imported", started: usageNow.Add(-5 * time.Hour), ended: usageNow.Add(-4 * time.Hour), lifetimeTokens: 500})
	f.run(fixtureRun{id: "attached-empty", mode: "attached", started: usageNow.Add(-5 * time.Hour), ended: usageNow.Add(-4 * time.Hour)})
	f.request("am", "k-am", "s-am", usageNow.Add(-90*time.Minute), 100)
	f.request("imported", "k-imp", "s-imp", usageNow.Add(-270*time.Minute), 500)
	ctx := context.Background()

	cases := []struct {
		scope      invocationreadmodel.RunScope
		wantRuns   int64
		wantTokens int64
	}{
		{invocationreadmodel.ScopeUnspecified, 1, 600},
		{invocationreadmodel.ScopeAgentManager, 1, 100},
		{invocationreadmodel.ScopeOperatorSessions, 1, 500},
		{invocationreadmodel.ScopeAll, 2, 600},
	}
	for _, tc := range cases {
		filter := usageFilter
		filter.Scope = tc.scope
		runs, err := f.repo.RunMetrics(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		usage, err := f.repo.WindowedUsage(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		wantRuns := tc.wantRuns
		if tc.scope == invocationreadmodel.ScopeUnspecified {
			wantRuns = 1 // run statistics default to Agent Manager runs
		}
		if runs.TotalRuns != wantRuns || usage.TotalTokens != tc.wantTokens {
			t.Fatalf("scope %q: runs=%d tokens=%d, want runs=%d tokens=%d (attached run with no usage never counts)", tc.scope, runs.TotalRuns, usage.TotalTokens, wantRuns, tc.wantTokens)
		}
	}
}

// One harness session copied into two harness homes is two imported runs with
// the same requests; they count once in tokens and once in throughput.
func TestSessionCopiesAreNotCountedTwice(t *testing.T) {
	f := newUsageFixture(t)
	ended := usageNow.Add(-time.Hour)
	f.run(fixtureRun{id: "orig", mode: "imported", started: ended.Add(-time.Hour), ended: ended, lifetimeTokens: 300})
	f.run(fixtureRun{id: "copy", mode: "imported", started: ended.Add(-time.Hour), ended: ended, lifetimeTokens: 300})
	for _, observed := range []time.Duration{-150 * time.Minute, -100 * time.Minute, -70 * time.Minute} {
		f.request("orig", "k1-orig", "same-session", usageNow.Add(observed), 100)
	}
	// The copy holds the identical requests under its own file key.
	for offset, observed := range []time.Duration{-150 * time.Minute, -100 * time.Minute, -70 * time.Minute} {
		f.exec(`INSERT OR IGNORE INTO transcript_files (file_key, path, root_kind, root_path, harness, session_id, run_id, first_seen_at) VALUES ('k2-copy', '/p/k2', 'codex', '/r', 'codex', 'same-session', 'copy', '2026-10-01T00:00:00Z')`)
		f.exec(`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at, model, turn_index, input_tokens) VALUES ('k2-copy', ?, 0, 'copy', ?, 'gpt-test', ?, 100)`,
			offset, usageNow.Add(observed).UTC().Format(time.RFC3339Nano), offset+1)
	}
	// turn_index of the original rows follows f.seq (1..3), matching the copy.
	filter := usageFilter
	filter.Scope = invocationreadmodel.ScopeOperatorSessions
	usage, err := f.repo.WindowedUsage(context.Background(), filter)
	if err != nil {
		t.Fatal(err)
	}
	runs, err := f.repo.RunMetrics(context.Background(), filter)
	if err != nil {
		t.Fatal(err)
	}
	if usage.TotalTokens != 300 || runs.TotalRuns != 1 {
		t.Fatalf("tokens=%d runs=%d, want 300 tokens and 1 run for one session seen twice", usage.TotalTokens, runs.TotalRuns)
	}
}

// Run time has a non-zero sample size and a median and p90 over executed runs.
func TestRunMetricsReportRunTimePercentilesWithSampleSize(t *testing.T) {
	f := newUsageFixture(t)
	for i, minutes := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		ended := usageNow.Add(-time.Duration(i+1) * time.Hour)
		f.run(fixtureRun{id: "r" + string(rune('a'+i)), mode: "codec_pipe", started: ended.Add(-time.Duration(minutes) * time.Minute), ended: ended})
	}
	f.run(fixtureRun{id: "operator", mode: "imported", started: usageNow.Add(-30 * time.Hour), ended: usageNow.Add(-time.Hour), lifetimeTokens: 1})
	runs, err := f.repo.RunMetrics(context.Background(), usageFilter)
	if err != nil {
		t.Fatal(err)
	}
	if runs.CompletedDurationRuns != 10 || runs.P50DurationMS != 5*60000 || runs.P90DurationMS != 9*60000 {
		t.Fatalf("run time n=%d p50=%.0f p90=%.0f, want n=10 p50=300000 p90=540000 over executed runs only", runs.CompletedDurationRuns, runs.P50DurationMS, runs.P90DurationMS)
	}
}

// Dollars exist only for per-token billed runs, apportioned to the window by
// the share of the run's tokens requested in it.
func TestWindowedUsageDollarsOnlyForMeteredRuns(t *testing.T) {
	f := newUsageFixture(t)
	ended := usageNow.Add(-time.Hour)
	f.run(fixtureRun{id: "metered", mode: "codec_pipe", billing: "metered", started: ended.Add(-48 * time.Hour), ended: ended, lifetimeTokens: 1000, lifetimeCostUSD: 10})
	f.run(fixtureRun{id: "plan", mode: "codec_pipe", billing: "subscription", started: ended.Add(-time.Hour), ended: ended, lifetimeTokens: 1000, lifetimeCostUSD: 10})
	f.request("metered", "k-m", "s-m", usageNow.Add(-30*time.Hour), 600)
	f.request("metered", "k-m", "s-m", usageNow.Add(-2*time.Hour), 400)
	f.request("plan", "k-p", "s-p", usageNow.Add(-2*time.Hour), 1000)
	window := usageNow.Add(-24 * time.Hour)
	filter := invocationreadmodel.Filter{From: &window, To: &usageNow}

	usage, err := f.repo.WindowedUsage(context.Background(), filter)
	if err != nil {
		t.Fatal(err)
	}
	if usage.TotalTokens != 1400 || usage.MeteredCostUSD < 3.99 || usage.MeteredCostUSD > 4.01 {
		t.Fatalf("usage=%+v, want 1400 tokens and $4 (40%% of the metered run's $10; the subscription run has no dollars)", usage)
	}
}

// Breakdown rows pair the runs that ended in the window with the tokens
// requested in it, and rank by tokens.
func TestRunBreakdownTokensFollowRequestTime(t *testing.T) {
	f := newUsageFixture(t)
	f.run(fixtureRun{id: "a", mode: "codec_pipe", runner: "codex", started: usageNow.Add(-40 * 24 * time.Hour), ended: usageNow.Add(-time.Hour), lifetimeTokens: 9_000})
	f.run(fixtureRun{id: "b", mode: "codec_pipe", runner: "claude-code", started: usageNow.Add(-3 * time.Hour), ended: usageNow.Add(-2 * time.Hour), status: "failed", lifetimeTokens: 50})
	f.request("a", "k-a", "s-a", usageNow.Add(-39*24*time.Hour), 8_900)
	f.request("a", "k-a", "s-a", usageNow.Add(-2*time.Hour), 100)
	f.request("b", "k-b", "s-b", usageNow.Add(-150*time.Minute), 50)

	rows, err := f.repo.RunBreakdown(context.Background(), usageFilter, "runner", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Key != "codex" || rows[0].TotalTokens != 100 || rows[0].RunCount != 1 || rows[1].Key != "claude-code" || rows[1].TotalTokens != 50 || rows[1].FailedCount != 1 {
		t.Fatalf("rows=%+v, want codex 100 tokens (not its lifetime 9000) then claude-code 50", rows)
	}
}

// The per-run distribution measures what each run spent inside the window.
func TestRunCostDistributionUsesWindowedTokensPerRun(t *testing.T) {
	f := newUsageFixture(t)
	for i, tokens := range []int{50_000, 200_000, 2_000_000, 30_000_000} {
		id := "run" + string(rune('a'+i))
		f.run(fixtureRun{id: id, mode: "codec_pipe", started: usageNow.Add(-40 * 24 * time.Hour), ended: usageNow.Add(-time.Hour), lifetimeTokens: int64(tokens) * 10})
		f.request(id, "k-"+id, "s-"+id, usageNow.Add(-2*time.Hour), tokens)
		f.request(id, "k-"+id, "s-"+id, usageNow.Add(-35*24*time.Hour), tokens*9)
	}
	f.run(fixtureRun{id: "idle", mode: "codec_pipe", started: usageNow.Add(-2 * time.Hour), ended: usageNow.Add(-time.Hour)})

	dist, err := f.repo.RunCostDistribution(context.Background(), usageFilter)
	if err != nil {
		t.Fatal(err)
	}
	if dist.SampleSize != 4 || dist.P50Tokens != 200_000 || dist.MaxTokens != 30_000_000 {
		t.Fatalf("distribution=%+v, want 4 runs, p50 200000, max 30000000", dist)
	}
	want := map[string]int64{"<100K": 1, "100K-500K": 1, "500K-1M": 0, "1M-5M": 1, "5M-25M": 0, "25M+": 1}
	for _, bucket := range dist.TokenBuckets {
		if bucket.RunCount != want[bucket.Label] {
			t.Fatalf("bucket %q = %d, want %d", bucket.Label, bucket.RunCount, want[bucket.Label])
		}
	}
}

// Fallback insights count the events inside the window by kind, pair and reason.
func TestFallbackInsightsCountsEventsInsideTheWindow(t *testing.T) {
	f := newUsageFixture(t)
	seq := 0
	event := func(at time.Time, eventType, data string) {
		seq++
		f.exec(`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data) VALUES (?, 'r', ?, ?, ?, ?)`, eventType+string(rune('a'+seq)), seq, eventType, at.UTC().Format(time.RFC3339Nano), data)
	}
	in := usageNow.Add(-time.Hour)
	event(in, "runner.fallback.attempted", `{"from":"codex","to":"claude-code","reason":"rate_limit","attempt_no":1}`)
	event(in, "runner.fallback.attempted", `{"from":"codex","to":"claude-code","reason":"rate_limit","attempt_no":1}`)
	event(in, "runner.fallback.exhausted", `{"primary":"codex","last_reason":"unavailable"}`)
	event(in, "model.fallback.attempted", `{"from":"a","to":"b","reason":"model_not_found","attempt_no":1}`)
	event(in, "policy.candidate.attempt", `{"outcome":"failed","failure_class":"timeout"}`)
	event(in, "policy.candidate.attempt", `{"outcome":"attempted"}`)
	event(usageNow.Add(-30*24*time.Hour), "runner.fallback.attempted", `{"from":"x","to":"y","reason":"old","attempt_no":1}`)

	got, err := f.repo.FallbackInsights(context.Background(), usageFilter)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunnerAttempts != 2 || got.RunnerExhausted != 1 || got.ModelAttempts != 1 || got.PolicyCandidateEvents != 2 {
		t.Fatalf("counts = %+v", got)
	}
	if len(got.RunnerPairs) != 1 || got.RunnerPairs[0] != (invocationreadmodel.FallbackPair{From: "codex", To: "claude-code", Reason: "rate_limit", Count: 2}) {
		t.Fatalf("runner pairs = %+v, want the old event excluded", got.RunnerPairs)
	}
	if len(got.RunnerReasons) != 2 || got.RunnerReasons[0] != (invocationreadmodel.CountRow{Key: "rate_limit", Count: 2}) {
		t.Fatalf("runner reasons = %+v", got.RunnerReasons)
	}
	if len(got.PolicyFailureClasses) != 1 || got.PolicyFailureClasses[0].Key != "timeout" || len(got.PolicyOutcomes) != 2 {
		t.Fatalf("policy = outcomes %+v classes %+v", got.PolicyOutcomes, got.PolicyFailureClasses)
	}
}

// The fallback read must use the partial index on its event types, not walk
// every event of the window.
func TestFallbackInsightsWindowUsesThePartialIndex(t *testing.T) {
	f := newUsageFixture(t)
	query := `EXPLAIN QUERY PLAN SELECT event_type, COUNT(*) FROM run_events WHERE event_type IN ` + fallbackEvents + ` AND timestamp >= ? AND timestamp < ? GROUP BY event_type`
	rows, err := f.db.QueryContext(context.Background(), query, "0000", "9999")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(detail, "run_events") {
			if !strings.Contains(detail, "idx_run_events_fallback") {
				t.Fatalf("plan = %q, want idx_run_events_fallback", detail)
			}
			return
		}
	}
	t.Fatal("no run_events step in the query plan")
}
