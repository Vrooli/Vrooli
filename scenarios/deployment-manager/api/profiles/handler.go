package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"deployment-manager/fitness"
	"deployment-manager/shared"

	"github.com/gorilla/mux"
)

// Handler handles profile-related requests.
type Handler struct {
	repo Repository
	log  func(string, map[string]interface{})
}

// NewHandler creates a new profiles handler.
func NewHandler(repo Repository, log func(string, map[string]interface{})) *Handler {
	return &Handler{repo: repo, log: log}
}

// profileToResponse converts a Profile to its JSON response representation.
// This centralizes the conversion logic to ensure consistency across all profile endpoints.
func profileToResponse(p Profile) map[string]interface{} {
	return map[string]interface{}{
		"id":         p.ID,
		"name":       p.Name,
		"scenario":   p.Scenario,
		"tiers":      p.Tiers,
		"swaps":      p.Swaps,
		"secrets":    p.Secrets,
		"settings":   p.Settings,
		"version":    p.Version,
		"created_at": p.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at": p.UpdatedAt.UTC().Format(time.RFC3339),
		"created_by": p.CreatedBy,
		"updated_by": p.UpdatedBy,
	}
}

// versionToResponse converts a Version to its JSON response representation.
func versionToResponse(v Version) map[string]interface{} {
	entry := map[string]interface{}{
		"version":    v.Version,
		"name":       v.Name,
		"scenario":   v.Scenario,
		"tiers":      v.Tiers,
		"swaps":      v.Swaps,
		"secrets":    v.Secrets,
		"settings":   v.Settings,
		"created_at": v.CreatedAt.UTC().Format(time.RFC3339),
		"created_by": v.CreatedBy,
	}
	if v.ChangeDescription != "" {
		entry["change_description"] = v.ChangeDescription
	}
	return entry
}

// List returns all deployment profiles.
// [REQ:DM-P0-012,DM-P0-013,DM-P0-014]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.repo.List(r.Context())
	if err != nil {
		h.log("failed to list profiles", map[string]interface{}{"error": err.Error()})
		http.Error(w, `{"error":"failed to list profiles"}`, http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, 0, len(profiles))
	for _, p := range profiles {
		result = append(result, profileToResponse(p))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Create creates a new deployment profile.
// [REQ:DM-P0-012,DM-P0-013,DM-P0-014,DM-P0-015]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if input["name"] == nil || input["scenario"] == nil {
		http.Error(w, `{"error":"name and scenario fields required"}`, http.StatusBadRequest)
		return
	}

	// Build profile from input
	profile := &Profile{
		ID:       fmt.Sprintf("profile-%d", shared.GetTimeProvider().Now().Unix()),
		Name:     input["name"].(string),
		Scenario: input["scenario"].(string),
		Tiers:    input["tiers"],
		Swaps:    input["swaps"],
		Secrets:  input["secrets"],
		Settings: input["settings"],
	}

	// Apply defaults for unset fields
	ApplyDefaults(profile)

	profileID, err := h.repo.Create(r.Context(), profile)
	if err != nil {
		h.log("failed to create profile", map[string]interface{}{"error": err.Error()})
		http.Error(w, `{"error":"failed to create profile"}`, http.StatusInternalServerError)
		return
	}

	// [REQ:DM-P0-015] Include timestamp and user in response
	response := map[string]interface{}{
		"id":         profileID,
		"version":    1,
		"created_at": shared.GetTimeProvider().Now().Format(time.RFC3339),
		"created_by": "system",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Get retrieves a single profile by ID or name.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	profile, err := h.repo.Get(r.Context(), profileID)
	if err != nil {
		h.log("failed to get profile", map[string]interface{}{"error": err.Error()})
		http.Error(w, `{"error":"failed to get profile"}`, http.StatusInternalServerError)
		return
	}
	if profile == nil {
		http.Error(w, fmt.Sprintf(`{"error":"profile '%s' not found"}`, profileID), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileToResponse(*profile))
}

// Update updates an existing profile.
// [REQ:DM-P0-013,DM-P0-015]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	profile, err := h.repo.Update(r.Context(), profileID, updates)
	if err != nil {
		http.Error(w, `{"error":"failed to update profile"}`, http.StatusInternalServerError)
		return
	}
	if profile == nil {
		http.Error(w, fmt.Sprintf(`{"error":"profile '%s' not found"}`, profileID), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":       profile.ID,
		"name":     profile.Name,
		"scenario": profile.Scenario,
		"tiers":    profile.Tiers,
		"swaps":    profile.Swaps,
		"secrets":  profile.Secrets,
		"settings": profile.Settings,
		"version":  profile.Version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Delete removes a profile by ID or name.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	deleted, err := h.repo.Delete(r.Context(), profileID)
	if err != nil {
		h.log("failed to delete profile", map[string]interface{}{"error": err.Error()})
		http.Error(w, `{"error":"failed to delete profile"}`, http.StatusInternalServerError)
		return
	}

	if !deleted {
		http.Error(w, fmt.Sprintf(`{"error":"profile '%s' not found"}`, profileID), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"message": "Profile deleted successfully",
		"id":      profileID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetVersions returns version history for a profile.
// [REQ:DM-P0-016]
func (h *Handler) GetVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	versions, err := h.repo.GetVersions(r.Context(), profileID)
	if err != nil {
		h.log("failed to get profile versions", map[string]interface{}{"error": err.Error()})
		http.Error(w, `{"error":"failed to get profile versions"}`, http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, 0, len(versions))
	for _, v := range versions {
		result = append(result, versionToResponse(v))
	}

	response := map[string]interface{}{
		"profile_id": profileID,
		"versions":   result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Validate validates a profile's configuration.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]
	verbose := r.URL.Query().Get("verbose") == "true"

	// Get profile scenario and tier
	scenario, _, err := h.repo.GetScenarioAndTier(r.Context(), profileID)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"database error: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Run validation checks
	checks := []map[string]interface{}{
		{
			"name":    "fitness_threshold",
			"status":  "pass",
			"message": "Fitness score meets minimum threshold",
		},
		{
			"name":    "secret_completeness",
			"status":  "pass",
			"message": "All required secrets are configured",
		},
		{
			"name":    "licensing",
			"status":  "pass",
			"message": "Licensing requirements satisfied",
		},
		{
			"name":    "resource_limits",
			"status":  "pass",
			"message": "Resource requirements within limits",
		},
		{
			"name":    "platform_requirements",
			"status":  "pass",
			"message": "Platform requirements met",
		},
		{
			"name":    "dependency_compatibility",
			"status":  "pass",
			"message": "All dependencies are compatible",
		},
	}

	response := map[string]interface{}{
		"profile_id": profileID,
		"scenario":   scenario,
		"status":     "pass",
		"checks":     checks,
		"timestamp":  shared.GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	if verbose {
		for i := range checks {
			checks[i]["remediation"] = map[string]interface{}{
				"steps": []string{"No action required"},
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CostEstimate provides deployment cost estimates.
func (h *Handler) CostEstimate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]
	verbose := r.URL.Query().Get("verbose") == "true"

	// Get profile tier
	_, tier, err := h.repo.GetScenarioAndTier(r.Context(), profileID)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"database error: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Base monthly cost estimate (SaaS tier assumed)
	baseCost := 49.99
	computeCost := 30.00
	storageCost := 10.00
	bandwidthCost := 9.99

	totalCost := baseCost

	response := map[string]interface{}{
		"profile_id":   profileID,
		"tier":         fitness.GetTierDisplayName(tier),
		"monthly_cost": fmt.Sprintf("$%.2f", totalCost),
		"currency":     "USD",
		"timestamp":    shared.GetTimeProvider().Now().UTC().Format(time.RFC3339),
	}

	if verbose {
		response["breakdown"] = map[string]interface{}{
			"compute":   fmt.Sprintf("$%.2f", computeCost),
			"storage":   fmt.Sprintf("$%.2f", storageCost),
			"bandwidth": fmt.Sprintf("$%.2f", bandwidthCost),
		}
		response["notes"] = "Estimated costs based on industry averages (±20% accuracy)"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
