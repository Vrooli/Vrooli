package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogcoverage"
)

func TestHealthOverviewMapsUnavailableCatalogToInternal(t *testing.T) {
	repoRoot := t.TempDir()
	h := &handler{repoRoot: repoRoot}
	// Pin a matching cached report so this exercises the health RPC's own
	// catalog-loading error mapping rather than the report warm-up response.
	h.reports.report = &catalogcoverage.Report{}
	h.reports.fingerprint = "current"
	h.reports.fpAt = time.Now()
	h.reports.fpValue = "current"

	_, err := h.GetHealthOverview(context.Background(), connect.NewRequest(&catalogv1.GetHealthOverviewRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.ErrorContains(t, err, "load catalog health assets")
}

func TestReconcileGraphMapsUnavailableRepositoryToInternal(t *testing.T) {
	repoRoot := t.TempDir()
	assetPath := filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog", "assets", "controls", "broken.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(assetPath), 0o755))
	require.NoError(t, os.WriteFile(assetPath, []byte("{"), 0o600))
	h := &handler{repoRoot: repoRoot}

	_, err := h.ReconcileGraph(context.Background(), connect.NewRequest(&catalogv1.ReconcileGraphRequest{}))

	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.ErrorContains(t, err, "reconcile catalog graph")
}

func TestRunGateMapsMalformedDefinitionsToInternal(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog", "config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
	require.NoError(t, os.WriteFile(configPath, []byte("{"), 0o600))
	h := &handler{repoRoot: repoRoot}

	_, err := h.RunGate(context.Background(), connect.NewRequest(&catalogv1.RunGateRequest{Gate: "release-provenance"}))

	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.ErrorContains(t, err, "load catalog gate definitions")
}
