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
	"strconv"
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

// ListSkills returns enriched skill records without crossing the HTTP
// compatibility boundary. Connect and the legacy REST endpoint share this
// implementation.
func (h *Handlers) ListSkills(ctx context.Context, opts FilterOptions) ([]Response, error) {
	store := h.storeFor(ctx)
	items, err := store.GetAll()
	if err != nil {
		return nil, err
	}
	items = Filter(items, opts)
	responses := make([]Response, 0, len(items))
	for _, item := range items {
		responses = append(responses, h.toResponse(item))
	}
	return responses, nil
}

// GetSkill returns one enriched skill record without crossing the HTTP
// compatibility boundary.
func (h *Handlers) GetSkill(ctx context.Context, id string) (Response, error) {
	store := h.storeFor(ctx)
	skill, folder, err := store.FindByID(id)
	if err != nil {
		return Response{}, fmt.Errorf("Skill not found")
	}
	content, err := store.GetContent(folder, skill.File)
	if err != nil {
		return Response{}, fmt.Errorf("Failed to load skill content")
	}
	response := h.toResponse(*skill)
	response.Content = content
	response.Folder = folder
	response.Variables = ExtractVariables(content)
	return response, nil
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
	tag := r.URL.Query().Get("tag")
	folder := r.URL.Query().Get("folder")
	modes := r.URL.Query()["modes"]
	withoutProgrammaticHome := r.URL.Query().Get("withoutProgrammaticHome") == "true"

	skills, err := h.ListSkills(r.Context(), FilterOptions{
		Tag: tag, Folder: folder, Modes: modes, WithoutProgrammaticHome: withoutProgrammaticHome,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(skills)
}

// SyncSkills builds the deterministic, content-bearing catalog used by
// clients that need change detection. It is transport-neutral.
func (h *Handlers) SyncSkills(ctx context.Context, tag string) (SyncResponse, error) {
	store := h.storeFor(ctx)
	items, err := store.GetAll()
	if err != nil {
		return SyncResponse{}, err
	}
	items = Filter(items, FilterOptions{Tag: tag})
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	responses := make([]Response, 0, len(items))
	var lastUpdated time.Time
	for _, item := range items {
		responses = append(responses, h.toResponseWithContent(store, item))
		if updated, parseErr := time.Parse(time.RFC3339, item.UpdatedAt); parseErr == nil && updated.After(lastUpdated) {
			lastUpdated = updated
		}
	}
	hashData, _ := json.Marshal(responses)
	hash := sha256.Sum256(hashData)
	return SyncResponse{Skills: responses, LastUpdated: lastUpdated.Format(time.RFC3339), Hash: hex.EncodeToString(hash[:])}, nil
}

// Sync handles GET /skills/sync - returns skills with content for syncing.
func (h *Handlers) Sync(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")
	response, err := h.SyncSkills(r.Context(), tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// Get handles GET /skills/{id} - returns a single skill.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	response, err := h.GetSkill(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "Skill not found" {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// Create handles POST /skills - creates a new skill.
func (h *Handlers) CreateSkill(ctx context.Context, req CreateRequest) (Response, error) {
	store := h.storeFor(ctx)
	// Validate folder
	if !IsWritableFolder(req.Folder) {
		return Response{}, fmt.Errorf("folder must be one of: local, drafts, core")
	}

	// Validate required fields
	if req.Name == "" || req.Content == "" {
		return Response{}, fmt.Errorf("Name and content are required")
	}

	// Generate unique ID if not provided
	if req.ID == "" {
		idExists := func(id string) bool {
			_, _, err := store.FindByID(id)
			return err == nil
		}

		uniqueID, err := GenerateUniqueID(req.Name, idExists)
		if err != nil {
			return Response{}, err
		}
		req.ID = uniqueID
	} else {
		// User provided explicit ID - check for conflict
		if _, _, err := store.FindByID(req.ID); err == nil {
			return Response{}, fmt.Errorf("Skill with this ID already exists")
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
		return Response{}, err
	}

	// Add new skill
	skills = append(skills, metadata)

	// Save content file
	if err := store.SaveContent(req.Folder, filename, req.Content); err != nil {
		return Response{}, err
	}

	// Save metadata
	if err := store.SaveMetadata(req.Folder, skills); err != nil {
		// Clean up content file on failure
		_ = store.DeleteContent(req.Folder, filename)
		return Response{}, err
	}

	response := h.toResponse(metadata)
	response.Content = req.Content
	response.Folder = req.Folder

	// Trigger async AI index update
	h.triggerIndexAsync(req.ID)
	h.invalidateGraph()
	return response, nil
}

// Create handles POST /skills - creates a new skill.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	response, err := h.CreateSkill(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "folder must be one of: local, drafts, core", "Name and content are required":
			status = http.StatusBadRequest
		case "Skill with this ID already exists":
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// UpdateSkill updates an existing skill without crossing the HTTP compatibility boundary.
// REST and Connect adapters share this transaction-preserving implementation.
func (h *Handlers) UpdateSkill(ctx context.Context, id string, req UpdateRequest) (Response, error) {
	store := h.storeFor(ctx)
	metrics := h.metricsFor(ctx)
	skill, folder, err := store.FindByID(id)
	if err != nil {
		return Response{}, fmt.Errorf("Skill not found")
	}
	if folder == "vendor" {
		overlayPath := filepath.Join("vendor", id, "overlays")
		if provider, ok := store.(interface{ ImportedSkillOverlayPath(string) string }); ok {
			overlayPath = provider.ImportedSkillOverlayPath(id)
		}
		return Response{}, fmt.Errorf("Cannot edit vendored skill in place; write an overlay under %s", overlayPath)
	}
	if req.Content != nil {
		current, err := store.GetContent(folder, skill.File)
		if err != nil {
			return Response{}, fmt.Errorf("Failed to read existing skill content")
		}
		if missing := removedTemplateVariables(current, *req.Content); len(missing) > 0 {
			return Response{}, fmt.Errorf("content removes existing template variables: %s", strings.Join(missing, ", "))
		}
	}

	// Only allow updates to local/drafts skills
	if !IsWritableFolder(folder) {
		return Response{}, fmt.Errorf("Cannot update core skills")
	}

	// Determine target folder (may be moving to a new folder)
	targetFolder := folder
	if req.Folder != nil && *req.Folder != "" && *req.Folder != folder {
		if !IsWritableFolder(*req.Folder) {
			return Response{}, fmt.Errorf("Cannot move skill to non-writable folder")
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
					return Response{}, fmt.Errorf("%s", errStr)
				}
				if strings.Contains(errStr, "invalid skill ID format") {
					return Response{}, fmt.Errorf("%s", errStr)
				}
				return Response{}, fmt.Errorf("%s", errStr)
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
			return Response{}, fmt.Errorf("Failed to read existing content for move")
		}

		// Use new content if provided, otherwise use current
		contentToSave := currentContent
		if req.Content != nil {
			contentToSave = *req.Content
		}

		// Save content to new folder
		if err := store.SaveContent(targetFolder, skill.File, contentToSave); err != nil {
			return Response{}, err
		}

		// Remove from old folder's metadata
		oldSkills, err := store.LoadMetadata(folder)
		if err != nil {
			// Rollback: delete from new folder
			_ = store.DeleteContent(targetFolder, skill.File)
			return Response{}, err
		}
		var filteredOld []Metadata
		for _, p := range oldSkills {
			if p.ID != id {
				filteredOld = append(filteredOld, p)
			}
		}
		if err := store.SaveMetadata(folder, filteredOld); err != nil {
			_ = store.DeleteContent(targetFolder, skill.File)
			return Response{}, err
		}

		// Add to new folder's metadata
		newSkills, err := store.LoadMetadata(targetFolder)
		if err != nil {
			// Rollback is complex here, but proceed - metadata was already removed
			return Response{}, err
		}
		newSkills = append(newSkills, *skill)
		if err := store.SaveMetadata(targetFolder, newSkills); err != nil {
			return Response{}, err
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
				return Response{}, fmt.Errorf("Failed to read existing content for rename")
			}
			// Write to new file (use req.Content if provided, otherwise old content)
			newContent := oldContent
			if req.Content != nil {
				newContent = *req.Content
			}
			if err := store.SaveContent(folder, skill.File, newContent); err != nil {
				return Response{}, err
			}
			// Delete old file
			_ = store.DeleteContent(folder, oldFile)
		} else if req.Content != nil {
			// No rename, just update content
			if err := store.SaveContent(folder, skill.File, *req.Content); err != nil {
				return Response{}, err
			}
		}

		// Load all skills and update the matching one
		skills, err := store.LoadMetadata(folder)
		if err != nil {
			return Response{}, err
		}

		for i, p := range skills {
			if p.ID == id {
				skills[i] = *skill
				break
			}
		}

		if err := store.SaveMetadata(folder, skills); err != nil {
			return Response{}, err
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

	return response, nil
}

// Update handles PUT /skills/{id} - updates an existing skill.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := h.UpdateSkill(r.Context(), mux.Vars(r)["id"], req)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case message == "Skill not found":
			status = http.StatusNotFound
		case strings.HasPrefix(message, "Cannot edit vendored skill") || message == "Cannot update core skills":
			status = http.StatusForbidden
		case message == "Cannot move skill to non-writable folder" || strings.HasPrefix(message, "content removes existing template variables:"):
			status = http.StatusBadRequest
		case strings.Contains(message, "already exists"):
			status = http.StatusConflict
		case strings.Contains(message, "invalid skill ID format"):
			status = http.StatusBadRequest
		}
		http.Error(w, message, status)
		return
	}

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

// DeleteSkill removes a writable skill and its associated metrics.
func (h *Handlers) DeleteSkill(ctx context.Context, id string) error {
	store := h.storeFor(ctx)
	metrics := h.metricsFor(ctx)
	skill, folder, err := store.FindByID(id)
	if err != nil {
		return fmt.Errorf("Skill not found")
	}

	// Only allow deletes from local/drafts
	if !IsWritableFolder(folder) {
		return fmt.Errorf("Cannot delete core skills")
	}

	// Remove from metadata
	skills, err := store.LoadMetadata(folder)
	if err != nil {
		return err
	}

	var filtered []Metadata
	for _, p := range skills {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}

	if err := store.SaveMetadata(folder, filtered); err != nil {
		return err
	}

	// Delete content file (ignore error - best effort)
	_ = store.DeleteContent(folder, skill.File)

	// Delete metrics from database
	_ = metrics.Delete(id)

	// Trigger async AI index delete
	h.triggerDeleteAsync(id)
	h.invalidateGraph()
	return nil
}

// Delete handles DELETE /skills/{id} - deletes a skill.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.DeleteSkill(r.Context(), vars["id"]); err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "Skill not found":
			status = http.StatusNotFound
		case "Cannot delete core skills":
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RecordSkillUsage records one use of a skill and returns the updated counters.
func (h *Handlers) RecordSkillUsage(ctx context.Context, id string) (int, time.Time, error) {
	store := h.storeFor(ctx)
	metrics := h.metricsFor(ctx)

	// Verify skill exists
	if _, _, err := store.FindByID(id); err != nil {
		return 0, time.Time{}, fmt.Errorf("Skill not found")
	}

	usageCount, lastUsed, err := metrics.RecordUsage(id)
	if err != nil {
		return 0, time.Time{}, err
	}
	return usageCount, lastUsed, nil
}

// RecordUsage handles POST /skills/{id}/use - records skill usage.
func (h *Handlers) RecordUsage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	usageCount, lastUsed, err := h.RecordSkillUsage(r.Context(), vars["id"])
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "Skill not found" {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "usage recorded",
		"usageCount": usageCount,
		"lastUsed":   lastUsed,
	})
}

// SetSkillRating updates a skill's effectiveness rating.
func (h *Handlers) SetSkillRating(ctx context.Context, id string, rating int, notes *string) error {
	store := h.storeFor(ctx)
	metrics := h.metricsFor(ctx)

	if rating < 1 || rating > 5 {
		return fmt.Errorf("Rating must be between 1 and 5")
	}

	// Verify skill exists
	if _, _, err := store.FindByID(id); err != nil {
		return fmt.Errorf("Skill not found")
	}

	return metrics.SetRating(id, rating, notes)
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

	if err := h.SetSkillRating(r.Context(), id, req.Rating, req.Notes); err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "Rating must be between 1 and 5":
			status = http.StatusBadRequest
		case "Skill not found":
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
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

// ListSkillVersions returns version history for a skill.
func (h *Handlers) ListSkillVersions(ctx context.Context, id string) (VersionsResponse, error) {
	store := h.storeFor(ctx)
	versions, err := store.GetVersions(id)
	if err != nil {
		return VersionsResponse{}, err
	}

	// Determine current version
	current := 1
	if len(versions) > 0 {
		current = versions[len(versions)-1].Version
	}

	return VersionsResponse{
		SkillID:  id,
		Current:  current,
		Versions: versions,
	}, nil
}

// GetVersions handles GET /skills/{id}/versions - returns version history.
func (h *Handlers) GetVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	response, err := h.ListSkillVersions(r.Context(), vars["id"])
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// RevertSkillVersion restores a skill version and creates a new current version.
func (h *Handlers) RevertSkillVersion(ctx context.Context, id string, version int) (RevertResponse, error) {
	store := h.storeFor(ctx)
	skill, folder, err := store.FindByID(id)
	if err != nil {
		return RevertResponse{}, fmt.Errorf("Skill not found")
	}

	// Only allow reverts for writable folders
	if !IsWritableFolder(folder) {
		return RevertResponse{}, fmt.Errorf("Cannot revert core skills")
	}

	// Get the version to revert to
	targetVersion, err := store.GetVersionContent(id, version)
	if err != nil {
		return RevertResponse{}, err
	}

	// Save current state as a new version before reverting
	currentContent, err := store.GetContent(folder, skill.File)
	if err != nil {
		return RevertResponse{}, fmt.Errorf("Failed to read current content")
	}
	_ = store.SaveVersion(id, folder, skill, currentContent)

	// Restore the old content
	if err := store.SaveContent(folder, skill.File, targetVersion.Content); err != nil {
		return RevertResponse{}, err
	}

	// Update metadata timestamp
	now := time.Now().Format(time.RFC3339)
	skill.UpdatedAt = now

	// Save metadata
	skills, err := store.LoadMetadata(folder)
	if err != nil {
		return RevertResponse{}, err
	}
	for i, p := range skills {
		if p.ID == id {
			skills[i] = *skill
			break
		}
	}
	if err := store.SaveMetadata(folder, skills); err != nil {
		return RevertResponse{}, err
	}

	// Get updated version list to determine new version number
	versions, _ := store.GetVersions(id)
	newVersion := 1
	if len(versions) > 0 {
		newVersion = versions[len(versions)-1].Version + 1
	}

	h.invalidateGraph()

	return RevertResponse{
		SkillID:    id,
		RevertedTo: version,
		NewVersion: newVersion,
		RestoredAt: now,
	}, nil
}

// RevertToVersion handles POST /skills/{id}/revert/{version} - reverts to a version.
func (h *Handlers) RevertToVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version, err := strconv.Atoi(vars["version"])
	if err != nil {
		http.Error(w, "Invalid version number", http.StatusBadRequest)
		return
	}
	response, err := h.RevertSkillVersion(r.Context(), vars["id"], version)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "Skill not found":
			status = http.StatusNotFound
		case "Cannot revert core skills":
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
