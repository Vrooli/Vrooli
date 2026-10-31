package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// cliClient is the shared typed Vrooli CLI client for scenario discovery. The
// var is package-level so tests can swap in a stubbed Runner.
var cliClient = vroolicli.New()

// ScenarioInfo holds metadata about a single scenario.
//
// HealthStatus is retained for API compatibility but is always nil: the
// `vrooli scenario status --json` contract (vrooli.cli.v1.ScenarioStatusItem)
// carries no health field, so there is nothing to populate it from.
type ScenarioInfo struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	HealthStatus *string  `json:"health_status"`
	Tags         []string `json:"tags"`
	Runtime      string   `json:"runtime"`
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

// List returns all known scenarios, caching results for cacheTTL. It reads the
// typed `vrooli scenario status --json` contract through the CLI client; a CLI
// or decode failure is propagated (never degraded to an empty list, which would
// silently report zero scenarios on a transient hiccup).
func (sl *ScenarioLocator) List(ctx context.Context) ([]ScenarioInfo, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if len(sl.cache) > 0 && time.Since(sl.cacheTime) < sl.cacheTTL {
		return sl.cache, nil
	}

	resp, err := cliClient.ScenarioStatuses(ctx)
	if err != nil {
		return nil, err
	}

	items := resp.GetScenarios()
	scenarios := make([]ScenarioInfo, 0, len(items))
	for _, item := range items {
		if item.GetName() == "" {
			continue
		}
		tags := item.GetTags()
		if tags == nil {
			tags = []string{}
		}
		scenarios = append(scenarios, ScenarioInfo{
			Name:        item.GetName(),
			DisplayName: item.GetDisplayName(),
			Description: item.GetDescription(),
			Status:      item.GetStatus(),
			Tags:        tags,
			Runtime:     item.GetRuntime(),
		})
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
