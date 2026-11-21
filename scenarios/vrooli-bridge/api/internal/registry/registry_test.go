package registry_test

import (
	"context"
	"testing"

	"vrooli-bridge/internal/registry"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scopecatalog"
)

// TestRegistry_DurableIdentityAndAtomicRevoke is the named requirement test for
// OT-P0-001: the registry persists a durable node record across a fresh
// repository handle (proving it survives a control-plane restart), and revoke
// removes the node from active use atomically (revoked_at stamped in one op,
// status terminal). Credential invalidation is layered on in Phase 2 (pairing);
// this proves the durable lifecycle half.
//
// [REQ:BRG-P0-001]
func TestRegistry_DurableIdentityAndAtomicRevoke(t *testing.T) {
	d, clk := newSchemaDB(t)
	ctx := context.Background()

	// Register through the service (validation path), then drop the handle and
	// re-open a fresh repository to prove the record is durable, not in-memory.
	svc := registry.NewService(registry.NewSQLiteRepository(d, clk), registry.WithGrantValidator(registry.NewCatalogGrantValidator(scopecatalog.Catalog{
		Scopes: []scopecatalog.Scope{{Scenario: "vrooli", Value: "vrooli:write"}},
	})))
	created, err := svc.Register(ctx, registry.RegisterInput{
		Name: "office-linux", OS: "linux", Arch: "amd64",
		Scopes: []string{"vrooli:write"},
	})
	require.NoError(t, err)

	freshRepo := registry.NewSQLiteRepository(d, clk)
	got, err := freshRepo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "office-linux", got.Name)
	require.Equal(t, []string{"vrooli:write"}, got.Scopes)
	require.False(t, got.Revoked())

	// Revoke atomically: one operation stamps revoked_at; the node is terminal.
	revoked, err := svc.Revoke(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, revoked.Revoked())

	afterRevoke, err := freshRepo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, afterRevoke.Revoked(), "revocation is durable")
	require.False(t, afterRevoke.RevokedAt.IsZero())
}
