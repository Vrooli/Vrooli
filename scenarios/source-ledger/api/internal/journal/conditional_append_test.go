package journal

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/require"
	"source-ledger/internal/policy"
	"testing"
)

func TestConditionalAppendRejectsStalePredecessorAndConflictingReplay(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	ctx := policy.WithScope(context.Background(), "team:director-swarm")
	empty := ""
	first, err := repo.Append(ctx, Entry{Scope: "team:director-swarm", Kind: "walk-checkpoint", Body: "active", FacetID: UnclassifiedFacet, ExpectedLatest: &empty, RequestKey: "one", ImportKey: "request:one"}, nil)
	require.NoError(t, err)
	_, err = repo.Append(ctx, Entry{Scope: "team:director-swarm", Kind: "walk-checkpoint", Body: "other", FacetID: UnclassifiedFacet, ExpectedLatest: &empty}, nil)
	require.ErrorIs(t, err, ErrAppendConflict)
	_, err = repo.Append(ctx, Entry{Scope: "team:director-swarm", Kind: "walk-checkpoint", Body: "completed", FacetID: UnclassifiedFacet, ExpectedLatest: &first.ID}, nil)
	require.NoError(t, err)
	replay, err := repo.Append(ctx, Entry{Scope: "team:director-swarm", Kind: "walk-checkpoint", Body: "active", FacetID: UnclassifiedFacet, ExpectedLatest: &empty, RequestKey: "one", ImportKey: "request:one"}, nil)
	require.NoError(t, err)
	require.Equal(t, first.ID, replay.ID)
	require.True(t, replay.Existing)
	_, err = repo.Append(ctx, Entry{Scope: "team:director-swarm", Kind: "walk-checkpoint", Body: "changed", FacetID: UnclassifiedFacet, RequestKey: "one", ImportKey: "request:one"}, nil)
	require.ErrorIs(t, err, ErrAppendConflict)
}

func TestConditionalAppendAllowsOnlyOneWriterForAPredecessor(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	ctx := policy.WithScope(context.Background(), "team:director-swarm")
	empty := ""
	results := make(chan error, 2)
	for _, body := range []string{"first", "second"} {
		go func(body string) {
			_, e := repo.Append(ctx, Entry{Scope: "team:director-swarm", Kind: "walk-checkpoint", Body: body, FacetID: UnclassifiedFacet, ExpectedLatest: &empty}, nil)
			results <- e
		}(body)
	}
	wins := 0
	for range 2 {
		if e := <-results; e == nil {
			wins++
		} else {
			require.ErrorIs(t, e, ErrAppendConflict)
		}
	}
	require.Equal(t, 1, wins)
}
