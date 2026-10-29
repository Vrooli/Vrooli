//go:build livecheck

package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/pricing"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageattribution"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// TestLiveUsageAttributionDryRun copies this host's recent usage facts from
// the live database (opened read-only) into a throwaway database, applies the
// production usage repair there, and prints what `usage burners` and `usage
// recommendations` would report. The live database is never written. Opt in
// with USAGE_ATTRIBUTION_LIVE=1; USAGE_ATTRIBUTION_LIVE_DB overrides the path.
func TestLiveUsageAttributionDryRun(t *testing.T) {
	if os.Getenv("USAGE_ATTRIBUTION_LIVE") != "1" {
		t.Skip("set USAGE_ATTRIBUTION_LIVE=1 to attribute the live database's usage in a temp copy")
	}
	live := os.Getenv("USAGE_ATTRIBUTION_LIVE_DB")
	if live == "" {
		live = filepath.Join("..", "..", "..", "..", "data", "agent-manager.db")
	}
	ctx := context.Background()
	now := time.Now().UTC()
	cutoff := now.Add(-9 * 24 * time.Hour)

	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	liveDB, err := sqlx.Connect("sqlite", "file:"+live+"?mode=ro&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer liveDB.Close()
	source := &DB{DB: liveDB, log: log}

	tempPath := os.Getenv("USAGE_ATTRIBUTION_LIVE_KEEP") // keep the repaired copy for inspection
	if tempPath == "" {
		tempPath = filepath.Join(t.TempDir(), "usage.db")
	}
	_ = os.Remove(tempPath)
	tempDB, err := sqlx.Connect("sqlite", "file:"+tempPath+"?_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer tempDB.Close()
	temp := &DB{DB: tempDB, log: log}
	if err := temp.initSchema(); err != nil {
		t.Fatal(err)
	}
	if _, err := temp.ExecContext(ctx, `ATTACH DATABASE ? AS src`, "file:"+live+"?mode=ro"); err != nil {
		t.Fatal(err)
	}
	since := cutoff.Format(time.RFC3339Nano)
	copies := []struct{ table, where string }{
		{"transcript_files", fmt.Sprintf("owner = 'run' OR mtime_unix_nano >= %d", cutoff.UnixNano())},
		{"transcript_usage", fmt.Sprintf("file_key IN (SELECT file_key FROM src.transcript_files WHERE owner = 'indexer' AND mtime_unix_nano >= %d)", cutoff.UnixNano())},
		{"agent_profiles", "1"},
		{"runs", "1"},
		{"run_events", "timestamp >= '" + since + "' AND event_type IN ('metric', 'tool_call')"},
		{"invocation_read_model_episodes", "occurred_at >= '" + since + "'"},
		{"invocation_read_model_runs", "occurred_at >= '" + now.Add(-15*24*time.Hour).Format(time.RFC3339Nano) + "'"},
		{"model_pricing", "1"},
		{"model_aliases", "1"},
	}
	for _, c := range copies {
		copied, err := copyLiveTable(ctx, temp, c.table, c.where)
		if err != nil {
			t.Fatalf("copy %s: %v", c.table, err)
		}
		t.Logf("copied %s: %d rows", c.table, copied)
	}

	var unknown, rows int
	_ = temp.GetContext(ctx, &rows, `SELECT COUNT(*) FROM transcript_usage`)
	_ = temp.GetContext(ctx, &unknown, `SELECT COUNT(*) FROM transcript_usage WHERE model IN ('', 'unknown')`)
	t.Logf("usage rows %d, still unknown model %d", rows, unknown)

	readings := NewPricingRepository(source, log).(pricing.QuotaReadingSource)
	pricer := usageattribution.NewPriceBook(pricing.NewServiceWithModelResolver(NewPricingRepository(temp, log), nil, log, pricing.NewCLIModelResolver()))
	before, err := usageattribution.Service{Store: NewUsageAttributionRepository(temp), Readings: readings, Pricer: pricer}.Current(ctx, "", time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	// D-16: index the run-owned files (Agent Manager run homes) with the
	// production indexer, summaries only, so their usage joins the one source.
	started := time.Now()
	indexed, err := indexRunOwnedFiles(ctx, temp, now)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("run-owned files: %s in %s", indexed, time.Since(started).Round(time.Second))
	after, err := usageattribution.Service{Store: NewUsageAttributionRepository(temp), Readings: readings, Pricer: pricer}.Current(ctx, "", time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	for i, result := range after.Results {
		if i >= len(before.Results) {
			break
		}
		b := before.Results[i]
		t.Logf("D-16 %s %s %s observed %.1f pts: before attributed %.2f + unattributed %.2f = %.2f → after attributed %.2f + unattributed %.2f = %.2f",
			result.Window.Provider, result.Window.Pool, result.Window.Name, result.Observed, b.Attributed(), b.Unattributed, b.Attributed()+b.Unattributed,
			result.Attributed(), result.Unattributed, result.Attributed()+result.Unattributed)
	}

	service := usageattribution.Service{
		Store:    NewUsageAttributionRepository(temp),
		Readings: readings,
		Pricer:   pricer,
	}
	started = time.Now()
	attribution, err := service.Current(ctx, "", time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("attribution in %s", time.Since(started).Round(time.Millisecond))
	for _, result := range attribution.Results {
		var sum float64
		for _, p := range result.Points {
			sum += p
		}
		c := result.Calibration
		t.Logf("== %s %s %s  %s → %s  observed %.1f pts = attributed %.2f + unattributed %.2f (Σ turns %.2f)  turns %d  calibration %s r²=%.2f slices=%d off-disk %.3f pts/h %s",
			result.Window.Provider, result.Window.Pool, result.Window.Name, result.From.Format(time.RFC3339), result.To.Format(time.RFC3339),
			result.Observed, result.Attributed(), result.Unattributed, sum, len(result.Turns), c.Method, c.RSquared, c.Slices, c.OffDiskPerHour, c.Note)
		for _, w := range c.Weights {
			t.Logf("   weight %s/%s cached/pt %.0f in×%.1f cc×%.1f out×%.1f (%s, %d slices)", w.Runner, w.Model, w.CachedPerPoint, w.InputWeight, w.CreationWeight, w.OutWeight, w.Source, w.Slices)
		}
		for _, by := range usageattribution.Dimensions {
			burners := usageattribution.Group(result, by, attribution.Runs, "")
			var total float64
			for _, b := range burners {
				total += b.Points
			}
			lines := []string{}
			for i, b := range burners {
				if i >= 5 {
					break
				}
				lines = append(lines, fmt.Sprintf("%s ≈%.2f (%s, %d runs)", trimLabel(b.Label), b.Points, b.Model, b.Runs))
			}
			t.Logf("   by %-9s Σ %.2f + unattributed %.2f = %.2f | %s", by, total, result.Unattributed, total+result.Unattributed, strings.Join(lines, " · "))
		}
		for _, line := range usageattribution.MeteredLines(result) {
			t.Logf("   metered %s ×%d %s %s", line.Kind, line.Count, line.State, line.Note)
		}
	}
	recs, _, err := service.Recommendations(ctx, "", usageattribution.WindowKey{}, now)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("dominant consumer: %s (%s)", recs.Dominant, recs.DominantLabel)
	for _, rule := range recs.Rules {
		t.Logf("rule %-10s fired=%v rows=%d candidates=%d %s", rule.Rule, rule.Fired, rule.Rows, rule.Candidates, rule.Reason)
	}
	t.Logf("%d plans, %d folded (<0.1 pts)", len(recs.Plans), len(recs.Folded))
	for i, plan := range recs.Plans {
		if i >= 5 {
			break
		}
		var sum float64
		for _, l := range plan.Levers {
			sum += l.Saved
		}
		t.Logf("#%d %s %s on %s (%s %s): combined −%.2f pts (levers alone sum −%.2f) | used %.2f so far, %.2f at this rate, %.0f%% of the window's burn, running=%v, recoverable %.2f in the %.0fh left, cross-cut=%v",
			plan.Rank, plan.TargetKind, trimLabel(plan.TargetLabel), plan.TargetID, plan.Provider, plan.Window, plan.Saved, sum, plan.Use.ConsumerPoints, plan.Use.AtThisRate, plan.Use.Share*100, plan.Use.StillRunning, plan.Recoverable, plan.Use.RemainingHours, plan.CrossCut)
		for _, l := range plan.Levers {
			t.Logf("     %-10s alone −%.2f, after larger levers −%.2f [%s] %s", l.Rule, l.Saved, l.Marginal, l.Confidence, trimLabel(l.Action))
		}
	}
	friction, err := service.Friction(ctx, "", now)
	if err != nil {
		t.Fatal(err)
	}
	problems := friction.Problems
	t.Logf("%d recurring friction problems; runs analysed %d of %d", len(problems), friction.RunsAnalysed, friction.RunsTotal)
	for i, p := range problems {
		if i >= 8 {
			break
		}
		t.Logf("friction %s %s | owner %s (%s) | %d episodes in %d runs | typical %s total %s | ≈%.2f pts | %s", p.Fingerprint[:8], p.Description, p.Owner, p.OwnerKind, p.Episodes, p.Runs, usageattribution.FormatDuration(p.TypicalMS), usageattribution.FormatDuration(p.WallClockMS), p.Points, p.Status)
	}
}

// indexRunOwnedFiles runs the production indexer over this host's Agent
// Manager run homes into temp. The operator roots are left out (their rows
// were copied); the host records nothing, as for any run-owned file.
func indexRunOwnedFiles(ctx context.Context, temp *DB, now time.Time) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	owned, err := (&runRepository{db: temp}).ListRunTranscriptPaths(ctx)
	if err != nil {
		return "", err
	}
	host := &runOwnedHost{owned: map[string]string{}}
	for path, id := range owned {
		host.owned[path] = id.String()
	}
	index := transcriptindex.New(NewTranscriptIndexRepository(temp), host, transcriptindex.Config{
		Roots: transcriptindex.RootConfig{Home: filepath.Join(os.TempDir(), "no-operator-roots"), RunStateRoot: filepath.Join(home, ".vrooli", "state", "vrooli", "agent-manager", "runs")},
		Now:   func() time.Time { return now },
	})
	var total transcriptindex.Summary
	for tick := 0; tick < 1000; tick++ {
		summary, err := index.RunOnce(ctx)
		if err != nil {
			return "", err
		}
		total.Read += summary.Read
		total.BytesRead += summary.BytesRead
		total.Failures = append(total.Failures, summary.Failures...)
		if summary.Deferred == 0 {
			break
		}
	}
	var files, reports int64
	_ = temp.GetContext(ctx, &files, `SELECT COUNT(*) FROM transcript_files WHERE owner = 'run'`)
	_ = temp.GetContext(ctx, &reports, `SELECT COUNT(*) FROM transcript_usage u JOIN transcript_files f ON f.file_key = u.file_key WHERE f.owner = 'run'`)
	return fmt.Sprintf("%d files, %d usage rows, %.2f GB read, %d host writes, failures %v", files, reports, float64(total.BytesRead)/1e9, host.writes, total.Failures), nil
}

// runOwnedHost is the indexer's host for run-owned files only: the indexer
// asks it for parsers and must record nothing through it.
type runOwnedHost struct {
	owned  map[string]string
	writes int
}

func (h *runOwnedHost) OwnedTranscripts(context.Context) (map[string]string, error) {
	return h.owned, nil
}

func (h *runOwnedHost) Adopt(_ context.Context, file transcriptindex.FileRow, _ string) (transcriptindex.Adoption, error) {
	h.writes++
	return transcriptindex.Adoption{Excluded: true}, nil
}

func (h *runOwnedHost) Parser(_ context.Context, file transcriptindex.FileRow) (runner.TranscriptParser, error) {
	if transcriptindex.Family(file.Harness) == transcriptindex.HarnessClaude {
		return codecs.NewClaudeForTest().NewTranscriptParser(), nil
	}
	return codecs.NewCodexForTest().NewTranscriptParser(), nil
}

func (h *runOwnedHost) Record(context.Context, string, []*domain.RunEvent, bool) error {
	h.writes++
	return nil
}

func (h *runOwnedHost) UpdateRun(context.Context, string, transcriptindex.RunUpdate) error {
	h.writes++
	return nil
}

func (h *runOwnedHost) DropRecall(context.Context, string) error {
	h.writes++
	return nil
}

func trimLabel(value string) string {
	if len(value) > 140 {
		return value[:140] + "…"
	}
	return value
}

// copyLiveTable copies the columns both schemas share, so additive columns
// in either database do not break the copy.
func copyLiveTable(ctx context.Context, temp *DB, table, where string) (int64, error) {
	var mainColumns, srcColumns []string
	if err := temp.SelectContext(ctx, &mainColumns, `SELECT name FROM pragma_table_info(?)`, table); err != nil {
		return 0, err
	}
	if err := temp.SelectContext(ctx, &srcColumns, `SELECT name FROM src.pragma_table_info(?)`, table); err != nil {
		return 0, err
	}
	shared := map[string]bool{}
	for _, column := range srcColumns {
		shared[column] = true
	}
	var columns []string
	for _, column := range mainColumns {
		if shared[column] {
			columns = append(columns, `"`+column+`"`)
		}
	}
	list := strings.Join(columns, ", ")
	result, err := temp.ExecContext(ctx, `INSERT OR IGNORE INTO main.`+table+` (`+list+`) SELECT `+list+` FROM src.`+table+` WHERE `+where)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
