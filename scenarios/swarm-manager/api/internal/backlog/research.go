// Research operations for backlog items: spawning research agents via
// agent-manager, fetching skill prompts, and recording prompt traces.
package backlog

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/prompttrace"
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

// fetchResearchPrompt loads a research prompt from prompt-manager.
func (h *Handler) fetchResearchPrompt(ctx context.Context, item BacklogItem, mode ResearchMode) (promptSelection, error) {
	entry, ok := promptcatalog.ResolveBacklogSkill(string(mode), string(item.Kind))
	if !ok {
		return promptSelection{}, fmt.Errorf("no prompt catalog entry for mode=%s kind=%s", mode, item.Kind)
	}
	skillID := entry.SkillID
	vars := buildVariableMap(item, h.store.ItemDir(item.Kind, item.Name))
	withScope := false
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
		"DELIVERABLE":      deliverable,
	}

	// Add workshop-specific variables.
	rounds, _ := LoadWorkshopRounds(itemFolder)
	vars["PLAN_DRAFT"] = LoadPlanContentByName(itemFolder, deliverable)
	vars["WORKSHOP_HISTORY"] = BuildWorkshopHistory(rounds)
	vars["ROUND_NUMBER"] = fmt.Sprintf("%03d", len(rounds)+1)

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
			httputil.NotFound(w, "[backlog] research", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[backlog] research", "failed to load backlog item")
		return
	}

	var req apipb.BacklogResearchRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[backlog] research", "invalid request body")
			return
		}
		normalizeResearchRequest(&req)
		if !httputil.ValidateProtoRequest(w, "[backlog] research", "invalid request body", &req) {
			return
		}
	}

	mode, modeErr := parseResearchMode(readOptionalString(req.Mode))
	if modeErr != nil {
		httputil.BadRequest(w, "[backlog] research", modeErr.Error())
		return
	}
	if err := validateResearchModeForKind(kind, mode); err != nil {
		httputil.BadRequest(w, "[backlog] research", err.Error())
		return
	}

	if mode == ResearchModeInitialize && item.Status != StatusBacklog {
		httputil.Conflict(w, "[backlog] research", "initialize is only available for items in 'backlog' status")
		return
	}
	if mode == ResearchModeFinalize {
		itemDir := h.store.ItemDir(kind, item.Name)
		latestRound, roundCount, loadErr := LoadLatestRound(itemDir)
		if loadErr != nil {
			httputil.InternalError(w, "[backlog] research", "failed to load workshop rounds for finalize")
			return
		}
		if latestRound == nil {
			httputil.Conflict(w, "[backlog] research", "finalize requires at least one workshop round")
			return
		}
		if CountPendingDecisions(latestRound) > 0 {
			httputil.Conflict(w, "[backlog] research", "finalize is only available after answering all workshop decisions")
			return
		}
		effective := ComputeEffectiveScores(latestRound.Readiness, roundCount, kind)
		if !IsReady(effective) {
			httputil.Conflict(w, "[backlog] research", "finalize is only available when the latest workshop round is ready")
			return
		}
		if !NeedsSynthesis(latestRound) {
			httputil.Conflict(w, "[backlog] research", "finalize is only available when the latest workshop answers have not been synthesized yet")
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
		h.agentService = agentmanager.NewAgentService(agentmanager.DefaultServiceConfig())
		service = h.agentService
	}

	selection, promptErr := h.fetchResearchPrompt(r.Context(), item, mode)
	prompt := selection.Prompt
	if promptErr != nil {
		log.Printf("[backlog] research: prompt fetch failed: %v", promptErr)
		prompt = "Use the backlog item folder as context and perform the requested research."
	}
	trace := prompttrace.Trace{
		SkillID:      selection.SkillID,
		Purpose:      "research",
		Variables:    selection.Variables,
		Prompt:       prompt,
		UsedFallback: promptErr != nil,
		CapturedAt:   prompttrace.NowRFC3339(),
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
				log.Printf("[backlog] research: warning: context path %q does not exist, skipping", p)
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

	if httputil.IsDryRun(r) {
		resp := map[string]any{
			"task_id":  "dry-run-task",
			"run_id":   "dry-run-run",
			"base_url": "",
			"created":  time.Now().UTC().Format(time.RFC3339),
			"dry_run":  true,
			"skill_id": selection.SkillID,
		}
		if err := httputil.JSONWithStatus(w, http.StatusOK, resp); err != nil {
			httputil.InternalError(w, "[backlog] research", "failed to encode dry-run response")
		}
		return
	}

	runResult, err := service.SpawnBacklog(r.Context(), agentmanager.BacklogSpawnRequest{
		Kind:        string(kind),
		Name:        item.Name,
		Title:       buildResearchTitle(item, mode),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		CreatedBy:   "swarm-manager",
		Purpose:     "research",
	})
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			httputil.ServiceUnavailable(w, "[backlog] research", "agent-manager is not available")
			return
		}
		httputil.InternalError(w, "[backlog] research", "failed to spawn research agent")
		return
	}

	resp := &apipb.BacklogResearchResponse{
		TaskId:  runResult.TaskID,
		RunId:   runResult.RunID,
		BaseUrl: runResult.BaseURL,
		Created: runResult.CreatedAt,
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[backlog] research", "failed to encode response")
		return
	}
	tracePath := prompttrace.ResearchTracePath(h.store.ItemDir(kind, item.Name))
	if err := prompttrace.Save(tracePath, trace); err != nil {
		log.Printf("[backlog] research: failed to save prompt trace: %v", err)
	}
}
