package asset

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogcoverage"
)

type recordingCatalog struct {
	store *catalogcoverage.EvidenceStore
	ctx   context.Context
	t     *testing.T
}

func (c recordingCatalog) RecordGate(_ context.Context, req *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	require.True(c.t, req.Msg.GetAll())
	require.Equal(c.t, "asset.one", req.Msg.GetAssetId())
	err := c.store.Save(c.ctx, []catalogcoverage.GateEvidence{{AssetID: "asset.one", Target: "catalog", Gate: "types", Version: "1.0.0", Result: "pass", SourceRevision: "rev", RecordedAt: "2026-11-15T00:00:00Z"}})
	return connect.NewResponse(&catalogv1.RunGateResponse{EvidenceRowsWritten: 1}), err
}

func (recordingCatalog) CheckAsset(context.Context, *connect.Request[catalogv1.CheckAssetRequest]) (*connect.Response[catalogv1.CheckAssetResponse], error) {
	return connect.NewResponse(&catalogv1.CheckAssetResponse{AssetId: "asset.one", Verdict: "PUBLISHABLE"}), nil
}

func TestRecordCheckRefreshesAndPersistsEvidence(t *testing.T) {
	ctx := context.Background()
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(catalogcoverage.Schema)))
	store := catalogcoverage.NewEvidenceStore(database)
	root := t.TempDir()
	generated := filepath.Join(root, "generated", "component-manifest.json")
	h := &handlers{
		catalog: recordingCatalog{store: store, ctx: ctx, t: t},
		buildCatalog: func(_ string) error {
			if err := os.MkdirAll(filepath.Dir(generated), 0o700); err != nil {
				return err
			}
			return os.WriteFile(generated, []byte(`{"generated":true}`), 0o600)
		},
	}

	err := h.recordCheckAsset("asset.one", root, false)
	require.NoError(t, err)
	body, err := os.ReadFile(generated)
	require.NoError(t, err)
	require.JSONEq(t, `{"generated":true}`, string(body))
	rows, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "types", rows[0].Gate)
}
