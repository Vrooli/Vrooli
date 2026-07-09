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

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/fileops"
	"swarm-manager/internal/fileserve"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathredact"
	"swarm-manager/internal/projectroot"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/workshop"

	repocontract "github.com/vrooli/repo-contract-go"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
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

	itemDir := h.store.ItemDir(kind, name)
	round, roundFile, apiErr := h.writeWorkshopRoundFile(kind, name, itemDir, &req)
	if apiErr != nil {
		apierr.MapError(w, "[backlog] workshop-save", apiErr)
		return
	}

	roundPath := filepath.Join(itemDir, "workshop", roundFile)
	fSize := fileSize(roundPath)
	fileNode := fileserve.FileNodeToProto(fileops.FileNode{
		Name: roundFile,
		Path: filepath.Join("workshop", roundFile),
		Type: "file",
		Size: fSize,
	})

	if apiErr := checkAcceptanceValidation(item, itemDir, round); apiErr != nil {
		apierr.MapError(w, "[backlog] workshop-save", apiErr)
		return
	}

	if round.Mode == "finalize" && kind != KindResearch {
		updated, apiErr := h.bindFinalizedPlanRef(r.Context(), item, req.Content)
		if apiErr != nil {
			apierr.MapError(w, "[backlog] workshop-save", apiErr)
			return
		}
		item = updated
	}

	autoAdvance := h.computeAutoAdvance(item, &round, kind, name, itemDir)

	resp := &apipb.WorkshopSaveResponse{
		File:        fileNode,
		AutoAdvance: autoAdvance,
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] workshop-save", apierr.Internal("failed to encode response"))
	}
}

// writeWorkshopRoundFile parses the round JSON from the request, redacts paths,
// writes the round file, logs analytics, and returns the workshop.Round, the
// round file name, and any error. Extracting file-write operations into this
// helper removes six branches from WorkshopSave.
func (h *Handler) writeWorkshopRoundFile(kind BacklogKind, name, itemDir string, req *apipb.WorkshopSaveRequest) (workshop.Round, string, *apierr.DomainError) {
	var round workshop.Round
	if err := json.Unmarshal([]byte(req.Content), &round); err != nil {
		return workshop.Round{}, "", apierr.BadRequest("content is not valid workshop round JSON")
	}
	round.PendingSynthesis = workshop.NeedsSynthesis(&round)
	encoded, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		return workshop.Round{}, "", apierr.Internal("failed to encode round content")
	}

	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o750); err != nil {
		slog.Error("failed to create workshop dir", "err", err)
		return workshop.Round{}, "", apierr.Internal("failed to create workshop directory")
	}

	roundFile := fmt.Sprintf("round-%03d.json", req.RoundNumber)
	roundPath := filepath.Join(workshopDir, roundFile)
	if redacted, changed := pathredact.NewForArtifactPath(roundPath).RedactBytes(roundPath, encoded); changed {
		encoded = redacted
	}
	if err := os.WriteFile(roundPath, encoded, 0o600); err != nil {
		slog.Error("failed to write round file", "path", roundPath, "err", err)
		return workshop.Round{}, "", apierr.Internal("failed to save round file")
	}

	if h.eventLogger != nil {
		summary := workshop.SummarizeRound(&round)
		h.eventLogger.EmitWorkshopRoundCompleted(string(kind)+"/"+name, eventlog.WorkshopRoundPayload{
			RoundNumber:            int(req.RoundNumber),
			Kind:                   string(kind),
			ItemsTotal:             summary.ItemsTotal,
			ItemsAnswered:          summary.ItemsAnswered,
			ItemsRecommendedChosen: summary.ItemsRecommendedChosen,
			ItemsFreeformChosen:    summary.ItemsFreeformChosen,
		})
	}

	slog.Info("workshop round saved", "kind", kind, "name", name, "file", roundFile, "bytes", fileSize(roundPath))
	return round, roundFile, nil
}

// fileSize returns the file size in bytes, or 0 if the file cannot be stat'd.
func fileSize(path string) int64 {
	info, _ := os.Stat(path)
	if info == nil {
		return 0
	}
	return info.Size()
}

// checkAcceptanceValidation runs the acceptance validator and returns a
// PlanStale error when the round is a finalize and acceptance globs are
// broken. Returns nil (no blocker) otherwise.
func checkAcceptanceValidation(item BacklogItem, itemDir string, round workshop.Round) *apierr.DomainError {
	accReport, accErr := runAcceptanceValidation(item, itemDir)
	if accErr != nil {
		slog.Warn("acceptance validation could not run", "kind", item.Kind, "name", item.Name, "err", accErr)
		return nil
	}
	if accReport != nil && !accReport.Clean() && round.Mode == "finalize" {
		return apierr.PlanStale(
			"finalization blocked: plan references paths that do not exist and are not declared in `creates`",
			map[string]any{
				"missingPaths": accReport.Problems,
			},
		)
	}
	return nil
}

type finalizePlanRefPayload struct {
	PlanRef  *PlanRef `json:"plan_ref,omitempty"`
	PlanID   string   `json:"plan_id,omitempty"`
	PlanSlug string   `json:"plan_slug,omitempty"`
}

func (h *Handler) bindFinalizedPlanRef(ctx context.Context, item BacklogItem, roundContent string) (BacklogItem, *apierr.DomainError) {
	ref, err := finalizedPlanRefFromRound(roundContent, item.PlanRef)
	if err != nil {
		return item, apierr.BadRequest("%s", err.Error())
	}
	if ref == nil {
		return item, apierr.Conflict("finalization requires a canonical plan_ref or plan_slug from plan-manager")
	}
	if h.planClient == nil {
		return item, apierr.Internal("plan-manager client is not configured")
	}
	lookup := firstNonBlank(ref.PlanID, ref.Slug)
	plan, err := h.planClient.GetPlan(ctx, lookup)
	if err != nil {
		return item, apierr.Conflict("finalization requires a resolvable plan-manager plan_ref: %s", err.Error())
	}
	resolved := &PlanRef{
		Provider: PlanRefProviderPlanManager,
		PlanID:   strings.TrimSpace(plan.GetId()),
		Slug:     strings.TrimSpace(plan.GetSlug()),
		Role:     PlanRefRoleExecutionSpec,
	}
	if resolved.PlanID == "" {
		resolved.PlanID = ref.PlanID
	}
	if resolved.Slug == "" {
		resolved.Slug = ref.Slug
	}
	if err := validatePlanRef(resolved, PlanRefRoleExecutionSpec); err != nil {
		return item, apierr.Conflict("finalization resolved invalid plan_ref: %s", err.Error())
	}
	item.PlanRef = resolved
	item.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := h.store.SaveItem(item); err != nil {
		return item, apierr.Internal("failed to save finalized plan_ref")
	}
	return item, nil
}

func finalizedPlanRefFromRound(roundContent string, existing *PlanRef) (*PlanRef, error) {
	var payload finalizePlanRefPayload
	if err := json.Unmarshal([]byte(roundContent), &payload); err != nil {
		return nil, fmt.Errorf("content is not valid workshop round JSON")
	}
	ref := normalizePlanRef(payload.PlanRef)
	if ref == nil {
		ref = normalizePlanRef(existing)
	}
	if ref == nil && (strings.TrimSpace(payload.PlanID) != "" || strings.TrimSpace(payload.PlanSlug) != "") {
		ref = &PlanRef{
			Provider: PlanRefProviderPlanManager,
			PlanID:   strings.TrimSpace(payload.PlanID),
			Slug:     strings.TrimSpace(payload.PlanSlug),
			Role:     PlanRefRoleExecutionSpec,
		}
	}
	if ref == nil {
		return nil, nil
	}
	if ref.Provider == "" {
		ref.Provider = PlanRefProviderPlanManager
	}
	if ref.Role == "" {
		ref.Role = PlanRefRoleExecutionSpec
	}
	if ref.PlanID == "" && ref.Slug == "" {
		return nil, fmt.Errorf("plan_ref requires plan_id or slug")
	}
	if ref.Provider != PlanRefProviderPlanManager {
		return nil, fmt.Errorf("plan_ref.provider must be %q", PlanRefProviderPlanManager)
	}
	if ref.Role != PlanRefRoleExecutionSpec {
		return nil, fmt.Errorf("plan_ref.role must be %q", PlanRefRoleExecutionSpec)
	}
	return ref, nil
}

// computeAutoAdvance determines whether and how to auto-advance the workshop
// after saving a round.
func (h *Handler) computeAutoAdvance(item BacklogItem, round *workshop.Round, kind BacklogKind, name, itemDir string) *apipb.WorkshopAutoAdvance {
	autoAdvance := &apipb.WorkshopAutoAdvance{Triggered: false, Reason: "disabled"}

	cfg, cfgErr := settings.NewStore("").Load()
	if cfgErr != nil {
		slog.Warn("failed to load settings for auto-advance", "err", cfgErr)
		cfg = settings.DefaultSettings()
	}

	_, roundCount, loadErr := workshop.LoadLatestRound(itemDir)
	if loadErr != nil {
		slog.Warn("failed to load rounds for auto-advance check", "err", loadErr)
		autoAdvance.Reason = "error"
		return autoAdvance
	}

	result := workshop.ShouldAutoAdvance(cfg.AutoAdvanceWorkshop, round, roundCount, string(kind), cfg.MaxAutoRounds)
	if result.NextMode == string(ResearchModeFinalize) && !workshop.NeedsSynthesis(round) {
		result.Advance = false
		result.NextMode = ""
	}
	autoAdvance.Reason = result.Reason
	if nextMode := resolveNextMode(result, cfg.AutoAdvanceWorkshop, round, roundCount, kind, cfg.MaxAutoRounds); nextMode != "" {
		autoAdvance.NextMode = &nextMode
	}

	if !result.Advance {
		return autoAdvance
	}

	runMode := ResearchModeWorkshop
	if result.NextMode == string(ResearchModeFinalize) {
		runMode = ResearchModeFinalize
	}

	if cfg.AutoAdvanceDelaySeconds > 0 {
		h.scheduleDeferredAdvance(autoAdvance, kind, name, itemDir, runMode, roundCount, cfg.AutoAdvanceDelaySeconds)
	} else {
		h.executeImmediateAdvance(autoAdvance, item, kind, name, runMode)
	}
	return autoAdvance
}

// scheduleDeferredAdvance writes a pending advance file and registers it with the ticker.
func (h *Handler) scheduleDeferredAdvance(aa *apipb.WorkshopAutoAdvance, kind BacklogKind, name, itemDir string, runMode ResearchMode, roundCount, delaySec int) {
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
		aa.Reason = "error"
		return
	}
	if h.workshopTicker != nil {
		h.workshopTicker.Register(string(kind), name, pa)
	}
	aa.Pending = true
	advanceAtStr := pa.AdvanceAt.Format(time.RFC3339)
	aa.AdvanceAt = &advanceAtStr
	aa.DelaySeconds = int32(delaySec)
	if aa.NextMode == nil {
		nm := string(runMode)
		aa.NextMode = &nm
	}
}

// executeImmediateAdvance spawns the workshop agent immediately.
func (h *Handler) executeImmediateAdvance(aa *apipb.WorkshopAutoAdvance, item BacklogItem, kind BacklogKind, name string, runMode ResearchMode) {
	runID, taskID, spawnErr := h.spawnWorkshopAsync(item, runMode)
	if spawnErr != nil {
		if errors.Is(spawnErr, agentactivity.ErrBacklogItemBusy) {
			slog.Info("auto-advance skipped: agent already active", "kind", kind, "name", name)
			aa.Reason = "agent_active"
		} else {
			slog.Error("auto-advance spawn failed", "kind", kind, "name", name, "err", spawnErr)
			aa.Reason = "error"
		}
		return
	}
	aa.Triggered = true
	aa.RunId = &runID
	aa.TaskId = &taskID
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
		PhaseKind:   researchLane(mode),
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

// WorkshopReset removes workshop data (rounds, clarifications, attachments) for
// a backlog item while preserving its spec and canonical plan_ref. Research
// items also remove conclusion.md because that remains their local deliverable.
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

	deliverableFile := ""
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

// ReWorkshop is the high-level "plan is stale, redo the workshop" trigger.
// It clears the existing workshop rounds and local research conclusion artifact
// (same as WorkshopReset), reverts status back to a draft state, and queues a
// fresh workshop round. The intended caller is the UI's stale-plan panel,
// which surfaces after spawn-time validation rejects an item with a
// plan_stale error; the CLI mirror is `swarm-manager backlog re-workshop`.
func (h *Handler) ReWorkshop(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "re-workshop")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] re-workshop", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] re-workshop", apierr.Internal("failed to load backlog item"))
		return
	}

	if h.activityChecker != nil && h.activityChecker.HasActiveAgent(r.Context(), string(kind), name) {
		apierr.MapError(w, "[backlog] re-workshop", apierr.Conflict("an agent is currently working on this item; try again after it finishes"))
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	deliverableFile := ""
	if strings.EqualFold(string(kind), "research") {
		deliverableFile = "conclusion.md"
	}

	deleted, err := workshop.ResetWorkshop(itemDir, deliverableFile)
	if err != nil {
		slog.Error("re-workshop reset failed", "err", err)
		apierr.MapError(w, "[backlog] re-workshop", apierr.Internal("failed to reset workshop artifacts"))
		return
	}

	// Force status back to backlog so the workshop ticker picks it up.
	statusReverted := false
	if item.Status != StatusBacklog {
		item.Status = StatusBacklog
		if err := h.store.SaveItem(item); err != nil {
			slog.Error("re-workshop failed to revert status", "err", err)
			apierr.MapError(w, "[backlog] re-workshop", apierr.Internal("failed to revert status"))
			return
		}
		statusReverted = true
	}

	// Kick off a fresh workshop round. Bypass dependency gating because the
	// caller has explicitly asked us to re-author against the current repo.
	go func(it BacklogItem) {
		if _, _, spawnErr := h.spawnWorkshopAsync(it, ResearchModeInitialize); spawnErr != nil {
			slog.Error("re-workshop spawn failed", "kind", it.Kind, "name", it.Name, "err", spawnErr)
		}
	}(item)

	slog.Info("re-workshop triggered", "kind", kind, "name", name, "deleted_rounds", deleted, "status_reverted", statusReverted)

	resp := &apipb.WorkshopResetResponse{
		DeletedRounds:  int32(deleted),
		StatusReverted: statusReverted,
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] re-workshop", apierr.Internal("failed to encode response"))
	}
}

// AcceptanceValidationArtifact captures the structured result of running
// the acceptance validator against an item's spec. It is persisted as
// `acceptance-validation.json` in the item directory after every workshop
// round save so the next round (and the operator) can see exactly which
// globs are stale and why. When `Problems` is empty the report is clean.
type AcceptanceValidationArtifact struct {
	GeneratedAt string                    `json:"generated_at"`
	RepoRootRef string                    `json:"repo_root_ref"`
	Problems    []projectroot.GlobProblem `json:"problems"`
}

// runAcceptanceValidation runs the relaxed validator (honors creates) for
// the given item, persists a structured artifact under the item directory,
// and returns the report. A nil return with a nil error means validation
// could not run (no acceptance globs declared, or repo root undiscoverable);
// the caller should treat that as "not blocking."
func runAcceptanceValidation(item BacklogItem, itemDir string) (*projectroot.AcceptanceReport, error) {
	if len(item.AcceptanceAllow) == 0 {
		return nil, nil
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return nil, fmt.Errorf("find repo root: %w", err)
	}
	report, valErr := projectroot.ValidateAcceptance(root, item.AcceptanceAllow, item.Creates)
	// Persist artifact regardless of success — a clean run records "no problems"
	// so consumers can distinguish "validated and clean" from "never validated."
	artifact := AcceptanceValidationArtifact{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		RepoRootRef: "path:.",
	}
	if report != nil {
		artifact.Problems = report.Problems
	}
	artifactPath := filepath.Join(itemDir, "acceptance-validation.json")
	redactedArtifact := any(artifact)
	if redacted, changed, err := pathredact.NewForArtifactPath(artifactPath).RedactJSONValue(artifact); err == nil && changed {
		redactedArtifact = redacted
	}
	data, marshalErr := json.MarshalIndent(redactedArtifact, "", "  ")
	if marshalErr == nil {
		if writeErr := os.WriteFile(artifactPath, data, 0o600); writeErr != nil {
			slog.Warn("backlog: persist redacted workshop artifact failed", "err", writeErr, "path", artifactPath)
		}
	}
	if valErr != nil && !errors.Is(valErr, projectroot.ErrAcceptanceMismatch) {
		return report, valErr
	}
	return report, nil
}
