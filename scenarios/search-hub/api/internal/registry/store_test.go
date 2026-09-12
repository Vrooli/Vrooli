package registry_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	localdb "search-hub/internal/database"
	"search-hub/internal/registry"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

// newStore returns a SQLite-backed Store with the production schema applied —
// the canonical compose pattern: db.NewSQLite + apidb.EnsureSchemas over the
// system + registry providers, so tests exercise the same shape main.go ships.
func newStore(t *testing.T) (registry.Store, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(registry.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return registry.NewSQLiteStore(d, clk), clk
}

func TestStoreUpsertInsertThenUpdate(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()
	d := validActive()

	created, token, err := store.Upsert(ctx, d, "")
	require.NoError(t, err)
	require.True(t, created, "first upsert inserts")
	require.NotEmpty(t, token, "first registration mints a control token")

	// Re-register the same leaf with a changed description → update, not insert.
	d2 := validActive()
	d2.Description = "Updated description."
	created, token2, err := store.Upsert(ctx, d2, "")
	require.NoError(t, err)
	require.False(t, created, "second upsert updates")
	require.Equal(t, token, token2, "re-register echoes the same token (empty presented)")

	got, err := store.Get(ctx, "cli-health.commands")
	require.NoError(t, err)
	require.Equal(t, "Updated description.", got.GetDescription())

	all, err := store.List(ctx, registry.ListFilter{})
	require.NoError(t, err)
	require.Len(t, all, 1, "upsert must not duplicate the leaf")
}

func TestStoreUpsertTokenOwnership(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	_, token, err := store.Upsert(ctx, validActive(), "")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// A re-register presenting a WRONG non-empty token is rejected (ownership).
	_, _, err = store.Upsert(ctx, validActive(), "deadbeef-not-the-token")
	require.Error(t, err)
	var mismatch registry.ErrTokenMismatch
	require.ErrorAs(t, err, &mismatch)

	// Presenting the CORRECT token succeeds and echoes it.
	_, echoed, err := store.Upsert(ctx, validActive(), token)
	require.NoError(t, err)
	require.Equal(t, token, echoed)
}

func TestStoreToken(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	_, token, err := store.Upsert(ctx, validActive(), "")
	require.NoError(t, err)

	got, err := store.Token(ctx, "cli-health.commands")
	require.NoError(t, err)
	require.Equal(t, token, got)

	_, err = store.Token(ctx, "nope.unknown")
	var notFound registry.ErrProviderNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestStoreUpsertRejectsInvalid(t *testing.T) {
	store, _ := newStore(t)
	d := validActive()
	d.Description = ""

	_, _, err := store.Upsert(context.Background(), d, "")
	require.Error(t, err)
	var invalid registry.ErrInvalidDescriptor
	require.ErrorAs(t, err, &invalid)
}

func TestStoreUpsertRoundTripsDescriptor(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()
	in := validActive()
	in.QueryHint = "prefix: find the command that"

	_, _, err := store.Upsert(ctx, in, "")
	require.NoError(t, err)

	got, err := store.Get(ctx, in.GetProviderId())
	require.NoError(t, err)
	require.Equal(t, in.GetProviderGroup(), got.GetProviderGroup())
	require.Equal(t, registryv1.Bucket_BUCKET_DO, got.GetBucket())
	require.Equal(t, "prefix: find the command that", got.GetQueryHint())
	require.Equal(t, "/vrooli.cli_health.v1.search.SearchService/Search", got.GetEndpoint().GetHttpJson().GetPath())
	require.Equal(t, "results", got.GetResultMapping().GetResultsPath())
	// Normalize defaults persisted.
	require.Equal(t, registryv1.ProviderState_PROVIDER_STATE_ACTIVE, got.GetState())
	require.Equal(t, registryv1.Scope_SCOPE_PROJECT, got.GetScope())
}

func TestStoreListFilters(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	// One ACTIVE DO command, one ACTIVE KNOW doc, one CAPABILITY_GAP code stub.
	require.NoError(t, mustUpsert(t, store, validActive()))

	doc := validActive()
	doc.ProviderId = "knowledge-observatory.docs"
	doc.ProviderGroup = "knowledge-observatory"
	doc.Bucket = registryv1.Bucket_BUCKET_KNOW
	doc.Type = "doc"
	require.NoError(t, mustUpsert(t, store, doc))

	gap := &registryv1.ProviderDescriptor{
		ProviderId:    "code.symbols",
		ProviderGroup: "code-reference",
		Bucket:        registryv1.Bucket_BUCKET_REUSE,
		Type:          "code",
		Description:   "Source symbols.",
		Lifecycle:     registryv1.Lifecycle_LIFECYCLE_PRODUCTION,
		State:         registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP,
		IntendedHome:  "code-reference",
	}
	require.NoError(t, mustUpsert(t, store, gap))

	t.Run("no filter returns all, id-ordered", func(t *testing.T) {
		all, err := store.List(ctx, registry.ListFilter{})
		require.NoError(t, err)
		require.Len(t, all, 3)
		require.Equal(t, "cli-health.commands", all[0].GetProviderId())
		require.Equal(t, "code.symbols", all[1].GetProviderId())
		require.Equal(t, "knowledge-observatory.docs", all[2].GetProviderId())
	})

	t.Run("filter by bucket", func(t *testing.T) {
		got, err := store.List(ctx, registry.ListFilter{Bucket: int32(registryv1.Bucket_BUCKET_KNOW)})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "knowledge-observatory.docs", got[0].GetProviderId())
	})

	t.Run("filter by type", func(t *testing.T) {
		got, err := store.List(ctx, registry.ListFilter{Type: "command"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "cli-health.commands", got[0].GetProviderId())
	})

	t.Run("filter by state finds gaps", func(t *testing.T) {
		got, err := store.List(ctx, registry.ListFilter{State: int32(registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP)})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "code.symbols", got[0].GetProviderId())
	})
}

func TestStoreGetNotFound(t *testing.T) {
	store, _ := newStore(t)
	_, err := store.Get(context.Background(), "nope.nope")
	var notFound registry.ErrProviderNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestStoreDeleteIdempotent(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()
	require.NoError(t, mustUpsert(t, store, validActive()))

	removed, err := store.Delete(ctx, "cli-health.commands")
	require.NoError(t, err)
	require.True(t, removed)

	removed, err = store.Delete(ctx, "cli-health.commands")
	require.NoError(t, err)
	require.False(t, removed, "second delete is a no-op")
}

func mustUpsert(t *testing.T, store registry.Store, d *registryv1.ProviderDescriptor) error {
	t.Helper()
	_, _, err := store.Upsert(context.Background(), d, "")
	return err
}
