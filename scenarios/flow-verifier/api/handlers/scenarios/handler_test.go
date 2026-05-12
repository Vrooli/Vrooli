package scenarios_test

import (
	"errors"
	"io"
	"log"
	"net/http"
	"testing"

	handlers "flow-verifier/handlers/scenarios"
	"flow-verifier/internal/clock"
	"flow-verifier/internal/scenarios"
	"flow-verifier/internal/server"
	"flow-verifier/internal/testutil/assertx"
	"flow-verifier/internal/testutil/httpx"

	"github.com/stretchr/testify/require"
)

// stubService is a hand-rolled fake of the scenarios.Service surface
// the handler depends on. Hand-rolled (not generated) so test setup
// reads as data rather than expectation chains.
type stubService struct {
	rows    []scenarios.Summary
	detail  scenarios.Detail
	root    string
	listErr error
	detErr  error
}

func (s *stubService) List() ([]scenarios.Summary, error) { return s.rows, s.listErr }
func (s *stubService) Detail(id string) (scenarios.Detail, error) {
	if s.detErr != nil {
		return scenarios.Detail{}, s.detErr
	}
	if id != s.detail.ID {
		return scenarios.Detail{}, errors.New("unexpected id: " + id)
	}
	return s.detail, nil
}
func (s *stubService) Root() string { return s.root }

func newLive(t *testing.T, svc handlers.Service) *httpx.LiveServer {
	t.Helper()
	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
		handlers.Module(svc),
	)
	return httpx.NewLiveServer(t, srv)
}

func TestModule_Shape(t *testing.T) {
	mod := handlers.Module(&stubService{})
	require.Equal(t, "scenarios", mod.Name)
	require.NotNil(t, mod.Mount)
	require.NotEmpty(t, mod.Endpoints, "scenarios ships GET /api/v1/scenarios and GET /api/v1/scenarios/{id}")
}

func TestList_HappyPath(t *testing.T) {
	svc := &stubService{
		root: "/repo",
		rows: []scenarios.Summary{
			{ID: "alpha", DisplayName: "Alpha", FlowCount: 3, Path: "/repo/scenarios/alpha"},
			{ID: "beta", DisplayName: "Beta", FlowCount: 0, Path: "/repo/scenarios/beta"},
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, http.MethodGet, "/api/v1/scenarios", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[map[string]any](t, body)
	require.Equal(t, "/repo", got["vrooliRoot"])
	rows, ok := got["scenarios"].([]any)
	require.True(t, ok, "scenarios must be an array; got %v", got)
	require.Len(t, rows, 2)
	first := rows[0].(map[string]any)
	require.Equal(t, "alpha", first["id"])
	require.EqualValues(t, 3, first["flowCount"])
}

func TestList_EmptyReturnsArrayNotNull(t *testing.T) {
	// Inventory page treats null and [] differently; the empty case is
	// "I scanned, nothing was there" not "scan failed".
	svc := &stubService{root: "/repo", rows: []scenarios.Summary{}}
	live := newLive(t, svc)
	resp, body := live.Do(t, http.MethodGet, "/api/v1/scenarios", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[map[string]any](t, body)
	rows, ok := got["scenarios"].([]any)
	require.True(t, ok)
	require.Empty(t, rows)
}

func TestList_ErrorPropagates(t *testing.T) {
	svc := &stubService{listErr: errors.New("disk gone")}
	live := newLive(t, svc)
	resp, _ := live.Do(t, http.MethodGet, "/api/v1/scenarios", nil)
	assertx.AssertStatus(t, resp, http.StatusInternalServerError)
}

func TestGet_HappyPath(t *testing.T) {
	svc := &stubService{detail: scenarios.Detail{
		Summary: scenarios.Summary{ID: "alpha", DisplayName: "Alpha", FlowCount: 1},
	}}
	live := newLive(t, svc)
	resp, body := live.Do(t, http.MethodGet, "/api/v1/scenarios/alpha", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[map[string]any](t, body)
	require.Equal(t, "alpha", got["id"])
}

func TestGet_NotFound(t *testing.T) {
	svc := &stubService{detErr: scenarios.ErrScenarioNotFound}
	live := newLive(t, svc)
	resp, _ := live.Do(t, http.MethodGet, "/api/v1/scenarios/missing", nil)
	assertx.AssertStatus(t, resp, http.StatusNotFound)
}
