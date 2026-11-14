package skills

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
	"prompt-manager/internal/metrics"
	"prompt-manager/internal/skillreads"
	"prompt-manager/internal/store"
)

type fakeUsage struct{ counts map[string]int }

func (f *fakeUsage) RecordUsage(skillID string) (int, time.Time, error) {
	if f.counts == nil {
		f.counts = map[string]int{}
	}
	f.counts[skillID]++
	return f.counts[skillID], time.Time{}, nil
}

func attributedRequest(t *testing.T, kind, memberID string) *http.Request {
	t.Helper()
	info := store.AttributionInfo{Kind: kind}
	if memberID != "" {
		info.MemberID = &memberID
	}
	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal attribution: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/skills/read", nil)
	r.Header.Set(store.AttributionHeaderName, base64.StdEncoding.EncodeToString(raw))
	return r
}

// newRecorderAt builds a recorder whose clock is pinned, so discovery-window
// boundaries are exercised deterministically.
func newRecorderAt(t *testing.T, now time.Time) (*ReadRecorder, *skillreads.Service, *store.DiscoveryCallStore, *fakeUsage) {
	t.Helper()
	dir := t.TempDir()
	reads := newSkillReadService(t)
	calls := store.NewDiscoveryCallStore(dir)
	usage := &fakeUsage{}
	rec := NewReadRecorder(reads, calls, usage)
	rec.now = func() time.Time { return now }
	return rec, reads, calls, usage
}

func TestRecordWritesOneEntryPerResolvedSkillAndCountsUsage(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	rec, reads, _, usage := newRecorderAt(t, now)

	rec.Record(attributedRequest(t, "agent-member", "skill-optimizer"), []string{"alpha", "beta"})

	entries, err := reads.ReadSince(0)
	if err != nil {
		t.Fatalf("ReadSince: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if usage.counts["alpha"] != 1 || usage.counts["beta"] != 1 {
		t.Fatalf("usage counts = %v, want one each", usage.counts)
	}
	// Kind must be recorded separately from the display caller: it is what lets
	// a consumer separate lane demand from audit traffic.
	if entries[0].CallerKind != "agent-member" {
		t.Fatalf("callerKind = %q, want agent-member", entries[0].CallerKind)
	}
	if entries[0].Caller != "agent-member/skill-optimizer" {
		t.Fatalf("caller = %q", entries[0].Caller)
	}
}

func TestRecordCreditsDiscoveryOnlyForTheSameCallerInsideTheWindow(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	cases := []struct {
		name       string
		callCaller string
		callAt     time.Time
		wantVia    bool
	}{
		{"same caller, recent", "agent-member/skill-optimizer", now.Add(-30 * time.Minute), true},
		{"same caller, stale", "agent-member/skill-optimizer", now.Add(-3 * time.Hour), false},
		{"different caller, recent", "agent-member/debt-curator", now.Add(-30 * time.Minute), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, reads, calls, _ := newRecorderAt(t, now)
			if err := calls.Append(store.DiscoveryCall{
				ID:      "call-1",
				At:      tc.callAt.Format(time.RFC3339),
				Caller:  tc.callCaller,
				Results: []store.DiscoveryCallResult{{ID: "alpha"}},
			}); err != nil {
				t.Fatalf("append call: %v", err)
			}

			rec.Record(attributedRequest(t, "agent-member", "skill-optimizer"), []string{"alpha"})

			entries, err := reads.ReadSince(0)
			if err != nil {
				t.Fatalf("ReadSince: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("entries = %d, want 1", len(entries))
			}
			if entries[0].ViaDiscovery != tc.wantVia {
				t.Fatalf("viaDiscovery = %v, want %v", entries[0].ViaDiscovery, tc.wantVia)
			}
			if tc.wantVia && entries[0].DiscoveryCallID != "call-1" {
				t.Fatalf("discoveryCallId = %q, want call-1", entries[0].DiscoveryCallID)
			}
		})
	}
}

// An unattributed read must never be credited to a discover call. Without a
// caller there is nothing distinguishing one agent's call from another's, and
// matching on recency alone would manufacture conversions.
func TestRecordNeverCreditsDiscoveryWithoutAttribution(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	rec, reads, calls, _ := newRecorderAt(t, now)
	if err := calls.Append(store.DiscoveryCall{
		ID:      "call-1",
		At:      now.Add(-time.Minute).Format(time.RFC3339),
		Results: []store.DiscoveryCallResult{{ID: "alpha"}},
	}); err != nil {
		t.Fatalf("append call: %v", err)
	}

	rec.Record(httptest.NewRequest(http.MethodPost, "/skills/read", nil), []string{"alpha"})

	entries, err := reads.ReadSince(0)
	if err != nil {
		t.Fatalf("ReadSince: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].ViaDiscovery {
		t.Fatal("unattributed read was credited to a discover call")
	}
}

// Telemetry must never be the reason a skill fails to be served.
func TestRecordToleratesMissingDependencies(t *testing.T) {
	rec := NewReadRecorder(nil, nil, nil)
	rec.Record(attributedRequest(t, "operator-direct", ""), []string{"alpha"})

	var nilRecorder *ReadRecorder
	nilRecorder.Record(attributedRequest(t, "operator-direct", ""), []string{"alpha"})
}

func newSkillReadService(t *testing.T) *skillreads.Service {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "reads.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(skillreads.Schema)); err != nil {
		t.Fatal(err)
	}
	return skillreads.NewService(skillreads.NewSQLRepository(db))
}

type contextReadRepository struct{ appendContext, readContext context.Context }

func (r *contextReadRepository) Append(ctx context.Context, _ skillreads.Read, _ time.Time, _ int, _ time.Duration) error {
	r.appendContext = ctx
	return nil
}
func (r *contextReadRepository) ReadSince(ctx context.Context, _ time.Duration, _ time.Time) ([]skillreads.Read, error) {
	r.readContext = ctx
	return nil, nil
}

func TestSkillReadTelemetryConsumersPreserveRequestRoutingContext(t *testing.T) {
	type routingKey struct{}
	ctx := context.WithValue(context.Background(), routingKey{}, "lease")
	repository := &contextReadRepository{}
	reads := skillreads.NewService(repository)
	recorder := NewReadRecorder(reads, nil, nil)
	recorder.Record(httptest.NewRequest(http.MethodPost, "/skills/read", nil).WithContext(ctx), []string{"skill"})
	reporter := NewUsageReporter(reads, nil)
	if _, err := reporter.ReportContext(ctx, time.Hour); err != nil {
		t.Fatal(err)
	}
	if repository.appendContext != ctx || repository.readContext != ctx {
		t.Fatal("recorder or reporter discarded request-scoped database routing")
	}
}

func TestSkillReadRecorderRoutesCounterAndTelemetryTogether(t *testing.T) {
	routed, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "live.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { routed.Close() })
	initialize := func(ctx context.Context, db *sql.DB) error {
		return database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema), database.SchemaProviderFunc(skillreads.Schema))
	}
	if err := initialize(context.Background(), routed.Primary()); err != nil {
		t.Fatal(err)
	}
	routed.SetTestPoolInitializer(initialize)
	if err := routed.InstallTestPool(context.Background(), filepath.Join(t.TempDir(), "test.db"), "read-counter-lease", 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { routed.ClearTestPool("read-counter-lease") })
	reads := skillreads.NewService(skillreads.NewSQLRepository(routed))
	recorder := NewReadRecorder(reads, nil, NewMetricsAdapter(metrics.NewRepository(routed)))
	for _, ctx := range []context.Context{context.Background(), database.WithTestMode(context.Background())} {
		recorder.Record(httptest.NewRequest(http.MethodPost, "/skills/read", nil).WithContext(ctx), []string{"skill"})
	}
	for _, ctx := range []context.Context{context.Background(), database.WithTestMode(context.Background())} {
		rows, err := reads.WithContext(ctx).ReadSince(0)
		if err != nil || len(rows) != 1 {
			t.Fatalf("telemetry crossed routing boundary: %+v %v", rows, err)
		}
		var count int
		if err := routed.QueryRowContext(ctx, "SELECT usage_count FROM skill_metrics WHERE skill_id = ?", "skill").Scan(&count); err != nil || count != 1 {
			t.Fatalf("counter crossed routing boundary: count=%d err=%v", count, err)
		}
	}
}
