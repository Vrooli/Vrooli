//go:build livecheck

package database

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/usageforecast"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// TestLiveQuotaBacktestAndCompaction is the SPEC A3/A4 live proof without a
// restart. It opens the live Agent Manager database read-only, copies
// quota_observations into a temporary database, prints the forecast backtest,
// runs the real start-up compaction on the copy and proves the backtest is
// unchanged. Opt in with AM_QUOTA_LIVE_BACKTEST=1 (AM_LIVE_DB overrides the
// path).
func TestLiveQuotaBacktestAndCompaction(t *testing.T) {
	if os.Getenv("AM_QUOTA_LIVE_BACKTEST") != "1" {
		t.Skip("set AM_QUOTA_LIVE_BACKTEST=1 to backtest the live quota readings on a temporary copy")
	}
	live := os.Getenv("AM_LIVE_DB")
	if live == "" {
		live = filepath.Join(DataDir(), "agent-manager.db")
	}
	copyPath := filepath.Join(t.TempDir(), "quota.db")
	raw, err := sqlx.Open("sqlite", "file:"+copyPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	defer raw.Close()
	started := time.Now()
	for _, stmt := range []string{
		`ATTACH DATABASE 'file:` + live + `?mode=ro' AS live`,
		`CREATE TABLE quota_observations AS SELECT * FROM live.quota_observations`,
		`DETACH DATABASE live`,
		`CREATE UNIQUE INDEX quota_observations_id ON quota_observations(id)`,
		`CREATE INDEX idx_quota_observations_scope_time ON quota_observations(provider, pool, window, observed_at DESC)`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	db := NewDB(raw, logrus.New())
	repo := NewPricingRepository(db, logrus.New())
	rows, bytes := liveQuotaFootprint(t, raw)
	t.Logf("copied %d quota_observations rows (%.1f MiB table+indexes) in %v", rows, float64(bytes)/(1<<20), time.Since(started).Round(time.Second))

	now := time.Now().UTC()
	before, err := repo.(pricing.QuotaReadingSource).QuotaReadingSpans(t.Context(), "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	report := usageforecast.Backtest(before, now)
	t.Logf("backtest over %d spans, %d completed windows, %d unobserved:\n%s", len(before), len(report.Windows), report.Unobserved, backtestTable(report))
	t.Logf("last 5 windows per scope at 25%%:\n%s", recentWindows(report))
	for _, s := range report.Summaries {
		if s.Windows > 0 && !s.BeatsBaseline {
			t.Errorf("%s @%.0f%%: forecast %.2f pts does not beat the baseline %.2f", s.Population, s.Fraction*100, s.MedianAbsErrorPoints, s.BaselineMedianAbsErrorPts)
		}
	}

	started = time.Now()
	result, err := repo.(pricing.QuotaObservationCompactor).CompactQuotaObservations(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	keptRows, keptBytes := liveQuotaFootprint(t, raw)
	t.Logf("compaction scanned %d, deleted %d rows in %v: %d rows remain; table+indexes %.1f → %.1f MiB (%.1f MiB reclaimable)",
		result.Scanned, result.Deleted, time.Since(started).Round(time.Second), keptRows, float64(bytes)/(1<<20), float64(keptBytes)/(1<<20), float64(bytes-keptBytes)/(1<<20))
	after, err := repo.(pricing.QuotaReadingSource).QuotaReadingSpans(t.Context(), "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	compacted := usageforecast.Backtest(after, now)
	if !reflect.DeepEqual(fmt.Sprint(report.Summaries), fmt.Sprint(compacted.Summaries)) || len(after) != len(before) {
		t.Fatalf("compaction changed the backtest:\nbefore %+v\nafter  %+v", report.Summaries, compacted.Summaries)
	}
	t.Logf("backtest after compaction is identical (%d spans before and after)", len(after))
	again, err := repo.(pricing.QuotaObservationCompactor).CompactQuotaObservations(t.Context())
	if err != nil || again.Deleted != 0 {
		t.Fatalf("second compaction=%+v err=%v must delete nothing", again, err)
	}
}

// liveQuotaFootprint counts rows and the pages the table and its indexes
// occupy (dbstat), the bytes a compaction frees for incremental vacuum.
func liveQuotaFootprint(t *testing.T, raw *sqlx.DB) (int64, int64) {
	t.Helper()
	var rows, bytes int64
	if err := raw.QueryRow(`SELECT COUNT(*) FROM quota_observations`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRow(`SELECT COALESCE(SUM(pgsize),0) FROM dbstat WHERE name IN ('quota_observations','quota_observations_id','idx_quota_observations_scope_time')`).Scan(&bytes); err != nil {
		t.Logf("dbstat unavailable: %v", err)
	}
	return rows, bytes
}

func backtestTable(r usageforecast.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-26s %5s %7s %10s %10s %6s %9s %9s\n", "population", "at", "windows", "model pts", "linear pts", "beats", "model h", "linear h")
	for _, s := range r.Summaries {
		fmt.Fprintf(&b, "%-26s %4.0f%% %7d %10.2f %10.2f %6v %9.1f %9.1f (n=%d)\n", s.Population, s.Fraction*100, s.Windows, s.MedianAbsErrorPoints, s.BaselineMedianAbsErrorPts, s.BeatsBaseline, s.MedianAbsErrorHours, s.BaselineMedianAbsErrorHrs, s.HourErrorWindows)
	}
	return b.String()
}

func recentWindows(r usageforecast.Report) string {
	var b strings.Builder
	for _, w := range r.Windows {
		if !w.OutcomeObserved || len(w.Checkpoints) == 0 {
			continue
		}
		fmt.Fprintf(&b, "  %s %s → %s final %.0f%%", w.Window.Scope(), w.Window.Start.Format("2006-01-02T15:04"), w.EndedAt.Format("01-02T15:04"), w.Final)
		for _, c := range w.Checkpoints {
			fmt.Fprintf(&b, " | @%.0f%% used %.0f model %.1f linear %.1f k=%.2f(n=%d)", c.Fraction*100, c.UsedAt, c.PredictedFinal, c.BaselineFinal, c.Calibration.Factor, c.Calibration.Windows)
		}
		b.WriteString("\n")
	}
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	byScope := map[string][]string{}
	var order []string
	for _, line := range lines {
		scope := strings.Fields(line)[0]
		if _, ok := byScope[scope]; !ok {
			order = append(order, scope)
		}
		byScope[scope] = append(byScope[scope], line)
	}
	var out []string
	for _, scope := range order {
		scoped := byScope[scope]
		out = append(out, scoped[max(0, len(scoped)-usageforecast.ScoredWindowsPerScope):]...)
	}
	return strings.Join(out, "\n")
}
