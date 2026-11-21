package scenarios

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
	scoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring/scoring_v1connect"
)

func TestSCSCompletenessSourceScoresPagesAndClamps(t *testing.T) {
	server := newScoringListServer(t, map[string]*scoringv1.ListScoresResponse{
		"": {
			Scores: []*scoringv1.ScoreRow{
				{Scenario: "alpha", Score: -4},
				{Scenario: "beta", Score: 88},
				{Scenario: "  ", Score: 50},
			},
			NextPageToken: "next",
		},
		"next": {
			Scores: []*scoringv1.ScoreRow{
				{Scenario: "gamma", Score: 140},
			},
		},
	})
	defer server.Close()

	source := NewSCSCompletenessSourceWithDeps(
		time.Second,
		staticURLResolver{url: server.URL},
		server.Client(),
	)
	got, err := source.Scores(context.Background())
	if err != nil {
		t.Fatalf("Scores() error = %v", err)
	}

	want := map[string]int{
		"alpha": 0,
		"beta":  88,
		"gamma": 100,
	}
	if len(got) != len(want) {
		t.Fatalf("Scores() len = %d, want %d: %#v", len(got), len(want), got)
	}
	for scenario, score := range want {
		if got[scenario] != score {
			t.Fatalf("Scores()[%q] = %d, want %d", scenario, got[scenario], score)
		}
	}
}

func TestSCSCompletenessSourceReturnsResolveError(t *testing.T) {
	source := NewSCSCompletenessSourceWithDeps(
		time.Second,
		staticURLResolver{err: errors.New("not running")},
		http.DefaultClient,
	)
	if _, err := source.Scores(context.Background()); err == nil {
		t.Fatal("Scores() error = nil, want resolver error")
	}
}

func newScoringListServer(t *testing.T, pages map[string]*scoringv1.ListScoresResponse) *httptest.Server {
	t.Helper()
	service := &stubScoreService{t: t, pages: pages}
	path, handler := scoringconnect.NewScoreServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return httptest.NewServer(mux)
}

type stubScoreService struct {
	t     *testing.T
	pages map[string]*scoringv1.ListScoresResponse
}

func (s *stubScoreService) GetScore(context.Context, *connect.Request[scoringv1.GetScoreRequest]) (*connect.Response[scoringv1.GetScoreResponse], error) {
	s.t.Fatal("GetScore should not be called by SCSCompletenessSource")
	return nil, nil
}

func (s *stubScoreService) GetScoreTrend(context.Context, *connect.Request[scoringv1.GetScoreTrendRequest]) (*connect.Response[scoringv1.GetScoreTrendResponse], error) {
	s.t.Fatal("GetScoreTrend should not be called by SCSCompletenessSource")
	return nil, nil
}

func (s *stubScoreService) ListScores(_ context.Context, req *connect.Request[scoringv1.ListScoresRequest]) (*connect.Response[scoringv1.ListScoresResponse], error) {
	if got := req.Msg.GetSortBy(); got != scoringv1.ScoreSortBy_SCORE_SORT_BY_SCENARIO {
		s.t.Fatalf("SortBy = %v, want scenario", got)
	}
	if got := req.Msg.GetOrder(); got != scoringv1.SortOrder_SORT_ORDER_ASC {
		s.t.Fatalf("Order = %v, want asc", got)
	}
	if got := req.Msg.GetPageSize(); got != completenessPageSize {
		s.t.Fatalf("PageSize = %d, want %d", got, completenessPageSize)
	}
	resp, ok := s.pages[req.Msg.GetPageToken()]
	if !ok {
		s.t.Fatalf("unexpected page token %q", req.Msg.GetPageToken())
	}
	return connect.NewResponse(resp), nil
}

type staticURLResolver struct {
	url string
	err error
}

func (r staticURLResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, r.err
}
