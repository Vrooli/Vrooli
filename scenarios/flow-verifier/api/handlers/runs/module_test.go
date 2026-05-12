package runs_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	handlers "flow-verifier/handlers/runs"
	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/runs"
	"flow-verifier/internal/server"
	"flow-verifier/internal/testutil/assertx"
	"flow-verifier/internal/testutil/db"
	"flow-verifier/internal/testutil/httpx"
	"flow-verifier/internal/testutil/mocks"

	apidb "github.com/vrooli/api-core/database"

	"github.com/stretchr/testify/require"
)

// newRunsLive wires the runs module behind a real httptest server with
// an empty in-memory SQLite database and a fake clock. Callers seed
// state by inserting into the returned *runs.Service.
func newRunsLive(t *testing.T) (*httpx.LiveServer, *runs.Service, *mocks.FakeClock) {
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

func mustRecord(t *testing.T, svc *runs.Service, in runs.Run) runs.Run {
	t.Helper()
	got, err := svc.Record(context.Background(), in)
	require.NoError(t, err)
	return got
}

// TestModule_Shape pins the public contract of the module.
func TestModule_Shape(t *testing.T) {
	live, svc, _ := newRunsLive(t)
	_ = live

	// Re-derive the module so we can introspect its descriptor without
	// going through the server harness.
	mod := handlers.ModuleWithService(svc)
	require.Equal(t, "runs", mod.Name)
	require.NotNil(t, mod.Mount)
	require.NotEmpty(t, mod.Endpoints, "runs ships GET /api/v1/runs and GET /api/v1/runs/{id}")
}

// TestList_Empty asserts that an empty store yields an empty array, not
// null, so the UI doesn't have to special-case the response shape.
func TestList_Empty(t *testing.T) {
	live, _, _ := newRunsLive(t)

	resp, body := live.Do(t, http.MethodGet, "/api/v1/runs", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustDecodeJSON[map[string]any](t, body)
	rows, ok := got["runs"].([]any)
	require.True(t, ok, "response must include runs array; got %v", got)
	require.Empty(t, rows)
}

// TestList_FilterAndOrdering covers the flowId filter and the newest-
// first ordering contract.
func TestList_FilterAndOrdering(t *testing.T) {
	live, svc, clk := newRunsLive(t)
	now := clk.Now()
	mustRecord(t, svc, runs.Run{
		FlowID: "a", FlowPath: "a.json", Root: "/r", Mode: runs.ModeCheck,
		Status: runs.StatusPassed, StartedAt: now.Add(-2 * time.Second), FinishedAt: now.Add(1 * time.Minute),
	})
	mustRecord(t, svc, runs.Run{
		FlowID: "b", FlowPath: "b.json", Root: "/r", Mode: runs.ModeCheck,
		Status: runs.StatusFailed, StartedAt: now.Add(-2 * time.Second), FinishedAt: now.Add(2 * time.Minute),
	})
	mustRecord(t, svc, runs.Run{
		FlowID: "a", FlowPath: "a.json", Root: "/r", Mode: runs.ModeCheck,
		Status: runs.StatusPassed, StartedAt: now.Add(-2 * time.Second), FinishedAt: now.Add(3 * time.Minute),
	})

	resp, body := live.Do(t, http.MethodGet, "/api/v1/runs", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[map[string][]runs.Run](t, body)
	require.Len(t, got["runs"], 3)
	require.Equal(t, "a", got["runs"][0].FlowID, "newest first")
	require.Equal(t, "b", got["runs"][1].FlowID)

	resp, body = live.Do(t, http.MethodGet, "/api/v1/runs?flowId=a", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got = assertx.MustDecodeJSON[map[string][]runs.Run](t, body)
	require.Len(t, got["runs"], 2)
	for _, r := range got["runs"] {
		require.Equal(t, "a", r.FlowID)
	}

	resp, body = live.Do(t, http.MethodGet, "/api/v1/runs?limit=1", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got = assertx.MustDecodeJSON[map[string][]runs.Run](t, body)
	require.Len(t, got["runs"], 1)
}

// TestList_BadLimit asserts that a malformed limit produces a 400 with
// the canonical invalid_request envelope rather than a 500.
func TestList_BadLimit(t *testing.T) {
	live, _, _ := newRunsLive(t)
	for _, raw := range []string{"abc", "-1"} {
		t.Run(raw, func(t *testing.T) {
			resp, _ := live.Do(t, http.MethodGet, "/api/v1/runs?limit="+raw, nil)
			assertx.AssertStatus(t, resp, http.StatusBadRequest)
		})
	}
}

// TestGet_NotFound asserts the 404 envelope for an unknown run id.
func TestGet_NotFound(t *testing.T) {
	live, _, _ := newRunsLive(t)
	resp, _ := live.Do(t, http.MethodGet, "/api/v1/runs/does-not-exist", nil)
	assertx.AssertStatus(t, resp, http.StatusNotFound)
}

// TestGet_CounterexamplePassthrough is the contract the Flow Detail UI
// depends on: the counterexample blob the recorder stored must round-trip
// through HTTP byte-for-byte so the Counterexample Diff view can render
// it without re-decoding.
func TestGet_CounterexamplePassthrough(t *testing.T) {
	live, svc, clk := newRunsLive(t)
	now := clk.Now()
	inserted := mustRecord(t, svc, runs.Run{
		FlowID: "notes.attachment-upload.ui", FlowPath: "ui/notes/flow/flow.json",
		Root: "/r", Mode: runs.ModeCheck, Status: runs.StatusFailed,
		Counterexample: `{"violated":"safety","trace":[{"state":"idle"},{"state":"busy"}]}`,
		ErrorMessage:   "quint produced counterexample",
		StartedAt:      now, FinishedAt: now.Add(time.Second),
	})

	resp, body := live.Do(t, http.MethodGet, "/api/v1/runs/"+inserted.ID, nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[runs.Run](t, body)
	require.Equal(t, inserted.ID, got.ID)
	require.Equal(t, runs.StatusFailed, got.Status)
	require.Equal(t, inserted.Counterexample, got.Counterexample)
	require.Equal(t, "quint produced counterexample", got.ErrorMessage)
}
