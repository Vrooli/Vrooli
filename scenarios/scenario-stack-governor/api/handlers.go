package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}

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

	// Scale the global timeout by the number of scenarios and rules to avoid
	// starvation when running many scenarios. Each scenario gets up to 2 minutes
	// per rule, capped at a 10-minute global maximum.
	globalTimeout := time.Duration(len(req.ScenarioNames)) * 2 * time.Minute
	if globalTimeout < 2*time.Minute {
		globalTimeout = 2 * time.Minute
	}
	if globalTimeout > 10*time.Minute {
		globalTimeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(r.Context(), globalTimeout)
	defer cancel()

	// If no specific scenarios requested, run against all (pass "").
	scenarioNames := req.ScenarioNames
	if len(scenarioNames) == 0 {
		scenarioNames = []string{""}
	}

	var timedOut bool
	results := []RuleResult{}
	for _, entry := range AllRules() {
		if !cfg.EnabledRules[entry.Definition.ID] {
			continue
		}

		// Check if context has been cancelled (timeout).
		if ctx.Err() != nil {
			timedOut = true
			break
		}

		// Merge results across all requested scenarios into one RuleResult per rule.
		var merged RuleResult
		for i, name := range scenarioNames {
			res := entry.Runner(ctx, repoRoot, strings.TrimSpace(name))
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
		merged.ComputeCounts()
		results = append(results, merged)
	}

	writeJSON(w, http.StatusOK, RunResponse{
		RepoRoot: repoRoot,
		Results:  results,
		TimedOut: timedOut,
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
		if ent.IsDir() && isScenarioDir(ent.Name()) {
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

	// Scale timeout: 2 minutes per scenario, capped at 10 minutes.
	fixTimeout := time.Duration(len(req.ScenarioNames)) * 2 * time.Minute
	if fixTimeout < 2*time.Minute {
		fixTimeout = 2 * time.Minute
	}
	if fixTimeout > 10*time.Minute {
		fixTimeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(r.Context(), fixTimeout)
	defer cancel()

	// Determine which rule IDs are requested (default: all fixable).
	requested := map[string]struct{}{}
	for _, id := range req.RuleIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			requested[id] = struct{}{}
		}
	}

	// Build the set of fixable entries to run.
	allRules := AllRules()
	var entriesToFix []RuleEntry
	for _, entry := range allRules {
		if entry.Fixer == nil {
			continue
		}
		if len(requested) > 0 {
			if _, ok := requested[entry.Definition.ID]; !ok {
				continue
			}
		}
		entriesToFix = append(entriesToFix, entry)
	}

	var results []FixResult
	for _, name := range req.ScenarioNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Track which fixer groups have already been called for this scenario.
		calledGroups := map[string]struct{}{}

		for _, entry := range entriesToFix {
			if entry.FixerGroup != "" {
				if _, done := calledGroups[entry.FixerGroup]; done {
					// Already called this fixer group for this scenario; skip.
					continue
				}
				calledGroups[entry.FixerGroup] = struct{}{}
			}

			fixResults := callFixerSafe(ctx, entry, repoRoot, name, req.DryRun)
			results = append(results, fixResults...)
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

// callFixerSafe calls a fixer with panic recovery so a single fixer panic
// doesn't crash the server. On panic, it returns a FixResult with the error.
func callFixerSafe(ctx context.Context, entry RuleEntry, repoRoot, scenarioName string, dryRun bool) (results []FixResult) {
	defer func() {
		if r := recover(); r != nil {
			results = []FixResult{{
				ScenarioName: scenarioName,
				RuleID:       entry.Definition.ID,
				Fixed:        false,
				Error:        fmt.Sprintf("fixer panicked: %v", r),
			}}
		}
	}()
	return entry.Fixer(ctx, repoRoot, scenarioName, dryRun)
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
