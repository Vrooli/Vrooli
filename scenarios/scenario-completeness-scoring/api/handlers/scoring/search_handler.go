package scoring

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	internalscoring "scenario-completeness-scoring/internal/scoring"

	"google.golang.org/protobuf/encoding/protojson"
)

const maxSearchRequestBytes = 64 << 10

// SearchRequest is the bounded Search Hub adapter request. Scenario is
// required as an explicit scenario:<slug> scope; it is never inferred from
// query text.
type SearchRequest struct {
	Query    string `json:"query"`
	Scenario string `json:"scenario"`
	Limit    int    `json:"limit,omitempty"`
}

// SearchResponse echoes request provenance and returns at most one scenario's
// cached score evidence. A request with no resolvable scope is an explicit,
// degraded empty result rather than a guessed target.
type SearchResponse struct {
	Query        string              `json:"query"`
	Scenario     string              `json:"scenario"`
	Source       string              `json:"source"`
	Basis        string              `json:"basis"`
	Results      []SearchResult      `json:"results"`
	Degradations []SearchDegradation `json:"degradations"`
}

type SearchResult struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	Score        float64             `json:"score"`
	Snippet      string              `json:"snippet"`
	Path         string              `json:"path"`
	Scenario     string              `json:"scenario"`
	Source       string              `json:"source"`
	Basis        string              `json:"basis"`
	CalculatedAt string              `json:"calculatedAt"`
	Digest       string              `json:"digest"`
	Maturity     json.RawMessage     `json:"maturity"`
	Composite    json.RawMessage     `json:"composite"`
	Freshness    json.RawMessage     `json:"freshness"`
	Approximate  bool                `json:"approximate"`
	Degradations []SearchDegradation `json:"degradations"`
}

type SearchDegradation struct {
	Code   string `json:"code"`
	State  string `json:"state,omitempty"`
	Reason string `json:"reason"`
}

func newSearchHandler(scorer Scorer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body := http.MaxBytesReader(w, r.Body, maxSearchRequestBytes)
		defer body.Close()
		var req SearchRequest
		decoder := json.NewDecoder(body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeSearchJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeSearchJSON(w, http.StatusBadRequest, map[string]string{"error": "request must contain one JSON object"})
			return
		}

		req.Scenario = strings.TrimSpace(req.Scenario)
		req.Query = strings.TrimSpace(req.Query)
		out := SearchResponse{
			Query: req.Query, Scenario: req.Scenario,
			Source:  "scenario-completeness-scoring.ScoreService.GetScore",
			Basis:   "cached_artifacts",
			Results: []SearchResult{}, Degradations: []SearchDegradation{},
		}
		if req.Scenario == "" {
			out.Degradations = append(out.Degradations, SearchDegradation{Code: "scenario_scope_required", Reason: "provide one explicit scenario scope (scenario:<slug>)"})
			writeSearchJSON(w, http.StatusOK, out)
			return
		}
		if req.Query == "" {
			out.Degradations = append(out.Degradations, SearchDegradation{Code: "query_required", Reason: "query text is required to rank cached maturity evidence"})
			writeSearchJSON(w, http.StatusOK, out)
			return
		}
		if req.Limit < 0 {
			out.Degradations = append(out.Degradations, SearchDegradation{Code: "invalid_limit", Reason: "limit must be zero or one; this endpoint is scoped to one scenario"})
			writeSearchJSON(w, http.StatusOK, out)
			return
		}
		if req.Limit > 1 {
			out.Degradations = append(out.Degradations, SearchDegradation{Code: "limit_capped", Reason: "this endpoint returns at most one result for the explicit scenario scope"})
		}

		result, err := scorer.GetScore(req.Scenario)
		if err != nil {
			code := "score_unavailable"
			reason := "cached score could not be read"
			if errors.Is(err, internalscoring.ErrUnknownScenario) {
				code = "unknown_scenario"
				reason = "the explicitly scoped scenario does not resolve"
			}
			out.Degradations = append(out.Degradations, SearchDegradation{Code: code, Reason: reason})
			writeSearchJSON(w, http.StatusOK, out)
			return
		}

		wire := resultToProto(result)
		evidence := maturityEvidence(result)
		queryTokens := searchTokens(req.Query)
		if len(queryTokens) == 0 {
			out.Degradations = append(out.Degradations, SearchDegradation{Code: "query_has_no_terms", Reason: "query contains no searchable letter or number terms"})
			writeSearchJSON(w, http.StatusOK, out)
			return
		}
		score := tokenOverlap(queryTokens, searchTokens(evidence))
		if score > 0 {
			maturity, _ := protojson.Marshal(wire.GetMaturity())
			composite, _ := protojson.Marshal(wire.GetComposite())
			freshness, _ := protojson.Marshal(wire.GetFreshness())
			approximate := false
			for _, dimension := range result.Maturity.Dimensions {
				approximate = approximate || dimension.Approximate
			}
			degradations := make([]SearchDegradation, 0, len(result.Degradations))
			for _, degradation := range result.Degradations {
				degradations = append(degradations, SearchDegradation{Code: degradation.Collector, State: degradation.State, Reason: degradation.Reason})
			}
			out.Results = append(out.Results, SearchResult{
				ID:       "scenario-completeness-scoring:" + result.Scenario,
				Title:    result.Scenario + " cached maturity and score",
				Score:    score,
				Snippet:  maturitySnippet(result),
				Path:     "scenarios/" + result.Scenario + "/",
				Scenario: result.Scenario,
				Source:   out.Source, Basis: out.Basis,
				CalculatedAt: result.CalculatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
				Digest:       result.Freshness.Digest,
				Maturity:     maturity, Composite: composite, Freshness: freshness,
				Approximate: approximate, Degradations: degradations,
			})
		}
		writeSearchJSON(w, http.StatusOK, out)
	}
}

func writeSearchJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func maturityEvidence(result internalscoring.Result) string {
	var b strings.Builder
	write := func(value string) { b.WriteString(value); b.WriteByte(' ') }
	write("cached maturity score freshness")
	write(result.Scenario)
	write(result.Category)
	write(result.Maturity.WorkingRung)
	write(result.Maturity.SatisfiedThrough)
	if result.Maturity.LadderClean {
		write("ladder clean")
	}
	for _, d := range result.Maturity.Dimensions {
		write(d.Dimension)
	}
	write(result.Composite.Classification)
	write(result.Composite.ClassificationLabel)
	write(result.Freshness.Digest)
	for _, g := range result.Composite.Groups {
		write(g.ID)
		write(g.Label)
		for _, m := range g.Metrics {
			write(m.ID)
			write(m.Label)
			write(m.Observed)
		}
	}
	for _, p := range result.Freshness.Phases {
		write(p.Phase)
		write(p.Verdict)
		write(p.LastStatus)
	}
	return b.String()
}

func maturitySnippet(result internalscoring.Result) string {
	var b strings.Builder
	b.WriteString("Cached maturity for ")
	b.WriteString(result.Scenario)
	if result.Maturity.WorkingRung != "" {
		b.WriteString(": working rung ")
		b.WriteString(result.Maturity.WorkingRung)
	} else if result.Maturity.LadderClean {
		b.WriteString(": ladder clean")
	} else {
		b.WriteString(": rung unavailable")
	}
	if result.Maturity.SatisfiedThrough != "" {
		b.WriteString("; satisfied through ")
		b.WriteString(result.Maturity.SatisfiedThrough)
	}
	b.WriteString("; composite score ")
	b.WriteString(strconv.Itoa(result.Composite.Score))
	b.WriteString("/100 (")
	b.WriteString(strings.TrimSpace(strings.Join([]string{result.Composite.ClassificationLabel, result.Composite.Classification}, "; ")))
	b.WriteByte(')')
	b.WriteString("; freshness digest ")
	if result.Freshness.Digest == "" {
		b.WriteString("unknown")
	} else {
		b.WriteString(result.Freshness.Digest)
	}
	return b.String()
}

func searchTokens(value string) map[string]struct{} {
	words := make(map[string]struct{})
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			words[strings.ToLower(b.String())] = struct{}{}
			b.Reset()
		}
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return words
}

func tokenOverlap(query, evidence map[string]struct{}) float64 {
	if len(query) == 0 {
		return 0
	}
	matched := 0
	for token := range query {
		if _, ok := evidence[token]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(query))
}
