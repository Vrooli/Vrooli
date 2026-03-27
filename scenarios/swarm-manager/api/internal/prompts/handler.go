// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package prompts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
)

type Handler struct {
	rootDir string
	client  promptmanager.AdminClient
}

func NewHandler(rootDir string, client promptmanager.AdminClient) *Handler {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = pathutil.ResolveScenarioRoot("swarm-manager")
	}
	if client == nil {
		client = promptmanager.NewHTTPClient()
	}
	return &Handler{
		rootDir: rootDir,
		client:  client,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/prompts/map", h.Map).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prompts/skills", h.ListSkills).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prompts/skills/{id}", h.GetSkill).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prompts/skills/{id}", h.UpdateSkill).Methods(http.MethodPut)
	r.HandleFunc("/api/v1/prompts/skills/{id}/versions", h.GetSkillVersions).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prompts/skills/{id}/revert/{version}", h.RevertSkillVersion).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/prompts/preview", h.Preview).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/prompts/simulate", h.Simulate).Methods(http.MethodPost)
}

type PromptBinding struct {
	Area        string   `json:"area"`
	Trigger     string   `json:"trigger"`
	Kind        string   `json:"kind,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	Operation   string   `json:"operation,omitempty"`
	SkillID     string   `json:"skill_id"`
	Purpose     string   `json:"purpose"`
	OutputPaths []string `json:"output_paths,omitempty"`
}

func promptBindings() []PromptBinding {
	return []PromptBinding{
		{
			Area:        "research",
			Trigger:     "Backlog Workshop: All Kinds",
			Kind:        "idea,research,fix,execute,chore",
			Mode:        "workshop",
			SkillID:     "swarm-manager-workshop",
			Purpose:     "Run one workshop round: generate questions, proposals, assess readiness, and update implementation plan.",
			OutputPaths: []string{"workshop/round-NNN.json", "plan.md"},
		},
		{
			Area:        "research",
			Trigger:     "Backlog Research: Idea Deep Research",
			Kind:        "idea",
			Mode:        "research",
			SkillID:     "swarm-manager-research-idea",
			Purpose:     "Evaluate feasibility, dependencies, and integration opportunities.",
			OutputPaths: []string{"research/summary.md"},
		},
		{
			Area:        "research",
			Trigger:     "Backlog Research: Fix Deep Research",
			Kind:        "fix",
			Mode:        "research",
			SkillID:     "swarm-manager-research-fix",
			Purpose:     "Root-cause analysis, blast radius, and safe remediation plan.",
			OutputPaths: []string{"research/summary.md"},
		},
		{
			Area:        "research",
			Trigger:     "Backlog Research: Execute/Research Deep Research",
			Kind:        "execute,research",
			Mode:        "research",
			SkillID:     "swarm-manager-research-general",
			Purpose:     "General investigation with actionable findings and recommendations.",
			OutputPaths: []string{"research/summary.md"},
		},
		{
			Area:    "process",
			Trigger: "Execution Start: All Kinds",
			Kind:    "idea,fix,execute,research,chore",
			Purpose: "Execution agent receives plan.md directly as the prompt. No skill template is used.",
		},
	}
}

func allowedSkillID(skillID string) bool {
	trimmed := strings.TrimSpace(skillID)
	return strings.HasPrefix(trimmed, "swarm-manager-")
}

func (h *Handler) Map(w http.ResponseWriter, _ *http.Request) {
	bindings := promptBindings()
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].Area == bindings[j].Area {
			return bindings[i].Trigger < bindings[j].Trigger
		}
		return bindings[i].Area < bindings[j].Area
	})
	if err := httputil.JSON(w, map[string]any{"items": bindings}); err != nil {
		httputil.InternalError(w, "[prompts] map", "failed to encode response")
	}
}

type PromptSkillSummary struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	DefaultScope    string   `json:"default_scope,omitempty"`
	Draft           bool     `json:"draft"`
	UpdatedAt       string   `json:"updated_at,omitempty"`
	CreatedAt       string   `json:"created_at,omitempty"`
	TriggerCount    int      `json:"trigger_count"`
	ImpactSummary   string   `json:"impact_summary"`
	CurrentContent  string   `json:"current_content,omitempty"`
	RequiredMissing []string `json:"required_missing,omitempty"`
}

func bindingCountBySkill() map[string]int {
	counts := make(map[string]int)
	for _, binding := range promptBindings() {
		counts[binding.SkillID]++
	}
	return counts
}

func requiredVariablesBySkill() map[string][]string {
	common := []string{"ITEM_TITLE", "ITEM_FOLDER"}
	return map[string][]string{
		"swarm-manager-workshop":         common,
		"swarm-manager-research-idea":    common,
		"swarm-manager-research-fix":     common,
		"swarm-manager-research-general": append([]string{}, common...),
	}
}

func extractTemplateVariables(content string) map[string]struct{} {
	vars := make(map[string]struct{})
	for {
		start := strings.Index(content, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(content[start+2:], "}}")
		if end < 0 {
			break
		}
		token := strings.TrimSpace(content[start+2 : start+2+end])
		if token != "" {
			vars[token] = struct{}{}
		}
		content = content[start+2+end+2:]
	}
	return vars
}

func missingRequiredVariables(skillID, content string) []string {
	required := requiredVariablesBySkill()[skillID]
	if len(required) == 0 {
		return nil
	}
	varSet := extractTemplateVariables(content)
	missing := make([]string, 0, len(required))
	for _, variable := range required {
		if _, ok := varSet[variable]; !ok {
			missing = append(missing, variable)
		}
	}
	sort.Strings(missing)
	return missing
}

func (h *Handler) ListSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := h.client.ListSkills(r.Context(), "swarm-manager")
	if err != nil {
		httputil.InternalError(w, "[prompts] list-skills", "failed to load prompt skills")
		return
	}
	counts := bindingCountBySkill()
	items := make([]PromptSkillSummary, 0, len(skills))
	for _, skill := range skills {
		if !allowedSkillID(skill.ID) {
			continue
		}
		count := counts[skill.ID]
		items = append(items, PromptSkillSummary{
			ID:            skill.ID,
			Name:          skill.Name,
			Description:   skill.Description,
			DefaultScope:  skill.DefaultScope,
			Draft:         skill.Draft,
			CreatedAt:     skill.CreatedAt,
			UpdatedAt:     skill.UpdatedAt,
			TriggerCount:  count,
			ImpactSummary: fmt.Sprintf("Affects %d trigger path(s).", count),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	if err := httputil.JSON(w, map[string]any{"items": items}); err != nil {
		httputil.InternalError(w, "[prompts] list-skills", "failed to encode response")
	}
}

func (h *Handler) GetSkill(w http.ResponseWriter, r *http.Request) {
	skillID := strings.TrimSpace(mux.Vars(r)["id"])
	if !allowedSkillID(skillID) {
		httputil.BadRequest(w, "[prompts] get-skill", "only swarm-manager skill IDs are supported")
		return
	}
	skill, err := h.client.GetSkill(r.Context(), skillID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "status 404") {
			httputil.NotFound(w, "[prompts] get-skill", "prompt skill not found")
			return
		}
		httputil.InternalError(w, "[prompts] get-skill", "failed to load prompt skill")
		return
	}
	count := bindingCountBySkill()[skill.ID]
	resp := PromptSkillSummary{
		ID:              skill.ID,
		Name:            skill.Name,
		Description:     skill.Description,
		DefaultScope:    skill.DefaultScope,
		Draft:           skill.Draft,
		UpdatedAt:       skill.UpdatedAt,
		CreatedAt:       skill.CreatedAt,
		TriggerCount:    count,
		ImpactSummary:   fmt.Sprintf("Affects %d trigger path(s).", count),
		CurrentContent:  skill.Content,
		RequiredMissing: missingRequiredVariables(skill.ID, skill.Content),
	}
	if err := httputil.JSON(w, map[string]any{"item": resp}); err != nil {
		httputil.InternalError(w, "[prompts] get-skill", "failed to encode response")
	}
}

type updatePromptSkillRequest struct {
	Content      *string `json:"content,omitempty"`
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	DefaultScope *string `json:"default_scope,omitempty"`
	Draft        *bool   `json:"draft,omitempty"`
}

func (h *Handler) UpdateSkill(w http.ResponseWriter, r *http.Request) {
	skillID := strings.TrimSpace(mux.Vars(r)["id"])
	if !allowedSkillID(skillID) {
		httputil.BadRequest(w, "[prompts] update-skill", "only swarm-manager skill IDs are supported")
		return
	}

	var req updatePromptSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.BadRequest(w, "[prompts] update-skill", "invalid request body")
		return
	}

	if req.Content != nil {
		missing := missingRequiredVariables(skillID, *req.Content)
		if len(missing) > 0 {
			httputil.BadRequest(w, "[prompts] update-skill", "missing required template variables: "+strings.Join(missing, ", "))
			return
		}
	}

	patch := promptmanager.PromptSkillUpdate{
		Content:      req.Content,
		Name:         req.Name,
		Description:  req.Description,
		DefaultScope: req.DefaultScope,
		Draft:        req.Draft,
	}
	skill, err := h.client.UpdateSkill(r.Context(), skillID, patch)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "status 404") {
			httputil.NotFound(w, "[prompts] update-skill", "prompt skill not found")
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "status 400") ||
			strings.Contains(strings.ToLower(err.Error()), "status 403") {
			httputil.BadRequest(w, "[prompts] update-skill", "prompt-manager rejected skill update")
			return
		}
		httputil.InternalError(w, "[prompts] update-skill", "failed to update prompt skill")
		return
	}
	count := bindingCountBySkill()[skill.ID]
	resp := PromptSkillSummary{
		ID:              skill.ID,
		Name:            skill.Name,
		Description:     skill.Description,
		DefaultScope:    skill.DefaultScope,
		Draft:           skill.Draft,
		UpdatedAt:       skill.UpdatedAt,
		CreatedAt:       skill.CreatedAt,
		TriggerCount:    count,
		ImpactSummary:   fmt.Sprintf("Affects %d trigger path(s).", count),
		CurrentContent:  skill.Content,
		RequiredMissing: missingRequiredVariables(skill.ID, skill.Content),
	}
	if err := httputil.JSON(w, map[string]any{"item": resp}); err != nil {
		httputil.InternalError(w, "[prompts] update-skill", "failed to encode response")
	}
}

func (h *Handler) GetSkillVersions(w http.ResponseWriter, r *http.Request) {
	skillID := strings.TrimSpace(mux.Vars(r)["id"])
	if !allowedSkillID(skillID) {
		httputil.BadRequest(w, "[prompts] versions", "only swarm-manager skill IDs are supported")
		return
	}

	versions, err := h.client.GetSkillVersions(r.Context(), skillID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "status 404") {
			httputil.NotFound(w, "[prompts] versions", "prompt skill not found")
			return
		}
		httputil.InternalError(w, "[prompts] versions", "failed to load prompt versions")
		return
	}
	if err := httputil.JSON(w, versions); err != nil {
		httputil.InternalError(w, "[prompts] versions", "failed to encode response")
	}
}

func (h *Handler) RevertSkillVersion(w http.ResponseWriter, r *http.Request) {
	skillID := strings.TrimSpace(mux.Vars(r)["id"])
	if !allowedSkillID(skillID) {
		httputil.BadRequest(w, "[prompts] revert", "only swarm-manager skill IDs are supported")
		return
	}
	versionRaw := strings.TrimSpace(mux.Vars(r)["version"])
	version := 0
	if _, err := fmt.Sscanf(versionRaw, "%d", &version); err != nil || version <= 0 {
		httputil.BadRequest(w, "[prompts] revert", "version must be a positive integer")
		return
	}
	if err := h.client.RevertSkillVersion(r.Context(), skillID, version); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "status 404") {
			httputil.NotFound(w, "[prompts] revert", "version or prompt skill not found")
			return
		}
		httputil.InternalError(w, "[prompts] revert", "failed to revert prompt skill")
		return
	}
	skill, err := h.client.GetSkill(r.Context(), skillID)
	if err != nil {
		httputil.InternalError(w, "[prompts] revert", "revert applied but failed to reload prompt skill")
		return
	}
	count := bindingCountBySkill()[skill.ID]
	resp := PromptSkillSummary{
		ID:              skill.ID,
		Name:            skill.Name,
		Description:     skill.Description,
		DefaultScope:    skill.DefaultScope,
		Draft:           skill.Draft,
		UpdatedAt:       skill.UpdatedAt,
		CreatedAt:       skill.CreatedAt,
		TriggerCount:    count,
		ImpactSummary:   fmt.Sprintf("Affects %d trigger path(s).", count),
		CurrentContent:  skill.Content,
		RequiredMissing: missingRequiredVariables(skill.ID, skill.Content),
	}
	if err := httputil.JSON(w, map[string]any{"item": resp}); err != nil {
		httputil.InternalError(w, "[prompts] revert", "failed to encode response")
	}
}

type previewRequest struct {
	SkillID   string            `json:"skill_id"`
	Variables map[string]string `json:"variables"`
	WithScope *bool             `json:"with_scope,omitempty"`
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	var req previewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.BadRequest(w, "[prompts] preview", "invalid request body")
		return
	}
	skillID := strings.TrimSpace(req.SkillID)
	if !allowedSkillID(skillID) {
		httputil.BadRequest(w, "[prompts] preview", "only swarm-manager skill IDs are supported")
		return
	}
	withScope := false
	if req.WithScope != nil {
		withScope = *req.WithScope
	}
	rendered, err := h.client.ReadSkill(r.Context(), skillID, req.Variables, withScope)
	if err != nil {
		httputil.InternalError(w, "[prompts] preview", "failed to render prompt")
		return
	}
	if err := httputil.JSON(w, map[string]any{
		"skill_id":   skillID,
		"with_scope": withScope,
		"variables":  req.Variables,
		"prompt":     rendered,
	}); err != nil {
		httputil.InternalError(w, "[prompts] preview", "failed to encode response")
	}
}

type simulateRequest struct {
	Kind            string            `json:"kind"`
	Mode            string            `json:"mode,omitempty"`
	Operation       string            `json:"operation,omitempty"`
	ItemName        string            `json:"item_name,omitempty"`
	ItemTitle       string            `json:"item_title,omitempty"`
	ItemDescription string            `json:"item_description,omitempty"`
	ItemStatus      string            `json:"item_status,omitempty"`
	ItemPriority    string            `json:"item_priority,omitempty"`
	ItemTags        string            `json:"item_tags,omitempty"`
	ItemFolder      string            `json:"item_folder,omitempty"`
	Variables       map[string]string `json:"variables,omitempty"`
}

func resolveResearchSkill(mode, kind string) string {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	switch normalizedMode {
	case "workshop":
		return "swarm-manager-workshop"
	default:
		switch normalizedKind {
		case "idea":
			return "swarm-manager-research-idea"
		case "fix":
			return "swarm-manager-research-fix"
		case "execute", "research", "chore":
			return "swarm-manager-research-general"
		default:
			return "swarm-manager-research-general"
		}
	}
}

func defaultVariables(req simulateRequest) map[string]string {
	vars := map[string]string{
		"ITEM_NAME":        strings.TrimSpace(req.ItemName),
		"ITEM_TITLE":       strings.TrimSpace(req.ItemTitle),
		"ITEM_DESCRIPTION": strings.TrimSpace(req.ItemDescription),
		"ITEM_KIND":        strings.TrimSpace(req.Kind),
		"ITEM_STATUS":      strings.TrimSpace(req.ItemStatus),
		"ITEM_PRIORITY":    strings.TrimSpace(req.ItemPriority),
		"ITEM_TAGS":        strings.TrimSpace(req.ItemTags),
		"ITEM_FOLDER":      strings.TrimSpace(req.ItemFolder),
	}
	for key, value := range req.Variables {
		vars[key] = value
	}
	return vars
}

func (h *Handler) Simulate(w http.ResponseWriter, r *http.Request) {
	var req simulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.BadRequest(w, "[prompts] simulate", "invalid request body")
		return
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" {
		httputil.BadRequest(w, "[prompts] simulate", "kind is required")
		return
	}
	area := "research"
	if strings.ToLower(strings.TrimSpace(req.Operation)) != "" {
		area = "process"
	}
	if area == "process" {
		httputil.BadRequest(w, "[prompts] simulate", "process executions use plan.md directly — no skill template to simulate")
		return
	}
	skillID := resolveResearchSkill(req.Mode, kind)
	vars := defaultVariables(req)
	rendered, err := h.client.ReadSkill(r.Context(), skillID, vars, false)
	if err != nil {
		httputil.InternalError(w, "[prompts] simulate", "failed to resolve prompt")
		return
	}
	if err := httputil.JSON(w, map[string]any{
		"area":      area,
		"kind":      kind,
		"mode":      strings.ToLower(strings.TrimSpace(req.Mode)),
		"operation": strings.ToLower(strings.TrimSpace(req.Operation)),
		"skill_id":  skillID,
		"variables": vars,
		"prompt":    rendered,
	}); err != nil {
		httputil.InternalError(w, "[prompts] simulate", "failed to encode response")
	}
}
