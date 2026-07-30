package variant

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/experimentation"
)

type WriteDependencies struct {
	Store      experimentation.ConfigStorer
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
	LogError   func(string, map[string]any)
}

type Response struct {
	ID           int                                 `json:"id,omitempty"`
	Slug         string                              `json:"slug"`
	Name         string                              `json:"name"`
	Description  string                              `json:"description,omitempty"`
	Weight       int                                 `json:"weight"`
	Status       string                              `json:"status"`
	Axes         map[string]string                   `json:"axes,omitempty"`
	HeaderConfig experimentation.LandingHeaderConfig `json:"header_config,omitempty"`
	UpdatedAt    string                              `json:"updated_at,omitempty"`
}

func response(snapshot *experimentation.VariantSnapshot) Response {
	return Response{Slug: snapshot.Variant.Slug, Name: snapshot.Variant.Name, Description: snapshot.Variant.Description, Weight: experimentation.VariantWeight(snapshot), Status: experimentation.NormalizeVariantStatus(snapshot.Variant.Status), Axes: snapshot.Variant.Axes, HeaderConfig: snapshot.Variant.HeaderConfig, UpdatedAt: time.Now().Format(time.RFC3339)}
}

func Update(deps WriteDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}
		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" {
			deps.WriteError(w, http.StatusBadRequest, "Variant slug required", "validation")
			return
		}
		existing, err := deps.Store.GetVariant(slug)
		if err != nil {
			deps.WriteError(w, http.StatusNotFound, "Variant not found.", "not_found")
			return
		}
		var request struct {
			Name         *string                              `json:"name,omitempty"`
			Description  *string                              `json:"description,omitempty"`
			Weight       *int                                 `json:"weight,omitempty"`
			Axes         map[string]string                    `json:"axes,omitempty"`
			HeaderConfig *experimentation.LandingHeaderConfig `json:"header_config,omitempty"`
			SEOConfig    json.RawMessage                      `json:"seo_config,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		if request.Name != nil {
			existing.Variant.Name = *request.Name
		}
		if request.Description != nil {
			existing.Variant.Description = *request.Description
		}
		if request.Weight != nil {
			existing.Variant.Weight = *request.Weight
		}
		if request.Axes != nil {
			existing.Variant.Axes = request.Axes
		}
		if request.HeaderConfig != nil {
			existing.Variant.HeaderConfig = experimentation.NormalizeLandingHeaderConfig(request.HeaderConfig, existing.Variant.Name)
		}
		if len(request.SEOConfig) > 0 {
			existing.Variant.SEOConfig = request.SEOConfig
		}
		if err := deps.Store.SaveVariant(slug, existing); err != nil {
			deps.LogError("variant_update_failed", map[string]any{"slug": slug, "error": err.Error()})
			deps.WriteError(w, http.StatusBadRequest, "Failed to update variant: "+err.Error(), "validation")
			return
		}
		deps.Log("variant_updated", map[string]any{"slug": slug})
		deps.WriteJSON(w, response(existing))
	}
}

func Delete(deps WriteDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}
		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" {
			deps.WriteError(w, http.StatusBadRequest, "Variant slug required", "validation")
			return
		}
		if err := deps.Store.DeleteVariant(slug); err != nil {
			deps.LogError("variant_delete_failed", map[string]any{"slug": slug, "error": err.Error()})
			deps.WriteError(w, http.StatusBadRequest, "Failed to delete variant: "+err.Error(), "validation")
			return
		}
		deps.Log("variant_deleted", map[string]any{"slug": slug})
		deps.WriteJSON(w, map[string]string{"message": "Variant deleted successfully", "slug": slug})
	}
}

// Export returns the complete persisted snapshot for an admin variant export.
func Export(deps WriteDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		slug, ok := adminVariantSlug(r, "/export")
		if !ok {
			deps.WriteError(w, http.StatusBadRequest, "Variant slug is required.", "validation")
			return
		}

		snapshot, err := deps.Store.GetVariant(slug)
		if err != nil {
			deps.LogError("variant_export_failed", map[string]any{"slug": slug, "error": err.Error()})
			deps.WriteError(w, http.StatusBadRequest, "Failed to export variant: "+err.Error(), "validation")
			return
		}

		deps.WriteJSON(w, snapshot)
	}
}

// Import saves a complete variant snapshot supplied by an administrator.
func Import(deps WriteDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		slug, ok := adminVariantSlug(r, "/import")
		if !ok {
			deps.WriteError(w, http.StatusBadRequest, "Variant slug is required.", "validation")
			return
		}

		var payload experimentation.VariantSnapshotInput
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request format.", "validation")
			return
		}
		if payload.Variant.Slug != slug {
			deps.WriteError(w, http.StatusBadRequest, "Payload slug does not match route slug.", "validation")
			return
		}

		snapshot := snapshotFromInput(payload)
		if err := deps.Store.SaveVariant(slug, snapshot); err != nil {
			deps.LogError("variant_import_failed", map[string]any{"slug": slug, "error": err.Error()})
			deps.WriteError(w, http.StatusBadRequest, "Failed to import variant: "+err.Error(), "validation")
			return
		}

		deps.Log("variant_imported", map[string]any{"slug": slug})
		deps.WriteJSON(w, snapshot)
	}
}

// Sync reloads persisted variant snapshots into the configuration store.
func Sync(deps WriteDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}

		if err := deps.Store.LoadAll(); err != nil {
			deps.LogError("variant_snapshot_sync_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to sync variant snapshots. Please try again.", "server_error")
			return
		}

		count := deps.Store.VariantCount()
		deps.Log("variant_snapshots_synced", map[string]any{"count": count})
		deps.WriteJSON(w, map[string]any{"status": "ok", "count": count})
	}
}

func adminVariantSlug(r *http.Request, suffix string) (string, bool) {
	const prefix = "/api/v1/admin/variants/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}

	slug := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	return slug, slug != "" && !strings.Contains(slug, "/")
}

func snapshotFromInput(payload experimentation.VariantSnapshotInput) *experimentation.VariantSnapshot {
	sections := make([]experimentation.VariantSection, 0, len(payload.Sections))
	for index, section := range payload.Sections {
		order := section.Order
		if order <= 0 {
			order = index + 1
		}

		enabled := true
		if section.Enabled != nil {
			enabled = *section.Enabled
		}

		sections = append(sections, experimentation.VariantSection{
			SectionType: section.SectionType,
			Content:     section.Content,
			Order:       order,
			Enabled:     enabled,
		})
	}

	return &experimentation.VariantSnapshot{
		Variant: experimentation.VariantSnapshotMeta{
			Slug:         payload.Variant.Slug,
			Name:         payload.Variant.Name,
			Description:  payload.Variant.Description,
			Axes:         payload.Variant.Axes,
			HeaderConfig: experimentation.NormalizeLandingHeaderConfig(payload.Variant.HeaderConfig, payload.Variant.Name),
			SEOConfig:    payload.Variant.SEOConfig,
		},
		Sections: sections,
	}
}
