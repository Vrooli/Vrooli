package skills

import (
	"encoding/json"
	"net/http"

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

// ListVariants handles GET /skills/{id}/variants
func (h *VariantHandlers) ListVariants(w http.ResponseWriter, r *http.Request) {
	skillID := mux.Vars(r)["id"]

	variants, err := h.variants.List(r.Context(), skillID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := make([]VariantResponse, 0, len(variants))
	for _, v := range variants {
		resp = append(resp, variantToResponse(v, ""))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetVariant handles GET /skills/{id}/variants/{vid}
func (h *VariantHandlers) GetVariant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["id"]
	variantID := vars["vid"]

	v, content, err := h.variants.GetWithContent(r.Context(), skillID, variantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(variantToResponse(*v, content))
}

// CreateVariant handles POST /skills/{id}/variants
func (h *VariantHandlers) CreateVariant(w http.ResponseWriter, r *http.Request) {
	skillID := mux.Vars(r)["id"]

	var req CreateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Name == "" || req.Content == "" {
		http.Error(w, "id, name, and content are required", http.StatusBadRequest)
		return
	}

	variant := &store.Variant{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.variants.Create(r.Context(), skillID, variant, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// Re-read to get the full object with timestamps
	created, content, err := h.variants.GetWithContent(r.Context(), skillID, req.ID)
	if err != nil {
		http.Error(w, "created but failed to read back: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(variantToResponse(*created, content))
}

// UpdateVariant handles PUT /skills/{id}/variants/{vid}
func (h *VariantHandlers) UpdateVariant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["id"]
	variantID := vars["vid"]

	var req UpdateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updates := &store.Variant{}
	if req.Name != nil {
		updates.Name = *req.Name
	}
	if req.Description != nil {
		updates.Description = *req.Description
	}

	if err := h.variants.Update(r.Context(), skillID, variantID, updates, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	updated, content, err := h.variants.GetWithContent(r.Context(), skillID, variantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(variantToResponse(*updated, content))
}

// DeleteVariant handles DELETE /skills/{id}/variants/{vid}
func (h *VariantHandlers) DeleteVariant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["id"]
	variantID := vars["vid"]

	if err := h.variants.Delete(r.Context(), skillID, variantID); err != nil {
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
