package catalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
)

func TestCatalogGraphRPCContracts(t *testing.T) {
	h := &handler{repoRoot: catalogTestRepoRoot(t)}
	ctx := context.Background()

	structure, err := h.GetCatalogStructure(ctx, connect.NewRequest(&catalogv1.GetCatalogStructureRequest{}))
	require.NoError(t, err)
	require.NotNil(t, structure.Msg.Structure)
	require.NotEmpty(t, structure.Msg.Structure.Population)
	require.NotEmpty(t, structure.Msg.Structure.Invariants)
	require.NotEmpty(t, structure.Msg.Structure.BlastRadius)
	for i := 1; i < len(structure.Msg.Structure.Population); i++ {
		require.GreaterOrEqual(t, structure.Msg.Structure.Population[i-1].Rung, structure.Msg.Structure.Population[i].Rung)
	}

	relationships, err := h.GetAssetRelationships(ctx, connect.NewRequest(&catalogv1.GetAssetRelationshipsRequest{AssetId: "react-component-library:NotificationBadge"}))
	require.NoError(t, err)
	require.Equal(t, "react-component-library:NotificationBadge", relationships.Msg.Relationships.Root.AssetId)
	require.NotEmpty(t, relationships.Msg.Relationships.Closure)
	require.NotEmpty(t, relationships.Msg.Relationships.ClosureBands)

	contract, err := h.GetAssetPortContract(ctx, connect.NewRequest(&catalogv1.GetAssetPortContractRequest{AssetId: "react-component-library:NotificationBadge"}))
	require.NoError(t, err)
	require.Equal(t, "react-component-library:NotificationBadge", contract.Msg.Contract.AssetId)
	require.GreaterOrEqual(t, contract.Msg.Contract.ClosureCount, int32(1))

	_, err = h.GetAssetRelationships(ctx, connect.NewRequest(&catalogv1.GetAssetRelationshipsRequest{AssetId: "missing:asset"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = h.GetAssetPortContract(ctx, connect.NewRequest(&catalogv1.GetAssetPortContractRequest{AssetId: "missing:asset"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestCatalogGraphRPCMapsUnavailableCatalogToInternal(t *testing.T) {
	h := &handler{repoRoot: t.TempDir()}
	_, err := h.GetCatalogStructure(context.Background(), connect.NewRequest(&catalogv1.GetCatalogStructureRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	_, err = h.GetAssetRelationships(context.Background(), connect.NewRequest(&catalogv1.GetAssetRelationshipsRequest{AssetId: "anything"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestReconcileGraphRPCReportsCatalogRows(t *testing.T) {
	h := &handler{repoRoot: catalogTestRepoRoot(t)}
	response, err := h.ReconcileGraph(context.Background(), connect.NewRequest(&catalogv1.ReconcileGraphRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, response.Msg.Assets)
	require.NotNil(t, response.Msg.Distribution)
}

func TestGraphErrorClassifiesUnexpectedErrors(t *testing.T) {
	err := graphError(errors.New("unexpected graph failure"))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func catalogTestRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		candidate := filepath.Join(dir, "scenarios", "react-component-library", "catalog", "config.json")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Clean(dir)
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "could not locate repository root from test working directory %q", dir)
		dir = parent
	}
}
