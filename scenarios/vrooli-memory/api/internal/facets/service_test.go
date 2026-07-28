package facets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	localdb "vrooli-memory/internal/database"
	"vrooli-memory/internal/journal"
	"vrooli-memory/internal/testutil/mocks"
)

func newService(t *testing.T) (*Service, *journal.SQLiteRepository) {
	t.Helper()
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:facets?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db.Primary())
	require.NoError(t, repo.Seed(context.Background()))
	return NewService(repo), journal.NewSQLiteRepository(db.Primary())
}

func TestSeedHasExactlySixStableFacets(t *testing.T) {
	s, _ := newService(t)
	require.NoError(t, s.Seed(context.Background()))
	items, err := s.List(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 6)
}

func TestUnknownFacetIsHardError(t *testing.T) {
	s, _ := newService(t)
	require.ErrorAs(t, s.Validate(context.Background(), "invented"), new(ErrUnknownFacet))
	require.NoError(t, s.Validate(context.Background(), UnclassifiedFacet))
}

func TestJournalWriteRejectsUnknownExplicitFacet(t *testing.T) {
	s, j := newService(t)
	_, err := journal.NewService(j, &mocks.FakeInference{ClassifyOut: "episode", EmbedOut: []float64{1}}, s).
		Append(context.Background(), journal.Entry{Body: "do not silently route this", FacetID: "invented"})
	var unknown ErrUnknownFacet
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "invented", unknown.ID)
}

func TestOnlyEpisodeIsCompactionEligibleAndPinExemptsIt(t *testing.T) {
	s, j := newService(t)
	ctx := context.Background()
	episode, err := j.Append(ctx, journal.Entry{Body: "episode", FacetID: "episode"}, nil)
	require.NoError(t, err)
	_, err = s.ReFacet(ctx, Assignment{EntryID: episode.ID, FacetID: "episode"})
	require.NoError(t, err)
	eligible, err := s.CompactionEligible(ctx, episode.ID)
	require.NoError(t, err)
	require.True(t, eligible)
	require.NoError(t, s.SetPin(ctx, episode.ID, true))
	eligible, err = s.CompactionEligible(ctx, episode.ID)
	require.NoError(t, err)
	require.False(t, eligible)
	fact, err := j.Append(ctx, journal.Entry{Body: "fact", FacetID: "environment-fact"}, nil)
	require.NoError(t, err)
	_, err = s.ReFacet(ctx, Assignment{EntryID: fact.ID, FacetID: "environment-fact"})
	require.NoError(t, err)
	eligible, err = s.CompactionEligible(ctx, fact.ID)
	require.NoError(t, err)
	require.False(t, eligible)
}

func TestRefacetRetainsHistory(t *testing.T) {
	s, j := newService(t)
	ctx := context.Background()
	entry, err := j.Append(ctx, journal.Entry{Body: "rule", FacetID: "episode"}, nil)
	require.NoError(t, err)
	_, err = s.ReFacet(ctx, Assignment{EntryID: entry.ID, FacetID: "episode"})
	require.NoError(t, err)
	_, err = s.ReFacet(ctx, Assignment{EntryID: entry.ID, FacetID: "standing-rule"})
	require.NoError(t, err)
	history, err := s.repo.Assignments(ctx, entry.ID)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, "episode", history[0].FacetID)
	require.Equal(t, "standing-rule", history[1].FacetID)
}

func TestSupersessionLeavesOriginalEntryRetrievable(t *testing.T) {
	s, j := newService(t)
	ctx := context.Background()
	original, err := j.Append(ctx, journal.Entry{Body: "old rule", FacetID: "standing-rule"}, nil)
	require.NoError(t, err)
	replacement, err := j.Append(ctx, journal.Entry{Body: "new rule", FacetID: "standing-rule"}, nil)
	require.NoError(t, err)
	require.NoError(t, s.MarkSuperseded(ctx, original.ID, replacement.ID))

	stored, err := j.Get(ctx, original.ID)
	require.NoError(t, err)
	require.Equal(t, original.Body, stored.Body)
}
