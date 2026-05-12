package verifications_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	handlers "flow-verifier/handlers/verifications"
	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/runs"
	"flow-verifier/internal/server"
	"flow-verifier/internal/testkit"
	"flow-verifier/internal/testutil/assertx"
	"flow-verifier/internal/testutil/db"
	"flow-verifier/internal/testutil/httpx"
	"flow-verifier/internal/testutil/mocks"

	apidb "github.com/vrooli/api-core/database"

	"github.com/stretchr/testify/require"
)

func newVerificationsLive(t *testing.T) (*httpx.LiveServer, *runs.Service, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(runs.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	svc := runs.NewService(runs.NewSQLiteRepository(d, clk))
	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
		handlers.ModuleWithService(svc),
	)
	return httpx.NewLiveServer(t, srv), svc, clk
}

type postResponse struct {
	Status string     `json:"status"`
	Error  string     `json:"error,omitempty"`
	Runs   []runs.Run `json:"runs"`
}

func postJSON(t *testing.T, live *httpx.LiveServer, body any) (*http.Response, []byte) {
	t.Helper()
	buf, err := json.Marshal(body)
	require.NoError(t, err)
	return live.Do(t, http.MethodPost, "/api/v1/verifications", bytes.NewReader(buf))
}

// TestModule_Shape pins the public contract of the module.
func TestModule_Shape(t *testing.T) {
	_, svc, _ := newVerificationsLive(t)
	mod := handlers.ModuleWithService(svc)
	require.Equal(t, "verifications", mod.Name)
	require.NotNil(t, mod.Mount)
	require.NotEmpty(t, mod.Endpoints,
		"verifications ships POST /api/v1/verifications and GET /api/v1/verifications/{runId}")
}

// TestPost_BadBody asserts that a malformed JSON body returns a 400
// envelope rather than a 500.
func TestPost_BadBody(t *testing.T) {
	live, _, _ := newVerificationsLive(t)
	resp, _ := live.Do(t, http.MethodPost, "/api/v1/verifications", bytes.NewReader([]byte("not-json")))
	assertx.AssertStatus(t, resp, http.StatusBadRequest)
}

// TestPost_BadMode rejects unknown mode strings rather than silently
// defaulting — the UI should never see a 200 hiding an unintended mode.
func TestPost_BadMode(t *testing.T) {
	live, _, _ := newVerificationsLive(t)
	root := t.TempDir()
	resp, _ := postJSON(t, live, map[string]string{"root": root, "mode": "explode"})
	assertx.AssertStatus(t, resp, http.StatusBadRequest)
}

// TestPost_EmptyRootPasses covers the happy edge case the dashboard hits
// on a fresh project: no flows on disk → pipeline reports "passed" with
// no runs recorded.
func TestPost_EmptyRootPasses(t *testing.T) {
	live, svc, _ := newVerificationsLive(t)
	root := t.TempDir()

	resp, body := postJSON(t, live, map[string]string{"root": root, "mode": "check"})
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[postResponse](t, body)
	require.Equal(t, "passed", got.Status)
	require.Empty(t, got.Runs, "no flows discovered means no runs recorded")

	// And the service-side store agrees.
	stored, err := svc.List(context.Background(), runs.ListQuery{})
	require.NoError(t, err)
	require.Empty(t, stored)
}

// TestPost_SingleFlowRecordsOneRow is the sync roundtrip contract: POST
// with one flow on disk → exactly one row recorded via the runs.Service
// seam and returned in the response. Uses Mode=check, which fails fast
// on the missing hand-authored sidecars without needing the full quint
// pipeline to execute — that keeps the test focused on the recorder
// seam (the one this handler owns).
func TestPost_SingleFlowRecordsOneRow(t *testing.T) {
	live, svc, _ := newVerificationsLive(t)
	root := t.TempDir()
	testkit.WriteFlowJSON(t, root, "api/example/flow/flow.json", testkit.ValidRawContract())

	resp, body := postJSON(t, live, map[string]string{"root": root, "mode": "check"})
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[postResponse](t, body)
	require.Equal(t, "failed", got.Status,
		"check mode against a flow with no hand-authored sidecars must fail; error: %s", got.Error)
	require.NotEmpty(t, got.Error)
	require.Len(t, got.Runs, 1, "exactly one row per discovered flow")
	require.Equal(t, "example.workflow.api", got.Runs[0].FlowID)
	require.Equal(t, runs.StatusFailed, got.Runs[0].Status)
	require.NotEmpty(t, got.Runs[0].ID, "recorder must populate id")

	// And the service-side store agrees: a second list query returns
	// the same row, proving the recorder wrote through to SQLite, not
	// just into the response buffer.
	stored, err := svc.List(context.Background(), runs.ListQuery{})
	require.NoError(t, err)
	require.Len(t, stored, 1)
	require.Equal(t, got.Runs[0].ID, stored[0].ID)
}

// TestGet_UnknownRun asserts the 404 envelope.
func TestGet_UnknownRun(t *testing.T) {
	live, _, _ := newVerificationsLive(t)
	resp, _ := live.Do(t, http.MethodGet, "/api/v1/verifications/does-not-exist", nil)
	assertx.AssertStatus(t, resp, http.StatusNotFound)
}

// TestGet_KnownRun is the contract the Run Detail page leans on:
// retrieving a previously-recorded run by id returns the same row shape,
// counterexample blob included.
func TestGet_KnownRun(t *testing.T) {
	live, svc, clk := newVerificationsLive(t)
	now := clk.Now()
	inserted, err := svc.Record(context.Background(), runs.Run{
		FlowID: "example.workflow.api", FlowPath: "api/example/flow/flow.json",
		Root: "/r", Mode: runs.ModeCheck, Status: runs.StatusFailed,
		Counterexample: `{"violated":"safety"}`,
		ErrorMessage:   "boom",
		StartedAt:      now, FinishedAt: now.Add(time.Second),
	})
	require.NoError(t, err)

	resp, body := live.Do(t, http.MethodGet, "/api/v1/verifications/"+inserted.ID, nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[runs.Run](t, body)
	require.Equal(t, inserted.ID, got.ID)
	require.Equal(t, runs.StatusFailed, got.Status)
	require.Equal(t, inserted.Counterexample, got.Counterexample)
}
