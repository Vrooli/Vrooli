package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gorilla/mux"
)

// CampaignSummary represents a lightweight deployment campaign row used by the UI.
type CampaignSummary struct {
	ID         string             `json:"id"`
	Scenario   string             `json:"scenario"`
	Tier       string             `json:"tier"`
	Status     string             `json:"status"`
	Progress   int                `json:"progress"`
	Blockers   int                `json:"blockers"`
	UpdatedAt  time.Time          `json:"updated_at"`
	NextAction string             `json:"next_action,omitempty"`
	Summary    *DeploymentSummary `json:"summary,omitempty"`
}

type campaignFilePayload struct {
	Campaigns []CampaignSummary `json:"campaigns"`
}

// CampaignHandlers exposes campaign list endpoints so the UI can render a sortable table
// and stepper without hard-coding scenarios.
type CampaignHandlers struct {
	scenarioCLI     ScenarioCLI
	manifestBuilder *ManifestBuilder
}

func NewCampaignHandlers(builder *ManifestBuilder) *CampaignHandlers {
	return &CampaignHandlers{
		scenarioCLI:     defaultScenarioCLI,
		manifestBuilder: builder,
	}
}

func NewCampaignHandlersWithCLI(cli ScenarioCLI, builder *ManifestBuilder) *CampaignHandlers {
	return &CampaignHandlers{
		scenarioCLI:     cli,
		manifestBuilder: builder,
	}
}

func (h *CampaignHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("", h.ListCampaigns).Methods("GET")
	r.HandleFunc("", h.UpsertCampaign).Methods("POST")
}

// ListCampaigns returns saved campaigns if present, otherwise seeds from the scenario list.
func (h *CampaignHandlers) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	includeReadiness := r.URL.Query().Get("include_readiness") == "true"

	seed := h.seedFromScenarios(ctx)
	fileCampaigns, err := h.loadCampaignsFromFile()
	if err == nil && len(fileCampaigns) > 0 {
		seed = append(seed, fileCampaigns...)
	}

	// Deduplicate by ID (scenario-tier combo) while preserving order
	seen := make(map[string]struct{})
	deduped := make([]CampaignSummary, 0, len(seed))
	for _, c := range seed {
		if _, ok := seen[c.ID]; ok {
			continue
		}
		seen[c.ID] = struct{}{}
		deduped = append(deduped, c)
	}

	// Keep deterministic order for UI sort defaults
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Scenario < deduped[j].Scenario
	})

	if includeReadiness {
		h.enrichWithReadiness(ctx, deduped)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"campaigns": deduped,
		"count":     len(deduped),
	})
}

func (h *CampaignHandlers) seedFromScenarios(ctx context.Context) []CampaignSummary {
	scenarios, err := h.scenarioCLI.ListScenarios(ctx)
	if err != nil {
		return nil
	}

	now := time.Now()
	seed := make([]CampaignSummary, 0, len(scenarios))
	for _, scenario := range scenarios {
		id := scenario.Name + "::tier-2-desktop"
		seed = append(seed, CampaignSummary{
			ID:         id,
			Scenario:   scenario.Name,
			Tier:       "tier-2-desktop",
			Status:     "unknown",
			Progress:   0,
			Blockers:   0,
			UpdatedAt:  now,
			NextAction: "Open deployment tab to run readiness",
		})
	}
	return seed
}

func (h *CampaignHandlers) loadCampaignsFromFile() ([]CampaignSummary, error) {
	path := filepath.Join(getVrooliRoot(), "scenarios", "secrets-manager", "data", "campaigns.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload campaignFilePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload.Campaigns, nil
}

// UpsertCampaign writes a campaign record to the data file so UI state persists.
func (h *CampaignHandlers) UpsertCampaign(w http.ResponseWriter, r *http.Request) {
	var incoming CampaignSummary
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if incoming.Scenario == "" || incoming.Tier == "" {
		http.Error(w, "scenario and tier are required", http.StatusBadRequest)
		return
	}
	if incoming.ID == "" {
		incoming.ID = incoming.Scenario + "::" + incoming.Tier
	}
	if incoming.UpdatedAt.IsZero() {
		incoming.UpdatedAt = time.Now()
	}

	existing, _ := h.loadCampaignsFromFile()
	updated := false
	for i, c := range existing {
		if c.ID == incoming.ID {
			existing[i] = incoming
			updated = true
			break
		}
	}
	if !updated {
		existing = append(existing, incoming)
	}

	payload := campaignFilePayload{Campaigns: existing}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		http.Error(w, "failed to encode campaigns", http.StatusInternalServerError)
		return
	}

	path := filepath.Join(getVrooliRoot(), "scenarios", "secrets-manager", "data", "campaigns.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		http.Error(w, "failed to ensure data directory", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		http.Error(w, "failed to persist campaign", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"campaign": incoming,
		"saved":    true,
	})
}

// enrichWithReadiness attaches readiness summaries per campaign using the manifest builder.
func (h *CampaignHandlers) enrichWithReadiness(ctx context.Context, campaigns []CampaignSummary) {
	if h.manifestBuilder == nil {
		return
	}
	for i, campaign := range campaigns {
		req := DeploymentManifestRequest{
			Scenario:        campaign.Scenario,
			Tier:            campaign.Tier,
			IncludeOptional: false,
		}
		manifest, err := h.manifestBuilder.Build(ctx, req)
		if err != nil {
			continue
		}
		campaigns[i].Summary = &manifest.Summary
		campaigns[i].Blockers = len(manifest.Summary.BlockingSecrets)
		campaigns[i].Progress = manifest.Summary.StrategizedSecrets
	}
}
