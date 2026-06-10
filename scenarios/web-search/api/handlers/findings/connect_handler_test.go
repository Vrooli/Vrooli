package findings_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"
	findingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings"

	handler "web-search/handlers/findings"
	"web-search/internal/clock"
	localdb "web-search/internal/database"
	"web-search/internal/findingindex"
	internalfindings "web-search/internal/findings"
	testdb "web-search/internal/testutil/db"
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
	t.Helper()
	d := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalfindings.Schema),
	))
	return internalfindings.NewService(internalfindings.NewSQLiteRepository(d, clock.System{}))
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
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Searcher: searcher, Clock: clock.System{}})

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
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: clock.System{}})
	_, err := h.AddFinding(ctx, connect.NewRequest(&findingsv1.AddFindingRequest{Claim: "   "}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCountFindingsDefaultsWindow(t *testing.T) {
	ctx := context.Background()
	svc := newFindingsService(t)
	_, err := svc.Add(ctx, internalfindings.NewFinding{Claim: "one", Source: internalfindings.SourceManual})
	require.NoError(t, err)
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: clock.System{}})
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
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Searcher: searcher, Surfacer: surfacer, Clock: clock.System{}})

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
	h := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: clock.System{}})

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
	h := handler.NewConnectHandler(handler.Deps{Service: svc, GC: gc, Clock: clock.System{}})

	resp, err := h.RunGC(ctx, connect.NewRequest(&findingsv1.RunGCRequest{DryRun: true}))
	require.NoError(t, err)
	require.True(t, gc.gotDryRun)
	require.True(t, resp.Msg.DryRun)
	require.Equal(t, []string{"a"}, resp.Msg.SupersededDecayed)
	require.Equal(t, []string{"b"}, resp.Msg.ColdArchiveCandidates)
	require.Equal(t, []string{"c"}, resp.Msg.StaleDisputes)
	require.Equal(t, []string{"d"}, resp.Msg.Orphans)

	// No GC wired ⇒ Unavailable.
	hNoGC := handler.NewConnectHandler(handler.Deps{Service: svc, Clock: clock.System{}})
	_, err = hNoGC.RunGC(ctx, connect.NewRequest(&findingsv1.RunGCRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
