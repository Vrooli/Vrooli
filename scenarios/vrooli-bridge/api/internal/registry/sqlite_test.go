package registry_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
	"vrooli-bridge/internal/registry"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "vrooli-bridge/internal/database"
)

func newSchemaDB(t *testing.T) (*sql.DB, *scheduletest.FakeClock) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(registry.Schema),
	))
	return d, clk
}

// [REQ:BRG-P0-001] A registered node persists its full durable identity and
// round-trips OS/arch/revision/endpoint/capabilities/scopes intact.
func TestSQLiteRepository_CreateGetRoundTrip(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := registry.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	created, err := repo.Create(ctx, registry.Node{
		Name:         "mac-mini",
		OS:           "darwin",
		Arch:         "arm64",
		Revision:     "abc123",
		Endpoint:     "https://node.local",
		Capabilities: []string{"scenario test*"},
		Scopes:       []string{"scenario test*", "registry list"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, clk.Now().UTC(), created.CreatedAt)

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "mac-mini", got.Name)
	require.Equal(t, "darwin", got.OS)
	require.Equal(t, "arm64", got.Arch)
	require.Equal(t, "abc123", got.Revision)
	require.Equal(t, "https://node.local", got.Endpoint)
	require.Equal(t, []string{"scenario test*"}, got.Capabilities)
	require.Equal(t, []string{"scenario test*", "registry list"}, got.Scopes)
	require.False(t, got.Revoked())
	require.True(t, got.LastSeenAt.IsZero())
}

func TestSQLiteRepository_GetMissingReturnsTyped(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := registry.NewSQLiteRepository(d, clk)
	_, err := repo.Get(context.Background(), "nope")
	require.ErrorAs(t, err, &registry.ErrNodeNotFound{})
}

// [REQ:BRG-P0-001] List returns nodes newest-first.
func TestSQLiteRepository_ListNewestFirst(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := registry.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	first, err := repo.Create(ctx, registry.Node{Name: "a", OS: "linux", Arch: "amd64"})
	require.NoError(t, err)
	clk.Advance(time.Minute)
	second, err := repo.Create(ctx, registry.Node{Name: "b", OS: "linux", Arch: "amd64"})
	require.NoError(t, err)

	nodes, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	require.Equal(t, second.ID, nodes[0].ID, "newest first")
	require.Equal(t, first.ID, nodes[1].ID)
}

// [REQ:BRG-P0-001] Revoke stamps revoked_at atomically and is idempotent.
func TestSQLiteRepository_RevokeIsIdempotent(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := registry.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	n, err := repo.Create(ctx, registry.Node{Name: "a", OS: "linux", Arch: "amd64"})
	require.NoError(t, err)

	clk.Advance(time.Hour)
	revoked, err := repo.Revoke(ctx, n.ID)
	require.NoError(t, err)
	require.True(t, revoked.Revoked())
	firstRevokedAt := revoked.RevokedAt

	// A second revoke is a no-op that returns the same revoked_at.
	clk.Advance(time.Hour)
	again, err := repo.Revoke(ctx, n.ID)
	require.NoError(t, err)
	require.Equal(t, firstRevokedAt, again.RevokedAt, "revoke must not re-stamp an already-revoked node")

	got, err := repo.Get(ctx, n.ID)
	require.NoError(t, err)
	require.True(t, got.Revoked())
}

func TestSQLiteRepository_RevokeMissing(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := registry.NewSQLiteRepository(d, clk)
	_, err := repo.Revoke(context.Background(), "nope")
	require.ErrorAs(t, err, &registry.ErrNodeNotFound{})
}

// [REQ:BRG-P0-001] Update mutates the editable surface and bumps updated_at,
// while preserving immutable fields (os/arch/created_at).
func TestSQLiteRepository_UpdatePreservesImmutable(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := registry.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	n, err := repo.Create(ctx, registry.Node{Name: "a", OS: "linux", Arch: "amd64", Capabilities: []string{"x"}})
	require.NoError(t, err)

	clk.Advance(time.Minute)
	updated, err := repo.Update(ctx, registry.Node{
		ID: n.ID, Name: "renamed", Endpoint: "https://new", Scopes: []string{"scenario test*"}, Revision: "rev9",
	})
	require.NoError(t, err)
	require.Equal(t, "renamed", updated.Name)
	require.Equal(t, "https://new", updated.Endpoint)
	require.Equal(t, []string{"scenario test*"}, updated.Scopes)
	require.Equal(t, "rev9", updated.Revision)
	require.Equal(t, "linux", updated.OS, "os immutable")
	require.Equal(t, n.CreatedAt, updated.CreatedAt, "created_at immutable")
	require.True(t, updated.UpdatedAt.After(n.UpdatedAt))
}

// [REQ:BRG-P0-003] TouchLastSeen records a heartbeat timestamp; a heartbeat
// from an unknown node is a silent no-op (never errors the presence path).
func TestSQLiteRepository_TouchLastSeen(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := registry.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	n, err := repo.Create(ctx, registry.Node{Name: "a", OS: "linux", Arch: "amd64"})
	require.NoError(t, err)

	seen := clk.Now().Add(2 * time.Hour)
	require.NoError(t, repo.TouchLastSeen(ctx, n.ID, seen))

	got, err := repo.Get(ctx, n.ID)
	require.NoError(t, err)
	require.Equal(t, seen.UTC(), got.LastSeenAt)

	// Unknown node: no error.
	require.NoError(t, repo.TouchLastSeen(ctx, "ghost", seen))
}

// TestSQLiteRepository_AppliesSchemaIdempotently guards the IF-NOT-EXISTS
// migrate-never-recreate contract.
func TestSQLiteRepository_AppliesSchemaIdempotently(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	apply := func() error {
		return apidb.EnsureSchemas(ctx, d,
			apidb.SchemaProviderFunc(localdb.SystemSchema),
			apidb.SchemaProviderFunc(registry.Schema),
		)
	}
	require.NoError(t, apply())
	require.NoError(t, apply())
}
