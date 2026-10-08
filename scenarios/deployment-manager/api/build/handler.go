package build

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deployment-manager/bundles"
	"deployment-manager/profiles"

	repocontract "github.com/vrooli/repo-contract-go"
)

// BuildRequest is the request body for the build endpoint.
type BuildRequest struct {
	// ProfileID is the profile to build from (required)
	ProfileID string `json:"profile_id"`
	// Scenario is the scenario name (optional if profile_id is provided)
	Scenario string `json:"scenario,omitempty"`
	// Platforms to build for (optional, defaults to all)
	Platforms []string `json:"platforms,omitempty"`
	// ServiceIDs to build (optional, defaults to all services with build config)
	ServiceIDs []string `json:"service_ids,omitempty"`
	// DryRun if true, validates but doesn't build
	DryRun bool `json:"dry_run,omitempty"`
}

// BuildResponse is the response from the build endpoint.
type BuildResponse struct {
	Status   string           `json:"status"`
	Scenario string           `json:"scenario"`
	Results  []BuildAllResult `json:"results"`
	Duration string           `json:"duration,omitempty"`
	Message  string           `json:"message,omitempty"`
}

// Handler handles build requests.
type Handler struct {
	profileRepo profiles.Repository
	vrooli      string // VROOLI_ROOT path
	log         func(string, map[string]interface{})
	autoBuilds  *AutoBuildStore
}

// NewHandler creates a new build handler.
func NewHandler(profileRepo profiles.Repository, log func(string, map[string]interface{})) *Handler {
	vrooli := resolveRepoRoot()
	return &Handler{
		profileRepo: profileRepo,
		vrooli:      vrooli,
		log:         log,
		autoBuilds:  NewAutoBuildStore(),
	}
}

// Build handles POST /api/v1/build requests.
func (h *Handler) Build(w http.ResponseWriter, r *http.Request) {
	var req BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.ProfileID == "" && req.Scenario == "" {
		http.Error(w, `{"error":"profile_id or scenario is required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute) // Long timeout for builds
	defer cancel()

	start := time.Now()

	// Get scenario from profile if not provided
	scenario := req.Scenario
	if req.ProfileID != "" && scenario == "" {
		profile, err := h.profileRepo.Get(ctx, req.ProfileID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load profile: %v"}`, err), http.StatusBadGateway)
			return
		}
		if profile == nil {
			http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
			return
		}
		scenario = profile.Scenario
	}

	// Fetch bundle manifest to get build configs
	manifest, err := bundles.FetchSkeletonBundle(ctx, scenario)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to fetch bundle manifest: %v"}`, err), http.StatusBadGateway)
		return
	}

	// Find services with build configs
	servicesToBuild := h.filterBuildableServices(manifest.Services, req.ServiceIDs)
	if len(servicesToBuild) == 0 {
		h.writeJSON(w, http.StatusOK, BuildResponse{
			Status:   "skipped",
			Scenario: scenario,
			Results:  []BuildAllResult{},
			Message:  "No services with build configuration found",
		})
		return
	}

	// Dry run - just report what would be built
	if req.DryRun {
		results := make([]BuildAllResult, 0, len(servicesToBuild))
		for _, svc := range servicesToBuild {
			platforms := req.Platforms
			if len(platforms) == 0 {
				for _, p := range SupportedPlatforms {
					platforms = append(platforms, p.Name)
				}
			}
			results = append(results, BuildAllResult{
				ServiceID:    svc.ID,
				Results:      make([]BuildResult, len(platforms)),
				AllSucceeded: false,
			})
		}
		h.writeJSON(w, http.StatusOK, BuildResponse{
			Status:   "dry_run",
			Scenario: scenario,
			Results:  results,
			Message:  fmt.Sprintf("Would build %d service(s) for %d platform(s)", len(servicesToBuild), len(req.Platforms)),
		})
		return
	}

	// Determine scenario directory
	scenarioDir := resolveScenarioDir(h.vrooli, scenario)

	// Build each service
	builder := NewBuilder(scenarioDir, h.log)
	allResults := make([]BuildAllResult, 0, len(servicesToBuild))
	allSucceeded := true

	for _, svc := range servicesToBuild {
		result, err := builder.BuildAll(ctx, svc.ID, svc.Build, req.Platforms)
		if err != nil {
			h.log("error", map[string]interface{}{
				"msg":     "build failed",
				"service": svc.ID,
				"error":   err.Error(),
			})
			allSucceeded = false
			continue
		}
		allResults = append(allResults, *result)
		if !result.AllSucceeded {
			allSucceeded = false
		}
	}

	status := "success"
	if !allSucceeded {
		status = "partial"
	}
	if len(allResults) == 0 {
		status = "failed"
	}

	h.writeJSON(w, http.StatusOK, BuildResponse{
		Status:   status,
		Scenario: scenario,
		Results:  allResults,
		Duration: time.Since(start).String(),
	})
}

// BuildStatus handles GET /api/v1/build/{build_id} requests.
func (h *Handler) BuildStatus(w http.ResponseWriter, r *http.Request) {
	// Build status tracking is not implemented yet
	// For now, builds are synchronous
	http.Error(w, `{"error":"build status tracking not implemented - builds are synchronous"}`, http.StatusNotImplemented)
}

// filterBuildableServices returns services that have build configs.
func (h *Handler) filterBuildableServices(services []bundles.ServiceEntry, serviceIDs []string) []bundles.ServiceEntry {
	var result []bundles.ServiceEntry

	// If specific service IDs requested, filter to those
	idSet := make(map[string]bool)
	if len(serviceIDs) > 0 {
		for _, id := range serviceIDs {
			idSet[id] = true
		}
	}

	for _, svc := range services {
		// Skip if no build config
		if svc.Build == nil {
			continue
		}

		// Skip if not in requested list (when list is specified)
		if len(serviceIDs) > 0 && !idSet[svc.ID] {
			continue
		}

		result = append(result, svc)
	}

	return result
}

// writeJSON writes a JSON response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func resolveRepoRoot() string {
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(key)); root != "" {
			if resolved, ok := canonicalRepoRootFromOverride(root); ok {
				return resolved
			}
			return filepath.Clean(root)
		}
	}
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		return root
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Clean(cwd)
	}
	return "."
}

func resolveScenarioDir(repoRoot, scenario string) string {
	if resolved, err := repocontract.ResolveScenarioPath(repoRoot, strings.TrimSpace(scenario)); err == nil {
		return resolved
	}
	return filepath.Join(repoRoot, "scenarios", strings.TrimSpace(scenario))
}

func canonicalRepoRootFromOverride(path string) (string, bool) {
	current := filepath.Clean(strings.TrimSpace(path))
	if current == "" || current == "." {
		return "", false
	}
	for depth := 0; depth < 25; depth++ {
		if resolved, err := repocontract.FindRepoRoot(current); err == nil {
			return resolved, true
		}
		contractPath := filepath.Join(current, ".vrooli", "repo-contract.json")
		if info, err := os.Stat(contractPath); err == nil && !info.IsDir() {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", false
}
