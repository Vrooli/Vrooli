package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ecosystem-manager/api/pkg/discovery"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// DiscoveryHandlers contains handlers for discovery-related endpoints
type DiscoveryHandlers struct {
	assembler *prompts.Assembler
}

// NewDiscoveryHandlers creates a new discovery handlers instance
func NewDiscoveryHandlers(assembler *prompts.Assembler) *DiscoveryHandlers {
	return &DiscoveryHandlers{
		assembler: assembler,
	}
}

// GetResourcesHandler returns discovered resources
func (h *DiscoveryHandlers) GetResourcesHandler(w http.ResponseWriter, r *http.Request) {
	resources, err := discovery.DiscoverResources()
	if err != nil {
		log.Printf("Failed to discover resources: %v", err)
		// Return empty array instead of error to prevent UI issues
		resources = []tasks.ResourceInfo{}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// GetScenariosHandler returns discovered scenarios
func (h *DiscoveryHandlers) GetScenariosHandler(w http.ResponseWriter, r *http.Request) {
	scenarios, err := discovery.DiscoverScenarios()
	if err != nil {
		log.Printf("Failed to discover scenarios: %v", err)
		// Return empty array instead of error to prevent UI issues
		scenarios = []tasks.ScenarioInfo{}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

// GetOperationsHandler returns available operations from prompt configuration
func (h *DiscoveryHandlers) GetOperationsHandler(w http.ResponseWriter, r *http.Request) {
	config := h.assembler.GetPromptsConfig()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config.Operations)
}

// GetCategoriesHandler returns available categories for the create task form
func (h *DiscoveryHandlers) GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	// Static categories for now - could be made dynamic in the future
	categories := map[string]interface{}{
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
			"productivity":    "Productivity",
			"ai-tools":        "AI Tools",
			"business":        "Business",
			"personal":        "Personal",
			"automation":      "Automation",
			"entertainment":   "Entertainment",
			"education":       "Education",
			"health-fitness":  "Health & Fitness",
			"finance":         "Finance",
			"communication":   "Communication",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// GetResourceStatusHandler returns detailed status for a specific resource
func (h *DiscoveryHandlers) GetResourceStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	vars := mux.Vars(r)
	resourceName := vars["name"]
	
	if resourceName == "" {
		http.Error(w, "Resource name is required", http.StatusBadRequest)
		return
	}
	
	// Find the resource
	resources, err := discovery.DiscoverResources()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to discover resources: %v", err), http.StatusInternalServerError)
		return
	}
	
	for _, resource := range resources {
		if resource.Name == resourceName {
			if err := json.NewEncoder(w).Encode(resource); err != nil {
				log.Printf("Error encoding resource status response: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}
	
	http.Error(w, "Resource not found", http.StatusNotFound)
}

// GetScenarioStatusHandler returns detailed status for a specific scenario
func (h *DiscoveryHandlers) GetScenarioStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	
	if scenarioName == "" {
		http.Error(w, "Scenario name is required", http.StatusBadRequest)
		return
	}
	
	// Find the scenario
	scenarios, err := discovery.DiscoverScenarios()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to discover scenarios: %v", err), http.StatusInternalServerError)
		return
	}
	
	for _, scenario := range scenarios {
		if scenario.Name == scenarioName {
			if err := json.NewEncoder(w).Encode(scenario); err != nil {
				log.Printf("Error encoding scenario status response: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}
	
	http.Error(w, "Scenario not found", http.StatusNotFound)
}