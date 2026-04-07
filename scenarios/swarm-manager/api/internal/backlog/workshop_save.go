// DOC: docs/internal/SEAMS.md#workshop-auto-trigger
// DOC: docs/reference/api-endpoints.md#workshop-save
//
// Dedicated workshop save endpoint and shared async spawn helper.
// The WorkshopSave handler saves round responses and auto-triggers either the
// next workshop round or a final synthesis pass. The spawnWorkshopAsync helper
// is reused by both WorkshopSave (auto-advance) and Create (auto-initialize).
package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/workshop"
)

// WorkshopSave saves a workshop round's responses and optionally auto-triggers
// the next workshop round or a final synthesis pass.
func (h *Handler) WorkshopSave(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "workshop-save")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] workshop-save", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] workshop-save", apierr.Internal("failed to load backlog item"))
		return
	}

	var req apipb.WorkshopSaveRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[backlog] workshop-save", apierr.BadRequest("invalid request body"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] workshop-save", "invalid request body", &req) {
		return
	}

	// Parse and validate the round content.
	var round workshop.Round
	if err := json.Unmarshal([]byte(req.Content), &round); err != nil {
		apierr.MapError(w, "[backlog] workshop-save", apierr.BadRequest("content is not valid workshop round JSON"))
		return
	}
	round.PendingSynthesis = workshop.NeedsSynthesis(&round)
	content, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		apierr.MapError(w, "[backlog] workshop-save", apierr.Internal("failed to encode round content"))
		return
	}

	// Write the round file.
	itemDir := h.store.ItemDir(kind, name)
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		slog.Error("failed to create workshop dir", "err", err)
		apierr.MapError(w, "[backlog] workshop-save", apierr.Internal("failed to create workshop directory"))
		return
	}

	roundFile := fmt.Sprintf("round-%03d.json", req.RoundNumber)
	roundPath := filepath.Join(workshopDir, roundFile)
	if err := os.WriteFile(roundPath, content, 0o644); err != nil {
		slog.Error("failed to write round file", "path", roundPath, "err", err)
		apierr.MapError(w, "[backlog] workshop-save", apierr.Internal("failed to save round file"))
		return
	}

	info, _ := os.Stat(roundPath)
	var fileSize int64
	if info != nil {
		fileSize = info.Size()
	}
	fileNode := backlogFileToProto(BacklogFile{
		Name: roundFile,
		Path: filepath.Join("workshop", roundFile),
		Type: "file",
		Size: fileSize,
	})

	slog.Info("workshop round saved", "kind", kind, "name", name, "file", roundFile, "bytes", fileSize)

	// Determine auto-advance.
	autoAdvance := &apipb.WorkshopAutoAdvance{Triggered: false, Reason: "disabled"}

	// Load settings to check auto_advance_workshop and maxAutoRounds.
	cfg, cfgErr := settings.NewStore("").Load()
	if cfgErr != nil {
		slog.Warn("failed to load settings for auto-advance", "err", cfgErr)
		cfg = settings.DefaultSettings()
	}
	// Load rounds to get the accurate count after save.
	_, roundCount, loadErr := workshop.LoadLatestRound(itemDir)
	if loadErr != nil {
		slog.Warn("failed to load rounds for auto-advance check", "err", loadErr)
		autoAdvance.Reason = "error"
	} else {
		result := workshop.ShouldAutoAdvance(cfg.AutoAdvanceWorkshop, &round, roundCount, string(kind), cfg.MaxAutoRounds)
		if result.NextMode == string(ResearchModeFinalize) && !workshop.NeedsSynthesis(&round) {
			result.Advance = false
			result.NextMode = ""
		}
		autoAdvance.Reason = result.Reason
		if nextMode := resolveNextMode(result, cfg.AutoAdvanceWorkshop, &round, roundCount, kind, cfg.MaxAutoRounds); nextMode != "" {
			autoAdvance.NextMode = &nextMode
		}
		if result.Advance {
			runMode := ResearchModeWorkshop
			if result.NextMode == string(ResearchModeFinalize) {
				runMode = ResearchModeFinalize
			}

			delaySec := cfg.AutoAdvanceDelaySeconds
			if delaySec > 0 {
				// Deferred advance: write a pending advance file and let the ticker fire it.
				// Cancel any existing pending advance first.
				deletePendingAdvance(itemDir)
				if h.workshopTicker != nil {
					h.workshopTicker.Unregister(string(kind), name)
				}

				now := time.Now().UTC()
				pa := PendingAdvance{
					CreatedAt:  now,
					AdvanceAt:  now.Add(time.Duration(delaySec) * time.Second),
					NextMode:   string(runMode),
					RoundCount: roundCount,
					Kind:       string(kind),
					Name:       name,
				}
				if writeErr := writePendingAdvance(itemDir, pa); writeErr != nil {
					slog.Error("failed to write pending advance", "kind", kind, "name", name, "err", writeErr)
					autoAdvance.Reason = "error"
				} else {
					if h.workshopTicker != nil {
						h.workshopTicker.Register(string(kind), name, pa)
					}
					autoAdvance.Pending = true
					advanceAtStr := pa.AdvanceAt.Format(time.RFC3339)
					autoAdvance.AdvanceAt = &advanceAtStr
					autoAdvance.DelaySeconds = int32(delaySec)
					if nextMode := autoAdvance.NextMode; nextMode == nil {
						nm := string(runMode)
						autoAdvance.NextMode = &nm
					}
				}
			} else {
				// Immediate advance (delay=0): spawn right away.
				runID, taskID, spawnErr := h.spawnWorkshopAsync(item, runMode)
				if spawnErr != nil {
					if errors.Is(spawnErr, agentactivity.ErrBacklogItemBusy) {
						slog.Info("auto-advance skipped: agent already active", "kind", kind, "name", name)
						autoAdvance.Reason = "agent_active"
					} else {
						slog.Error("auto-advance spawn failed", "kind", kind, "name", name, "err", spawnErr)
						autoAdvance.Reason = "error"
					}
				} else {
					autoAdvance.Triggered = true
					autoAdvance.RunId = &runID
					autoAdvance.TaskId = &taskID
				}
			}
		}
	}

	resp := &apipb.WorkshopSaveResponse{
		File:        fileNode,
		AutoAdvance: autoAdvance,
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] workshop-save", apierr.Internal("failed to encode response"))
	}
}

func resolveNextMode(result workshop.AutoAdvanceResult, autoAdvanceEnabled bool, round *workshop.Round, roundCount int, kind BacklogKind, maxAutoRounds int) string {
	if strings.TrimSpace(result.NextMode) != "" {
		if result.NextMode == string(ResearchModeFinalize) && !workshop.NeedsSynthesis(round) {
			return ""
		}
		return result.NextMode
	}
	if round == nil || workshop.CountPendingDecisions(round) > 0 {
		return ""
	}
	if !workshop.NeedsSynthesis(round) {
		return ""
	}
	effective := workshop.ComputeEffectiveScores(round.Readiness, roundCount, string(kind))
	if workshop.IsReady(effective) {
		return string(ResearchModeFinalize)
	}
	if !autoAdvanceEnabled || roundCount >= maxAutoRounds {
		return string(ResearchModeWorkshop)
	}
	return ""
}

// spawnWorkshopAsync spawns a workshop/finalize/initialize agent for the given item.
// Per-item idempotency is enforced by the centralized guard in
// agentactivity.Service.spawnTracked — no local lock needed here.
func (h *Handler) spawnWorkshopAsync(item BacklogItem, mode ResearchMode) (runID, taskID string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	service := h.agentService
	if service == nil {
		return "", "", agentmanager.ErrNotAvailable
	}

	selection, promptErr := h.fetchResearchPrompt(ctx, item, mode)
	prompt := selection.Prompt
	if promptErr != nil {
		slog.Warn("auto-workshop prompt fetch failed, using fallback", "kind", item.Kind, "name", item.Name, "err", promptErr)
		prompt = "Use the backlog item folder as context and perform the requested workshop refinement."
	}

	activityCtx := agentactivity.WithSpec(ctx, agentactivity.Spec{
		OwnerType:   agentactivity.OwnerBacklog,
		OwnerKind:   string(item.Kind),
		OwnerName:   item.Name,
		OwnerTitle:  item.Title,
		Purpose:     researchPurpose(mode),
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"entrypoint":     "backlog.workshop_auto_advance",
			"mode":           string(mode),
			"auto_triggered": "true",
			"skill_id":       selection.SkillID,
		},
	})

	runResult, err := service.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:        string(item.Kind),
		Name:        item.Name,
		Title:       buildResearchTitle(item, mode),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   ".",
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "research",
		Environment: map[string]string{"VROOLI_SPAWN_SOURCE": string(item.Kind) + "/" + item.Name},
	})
	if err != nil {
		return "", "", fmt.Errorf("spawn failed: %w", err)
	}

	return runResult.RunID, runResult.TaskID, nil
}

// WorkshopDeleteRound deletes a workshop round and renumbers subsequent rounds.
func (h *Handler) WorkshopDeleteRound(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "workshop-delete-round")
	if !ok {
		return
	}

	if _, err := h.store.LoadItem(kind, name); err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] workshop-delete-round", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] workshop-delete-round", apierr.Internal("failed to load backlog item"))
		return
	}

	var req apipb.WorkshopDeleteRoundRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[backlog] workshop-delete-round", apierr.BadRequest("invalid request body"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] workshop-delete-round", "invalid request body", &req) {
		return
	}

	// Prevent deletion while an agent is working on this item.
	if h.activityChecker != nil && h.activityChecker.HasActiveAgent(r.Context(), string(kind), name) {
		apierr.MapError(w, "[backlog] workshop-delete-round", apierr.Conflict("an agent is currently working on this item; try again after it finishes"))
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	remaining, err := workshop.DeleteRoundAndRenumber(itemDir, int(req.RoundNumber))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierr.MapError(w, "[backlog] workshop-delete-round", apierr.NotFound("%s", err.Error()))
			return
		}
		slog.Error("workshop-delete-round failed", "err", err)
		apierr.MapError(w, "[backlog] workshop-delete-round", apierr.Internal("failed to delete workshop round"))
		return
	}

	slog.Info("workshop round deleted", "round", req.RoundNumber, "kind", kind, "name", name, "remaining", remaining)

	resp := &apipb.WorkshopDeleteRoundResponse{
		DeletedRound:    req.RoundNumber,
		RemainingRounds: int32(remaining),
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] workshop-delete-round", apierr.Internal("failed to encode response"))
	}
}

// WorkshopReset removes all workshop data (rounds, clarifications, attachments,
// deliverable) for a backlog item while preserving its spec.
func (h *Handler) WorkshopReset(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "workshop-reset")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] workshop-reset", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] workshop-reset", apierr.Internal("failed to load backlog item"))
		return
	}

	// Prevent reset while an agent is working on this item.
	if h.activityChecker != nil && h.activityChecker.HasActiveAgent(r.Context(), string(kind), name) {
		apierr.MapError(w, "[backlog] workshop-reset", apierr.Conflict("an agent is currently working on this item; try again after it finishes"))
		return
	}

	itemDir := h.store.ItemDir(kind, name)

	// Determine deliverable file based on kind.
	deliverableFile := "plan.md"
	if strings.EqualFold(string(kind), "research") {
		deliverableFile = "conclusion.md"
	}

	deleted, err := workshop.ResetWorkshop(itemDir, deliverableFile)
	if err != nil {
		slog.Error("workshop-reset failed", "err", err)
		apierr.MapError(w, "[backlog] workshop-reset", apierr.Internal("failed to reset workshop"))
		return
	}

	// Auto-revert status from "ready" to "backlog" since readiness was
	// determined by the workshop rounds that were just deleted.
	statusReverted := false
	if item.Status == StatusReady {
		item.Status = StatusBacklog
		if err := h.store.SaveItem(item); err != nil {
			slog.Error("workshop-reset failed to revert status", "err", err)
			apierr.MapError(w, "[backlog] workshop-reset", apierr.Internal("failed to revert status"))
			return
		}
		statusReverted = true
	}

	slog.Info("workshop reset", "kind", kind, "name", name, "deleted_rounds", deleted, "status_reverted", statusReverted)

	resp := &apipb.WorkshopResetResponse{
		DeletedRounds:  int32(deleted),
		StatusReverted: statusReverted,
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] workshop-reset", apierr.Internal("failed to encode response"))
	}
}
