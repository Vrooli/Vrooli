package main

import (
	"encoding/json"
	"net/http"
)

// handleVariantSelect handles GET /api/v1/variants/select (OT-P0-016: AB-API)
// Returns the full variant object for frontend compatibility
// [REQ:SIGNAL-FEEDBACK] Logs variant selection for observability
func handleVariantSelect(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		variant, err := vs.SelectVariant()
		if err != nil {
			logStructuredError("variant_select_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to select variant. Please try again.", ApiErrorTypeServerError)
			return
		}

		// Log successful variant selection for traffic analysis
		logStructured("variant_selected", map[string]interface{}{
			"slug":   variant.Slug,
			"name":   variant.Name,
			"status": variant.Status,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			logStructuredError("variant_select_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handlePublicVariantBySlug handles GET /api/v1/public/variants/{slug} (no auth required)
// Used by the public landing page for URL-based variant selection
func handlePublicVariantBySlug(vs *VariantService) http.HandlerFunc {
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

		variant, err := vs.GetVariantBySlug(slug)
		if err != nil {
			logStructuredError("public_variant_fetch_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusNotFound, "Variant not found.", ApiErrorTypeNotFound)
			return
		}

		// Only return active variants for public access
		if variant.Status != "active" {
			writeJSONError(w, http.StatusNotFound, "Variant not available.", ApiErrorTypeNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			logStructuredError("public_variant_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantBySlug handles GET /api/v1/variants/{slug} (OT-P0-014: AB-URL)
func handleVariantBySlug(vs *VariantService) http.HandlerFunc {
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

		variant, err := vs.GetVariantBySlug(slug)
		if err != nil {
			logStructuredError("variant_fetch_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusNotFound, "Variant not found.", ApiErrorTypeNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			logStructuredError("variant_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantsList handles GET /api/v1/variants (OT-P0-017: AB-CRUD)
func handleVariantsList(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}

		statusFilter := r.URL.Query().Get("status")

		variants, err := vs.ListVariants(statusFilter)
		if err != nil {
			logStructuredError("variant_list_failed", map[string]interface{}{
				"error":  err.Error(),
				"filter": statusFilter,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to load variants. Please try again.", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"variants": variants,
		}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to encode response", ApiErrorTypeServerError)
		}
	}
}

// handleVariantCreate handles POST /api/v1/variants (OT-P0-017: AB-CRUD)
// DEPRECATED: Use handleVariantCreateWithSections instead
//nolint:unused // legacy handler retained for backward compatibility
func handleVariantCreate(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
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
			writeJSONError(w, http.StatusBadRequest, "Invalid request format.", ApiErrorTypeValidation)
			return
		}

		if req.Slug == "" || req.Name == "" {
			writeJSONError(w, http.StatusBadRequest, "Slug and name are required.", ApiErrorTypeValidation)
			return
		}

		if len(req.Axes) == 0 {
			writeJSONError(w, http.StatusBadRequest, "Axes selection is required.", ApiErrorTypeValidation)
			return
		}

		variant, err := vs.CreateVariant(req.Slug, req.Name, req.Description, req.Weight, req.Axes)
		if err != nil {
			logStructuredError("variant_create_failed", map[string]interface{}{
				"slug":  req.Slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to create variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			logStructuredError("variant_create_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantCreateWithSections creates a variant and copies sections from Control
// [REQ:SIGNAL-FEEDBACK] Logs variant creation for admin audit trail
func handleVariantCreateWithSections(vs *VariantService, cs *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
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
			writeJSONError(w, http.StatusBadRequest, "Invalid request format.", ApiErrorTypeValidation)
			return
		}

		if req.Slug == "" || req.Name == "" {
			writeJSONError(w, http.StatusBadRequest, "Slug and name are required.", ApiErrorTypeValidation)
			return
		}

		if len(req.Axes) == 0 {
			writeJSONError(w, http.StatusBadRequest, "Axes selection is required.", ApiErrorTypeValidation)
			return
		}

		variant, err := vs.CreateVariant(req.Slug, req.Name, req.Description, req.Weight, req.Axes)
		if err != nil {
			logStructuredError("variant_create_with_sections_failed", map[string]interface{}{
				"slug":  req.Slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to create variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		// Copy sections from Control variant (id=1) to give new variant default content
		sectionsCopied := false
		if err := cs.CopySectionsFromVariant(1, int64(variant.ID)); err != nil {
			// Log error but don't fail - variant was created successfully (graceful degradation)
			logStructuredError("variant_section_copy_failed", map[string]interface{}{
				"variant_id":   variant.ID,
				"variant_slug": variant.Slug,
				"error":        err.Error(),
			})
		} else {
			sectionsCopied = true
		}

		// Log successful variant creation for admin audit trail
		logStructured("variant_created", map[string]interface{}{
			"slug":            variant.Slug,
			"name":            variant.Name,
			"weight":          variant.Weight,
			"sections_copied": sectionsCopied,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			logStructuredError("variant_create_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantUpdate handles PATCH /api/v1/variants/{slug} (OT-P0-017: AB-CRUD)
// [REQ:SIGNAL-FEEDBACK] Logs variant updates for admin audit trail
func handleVariantUpdate(vs *VariantService) http.HandlerFunc {
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

		var req struct {
			Name         *string              `json:"name,omitempty"`
			Description  *string              `json:"description,omitempty"`
			Weight       *int                 `json:"weight,omitempty"`
			Axes         map[string]string    `json:"axes,omitempty"`
			HeaderConfig *LandingHeaderConfig `json:"header_config,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		variant, err := vs.UpdateVariant(slug, req.Name, req.Description, req.Weight, req.Axes, req.HeaderConfig)
		if err != nil {
			logStructuredError("variant_update_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to update variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		// Log successful variant update for admin audit trail
		updateFields := []string{}
		if req.Name != nil {
			updateFields = append(updateFields, "name")
		}
		if req.Description != nil {
			updateFields = append(updateFields, "description")
		}
		if req.Weight != nil {
			updateFields = append(updateFields, "weight")
		}
		if len(req.Axes) > 0 {
			updateFields = append(updateFields, "axes")
		}
		if req.HeaderConfig != nil {
			updateFields = append(updateFields, "header_config")
		}
		logStructured("variant_updated", map[string]interface{}{
			"slug":           slug,
			"updated_fields": updateFields,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(variant); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to encode response", ApiErrorTypeServerError)
		}
	}
}

// handleVariantArchive handles POST /api/v1/variants/{slug}/archive (OT-P0-018: AB-ARCHIVE)
// [REQ:SIGNAL-FEEDBACK] Logs variant archival for admin audit trail
func handleVariantArchive(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		slug = slug[:len(slug)-len("/archive")]

		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug required", ApiErrorTypeValidation)
			return
		}

		if err := vs.ArchiveVariant(slug); err != nil {
			logStructuredError("variant_archive_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to archive variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		// Log successful variant archival for admin audit trail
		logStructured("variant_archived", map[string]interface{}{
			"slug": slug,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"message": "Variant archived successfully",
			"slug":    slug,
		}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to encode response", ApiErrorTypeServerError)
		}
	}
}

// handleVariantDelete handles DELETE /api/v1/variants/{slug} (OT-P0-017: AB-CRUD)
// [REQ:SIGNAL-FEEDBACK] Logs variant deletion for admin audit trail
func handleVariantDelete(vs *VariantService) http.HandlerFunc {
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

		if err := vs.DeleteVariant(slug); err != nil {
			logStructuredError("variant_delete_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, "Failed to delete variant: "+err.Error(), ApiErrorTypeValidation)
			return
		}

		// Log successful variant deletion for admin audit trail
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
// Returns the full variant snapshot (metadata + sections) for bulk edits.
func handleVariantExport(vs *VariantService, cs *ContentService) http.HandlerFunc {
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

		snapshot, err := vs.ExportVariantSnapshot(slug, cs)
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
// Accepts a full variant snapshot and applies it transactionally.
func handleVariantImport(vs *VariantService, cs *ContentService) http.HandlerFunc {
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

		snapshot, err := vs.ImportVariantSnapshot(slug, payload, cs)
		if err != nil {
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

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			logStructuredError("variant_import_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleVariantSnapshotSync handles POST /api/v1/admin/variants/sync
// Re-imports variant snapshots from disk into the database.
func handleVariantSnapshotSync(vs *VariantService, cs *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		if err := syncVariantSnapshots(vs, cs); err != nil {
			logStructuredError("variant_snapshot_sync_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to sync variant snapshots. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("variant_snapshots_synced", nil)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			logStructuredError("variant_sync_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}
