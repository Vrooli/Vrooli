package asset

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"
	clitest "github.com/vrooli/cli-core/cliapptest"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogcoverage"
)

type recordingCatalog struct {
	store        *catalogcoverage.EvidenceStore
	ctx          context.Context
	t            *testing.T
	checkResp    *catalogv1.CheckAssetResponse
	checkErr     error
	checkRequest *catalogv1.CheckAssetRequest
}

func TestTotalStageSecondsSumsAllStages(t *testing.T) {
	require.Equal(t, 3.75, totalStageSeconds([]*catalogv1.AssetCheckStage{
		{Name: "types", Seconds: 1.25},
		{Name: "tests", Seconds: 2.5},
	}))
	require.Equal(t, float64(0), totalStageSeconds(nil))
}

func TestExitErrorPreservesMessageAndStatus(t *testing.T) {
	err := exitError{code: 7, message: "asset is blocked"}
	require.EqualError(t, err, "asset is blocked")
	require.Equal(t, 7, err.ExitCode())
}

func TestBuildDerivedCatalogPropagatesCommandFailure(t *testing.T) {
	bin := t.TempDir()
	node := filepath.Join(bin, "node")
	require.NoError(t, os.WriteFile(node, []byte("#!/bin/sh\nexit 19\n"), 0o700))
	t.Setenv("PATH", bin)
	root := filepath.Join(t.TempDir(), "scenarios", "react-component-library")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "..", "..", "packages", "react-component-library"), 0o700))
	err := buildDerivedCatalog(root)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 19, exitErr.ExitCode())
}

func TestBuildDerivedCatalogReportsMissingExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := filepath.Join(t.TempDir(), "scenarios", "react-component-library")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "..", "..", "packages", "react-component-library"), 0o700))
	err := buildDerivedCatalog(root)
	require.Error(t, err)
	require.True(t, errors.Is(err, exec.ErrNotFound))
}

func (c recordingCatalog) RecordGate(_ context.Context, req *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	require.True(c.t, req.Msg.GetAll())
	require.Equal(c.t, "asset.one", req.Msg.GetAssetId())
	err := c.store.Save(c.ctx, []catalogcoverage.GateEvidence{{AssetID: "asset.one", Target: "catalog", Gate: "types", Version: "1.0.0", Result: "pass", SourceRevision: "rev", RecordedAt: "2026-11-15T00:00:00Z"}})
	return connect.NewResponse(&catalogv1.RunGateResponse{EvidenceRowsWritten: 1}), err
}

func (c *recordingCatalog) CheckAsset(_ context.Context, req *connect.Request[catalogv1.CheckAssetRequest]) (*connect.Response[catalogv1.CheckAssetResponse], error) {
	c.checkRequest = req.Msg
	if c.checkErr != nil {
		return nil, c.checkErr
	}
	if c.checkResp == nil {
		c.checkResp = &catalogv1.CheckAssetResponse{AssetId: "asset.one", Verdict: "PUBLISHABLE"}
	}
	return connect.NewResponse(c.checkResp), nil
}

func TestCheckValidatesArgumentsAndVerdicts(t *testing.T) {
	core := clitest.NewTestApp(t, nil)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "asset-id"}}, Flags: []cliapp.Flag{{Name: "profile"}, {Name: "version"}, {Name: "json"}}}
	for _, tc := range []struct {
		name        string
		positionals map[string]string
		flags       map[string]string
		verdict     string
		wantErr     string
		wantCode    int
	}{
		{name: "missing asset", wantErr: "usage: asset check", wantCode: 2},
		{name: "invalid profile", positionals: map[string]string{"asset-id": "asset.one"}, flags: map[string]string{"profile": "preview"}, wantErr: "profile must be publish or edit", wantCode: 2},
		{name: "blocked", positionals: map[string]string{"asset-id": "asset.one"}, verdict: "BLOCKED", wantErr: "asset asset.one: BLOCKED", wantCode: 1},
		{name: "stale tests", positionals: map[string]string{"asset-id": "asset.one"}, verdict: "STALE_TESTS", wantErr: "asset asset.one: STALE_TESTS", wantCode: 1},
		{name: "stale catalog", positionals: map[string]string{"asset-id": "asset.one"}, verdict: "STALE_CATALOG", wantErr: "run `react-component-library asset record-check asset.one`", wantCode: 1},
		{name: "unknown verdict", positionals: map[string]string{"asset-id": "asset.one"}, verdict: "UNAVAILABLE", wantErr: "asset asset.one: UNAVAILABLE", wantCode: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := &recordingCatalog{checkResp: &catalogv1.CheckAssetResponse{AssetId: "asset.one", Verdict: tc.verdict}}
			h := &handlers{catalog: catalog}
			ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Positionals: tc.positionals, Flags: tc.flags})
			err := h.check(ctx)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
			var coded interface{ ExitCode() int }
			require.ErrorAs(t, err, &coded)
			require.Equal(t, tc.wantCode, coded.ExitCode())
		})
	}
}

func TestCheckPublishesHumanAndJSONVerdictsAndForwardsNormalizedInputs(t *testing.T) {
	core := clitest.NewTestApp(t, nil)
	for _, jsonOutput := range []bool{false, true} {
		t.Run(map[bool]string{false: "human", true: "json"}[jsonOutput], func(t *testing.T) {
			catalog := &recordingCatalog{checkResp: &catalogv1.CheckAssetResponse{AssetId: "asset.one", Verdict: "PUBLISHABLE", Stages: []*catalogv1.AssetCheckStage{{Name: "types", Status: "pass", Detail: "ok", Seconds: 1.5}}}}
			h := &handlers{catalog: catalog}
			flags := map[string]string{"profile": " edit ", "version": " 1.2.0 "}
			if jsonOutput {
				flags["json"] = "true"
			}
			ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "asset-id"}}, Flags: []cliapp.Flag{{Name: "profile"}, {Name: "version"}, {Name: "json"}}}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"asset-id": " asset.one "}, Flags: flags})
			require.NoError(t, h.check(ctx))
			require.Equal(t, "asset.one", catalog.checkRequest.AssetId)
			require.Equal(t, "edit", catalog.checkRequest.Profile)
			require.Equal(t, "1.2.0", catalog.checkRequest.Version)
		})
	}
}

type assetCatalogStub struct {
	checkResp               *catalogv1.CheckAssetResponse
	checkErr                error
	recordResp              *catalogv1.RunGateResponse
	recordErr               error
	checkCalls, recordCalls int
}

func (s *assetCatalogStub) CheckAsset(_ context.Context, _ *connect.Request[catalogv1.CheckAssetRequest]) (*connect.Response[catalogv1.CheckAssetResponse], error) {
	s.checkCalls++
	if s.checkErr != nil {
		return nil, s.checkErr
	}
	if s.checkResp == nil {
		return nil, nil
	}
	return connect.NewResponse(s.checkResp), nil
}

func (s *assetCatalogStub) RecordGate(_ context.Context, _ *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	s.recordCalls++
	if s.recordErr != nil {
		return nil, s.recordErr
	}
	if s.recordResp == nil {
		return nil, nil
	}
	return connect.NewResponse(s.recordResp), nil
}

func TestRecordCheckAssetStopsAtEachFailedStep(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		buildErr, recordErr, checkErr error
		recordResp                    *catalogv1.RunGateResponse
		checkResp                     *catalogv1.CheckAssetResponse
		want                          string
	}{
		{name: "refresh failure", buildErr: errors.New("build failed"), want: "refresh derived catalog files"},
		{name: "record failure", recordErr: errors.New("write failed"), recordResp: &catalogv1.RunGateResponse{}, want: "record asset gate evidence"},
		{name: "missing recorded response", want: "no recorded gate result"},
		{name: "check failure", recordResp: &catalogv1.RunGateResponse{}, checkErr: errors.New("read failed"), want: "check asset after recording evidence"},
		{name: "missing verdict", recordResp: &catalogv1.RunGateResponse{}, want: "no asset check verdict"},
		{name: "blocked", recordResp: &catalogv1.RunGateResponse{}, checkResp: &catalogv1.CheckAssetResponse{Verdict: "BLOCKED"}, want: "asset asset.one: BLOCKED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := &assetCatalogStub{checkResp: tc.checkResp, checkErr: tc.checkErr, recordResp: tc.recordResp, recordErr: tc.recordErr}
			h := &handlers{catalog: catalog, buildCatalog: func(string) error { return tc.buildErr }}
			err := h.recordCheckAsset("asset.one", t.TempDir(), false)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestRecordCheckAssetRendersJSONAfterSuccessfulWrite(t *testing.T) {
	catalog := &assetCatalogStub{recordResp: &catalogv1.RunGateResponse{EvidenceRowsWritten: 3}, checkResp: &catalogv1.CheckAssetResponse{AssetId: "asset.one", Verdict: "PUBLISHABLE"}}
	h := &handlers{catalog: catalog, buildCatalog: func(string) error { return nil }}
	require.NoError(t, h.recordCheckAsset("asset.one", t.TempDir(), true))
	require.Equal(t, 1, catalog.recordCalls)
	require.Equal(t, 1, catalog.checkCalls)
}

func TestRecordCheckRefreshesAndPersistsEvidence(t *testing.T) {
	ctx := context.Background()
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(catalogcoverage.Schema)))
	store := catalogcoverage.NewEvidenceStore(database)
	root := t.TempDir()
	generated := filepath.Join(root, "generated", "component-manifest.json")
	h := &handlers{
		catalog: &recordingCatalog{store: store, ctx: ctx, t: t},
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
