package findings_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"
	findingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings"

	handler "web-search/handlers/findings"
	localdb "web-search/internal/database"
	"web-search/internal/findingindex"
	internalfindings "web-search/internal/findings"

	testdb "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/scheduletest"
)

// fakeSearcher returns canned hits in order, simulating the semantic index.
type fakeSearcher struct {
	hits   []findingindex.Hit
	method string
}

func (f *fakeSearcher) Search(ctx context.Context, query string, limit int) ([]findingindex.Hit, string, error) {
	return f.hits, f.method, nil
}

func newFindingsService(t *testing.T) internalfindings.Service {
	return newFindingsServiceAtClock(t, schedule.System())
}

// newFindingsServiceAtClock builds the findings service over a fresh SQLite DB
// whose row timestamps come from clk, so handler-level decay assertions are
// deterministic.
func newFindingsServiceAtClock(t *testing.T, clk schedule.Clock) internalfindings.Service {
	t.Helper()
	d := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalfindings.Schema),
	))
	return internalfindings.NewService(internalfindings.NewSQLiteRepository(d, clk))
}

func TestSearchFindingsProjectsAndExcludesSuperseded(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)

	a, err := svc.Add(ctx, internalfindings.NewFinding{Claim: "claude opus 4.8 ships", Source: internalfindings.SourceManual})
	require.NoError(t, err)
	b, err := svc.Add(ctx, internalfindings.NewFinding{Claim: "older claude release", Source: internalfindings.SourceManual})
	require.NoError(t, err)
	_, err = svc.Supersede(ctx, b.ID, a.ID, "outdated")
	require.NoError(t, err)

	// The fake index returns BOTH ids (as if it were stale); the handler must
	// drop the superseded one by default and surface it with include_archived.
	searcher := &fakeSearcher{method: "dense", hits: []findingindex.Hit{
		{FindingID: a.ID, Score: 0.91},
		{FindingID: b.ID, Score: 0.40, Weak: true},
	}}
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Searcher: searcher, Clock: schedule.System()})

	resp, err := h.SearchFindings(ctx, connect.NewRequest(&findingsv1.SearchFindingsRequest{Query: "claude opus", Limit: 10}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Hits, 1)
	require.Equal(t, a.ID, resp.Msg.Hits[0].Finding.Id)
	require.Equal(t, "dense", resp.Msg.Method)

	respArch, err := h.SearchFindings(ctx, connect.NewRequest(&findingsv1.SearchFindingsRequest{Query: "older claude", Limit: 10, IncludeArchived: true}))
	require.NoError(t, err)
	var sawSuperseded bool
	for _, hit := range respArch.Msg.Hits {
		if hit.Finding.Id == b.ID {
			sawSuperseded = true
		}
	}
	require.True(t, sawSuperseded, "include_archived must surface the superseded finding")
}

func TestAddFindingValidatesClaim(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: schedule.System()})
	_, err := h.AddFinding(ctx, connect.NewRequest(&findingsv1.AddFindingRequest{Claim: "   "}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCountFindingsDefaultsWindow(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)
	_, err := svc.Add(ctx, internalfindings.NewFinding{Claim: "one", Source: internalfindings.SourceManual})
	require.NoError(t, err)
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: schedule.System()})
	resp, err := h.CountFindings(ctx, connect.NewRequest(&findingsv1.CountFindingsRequest{}))
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Msg.Count)
}

// recordingSurfacer captures the ids the handler enqueues, proving the search
// path hands surfacing to the seam rather than writing synchronously.
type recordingSurfacer struct {
	calls [][]string
}

func (r *recordingSurfacer) Surfaced(ids []string) {
	r.calls = append(r.calls, ids)
}

// TestSearchFindingsSurfacesAsync proves SearchFindings enqueues the surfaced
// ids on the Surfacer seam (fire-and-forget) — never writing usage on the hot
// path — and records only the semantic hits actually returned, not superseded
// archived appends.
func TestSearchFindingsSurfacesAsync(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)
	a, err := svc.Add(ctx, internalfindings.NewFinding{Claim: "surfaced one", Source: internalfindings.SourceManual})
	require.NoError(t, err)
	b, err := svc.Add(ctx, internalfindings.NewFinding{Claim: "surfaced two", Source: internalfindings.SourceManual})
	require.NoError(t, err)

	searcher := &fakeSearcher{method: "dense", hits: []findingindex.Hit{
		{FindingID: a.ID, Score: 0.9},
		{FindingID: b.ID, Score: 0.8},
	}}
	surfacer := &recordingSurfacer{}
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Searcher: searcher, Surfacer: surfacer, Clock: schedule.System()})

	resp, err := h.SearchFindings(ctx, connect.NewRequest(&findingsv1.SearchFindingsRequest{Query: "surfaced", Limit: 10}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Hits, 2)

	// The seam saw exactly the returned hit ids — proof surfacing went through the
	// async seam, not a synchronous repo write (no usage row exists yet).
	require.Len(t, surfacer.calls, 1)
	require.ElementsMatch(t, []string{a.ID, b.ID}, surfacer.calls[0])

	// No synchronous usage write happened on the hot path.
	eff, err := svc.ListEffectiveness(ctx, false, 10)
	require.NoError(t, err)
	for _, e := range eff {
		require.Zero(t, e.Usage.SurfacedCount, "search must not write usage synchronously")
	}
}

// TestListEffectivenessBlendsAndRecordUsage exercises the effectiveness read
// path and the explicit RecordUsage write through the handler.
func TestListEffectivenessBlendsAndRecordUsage(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)
	f, err := svc.Add(ctx, internalfindings.NewFinding{Claim: "effective claim", Confidence: 0.8, Source: internalfindings.SourceManual})
	require.NoError(t, err)
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: schedule.System()})

	// Before any usage: factor 1.0 (fresh), effective_score == effective_confidence.
	resp, err := h.ListEffectiveness(ctx, connect.NewRequest(&findingsv1.ListEffectivenessRequest{Limit: 10}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Items, 1)
	item := resp.Msg.Items[0]
	require.InDelta(t, 1.0, item.UsageFactor, 1e-9)
	require.InDelta(t, item.EffectiveConfidence, item.EffectiveScore, 1e-9)
	require.Zero(t, item.SurfacedCount)

	// RecordUsage bumps the used counter.
	useResp, err := h.RecordUsage(ctx, connect.NewRequest(&findingsv1.RecordUsageRequest{Id: f.ID}))
	require.NoError(t, err)
	require.Equal(t, f.ID, useResp.Msg.Finding.Id)

	resp, err = h.ListEffectiveness(ctx, connect.NewRequest(&findingsv1.ListEffectivenessRequest{Limit: 10}))
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.Msg.Items[0].UsedCount)

	// A bogus RecordUsage id is NotFound.
	_, err = h.RecordUsage(ctx, connect.NewRequest(&findingsv1.RecordUsageRequest{Id: "nope"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// TestListDisputesProjectsDisputeQueueEntries pins the dispute review queue
// surface: ListDisputes returns ONLY disputed findings, each carrying the
// reviewable fields — finding id, the contradiction description (dispute
// note), the conflicting source citations, status DISPUTED, and created_at.
func TestListDisputesProjectsDisputeQueueEntries(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)

	disputed, err := svc.Add(ctx, internalfindings.NewFinding{
		Claim: "contested claim",
		Citations: []internalfindings.NewCitation{
			{URL: "https://one.example", Title: "Source One"},
			{URL: "https://two.example", Title: "Source Two"},
		},
	})
	require.NoError(t, err)
	_, err = svc.Flag(ctx, disputed.ID, "source one contradicts source two")
	require.NoError(t, err)
	_, err = svc.Add(ctx, internalfindings.NewFinding{Claim: "uncontested"})
	require.NoError(t, err)

	h := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: schedule.System()})
	resp, err := h.ListDisputes(ctx, connect.NewRequest(&findingsv1.ListDisputesRequest{Limit: 10}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Findings, 1, "only disputed findings enter the review queue")

	entry := resp.Msg.Findings[0]
	require.Equal(t, disputed.ID, entry.Id)
	require.Equal(t, findingsv1.FindingStatus_FINDING_STATUS_DISPUTED, entry.Status)
	require.Equal(t, "source one contradicts source two", entry.DisputeNote)
	require.Len(t, entry.Citations, 2, "the conflicting source references ride along")
	require.NotNil(t, entry.CreatedAt)
}

// TestSearchSurfacesDisputedWithConflictSignal asserts the default search
// surface returns disputed findings WITH their conflict signal — status
// DISPUTED plus the dispute note and both conflicting source citations — so
// every consumer can render the "sources conflict" warning.
func TestSearchSurfacesDisputedWithConflictSignal(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)

	f, err := svc.Add(ctx, internalfindings.NewFinding{
		Claim: "the contested rate is 4 percent",
		Citations: []internalfindings.NewCitation{
			{URL: "https://bank.example", Title: "Bank"},
			{URL: "https://press.example", Title: "Press"},
		},
	})
	require.NoError(t, err)
	_, err = svc.Flag(ctx, f.ID, "bank and press disagree")
	require.NoError(t, err)

	searcher := &fakeSearcher{method: "dense", hits: []findingindex.Hit{{FindingID: f.ID, Score: 0.8}}}
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Searcher: searcher, Clock: schedule.System()})

	resp, err := h.SearchFindings(ctx, connect.NewRequest(&findingsv1.SearchFindingsRequest{Query: "contested rate", Limit: 5}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Hits, 1, "disputed findings stay in the default results")
	got := resp.Msg.Hits[0].Finding
	require.Equal(t, findingsv1.FindingStatus_FINDING_STATUS_DISPUTED, got.Status, "the conflict warning signal")
	require.Equal(t, "bank and press disagree", got.DisputeNote)
	require.Len(t, got.Citations, 2, "both conflicting sources are returned")
}

// TestListEffectivenessReturnsDecayedConfidence pins age decay on the read
// path: a finding stored 180 days (one half-life) ago is returned with an
// effective confidence of about half its stored confidence. Storage is never
// mutated — the decayed value is computed at read time against the schedule.
func TestListEffectivenessReturnsDecayedConfidence(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := scheduletest.New(created)
	svc := newFindingsServiceAtClock(t, clk)

	f, err := svc.Add(ctx, internalfindings.NewFinding{Claim: "aging claim", Confidence: 0.8})
	require.NoError(t, err)

	clk.SetNow(created.Add(internalfindings.DecayHalfLife))
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: clk})

	resp, err := h.ListEffectiveness(ctx, connect.NewRequest(&findingsv1.ListEffectivenessRequest{Limit: 10}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Items, 1)
	item := resp.Msg.Items[0]
	require.Equal(t, f.ID, item.Finding.Id)
	require.InDelta(t, 0.8, item.Finding.Confidence, 1e-9, "stored confidence is untouched")
	require.InDelta(t, 0.4, item.EffectiveConfidence, 1e-3, "one half-life decays the read value to half")
	require.Less(t, item.EffectiveConfidence, item.Finding.Confidence,
		"an aged finding reads below its stored confidence")
}

// fakeGC is a GCRunner that returns a canned report and records the dry-run flag.
type fakeGC struct {
	report    internalfindings.GCReport
	gotDryRun bool
}

func (g *fakeGC) Run(_ context.Context, dryRun bool) (internalfindings.GCReport, error) {
	g.gotDryRun = dryRun
	g.report.DryRun = dryRun
	return g.report, nil
}

// TestRunGCProjectsReport asserts the handler threads the dry-run flag and maps
// the report onto the wire response; and that an unconfigured GC is Unavailable.
func TestRunGCProjectsReport(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)
	gc := &fakeGC{report: internalfindings.GCReport{
		SupersededDecayed:     []string{"a"},
		ColdArchiveCandidates: []string{"b"},
		StaleDisputes:         []string{"c"},
		Orphans:               []string{"d"},
	}}
	h := handler.NewConnectHandler(handler.Deps{Service: svc, GC: gc, Clock: schedule.System()})

	resp, err := h.RunGC(ctx, connect.NewRequest(&findingsv1.RunGCRequest{DryRun: true}))
	require.NoError(t, err)
	require.True(t, gc.gotDryRun)
	require.True(t, resp.Msg.DryRun)
	require.Equal(t, []string{"a"}, resp.Msg.SupersededDecayed)
	require.Equal(t, []string{"b"}, resp.Msg.ColdArchiveCandidates)
	require.Equal(t, []string{"c"}, resp.Msg.StaleDisputes)
	require.Equal(t, []string{"d"}, resp.Msg.Orphans)

	// No GC wired ⇒ Unavailable.
	hNoGC := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: schedule.System()})
	_, err = hNoGC.RunGC(ctx, connect.NewRequest(&findingsv1.RunGCRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
