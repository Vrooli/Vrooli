package bindings

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/repo-contract-go"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	internalbindings "program-runtime/internal/bindings"
)

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 8 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}

func liveRegistry(t *testing.T) *internalbindings.Registry {
	t.Helper()
	root, err := repocontract.ResolveRepoRoot()
	require.NoError(t, err)
	registry, err := internalbindings.Load(root)
	require.NoError(t, err)
	return registry
}

func TestResolveActCellsLoadsOwnedDenominatorWhenRequestIsEmpty(t *testing.T) {
	service := &service{registry: liveRegistry(t)}
	response, err := service.ResolveActCells(context.Background(), connect.NewRequest(&bindingsv1.ResolveActCellsRequest{}))
	require.NoError(t, err)
	require.Len(t, response.Msg.GetCells(), 28)
	require.Equal(t, int32(28), response.Msg.GetAuditedCells())
	require.Equal(t, int32(28), response.Msg.GetTotalCells())
	require.Equal(t, "partial", response.Msg.GetDenominatorConfidence())
}

func TestResolveActCellsNamesUnavailableDenominator(t *testing.T) {
	service := &service{registry: liveRegistry(t), actSpacePath: filepath.Join(t.TempDir(), "missing-act-space.md")}
	_, err := service.ResolveActCells(context.Background(), connect.NewRequest(&bindingsv1.ResolveActCellsRequest{}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Act denominator")
	require.Contains(t, err.Error(), "missing-act-space.md")
}
