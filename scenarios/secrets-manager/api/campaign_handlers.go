package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func configureCampaignRoots() error {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return err
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "secrets-manager"})
	if err != nil {
		return err
	}
	campaignRoots = filerouting.New(paths)
	return nil
}

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
	LastStep   string             `json:"last_step,omitempty"`
	Summary    *DeploymentSummary `json:"summary,omitempty"`
}

type campaignFilePayload struct {
	Campaigns []CampaignSummary `json:"campaigns"`
}

// CampaignStore abstracts persistence so we can move from file to DB cleanly.
type CampaignStore interface {
	List(ctx context.Context, scenarioFilter string) ([]CampaignSummary, error)
	Upsert(ctx context.Context, campaign CampaignSummary) error
}

// CampaignHandlers exposes campaign list endpoints so the UI can render a sortable table
// and stepper without hard-coding scenarios.
type CampaignHandlers struct {
	scenarioCLI     ScenarioCLI
	manifestBuilder ManifestBuilderAPI
	store           CampaignStore
}

// ManifestBuilderAPI captures the Build capability used by campaigns to fetch readiness summaries.
type ManifestBuilderAPI interface {
	Build(ctx context.Context, req DeploymentManifestRequest) (*DeploymentManifest, error)
}

func NewCampaignHandlers(builder ManifestBuilderAPI, store CampaignStore) *CampaignHandlers {
	return &CampaignHandlers{
		scenarioCLI:     defaultScenarioCLI,
		manifestBuilder: builder,
		store:           store,
	}
}

func NewCampaignHandlersWithCLI(cli ScenarioCLI, builder ManifestBuilderAPI, store CampaignStore) *CampaignHandlers {
	return &CampaignHandlers{
		scenarioCLI:     cli,
		manifestBuilder: builder,
		store:           store,
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
	scenarioFilter := strings.TrimSpace(r.URL.Query().Get("scenario"))

	seed := h.seedFromScenarios(ctx)
	var persisted []CampaignSummary
	var loadErr error
	if h.store != nil {
		persisted, loadErr = h.store.List(ctx, scenarioFilter)
		if loadErr != nil {
			http.Error(w, fmt.Sprintf("failed to load campaigns: %v", loadErr), http.StatusInternalServerError)
			return
		}
		// Ensure seed scenarios exist in the store so subsequent reads stay in sync.
		for _, c := range seed {
			_ = h.store.Upsert(ctx, c)
		}
	} else {
		fileCampaigns, err := h.loadCampaignsFromFile(ctx)
		if err == nil && len(fileCampaigns) > 0 {
			persisted = append(persisted, fileCampaigns...)
		}
	}

	// Deduplicate by ID (scenario-tier combo) while preserving order
	seen := make(map[string]struct{})
	combined := append(persisted, seed...)
	deduped := make([]CampaignSummary, 0, len(combined))
	for _, c := range combined {
		if _, ok := seen[c.ID]; ok {
			continue
		}
		seen[c.ID] = struct{}{}
		deduped = append(deduped, c)
	}

	if scenarioFilter != "" {
		filtered := deduped[:0]
		for _, c := range deduped {
			if strings.EqualFold(c.Scenario, scenarioFilter) {
				filtered = append(filtered, c)
			}
		}
		deduped = filtered
	}

	// Keep deterministic order for UI sort defaults
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Scenario < deduped[j].Scenario
	})

	if includeReadiness {
		h.enrichWithReadiness(ctx, deduped, scenarioFilter)
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

func (h *CampaignHandlers) loadCampaignsFromFile(ctx context.Context) ([]CampaignSummary, error) {
	path, err := campaignsFilePath(ctx)
	if err != nil {
		return nil, err
	}
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

	// Ensure Progress/Blockers align with the attached summary if present so UI tables stay consistent.
	if incoming.Summary != nil {
		incoming.Progress = incoming.Summary.StrategizedSecrets
		incoming.Blockers = len(incoming.Summary.BlockingSecrets)
	}

	if h.store != nil {
		if err := h.store.Upsert(r.Context(), incoming); err != nil {
			http.Error(w, fmt.Sprintf("failed to persist campaign: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		existing, _ := h.loadCampaignsFromFile(r.Context())
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

		path, err := campaignsFilePath(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to resolve campaigns path: %v", err), http.StatusInternalServerError)
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			http.Error(w, "failed to ensure data directory", http.StatusInternalServerError)
			return
		}
		if err := storage.WriteFileAtomic(path, data, storage.SecretFilePerm); err != nil {
			http.Error(w, "failed to persist campaign", http.StatusInternalServerError)
			return
		}
		if campaignRoots != nil {
			campaignRoots.RecordWrite(r.Context())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"campaign": incoming,
		"saved":    true,
	})
}

func campaignsFilePath(ctx context.Context) (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "secrets-manager"})
	if err != nil {
		return "", fmt.Errorf("resolve campaigns path: %w", err)
	}
	root := paths.DataDir
	if campaignRoots != nil {
		root, err = campaignRoots.Pick(ctx, storage.ClassData)
		if err != nil {
			return "", fmt.Errorf("select campaigns path: %w", err)
		}
	}
	path := filepath.Join(root, "campaigns.json")
	if campaignRoots != nil && root != paths.DataDir {
		return path, nil
	}
	if err := migrateLegacyCampaigns(path); err != nil {
		return "", err
	}
	return path, nil
}

func migrateLegacyCampaigns(dst string) error {
	legacy := filepath.Join(getVrooliRoot(), "scenarios", "secrets-manager", "data", "campaigns.json")
	if _, err := os.Stat(legacy); err != nil {
		return nil
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("ensure campaigns destination dir: %w", err)
	}
	if err := os.Rename(legacy, dst); err != nil {
		return fmt.Errorf("migrate campaigns file: %w", err)
	}
	return nil
}

// enrichWithReadiness attaches readiness summaries per campaign using the manifest builder.
func (h *CampaignHandlers) enrichWithReadiness(ctx context.Context, campaigns []CampaignSummary, scenarioFilter string) {
	if h.manifestBuilder == nil {
		return
	}
	for i, campaign := range campaigns {
		if scenarioFilter != "" && !strings.EqualFold(campaign.Scenario, scenarioFilter) {
			continue
		}
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
