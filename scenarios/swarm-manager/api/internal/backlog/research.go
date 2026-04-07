// Research operations for backlog items: spawning research agents via
// agent-manager, fetching skill prompts, and recording prompt traces.
package backlog

import (
	"context"
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
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/prompttrace"
	"swarm-manager/internal/workshop"
)

// parseResearchMode normalizes a raw mode string into a ResearchMode constant.
// Returns an error for unrecognized values.
func parseResearchMode(raw string) (ResearchMode, error) {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	switch candidate {
	case "workshop", "":
		return ResearchModeWorkshop, nil
	case "finalize":
		return ResearchModeFinalize, nil
	case "initialize":
		return ResearchModeInitialize, nil
	// Capture-related modes (clarify, suggest, enhance) are treated as workshop.
	case "clarify", "suggest", "enhance":
		return ResearchModeWorkshop, nil
	default:
		return "", fmt.Errorf("unsupported research mode %q: must be workshop, finalize, or initialize", candidate)
	}
}

// validateResearchModeForKind checks whether a mode is valid for a given kind.
func validateResearchModeForKind(_ BacklogKind, _ ResearchMode) error {
	// All modes are valid for all kinds.
	return nil
}

// normalizeResearchRequest trims and lowercases optional fields on a research
// request, clearing empty strings to nil.
func normalizeResearchRequest(req *apipb.BacklogResearchRequest) {
	if req == nil {
		return
	}
	if req.Mode != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*req.Mode))
		if trimmed == "" {
			req.Mode = nil
		} else {
			req.Mode = &trimmed
		}
	}
}

// readOptionalString dereferences a string pointer, returning "" for nil.
func readOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// buildResearchTitle constructs a human-readable title for the research agent
// task, prefixed with the research mode.
func buildResearchTitle(item BacklogItem, mode ResearchMode) string {
	label := strings.TrimSpace(item.Title)
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "backlog item"
	}
	switch mode {
	case ResearchModeWorkshop:
		return "Workshop: " + label
	case ResearchModeFinalize:
		return "Finalize: " + label
	case ResearchModeInitialize:
		return "Initialize: " + label
	default:
		return "Workshop: " + label
	}
}

func researchPurpose(mode ResearchMode) agentactivity.Purpose {
	switch mode {
	case ResearchModeInitialize:
		return agentactivity.PurposeInitialize
	case ResearchModeFinalize:
		return agentactivity.PurposeFinalize
	case ResearchModeWorkshop:
		return agentactivity.PurposeWorkshop
	default:
		return agentactivity.PurposeResearch
	}
}

// fetchResearchPrompt loads a research prompt from prompt-manager.
func (h *Handler) fetchResearchPrompt(ctx context.Context, item BacklogItem, mode ResearchMode) (promptSelection, error) {
	entry, ok := promptcatalog.ResolveBacklogSkill(string(mode), string(item.Kind))
	if !ok {
		return promptSelection{}, fmt.Errorf("no prompt catalog entry for mode=%s kind=%s", mode, item.Kind)
	}
	skillID := entry.SkillID
	vars := buildVariableMap(item, h.store.ItemDir(item.Kind, item.Name))
	withScope := false

	// Use experiment-aware read if the catalog entry has an active experiment.
	if strings.TrimSpace(entry.ExperimentID) != "" {
		result, err := h.promptClient.ReadSkillWithExperiment(ctx, skillID, vars, withScope, entry.ExperimentID)
		if err != nil {
			return promptSelection{SkillID: skillID, Variables: vars, ExperimentID: entry.ExperimentID}, err
		}
		return promptSelection{
			SkillID:      skillID,
			Variables:    vars,
			Prompt:       result.Content,
			ExperimentID: entry.ExperimentID,
			VariantID:    result.VariantID,
		}, nil
	}

	prompt, err := h.promptClient.ReadSkill(ctx, skillID, vars, withScope)
	if err != nil {
		return promptSelection{SkillID: skillID, Variables: vars}, err
	}
	return promptSelection{
		SkillID:   skillID,
		Variables: vars,
		Prompt:    prompt,
	}, nil
}

// buildVariableMap creates the template variable map for prompt-manager skill rendering.
func buildVariableMap(item BacklogItem, itemFolder string) map[string]string {
	deliverable := DeliverableForKind(item.Kind)
	vars := map[string]string{
		"ITEM_NAME":        item.Name,
		"ITEM_TITLE":       item.Title,
		"ITEM_DESCRIPTION": item.Description,
		"ITEM_KIND":        string(item.Kind),
		"ITEM_STATUS":      string(item.Status),
		"ITEM_PRIORITY":    fmt.Sprintf("%d", item.Priority),
		"ITEM_TAGS":        strings.Join(item.Tags, ", "),
		"ITEM_FOLDER":      itemFolder,
		"ITEM_INITIATIVE":  item.Initiative,
		"DELIVERABLE":      deliverable,
	}

	// Add workshop-specific variables.
	rounds, _ := LoadWorkshopRounds(itemFolder)
	vars["PLAN_DRAFT"] = LoadPlanContentByName(itemFolder, deliverable)
	vars["WORKSHOP_HISTORY"] = BuildWorkshopHistory(rounds)
	vars["ROUND_NUMBER"] = fmt.Sprintf("%03d", len(rounds)+1)

	// Add distilled clarification context notes for future rounds.
	clarifications, _ := workshop.LoadAllClarifications(itemFolder)
	var clarNotes []string
	for _, c := range clarifications {
		if c.Status == "resolved" && c.LatestImpact != nil && c.LatestImpact.ContextNote != "" {
			clarNotes = append(clarNotes, fmt.Sprintf("[Round %d, Item %s] %s",
				c.RoundNumber, c.ItemID, c.LatestImpact.ContextNote))
		}
	}
	vars["CLARIFICATION_NOTES"] = strings.Join(clarNotes, "\n")

	return vars
}

// Research spawns a research agent via agent-manager for the specified backlog item.
func (h *Handler) Research(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "research")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] research", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to load backlog item"))
		return
	}

	var req apipb.BacklogResearchRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			apierr.MapError(w, "[backlog] research", apierr.BadRequest("invalid request body"))
			return
		}
		normalizeResearchRequest(&req)
		if !httputil.ValidateProtoRequest(w, "[backlog] research", "invalid request body", &req) {
			return
		}
	}

	mode, modeErr := parseResearchMode(readOptionalString(req.Mode))
	if modeErr != nil {
		apierr.MapError(w, "[backlog] research", apierr.BadRequest("%s", modeErr.Error()))
		return
	}
	if err := validateResearchModeForKind(kind, mode); err != nil {
		apierr.MapError(w, "[backlog] research", apierr.BadRequest("%s", err.Error()))
		return
	}

	confirm := req.GetConfirm()
	force := req.GetForce()

	// Dependency blocking applies to initialize and workshop modes.
	// Finalize, clarify, suggest, and enhance skip dep checks — once a
	// workshop has started or is being refined, it should complete
	// regardless of dependency state.
	if mode == ResearchModeInitialize || mode == ResearchModeWorkshop {
		depReasons, depErr := EvaluateDependencyBlocking(item, h.store)
		if depErr != nil {
			slog.Error("research dependency check failed", "kind", kind, "name", name, "err", depErr)
			apierr.MapError(w, "[backlog] research", apierr.Internal("failed to check dependencies"))
			return
		}
		if len(depReasons) > 0 {
			protoReasons := make([]*apipb.BlockingReason, len(depReasons))
			for i, r := range depReasons {
				protoReasons[i] = &apipb.BlockingReason{Message: r.Message, Forceable: r.Forceable}
			}
			if !confirm {
				resp := &apipb.BacklogResearchResponse{
					DryRun:          true,
					Started:         false,
					Message:         "Research blocked by dependencies. Use confirm=true and force=true (CLI: --execute --force) to override.",
					BlockingReasons: protoReasons,
				}
				if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
					apierr.MapError(w, "[backlog] research", apierr.Internal("failed to encode blocked response"))
				}
				return
			}
			if !force || HasNonForceableReasons(depReasons) {
				resp := &apipb.BacklogResearchResponse{
					DryRun:          true,
					Started:         false,
					Message:         "Research blocked by dependencies.",
					BlockingReasons: protoReasons,
				}
				if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
					apierr.MapError(w, "[backlog] research", apierr.Internal("failed to encode blocked response"))
				}
				return
			}
			// force=true and all reasons are forceable — proceed.
		}
	}

	if mode == ResearchModeInitialize && item.Status != StatusBacklog {
		apierr.MapError(w, "[backlog] research", apierr.Conflict("initialize is only available for items in 'backlog' status"))
		return
	}
	if mode == ResearchModeFinalize {
		itemDir := h.store.ItemDir(kind, item.Name)
		latestRound, roundCount, loadErr := LoadLatestRound(itemDir)
		if loadErr != nil {
			apierr.MapError(w, "[backlog] research", apierr.Internal("failed to load workshop rounds for finalize"))
			return
		}
		if latestRound == nil {
			apierr.MapError(w, "[backlog] research", apierr.Conflict("finalize requires at least one workshop round"))
			return
		}
		if CountPendingDecisions(latestRound) > 0 {
			apierr.MapError(w, "[backlog] research", apierr.Conflict("finalize is only available after answering all workshop decisions"))
			return
		}
		effective := ComputeEffectiveScores(latestRound.Readiness, roundCount, kind)
		if !IsReady(effective) {
			apierr.MapError(w, "[backlog] research", apierr.Conflict("finalize is only available when the latest workshop round is ready"))
			return
		}
		if !NeedsSynthesis(latestRound) {
			apierr.MapError(w, "[backlog] research", apierr.Conflict("finalize is only available when the latest workshop answers have not been synthesized yet"))
			return
		}
	}

	scopePath := "." // Always use project root for sandbox overlay.
	projectRoot := strings.TrimSpace(readOptionalString(req.ProjectRoot))
	if projectRoot == "" {
		projectRoot = "."
	}

	service := h.agentService
	if service == nil {
		apierr.MapError(w, "[backlog] research", apierr.Unavailable("agent-manager is not available"))
		return
	}

	selection, promptErr := h.fetchResearchPrompt(r.Context(), item, mode)
	prompt := selection.Prompt
	if promptErr != nil {
		slog.Warn("research prompt fetch failed, using fallback", "err", promptErr)
		prompt = "Use the backlog item folder as context and perform the requested research."
	}
	trace := prompttrace.Trace{
		SkillID:      selection.SkillID,
		Purpose:      "research",
		Variables:    selection.Variables,
		Prompt:       prompt,
		UsedFallback: promptErr != nil,
		CapturedAt:   prompttrace.NowRFC3339(),
		ExperimentID: selection.ExperimentID,
		VariantID:    selection.VariantID,
	}
	if strings.TrimSpace(readOptionalString(req.Prompt)) != "" {
		prompt = prompt + "\n\nAdditional context from user:\n" + strings.TrimSpace(readOptionalString(req.Prompt))
		trace.Prompt = prompt
	}

	// Append attached context sections from request.
	if len(req.ContextPaths) > 0 {
		prompt += "\n\nAttached files for reference:\n"
		for _, p := range req.ContextPaths {
			if _, statErr := os.Stat(p); statErr != nil {
				slog.Warn("research context path does not exist, skipping", "path", p)
				continue
			}
			prompt += "- " + p + "\n"
		}
		trace.Prompt = prompt
	}
	archiveDir := filepath.Join(h.store.ItemDir(kind, item.Name), "archive")
	if len(req.ContextTargetIds) > 0 {
		targets, parseErr := ParseArchiveTargets(archiveDir)
		if parseErr == nil && len(targets) > 0 {
			idSet := make(map[string]bool, len(req.ContextTargetIds))
			for _, id := range req.ContextTargetIds {
				idSet[id] = true
			}
			prompt += "\n\nAttached operational targets:\n"
			for _, t := range targets {
				if idSet[t.ID] {
					prompt += fmt.Sprintf("- [%s] %s | %s | %s (status: %s)\n", t.Criticality, t.ID, t.Title, t.Notes, t.Status)
				}
			}
			trace.Prompt = prompt
		}
	}
	if len(req.ContextRequirementIds) > 0 {
		groups, parseErr := ParseArchiveRequirements(archiveDir)
		if parseErr == nil && len(groups) > 0 {
			idSet := make(map[string]bool, len(req.ContextRequirementIds))
			for _, id := range req.ContextRequirementIds {
				idSet[id] = true
			}
			var flatReqs []ArchiveRequirement
			var walkGroups func([]ArchiveRequirementGroup)
			walkGroups = func(gs []ArchiveRequirementGroup) {
				for _, g := range gs {
					for _, r := range g.Requirements {
						if idSet[r.ID] {
							flatReqs = append(flatReqs, r)
						}
					}
					walkGroups(g.Children)
				}
			}
			walkGroups(groups)
			if len(flatReqs) > 0 {
				prompt += "\n\nAttached requirements:\n"
				for _, r := range flatReqs {
					prompt += fmt.Sprintf("- [%s] %s: %s (status: %s)\n", r.ID, r.Title, r.Description, r.Status)
				}
				trace.Prompt = prompt
			}
		}
	}

	// Cancel any pending auto-advance for workshop/finalize modes. This
	// prevents a deferred auto-advance from racing with the user's manual
	// "Next Round" click. The centralized guard in agentactivity.spawnTracked
	// is the authoritative backstop against double-spawns.
	if mode == ResearchModeWorkshop || mode == ResearchModeFinalize {
		itemDir := h.store.ItemDir(kind, item.Name)
		if deletePendingAdvance(itemDir) {
			slog.Info("research: cancelled pending auto-advance", "kind", kind, "name", name, "mode", mode)
		}
		if h.workshopTicker != nil {
			h.workshopTicker.Unregister(string(kind), name)
		}
	}

	if httputil.IsDryRun(r) {
		resp := &apipb.BacklogResearchResponse{
			TaskId:  "dry-run-task",
			RunId:   "dry-run-run",
			BaseUrl: "",
			Created: time.Now().UTC().Format(time.RFC3339),
			DryRun:  true,
			Started: false,
			Message: "Dry run. No agent spawned.",
		}
		if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
			apierr.MapError(w, "[backlog] research", apierr.Internal("failed to encode dry-run response"))
		}
		return
	}

	activityCtx := agentactivity.WithSpec(r.Context(), agentactivity.Spec{
		OwnerType:   agentactivity.OwnerBacklog,
		OwnerKind:   string(kind),
		OwnerName:   item.Name,
		OwnerTitle:  item.Title,
		Purpose:     researchPurpose(mode),
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"entrypoint": "backlog.research",
			"mode":       string(mode),
			"skill_id":   selection.SkillID,
		},
	})

	runResult, err := service.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:        string(kind),
		Name:        item.Name,
		Title:       buildResearchTitle(item, mode),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		CreatedBy:   "swarm-manager",
		Purpose:     "research",
		Environment: map[string]string{"VROOLI_SPAWN_SOURCE": string(kind) + "/" + item.Name},
	})
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			apierr.MapError(w, "[backlog] research", apierr.Unavailable("agent-manager is not available"))
			return
		}
		if errors.Is(err, agentactivity.ErrBacklogItemBusy) {
			apierr.MapError(w, "[backlog] research", apierr.Conflict("an agent is already active for this backlog item"))
			return
		}
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to spawn research agent"))
		return
	}

	resp := &apipb.BacklogResearchResponse{
		TaskId:  runResult.TaskID,
		RunId:   runResult.RunID,
		BaseUrl: runResult.BaseURL,
		Created: runResult.CreatedAt,
		DryRun:  false,
		Started: true,
		Message: "Research agent spawned successfully.",
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to encode response"))
		return
	}
	tracePath := prompttrace.ResearchTracePath(h.store.ItemDir(kind, item.Name))
	if err := prompttrace.Save(tracePath, trace); err != nil {
		slog.Warn("failed to save prompt trace", "err", err)
	}
}
