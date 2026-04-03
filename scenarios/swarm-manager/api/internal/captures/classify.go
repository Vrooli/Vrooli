package captures

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/promptcatalog"
)

// Classify spawns a classification agent for a capture (for retry).
func (h *Handler) Classify(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] classify", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to load capture"))
		return
	}

	// Update status to classifying.
	cap.Status = "classifying"
	cap.Classification = nil
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to update capture"))
		return
	}

	// Remove stale classification file if present.
	_ = os.Remove(h.classificationPath(id))

	runResult, err := h.spawnClassifyAgent(r, cap)
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			apierr.MapError(w, "[captures] classify", apierr.Unavailable("agent-manager is not available"))
			return
		}
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to spawn classification agent"))
		return
	}

	h.invalidateTopologyGraph()
	_ = httputil.JSON(w, map[string]any{
		"task_id":  runResult.TaskID,
		"run_id":   runResult.RunID,
		"base_url": runResult.BaseURL,
		"created":  time.Now().UTC().Format(time.RFC3339),
	})
}

// spawnClassifyAgent spawns the classification agent for a capture.
func (h *Handler) spawnClassifyAgent(r *http.Request, cap *capture) (*agentmanager.RunResult, error) {
	service := h.agentService
	if service == nil {
		return nil, agentmanager.ErrNotAvailable
	}
	if !service.IsEnabled() {
		return nil, agentmanager.ErrNotAvailable
	}

	entry, ok := promptcatalog.ResolveCaptureSkill()
	if !ok {
		return nil, fmt.Errorf("capture prompt catalog entry missing")
	}
	skillID := entry.SkillID
	variables := map[string]string{
		"CAPTURE_TEXT": cap.Text,
		"CAPTURE_ID":   cap.ID,
	}

	prompt, err := h.promptClient.ReadSkill(r.Context(), skillID, variables, false)
	if err != nil {
		return nil, fmt.Errorf("fetch classify prompt: %w", err)
	}

	activityCtx := agentactivity.WithSpec(r.Context(), agentactivity.Spec{
		OwnerType:   agentactivity.OwnerCapture,
		OwnerName:   cap.ID,
		OwnerTitle:  truncate(cap.Text, 120),
		Purpose:     agentactivity.PurposeClassify,
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"entrypoint": "captures.classify",
			"skill_id":   skillID,
		},
	})

	result, err := service.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:        "capture",
		Name:        cap.ID,
		Title:       "Classify capture: " + truncate(cap.Text, 60),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   h.captureDir(cap.ID),
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "classify",
		Environment: map[string]string{"VROOLI_SPAWN_SOURCE": "capture/" + cap.ID},
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
