package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

// VariantHandlers provides HTTP handlers for variant operations.
type VariantHandlers struct {
	variants store.VariantStore
	skills   store.SkillStore
}

// NewVariantHandlers creates variant handlers.
func NewVariantHandlers(variants store.VariantStore, skills store.SkillStore) *VariantHandlers {
	return &VariantHandlers{variants: variants, skills: skills}
}

// ListSkillVariants returns all variants for a skill.
func (h *VariantHandlers) ListSkillVariants(ctx context.Context, skillID string) ([]VariantResponse, error) {
	variants, err := h.variants.List(ctx, skillID)
	if err != nil {
		return nil, err
	}

	resp := make([]VariantResponse, 0, len(variants))
	for _, v := range variants {
		resp = append(resp, variantToResponse(v, ""))
	}

	return resp, nil
}

// ListVariants handles GET /skills/{id}/variants
func (h *VariantHandlers) ListVariants(w http.ResponseWriter, r *http.Request) {
	variants, err := h.ListSkillVariants(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(variants)
}

// GetSkillVariant returns one variant and its content.
func (h *VariantHandlers) GetSkillVariant(ctx context.Context, skillID, variantID string) (VariantResponse, error) {
	v, content, err := h.variants.GetWithContent(ctx, skillID, variantID)
	if err != nil {
		return VariantResponse{}, err
	}
	return variantToResponse(*v, content), nil
}

// GetVariant handles GET /skills/{id}/variants/{vid}
func (h *VariantHandlers) GetVariant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	variant, err := h.GetSkillVariant(r.Context(), vars["id"], vars["vid"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(variant)
}

// CreateSkillVariant creates a variant and reads it back with its generated metadata.
func (h *VariantHandlers) CreateSkillVariant(ctx context.Context, skillID string, req CreateVariantRequest) (VariantResponse, error) {
	if req.ID == "" || req.Name == "" || req.Content == "" {
		return VariantResponse{}, fmt.Errorf("id, name, and content are required")
	}

	variant := &store.Variant{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.variants.Create(ctx, skillID, variant, req.Content); err != nil {
		return VariantResponse{}, err
	}

	// Re-read to get the full object with timestamps
	created, content, err := h.variants.GetWithContent(ctx, skillID, req.ID)
	if err != nil {
		return VariantResponse{}, fmt.Errorf("created but failed to read back: %w", err)
	}
	return variantToResponse(*created, content), nil
}

// CreateVariant handles POST /skills/{id}/variants
func (h *VariantHandlers) CreateVariant(w http.ResponseWriter, r *http.Request) {
	skillID := mux.Vars(r)["id"]
	var req CreateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := h.CreateSkillVariant(r.Context(), skillID, req)
	if err != nil {
		status := http.StatusConflict
		if err.Error() == "id, name, and content are required" {
			status = http.StatusBadRequest
		} else if strings.HasPrefix(err.Error(), "created but failed") {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// UpdateSkillVariant updates a variant and reads it back with its content.
func (h *VariantHandlers) UpdateSkillVariant(ctx context.Context, skillID, variantID string, req UpdateVariantRequest) (VariantResponse, error) {
	updates := &store.Variant{}
	if req.Name != nil {
		updates.Name = *req.Name
	}
	if req.Description != nil {
		updates.Description = *req.Description
	}

	if err := h.variants.Update(ctx, skillID, variantID, updates, req.Content); err != nil {
		return VariantResponse{}, err
	}

	updated, content, err := h.variants.GetWithContent(ctx, skillID, variantID)
	if err != nil {
		return VariantResponse{}, err
	}
	return variantToResponse(*updated, content), nil
}

// UpdateVariant handles PUT /skills/{id}/variants/{vid}
func (h *VariantHandlers) UpdateVariant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req UpdateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updated, err := h.UpdateSkillVariant(r.Context(), vars["id"], vars["vid"], req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

// DeleteSkillVariant deletes a variant.
func (h *VariantHandlers) DeleteSkillVariant(ctx context.Context, skillID, variantID string) error {
	return h.variants.Delete(ctx, skillID, variantID)
}

// DeleteVariant handles DELETE /skills/{id}/variants/{vid}
func (h *VariantHandlers) DeleteVariant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["id"]
	variantID := vars["vid"]

	if err := h.DeleteSkillVariant(r.Context(), skillID, variantID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func variantToResponse(v store.Variant, content string) VariantResponse {
	return VariantResponse{
		ID:          v.ID,
		SkillID:     v.SkillID,
		Name:        v.Name,
		Description: v.Description,
		Content:     content,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
		Revision:    v.Revision,
	}
}
