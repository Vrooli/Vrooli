package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxRequirementSearchLimit = 25

var exactScenarioSlug = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

type requirementsSearchRequest struct {
	Query    string `json:"query"`
	Scenario string `json:"scenario"`
	Limit    int    `json:"limit"`
}

type requirementsSearchResponse struct {
	Query        string                          `json:"query"`
	Scenario     string                          `json:"scenario,omitempty"`
	Scope        string                          `json:"scope,omitempty"`
	Results      []requirementSearchResult       `json:"results"`
	GeneratedAt  time.Time                       `json:"generatedAt,omitempty"`
	Summary      *RequirementsSummary            `json:"summary,omitempty"`
	Degradations []requirementsSearchDegradation `json:"degradations"`
}

type requirementsSearchDegradation struct {
	Reason    string `json:"reason"`
	Leg       string `json:"leg"`
	Retryable bool   `json:"retryable"`
}

type requirementSearchResult struct {
	ResultType  string                      `json:"resultType"`
	ID          string                      `json:"id"`
	Title       string                      `json:"title"`
	Status      string                      `json:"status"`
	LiveStatus  string                      `json:"liveStatus"`
	Criticality string                      `json:"criticality,omitempty"`
	Module      string                      `json:"module"`
	Snippet     string                      `json:"snippet"`
	Score       float64                     `json:"score"`
	Path        string                      `json:"path"`
	Summary     *RequirementsSummary        `json:"summary,omitempty"`
	Source      requirementSearchProvenance `json:"source"`
}

type requirementSearchProvenance struct {
	Owner              string    `json:"owner"`
	Scenario           string    `json:"scenario"`
	Module             string    `json:"module"`
	Path               string    `json:"path"`
	GeneratedAt        time.Time `json:"generatedAt"`
	EvidenceBasis      string    `json:"evidenceBasis"`
	Freshness          string    `json:"freshness"`
	CurrentRunVerified bool      `json:"currentRunVerified"`
	LiveStatusKnown    bool      `json:"liveStatusKnown"`
}

// handleSearchScenarioRequirements searches only the exact scenario named in
// the request. It reuses the requirements view loader used by the GET endpoint;
// lexical scores describe query relevance and never indicate test status.
func (s *Server) handleSearchScenarioRequirements(w http.ResponseWriter, r *http.Request) {
	var req requirementsSearchRequest
	if r.Body != nil {
		defer r.Body.Close()
		decoder := json.NewDecoder(io.LimitReader(r.Body, 32*1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil && err != io.EOF {
			s.writeError(w, http.StatusBadRequest, "invalid search payload")
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			s.writeError(w, http.StatusBadRequest, "search payload must contain one JSON object")
			return
		}
	}

	query := strings.TrimSpace(req.Query)
	scenario := strings.TrimSpace(req.Scenario)
	limit := req.Limit
	if limit == 0 {
		limit = 10
	}
	if limit < 0 {
		s.writeError(w, http.StatusBadRequest, "limit must be positive")
		return
	}
	if limit > maxRequirementSearchLimit {
		limit = maxRequirementSearchLimit
	}

	response := requirementsSearchResponse{
		Query:        query,
		Scenario:     scenario,
		Results:      []requirementSearchResult{},
		Degradations: []requirementsSearchDegradation{},
	}
	if query == "" {
		response.Degradations = append(response.Degradations, requirementsSearchDegradation{Reason: "query_required", Leg: "query"})
		s.writeJSON(w, http.StatusOK, response)
		return
	}
	if scenario == "" {
		response.Degradations = append(response.Degradations, requirementsSearchDegradation{Reason: "scenario_scope_required", Leg: "scope"})
		s.writeJSON(w, http.StatusOK, response)
		return
	}
	if !exactScenarioSlug.MatchString(scenario) {
		response.Degradations = append(response.Degradations, requirementsSearchDegradation{Reason: "scenario_scope_invalid", Leg: "scope"})
		s.writeJSON(w, http.StatusOK, response)
		return
	}
	response.Scope = "scenario:" + scenario

	scenarioDir := s.resolveScenarioDir(scenario)
	if scenarioDir == "" {
		response.Degradations = append(response.Degradations, requirementsSearchDegradation{Reason: "scenario_not_found", Leg: "scope"})
		s.writeJSON(w, http.StatusOK, response)
		return
	}

	// Use exactly the same owner snapshot view as GET /scenarios/{name}/requirements.
	view := s.loadScenarioRequirementsView(scenarioDir, scenario)
	if view == nil {
		response.Degradations = append(response.Degradations, requirementsSearchDegradation{Reason: "requirements_view_unavailable", Leg: "requirements", Retryable: true})
		s.writeJSON(w, http.StatusOK, response)
		return
	}
	response.GeneratedAt = view.GeneratedAt
	response.Summary = &view.Summary

	// The owner loader's compatibility fallback can return an empty view when the
	// requirements index is unavailable. Surface that state instead of making it
	// look like a valid empty requirements set.
	modules, _, _, loadErr := s.loadRequirementModules(scenarioDir)
	if loadErr != nil {
		response.Degradations = append(response.Degradations, requirementsSearchDegradation{Reason: "requirements_source_unavailable", Leg: "requirements", Retryable: true})
		s.writeJSON(w, http.StatusOK, response)
		return
	}
	paths := newRequirementsPaths(scenarioDir)
	index, indexErr := s.loadRequirementsIndex(paths.indexPath)
	if indexErr != nil {
		response.Degradations = append(response.Degradations, requirementsSearchDegradation{Reason: "requirements_source_unavailable", Leg: "requirements", Retryable: true})
		s.writeJSON(w, http.StatusOK, response)
		return
	}
	if len(modules) < len(index.Imports)+boolCount(len(index.Requirements) > 0) {
		response.Degradations = append(response.Degradations, requirementsSearchDegradation{Reason: "requirements_modules_partial", Leg: "requirements", Retryable: true})
	}
	evidenceBasis := "Test Genie owner requirements view; recorded status, not a current test run"
	freshness := "owner view generatedAt; this timestamp is not current test freshness"

	for _, module := range view.Modules {
		for _, requirement := range module.Requirements {
			score := requirementLexicalScore(query, requirement, module.Name)
			if score <= 0 {
				continue
			}
			path := module.FilePath
			response.Results = append(response.Results, requirementSearchResult{
				ResultType:  "requirement",
				ID:          requirement.ID,
				Title:       requirement.Title,
				Status:      requirement.Status,
				LiveStatus:  requirement.LiveStatus,
				Criticality: requirement.Criticality,
				Module:      module.Name,
				Snippet:     requirementSnippet(requirement),
				Score:       score,
				Path:        path,
				Source: requirementSearchProvenance{
					Owner:              "test-genie",
					Scenario:           scenario,
					Module:             module.Name,
					Path:               filepath.ToSlash(path),
					GeneratedAt:        view.GeneratedAt,
					EvidenceBasis:      evidenceBasis,
					Freshness:          freshness,
					CurrentRunVerified: false,
					LiveStatusKnown:    requirement.LiveStatus != "" && requirement.LiveStatus != "unknown",
				},
			})
		}
	}
	if score := requirementSummaryLexicalScore(query, scenario, view.Summary); score > 0 {
		response.Results = append(response.Results, requirementSearchResult{
			ResultType: "summary",
			ID:         "requirements-summary:" + scenario,
			Title:      "Requirements status summary for " + scenario,
			LiveStatus: "unknown",
			Module:     "requirements",
			Snippet:    requirementSummarySnippet(view.Summary),
			Score:      score,
			Path:       "requirements/",
			Summary:    &view.Summary,
			Source: requirementSearchProvenance{
				Owner:              "test-genie",
				Scenario:           scenario,
				Module:             "requirements",
				Path:               "requirements/",
				GeneratedAt:        view.GeneratedAt,
				EvidenceBasis:      evidenceBasis,
				Freshness:          freshness,
				CurrentRunVerified: false,
				LiveStatusKnown:    false,
			},
		})
	}
	sort.SliceStable(response.Results, func(i, j int) bool {
		if response.Results[i].Score != response.Results[j].Score {
			return response.Results[i].Score > response.Results[j].Score
		}
		if response.Results[i].ID != response.Results[j].ID {
			return response.Results[i].ID < response.Results[j].ID
		}
		return response.Results[i].Path < response.Results[j].Path
	})
	if len(response.Results) > limit {
		response.Results = response.Results[:limit]
	}
	s.writeJSON(w, http.StatusOK, response)
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func requirementLexicalScore(query string, requirement RequirementItem, module string) float64 {
	terms := lexicalTerms(query)
	if len(terms) == 0 {
		return 0
	}
	id := strings.ToLower(requirement.ID)
	title := strings.ToLower(requirement.Title)
	text := strings.ToLower(strings.Join([]string{requirement.Description, module}, " "))
	matched := 0
	for term := range terms {
		if strings.Contains(id, term) || strings.Contains(title, term) || strings.Contains(text, term) {
			matched++
		}
	}
	return float64(matched) / float64(len(terms))
}

func requirementSummaryLexicalScore(query, scenario string, summary RequirementsSummary) float64 {
	fields := []string{
		"requirements requirement status validation complete incomplete in progress pending planned passed failed skipped not run unknown critical gap",
		scenario,
		"total requirements " + strings.TrimSpace(stringCount(summary.TotalRequirements)),
		"total validations " + strings.TrimSpace(stringCount(summary.TotalValidations)),
		"critical gap " + strings.TrimSpace(stringCount(summary.CriticalGap)),
	}
	for key, count := range summary.ByDeclaredStatus {
		fields = append(fields, key+" "+stringCount(count))
	}
	for key, count := range summary.ByLiveStatus {
		fields = append(fields, key+" "+stringCount(count))
	}
	req := RequirementItem{Title: "Requirements status summary", Description: strings.Join(fields, " ")}
	return requirementLexicalScore(query, req, "requirements summary")
}

func stringCount(value int) string {
	return strconv.Itoa(value)
}

func requirementSummarySnippet(summary RequirementsSummary) string {
	return fmt.Sprintf("Requirements summary: %d requirements, %d validations, %.1f%% declared completion, %d critical gaps. Live-status counts are owner-reported snapshot values, not current test execution.",
		summary.TotalRequirements, summary.TotalValidations, summary.CompletionRate, summary.CriticalGap)
}

func lexicalTerms(value string) map[string]struct{} {
	terms := make(map[string]struct{})
	for _, field := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if len(field) >= 2 {
			terms[field] = struct{}{}
		}
	}
	return terms
}

func requirementSnippet(requirement RequirementItem) string {
	text := strings.TrimSpace(requirement.Title)
	if description := strings.TrimSpace(requirement.Description); description != "" && !strings.EqualFold(description, text) {
		if text != "" {
			text += " — "
		}
		text += description
	}
	const maxSnippetRunes = 240
	runes := []rune(text)
	if len(runes) > maxSnippetRunes {
		return string(runes[:maxSnippetRunes]) + "…"
	}
	return text
}
