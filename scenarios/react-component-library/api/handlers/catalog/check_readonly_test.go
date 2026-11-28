package catalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogbuild"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	componenttests "react-component-library/internal/componenttests"
)

func TestCheckAssetStaleProjectionNeverRepairsFilesOrEvidence(t *testing.T) {
	h, store, ctx := readOnlyCheckFixture(t)
	marker := filepath.Join(h.repoRoot, "generated", "marker.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(marker), 0o700))
	require.NoError(t, os.WriteFile(marker, []byte("before"), 0o600))
	beforeRows, err := store.List(ctx)
	require.NoError(t, err)
	builds, gateCalls, testCalls := 0, 0, 0
	h.checkBuild = func(_ string, opts catalogbuild.Options) (catalogbuild.Report, error) {
		builds++
		require.True(t, opts.Check)
		return catalogbuild.Report{Status: "stale"}, errors.New("generated projection is stale")
	}
	h.checkGate = func(context.Context, string, string) (*connect.Response[catalogv1.RunGateResponse], error) {
		gateCalls++
		return connect.NewResponse(&catalogv1.RunGateResponse{}), nil
	}
	h.checkTest = func(context.Context, componenttests.Request) (componenttests.Report, bool, error) {
		testCalls++
		return componenttests.Report{}, false, nil
	}

	for attempt := 0; attempt < 3; attempt++ {
		response, callErr := h.CheckAsset(ctx, connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "asset.one", RunTests: true}))
		require.NoError(t, callErr)
		require.Equal(t, "STALE_CATALOG", response.Msg.GetVerdict())
		body, readErr := os.ReadFile(marker)
		require.NoError(t, readErr)
		require.Equal(t, "before", string(body))
		afterRows, listErr := store.List(ctx)
		require.NoError(t, listErr)
		require.Equal(t, beforeRows, afterRows, "read attempt %d changed evidence", attempt+1)
	}
	require.Equal(t, 3, builds)
	require.Zero(t, gateCalls)
	require.Zero(t, testCalls)
}

func TestCheckAssetRunTestsFlagStillOnlyReusesAndDoesNotWrite(t *testing.T) {
	h, store, ctx := readOnlyCheckFixture(t)
	marker := filepath.Join(h.repoRoot, "generated", "marker.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(marker), 0o700))
	require.NoError(t, os.WriteFile(marker, []byte("before"), 0o600))
	beforeRows, err := store.List(ctx)
	require.NoError(t, err)
	builds, gateCalls, testCalls := 0, 0, 0
	h.checkBuild = func(_ string, opts catalogbuild.Options) (catalogbuild.Report, error) {
		builds++
		require.True(t, opts.Check)
		return catalogbuild.Report{Status: "current"}, nil
	}
	h.checkGate = func(gateCtx context.Context, gate, assetID string) (*connect.Response[catalogv1.RunGateResponse], error) {
		require.Equal(t, "asset.one", assetID)
		_, skipCalibration := gateCtx.Value(skipCalibrationContextKey{}).(bool)
		require.True(t, skipCalibration, "read checks must not update calibration quarantine state")
		gateCalls++
		return connect.NewResponse(&catalogv1.RunGateResponse{}), nil
	}
	h.checkAsset = func(context.Context, string) (components.Component, error) {
		return components.Component{ID: "component-id", CatalogID: "asset.one", LibraryID: "library:One", LatestVersion: "1.2.3", AssetKind: components.AssetKindComponent}, nil
	}
	h.checkTest = func(_ context.Context, req componenttests.Request) (componenttests.Report, bool, error) {
		testCalls++
		require.Equal(t, "library:One", req.ComponentID)
		return componenttests.Report{}, false, nil
	}

	for attempt := 0; attempt < 3; attempt++ {
		response, callErr := h.CheckAsset(ctx, connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: "asset.one", RunTests: true}))
		require.NoError(t, callErr)
		require.Equal(t, "STALE_TESTS", response.Msg.GetVerdict())
		body, readErr := os.ReadFile(marker)
		require.NoError(t, readErr)
		require.Equal(t, "before", string(body))
		afterRows, listErr := store.List(ctx)
		require.NoError(t, listErr)
		require.Equal(t, beforeRows, afterRows, "read attempt %d changed evidence", attempt+1)
	}
	require.Equal(t, 3, builds)
	require.Equal(t, 6, gateCalls, "one all-gate and one package gate inspection per call")
	require.Equal(t, 3, testCalls)
}

func readOnlyCheckFixture(t *testing.T) (*handler, *catalogcoverage.EvidenceStore, context.Context) {
	t.Helper()
	ctx := context.Background()
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(catalogcoverage.Schema)))
	store := catalogcoverage.NewEvidenceStore(database)
	require.NoError(t, store.Save(ctx, []catalogcoverage.GateEvidence{{AssetID: "seed", Target: "test", Gate: "release-provenance", Version: "1.0.0", Result: "pass", SourceRevision: "seed", RecordedAt: "2026-11-15T00:00:00Z"}}))
	return &handler{repoRoot: t.TempDir(), evidence: store, checkCache: map[string]*catalogv1.CheckAssetResponse{}}, store, ctx
}
