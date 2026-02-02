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
	"path/filepath"
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
	storeDir  string          // Absolute path to store directory for computing file paths
}

// NewHandlers creates a new skills handler.
// Accepts any implementation of SkillStore and MetricsService interfaces.
// storeDir should be an absolute path to the store directory for computing file paths.
func NewHandlers(store SkillStore, metrics MetricsService, storeDir string) *Handlers {
	return &Handlers{
		store:    store,
		metrics:  metrics,
		storeDir: storeDir,
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
	_ = json.NewEncoder(w).Encode(responses)
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
	_ = json.NewEncoder(w).Encode(syncResponse)
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
	response.Variables = ExtractVariables(content)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
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
		_ = h.store.DeleteContent(req.Folder, filename)
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
	_ = json.NewEncoder(w).Encode(response)
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

	// Determine target folder (may be moving to a new folder)
	targetFolder := folder
	if req.Folder != nil && *req.Folder != "" && *req.Folder != folder {
		if !IsWritableFolder(*req.Folder) {
			http.Error(w, "Cannot move skill to non-writable folder", http.StatusBadRequest)
			return
		}
		targetFolder = *req.Folder
	}

	// Track old filename for potential rename
	oldFile := skill.File

	// Check if this is a skill ID rename (file field change means ID change)
	if req.File != nil && *req.File != "" {
		// Validate new filename and extract ID
		newFile := *req.File
		if !strings.HasSuffix(newFile, ".md") {
			newFile = newFile + ".md"
		}
		newID := strings.TrimSuffix(filepath.Base(newFile), ".md")

		// If the ID is changing, perform a rename operation
		if newID != id {
			renamedSkill, err := h.store.Rename(id, newID)
			if err != nil {
				// Check for known error types
				errStr := err.Error()
				if strings.Contains(errStr, "already exists") {
					http.Error(w, errStr, http.StatusConflict)
					return
				}
				if strings.Contains(errStr, "invalid skill ID format") {
					http.Error(w, errStr, http.StatusBadRequest)
					return
				}
				http.Error(w, errStr, http.StatusInternalServerError)
				return
			}

			// Update AI index: delete old, index new
			h.triggerDeleteAsync(id)
			h.triggerIndexAsync(newID)

			// Migrate metrics (best effort)
			if oldMetrics, err := h.metrics.Get(id); err == nil && oldMetrics != nil {
				// Copy usage data to new ID, then delete old
				for i := 0; i < oldMetrics.UsageCount; i++ {
					_, _, _ = h.metrics.RecordUsage(newID)
				}
				if oldMetrics.EffectivenessRating != nil {
					_ = h.metrics.SetRating(newID, *oldMetrics.EffectivenessRating, oldMetrics.Notes)
				}
				_ = h.metrics.Delete(id)
			}

			// Continue with the renamed skill and new ID
			skill = renamedSkill
			id = newID
			folder = targetFolder // After rename, folder stays same
		} else {
			// Just update the file field (no actual ID change)
			skill.File = newFile
		}
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

	// Handle folder move
	if targetFolder != folder {
		// Read current content
		currentContent, err := h.store.GetContent(folder, oldFile)
		if err != nil {
			http.Error(w, "Failed to read existing content for move", http.StatusInternalServerError)
			return
		}

		// Use new content if provided, otherwise use current
		contentToSave := currentContent
		if req.Content != nil {
			contentToSave = *req.Content
		}

		// Save content to new folder
		if err := h.store.SaveContent(targetFolder, skill.File, contentToSave); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Remove from old folder's metadata
		oldSkills, err := h.store.LoadMetadata(folder)
		if err != nil {
			// Rollback: delete from new folder
			_ = h.store.DeleteContent(targetFolder, skill.File)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var filteredOld []Metadata
		for _, p := range oldSkills {
			if p.ID != id {
				filteredOld = append(filteredOld, p)
			}
		}
		if err := h.store.SaveMetadata(folder, filteredOld); err != nil {
			_ = h.store.DeleteContent(targetFolder, skill.File)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add to new folder's metadata
		newSkills, err := h.store.LoadMetadata(targetFolder)
		if err != nil {
			// Rollback is complex here, but proceed - metadata was already removed
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		newSkills = append(newSkills, *skill)
		if err := h.store.SaveMetadata(targetFolder, newSkills); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Move version history
		h.moveVersionHistory(id, folder, targetFolder)

		// Delete old content file
		_ = h.store.DeleteContent(folder, oldFile)
	} else {
		// Same folder - handle file rename or content update
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
			_ = h.store.DeleteContent(folder, oldFile)
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
	}

	response := h.toResponse(*skill)
	response.Folder = targetFolder

	// Load content for response
	if content, err := h.store.GetContent(targetFolder, skill.File); err == nil {
		response.Content = content
	}

	// Trigger async AI index update
	h.triggerIndexAsync(id)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// moveVersionHistory moves version history from one folder to another.
func (h *Handlers) moveVersionHistory(skillID, fromFolder, toFolder string) {
	// Load versions from source folder
	fromVersions, err := h.store.LoadVersions(fromFolder)
	if err != nil {
		return
	}

	vf, ok := fromVersions[skillID]
	if !ok || len(vf.Versions) == 0 {
		return // No version history to move
	}

	// Load versions for target folder
	toVersions, err := h.store.LoadVersions(toFolder)
	if err != nil {
		toVersions = make(map[string]*VersionFile)
	}

	// Move the version file entry
	toVersions[skillID] = vf
	delete(fromVersions, skillID)

	// Save both
	_ = h.store.SaveVersions(toFolder, toVersions)
	_ = h.store.SaveVersions(fromFolder, fromVersions)
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
	_ = h.store.DeleteContent(folder, skill.File)

	// Delete metrics from database
	_ = h.metrics.Delete(id)

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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
	var folder string
	if parts := strings.SplitN(p.File, "/", 2); len(parts) == 2 {
		folder = parts[0]
		response.Folder = folder
		response.File = parts[1]
	} else {
		// No folder prefix - just the filename
		response.File = p.File
	}

	// Compute absolute paths to skill directory and content file
	// Storage structure: store/skills/packs/{pack}/{skillId}/SKILL.md
	if h.storeDir != "" && folder != "" {
		skillDir := filepath.Join(h.storeDir, "skills", "packs", folder, p.ID)
		response.SkillDir = skillDir
		response.ContentPath = filepath.Join(skillDir, "SKILL.md")
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
			response.Variables = ExtractVariables(content)
		}
	}

	return response
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
	_ = json.NewEncoder(w).Encode(response)
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
	_ = h.store.SaveVersion(id, folder, skill, currentContent)

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
	_ = json.NewEncoder(w).Encode(response)
}
