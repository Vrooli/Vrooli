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

func TestRegistryDeploymentTierIsDurableAndUpdatePreservesIt(t *testing.T) {
	d, clk := newSchemaDB(t)
	ctx := context.Background()
	svc := registry.NewService(registry.NewSQLiteRepository(d, clk))
	created, err := svc.Register(ctx, registry.RegisterInput{Name: "target", OS: "linux", Arch: "amd64", DeploymentTier: "vps"})
	require.NoError(t, err)
	require.Equal(t, "vps", created.DeploymentTier)
	freshRepo := registry.NewSQLiteRepository(d, clk)
	loaded, err := freshRepo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "vps", loaded.DeploymentTier)
	updated, err := svc.Update(ctx, registry.UpdateInput{ID: created.ID, Name: created.Name, Endpoint: created.Endpoint, Capabilities: created.Capabilities, Scopes: created.Scopes, Revision: created.Revision, Kind: created.Kind})
	require.NoError(t, err)
	require.Equal(t, "vps", updated.DeploymentTier, "an unrelated owner update must preserve the tier")
}

func TestRegistryEnrollmentGenerationIsBridgeAssignedMonotonicAndDurable(t *testing.T) {
	d, clk := newSchemaDB(t)
	ctx := context.Background()
	svc := registry.NewService(registry.NewSQLiteRepository(d, clk))
	node, err := svc.Register(ctx, registry.RegisterInput{Name: "target", OS: "linux", Arch: "amd64"})
	require.NoError(t, err)
	require.Zero(t, node.EnrollmentGeneration, "newly registered nodes are unbound until Bridge links a Machine")

	bound, err := svc.SetEnrollmentGeneration(ctx, node.ID, 2)
	require.NoError(t, err)
	require.EqualValues(t, 2, bound.EnrollmentGeneration)
	retry, err := svc.SetEnrollmentGeneration(ctx, node.ID, 2)
	require.NoError(t, err)
	require.EqualValues(t, 2, retry.EnrollmentGeneration)
	updated, err := svc.Update(ctx, registry.UpdateInput{ID: node.ID, Name: node.Name, Kind: node.Kind, Revision: "rev-next"})
	require.NoError(t, err)
	require.EqualValues(t, 2, updated.EnrollmentGeneration, "ordinary caller updates cannot choose or clear generation")
	_, err = svc.SetEnrollmentGeneration(ctx, node.ID, 1)
	var conflict registry.ErrEnrollmentGenerationConflict
	require.ErrorAs(t, err, &conflict, "a stale Bridge write cannot roll generation back")

	loaded, err := registry.NewSQLiteRepository(d, clk).Get(ctx, node.ID)
	require.NoError(t, err)
	require.EqualValues(t, 2, loaded.EnrollmentGeneration)
}
