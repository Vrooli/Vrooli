package bundles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"deployment-manager/codesigning"
	"deployment-manager/codesigning/generation"
	"deployment-manager/profiles"
	"deployment-manager/secrets"
)

// MergeSecretsRequest is the request body for merge-secrets endpoint.
type MergeSecretsRequest struct {
	Scenario string                 `json:"scenario"`
	Tier     string                 `json:"tier"`
	Manifest Manifest               `json:"manifest"`
	Raw      map[string]interface{} `json:"-"`
}

// AssembleBundleRequest is the request body for assemble endpoint.
type AssembleBundleRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier"`
	ProfileID      string `json:"profile_id,omitempty"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
}

// ExportBundleRequest is the request body for bundle export endpoint.
type ExportBundleRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier"`
	ProfileID      string `json:"profile_id,omitempty"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
}

// ExportBundleResponse is the response for bundle export endpoint.
type ExportBundleResponse struct {
	Status      string   `json:"status"`
	Schema      string   `json:"schema"`
	Scenario    string   `json:"scenario"`
	Tier        string   `json:"tier"`
	Manifest    Manifest `json:"manifest"`
	Checksum    string   `json:"checksum"`
	GeneratedAt string   `json:"generated_at"`
}

// Handler handles bundle-related requests.
type Handler struct {
	secretsClient *secrets.Client
	profileRepo   profiles.Repository
	signingRepo   codesigning.Repository
	signingGen    generation.Generator
	log           func(string, map[string]interface{})
}

// NewHandler creates a new bundles handler.
func NewHandler(secretsClient *secrets.Client, profileRepo profiles.Repository, log func(string, map[string]interface{})) *Handler {
	return &Handler{
		secretsClient: secretsClient,
		profileRepo:   profileRepo,
		signingGen:    generation.NewGenerator(nil),
		log:           log,
	}
}

// NewHandlerWithSigning creates a new bundles handler with signing support.
func NewHandlerWithSigning(secretsClient *secrets.Client, profileRepo profiles.Repository, signingRepo codesigning.Repository, log func(string, map[string]interface{})) *Handler {
	return &Handler{
		secretsClient: secretsClient,
		profileRepo:   profileRepo,
		signingRepo:   signingRepo,
		signingGen:    generation.NewGenerator(nil),
		log:           log,
	}
}

// ValidateBundle validates a desktop bundle manifest.
func (h *Handler) ValidateBundle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to read bundle: %v"}`, err), http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, `{"error":"bundle manifest required"}`, http.StatusBadRequest)
		return
	}

	if err := ValidateManifestBytes(body); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"bundle failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "valid",
		"schema": "desktop.v0.1",
	})
}

// MergeBundleSecrets merges secret plans into a bundle manifest.
func (h *Handler) MergeBundleSecrets(w http.ResponseWriter, r *http.Request) {
	var req MergeSecretsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if req.Scenario == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}
	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}

	// Re-validate manifest before merging.
	rawPayload, err := json.Marshal(req.Manifest)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to marshal manifest: %v"}`, err), http.StatusBadRequest)
		return
	}
	if err := ValidateManifestBytes(rawPayload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	secretPlans, err := h.secretsClient.FetchBundleSecrets(r.Context(), req.Scenario, req.Tier)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
		return
	}

	manifest := req.Manifest
	if err := ApplyBundleSecrets(&manifest, secretPlans); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(manifest)
}

// AssembleBundle assembles a complete bundle manifest for a scenario.
func (h *Handler) AssembleBundle(w http.ResponseWriter, r *http.Request) {
	var req AssembleBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Scenario) == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}
	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}
	includeSecrets := true
	if req.IncludeSecrets != nil {
		includeSecrets = *req.IncludeSecrets
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	manifest, err := FetchSkeletonBundle(ctx, req.Scenario)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to build bundle","details":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	// Apply swaps from profile if provided
	if req.ProfileID != "" && h.profileRepo != nil {
		profileSwaps, err := h.profileRepo.GetSwaps(ctx, req.ProfileID)
		if err != nil && err != profiles.ErrNotFound {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load profile swaps: %v"}`, err), http.StatusBadGateway)
			return
		}
		if len(profileSwaps) > 0 {
			h.applySwapsToManifest(manifest, profileSwaps)
		}
	}

	// Apply signing config from profile if available
	if req.ProfileID != "" {
		if err := h.loadSigningConfig(ctx, req.ProfileID, manifest); err != nil {
			h.log("warn", map[string]interface{}{
				"msg":        "failed to load signing config, continuing without",
				"profile_id": req.ProfileID,
				"error":      err.Error(),
			})
			// Non-fatal: continue without signing config
		}
	}

	if includeSecrets {
		secretPlans, err := h.secretsClient.FetchBundleSecrets(ctx, req.Scenario, req.Tier)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
			return
		}
		if err := ApplyBundleSecrets(manifest, secretPlans); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Validate assembled manifest to guarantee schema compliance before handing off.
	payload, _ := json.Marshal(manifest)
	if err := ValidateManifestBytes(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"assembled manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "assembled",
		"schema":   "desktop.v0.1",
		"manifest": manifest,
	})
}

// applySwapsToManifest adds profile swaps to the manifest's swap list.
func (h *Handler) applySwapsToManifest(manifest *Manifest, profileSwaps []profiles.Swap) {
	for _, ps := range profileSwaps {
		manifest.Swaps = append(manifest.Swaps, ManifestSwap{
			Original:    ps.From,
			Replacement: ps.To,
			Reason:      ps.Reason,
			Limitations: ps.Limitations,
		})
	}
}

// ExportBundle assembles and exports a signed bundle manifest with checksum.
// This is the endpoint for generating production-ready bundle.json files.
func (h *Handler) ExportBundle(w http.ResponseWriter, r *http.Request) {
	var req ExportBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Scenario) == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}
	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}
	includeSecrets := true
	if req.IncludeSecrets != nil {
		includeSecrets = *req.IncludeSecrets
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	manifest, err := FetchSkeletonBundle(ctx, req.Scenario)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to build bundle","details":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	// Apply swaps from profile if provided
	if req.ProfileID != "" && h.profileRepo != nil {
		profileSwaps, err := h.profileRepo.GetSwaps(ctx, req.ProfileID)
		if err != nil && err != profiles.ErrNotFound {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load profile swaps: %v"}`, err), http.StatusBadGateway)
			return
		}
		if len(profileSwaps) > 0 {
			h.applySwapsToManifest(manifest, profileSwaps)
		}
	}

	// Apply signing config from profile if available
	if req.ProfileID != "" {
		if err := h.loadSigningConfig(ctx, req.ProfileID, manifest); err != nil {
			h.log("warn", map[string]interface{}{
				"msg":        "failed to load signing config, continuing without",
				"profile_id": req.ProfileID,
				"error":      err.Error(),
			})
			// Non-fatal: continue without signing config
		}
	}

	if includeSecrets {
		secretPlans, err := h.secretsClient.FetchBundleSecrets(ctx, req.Scenario, req.Tier)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
			return
		}
		if err := ApplyBundleSecrets(manifest, secretPlans); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Validate assembled manifest before export.
	payload, err := json.Marshal(manifest)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to serialize manifest: %v"}`, err), http.StatusInternalServerError)
		return
	}
	if err := ValidateManifestBytes(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"assembled manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Calculate SHA256 checksum of the manifest content.
	hash := sha256.Sum256(payload)
	checksum := hex.EncodeToString(hash[:])

	generatedAt := time.Now().UTC().Format(time.RFC3339)

	response := ExportBundleResponse{
		Status:      "exported",
		Schema:      "desktop.v0.1",
		Scenario:    req.Scenario,
		Tier:        req.Tier,
		Manifest:    *manifest,
		Checksum:    checksum,
		GeneratedAt: generatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// GenerateSigningConfigRequest is the request body for signing config generation.
type GenerateSigningConfigRequest struct {
	ProfileID    string   `json:"profile_id"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// GenerateSigningConfigResponse is the response for signing config generation.
type GenerateSigningConfigResponse struct {
	Status               string                                 `json:"status"`
	ProfileID            string                                 `json:"profile_id"`
	ElectronBuilderConfig map[string]interface{}                `json:"electron_builder_config,omitempty"`
	Files                map[string]string                      `json:"files,omitempty"`
	Message              string                                 `json:"message,omitempty"`
}

// GenerateSigningConfig generates electron-builder signing configuration and supporting files.
// This endpoint takes a profile ID and generates the electron-builder config,
// entitlements.plist, and notarize script based on the profile's signing settings.
// POST /api/v1/bundles/signing-config
func (h *Handler) GenerateSigningConfig(w http.ResponseWriter, r *http.Request) {
	var req GenerateSigningConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.ProfileID == "" {
		http.Error(w, `{"error":"profile_id is required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Check if signing repository is available
	if h.signingRepo == nil {
		http.Error(w, `{"error":"signing configuration not available"}`, http.StatusServiceUnavailable)
		return
	}

	// Fetch signing config from repository
	signingConfig, err := h.signingRepo.Get(ctx, req.ProfileID)
	if err != nil {
		h.log("error", map[string]interface{}{
			"msg":        "failed to get signing config",
			"profile_id": req.ProfileID,
			"error":      err.Error(),
		})
		http.Error(w, `{"error":"failed to retrieve signing configuration"}`, http.StatusInternalServerError)
		return
	}

	if signingConfig == nil || !signingConfig.Enabled {
		h.writeJSON(w, http.StatusOK, GenerateSigningConfigResponse{
			Status:    "disabled",
			ProfileID: req.ProfileID,
			Message:   "Code signing is not enabled for this profile",
		})
		return
	}

	// Generate electron-builder config
	ebConfig, err := generation.GenerateElectronBuilderJSON(signingConfig, nil)
	if err != nil {
		h.log("error", map[string]interface{}{
			"msg":   "failed to generate electron-builder config",
			"error": err.Error(),
		})
		http.Error(w, `{"error":"failed to generate electron-builder config"}`, http.StatusInternalServerError)
		return
	}

	// Generate all supporting files (entitlements, notarize script)
	generatedFiles, err := h.signingGen.GenerateAll(signingConfig)
	if err != nil {
		h.log("error", map[string]interface{}{
			"msg":   "failed to generate signing files",
			"error": err.Error(),
		})
		http.Error(w, `{"error":"failed to generate signing files"}`, http.StatusInternalServerError)
		return
	}

	// Convert byte slices to strings for JSON response
	files := make(map[string]string)
	for path, content := range generatedFiles {
		files[path] = string(content)
	}

	h.log("info", map[string]interface{}{
		"msg":         "generated signing config",
		"profile_id":  req.ProfileID,
		"files_count": len(files),
	})

	h.writeJSON(w, http.StatusOK, GenerateSigningConfigResponse{
		Status:               "generated",
		ProfileID:            req.ProfileID,
		ElectronBuilderConfig: ebConfig,
		Files:                files,
		Message:              fmt.Sprintf("Generated %d signing file(s)", len(files)),
	})
}

// writeJSON writes a JSON response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// LoadSigningConfig loads signing configuration for a profile and applies it to the manifest.
// This is called during AssembleBundle and ExportBundle when a profile has signing configured.
func (h *Handler) loadSigningConfig(ctx context.Context, profileID string, manifest *Manifest) error {
	if h.signingRepo == nil || profileID == "" {
		return nil
	}

	signingConfig, err := h.signingRepo.Get(ctx, profileID)
	if err != nil {
		return fmt.Errorf("failed to load signing config: %w", err)
	}

	if signingConfig != nil && signingConfig.Enabled {
		manifest.CodeSigning = signingConfig
	}

	return nil
}
