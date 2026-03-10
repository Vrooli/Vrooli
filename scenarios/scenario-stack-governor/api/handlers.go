package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	// If no specific scenarios requested, run against all (pass "").
	scenarioNames := req.ScenarioNames
	if len(scenarioNames) == 0 {
		scenarioNames = []string{""}
	}

	type ruleRunner func(ctx context.Context, repoRoot, scenarioName string) RuleResult

	runners := map[string]ruleRunner{
		"GO_CLI_WORKSPACE_INDEPENDENCE":       RunGoCliWorkspaceIndependence,
		"REACT_VITE_UI_INSTALLS_DEPENDENCIES": RunReactViteUIInstallsDependencies,
		"MAKEFILE_STRUCTURE":                  RunMakefileStructure,
		"MAKEFILE_LIFECYCLE":                  RunMakefileLifecycle,
		"MAKEFILE_QUALITY":                    RunMakefileQuality,
	}

	results := []RuleResult{}
	for _, rule := range AllRuleDefinitions() {
		if !cfg.EnabledRules[rule.ID] {
			continue
		}
		runner, ok := runners[rule.ID]
		if !ok {
			continue
		}

		// Merge results across all requested scenarios into one RuleResult per rule.
		var merged RuleResult
		for i, name := range scenarioNames {
			res := runner(ctx, repoRoot, strings.TrimSpace(name))
			if i == 0 {
				merged = res
			} else {
				merged.Findings = append(merged.Findings, res.Findings...)
				merged.FinishedAt = res.FinishedAt
				if !res.Passed {
					merged.Passed = false
				}
			}
		}
		results = append(results, merged)
	}

	writeJSON(w, http.StatusOK, RunResponse{
		RepoRoot: repoRoot,
		Results:  results,
	})
}

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	repoRoot, err := FindRepoRoot(s.scenarioRoot)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	scenarios := []string{}
	for _, ent := range entries {
		if ent.IsDir() {
			scenarios = append(scenarios, ent.Name())
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"scenarios": scenarios,
	})
}

func (s *Server) handleFix(w http.ResponseWriter, r *http.Request) {
	var req FixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if len(req.ScenarioNames) == 0 {
		writeError(w, http.StatusBadRequest, "scenario_names is required")
		return
	}

	repoRoot, err := FindRepoRoot(s.scenarioRoot)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	// Determine which rule IDs are requested (default: all fixable).
	requested := map[string]struct{}{}
	for _, id := range req.RuleIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			requested[id] = struct{}{}
		}
	}

	// Check which fixable rule categories are requested.
	makefileRuleIDs := []string{"MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"}
	wantMakefile := len(requested) == 0 // default: all fixable
	wantGoCli := len(requested) == 0
	wantReactVite := len(requested) == 0
	if !wantMakefile || !wantGoCli || !wantReactVite {
		for _, id := range makefileRuleIDs {
			if _, ok := requested[id]; ok {
				wantMakefile = true
				break
			}
		}
		if _, ok := requested["GO_CLI_WORKSPACE_INDEPENDENCE"]; ok {
			wantGoCli = true
		}
		if _, ok := requested["REACT_VITE_UI_INSTALLS_DEPENDENCIES"]; ok {
			wantReactVite = true
		}
	}

	var results []FixResult
	for _, name := range req.ScenarioNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if wantMakefile {
			results = append(results, FixMakefileAll(ctx, repoRoot, name, req.DryRun)...)
		}
		if wantGoCli {
			results = append(results, FixGoCliWorkspaceIndependence(ctx, repoRoot, name, req.DryRun)...)
		}
		if wantReactVite {
			results = append(results, FixReactViteUIInstallsDependencies(ctx, repoRoot, name, req.DryRun)...)
		}
	}

	// Filter results to only requested rule IDs if a filter was set.
	if len(requested) > 0 {
		filtered := make([]FixResult, 0, len(results))
		for _, r := range results {
			if _, ok := requested[r.RuleID]; ok {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	writeJSON(w, http.StatusOK, FixResponse{
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
