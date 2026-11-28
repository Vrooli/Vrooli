package scoring_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scenario-completeness-scoring/handlers/scoring"
	"scenario-completeness-scoring/internal/freshness"
	internalscoring "scenario-completeness-scoring/internal/scoring"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type searchScorer struct {
	result internalscoring.Result
	err    error
	calls  []string
}

func (s *searchScorer) GetScore(scenario string) (internalscoring.Result, error) {
	s.calls = append(s.calls, scenario)
	return s.result, s.err
}

func representativeResult() internalscoring.Result {
	return internalscoring.Result{
		Scenario: "sample-scenario",
		Category: "utility",
		Maturity: internalscoring.Maturity{
			WorkingRung:      "R2 Safe",
			SatisfiedThrough: "R1 Ready",
			Dimensions:       []internalscoring.DimensionCount{{Dimension: "tests", ErrorPlus: 1, Total: 3, Approximate: true}},
		},
		Composite: internalscoring.Composite{
			Score: 88, Classification: "strong", ClassificationLabel: "Strong",
			Groups: []internalscoring.Group{{ID: "quality", Label: "Quality"}},
		},
		Freshness:    freshness.Result{Digest: "td:abc123", Phases: []freshness.PhaseStatus{{Phase: "unit", Verdict: "fresh", LastStatus: "passed"}}},
		CalculatedAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC),
	}
}

func TestSearchHandlerRanksOnlyByQueryEvidenceOverlap(t *testing.T) {
	scorer := &searchScorer{result: representativeResult()}
	rw := callSearch(t, scorer, `{"query":"cached safe","scenario":"sample-scenario","limit":1}`)

	require.Equal(t, http.StatusOK, rw.Code)
	var body struct {
		Query    string `json:"query"`
		Scenario string `json:"scenario"`
		Results  []struct {
			ID           string          `json:"id"`
			Score        float64         `json:"score"`
			Snippet      string          `json:"snippet"`
			Source       string          `json:"source"`
			Basis        string          `json:"basis"`
			CalculatedAt string          `json:"calculatedAt"`
			Digest       string          `json:"digest"`
			Approximate  bool            `json:"approximate"`
			Composite    json.RawMessage `json:"composite"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &body))
	require.Equal(t, "cached safe", body.Query)
	require.Equal(t, "sample-scenario", body.Scenario)
	require.Len(t, body.Results, 1)
	require.Equal(t, "scenario-completeness-scoring:sample-scenario", body.Results[0].ID)
	require.Equal(t, 1.0, body.Results[0].Score, "both query terms occur in maturity evidence")
	require.Contains(t, body.Results[0].Snippet, "composite score 88/100 (Strong; strong)")
	require.Equal(t, "scenario-completeness-scoring.ScoreService.GetScore", body.Results[0].Source)
	require.Equal(t, "cached_artifacts", body.Results[0].Basis)
	require.Equal(t, "2026-11-23T12:00:00Z", body.Results[0].CalculatedAt)
	require.Equal(t, "td:abc123", body.Results[0].Digest)
	require.True(t, body.Results[0].Approximate)
	var composite map[string]any
	require.NoError(t, json.Unmarshal(body.Results[0].Composite, &composite))
	require.Equal(t, float64(88), composite["score"], "composite remains payload evidence, not search relevance")
	require.Equal(t, []string{"sample-scenario"}, scorer.calls)
}

func TestSearchHandlerEmptyAndUnknownScopesDegradeWithoutGuessing(t *testing.T) {
	t.Run("empty scope", func(t *testing.T) {
		scorer := &searchScorer{result: representativeResult()}
		w := callSearch(t, scorer, `{"query":"maturity"}`)
		require.Equal(t, http.StatusOK, w.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Empty(t, body["results"])
		require.NotEmpty(t, body["degradations"])
		require.Empty(t, scorer.calls)
	})

	t.Run("unknown explicit scenario", func(t *testing.T) {
		scorer := &searchScorer{err: internalscoring.ErrUnknownScenario}
		w := callSearch(t, scorer, `{"query":"maturity","scenario":"does-not-exist"}`)
		require.Equal(t, http.StatusOK, w.Code)
		var body struct {
			Results      []any `json:"results"`
			Degradations []struct {
				Code string `json:"code"`
			} `json:"degradations"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Empty(t, body.Results)
		require.Equal(t, "unknown_scenario", body.Degradations[0].Code)
		require.Equal(t, []string{"does-not-exist"}, scorer.calls)
	})
}

func TestSearchHandlerNoOverlapReturnsNoHitAndPreservesScoreSeparation(t *testing.T) {
	scorer := &searchScorer{result: representativeResult()}
	w := callSearch(t, scorer, `{"query":"database migration","scenario":"sample-scenario"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Results []any `json:"results"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Empty(t, body.Results)
}

func TestSearchHandlerUnknownFieldsRejected(t *testing.T) {
	scorer := &searchScorer{}
	w := callSearch(t, scorer, `{"query":"maturity","scenario":"sample-scenario","scope":"all"}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Empty(t, scorer.calls)
}

func TestModuleMountsSearchEndpoint(t *testing.T) {
	scorer := &searchScorer{result: representativeResult()}
	m := scoring.Module(scorer, nil, nil)
	r := mux.NewRouter()
	m.Mount(r)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/scores", strings.NewReader(`{"query":"maturity","scenario":"sample-scenario"}`))
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Len(t, scorer.calls, 1)
}

func callSearch(t *testing.T, scorer *searchScorer, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := mux.NewRouter()
	scoring.Module(scorer, nil, nil).Mount(r)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/scores", strings.NewReader(body))
	r.ServeHTTP(rw, req)
	return rw
}

var _ scoring.Scorer = (*searchScorer)(nil)
