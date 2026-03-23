// Archive HTTP handlers for operational targets, requirements modules,
// and batch review operations within a backlog item's archive folder.
package backlog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"swarm-manager/internal/httputil"
)

// GetArchiveTargets returns operational targets and requirements parsed from a backlog item's archive.
func (h *Handler) GetArchiveTargets(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "archive targets")
	if !ok {
		return
	}

	archiveDir := filepath.Join(h.store.ItemDir(kind, name), "archive")
	info, err := os.Stat(archiveDir)
	if err != nil || !info.IsDir() {
		_ = httputil.JSON(w, map[string]any{
			"targets":      []any{},
			"requirements": []any{},
			"has_archive":  false,
		})
		return
	}

	targets, err := ParseArchiveTargets(archiveDir)
	if err != nil {
		targets = []ArchiveTarget{}
	}

	// Merge review state into targets.
	itemDir := h.store.ItemDir(kind, name)
	reviewState, _ := ReadReviewState(itemDir)
	if len(reviewState) > 0 {
		// Build target ID set for pruning.
		targetIDs := make(map[string]bool, len(targets))
		for i := range targets {
			targetIDs[targets[i].ID] = true
		}
		PruneReviewState(reviewState, targetIDs)
	}

	// Build response targets with review fields merged in.
	type targetWithReview struct {
		ArchiveTarget
		ReviewedAt    string `json:"reviewed_at,omitempty"`
		ReviewComment string `json:"review_comment,omitempty"`
		ReviewStatus  string `json:"review_status,omitempty"`
	}
	respTargets := make([]targetWithReview, len(targets))
	for i, t := range targets {
		respTargets[i] = targetWithReview{ArchiveTarget: t}
		if rs, ok := reviewState[t.ID]; ok {
			respTargets[i].ReviewedAt = rs.ReviewedAt
			respTargets[i].ReviewComment = rs.ReviewComment
			respTargets[i].ReviewStatus = rs.ReviewStatus
		}
	}

	requirements, err := ParseArchiveRequirements(archiveDir)
	if err != nil {
		requirements = []ArchiveRequirementGroup{}
	}

	_ = httputil.JSON(w, map[string]any{
		"targets":      respTargets,
		"requirements": requirements,
		"has_archive":  true,
	})
}

// CreateTargetHandler creates a new operational target in PRD.md.
func (h *Handler) CreateTargetHandler(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "create target")
	if !ok {
		return
	}

	var body ArchiveTarget
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "[backlog] create target", "invalid JSON body")
		return
	}
	if body.Title == "" {
		httputil.BadRequest(w, "[backlog] create target", "title is required")
		return
	}

	dir := h.store.ItemDir(kind, name)
	if err := CreateTarget(dir, body); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			httputil.BadRequest(w, "[backlog] create target", err.Error())
			return
		}
		httputil.InternalError(w, "[backlog] create target", err.Error())
		return
	}

	_ = httputil.JSONWithStatus(w, http.StatusCreated, map[string]any{"ok": true})
}

// UpdateTargetHandler updates an existing operational target in PRD.md.
func (h *Handler) UpdateTargetHandler(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "update target")
	if !ok {
		return
	}
	targetID := mux.Vars(r)["targetId"]
	if targetID == "" {
		httputil.BadRequest(w, "[backlog] update target", "targetId is required")
		return
	}

	var body ArchiveTarget
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "[backlog] update target", "invalid JSON body")
		return
	}

	dir := h.store.ItemDir(kind, name)
	if err := UpdateTarget(dir, targetID, body); err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.NotFound(w, "[backlog] update target", err.Error())
			return
		}
		httputil.InternalError(w, "[backlog] update target", err.Error())
		return
	}

	_ = httputil.JSON(w, map[string]any{"ok": true})
}

// DeleteTargetHandler removes an operational target from PRD.md.
func (h *Handler) DeleteTargetHandler(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "delete target")
	if !ok {
		return
	}
	targetID := mux.Vars(r)["targetId"]
	if targetID == "" {
		httputil.BadRequest(w, "[backlog] delete target", "targetId is required")
		return
	}

	dir := h.store.ItemDir(kind, name)
	if err := DeleteTarget(dir, targetID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.NotFound(w, "[backlog] update target", err.Error())
			return
		}
		httputil.InternalError(w, "[backlog] delete target", err.Error())
		return
	}

	_ = httputil.JSON(w, map[string]any{"ok": true})
}

// UpdateModuleRequirementsHandler replaces the requirements array in a module.
func (h *Handler) UpdateModuleRequirementsHandler(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "update module requirements")
	if !ok {
		return
	}
	moduleID := mux.Vars(r)["moduleId"]
	if moduleID == "" {
		httputil.BadRequest(w, "[backlog] update module requirements", "moduleId is required")
		return
	}

	var body struct {
		Requirements []json.RawMessage `json:"requirements"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "[backlog] update module requirements", "invalid JSON body")
		return
	}

	dir := h.store.ItemDir(kind, name)
	if err := WriteModuleRequirements(dir, moduleID, body.Requirements); err != nil {
		httputil.InternalError(w, "[backlog] update module requirements", err.Error())
		return
	}

	_ = httputil.JSON(w, map[string]any{"ok": true})
}

// CreateModuleHandler creates a new requirements module.
func (h *Handler) CreateModuleHandler(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "create module")
	if !ok {
		return
	}

	var body struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Position    int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "[backlog] create module", "invalid JSON body")
		return
	}
	if body.ID == "" {
		httputil.BadRequest(w, "[backlog] create module", "id is required")
		return
	}

	dir := h.store.ItemDir(kind, name)
	input := CreateModuleInput{
		ID:          body.ID,
		Title:       body.Title,
		Description: body.Description,
	}
	if err := CreateModule(dir, input, body.Position); err != nil {
		httputil.InternalError(w, "[backlog] create module", err.Error())
		return
	}

	_ = httputil.JSONWithStatus(w, http.StatusCreated, map[string]any{"ok": true, "id": body.ID})
}

// UpdateModuleMetaHandler updates a module's title and description.
func (h *Handler) UpdateModuleMetaHandler(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "update module meta")
	if !ok {
		return
	}
	moduleID := mux.Vars(r)["moduleId"]
	if moduleID == "" {
		httputil.BadRequest(w, "[backlog] update module meta", "moduleId is required")
		return
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "[backlog] update module meta", "invalid JSON body")
		return
	}

	dir := h.store.ItemDir(kind, name)
	if err := UpdateModuleMeta(dir, moduleID, body.Title, body.Description); err != nil {
		httputil.InternalError(w, "[backlog] update module meta", err.Error())
		return
	}

	_ = httputil.JSON(w, map[string]any{"ok": true})
}

// DeleteModuleHandler removes a requirements module.
func (h *Handler) DeleteModuleHandler(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "delete module")
	if !ok {
		return
	}
	moduleID := mux.Vars(r)["moduleId"]
	if moduleID == "" {
		httputil.BadRequest(w, "[backlog] delete module", "moduleId is required")
		return
	}

	dir := h.store.ItemDir(kind, name)
	if err := DeleteModule(dir, moduleID); err != nil {
		httputil.InternalError(w, "[backlog] delete module", err.Error())
		return
	}

	_ = httputil.JSON(w, map[string]any{"ok": true})
}

// BatchReviewHandler applies review status updates to targets and/or requirements in batch.
func (h *Handler) BatchReviewHandler(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "batch review")
	if !ok {
		return
	}

	var body struct {
		Items []struct {
			ID            string `json:"id"`
			Type          string `json:"type"`          // "target" or "requirement"
			ModuleID      string `json:"module_id"`     // required for requirements
			ReviewStatus  string `json:"review_status"` // "approved", "flagged", "unreviewed"
			ReviewComment string `json:"review_comment"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "[backlog] batch review", "invalid JSON body")
		return
	}
	if len(body.Items) == 0 {
		httputil.BadRequest(w, "[backlog] batch review", "items array is required")
		return
	}

	dir := h.store.ItemDir(kind, name)
	now := r.URL.Query().Get("now") // allow test override
	if now == "" {
		now = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	}

	// Separate target and requirement updates.
	targetUpdates := map[string]ReviewState{}
	reqUpdates := map[string]map[string]RequirementReviewUpdate{}

	for _, item := range body.Items {
		switch item.Type {
		case "target":
			rs := ReviewState{
				ReviewStatus:  item.ReviewStatus,
				ReviewComment: item.ReviewComment,
			}
			if item.ReviewStatus != "unreviewed" {
				rs.ReviewedAt = now
			}
			targetUpdates[item.ID] = rs
		case "requirement":
			if item.ModuleID == "" {
				httputil.BadRequest(w, "[backlog] batch review", "module_id is required for requirement items")
				return
			}
			if reqUpdates[item.ModuleID] == nil {
				reqUpdates[item.ModuleID] = map[string]RequirementReviewUpdate{}
			}
			reviewed := ""
			if item.ReviewStatus != "unreviewed" {
				reviewed = now
			}
			reqUpdates[item.ModuleID][item.ID] = RequirementReviewUpdate{
				ReviewStatus:  item.ReviewStatus,
				ReviewComment: item.ReviewComment,
				ReviewedAt:    reviewed,
			}
		default:
			httputil.BadRequest(w, "[backlog] batch review", "type must be 'target' or 'requirement'")
			return
		}
	}

	// Apply target review updates.
	if len(targetUpdates) > 0 {
		state, err := ReadReviewState(dir)
		if err != nil {
			httputil.InternalError(w, "[backlog] batch review", err.Error())
			return
		}
		for id, rs := range targetUpdates {
			if rs.ReviewStatus == "unreviewed" {
				delete(state, id)
			} else {
				state[id] = rs
			}
		}
		if err := WriteReviewState(dir, state); err != nil {
			httputil.InternalError(w, "[backlog] batch review", err.Error())
			return
		}
	}

	// Apply requirement review updates by reading each module, patching, and writing back.
	if len(reqUpdates) > 0 {
		for moduleID, updates := range reqUpdates {
			if err := PatchModuleReviewState(dir, moduleID, updates); err != nil {
				httputil.InternalError(w, "[backlog] batch review", fmt.Sprintf("module %s: %s", moduleID, err.Error()))
				return
			}
		}
	}

	_ = httputil.JSON(w, map[string]any{"ok": true})
}
