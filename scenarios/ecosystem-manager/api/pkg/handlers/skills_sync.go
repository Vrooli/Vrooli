package handlers

import (
	"net/http"

	"github.com/ecosystem-manager/api/pkg/autosteer"
)

// SkillsSyncHandlers provides HTTP handlers for skills synchronization with prompt-manager.
type SkillsSyncHandlers struct {
	promptLoader *autosteer.PromptLoader
}

// NewSkillsSyncHandlers creates a new SkillsSyncHandlers instance.
func NewSkillsSyncHandlers(loader *autosteer.PromptLoader) *SkillsSyncHandlers {
	return &SkillsSyncHandlers{promptLoader: loader}
}

// SyncSkillsHandler triggers a refresh of skills from prompt-manager.
// POST /api/skills/sync
func (h *SkillsSyncHandlers) SyncSkillsHandler(w http.ResponseWriter, r *http.Request) {
	err := h.promptLoader.RefreshCache()
	status := map[string]any{
		"available": h.promptLoader.IsAvailable(),
	}
	if err != nil {
		status["success"] = false
		status["error"] = err.Error()
	} else {
		status["success"] = true
		status["skillCount"] = len(h.promptLoader.GetCachedSkills())
	}
	writeJSON(w, status, http.StatusOK)
}

// ListSkillsHandler returns all cached skills from prompt-manager.
// GET /api/skills
func (h *SkillsSyncHandlers) ListSkillsHandler(w http.ResponseWriter, r *http.Request) {
	skills := h.promptLoader.GetCachedSkills()
	if skills == nil {
		skills = []autosteer.PromptResponse{}
	}
	writeJSON(w, skills, http.StatusOK)
}
