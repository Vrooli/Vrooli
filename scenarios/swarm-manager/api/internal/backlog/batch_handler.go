// Batch operations for backlog items: atomic multi-item creation with
// dependency validation, initiative assignment, and rollback on failure.
package backlog

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/httputil"
)

// InitiativeAssigner abstracts initiative operations needed by batch create,
// avoiding a direct import of the initiatives package (which imports backlog).
type InitiativeAssigner interface {
	// Get loads the current state of an initiative for validation or rollback.
	Get(name string) (*InitiativeSnapshot, error)
	// Create persists a new initiative with explicit metadata.
	Create(spec InitiativeSpec) error
	// Update mutates initiative metadata without changing item membership.
	Update(spec InitiativeSpec) error
	// Replace restores an initiative snapshot, including item membership.
	Replace(snapshot InitiativeSnapshot) error
	// Delete removes an initiative entirely.
	Delete(name string) error
	// AddItems appends item references ("kind/name") to the named initiative.
	AddItems(name string, items []string) error
}

// InitiativeSpec describes the canonical metadata for an initiative.
type InitiativeSpec struct {
	Name        string
	Title       string
	Description string
	Status      string
}

// InitiativeSnapshot captures the full persisted state of an initiative.
type InitiativeSnapshot struct {
	Name        string
	Title       string
	Description string
	Status      string
	Items       []string
}

// SetInitiativeAssigner injects the initiative assigner for batch operations.
// Called from main.go after both packages are initialized.
func (h *Handler) SetInitiativeAssigner(ia InitiativeAssigner) {
	h.initiativeAssigner = ia
}

// batchCreateRequest is the JSON request body for batch-creating backlog items.
type batchCreateRequest struct {
	Items       []batchCreateItem       `json:"items"`
	Initiatives []batchCreateInitiative `json:"initiatives,omitempty"`
	Preview     bool                    `json:"preview,omitempty"`
}

// batchCreateItem mirrors the fields of a single backlog item creation request.
type batchCreateItem struct {
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	Kind            string   `json:"kind"`
	Priority        *int32   `json:"priority,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
	Initiative      string   `json:"initiative,omitempty"`
	Effort          *string  `json:"effort,omitempty"`
	AcceptanceAllow []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string `json:"acceptance_deny,omitempty"`
}

// batchCreateInitiative describes initiative metadata supplied with a batch import.
type batchCreateInitiative struct {
	Name        string  `json:"name"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// batchCreateInitiativeResult reports what the batch import will do or did for
// initiative metadata.
type batchCreateInitiativeResult struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Action      string `json:"action"`
}

type resolvedInitiativePlan struct {
	spec     InitiativeSpec
	existing *InitiativeSnapshot
	action   string
}

// batchCreateResponse is the JSON response for a successful batch create.
type batchCreateResponse struct {
	Items       []BacklogItem                 `json:"items"`
	Initiatives []batchCreateInitiativeResult `json:"initiatives,omitempty"`
	Count       int                           `json:"count"`
	Preview     bool                          `json:"preview,omitempty"`
	Warnings    []string                      `json:"warnings,omitempty"`
}

// BatchCreate creates multiple backlog items atomically.
// All items are validated before any are created. If any validation fails,
// no items are created. If initiative is specified, all items are assigned to it.
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req batchCreateRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("%s", "invalid request body: "+httputil.TruncateErrorMessage(err, 240)))
		return
	}

	if len(req.Items) == 0 {
		apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("at least one item is required"))
		return
	}

	const maxBatchSize = 100
	if len(req.Items) > maxBatchSize {
		apierr.MapError(w, "[backlog] batch-create",
			apierr.BadRequest("batch size %d exceeds maximum of %d", len(req.Items), maxBatchSize))
		return
	}

	// Phase 1: Validate all items and check for duplicate names within the batch.
	type validatedItem struct {
		item BacklogItem
		kind BacklogKind
	}
	validated := make([]validatedItem, 0, len(req.Items))
	batchNames := make(map[string]bool, len(req.Items))
	referencedInitiatives := make(map[string]bool)

	now := time.Now().UTC().Format(time.RFC3339)
	providedInitiatives := make(map[string]batchCreateInitiative, len(req.Initiatives))
	for i, raw := range req.Initiatives {
		name := strings.TrimSpace(raw.Name)
		if name == "" {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("initiatives[%d]: name is required", i))
			return
		}
		if strings.TrimSpace(raw.Title) == "" {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("initiatives[%d]: title is required", i))
			return
		}
		if raw.Status != nil {
			status := strings.TrimSpace(*raw.Status)
			if !isValidInitiativeStatus(status) {
				apierr.MapError(w, "[backlog] batch-create",
					apierr.BadRequest("initiatives[%d]: status must be active, completed, or archived", i))
				return
			}
		}
		if _, exists := providedInitiatives[name]; exists {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("initiatives[%d]: duplicate initiative %q", i, name))
			return
		}
		raw.Name = name
		raw.Title = strings.TrimSpace(raw.Title)
		if raw.Description != nil {
			description := strings.TrimSpace(*raw.Description)
			raw.Description = &description
		}
		if raw.Status != nil {
			status := strings.TrimSpace(*raw.Status)
			raw.Status = &status
		}
		providedInitiatives[name] = raw
	}

	for i, raw := range req.Items {
		// Validate kind.
		if strings.TrimSpace(raw.Kind) == "" {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: kind is required", i))
			return
		}
		kind, err := ParseBacklogKind(raw.Kind)
		if err != nil {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: %s", i, err.Error()))
			return
		}

		// Validate title.
		if strings.TrimSpace(raw.Title) == "" {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: title is required", i))
			return
		}

		// Sanitize name.
		name := strings.TrimSpace(raw.Name)
		if name == "" {
			name = raw.Title
		}
		name = sanitizeName(name)
		if name == "" {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: name is required", i))
			return
		}

		// Check for duplicate names within the batch (using kind/name key).
		key := string(kind) + "/" + name
		if batchNames[key] {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: duplicate item %q in batch", i, key))
			return
		}
		batchNames[key] = true

		// Validate priority.
		priority := 5
		if raw.Priority != nil {
			if *raw.Priority < 1 || *raw.Priority > 10 {
				apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: priority must be between 1 and 10", i))
				return
			}
			priority = int(*raw.Priority)
		}

		// Check for conflicts with existing items on disk.
		itemDir := h.store.ItemDir(kind, name)
		if _, err := os.Stat(itemDir); err == nil {
			apierr.MapError(w, "[backlog] batch-create", apierr.Conflict("item[%d]: %q already exists", i, key))
			return
		}

		tags := raw.Tags
		if tags == nil {
			tags = []string{}
		}

		dependsOn := raw.DependsOn
		if dependsOn == nil {
			dependsOn = []string{}
		}
		initiativeName := strings.TrimSpace(raw.Initiative)
		if initiativeName != "" {
			referencedInitiatives[initiativeName] = true
		}

		effort := ""
		if raw.Effort != nil {
			normalized, err := validateEffort(*raw.Effort)
			if err != nil {
				apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: %s", i, err.Error()))
				return
			}
			effort = normalized
		}

		if err := validateGlobs(raw.AcceptanceAllow); err != nil {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: acceptance_allow: %s", i, err.Error()))
			return
		}
		if err := validateGlobs(raw.AcceptanceDeny); err != nil {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: acceptance_deny: %s", i, err.Error()))
			return
		}

		item := BacklogItem{
			Name:            name,
			Title:           raw.Title,
			Description:     strings.TrimSpace(raw.Description),
			Status:          StatusBacklog,
			Priority:        priority,
			Tags:            tags,
			Created:         now,
			Updated:         now,
			Kind:            kind,
			DependsOn:       dependsOn,
			Initiative:      initiativeName,
			Effort:          effort,
			AcceptanceAllow: raw.AcceptanceAllow,
			AcceptanceDeny:  raw.AcceptanceDeny,
		}

		validated = append(validated, validatedItem{item: item, kind: kind})
	}

	for name := range providedInitiatives {
		if !referencedInitiatives[name] {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("initiative %q is not referenced by any item", name))
			return
		}
	}

	var (
		initiativePlans   map[string]resolvedInitiativePlan
		initiativeResults []batchCreateInitiativeResult
		err               error
	)
	if len(referencedInitiatives) > 0 {
		if h.initiativeAssigner == nil {
			apierr.MapError(w, "[backlog] batch-create", apierr.Internal("initiative support not configured"))
			return
		}
		initiativePlans, initiativeResults, err = h.resolveInitiativePlans(referencedInitiatives, providedInitiatives)
		if err != nil {
			apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("%s", err.Error()))
			return
		}
	}

	// Phase 2: Validate dependencies — references must exist either in the
	// batch or already on disk.
	for i, v := range validated {
		for _, ref := range v.item.DependsOn {
			if batchNames[ref] {
				continue // dependency is within this batch
			}
			// Must exist on disk.
			depKind, depName, err := parseDependencyRef(ref)
			if err != nil {
				apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: %s", i, err.Error()))
				return
			}
			if _, loadErr := h.store.LoadItem(depKind, depName); errors.Is(loadErr, ErrNotFound) {
				apierr.MapError(w, "[backlog] batch-create", apierr.BadRequest("item[%d]: dependency %q does not exist", i, ref))
				return
			}
		}
	}

	// Phase 3: Build dependency graph (batch + existing items) and check for cycles.
	existingItems, err := h.store.LoadAll(nil)
	if err != nil {
		slog.Error("failed to load existing items", "err", err)
		apierr.MapError(w, "[backlog] batch-create", apierr.Internal("failed to load existing items for cycle check"))
		return
	}

	g := depgraph.New()
	for _, v := range validated {
		key := string(v.item.Kind) + "/" + v.item.Name
		g.AddNode(key, v.item.DependsOn)
	}
	for _, existing := range existingItems {
		key := string(existing.Kind) + "/" + existing.Name
		g.AddNode(key, existing.DependsOn)
	}
	if cycle, found := g.DetectCycle(); found {
		apierr.MapError(w, "[backlog] batch-create",
			apierr.BadRequest("dependency cycle detected: %s", strings.Join(cycle, " -> ")))
		return
	}

	if req.Preview {
		previewItems := make([]BacklogItem, 0, len(validated))
		for _, validatedItem := range validated {
			previewItems = append(previewItems, validatedItem.item)
		}
		resp := batchCreateResponse{
			Items:       previewItems,
			Initiatives: initiativeResults,
			Count:       len(validated),
			Preview:     true,
		}
		if err := httputil.JSON(w, resp); err != nil {
			apierr.MapError(w, "[backlog] batch-create", apierr.Internal("failed to encode response"))
		}
		return
	}

	// Phase 4: Apply initiative metadata changes before item creation so the full
	// import can roll back to a clean state if any later step fails.
	appliedInitiatives := make([]resolvedInitiativePlan, 0, len(initiativePlans))
	for _, name := range orderedInitiativeNames(initiativePlans) {
		plan := initiativePlans[name]
		switch plan.action {
		case "create":
			if createErr := h.initiativeAssigner.Create(plan.spec); createErr != nil {
				slog.Error("failed to create initiative", "initiative", name, "err", createErr)
				apierr.MapError(w, "[backlog] batch-create", apierr.Internal("%s", "failed to create initiative: "+httputil.TruncateErrorMessage(createErr, 240)))
				return
			}
		case "update":
			if updateErr := h.initiativeAssigner.Update(plan.spec); updateErr != nil {
				slog.Error("failed to update initiative", "initiative", name, "err", updateErr)
				apierr.MapError(w, "[backlog] batch-create", apierr.Internal("%s", "failed to update initiative: "+httputil.TruncateErrorMessage(updateErr, 240)))
				return
			}
		}
		appliedInitiatives = append(appliedInitiatives, plan)
	}

	// Phase 5: Create all items atomically. Track created directories for rollback.
	createdDirs := make([]string, 0, len(validated))
	createdItems := make([]BacklogItem, 0, len(validated))

	for _, v := range validated {
		itemDir := h.store.ItemDir(v.kind, v.item.Name)
		if mkErr := os.MkdirAll(itemDir, 0o755); mkErr != nil {
			rollbackBatchCreate(createdDirs, appliedInitiatives, h.initiativeAssigner)
			slog.Error("failed to create directory", "item", v.item.Name, "err", mkErr)
			apierr.MapError(w, "[backlog] batch-create", apierr.Internal("failed to create item directory"))
			return
		}
		createdDirs = append(createdDirs, itemDir)

		if saveErr := h.store.SaveItem(v.item); saveErr != nil {
			rollbackBatchCreate(createdDirs, appliedInitiatives, h.initiativeAssigner)
			slog.Error("failed to save item", "item", v.item.Name, "err", saveErr)
			apierr.MapError(w, "[backlog] batch-create", apierr.Internal("failed to save item"))
			return
		}

		createdItems = append(createdItems, v.item)
	}

	// Phase 6: Add new items to each initiative. This is part of the atomic
	// import contract, so failures trigger a full rollback.
	for name, refs := range groupItemRefsByInitiative(createdItems) {
		if addErr := h.initiativeAssigner.AddItems(name, refs); addErr != nil {
			rollbackBatchCreate(createdDirs, appliedInitiatives, h.initiativeAssigner)
			slog.Error("failed to add items to initiative", "initiative", name, "err", addErr)
			apierr.MapError(w, "[backlog] batch-create", apierr.Internal("%s", "failed to assign items to initiative: "+httputil.TruncateErrorMessage(addErr, 240)))
			return
		}
	}

	// Phase 7: Auto-trigger workshops for items whose dependencies are met.
	for _, item := range createdItems {
		h.maybeAutoWorkshop(item, false)
	}

	slog.Info("batch-created items", "count", len(createdItems))
	h.invalidateAllGraphLenses()

	resp := batchCreateResponse{
		Items:       createdItems,
		Initiatives: initiativeResults,
		Count:       len(createdItems),
	}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[backlog] batch-create", apierr.Internal("failed to encode response"))
	}
}
