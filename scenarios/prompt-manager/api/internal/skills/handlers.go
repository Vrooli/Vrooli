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

	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

// GraphInvalidator allows triggering graph index invalidation.
type GraphInvalidator interface {
	Invalidate()
}

// Handlers provides HTTP handlers for skill operations.
// Depends on interfaces (SkillStore, MetricsService) for testability.
type Handlers struct {
	store            SkillStore
	metrics          MetricsService
	aiIndexer        AISearchIndexer       // Optional: nil if AI search not available
	graphInvalidator GraphInvalidator      // Optional: nil if graph not available
	experimentStore  store.ExperimentStore // Optional: for variant-aware read
	variantStore     store.VariantStore    // Optional: for variant-aware read
	packSkillStore   store.SkillStore      // Optional: for variant-aware read (pack-based)
	identityVerifier IdentityVerifier      // Optional: verifies workflow provenance on skill reads
	readRecorder     *ReadRecorder         // Optional: records skill-read telemetry
	usageReporter    *UsageReporter        // Optional: serves the per-skill usage report
	configDir        string                // Absolute path to store directory for computing file paths
}

// storeFor binds the legacy SkillStore API to the request context when its
// implementation supports it. This keeps test-mode file routing scoped to the
// request instead of storing mutable context on a shared handler.
func (h *Handlers) storeFor(ctx context.Context) SkillStore {
	if scoped, ok := h.store.(interface {
		WithContext(context.Context) SkillStore
	}); ok {
		return scoped.WithContext(ctx)
	}
	return h.store
}

func (h *Handlers) metricsFor(ctx context.Context) MetricsService {
	if scoped, ok := h.metrics.(interface {
		WithContext(context.Context) MetricsService
	}); ok {
		return scoped.WithContext(ctx)
	}
	return h.metrics
}

func (h *Handlers) SetIdentityVerifier(verifier IdentityVerifier) { h.identityVerifier = verifier }

// NewHandlers creates a new skills handler.
// Accepts any implementation of SkillStore and MetricsService interfaces.
// configDir should be an absolute path to the store directory for computing file paths.
func NewHandlers(store SkillStore, metrics MetricsService, configDir string) *Handlers {
	return &Handlers{
		store:     store,
		metrics:   metrics,
		configDir: configDir,
	}
}

// SetAIIndexer sets the AI search indexer for async index updates.
// This is called after the aisearch.Service is initialized to avoid circular deps.
func (h *Handlers) SetAIIndexer(indexer AISearchIndexer) {
	h.aiIndexer = indexer
}

// SetReadRecorder sets the skill-read telemetry recorder. A nil recorder
// disables recording; a read is never failed because telemetry is unavailable.
func (h *Handlers) SetReadRecorder(recorder *ReadRecorder) {
	h.readRecorder = recorder
}

// SetUsageReporter sets the aggregator behind the per-skill usage report.
func (h *Handlers) SetUsageReporter(reporter *UsageReporter) {
	h.usageReporter = reporter
}

// SetExperimentStores sets the stores needed for variant-aware read.
func (h *Handlers) SetExperimentStores(experiments store.ExperimentStore, variants store.VariantStore, skills store.SkillStore) {
	h.experimentStore = experiments
	h.variantStore = variants
	h.packSkillStore = skills
}

// SetGraphInvalidator sets the graph invalidator.
func (h *Handlers) SetGraphInvalidator(inv GraphInvalidator) {
	h.graphInvalidator = inv
}

// invalidateGraph triggers graph index invalidation if available.
func (h *Handlers) invalidateGraph() {
	if h.graphInvalidator != nil {
		h.graphInvalidator.Invalidate()
	}
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

// templateVariableKeys extracts the explicit {{variable}} contract from a
// skill body. The Prompt Manager owns the durable skill store, so this generic
// check protects every write path (CLI, API, and Swarm Manager proxy) instead
// of relying on a caller-specific catalog guard.
func templateVariableKeys(content string) map[string]struct{} {
	keys := make(map[string]struct{})
	for {
		start := strings.Index(content, "{{")
		if start < 0 {
			return keys
		}
		content = content[start+2:]
		end := strings.Index(content, "}}")
		if end < 0 {
			return keys
		}
		if key := strings.TrimSpace(content[:end]); key != "" {
			keys[key] = struct{}{}
		}
		content = content[end+2:]
	}
}

func removedTemplateVariables(previous, replacement string) []string {
	before := templateVariableKeys(previous)
	after := templateVariableKeys(replacement)
	missing := make([]string, 0)
	for key := range before {
		if _, present := after[key]; !present {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}

// List handles GET /skills - returns all skills with optional filtering.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	store := h.storeFor(r.Context())
	tag := r.URL.Query().Get("tag")
	folder := r.URL.Query().Get("folder")
	modes := r.URL.Query()["modes"]
	withoutProgrammaticHome := r.URL.Query().Get("withoutProgrammaticHome") == "true"

	skills, err := store.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply domain filters
	skills = Filter(skills, FilterOptions{
		Tag:                     tag,
		Folder:                  folder,
		Modes:                   modes,
		WithoutProgrammaticHome: withoutProgrammaticHome,
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
	store := h.storeFor(r.Context())
	tag := r.URL.Query().Get("tag")

	skills, err := store.GetAll()
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
		response := h.toResponseWithContent(store, p)
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
	store := h.storeFor(r.Context())
	vars := mux.Vars(r)
	id := vars["id"]

	skill, folder, err := store.FindByID(id)
	if err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}
	// Load content
	content, err := store.GetContent(folder, skill.File)
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
	store := h.storeFor(r.Context())
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate folder
	if !IsWritableFolder(req.Folder) {
		http.Error(w, "folder must be one of: local, drafts, core", http.StatusBadRequest)
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
			_, _, err := store.FindByID(id)
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
		if _, _, err := store.FindByID(req.ID); err == nil {
			http.Error(w, "Skill with this ID already exists", http.StatusConflict)
			return
		}
	}

	now := time.Now().Format(time.RFC3339)
	filename := req.ID + ".md"

	// Create metadata entry
	metadata := Metadata{
		ID:               req.ID,
		File:             filename,
		Name:             req.Name,
		Description:      req.Description,
		Modes:            req.Modes,
		Tags:             req.Tags,
		Icon:             req.Icon,
		TargetToolID:     req.TargetToolID,
		DefaultScope:     req.DefaultScope,
		TargetDimensions: req.TargetDimensions,
		ProgrammaticHome: req.ProgrammaticHome,
		Draft:            req.Draft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Load existing skills for the folder
	skills, err := store.LoadMetadata(req.Folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add new skill
	skills = append(skills, metadata)

	// Save content file
	if err := store.SaveContent(req.Folder, filename, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save metadata
	if err := store.SaveMetadata(req.Folder, skills); err != nil {
		// Clean up content file on failure
		_ = store.DeleteContent(req.Folder, filename)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := h.toResponse(metadata)
	response.Content = req.Content
	response.Folder = req.Folder

	// Trigger async AI index update
	h.triggerIndexAsync(req.ID)
	h.invalidateGraph()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// Update handles PUT /skills/{id} - updates an existing skill.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	store := h.storeFor(r.Context())
	metrics := h.metricsFor(r.Context())
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	skill, folder, err := store.FindByID(id)
	if err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}
	if folder == "vendor" {
		overlayPath := filepath.Join("vendor", id, "overlays")
		if provider, ok := store.(interface{ ImportedSkillOverlayPath(string) string }); ok {
			overlayPath = provider.ImportedSkillOverlayPath(id)
		}
		http.Error(w, "Cannot edit vendored skill in place; write an overlay under "+overlayPath, http.StatusForbidden)
		return
	}
	if req.Content != nil {
		current, err := store.GetContent(folder, skill.File)
		if err != nil {
			http.Error(w, "Failed to read existing skill content", http.StatusInternalServerError)
			return
		}
		if missing := removedTemplateVariables(current, *req.Content); len(missing) > 0 {
			http.Error(w, "content removes existing template variables: "+strings.Join(missing, ", "), http.StatusBadRequest)
			return
		}
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
			renamedSkill, err := store.Rename(id, newID)
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
			if oldMetrics, err := metrics.Get(id); err == nil && oldMetrics != nil {
				// Copy usage data to new ID, then delete old
				for i := 0; i < oldMetrics.UsageCount; i++ {
					_, _, _ = metrics.RecordUsage(newID)
				}
				if oldMetrics.EffectivenessRating != nil {
					_ = metrics.SetRating(newID, *oldMetrics.EffectivenessRating, oldMetrics.Notes)
				}
				_ = metrics.Delete(id)
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
	if req.DefaultScope != nil {
		skill.DefaultScope = *req.DefaultScope
	}
	if req.TargetDimensions != nil {
		skill.TargetDimensions = req.TargetDimensions
	}
	if req.ClearProgrammaticHome {
		skill.ProgrammaticHome = nil
	} else if req.ProgrammaticHome != nil {
		skill.ProgrammaticHome = req.ProgrammaticHome
	}

	skill.UpdatedAt = time.Now().Format(time.RFC3339)

	// Handle folder move
	if targetFolder != folder {
		// Read current content
		currentContent, err := store.GetContent(folder, oldFile)
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
		if err := store.SaveContent(targetFolder, skill.File, contentToSave); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Remove from old folder's metadata
		oldSkills, err := store.LoadMetadata(folder)
		if err != nil {
			// Rollback: delete from new folder
			_ = store.DeleteContent(targetFolder, skill.File)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var filteredOld []Metadata
		for _, p := range oldSkills {
			if p.ID != id {
				filteredOld = append(filteredOld, p)
			}
		}
		if err := store.SaveMetadata(folder, filteredOld); err != nil {
			_ = store.DeleteContent(targetFolder, skill.File)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add to new folder's metadata
		newSkills, err := store.LoadMetadata(targetFolder)
		if err != nil {
			// Rollback is complex here, but proceed - metadata was already removed
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		newSkills = append(newSkills, *skill)
		if err := store.SaveMetadata(targetFolder, newSkills); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Move version history
		h.moveVersionHistory(store, id, folder, targetFolder)

		// Delete old content file
		_ = store.DeleteContent(folder, oldFile)
	} else {
		// Same folder - handle file rename or content update
		if skill.File != oldFile {
			// Read old content
			oldContent, err := store.GetContent(folder, oldFile)
			if err != nil {
				http.Error(w, "Failed to read existing content for rename", http.StatusInternalServerError)
				return
			}
			// Write to new file (use req.Content if provided, otherwise old content)
			newContent := oldContent
			if req.Content != nil {
				newContent = *req.Content
			}
			if err := store.SaveContent(folder, skill.File, newContent); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Delete old file
			_ = store.DeleteContent(folder, oldFile)
		} else if req.Content != nil {
			// No rename, just update content
			if err := store.SaveContent(folder, skill.File, *req.Content); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Load all skills and update the matching one
		skills, err := store.LoadMetadata(folder)
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

		if err := store.SaveMetadata(folder, skills); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	response := h.toResponse(*skill)
	response.Folder = targetFolder

	// Load content for response
	if content, err := store.GetContent(targetFolder, skill.File); err == nil {
		response.Content = content
	}

	// Trigger async AI index update
	h.triggerIndexAsync(id)
	h.invalidateGraph()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// moveVersionHistory moves version history from one folder to another.
func (h *Handlers) moveVersionHistory(store SkillStore, skillID, fromFolder, toFolder string) {
	// Load versions from source folder
	fromVersions, err := store.LoadVersions(fromFolder)
	if err != nil {
		return
	}

	vf, ok := fromVersions[skillID]
	if !ok || len(vf.Versions) == 0 {
		return // No version history to move
	}

	// Load versions for target folder
	toVersions, err := store.LoadVersions(toFolder)
	if err != nil {
		toVersions = make(map[string]*VersionFile)
	}

	// Move the version file entry
	toVersions[skillID] = vf
	delete(fromVersions, skillID)

	// Save both
	_ = store.SaveVersions(toFolder, toVersions)
	_ = store.SaveVersions(fromFolder, fromVersions)
}

// Delete handles DELETE /skills/{id} - deletes a skill.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	store := h.storeFor(r.Context())
	metrics := h.metricsFor(r.Context())
	vars := mux.Vars(r)
	id := vars["id"]

	skill, folder, err := store.FindByID(id)
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
	skills, err := store.LoadMetadata(folder)
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

	if err := store.SaveMetadata(folder, filtered); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete content file (ignore error - best effort)
	_ = store.DeleteContent(folder, skill.File)

	// Delete metrics from database
	_ = metrics.Delete(id)

	// Trigger async AI index delete
	h.triggerDeleteAsync(id)
	h.invalidateGraph()

	w.WriteHeader(http.StatusNoContent)
}

// RecordUsage handles POST /skills/{id}/use - records skill usage.
func (h *Handlers) RecordUsage(w http.ResponseWriter, r *http.Request) {
	store := h.storeFor(r.Context())
	metrics := h.metricsFor(r.Context())
	vars := mux.Vars(r)
	id := vars["id"]

	// Verify skill exists
	if _, _, err := store.FindByID(id); err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	usageCount, lastUsed, err := metrics.RecordUsage(id)
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
	store := h.storeFor(r.Context())
	metrics := h.metricsFor(r.Context())
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
	if _, _, err := store.FindByID(id); err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	if err := metrics.SetRating(id, req.Rating, req.Notes); err != nil {
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
		ID:               p.ID,
		Name:             p.Name,
		Description:      p.Description,
		Modes:            p.Modes,
		Tags:             p.Tags,
		Icon:             p.Icon,
		TargetToolID:     p.TargetToolID,
		DefaultScope:     p.DefaultScope,
		TargetDimensions: p.TargetDimensions,
		ProgrammaticHome: p.ProgrammaticHome,
		Draft:            p.Draft,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		Revision:         p.Revision,
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
	if h.configDir != "" && folder != "" {
		skillDir := filepath.Join(h.configDir, "skills", "packs", folder, p.ID)
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

func (h *Handlers) toResponseWithContent(store SkillStore, p Metadata) Response {
	response := h.toResponse(p)

	// Extract folder and filename
	parts := strings.SplitN(p.File, "/", 2)
	if len(parts) == 2 {
		content, err := store.GetContent(parts[0], parts[1])
		if err == nil {
			content = StripFrontmatter(content)
			response.Content = content
			response.Variables = ExtractVariables(content)
		}
	}

	return response
}

// GetVersions handles GET /skills/{id}/versions - returns version history.
func (h *Handlers) GetVersions(w http.ResponseWriter, r *http.Request) {
	store := h.storeFor(r.Context())
	vars := mux.Vars(r)
	id := vars["id"]

	versions, err := store.GetVersions(id)
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
	store := h.storeFor(r.Context())
	vars := mux.Vars(r)
	id := vars["id"]
	versionStr := vars["version"]

	// Parse version number
	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		http.Error(w, "Invalid version number", http.StatusBadRequest)
		return
	}

	skill, folder, err := store.FindByID(id)
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
	targetVersion, err := store.GetVersionContent(id, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Save current state as a new version before reverting
	currentContent, err := store.GetContent(folder, skill.File)
	if err != nil {
		http.Error(w, "Failed to read current content", http.StatusInternalServerError)
		return
	}
	_ = store.SaveVersion(id, folder, skill, currentContent)

	// Restore the old content
	if err := store.SaveContent(folder, skill.File, targetVersion.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update metadata timestamp
	now := time.Now().Format(time.RFC3339)
	skill.UpdatedAt = now

	// Save metadata
	skills, err := store.LoadMetadata(folder)
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
	if err := store.SaveMetadata(folder, skills); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get updated version list to determine new version number
	versions, _ := store.GetVersions(id)
	newVersion := 1
	if len(versions) > 0 {
		newVersion = versions[len(versions)-1].Version + 1
	}

	h.invalidateGraph()

	response := RevertResponse{
		SkillID:    id,
		RevertedTo: version,
		NewVersion: newVersion,
		RestoredAt: now,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
