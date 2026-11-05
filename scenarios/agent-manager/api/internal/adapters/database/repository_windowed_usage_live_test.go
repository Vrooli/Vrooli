//go:build livecheck

package database

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"

	"github.com/jmoiron/sqlx"
)

// TestLiveWindowedUsageMatchesTranscriptSum compares the seven-day token total
// Details reports with a plain sum over transcript_usage for the same window,
// on the live database opened read-only (P2-A1: within 1%). Opt in with
// WINDOWED_USAGE_LIVE=1; WINDOWED_USAGE_LIVE_DB overrides the path.
func TestLiveWindowedUsageMatchesTranscriptSum(t *testing.T) {
	if os.Getenv("WINDOWED_USAGE_LIVE") != "1" {
		t.Skip("set WINDOWED_USAGE_LIVE=1 to compare windowed usage with the live database")
	}
	live := os.Getenv("WINDOWED_USAGE_LIVE_DB")
	if live == "" {
		live = filepath.Join("..", "..", "..", "..", "data", "agent-manager.db")
	}
	db, err := sqlx.Connect("sqlite", "file:"+live+"?mode=ro&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := &invocationReadModelRepository{db: &DB{DB: db}}

	to := time.Now().UTC()
	from := to.Add(-7 * 24 * time.Hour)
	filter := invocationreadmodel.Filter{From: &from, To: &to}
	usage, err := repo.WindowedUsage(context.Background(), filter)
	if err != nil {
		t.Fatal(err)
	}
	var raw int64
	if err := db.Get(&raw, `SELECT COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens), 0) FROM transcript_usage WHERE observed_at >= ? AND observed_at < ?`,
		from.Format(time.RFC3339Nano), to.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	var lifetime int64
	if err := db.Get(&lifetime, `SELECT COALESCE(SUM(total_tokens), 0) FROM invocation_read_model_runs WHERE occurred_at >= ? AND occurred_at < ?`, from.Format(time.RFC3339Nano), to.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	runs, err := repo.RunMetrics(context.Background(), filter)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("7d tokens: windowed=%d transcript_usage SUM=%d (old lifetime sum at ended_at=%d); requests=%d runs=%d; AM runs ended=%d durations n=%d p50=%.0fms p90=%.0fms",
		usage.TotalTokens, raw, lifetime, usage.Requests, usage.Runs, runs.TotalRuns, runs.CompletedDurationRuns, runs.P50DurationMS, runs.P90DurationMS)
	ctx := context.Background()
	timed := func(name string) func() {
		start := time.Now()
		return func() { t.Logf("  %s took %s", name, time.Since(start).Round(time.Millisecond)) }
	}
	for _, dimension := range []string{"runner", "model", "profile", "workload", "workload_kind"} {
		done := timed("breakdown " + dimension)
		rows, err := repo.RunBreakdown(ctx, filter, dimension, 5)
		done()
		if err != nil {
			t.Fatalf("breakdown %s: %v", dimension, err)
		}
		var top string
		if len(rows) > 0 {
			top = rows[0].Value
		}
		t.Logf("breakdown %-13s rows=%d top=%q tokens=%d", dimension, len(rows), top, func() int64 {
			if len(rows) == 0 {
				return 0
			}
			return rows[0].TotalTokens
		}())
	}
	for _, scope := range []invocationreadmodel.RunScope{invocationreadmodel.ScopeAgentManager, invocationreadmodel.ScopeOperatorSessions, invocationreadmodel.ScopeAll} {
		scoped := filter
		scoped.Scope = scope
		done := timed("run metrics " + string(scope))
		m, err := repo.RunMetrics(ctx, scoped)
		done()
		if err != nil {
			t.Fatalf("scope %s: %v", scope, err)
		}
		u, err := repo.WindowedUsage(ctx, scoped)
		if err != nil {
			t.Fatalf("scope %s usage: %v", scope, err)
		}
		t.Logf("scope %-17s runs=%d terminal=%d tokens=%d", scope, m.TotalRuns, m.TerminalRuns, u.TotalTokens)
	}
	done := timed("distribution")
	dist, err := repo.RunCostDistribution(ctx, filter)
	done()
	done = timed("token attribution")
	if err != nil {
		t.Fatal(err)
	}
	attribution, err := repo.TokenAttribution(ctx, filter, "capability", "footprint", 5)
	if err != nil {
		t.Fatal(err)
	}
	done()
	fallback, err := repo.FallbackInsights(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("distribution runs=%d p50=%d p90=%d; attribution rows=%d; fallback policy events=%d", dist.SampleSize, dist.P50Tokens, dist.P90Tokens, len(attribution), fallback.PolicyCandidateEvents)
	if raw == 0 || math.Abs(float64(usage.TotalTokens-raw))/float64(raw) > 0.01 {
		t.Fatalf("windowed total %d differs from transcript_usage sum %d by more than 1%%", usage.TotalTokens, raw)
	}
}
