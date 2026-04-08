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

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/promptmanager"
)

type Handler struct {
	rootDir          string
	client           promptmanager.AdminClient
	experimentClient promptmanager.ExperimentClient
}

func NewHandler(rootDir string, client promptmanager.AdminClient) *Handler {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = pathutil.ResolveScenarioRoot("swarm-manager")
	}
	if client == nil {
		client = promptmanager.NewHTTPClient()
	}
	h := &Handler{
		rootDir: rootDir,
		client:  client,
	}
	// If the admin client also implements ExperimentClient, use it.
	if ec, ok := client.(promptmanager.ExperimentClient); ok {
		h.experimentClient = ec
	}
	return h
}

// SetExperimentClient injects an experiment client for outcome and analysis operations.
func (h *Handler) SetExperimentClient(ec promptmanager.ExperimentClient) {
	h.experimentClient = ec
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/prompts/catalog", h.Catalog).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prompts/skills", h.ListSkills).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prompts/skills/{id}", h.GetSkill).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prompts/skills/{id}", h.UpdateSkill).Methods(http.MethodPut)
	r.HandleFunc("/api/v1/prompts/skills/{id}/versions", h.GetSkillVersions).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/prompts/skills/{id}/revert/{version}", h.RevertSkillVersion).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/prompts/preview", h.Preview).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/prompts/simulate", h.Simulate).Methods(http.MethodPost)

	// Experiment results (uses the same prompt-manager client as an ExperimentClient).
	expHandler := NewExperimentHandler(h.experimentClient)
	expHandler.RegisterRoutes(r)
}

type PromptSkillSummary struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	DefaultScope    string   `json:"default_scope,omitempty"`
	Draft           bool     `json:"draft"`
	UpdatedAt       string   `json:"updated_at,omitempty"`
	CreatedAt       string   `json:"created_at,omitempty"`
	UsageType       string   `json:"usage_type"`
	Groups          []string `json:"groups,omitempty"`
	TriggerCount    int      `json:"trigger_count"`
	ImpactSummary   string   `json:"impact_summary"`
	CurrentContent  string   `json:"current_content,omitempty"`
	RequiredMissing []string `json:"required_missing,omitempty"`
}

func allowedSkillID(skillID string) bool {
	return promptcatalog.IsKnownSkillID(skillID)
}

// classifyClientError maps prompt-manager HTTP error status codes to appropriate
// API error responses.
func classifyClientError(err error, ctx, notFoundMsg, fallbackMsg string) error {
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "status 404") {
		return apierr.NotFound("%s", notFoundMsg)
	}
	if strings.Contains(lower, "status 400") || strings.Contains(lower, "status 403") {
		return apierr.BadRequest("prompt-manager rejected request")
	}
	return apierr.Internal("%s", fallbackMsg)
}

func (h *Handler) Catalog(w http.ResponseWriter, _ *http.Request) {
	items := promptcatalog.Entries()
	if err := httputil.JSON(w, map[string]any{"items": items}); err != nil {
		apierr.MapError(w, "[prompts] catalog", apierr.Internal("failed to encode response"))
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
	required := promptcatalog.VariableKeysForSkill(skillID)
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

func buildSkillSummary(skill promptmanager.PromptSkill) PromptSkillSummary {
	return PromptSkillSummary{
		ID:              skill.ID,
		Name:            skill.Name,
		Description:     skill.Description,
		DefaultScope:    skill.DefaultScope,
		Draft:           skill.Draft,
		UpdatedAt:       skill.UpdatedAt,
		CreatedAt:       skill.CreatedAt,
		UsageType:       string(promptcatalog.SkillUsageType(skill.ID)),
		Groups:          promptcatalog.SkillGroups(skill.ID),
		TriggerCount:    promptcatalog.SkillUsageCount(skill.ID),
		ImpactSummary:   promptcatalog.SkillImpactSummary(skill.ID),
		CurrentContent:  skill.Content,
		RequiredMissing: missingRequiredVariables(skill.ID, skill.Content),
	}
}

func (h *Handler) ListSkills(w http.ResponseWriter, r *http.Request) {
	catalogSkills := promptcatalog.SkillEntries()
	items := make([]PromptSkillSummary, 0, len(catalogSkills))
	for _, entry := range catalogSkills {
		skill, err := h.client.GetSkill(r.Context(), entry.SkillID)
		if err != nil {
			apierr.MapError(w, "[prompts] list-skills",
				classifyClientError(err, "[prompts] list-skills", "prompt skill not found: "+entry.SkillID, "failed to load prompt skills"))
			return
		}
		items = append(items, buildSkillSummary(skill))
	}
	if err := httputil.JSON(w, map[string]any{"items": items}); err != nil {
		apierr.MapError(w, "[prompts] list-skills", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) GetSkill(w http.ResponseWriter, r *http.Request) {
	skillID := strings.TrimSpace(mux.Vars(r)["id"])
	if !allowedSkillID(skillID) {
		apierr.MapError(w, "[prompts] get-skill", apierr.BadRequest("skill is not part of the prompt catalog"))
		return
	}
	skill, err := h.client.GetSkill(r.Context(), skillID)
	if err != nil {
		apierr.MapError(w, "[prompts] get-skill",
			classifyClientError(err, "[prompts] get-skill", "prompt skill not found", "failed to load prompt skill"))
		return
	}
	resp := buildSkillSummary(skill)
	if err := httputil.JSON(w, map[string]any{"item": resp}); err != nil {
		apierr.MapError(w, "[prompts] get-skill", apierr.Internal("failed to encode response"))
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
		apierr.MapError(w, "[prompts] update-skill", apierr.BadRequest("skill is not part of the prompt catalog"))
		return
	}

	var req updatePromptSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[prompts] update-skill", apierr.BadRequest("invalid request body"))
		return
	}

	if req.Content != nil {
		missing := missingRequiredVariables(skillID, *req.Content)
		if len(missing) > 0 {
			apierr.MapError(w, "[prompts] update-skill", apierr.BadRequest("missing required template variables: %s", strings.Join(missing, ", ")))
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
		apierr.MapError(w, "[prompts] update-skill",
			classifyClientError(err, "[prompts] update-skill", "prompt skill not found", "failed to update prompt skill"))
		return
	}
	resp := buildSkillSummary(skill)
	if err := httputil.JSON(w, map[string]any{"item": resp}); err != nil {
		apierr.MapError(w, "[prompts] update-skill", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) GetSkillVersions(w http.ResponseWriter, r *http.Request) {
	skillID := strings.TrimSpace(mux.Vars(r)["id"])
	if !allowedSkillID(skillID) {
		apierr.MapError(w, "[prompts] versions", apierr.BadRequest("skill is not part of the prompt catalog"))
		return
	}

	versions, err := h.client.GetSkillVersions(r.Context(), skillID)
	if err != nil {
		apierr.MapError(w, "[prompts] versions",
			classifyClientError(err, "[prompts] versions", "prompt skill not found", "failed to load prompt versions"))
		return
	}
	if err := httputil.JSON(w, versions); err != nil {
		apierr.MapError(w, "[prompts] versions", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) RevertSkillVersion(w http.ResponseWriter, r *http.Request) {
	skillID := strings.TrimSpace(mux.Vars(r)["id"])
	if !allowedSkillID(skillID) {
		apierr.MapError(w, "[prompts] revert", apierr.BadRequest("skill is not part of the prompt catalog"))
		return
	}
	versionRaw := strings.TrimSpace(mux.Vars(r)["version"])
	version := 0
	if _, err := fmt.Sscanf(versionRaw, "%d", &version); err != nil || version <= 0 {
		apierr.MapError(w, "[prompts] revert", apierr.BadRequest("version must be a positive integer"))
		return
	}
	if err := h.client.RevertSkillVersion(r.Context(), skillID, version); err != nil {
		apierr.MapError(w, "[prompts] revert",
			classifyClientError(err, "[prompts] revert", "version or prompt skill not found", "failed to revert prompt skill"))
		return
	}
	skill, err := h.client.GetSkill(r.Context(), skillID)
	if err != nil {
		apierr.MapError(w, "[prompts] revert", apierr.Internal("revert applied but failed to reload prompt skill"))
		return
	}
	resp := buildSkillSummary(skill)
	if err := httputil.JSON(w, map[string]any{"item": resp}); err != nil {
		apierr.MapError(w, "[prompts] revert", apierr.Internal("failed to encode response"))
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
		apierr.MapError(w, "[prompts] preview", apierr.BadRequest("invalid request body"))
		return
	}
	skillID := strings.TrimSpace(req.SkillID)
	if !allowedSkillID(skillID) {
		apierr.MapError(w, "[prompts] preview", apierr.BadRequest("skill is not part of the prompt catalog"))
		return
	}
	withScope := false
	if req.WithScope != nil {
		withScope = *req.WithScope
	}
	rendered, err := h.client.ReadSkill(r.Context(), skillID, req.Variables, withScope)
	if err != nil {
		apierr.MapError(w, "[prompts] preview", apierr.Internal("failed to render prompt"))
		return
	}
	if err := httputil.JSON(w, map[string]any{
		"skill_id":   skillID,
		"with_scope": withScope,
		"variables":  req.Variables,
		"prompt":     rendered,
	}); err != nil {
		apierr.MapError(w, "[prompts] preview", apierr.Internal("failed to encode response"))
	}
}

type simulateRequest struct {
	Kind            string            `json:"kind"`
	Mode            string            `json:"mode,omitempty"`
	ItemName        string            `json:"item_name,omitempty"`
	ItemTitle       string            `json:"item_title,omitempty"`
	ItemDescription string            `json:"item_description,omitempty"`
	ItemStatus      string            `json:"item_status,omitempty"`
	ItemPriority    string            `json:"item_priority,omitempty"`
	ItemTags        string            `json:"item_tags,omitempty"`
	ItemFolder      string            `json:"item_folder,omitempty"`
	Variables       map[string]string `json:"variables,omitempty"`
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
		apierr.MapError(w, "[prompts] simulate", apierr.BadRequest("invalid request body"))
		return
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" {
		apierr.MapError(w, "[prompts] simulate", apierr.BadRequest("kind is required"))
		return
	}
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "workshop"
	}
	entry, ok := promptcatalog.ResolveBacklogSkill(mode, kind)
	if !ok {
		apierr.MapError(w, "[prompts] simulate", apierr.BadRequest("mode must be workshop, initialize, or finalize for the selected kind"))
		return
	}
	vars := defaultVariables(req)
	rendered, err := h.client.ReadSkill(r.Context(), entry.SkillID, vars, false)
	if err != nil {
		apierr.MapError(w, "[prompts] simulate", apierr.Internal("failed to resolve prompt"))
		return
	}
	if err := httputil.JSON(w, map[string]any{
		"entry_id":   entry.ID,
		"group":      entry.Group,
		"usage_type": entry.UsageType,
		"kind":       kind,
		"mode":       mode,
		"skill_id":   entry.SkillID,
		"variables":  vars,
		"prompt":     rendered,
	}); err != nil {
		apierr.MapError(w, "[prompts] simulate", apierr.Internal("failed to encode response"))
	}
}
