package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

// TestInvocationReadModelReadersHonourRunScope seeds one Agent Manager run and
// one operator session; with a scope set every reader returns only runs in it,
// and without one both.
func TestInvocationReadModelReadersHonourRunScope(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	shown, hidden := "", ""
	for _, spec := range runsignal.PatternSpecs() {
		if runsignal.Shown(spec.ID) && shown == "" {
			shown = spec.ID
		} else if !runsignal.Shown(spec.ID) && hidden == "" {
			hidden = spec.ID
		}
	}
	if shown == "" || hidden == "" {
		t.Skip("need one shown and one hidden pattern")
	}
	seed := func(id, mode string) {
		if _, err := db.ExecContext(ctx, `INSERT INTO runs (id, execution_mode, status) VALUES (?, ?, 'complete')`, id, mode); err != nil {
			t.Fatal(err)
		}
		started, ended := now, now.Add(time.Minute)
		episode := func(suffix, pattern string) invocationreadmodel.Episode {
			return invocationreadmodel.Episode{FrictionEpisode: runsignal.FrictionEpisode{EpisodeID: id + suffix, Pattern: pattern, CauseScope: "toolchain", Fingerprint: id + suffix}, RunID: id, OccurredAt: now, TimeBasis: "call_event", RunStatus: "complete"}
		}
		fact := invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: id + "-call", ToolName: "shell", Ownership: "external", Outcome: "success", Fingerprint: id, Availability: "available"}, RunID: id, OccurredAt: now, TimeBasis: "call_event", RunStatus: "complete"}
		run := invocationreadmodel.RunFact{RunID: id, OccurredAt: ended, CreatedAt: started, StartedAt: &started, EndedAt: &ended, DurationMS: 60000, Status: "complete", TotalTokens: 10, CostTimeBasis: "ended_at", ProjectedAt: now}
		if err := repo.ReplaceProjection(ctx, []invocationreadmodel.Fact{fact}, nil, []invocationreadmodel.Episode{episode("-shown", shown), episode("-hidden", hidden)}, nil, invocationreadmodel.Watermark{RunID: id, LastEventID: id + "-call", LastEventAt: now, ClassifierVersion: "v1", ProjectedAt: now}, run); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO run_findings (id,run_id,investigation_run_id,category,severity,recommendation_text,evidence,target_path,fingerprint,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, uuid.New().String(), id, id, "efficiency", "warning", "fixture", "", "", "fp-"+id, SQLiteTime(now.Add(time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	seed("in-scope", "headless")
	seed("out-of-scope", "interactive")

	base := invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}
	scoped := base
	scoped.Scope = invocationreadmodel.ScopeAgentManager
	check := func(name string, f invocationreadmodel.Filter, want int) {
		t.Helper()
		ids := func(c invocationreadmodel.Cohort, err error) int {
			if err != nil {
				t.Fatal(err)
			}
			return c.MatchedRuns
		}
		if got := ids(repo.Cohort(ctx, f, 10)); got != want {
			t.Errorf("%s fact cohort runs=%d want %d", name, got, want)
		}
		episodeFilter := f
		episodeFilter.EpisodePattern = shown
		c, err := repo.Cohort(ctx, episodeFilter, 10)
		if err != nil || c.MatchedRuns != want {
			t.Errorf("%s episode cohort=%+v err=%v want runs %d", name, c, err, want)
		}
		episodes, err := repo.Episodes(ctx, f, 10)
		if err != nil || len(episodes) != want {
			t.Errorf("%s episodes=%d err=%v want %d", name, len(episodes), err, want)
		}
		hiddenCount, err := repo.HiddenEpisodes(ctx, f)
		if err != nil || hiddenCount != want {
			t.Errorf("%s hidden=%d err=%v want %d", name, hiddenCount, err, want)
		}
		findings, err := repo.FindingMetrics(ctx, f)
		if err != nil || findings.TotalFindings != int64(want) {
			t.Errorf("%s findings=%+v err=%v want %d", name, findings, err, want)
		}
	}
	check("scoped", scoped, 1)
	check("unscoped", base, 2)
	cohort, err := repo.Cohort(ctx, scoped, 10)
	if err != nil || len(cohort.RunIDs) != 1 || cohort.RunIDs[0] != "in-scope" {
		t.Fatalf("scoped cohort=%+v err=%v", cohort, err)
	}
}

// TestInvocationReadModelEpisodeReadsHonourWorkload: episodes carry no
// workload column, so a workload filter must reach them through their run.
func TestInvocationReadModelEpisodeReadsHonourWorkload(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	shown, hidden := "", ""
	for _, spec := range runsignal.PatternSpecs() {
		if runsignal.Shown(spec.ID) && shown == "" {
			shown = spec.ID
		} else if !runsignal.Shown(spec.ID) && hidden == "" {
			hidden = spec.ID
		}
	}
	if shown == "" || hidden == "" {
		t.Skip("need one shown and one hidden pattern")
	}
	for _, workload := range []string{"alpha", "beta"} {
		id := "run-" + workload
		if _, err := db.ExecContext(ctx, `INSERT INTO runs (id, execution_mode, status) VALUES (?, 'headless', 'complete')`, id); err != nil {
			t.Fatal(err)
		}
		started, ended := now, now.Add(time.Minute)
		episode := func(suffix, pattern string) invocationreadmodel.Episode {
			return invocationreadmodel.Episode{FrictionEpisode: runsignal.FrictionEpisode{EpisodeID: id + suffix, Pattern: pattern, CauseScope: "toolchain", Fingerprint: id + suffix}, RunID: id, OccurredAt: now, TimeBasis: "call_event", RunStatus: "complete"}
		}
		fact := invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: id + "-call", ToolName: "shell", Ownership: "external", Outcome: "success", Fingerprint: id, Availability: "available"}, RunID: id, OccurredAt: now, TimeBasis: "call_event", RunStatus: "complete"}
		run := invocationreadmodel.RunFact{RunID: id, OccurredAt: ended, CreatedAt: started, StartedAt: &started, EndedAt: &ended, DurationMS: 60000, Status: "complete", TotalTokens: 10, CostTimeBasis: "ended_at", ProjectedAt: now}
		if err := repo.ReplaceProjection(ctx, []invocationreadmodel.Fact{fact}, nil, []invocationreadmodel.Episode{episode("-shown", shown), episode("-hidden", hidden)}, nil, invocationreadmodel.Watermark{RunID: id, LastEventID: id + "-call", LastEventAt: now, ClassifierVersion: "v1", ProjectedAt: now}, run); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE invocation_read_model_runs SET workload_kind='kind-'||?, workload_key=? WHERE run_id=?`, workload, workload, id); err != nil {
			t.Fatal(err)
		}
	}
	window := invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}
	if all, err := repo.Episodes(ctx, window, 10); err != nil || len(all) != 2 {
		t.Fatalf("unscoped episodes=%d err=%v want 2", len(all), err)
	}
	for _, f := range []invocationreadmodel.Filter{
		func() invocationreadmodel.Filter { c := window; c.WorkloadKey = "alpha"; return c }(),
		func() invocationreadmodel.Filter { c := window; c.WorkloadKind = "kind-alpha"; return c }(),
	} {
		episodes, err := repo.Episodes(ctx, f, 10)
		if err != nil || len(episodes) != 1 || episodes[0].RunID != "run-alpha" {
			t.Errorf("workload episodes=%+v err=%v want only run-alpha", episodes, err)
		}
		if n, err := repo.HiddenEpisodes(ctx, f); err != nil || n != 1 {
			t.Errorf("workload hidden=%d err=%v want 1", n, err)
		}
	}
}
