package backlog

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/workshop"
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
	}

	itemDir := h.store.ItemDir(kind, name)
	if err := os.Mkdir(itemDir, 0o755); err != nil {
		if os.IsExist(err) {
			apierr.MapError(w, "[backlog] create", apierr.Conflict("backlog item already exists"))
			return
		}
		// Parent dir may not exist for the first item of this kind — ensure it, then retry.
		if mkErr := os.MkdirAll(filepath.Dir(itemDir), 0o755); mkErr != nil {
			log.Printf("[backlog] create: failed to create parent directory for %q: %v", name, mkErr)
			apierr.MapError(w, "[backlog] create", apierr.Internal("failed to create backlog directory"))
			return
		}
		if retryErr := os.Mkdir(itemDir, 0o755); retryErr != nil {
			if os.IsExist(retryErr) {
				apierr.MapError(w, "[backlog] create", apierr.Conflict("backlog item already exists"))
				return
			}
			log.Printf("[backlog] create: failed to create directory for %q: %v", name, retryErr)
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
		log.Printf("[backlog] create: failed to save %q: %v", name, err)
		apierr.MapError(w, "[backlog] create", apierr.Internal("failed to save backlog item"))
		return
	}

	// Auto-initialize workshop for new items (unless disabled in settings or blocked by deps).
	h.maybeAutoWorkshop(item, false)

	log.Printf("[backlog] created: %q (kind=%s, priority=%d, status=%s)", name, kind, priority, StatusBacklog)
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
		log.Printf("[backlog] auto-workshop: settings load error for %s/%s: %v, using defaults", item.Kind, item.Name, err)
		cfg = settings.DefaultSettings()
	}
	if !workshop.ShouldAutoInitialize(cfg.AutoInitializeWorkshop) {
		return
	}
	if !forceOverride && len(item.DependsOn) > 0 {
		depStatuses, err := h.store.CheckWorkshopDependencies(item.DependsOn)
		if err != nil {
			log.Printf("[backlog] auto-workshop: dep check error for %s/%s: %v, proceeding anyway", item.Kind, item.Name, err)
		} else {
			result := workshop.CheckWorkshopDependencies(depStatuses)
			if result.Blocked {
				log.Printf("[backlog] auto-workshop: blocked for %s/%s by deps: %v", item.Kind, item.Name, result.BlockingDeps)
				return
			}
		}
	}
	go func() {
		_, _, spawnErr := h.spawnWorkshopAsync(item, ResearchModeInitialize)
		if spawnErr != nil {
			log.Printf("[backlog] auto-init: failed for %s/%s: %v", item.Kind, item.Name, spawnErr)
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
		log.Printf("[backlog] cascade: failed to load items: %v", err)
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

		depStatuses, err := h.store.CheckWorkshopDependencies(item.DependsOn)
		if err != nil {
			log.Printf("[backlog] cascade: dep check failed for %s/%s: %v", item.Kind, item.Name, err)
			continue
		}
		result := workshop.CheckWorkshopDependencies(depStatuses)
		if result.Blocked {
			continue
		}

		itemDir := h.store.ItemDir(item.Kind, item.Name)
		_, roundCount, _ := workshop.LoadLatestRound(itemDir)
		if roundCount > 0 {
			continue
		}

		log.Printf("[backlog] cascade: triggering workshop for %s/%s (unblocked by %s)", item.Kind, item.Name, readyKey)
		go func(it BacklogItem) {
			_, _, spawnErr := h.spawnWorkshopAsync(it, ResearchModeInitialize)
			if spawnErr != nil {
				log.Printf("[backlog] cascade: failed for %s/%s: %v", it.Kind, it.Name, spawnErr)
			}
		}(item)
	}
}
