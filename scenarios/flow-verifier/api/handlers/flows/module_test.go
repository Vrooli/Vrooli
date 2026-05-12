package flows_test

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"testing"

	handlers "flow-verifier/handlers/flows"
	"flow-verifier/internal/clock"
	"flow-verifier/internal/flows"
	"flow-verifier/internal/server"
	"flow-verifier/internal/testkit"
	"flow-verifier/internal/testutil/assertx"
	"flow-verifier/internal/testutil/httpx"

	"github.com/stretchr/testify/require"
)

// newFlowsLive wires the flows module behind a real httptest server.
// flows discovery is filesystem-truth, so callers seed by writing
// flow.json files under the returned root and passing it as ?root=.
func newFlowsLive(t *testing.T) *httpx.LiveServer {
	t.Helper()
	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
		handlers.Module(),
	)
	return httpx.NewLiveServer(t, srv)
}

// TestModule_Shape pins the public contract of the module.
func TestModule_Shape(t *testing.T) {
	mod := handlers.Module()
	require.Equal(t, "flows", mod.Name)
	require.NotNil(t, mod.Mount)
	require.NotEmpty(t, mod.Endpoints, "flows ships GET /api/v1/flows and GET /api/v1/flows/{id}")
}

// TestList_EmptyRoot is the path the Inventory page hits on first load
// before the user has flows on disk: 200 with an empty array (not null).
func TestList_EmptyRoot(t *testing.T) {
	live := newFlowsLive(t)
	root := t.TempDir()

	resp, body := live.Do(t, http.MethodGet, "/api/v1/flows?root="+url.QueryEscape(root), nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[map[string]any](t, body)
	rows, ok := got["flows"].([]any)
	require.True(t, ok, "response must include flows array; got %v", got)
	require.Empty(t, rows)
}

// TestList_SingleFlow seeds one valid flow under the temp root and
// asserts the summary the UI consumes (flowId, language, schemaVersion).
func TestList_SingleFlow(t *testing.T) {
	live := newFlowsLive(t)
	root := t.TempDir()
	testkit.WriteFlowJSON(t, root, "api/example/flow/flow.json", testkit.ValidRawContract())

	resp, body := live.Do(t, http.MethodGet, "/api/v1/flows?root="+url.QueryEscape(root), nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[map[string][]flows.Summary](t, body)
	require.Len(t, got["flows"], 1)
	require.Equal(t, "example.workflow.api", got["flows"][0].FlowID)
	require.Equal(t, "go", got["flows"][0].Language)
	require.NotZero(t, got["flows"][0].SchemaVer)
}

// TestList_SchemaInvalid covers the unhappy path the UI sees when a flow
// on disk fails to compile: 400 with the invalid_request envelope, not a
// 500 or silent skip.
func TestList_SchemaInvalid(t *testing.T) {
	live := newFlowsLive(t)
	root := t.TempDir()
	// Drop a flow.json that's syntactically JSON but semantically empty
	// — compile reports schemaVersion + required-field violations.
	testkit.WriteFile(t, root, "api/broken/flow/flow.json", `{"flowId":""}`)

	resp, _ := live.Do(t, http.MethodGet, "/api/v1/flows?root="+url.QueryEscape(root), nil)
	assertx.AssertStatus(t, resp, http.StatusBadRequest)
}

// TestGet_UnknownFlow asserts the 404 envelope the Flow Detail page
// shows when a user navigates to a flowId that no longer exists.
func TestGet_UnknownFlow(t *testing.T) {
	live := newFlowsLive(t)
	root := t.TempDir()

	resp, _ := live.Do(t, http.MethodGet, "/api/v1/flows/does.not.exist?root="+url.QueryEscape(root), nil)
	assertx.AssertStatus(t, resp, http.StatusNotFound)
}

// TestGet_DetailPayload is the contract the Flow Detail page leans on:
// the response is the structured FlowDetail with states/events/
// transitions/traces plus the text report (kept for the CLI's parity).
func TestGet_DetailPayload(t *testing.T) {
	live := newFlowsLive(t)
	root := t.TempDir()
	testkit.WriteFlowJSON(t, root, "api/example/flow/flow.json", testkit.ValidRawContract())

	resp, body := live.Do(t, http.MethodGet,
		"/api/v1/flows/example.workflow.api?root="+url.QueryEscape(root), nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[flows.FlowDetail](t, body)
	require.Equal(t, "example.workflow.api", got.FlowID)
	require.Equal(t, "go", got.Language)
	require.NotEmpty(t, got.States, "states must be present for the state-graph viewer")
	require.NotEmpty(t, got.Events, "events must be present for the state-graph viewer")
	require.NotEmpty(t, got.Transitions, "transitions matrix must be present for graph edges")
	require.NotEmpty(t, got.InitialState, "initialState must identify the entry node")
	require.Contains(t, got.Report, "example.workflow.api")
}
