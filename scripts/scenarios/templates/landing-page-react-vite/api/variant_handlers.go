package main

import (
	"encoding/json"
	"net/http"
)

// handleVariantSelect handles GET /api/v1/variants/select (OT-P0-016: AB-API)
// Returns the full variant object for frontend compatibility
func handleVariantSelect(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		variant, err := vs.SelectVariant()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(variant)
	}
}

// handlePublicVariantBySlug handles GET /api/v1/public/variants/{slug} (no auth required)
// Used by the public landing page for URL-based variant selection
func handlePublicVariantBySlug(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/public/variants/"):]
		if slug == "" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		variant, err := vs.GetVariantBySlug(slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Only return active variants for public access
		if variant.Status != "active" {
			http.Error(w, "Variant not available", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantBySlug handles GET /api/v1/variants/{slug} (OT-P0-014: AB-URL)
func handleVariantBySlug(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" || slug == "select" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		variant, err := vs.GetVariantBySlug(slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantsList handles GET /api/v1/variants (OT-P0-017: AB-CRUD)
func handleVariantsList(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		statusFilter := r.URL.Query().Get("status")

		variants, err := vs.ListVariants(statusFilter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"variants": variants,
		})
	}
}

// handleVariantCreate handles POST /api/v1/variants (OT-P0-017: AB-CRUD)
// DEPRECATED: Use handleVariantCreateWithSections instead
func handleVariantCreate(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Slug        string            `json:"slug"`
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Weight      int               `json:"weight"`
			Axes        map[string]string `json:"axes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Slug == "" || req.Name == "" {
			http.Error(w, "slug and name are required", http.StatusBadRequest)
			return
		}

		if len(req.Axes) == 0 {
			http.Error(w, "axes selection is required", http.StatusBadRequest)
			return
		}

		variant, err := vs.CreateVariant(req.Slug, req.Name, req.Description, req.Weight, req.Axes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantCreateWithSections creates a variant and copies sections from Control
func handleVariantCreateWithSections(vs *VariantService, cs *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Slug        string            `json:"slug"`
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Weight      int               `json:"weight"`
			Axes        map[string]string `json:"axes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Slug == "" || req.Name == "" {
			http.Error(w, "slug and name are required", http.StatusBadRequest)
			return
		}

		if len(req.Axes) == 0 {
			http.Error(w, "axes selection is required", http.StatusBadRequest)
			return
		}

		variant, err := vs.CreateVariant(req.Slug, req.Name, req.Description, req.Weight, req.Axes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Copy sections from Control variant (id=1) to give new variant default content
		if err := cs.CopySectionsFromVariant(1, int64(variant.ID)); err != nil {
			// Log error but don't fail - variant was created successfully
			logStructuredError("Failed to copy sections to new variant", map[string]interface{}{
				"variant_id":   variant.ID,
				"variant_slug": variant.Slug,
				"error":        err.Error(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantUpdate handles PATCH /api/v1/variants/{slug} (OT-P0-017: AB-CRUD)
func handleVariantUpdate(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		var req struct {
			Name        *string           `json:"name,omitempty"`
			Description *string           `json:"description,omitempty"`
			Weight      *int              `json:"weight,omitempty"`
			Axes        map[string]string `json:"axes,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		variant, err := vs.UpdateVariant(slug, req.Name, req.Description, req.Weight, req.Axes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantArchive handles POST /api/v1/variants/{slug}/archive (OT-P0-018: AB-ARCHIVE)
func handleVariantArchive(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		slug = slug[:len(slug)-len("/archive")]

		if slug == "" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		if err := vs.ArchiveVariant(slug); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Variant archived successfully",
			"slug":    slug,
		})
	}
}

// handleVariantDelete handles DELETE /api/v1/variants/{slug} (OT-P0-017: AB-CRUD)
func handleVariantDelete(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		if err := vs.DeleteVariant(slug); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Variant deleted successfully",
			"slug":    slug,
		})
	}
}
