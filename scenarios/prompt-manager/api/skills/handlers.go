// Package skills provides the core domain types and operations for skill management.
//
// DOC: docs/reference/api-endpoints.md#skills
// DOC: docs/internal/SEAMS.md#1-skillsskillstore-interface
package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for skill operations.
// Depends on interfaces (SkillStore, MetricsService) for testability.
type Handlers struct {
	store     SkillStore
	metrics   MetricsService
	aiIndexer AISearchIndexer // Optional: nil if AI search not available
}

// NewHandlers creates a new skills handler.
// Accepts any implementation of SkillStore and MetricsService interfaces.
func NewHandlers(store SkillStore, metrics MetricsService) *Handlers {
	return &Handlers{
		store:   store,
		metrics: metrics,
	}
}

// SetAIIndexer sets the AI search indexer for async index updates.
// This is called after the aisearch.Service is initialized to avoid circular deps.
func (h *Handlers) SetAIIndexer(indexer AISearchIndexer) {
	h.aiIndexer = indexer
}

// triggerIndexAsync asynchronously indexes a skill if AI search is available.
func (h *Handlers) triggerIndexAsync(skillID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		ctx := context.Background()
		if err := h.aiIndexer.IndexSkill(ctx, skillID); err != nil {
			// Log but don't fail - indexing is best effort
			fmt.Printf("[skills] AI index update failed for %s: %v\n", skillID, err)
		}
	}()
}

// triggerDeleteAsync asynchronously removes a skill from the index.
func (h *Handlers) triggerDeleteAsync(skillID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		ctx := context.Background()
		if err := h.aiIndexer.DeleteFromIndex(ctx, skillID); err != nil {
			// Log but don't fail - indexing is best effort
			fmt.Printf("[skills] AI index delete failed for %s: %v\n", skillID, err)
		}
	}()
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

	// Trigger async AI index update
	h.triggerIndexAsync(req.ID)

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

	// Track old filename for potential rename
	oldFile := skill.File

	// Update fields
	if req.File != nil && *req.File != "" {
		// Validate new filename
		newFile := *req.File
		if !strings.HasSuffix(newFile, ".md") {
			newFile = newFile + ".md"
		}
		skill.File = newFile
	}
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

	// Handle file rename if filename changed
	if skill.File != oldFile {
		// Read old content
		oldContent, err := h.store.GetContent(folder, oldFile)
		if err != nil {
			http.Error(w, "Failed to read existing content for rename", http.StatusInternalServerError)
			return
		}
		// Write to new file (use req.Content if provided, otherwise old content)
		newContent := oldContent
		if req.Content != nil {
			newContent = *req.Content
		}
		if err := h.store.SaveContent(folder, skill.File, newContent); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Delete old file
		h.store.DeleteContent(folder, oldFile)
	} else if req.Content != nil {
		// No rename, just update content
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

	// Trigger async AI index update
	h.triggerIndexAsync(id)

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

	// Trigger async AI index delete
	h.triggerDeleteAsync(id)

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

	// Extract folder and filename from file path (format: "folder/filename.md")
	if parts := strings.SplitN(p.File, "/", 2); len(parts) == 2 {
		response.Folder = parts[0]
		response.File = parts[1]
	} else {
		// No folder prefix - just the filename
		response.File = p.File
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

// Display handles POST /skills/display - formats multiple skills into one output.
func (h *Handlers) Display(w http.ResponseWriter, r *http.Request) {
	var req DisplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Identifiers) == 0 {
		http.Error(w, "Identifiers are required", http.StatusBadRequest)
		return
	}

	resolve := strings.ToLower(strings.TrimSpace(req.Resolve))
	if resolve == "" {
		resolve = "auto"
	}
	if !isValidResolveMode(resolve) {
		http.Error(w, "Resolve must be 'auto', 'id', 'file', or 'name'", http.StatusBadRequest)
		return
	}

	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "xml"
	}
	if format != "xml" && format != "markdown" && format != "json" {
		http.Error(w, "Format must be 'xml', 'markdown', or 'json'", http.StatusBadRequest)
		return
	}

	allowMissing := true
	if req.AllowMissing != nil {
		allowMissing = *req.AllowMissing
	}

	indexed, err := loadIndexedSkills(h.store)
	if err != nil {
		http.Error(w, "Failed to load skills", http.StatusInternalServerError)
		return
	}

	resp := DisplayResponse{Format: format, Resolve: resolve}

	var responses []Response
	for _, identifier := range req.Identifiers {
		matches := resolveIdentifier(identifier, resolve, indexed)
		switch len(matches) {
		case 0:
			resp.Missing = append(resp.Missing, ReadIssue{
				Identifier: identifier,
				Reason:     "not_found",
			})
		case 1:
			readSkill, err := h.buildReadResponse(matches[0])
			if err != nil {
				http.Error(w, "Failed to load skill content", http.StatusInternalServerError)
				return
			}
			responses = append(responses, readSkill)
		default:
			resp.Ambiguous = append(resp.Ambiguous, ReadAmbiguous{
				Identifier: identifier,
				Candidates: buildCandidates(matches),
			})
		}
	}

	if !allowMissing && (len(resp.Missing) > 0 || len(resp.Ambiguous) > 0) {
		status := http.StatusNotFound
		if len(resp.Ambiguous) > 0 {
			status = http.StatusConflict
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(responses) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(resp)
		return
	}

	var combined string
	switch format {
	case "xml":
		combined = displayToXML(responses)
	case "markdown":
		combined = displayToMarkdown(responses)
	case "json":
		combined = displayToJSON(responses)
	}

	totalTokens := (len(combined) + 3) / 4
	resp.Combined = combined
	resp.SkillCount = len(responses)
	resp.TotalTokens = totalTokens

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// displayToXML generates XML output for displayed skills.
func displayToXML(skills []Response) string {
	var b strings.Builder
	b.WriteString("<skills count=\"")
	b.WriteString(fmt.Sprintf("%d", len(skills)))
	b.WriteString("\">\n")

	for _, p := range skills {
		b.WriteString("  <skill id=\"")
		b.WriteString(escapeXML(p.ID))
		b.WriteString("\" name=\"")
		b.WriteString(escapeXML(p.Name))
		b.WriteString("\"><![CDATA[\n")
		b.WriteString(p.Content)
		b.WriteString("\n]]></skill>\n")
	}

	b.WriteString("</skills>")
	return b.String()
}

// displayToMarkdown generates Markdown output for displayed skills.
func displayToMarkdown(skills []Response) string {
	var b strings.Builder
	b.WriteString("# Combined Skills (")
	b.WriteString(fmt.Sprintf("%d", len(skills)))
	b.WriteString(")\n\n")
	b.WriteString("---\n\n")

	for i, p := range skills {
		b.WriteString("## ")
		b.WriteString(fmt.Sprintf("%d", i+1))
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

// displayToJSON generates JSON output for displayed skills.
func displayToJSON(skills []Response) string {
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

// GetVersions handles GET /skills/{id}/versions - returns version history.
func (h *Handlers) GetVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	versions, err := h.store.GetVersions(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine current version
	current := 1
	if len(versions) > 0 {
		current = versions[len(versions)-1].Version
	}

	response := VersionsResponse{
		SkillID:  id,
		Current:  current,
		Versions: versions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RevertToVersion handles POST /skills/{id}/revert/{version} - reverts to a version.
func (h *Handlers) RevertToVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	versionStr := vars["version"]

	// Parse version number
	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		http.Error(w, "Invalid version number", http.StatusBadRequest)
		return
	}

	skill, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	// Only allow reverts for writable folders
	if !IsWritableFolder(folder) {
		http.Error(w, "Cannot revert core skills", http.StatusForbidden)
		return
	}

	// Get the version to revert to
	targetVersion, err := h.store.GetVersionContent(id, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Save current state as a new version before reverting
	currentContent, err := h.store.GetContent(folder, skill.File)
	if err != nil {
		http.Error(w, "Failed to read current content", http.StatusInternalServerError)
		return
	}
	h.store.SaveVersion(id, folder, skill, currentContent)

	// Restore the old content
	if err := h.store.SaveContent(folder, skill.File, targetVersion.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update metadata timestamp
	now := time.Now().Format(time.RFC3339)
	skill.UpdatedAt = now

	// Save metadata
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

	// Get updated version list to determine new version number
	versions, _ := h.store.GetVersions(id)
	newVersion := 1
	if len(versions) > 0 {
		newVersion = versions[len(versions)-1].Version + 1
	}

	response := RevertResponse{
		SkillID:    id,
		RevertedTo: version,
		NewVersion: newVersion,
		RestoredAt: now,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
