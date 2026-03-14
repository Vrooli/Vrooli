package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// ScenarioInfo holds metadata about a single scenario.
type ScenarioInfo struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	HealthStatus *string        `json:"health_status"`
	Tags         []string       `json:"tags"`
	Runtime      string         `json:"runtime"`
}

// ScenarioLocator discovers available scenarios via the vrooli CLI.
type ScenarioLocator struct {
	mu        sync.Mutex
	cache     []ScenarioInfo
	cacheTime time.Time
	cacheTTL  time.Duration
}

// NewScenarioLocator creates a locator with the given cache TTL.
func NewScenarioLocator(ttl time.Duration) *ScenarioLocator {
	return &ScenarioLocator{cacheTTL: ttl}
}

// List returns all known scenarios, caching results for cacheTTL.
func (sl *ScenarioLocator) List(ctx context.Context) ([]ScenarioInfo, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if len(sl.cache) > 0 && time.Since(sl.cacheTime) < sl.cacheTTL {
		return sl.cache, nil
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(callCtx, "vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var resp struct {
		Scenarios []json.RawMessage `json:"scenarios"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, err
	}

	scenarios := make([]ScenarioInfo, 0, len(resp.Scenarios))
	for _, raw := range resp.Scenarios {
		var s ScenarioInfo
		if err := json.Unmarshal(raw, &s); err != nil {
			log.Printf("WARNING: skipping unparseable scenario entry: %v", err)
			continue
		}
		if s.Tags == nil {
			s.Tags = []string{}
		}
		scenarios = append(scenarios, s)
	}

	sl.cache = scenarios
	sl.cacheTime = time.Now()
	return scenarios, nil
}

// handleScenarioList returns the list of all scenarios as JSON.
func (s *Server) handleScenarioList(w http.ResponseWriter, r *http.Request) {
	scenarios, err := s.scenarioLocator.List(r.Context())
	if err != nil {
		log.Printf("ERROR: scenario list failed: %v", err)
		http.Error(w, "failed to list scenarios", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scenarios); err != nil {
		log.Printf("ERROR: encoding scenario list: %v", err)
	}
}
