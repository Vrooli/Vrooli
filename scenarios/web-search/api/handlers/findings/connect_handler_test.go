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
