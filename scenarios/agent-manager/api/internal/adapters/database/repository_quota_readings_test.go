package database

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/usageforecast"

	"github.com/sirupsen/logrus"
)

// Codex repeats one reading on every token_count frame and jitters resets_at
// by seconds; the forecaster's read collapses that to one span per reading.
func TestQuotaReadingSpansCollapseRepeatedFramesPerReading(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPricingRepository(db, logrus.New())
	store := repo.(pricing.QuotaObservationRepository)
	base := time.Date(2026, 10, 29, 1, 0, 0, 0, time.UTC)
	record := func(offset time.Duration, percent float64, resetJitter time.Duration, provider string) {
		t.Helper()
		minutes := int64(10080)
		reset := time.Date(2026, 11, 4, 6, 57, 18, 0, time.UTC).Add(resetJitter)
		o := &pricing.QuotaObservation{Provider: provider, Pool: "primary", Window: "10080m", ObservedAt: base.Add(offset), Standing: pricing.QuotaStandingAvailable, UsedPercent: &percent, WindowMinutes: &minutes, ResetAt: &reset, Provenance: "codex:event_msg.token_count.rate_limits"}
		o.ID = pricing.QuotaObservationID(o)
		if err := store.RecordQuotaObservation(t.Context(), o); err != nil {
			t.Fatal(err)
		}
	}
	record(0, 27, 0, "openai")
	record(30*time.Second, 27, 2*time.Second, "openai")
	record(10*time.Minute, 27, -time.Second, "openai")
	record(20*time.Minute, 28, 3*time.Second, "openai")
	record(-48*time.Hour, 90, 0, "openai")
	record(5*time.Minute, 11, 0, "anthropic")

	spans, err := repo.(pricing.QuotaReadingSource).QuotaReadingSpans(t.Context(), "openai", base.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 {
		t.Fatalf("spans=%+v want one per reading", spans)
	}
	first := spans[0]
	// The 30s repeat is deduplicated at write time; the 10-minute one is a
	// heartbeat.
	if *first.UsedPercent != 27 || first.Samples != 2 || !first.FirstAt.Equal(base) || !first.LastAt.Equal(base.Add(10*time.Minute)) {
		t.Fatalf("first span=%+v", first)
	}
	if !first.ResetAt.Equal(time.Date(2026, 11, 4, 6, 57, 0, 0, time.UTC)) || *first.WindowMinutes != 10080 || first.Standing != pricing.QuotaStandingAvailable {
		t.Fatalf("reset must be truncated to the minute: %+v", first)
	}
	if *spans[1].UsedPercent != 28 {
		t.Fatalf("spans must be ordered by first reading: %+v", spans)
	}
	all, err := repo.(pricing.QuotaReadingSource).QuotaReadingSpans(t.Context(), "", time.Time{})
	if err != nil || len(all) != 4 {
		t.Fatalf("all providers and history: %d %v", len(all), err)
	}
}

// codexFrames is a Codex rollout's token_count stream: one frame every
// `every`, used percent climbing one point per `perPoint`, all under one
// reset time (nil for pre-2025-11 rollouts that reported none).
func codexFrames(pool string, start time.Time, n int, every, perPoint time.Duration, reset *time.Time) []*pricing.QuotaObservation {
	minutes := int64(300)
	out := make([]*pricing.QuotaObservation, 0, n)
	for i := 0; i < n; i++ {
		at := start.Add(time.Duration(i) * every)
		percent := float64(at.Sub(start) / perPoint)
		o := &pricing.QuotaObservation{Provider: "openai", Pool: pool, Window: "300m", ObservedAt: at, Standing: pricing.QuotaStandingAvailable, UsedPercent: &percent, WindowMinutes: &minutes, ResetAt: reset, Provenance: "codex:event_msg.token_count.rate_limits"}
		o.ID = pricing.QuotaObservationID(o)
		out = append(out, o)
	}
	return out
}

func quotaRowCount(t *testing.T, db *DB) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM quota_observations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// SPEC A3 "deduped": a reading is stored when it changes the window state
// and otherwise at most once per heartbeat, also when concurrent sessions'
// files are indexed out of time order (each file is tailed forward).
func TestRecordQuotaObservationStoresStateChangesAndHeartbeatsOnly(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewPricingRepository(db, logrus.New()).(pricing.QuotaObservationRepository)
	start := time.Date(2026, 10, 29, 1, 0, 0, 0, time.UTC)
	reset := start.Add(5 * time.Hour)
	// Two sessions read the same window for 30 minutes, a frame every 20s
	// each, interleaved 10s apart; the percent rises every 10 minutes. The
	// later session's file is indexed first.
	early := codexFrames("primary", start, 90, 20*time.Second, 10*time.Minute, &reset)
	late := codexFrames("primary", start.Add(10*time.Second), 90, 20*time.Second, 10*time.Minute, &reset)
	for _, frame := range append(late, early...) {
		if err := store.RecordQuotaObservation(t.Context(), frame); err != nil {
			t.Fatal(err)
		}
	}
	if got := quotaRowCount(t, db); got < 6 || got > 12 {
		t.Fatalf("rows=%d want 2–4 per reading (first frame and 5-minute heartbeats) from 180 frames", got)
	}
	spans, err := store.(pricing.QuotaReadingSource).QuotaReadingSpans(t.Context(), "", time.Time{})
	if err != nil || len(spans) != 3 {
		t.Fatalf("spans=%d err=%v want one per reading", len(spans), err)
	}
	for i, span := range spans {
		first := start.Add(time.Duration(i) * 10 * time.Minute)
		if !span.FirstAt.Equal(first) || *span.UsedPercent != float64(i) || span.LastAt.Before(first.Add(10*time.Minute-pricing.QuotaHeartbeat)) {
			t.Errorf("span %d %v..%v at %v%% want first frame %v, last within a heartbeat of its end", i, span.FirstAt, span.LastAt, *span.UsedPercent, first)
		}
	}
}

// legacyQuotaRows inserts frames the way rows were stored before write-time
// deduplication: every frame.
func legacyQuotaRows(t *testing.T, db *DB, frames []*pricing.QuotaObservation) {
	t.Helper()
	for _, o := range frames {
		var reset any
		if o.ResetAt != nil {
			reset = SQLiteTime(o.ResetAt.UTC())
		}
		if _, err := db.ExecContext(t.Context(), insertQuotaObservationSQL, o.AccountScope, o.ID, o.Provider, o.Pool, o.Window, SQLiteTime(o.ObservedAt.UTC()), o.Standing, o.Used, o.Limit, o.Remaining, o.UsedPercent, o.WindowMinutes, reset, o.SourceRunID, o.Freshness, o.Provenance, o.Uncertainty, o.EvidenceRef); err != nil {
			t.Fatal(err)
		}
	}
}

// The start-up compaction removes the repeated rows stored before
// deduplication without changing any reading span, so the forecast and its
// backtest read the same history; a second pass deletes nothing.
func TestCompactQuotaObservationsPreservesSpansAndBacktestAndIsIdempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPricingRepository(db, logrus.New())
	source := repo.(pricing.QuotaReadingSource)
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	var frames []*pricing.QuotaObservation
	for d := 0; d < 8; d++ {
		windowStart := start.Add(time.Duration(d) * 24 * time.Hour)
		reset := windowStart.Add(5 * time.Hour)
		// Four hours of frames every 7s, one point per 3 minutes.
		frames = append(frames, codexFrames("primary", windowStart, 4*3600/7, 7*time.Second, 3*time.Minute, &reset)...)
		// The same shape without a reset time, as pre-2025-11 rollouts.
		frames = append(frames, codexFrames("secondary", windowStart, 4*3600/7, 7*time.Second, 3*time.Minute, nil)...)
	}
	legacyQuotaRows(t, db, frames)
	now := start.Add(9 * 24 * time.Hour)
	before, err := source.QuotaReadingSpans(t.Context(), "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	rowsBefore := quotaRowCount(t, db)

	result, err := repo.(pricing.QuotaObservationCompactor).CompactQuotaObservations(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	after, err := source.QuotaReadingSpans(t.Context(), "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != rowsBefore || result.Deleted == 0 || quotaRowCount(t, db) != rowsBefore-result.Deleted {
		t.Fatalf("compaction=%+v rows before=%d after=%d", result, rowsBefore, quotaRowCount(t, db))
	}
	// One first frame per reading, plus heartbeats and last frames: under a
	// tenth of the 7-second frames survive.
	if kept := rowsBefore - result.Deleted; kept*10 > rowsBefore {
		t.Fatalf("kept %d of %d rows", kept, rowsBefore)
	}
	if len(after) != len(before) {
		t.Fatalf("spans %d → %d", len(before), len(after))
	}
	for i := range before {
		b, a := before[i], after[i]
		b.Samples, a.Samples = 0, 0
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("span %d changed:\n%+v\n%+v", i, b, a)
		}
	}
	if b, a := usageforecast.Backtest(before, now), usageforecast.Backtest(after, now); !reflect.DeepEqual(fmt.Sprint(b.Summaries), fmt.Sprint(a.Summaries)) || len(b.Windows) == 0 {
		t.Fatalf("backtest changed:\n%+v\n%+v", b.Summaries, a.Summaries)
	}
	again, err := repo.(pricing.QuotaObservationCompactor).CompactQuotaObservations(t.Context())
	if err != nil || again.Deleted != 0 {
		t.Fatalf("second pass=%+v err=%v must delete nothing", again, err)
	}
}

// Account scopes partition identity, heartbeat deduplication and compaction.
// The legacy forecast has no account dimension, so it must exclude scoped data
// rather than merge it into the existing unknown-account history.
func TestScopedQuotaObservationsRemainIsolated(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPricingRepository(db, logrus.New())
	store := repo.(pricing.QuotaObservationRepository)
	scoped := repo.(pricing.ScopedQuotaObservationRepository)
	base := time.Date(2026, 10, 29, 1, 0, 0, 0, time.UTC)
	reset := base.Add(5 * time.Hour)
	zero := 0.0
	for _, account := range []string{"", "account-a", "account-b"} {
		for _, delta := range []time.Duration{0, time.Minute, 6 * time.Minute} {
			o := &pricing.QuotaObservation{AccountScope: account, Provider: "openai", Pool: "primary", Window: "300m", ObservedAt: base.Add(delta), Standing: pricing.QuotaStandingAvailable, UsedPercent: &zero, ResetAt: &reset, Provenance: "test"}
			if err := store.RecordQuotaObservation(t.Context(), o); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := quotaRowCount(t, db); got != 6 {
		t.Fatalf("stored rows=%d; want two heartbeats for each of three scopes", got)
	}
	for _, account := range []string{"account-a", "account-b"} {
		rows, err := scoped.ListScopedQuotaObservations(t.Context(), account, "openai", "primary", "300m", 10)
		if err != nil || len(rows) != 2 || rows[0].AccountScope != account || rows[0].UsedPercent == nil || *rows[0].UsedPercent != 0 {
			t.Fatalf("account %q rows=%+v err=%v", account, rows, err)
		}
	}
	for _, account := range []string{"account-a", "account-b"} {
		spans, err := repo.(pricing.ScopedQuotaReadingSource).ScopedQuotaReadingSpans(t.Context(), account, "openai", time.Time{})
		if err != nil || len(spans) != 1 || spans[0].AccountScope != account || spans[0].Samples != 2 {
			t.Fatalf("scoped forecast spans for %q: %+v, %v", account, spans, err)
		}
	}
	if _, err := repo.(pricing.ScopedQuotaReadingSource).ScopedQuotaReadingSpans(t.Context(), "", "", time.Time{}); err == nil {
		t.Fatal("empty scope must not select all scoped forecast readings")
	}
	if _, err := scoped.ListScopedQuotaObservations(t.Context(), "", "", "", "", 10); err == nil {
		t.Fatal("empty account scope must not mean all accounts")
	}
	spans, err := repo.(pricing.QuotaReadingSource).QuotaReadingSpans(t.Context(), "openai", time.Time{})
	if err != nil || len(spans) != 1 || spans[0].Samples != 2 {
		t.Fatalf("legacy forecast must see only unscoped history: spans=%+v err=%v", spans, err)
	}
	if _, err := repo.(pricing.QuotaObservationCompactor).CompactQuotaObservations(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, account := range []string{"account-a", "account-b"} {
		rows, err := scoped.ListScopedQuotaObservations(t.Context(), account, "openai", "primary", "300m", 10)
		if err != nil || len(rows) == 0 {
			t.Fatalf("compaction discarded scope %q: %+v, %v", account, rows, err)
		}
	}
}

func TestQuotaScopeMigrationPreservesLegacyUnknownRows(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	// Recreate the pre-scope table shape with existing evidence in place.
	if _, err := db.Exec(`DROP INDEX idx_quota_observations_account_scope_time`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE quota_observations DROP COLUMN account_scope`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO quota_observations (id,provider,pool,window,observed_at,standing,provenance) VALUES ('legacy','openai','primary','300m','2026-10-29T01:00:00Z','available','test')`); err != nil {
		t.Fatal(err)
	}
	if err := db.InitializeSchema(); err != nil {
		t.Fatalf("migrate legacy table: %v", err)
	}
	var scope string
	if err := db.Get(&scope, `SELECT account_scope FROM quota_observations WHERE id='legacy'`); err != nil || scope != "" {
		t.Fatalf("legacy scope=%q, err=%v; want unknown", scope, err)
	}
	if err := db.InitializeSchema(); err != nil {
		t.Fatalf("repeat schema initialization: %v", err)
	}
}

func TestCompactionDoesNotMergeAccountScopes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPricingRepository(db, logrus.New())
	base := time.Date(2026, 10, 29, 1, 0, 0, 0, time.UTC)
	// Insert pre-compaction rows directly to model repeated historical frames.
	for _, account := range []string{"account-a", "account-b"} {
		for i := 0; i < 3; i++ {
			if _, err := db.Exec(`INSERT INTO quota_observations (id,account_scope,provider,pool,window,observed_at,standing,used_percent,provenance) VALUES (?,?,?,?,?,?,?,?,?)`, fmt.Sprintf("%s-%d", account, i), account, "openai", "primary", "300m", SQLiteTime(base.Add(time.Duration(i)*time.Minute)), "available", 20, "test"); err != nil {
				t.Fatal(err)
			}
		}
	}
	result, err := repo.(pricing.QuotaObservationCompactor).CompactQuotaObservations(t.Context())
	if err != nil || result.Deleted != 2 {
		t.Fatalf("compaction=%+v err=%v; want one redundant row per account", result, err)
	}
	for _, account := range []string{"account-a", "account-b"} {
		rows, err := repo.(pricing.ScopedQuotaObservationRepository).ListScopedQuotaObservations(t.Context(), account, "", "", "", 10)
		if err != nil || len(rows) != 2 {
			t.Fatalf("scope %q rows=%+v err=%v; want first and last", account, rows, err)
		}
	}
}
