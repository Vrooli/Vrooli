package captures

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/promptcatalog"

	"github.com/gorilla/mux"
)

// classifyFailureReason maps a spawn error to a user-facing failure category.
func classifyFailureReason(err error) string {
	if errors.Is(err, agentmanager.ErrNotAvailable) {
		return FailDependencyUnavailable
	}
	msg := err.Error()
	// Prompt catalog entry not registered locally.
	if strings.Contains(msg, "prompt catalog entry missing") {
		return FailPromptMissing
	}
	// Prompt fetch failed — could be prompt-manager down (connection refused,
	// timeout) or the skill genuinely not found. Either way the dependency
	// chain is broken and the user should retry once services are healthy.
	if strings.Contains(msg, "fetch classify prompt") {
		if strings.Contains(msg, "connection refused") ||
			strings.Contains(msg, "no such host") ||
			strings.Contains(msg, "dial tcp") ||
			strings.Contains(msg, "i/o timeout") {
			return FailDependencyUnavailable
		}
		return FailPromptMissing
	}
	return FailAgentError
}

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

	// Update status to classifying, clearing any previous failure info.
	cap.Status = "classifying"
	cap.FailureReason = ""
	cap.Classification = nil
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to update capture"))
		return
	}

	// Remove stale classification file if present.
	_ = os.Remove(h.classificationPath(id))

	runResult, err := h.spawnClassifyAgent(r, cap)
	if err != nil {
		// Store categorized failure on the capture so the UI can display it.
		reason := classifyFailureReason(err)
		cap.Status = "failed"
		cap.FailureReason = reason
		_ = h.writeCapture(cap)

		slog.Error("classification retry spawn failed",
			"capture_id", id,
			"failure_reason", reason,
			"error", err,
		)

		if errors.Is(err, agentmanager.ErrNotAvailable) {
			apierr.MapError(w, "[captures] classify", apierr.Unavailable("agent-manager is not available — try again once it's running"))
			return
		}
		if reason == FailPromptMissing {
			apierr.MapError(w, "[captures] classify", apierr.Unavailable("classification skill not found — prompt-manager may not be ready"))
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
		PhaseKind:   string(agentactivity.LaneInvestigate),
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
