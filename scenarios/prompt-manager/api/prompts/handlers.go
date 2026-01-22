// Package prompts provides the core domain types and operations for prompt management.
package prompts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for prompt operations.
// Depends on interfaces (PromptStore, MetricsService) for testability.
type Handlers struct {
	store   PromptStore
	metrics MetricsService
}

// NewHandlers creates a new prompts handler.
// Accepts any implementation of PromptStore and MetricsService interfaces.
func NewHandlers(store PromptStore, metrics MetricsService) *Handlers {
	return &Handlers{
		store:   store,
		metrics: metrics,
	}
}

// List handles GET /prompts - returns all prompts with optional filtering.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")
	folder := r.URL.Query().Get("folder")
	modes := r.URL.Query()["modes"]

	prompts, err := h.store.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply domain filters
	prompts = Filter(prompts, FilterOptions{
		Tag:    tag,
		Folder: folder,
		Modes:  modes,
	})

	// Convert to response format with metrics
	responses := make([]Response, 0, len(prompts))
	for _, p := range prompts {
		responses = append(responses, h.toResponse(p))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// Sync handles GET /prompts/sync - returns prompts with content for syncing.
func (h *Handlers) Sync(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")

	prompts, err := h.store.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply domain filters
	prompts = Filter(prompts, FilterOptions{Tag: tag})

	// Sort for consistent hashing
	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].ID < prompts[j].ID
	})

	// Build response with content
	var responses []Response
	var lastUpdated time.Time

	for _, p := range prompts {
		response := h.toResponseWithContent(p)
		responses = append(responses, response)

		// Track latest update time
		if updated, err := time.Parse(time.RFC3339, p.UpdatedAt); err == nil {
			if updated.After(lastUpdated) {
				lastUpdated = updated
			}
		}
	}

	// Generate hash for change detection
	hashData, _ := json.Marshal(responses)
	hash := sha256.Sum256(hashData)
	hashStr := hex.EncodeToString(hash[:])

	syncResponse := SyncResponse{
		Prompts:     responses,
		LastUpdated: lastUpdated.Format(time.RFC3339),
		Hash:        hashStr,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(syncResponse)
}

// Get handles GET /prompts/{id} - returns a single prompt.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	prompt, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	}

	// Load content
	content, err := h.store.GetContent(folder, prompt.File)
	if err != nil {
		http.Error(w, "Failed to load prompt content", http.StatusInternalServerError)
		return
	}

	response := h.toResponse(*prompt)
	response.Content = content
	response.Folder = folder

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Create handles POST /prompts - creates a new prompt.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate folder
	if !IsWritableFolder(req.Folder) {
		http.Error(w, "Can only create prompts in 'local' or 'drafts' folders", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Content == "" {
		http.Error(w, "Name and content are required", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	if req.ID == "" {
		req.ID = Slugify(req.Name)
	}

	// Check if ID already exists
	if _, _, err := h.store.FindByID(req.ID); err == nil {
		http.Error(w, "Prompt with this ID already exists", http.StatusConflict)
		return
	}

	now := time.Now().Format(time.RFC3339)
	filename := req.ID + ".md"

	// Create metadata entry
	metadata := Metadata{
		ID:           req.ID,
		File:         filename,
		Name:         req.Name,
		Description:  req.Description,
		Modes:        req.Modes,
		Tags:         req.Tags,
		Icon:         req.Icon,
		TargetToolID: req.TargetToolID,
		Draft:        req.Draft,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Load existing prompts for the folder
	prompts, err := h.store.LoadMetadata(req.Folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add new prompt
	prompts = append(prompts, metadata)

	// Save content file
	if err := h.store.SaveContent(req.Folder, filename, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save metadata
	if err := h.store.SaveMetadata(req.Folder, prompts); err != nil {
		// Clean up content file on failure
		h.store.DeleteContent(req.Folder, filename)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := h.toResponse(metadata)
	response.Content = req.Content
	response.Folder = req.Folder

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Update handles PUT /prompts/{id} - updates an existing prompt.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	prompt, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	}

	// Only allow updates to local/drafts prompts
	if !IsWritableFolder(folder) {
		http.Error(w, "Cannot update core prompts", http.StatusForbidden)
		return
	}

	// Update fields
	if req.Name != nil {
		prompt.Name = *req.Name
	}
	if req.Description != nil {
		prompt.Description = *req.Description
	}
	if req.Modes != nil {
		prompt.Modes = req.Modes
	}
	if req.Tags != nil {
		prompt.Tags = req.Tags
	}
	if req.Icon != nil {
		prompt.Icon = *req.Icon
	}
	if req.TargetToolID != nil {
		prompt.TargetToolID = req.TargetToolID
	}
	if req.Draft != nil {
		prompt.Draft = *req.Draft
	}

	prompt.UpdatedAt = time.Now().Format(time.RFC3339)

	// Update content if provided
	if req.Content != nil {
		if err := h.store.SaveContent(folder, prompt.File, *req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Load all prompts and update the matching one
	prompts, err := h.store.LoadMetadata(folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, p := range prompts {
		if p.ID == id {
			prompts[i] = *prompt
			break
		}
	}

	if err := h.store.SaveMetadata(folder, prompts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := h.toResponse(*prompt)
	response.Folder = folder

	// Load content for response
	if content, err := h.store.GetContent(folder, prompt.File); err == nil {
		response.Content = content
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Delete handles DELETE /prompts/{id} - deletes a prompt.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	prompt, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	}

	// Only allow deletes from local/drafts
	if !IsWritableFolder(folder) {
		http.Error(w, "Cannot delete core prompts", http.StatusForbidden)
		return
	}

	// Remove from metadata
	prompts, err := h.store.LoadMetadata(folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var filtered []Metadata
	for _, p := range prompts {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}

	if err := h.store.SaveMetadata(folder, filtered); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete content file (ignore error - best effort)
	h.store.DeleteContent(folder, prompt.File)

	// Delete metrics from database
	h.metrics.Delete(id)

	w.WriteHeader(http.StatusNoContent)
}

// RecordUsage handles POST /prompts/{id}/use - records prompt usage.
func (h *Handlers) RecordUsage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Verify prompt exists
	if _, _, err := h.store.FindByID(id); err != nil {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	}

	usageCount, lastUsed, err := h.metrics.RecordUsage(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "usage recorded",
		"usageCount": usageCount,
		"lastUsed":   lastUsed,
	})
}

// SetRating handles PUT /prompts/{id}/rating - sets effectiveness rating.
func (h *Handlers) SetRating(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Rating int     `json:"rating"`
		Notes  *string `json:"notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Verify prompt exists
	if _, _, err := h.store.FindByID(id); err != nil {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	}

	if err := h.metrics.SetRating(id, req.Rating, req.Notes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "rating updated",
		"rating": req.Rating,
	})
}

// Helper functions

func (h *Handlers) toResponse(p Metadata) Response {
	response := Response{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		Modes:        p.Modes,
		Tags:         p.Tags,
		Icon:         p.Icon,
		TargetToolID: p.TargetToolID,
		Draft:        p.Draft,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}

	// Extract folder from file path
	if parts := strings.SplitN(p.File, "/", 2); len(parts) > 0 {
		response.Folder = parts[0]
	}

	// Load metrics from database
	if m, err := h.metrics.Get(p.ID); err == nil && m != nil {
		response.UsageCount = m.UsageCount
		if m.LastUsed != nil {
			lastUsed := m.LastUsed.Format(time.RFC3339)
			response.LastUsed = &lastUsed
		}
		response.EffectivenessRating = m.EffectivenessRating
	}

	return response
}

func (h *Handlers) toResponseWithContent(p Metadata) Response {
	response := h.toResponse(p)

	// Extract folder and filename
	parts := strings.SplitN(p.File, "/", 2)
	if len(parts) == 2 {
		content, err := h.store.GetContent(parts[0], parts[1])
		if err == nil {
			response.Content = content
		}
	}

	return response
}
