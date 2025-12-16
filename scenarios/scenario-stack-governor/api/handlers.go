package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.configStore.Load(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type ruleWithState struct {
		RuleDefinition
		Enabled bool `json:"enabled"`
	}

	out := make([]ruleWithState, 0, len(AllRuleDefinitions()))
	for _, rule := range AllRuleDefinitions() {
		out = append(out, ruleWithState{
			RuleDefinition: rule,
			Enabled:        cfg.EnabledRules[rule.ID],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rules":  out,
		"config": cfg,
	})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.configStore.Load(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg RulesConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.configStore.Save(r.Context(), cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out, err := s.configStore.Load(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRunRules(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // optional

	cfg, err := s.configStore.Load(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	repoRoot, err := FindRepoRoot(s.scenarioRoot)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	results := []RuleResult{}
	for _, rule := range AllRuleDefinitions() {
		if !cfg.EnabledRules[rule.ID] {
			continue
		}
		switch rule.ID {
		case "GO_CLI_WORKSPACE_INDEPENDENCE":
			results = append(results, RunGoCliWorkspaceIndependence(ctx, repoRoot, req.ScenarioName))
		case "REACT_VITE_UI_INSTALLS_DEPENDENCIES":
			results = append(results, RunReactViteUIInstallsDependencies(ctx, repoRoot, req.ScenarioName))
		default:
			// unknown rule IDs are removed during config normalization
		}
	}

	writeJSON(w, http.StatusOK, RunResponse{
		RepoRoot: repoRoot,
		Results:  results,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": message,
	})
}
