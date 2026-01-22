package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"
)

// VariantResponse is the flat variant format expected by the UI
type VariantResponse struct {
	ID           int                 `json:"id,omitempty"`
	Slug         string              `json:"slug"`
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	Weight       int                 `json:"weight"`
	Status       string              `json:"status"`
	Axes         map[string]string   `json:"axes,omitempty"`
	HeaderConfig LandingHeaderConfig `json:"header_config,omitempty"`
	CreatedAt    string              `json:"created_at,omitempty"`
	UpdatedAt    string              `json:"updated_at,omitempty"`
}

// snapshotToVariantResponse converts a VariantSnapshot to the flat VariantResponse format
func snapshotToVariantResponse(snapshot *VariantSnapshot) VariantResponse {
	return VariantResponse{
		Slug:         snapshot.Variant.Slug,
		Name:         snapshot.Variant.Name,
		Description:  snapshot.Variant.Description,
		Weight:       getVariantWeight(snapshot),
		Status:       "active",
		Axes:         snapshot.Variant.Axes,
		HeaderConfig: snapshot.Variant.HeaderConfig,
		UpdatedAt:    time.Now().Format(time.RFC3339), // Use current time as approximation
	}
}

// getVariantWeight returns the weight for a variant (default 50 if not set)
func getVariantWeight(snapshot *VariantSnapshot) int {
	if snapshot.Variant.Weight > 0 {
		return snapshot.Variant.Weight
	}
	return 50 // Default weight
}

// selectWeightedRandomVariant picks a random variant based on weights
// All variants with weight > 0 participate; weight 0 means disabled
func selectWeightedRandomVariant(variants []*VariantSnapshot) *VariantSnapshot {
	if len(variants) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0
	for _, v := range variants {
		w := getVariantWeight(v)
		if w > 0 {
			totalWeight += w
		}
	}

	// If all weights are 0, return first variant
	if totalWeight == 0 {
		return variants[0]
	}

	// Pick a random point in the total weight range
	pick := rand.Intn(totalWeight)

	// Find which variant this falls into
	cumulative := 0
	for _, v := range variants {
		w := getVariantWeight(v)
		if w > 0 {
			cumulative += w
			if pick < cumulative {
				return v
			}
		}
	}

	// Fallback (shouldn't happen)
	return variants[0]
}

// handleVariantSelect handles GET /api/v1/variants/select (OT-P0-016: AB-API)
// Returns a randomly selected variant based on weights for A/B testing
// Transforms VariantSnapshot to flat VariantResponse format for UI compatibility
func handleVariantSelect(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		snapshots := cs.ListVariants()
		if len(snapshots) == 0 {
			writeJSONError(w, http.StatusInternalServerError, "No variants available.", ApiErrorTypeServerError)
			return
		}

		// Use weighted random selection for A/B testing
		snapshot := selectWeightedRandomVariant(snapshots)

		logStructured("variant_selected", map[string]interface{}{
			"slug":   snapshot.Variant.Slug,
			"name":   snapshot.Variant.Name,
			"weight": getVariantWeight(snapshot),
		})

		// Transform to flat format expected by UI
		variant := snapshotToVariantResponse(snapshot)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			logStructuredError("variant_select_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handlePublicVariantBySlug handles GET /api/v1/public/variants/{slug} (no auth required)
// Used by the public landing page for URL-based variant selection
// Transforms VariantSnapshot to flat VariantResponse format for UI compatibility
func handlePublicVariantBySlug(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		slug := r.URL.Path[len("/api/v1/public/variants/"):]
		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug is required.", ApiErrorTypeValidation)
			return
		}

		snapshot, err := cs.GetVariant(slug)
		if err != nil {
			logStructuredError("public_variant_fetch_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusNotFound, "Variant not found.", ApiErrorTypeNotFound)
			return
		}

		// Transform to flat format expected by UI
		variant := snapshotToVariantResponse(snapshot)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			logStructuredError("public_variant_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantBySlug handles GET /api/v1/variants/{slug} (OT-P0-014: AB-URL)
// Transforms VariantSnapshot to flat VariantResponse format for UI compatibility
func handleVariantBySlug(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" || slug == "select" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug is required.", ApiErrorTypeValidation)
			return
		}

		snapshot, err := cs.GetVariant(slug)
		if err != nil {
			logStructuredError("variant_fetch_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusNotFound, "Variant not found.", ApiErrorTypeNotFound)
			return
		}

		// Transform to flat format expected by UI
		variant := snapshotToVariantResponse(snapshot)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			logStructuredError("variant_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantsList handles GET /api/v1/variants (OT-P0-017: AB-CRUD)
// Returns all variants from ConfigStore (loaded from JSON files)
// Transforms VariantSnapshot to flat VariantResponse format for UI compatibility
func handleVariantsList(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}

		snapshots := cs.ListVariants()

		// Transform to flat format expected by UI
		variants := make([]VariantResponse, 0, len(snapshots))
		for _, snapshot := range snapshots {
			variants = append(variants, snapshotToVariantResponse(snapshot))
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"variants": variants,
		}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to encode response", ApiErrorTypeServerError)
		}
	}
}

// handleVariantUpdate handles PATCH /api/v1/variants/{slug} (OT-P0-017: AB-CRUD)
// Updates variant in ConfigStore and writes to JSON file
func handleVariantUpdate(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug required", ApiErrorTypeValidation)
			return
		}

		// Get existing variant
		existing, err := cs.GetVariant(slug)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "Variant not found.", ApiErrorTypeNotFound)
			return
		}

		// Parse update request
		var req struct {
			Name         *string              `json:"name,omitempty"`
			Description  *string              `json:"description,omitempty"`
			Weight       *int                 `json:"weight,omitempty"`
			Axes         map[string]string    `json:"axes,omitempty"`
			HeaderConfig *LandingHeaderConfig `json:"header_config,omitempty"`
			SEOConfig    json.RawMessage      `json:"seo_config,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		// Apply updates
		if req.Name != nil {
			existing.Variant.Name = *req.Name
		}
		if req.Description != nil {
			existing.Variant.Description = *req.Description
		}
		if req.Weight != nil {
			existing.Variant.Weight = *req.Weight
		}
		if req.Axes != nil {
			existing.Variant.Axes = req.Axes
		}
		if req.HeaderConfig != nil {
			existing.Variant.HeaderConfig = normalizeLandingHeaderConfig(req.HeaderConfig, existing.Variant.Name)
		}
		if len(req.SEOConfig) > 0 {
			existing.Variant.SEOConfig = req.SEOConfig
		}

		// Save to JSON file
		if err := cs.SaveVariant(slug, existing); err != nil {
			logStructuredError("variant_update_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to update variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		logStructured("variant_updated", map[string]interface{}{
			"slug": slug,
		})

		// Transform to flat format expected by UI
		variant := snapshotToVariantResponse(existing)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to encode response", ApiErrorTypeServerError)
		}
	}
}

// handleVariantDelete handles DELETE /api/v1/variants/{slug} (OT-P0-017: AB-CRUD)
// Deletes variant JSON file and removes from ConfigStore
func handleVariantDelete(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug required", ApiErrorTypeValidation)
			return
		}

		if err := cs.DeleteVariant(slug); err != nil {
			logStructuredError("variant_delete_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to delete variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		logStructured("variant_deleted", map[string]interface{}{
			"slug": slug,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"message": "Variant deleted successfully",
			"slug":    slug,
		}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to encode response", ApiErrorTypeServerError)
		}
	}
}

// handleVariantExport handles GET /api/v1/admin/variants/{slug}/export
// Returns the variant snapshot from ConfigStore
func handleVariantExport(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		slug := r.URL.Path[len("/api/v1/admin/variants/"):]
		slug = slug[:len(slug)-len("/export")]
		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug is required.", ApiErrorTypeValidation)
			return
		}

		snapshot, err := cs.GetVariant(slug)
		if err != nil {
			logStructuredError("variant_export_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to export variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			logStructuredError("variant_export_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantImport handles PUT /api/v1/admin/variants/{slug}/import
// Saves a full variant snapshot to its JSON file
func handleVariantImport(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		slug := r.URL.Path[len("/api/v1/admin/variants/"):]
		slug = slug[:len(slug)-len("/import")]
		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug is required.", ApiErrorTypeValidation)
			return
		}

		var payload VariantSnapshotInput
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request format.", ApiErrorTypeValidation)
			return
		}

		// Validate slug matches
		if payload.Variant.Slug != slug {
			writeJSONError(w, http.StatusBadRequest, "Payload slug does not match route slug.", ApiErrorTypeValidation)
			return
		}

		// Convert input to snapshot
		headerCfg := normalizeLandingHeaderConfig(payload.Variant.HeaderConfig, payload.Variant.Name)
		sections := make([]VariantSection, 0, len(payload.Sections))
		for idx, sec := range payload.Sections {
			order := sec.Order
			if order <= 0 {
				order = idx + 1
			}
			enabled := true
			if sec.Enabled != nil {
				enabled = *sec.Enabled
			}
			sections = append(sections, VariantSection{
				SectionType: sec.SectionType,
				Content:     sec.Content,
				Order:       order,
				Enabled:     enabled,
			})
		}

		snapshot := &VariantSnapshot{
			Variant: VariantSnapshotMeta{
				Slug:         payload.Variant.Slug,
				Name:         payload.Variant.Name,
				Description:  payload.Variant.Description,
				Axes:         payload.Variant.Axes,
				HeaderConfig: headerCfg,
				SEOConfig:    payload.Variant.SEOConfig,
			},
			Sections: sections,
		}

		if err := cs.SaveVariant(slug, snapshot); err != nil {
			logStructuredError("variant_import_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to import variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		logStructured("variant_imported", map[string]interface{}{
			"slug": slug,
		})

		// Return the saved snapshot
		saved, _ := cs.GetVariant(slug)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(saved); err != nil {
			logStructuredError("variant_import_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantSnapshotSync handles POST /api/v1/admin/variants/sync
// Reloads all variants from JSON files into memory
func handleVariantSnapshotSync(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		if err := cs.LoadAll(); err != nil {
			logStructuredError("variant_snapshot_sync_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to sync variant snapshots. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("variant_snapshots_synced", map[string]interface{}{
			"count": cs.VariantCount(),
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"count":  cs.VariantCount(),
		}); err != nil {
			logStructuredError("variant_sync_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}
