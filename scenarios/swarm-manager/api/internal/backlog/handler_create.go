package backlog

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/workshop"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func validateCreateBacklogItemRequest(req *apipb.CreateBacklogItemRequest) string {
	if strings.TrimSpace(req.Title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(req.Kind) == "" {
		return "kind is required"
	}
	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 10 {
			return "priority must be between 1 and 10"
		}
	}
	return ""
}

func normalizeCreateBacklogItemRequest(req *apipb.CreateBacklogItemRequest) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = strings.TrimSpace(req.Title)
	}
	req.Kind = strings.ToLower(strings.TrimSpace(req.Kind))
	if req.Effort != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*req.Effort))
		if normalized == "" {
			req.Effort = nil
		} else {
			req.Effort = &normalized
		}
	}
}

// Create creates a new backlog item.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req apipb.CreateBacklogItemRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("invalid request body"))
		return
	}
	normalizeCreateBacklogItemRequest(&req)
	if !httputil.ValidateProtoRequest(w, "[backlog] create", "invalid request body", &req) {
		return
	}
	if validationErr := validateCreateBacklogItemRequest(&req); validationErr != "" {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", validationErr))
		return
	}

	kind, err := ParseBacklogKind(req.Kind)
	if err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", err.Error()))
		return
	}

	// Sanitize name (folder-safe). Allow title fallback.
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = req.Title
	}
	name = sanitizeName(name)
	if name == "" {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("name is required"))
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	priority := 5
	if req.Priority != nil {
		priority = int(*req.Priority)
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	dependsOn := req.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}
	initiative := ""
	if req.Initiative != nil {
		initiative = strings.TrimSpace(*req.Initiative)
	}
	if err := h.validateInitiativeReference(initiative); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", err.Error()))
		return
	}

	effort := ""
	if req.Effort != nil {
		normalized, err := validateEffort(*req.Effort)
		if err != nil {
			apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", err.Error()))
			return
		}
		effort = normalized
	}

	if err := validateGlobs(req.AcceptanceAllow); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", "acceptance_allow: "+err.Error()))
		return
	}
	if err := validateGlobs(req.AcceptanceDeny); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", "acceptance_deny: "+err.Error()))
		return
	}

	spawnedFrom := ""
	if req.SpawnedFrom != nil {
		spawnedFrom = strings.TrimSpace(*req.SpawnedFrom)
	}
	note := ""
	if req.Note != nil {
		note = strings.TrimSpace(*req.Note)
	}

	prov := identity.FromContext(r.Context())
	item := BacklogItem{
		Name:            name,
		Title:           req.Title,
		Description:     description,
		Status:          StatusBacklog,
		Priority:        priority,
		Tags:            tags,
		Created:         now,
		Updated:         now,
		Kind:            kind,
		DependsOn:       dependsOn,
		Initiative:      initiative,
		Effort:          effort,
		AcceptanceAllow: req.AcceptanceAllow,
		AcceptanceDeny:  req.AcceptanceDeny,
		SpawnedFrom:     spawnedFrom,
		Note:            note,
		CreatedBy:       &prov,
	}

	itemDir := h.store.ItemDir(kind, name)
	if err := os.Mkdir(itemDir, 0o755); err != nil {
		if os.IsExist(err) {
			apierr.MapError(w, "[backlog] create", apierr.Conflict("backlog item already exists"))
			return
		}
		// Parent dir may not exist for the first item of this kind — ensure it, then retry.
		if mkErr := os.MkdirAll(filepath.Dir(itemDir), 0o755); mkErr != nil {
			slog.Error("failed to create parent directory", "name", name, "err", mkErr)
			apierr.MapError(w, "[backlog] create", apierr.Internal("failed to create backlog directory"))
			return
		}
		if retryErr := os.Mkdir(itemDir, 0o755); retryErr != nil {
			if os.IsExist(retryErr) {
				apierr.MapError(w, "[backlog] create", apierr.Conflict("backlog item already exists"))
				return
			}
			slog.Error("failed to create directory", "name", name, "err", retryErr)
			apierr.MapError(w, "[backlog] create", apierr.Internal("failed to create backlog directory"))
			return
		}
	}

	// Validate dependencies exist and check for cycles.
	if len(item.DependsOn) > 0 {
		if err := h.store.ValidateDependencies(item.DependsOn); err != nil {
			_ = os.RemoveAll(itemDir)
			apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", err.Error()))
			return
		}
		if err := h.checkDependencyCycles(item); err != nil {
			_ = os.RemoveAll(itemDir)
			apierr.MapError(w, "[backlog] create", apierr.BadRequest("%s", err.Error()))
			return
		}
	}

	if err := h.store.SaveItem(item); err != nil {
		_ = os.RemoveAll(itemDir)
		slog.Error("failed to save item", "name", name, "err", err)
		apierr.MapError(w, "[backlog] create", apierr.Internal("failed to save backlog item"))
		return
	}

	// Auto-initialize workshop for new items (unless disabled in settings or blocked by deps).
	h.maybeAutoWorkshop(item, false)

	slog.Info("item created", "name", name, "kind", kind, "priority", priority, "status", StatusBacklog)
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogCreated(string(kind)+"/"+name, string(kind), string(StatusBacklog), priority, item.Initiative, item.Effort)
	}
	h.invalidateAllGraphLenses()
	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[backlog] create", apierr.Internal("failed to encode response"))
	}
}

// maybeAutoWorkshop checks the global auto_initialize_workshop setting,
// dependency readiness, and spawns the first workshop round asynchronously
// if appropriate.
func (h *Handler) maybeAutoWorkshop(item BacklogItem, forceOverride bool) {
	cfg, err := settings.NewStore("").Load()
	if err != nil {
		slog.Warn("auto-workshop settings load error, using defaults", "kind", item.Kind, "name", item.Name, "err", err)
		cfg = settings.DefaultSettings()
	}
	if !workshop.ShouldAutoInitialize(cfg.AutoInitializeWorkshop) {
		return
	}
	if !forceOverride && len(item.DependsOn) > 0 {
		reasons, err := EvaluateDependencyBlocking(item, h.store)
		if err != nil {
			slog.Warn("auto-workshop dep check error, proceeding anyway", "kind", item.Kind, "name", item.Name, "err", err)
		} else if len(reasons) > 0 {
			slog.Info("auto-workshop blocked by deps", "kind", item.Kind, "name", item.Name, "reasons", reasons)
			return
		}
	}
	go func() {
		_, _, spawnErr := h.spawnWorkshopAsync(item, ResearchModeInitialize)
		if spawnErr != nil {
			slog.Error("auto-init failed", "kind", item.Kind, "name", item.Name, "err", spawnErr)
		}
	}()
}

// cascadeWorkshopTrigger finds items that depend on the given item and
// auto-triggers their workshops if all their dependencies are now met.
// Only triggers for items still in "backlog" status with no existing
// workshop rounds.
func (h *Handler) cascadeWorkshopTrigger(readyItem BacklogItem) {
	readyKey := string(readyItem.Kind) + "/" + readyItem.Name

	allItems, err := h.store.LoadAll(nil)
	if err != nil {
		slog.Error("cascade failed to load items", "err", err)
		return
	}

	for _, item := range allItems {
		if item.Status != StatusBacklog {
			continue
		}
		dependsOnReady := false
		for _, dep := range item.DependsOn {
			if dep == readyKey {
				dependsOnReady = true
				break
			}
		}
		if !dependsOnReady {
			continue
		}

		reasons, err := EvaluateDependencyBlocking(item, h.store)
		if err != nil {
			slog.Warn("cascade dep check failed", "kind", item.Kind, "name", item.Name, "err", err)
			continue
		}
		if len(reasons) > 0 {
			continue
		}

		itemDir := h.store.ItemDir(item.Kind, item.Name)
		_, roundCount, _ := workshop.LoadLatestRound(itemDir)
		if roundCount > 0 {
			continue
		}

		slog.Info("cascade triggering workshop", "kind", item.Kind, "name", item.Name, "unblocked_by", readyKey)
		go func(it BacklogItem) {
			_, _, spawnErr := h.spawnWorkshopAsync(it, ResearchModeInitialize)
			if spawnErr != nil {
				slog.Error("cascade workshop spawn failed", "kind", it.Kind, "name", it.Name, "err", spawnErr)
			}
		}(item)
	}
}
