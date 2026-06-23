package completeness

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
	scoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring/scoring_v1connect"
)

// stubScoreHandler implements the generated ScoreServiceHandler with canned
// GetScore output, standing in for a real completeness-scoring instance.
type stubScoreHandler struct {
	resp *scoringv1.GetScoreResponse
	err  error
}

func (s *stubScoreHandler) GetScore(_ context.Context, _ *connect.Request[scoringv1.GetScoreRequest]) (*connect.Response[scoringv1.GetScoreResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.resp), nil
}

func (s *stubScoreHandler) GetScoreTrend(context.Context, *connect.Request[scoringv1.GetScoreTrendRequest]) (*connect.Response[scoringv1.GetScoreTrendResponse], error) {
	return connect.NewResponse(&scoringv1.GetScoreTrendResponse{}), nil
}

func (s *stubScoreHandler) ListScores(context.Context, *connect.Request[scoringv1.ListScoresRequest]) (*connect.Response[scoringv1.ListScoresResponse], error) {
	return connect.NewResponse(&scoringv1.ListScoresResponse{}), nil
}

// newStubClient mounts the stub handler on an httptest server and returns a
// Client whose resolver points at it (bypassing the discovery registry).
func newStubClient(t *testing.T, h *stubScoreHandler) *Client {
	t.Helper()
	path, handler := scoringconnect.NewScoreServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Client{
		httpClient: srv.Client(),
		resolve:    func(context.Context) (string, error) { return srv.URL, nil },
	}
}

func TestClientScore_MapsPayload(t *testing.T) {
	c := newStubClient(t, &stubScoreHandler{resp: &scoringv1.GetScoreResponse{
		Maturity:  &scoringv1.MaturityHeadline{WorkingRung: "R1 Safe & standards-clean", BuildPassing: true},
		Composite: otGroup("12 total, 6 passing (50%)"),
	}})
	got, err := c.Score(context.Background(), "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.WorkingRung != "R1 Safe & standards-clean" || !got.BuildPassing {
		t.Errorf("rung/build mismatch: %+v", got)
	}
	if got.OTTotal != 12 || got.OTPassing != 6 || got.OTPercentage != 50 {
		t.Errorf("OT mismatch: %+v", got)
	}
}

// TestClientScore_PropagatesError proves the D2 contract: a transport/RPC error
// is returned (not swallowed into a zero-value fallback), so the controller can
// degrade loudly.
func TestClientScore_PropagatesError(t *testing.T) {
	c := newStubClient(t, &stubScoreHandler{err: connect.NewError(connect.CodeUnavailable, errors.New("down"))})
	if _, err := c.Score(context.Background(), "demo"); err == nil {
		t.Fatal("expected an error to propagate, got nil")
	}
}

func TestClientScore_ResolveErrorPropagates(t *testing.T) {
	c := &Client{
		httpClient: http.DefaultClient,
		resolve:    func(context.Context) (string, error) { return "", errors.New("no registry") },
	}
	if _, err := c.Score(context.Background(), "demo"); err == nil {
		t.Fatal("expected resolve error to propagate")
	}
}
