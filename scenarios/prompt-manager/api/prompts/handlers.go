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

// Combine handles POST /prompts/combine - combines multiple prompts into one output.
func (h *Handlers) Combine(w http.ResponseWriter, r *http.Request) {
	var req CombineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.PromptIDs) == 0 {
		http.Error(w, "At least one prompt ID is required", http.StatusBadRequest)
		return
	}

	// Default format
	format := req.Format
	if format == "" {
		format = "xml"
	}
	if format != "xml" && format != "markdown" && format != "json" {
		http.Error(w, "Format must be 'xml', 'markdown', or 'json'", http.StatusBadRequest)
		return
	}

	// Collect prompts
	var responses []Response
	for _, id := range req.PromptIDs {
		prompt, folder, err := h.store.FindByID(id)
		if err != nil {
			continue // Skip missing prompts
		}

		content, err := h.store.GetContent(folder, prompt.File)
		if err != nil {
			continue
		}

		response := h.toResponse(*prompt)
		response.Content = content
		response.Folder = folder
		responses = append(responses, response)
	}

	if len(responses) == 0 {
		http.Error(w, "No valid prompts found", http.StatusNotFound)
		return
	}

	// Generate combined output
	var combined string
	switch format {
	case "xml":
		combined = combineToXML(responses)
	case "markdown":
		combined = combineToMarkdown(responses)
	case "json":
		combined = combineToJSON(responses)
	}

	// Estimate tokens (~4 chars per token)
	totalTokens := (len(combined) + 3) / 4

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CombineResponse{
		Combined:    combined,
		PromptCount: len(responses),
		TotalTokens: totalTokens,
		Format:      format,
	})
}

// combineToXML generates XML output for combined prompts.
func combineToXML(prompts []Response) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<combined-prompts count=\"")
	b.WriteString(string(rune('0' + len(prompts))))
	b.WriteString("\">\n")

	for _, p := range prompts {
		modes := strings.Join(p.Modes, "/")
		b.WriteString("  <prompt id=\"")
		b.WriteString(escapeXML(p.ID))
		b.WriteString("\" name=\"")
		b.WriteString(escapeXML(p.Name))
		b.WriteString("\"")
		if modes != "" {
			b.WriteString(" modes=\"")
			b.WriteString(escapeXML(modes))
			b.WriteString("\"")
		}
		b.WriteString(">\n")

		if p.Description != "" {
			b.WriteString("    <description>")
			b.WriteString(escapeXML(p.Description))
			b.WriteString("</description>\n")
		}

		if len(p.Tags) > 0 {
			b.WriteString("    <tags>")
			b.WriteString(escapeXML(strings.Join(p.Tags, ", ")))
			b.WriteString("</tags>\n")
		}

		b.WriteString("    <content><![CDATA[\n")
		b.WriteString(p.Content)
		b.WriteString("\n]]></content>\n")
		b.WriteString("  </prompt>\n")
	}

	b.WriteString("</combined-prompts>")
	return b.String()
}

// combineToMarkdown generates Markdown output for combined prompts.
func combineToMarkdown(prompts []Response) string {
	var b strings.Builder
	b.WriteString("# Combined Prompts (")
	b.WriteString(string(rune('0' + len(prompts))))
	b.WriteString(")\n\n")
	b.WriteString("---\n\n")

	for i, p := range prompts {
		b.WriteString("## ")
		b.WriteString(string(rune('1' + i)))
		b.WriteString(". ")
		b.WriteString(p.Name)
		b.WriteString("\n\n")

		if p.Description != "" {
			b.WriteString("> ")
			b.WriteString(p.Description)
			b.WriteString("\n\n")
		}

		if len(p.Modes) > 0 {
			b.WriteString("**Modes:** ")
			b.WriteString(strings.Join(p.Modes, " / "))
			b.WriteString("\n")
		}

		if len(p.Tags) > 0 {
			b.WriteString("**Tags:** ")
			for i, tag := range p.Tags {
				if i > 0 {
					b.WriteString(" ")
				}
				b.WriteString("`")
				b.WriteString(tag)
				b.WriteString("`")
			}
			b.WriteString("\n")
		}

		b.WriteString("\n### Content\n\n```\n")
		b.WriteString(p.Content)
		b.WriteString("\n```\n\n---\n\n")
	}

	return b.String()
}

// combineToJSON generates JSON output for combined prompts.
func combineToJSON(prompts []Response) string {
	type jsonPrompt struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Modes       []string `json:"modes,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		Content     string   `json:"content"`
	}

	type jsonOutput struct {
		Combined bool         `json:"combined"`
		Count    int          `json:"count"`
		Prompts  []jsonPrompt `json:"prompts"`
	}

	output := jsonOutput{
		Combined: true,
		Count:    len(prompts),
		Prompts:  make([]jsonPrompt, len(prompts)),
	}

	for i, p := range prompts {
		output.Prompts[i] = jsonPrompt{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Modes:       p.Modes,
			Tags:        p.Tags,
			Content:     p.Content,
		}
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	return string(data)
}

// escapeXML escapes special XML characters.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
