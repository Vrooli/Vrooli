package handlers

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/discovery"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/gorilla/mux"
)

// discoveryCache caches expensive CLI-based discovery results
type discoveryCache struct {
	resources     []tasks.ResourceInfo
	scenarios     []tasks.ScenarioInfo
	resourcesTime time.Time
	scenariosTime time.Time
	cacheDuration time.Duration
	mu            sync.RWMutex
}

func newDiscoveryCache(ttl time.Duration) *discoveryCache {
	return &discoveryCache{
		cacheDuration: ttl,
	}
}

func (c *discoveryCache) getResources() ([]tasks.ResourceInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Since(c.resourcesTime) > c.cacheDuration {
		return nil, false
	}
	return c.resources, true
}

func (c *discoveryCache) setResources(resources []tasks.ResourceInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resources = resources
	c.resourcesTime = time.Now()
}

func (c *discoveryCache) getScenarios() ([]tasks.ScenarioInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Since(c.scenariosTime) > c.cacheDuration {
		return nil, false
	}
	return c.scenarios, true
}

func (c *discoveryCache) setScenarios(scenarios []tasks.ScenarioInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scenarios = scenarios
	c.scenariosTime = time.Now()
}

// DiscoveryHandlers contains handlers for discovery-related endpoints
type DiscoveryHandlers struct {
	assembler *prompts.Assembler
	cache     *discoveryCache
}

// NewDiscoveryHandlers creates a new discovery handlers instance
func NewDiscoveryHandlers(assembler *prompts.Assembler) *DiscoveryHandlers {
	return &DiscoveryHandlers{
		assembler: assembler,
		cache:     newDiscoveryCache(30 * time.Second), // 30 second cache TTL
	}
}

func shouldRefresh(r *http.Request) bool {
	q := r.URL.Query().Get("refresh")
	return q == "1" || strings.EqualFold(q, "true")
}

func (h *DiscoveryHandlers) loadResources(refresh bool) ([]tasks.ResourceInfo, bool) {
	if !refresh {
		if cached, hit := h.cache.getResources(); hit {
			return cached, true
		}
	}

	resources, err := discovery.DiscoverResources()
	if err != nil {
		log.Printf("Failed to discover resources: %v", err)
		// Return empty array instead of error to prevent UI issues
		resources = []tasks.ResourceInfo{}
	}

	h.cache.setResources(resources)
	return resources, false
}

func (h *DiscoveryHandlers) loadScenarios(refresh bool) ([]tasks.ScenarioInfo, bool) {
	if !refresh {
		if cached, hit := h.cache.getScenarios(); hit {
			return cached, true
		}
	}

	scenarios, err := discovery.DiscoverScenarios()
	if err != nil {
		log.Printf("Failed to discover scenarios: %v", err)
		// Return empty array instead of error to prevent UI issues
		scenarios = []tasks.ScenarioInfo{}
	}

	h.cache.setScenarios(scenarios)
	return scenarios, false
}

// GetResourcesHandler returns discovered resources
func (h *DiscoveryHandlers) GetResourcesHandler(w http.ResponseWriter, r *http.Request) {
	resources, cacheHit := h.loadResources(shouldRefresh(r))
	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}
	writeJSON(w, resources, http.StatusOK)
}

// GetScenariosHandler returns discovered scenarios
func (h *DiscoveryHandlers) GetScenariosHandler(w http.ResponseWriter, r *http.Request) {
	scenarios, cacheHit := h.loadScenarios(shouldRefresh(r))
	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}
	writeJSON(w, scenarios, http.StatusOK)
}

// GetOperationsHandler returns available operations from prompt configuration
func (h *DiscoveryHandlers) GetOperationsHandler(w http.ResponseWriter, r *http.Request) {
	config := h.assembler.GetPromptsConfig()
	writeJSON(w, config.Operations, http.StatusOK)
}

// GetCategoriesHandler returns available categories for the create task form
func (h *DiscoveryHandlers) GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	// Static categories for now - could be made dynamic in the future
	categories := map[string]any{
		"resource_categories": map[string]string{
			"ai-ml":         "AI/ML",
			"communication": "Communication",
			"data":          "Data",
			"security":      "Security",
			"automation":    "Automation",
			"monitoring":    "Monitoring",
			"storage":       "Storage",
			"networking":    "Networking",
			"development":   "Development",
			"productivity":  "Productivity",
			"business":      "Business",
		},
		"scenario_categories": map[string]string{
			"productivity":   "Productivity",
			"ai-tools":       "AI Tools",
			"business":       "Business",
			"personal":       "Personal",
			"automation":     "Automation",
			"entertainment":  "Entertainment",
			"education":      "Education",
			"health-fitness": "Health & Fitness",
			"finance":        "Finance",
			"communication":  "Communication",
		},
	}

	writeJSON(w, categories, http.StatusOK)
}

// GetResourceStatusHandler returns detailed status for a specific resource
func (h *DiscoveryHandlers) GetResourceStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceName := vars["name"]

	if resourceName == "" {
		writeError(w, "Resource name is required", http.StatusBadRequest)
		return
	}

	resources, cacheHit := h.loadResources(shouldRefresh(r))
	w.Header().Set("X-Cache", map[bool]string{true: "HIT", false: "MISS"}[cacheHit])

	for _, resource := range resources {
		if resource.Name == resourceName {
			writeJSON(w, resource, http.StatusOK)
			return
		}
	}

	writeError(w, "Resource not found", http.StatusNotFound)
}

// GetScenarioStatusHandler returns detailed status for a specific scenario
func (h *DiscoveryHandlers) GetScenarioStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		writeError(w, "Scenario name is required", http.StatusBadRequest)
		return
	}

	scenarios, cacheHit := h.loadScenarios(shouldRefresh(r))
	w.Header().Set("X-Cache", map[bool]string{true: "HIT", false: "MISS"}[cacheHit])

	for _, scenario := range scenarios {
		if scenario.Name == scenarioName {
			writeJSON(w, scenario, http.StatusOK)
			return
		}
	}

	writeError(w, "Scenario not found", http.StatusNotFound)
}
