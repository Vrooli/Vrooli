// Package skills provides the core domain types and operations for skill management.
package skills

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

// Handlers provides HTTP handlers for skill operations.
// Depends on interfaces (SkillStore, MetricsService) for testability.
type Handlers struct {
	store   SkillStore
	metrics MetricsService
}

// NewHandlers creates a new skills handler.
// Accepts any implementation of SkillStore and MetricsService interfaces.
func NewHandlers(store SkillStore, metrics MetricsService) *Handlers {
	return &Handlers{
		store:   store,
		metrics: metrics,
	}
}

// List handles GET /skills - returns all skills with optional filtering.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")
	folder := r.URL.Query().Get("folder")
	modes := r.URL.Query()["modes"]

	skills, err := h.store.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply domain filters
	skills = Filter(skills, FilterOptions{
		Tag:    tag,
		Folder: folder,
		Modes:  modes,
	})

	// Convert to response format with metrics
	responses := make([]Response, 0, len(skills))
	for _, p := range skills {
		responses = append(responses, h.toResponse(p))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// Sync handles GET /skills/sync - returns skills with content for syncing.
func (h *Handlers) Sync(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")

	skills, err := h.store.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply domain filters
	skills = Filter(skills, FilterOptions{Tag: tag})

	// Sort for consistent hashing
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].ID < skills[j].ID
	})

	// Build response with content
	var responses []Response
	var lastUpdated time.Time

	for _, p := range skills {
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
		Skills:      responses,
		LastUpdated: lastUpdated.Format(time.RFC3339),
		Hash:        hashStr,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(syncResponse)
}

// Get handles GET /skills/{id} - returns a single skill.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	skill, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	// Load content
	content, err := h.store.GetContent(folder, skill.File)
	if err != nil {
		http.Error(w, "Failed to load skill content", http.StatusInternalServerError)
		return
	}

	response := h.toResponse(*skill)
	response.Content = content
	response.Folder = folder

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Create handles POST /skills - creates a new skill.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate folder
	if !IsWritableFolder(req.Folder) {
		http.Error(w, "Can only create skills in 'local' or 'drafts' folders", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Content == "" {
		http.Error(w, "Name and content are required", http.StatusBadRequest)
		return
	}

	// Generate unique ID if not provided
	if req.ID == "" {
		idExists := func(id string) bool {
			_, _, err := h.store.FindByID(id)
			return err == nil
		}

		uniqueID, err := GenerateUniqueID(req.Name, idExists)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		req.ID = uniqueID
	} else {
		// User provided explicit ID - check for conflict
		if _, _, err := h.store.FindByID(req.ID); err == nil {
			http.Error(w, "Skill with this ID already exists", http.StatusConflict)
			return
		}
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

	// Load existing skills for the folder
	skills, err := h.store.LoadMetadata(req.Folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add new skill
	skills = append(skills, metadata)

	// Save content file
	if err := h.store.SaveContent(req.Folder, filename, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save metadata
	if err := h.store.SaveMetadata(req.Folder, skills); err != nil {
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

// Update handles PUT /skills/{id} - updates an existing skill.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	skill, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	// Only allow updates to local/drafts skills
	if !IsWritableFolder(folder) {
		http.Error(w, "Cannot update core skills", http.StatusForbidden)
		return
	}

	// Update fields
	if req.Name != nil {
		skill.Name = *req.Name
	}
	if req.Description != nil {
		skill.Description = *req.Description
	}
	if req.Modes != nil {
		skill.Modes = req.Modes
	}
	if req.Tags != nil {
		skill.Tags = req.Tags
	}
	if req.Icon != nil {
		skill.Icon = *req.Icon
	}
	if req.TargetToolID != nil {
		skill.TargetToolID = req.TargetToolID
	}
	if req.Draft != nil {
		skill.Draft = *req.Draft
	}

	skill.UpdatedAt = time.Now().Format(time.RFC3339)

	// Update content if provided
	if req.Content != nil {
		if err := h.store.SaveContent(folder, skill.File, *req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Load all skills and update the matching one
	skills, err := h.store.LoadMetadata(folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, p := range skills {
		if p.ID == id {
			skills[i] = *skill
			break
		}
	}

	if err := h.store.SaveMetadata(folder, skills); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := h.toResponse(*skill)
	response.Folder = folder

	// Load content for response
	if content, err := h.store.GetContent(folder, skill.File); err == nil {
		response.Content = content
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Delete handles DELETE /skills/{id} - deletes a skill.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	skill, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	// Only allow deletes from local/drafts
	if !IsWritableFolder(folder) {
		http.Error(w, "Cannot delete core skills", http.StatusForbidden)
		return
	}

	// Remove from metadata
	skills, err := h.store.LoadMetadata(folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var filtered []Metadata
	for _, p := range skills {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}

	if err := h.store.SaveMetadata(folder, filtered); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete content file (ignore error - best effort)
	h.store.DeleteContent(folder, skill.File)

	// Delete metrics from database
	h.metrics.Delete(id)

	w.WriteHeader(http.StatusNoContent)
}

// RecordUsage handles POST /skills/{id}/use - records skill usage.
func (h *Handlers) RecordUsage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Verify skill exists
	if _, _, err := h.store.FindByID(id); err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
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

// SetRating handles PUT /skills/{id}/rating - sets effectiveness rating.
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

	// Verify skill exists
	if _, _, err := h.store.FindByID(id); err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
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

// Combine handles POST /skills/combine - combines multiple skills into one output.
func (h *Handlers) Combine(w http.ResponseWriter, r *http.Request) {
	var req CombineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.SkillIDs) == 0 {
		http.Error(w, "At least one skill ID is required", http.StatusBadRequest)
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

	// Collect skills
	var responses []Response
	for _, id := range req.SkillIDs {
		skill, folder, err := h.store.FindByID(id)
		if err != nil {
			continue // Skip missing skills
		}

		content, err := h.store.GetContent(folder, skill.File)
		if err != nil {
			continue
		}

		response := h.toResponse(*skill)
		response.Content = content
		response.Folder = folder
		responses = append(responses, response)
	}

	if len(responses) == 0 {
		http.Error(w, "No valid skills found", http.StatusNotFound)
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
		SkillCount:  len(responses),
		TotalTokens: totalTokens,
		Format:      format,
	})
}

// combineToXML generates XML output for combined skills.
func combineToXML(skills []Response) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<combined-skills count=\"")
	b.WriteString(string(rune('0' + len(skills))))
	b.WriteString("\">\n")

	for _, p := range skills {
		modes := strings.Join(p.Modes, "/")
		b.WriteString("  <skill id=\"")
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
		b.WriteString("  </skill>\n")
	}

	b.WriteString("</combined-skills>")
	return b.String()
}

// combineToMarkdown generates Markdown output for combined skills.
func combineToMarkdown(skills []Response) string {
	var b strings.Builder
	b.WriteString("# Combined Skills (")
	b.WriteString(string(rune('0' + len(skills))))
	b.WriteString(")\n\n")
	b.WriteString("---\n\n")

	for i, p := range skills {
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

// combineToJSON generates JSON output for combined skills.
func combineToJSON(skills []Response) string {
	type jsonSkill struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Modes       []string `json:"modes,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		Content     string   `json:"content"`
	}

	type jsonOutput struct {
		Combined bool        `json:"combined"`
		Count    int         `json:"count"`
		Skills   []jsonSkill `json:"skills"`
	}

	output := jsonOutput{
		Combined: true,
		Count:    len(skills),
		Skills:   make([]jsonSkill, len(skills)),
	}

	for i, p := range skills {
		output.Skills[i] = jsonSkill{
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
