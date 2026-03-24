// Batch operations for backlog items: atomic multi-item creation with
// dependency validation, initiative assignment, and rollback on failure.
package backlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/httputil"
)

// InitiativeAssigner abstracts initiative operations needed by batch create,
// avoiding a direct import of the initiatives package (which imports backlog).
type InitiativeAssigner interface {
	// EnsureExists verifies the named initiative exists, creating it if not.
	EnsureExists(name string) error
	// AddItems appends item references ("kind/name") to the named initiative.
	AddItems(name string, items []string) error
}

// SetInitiativeAssigner injects the initiative assigner for batch operations.
// Called from main.go after both packages are initialized.
func (h *Handler) SetInitiativeAssigner(ia InitiativeAssigner) {
	h.initiativeAssigner = ia
}

// batchCreateRequest is the JSON request body for batch-creating backlog items.
type batchCreateRequest struct {
	Items      []batchCreateItem `json:"items"`
	Initiative string            `json:"initiative,omitempty"`
}

// batchCreateItem mirrors the fields of a single backlog item creation request.
type batchCreateItem struct {
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	Kind            string   `json:"kind"`
	Priority        *int32   `json:"priority,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	ResearchTarget  *string  `json:"research_target,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
	Effort          *string  `json:"effort,omitempty"`
	Scope           *string  `json:"scope,omitempty"`
	AcceptanceAllow []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string `json:"acceptance_deny,omitempty"`
}

// batchCreateResponse is the JSON response for a successful batch create.
type batchCreateResponse struct {
	Items      []BacklogItem `json:"items"`
	Initiative string        `json:"initiative,omitempty"`
	Count      int           `json:"count"`
	Warnings   []string      `json:"warnings,omitempty"`
}

// BatchCreate creates multiple backlog items atomically.
// All items are validated before any are created. If any validation fails,
// no items are created. If initiative is specified, all items are assigned to it.
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req batchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.BadRequest(w, "[backlog] batch-create", "invalid request body: "+httputil.TruncateErrorMessage(err, 240))
		return
	}

	if len(req.Items) == 0 {
		httputil.BadRequest(w, "[backlog] batch-create", "at least one item is required")
		return
	}

	const maxBatchSize = 100
	if len(req.Items) > maxBatchSize {
		httputil.BadRequest(w, "[backlog] batch-create",
			fmt.Sprintf("batch size %d exceeds maximum of %d", len(req.Items), maxBatchSize))
		return
	}

	// Phase 1: Validate all items and check for duplicate names within the batch.
	type validatedItem struct {
		item BacklogItem
		kind BacklogKind
	}
	validated := make([]validatedItem, 0, len(req.Items))
	batchNames := make(map[string]bool, len(req.Items))

	now := time.Now().UTC().Format(time.RFC3339)
	initiativeName := strings.TrimSpace(req.Initiative)

	for i, raw := range req.Items {
		// Validate kind.
		if strings.TrimSpace(raw.Kind) == "" {
			httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: kind is required", i))
			return
		}
		kind, err := ParseBacklogKind(raw.Kind)
		if err != nil {
			httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: %s", i, err.Error()))
			return
		}

		// Validate title.
		if strings.TrimSpace(raw.Title) == "" {
			httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: title is required", i))
			return
		}

		// Sanitize name.
		name := strings.TrimSpace(raw.Name)
		if name == "" {
			name = raw.Title
		}
		name = sanitizeName(name)
		if name == "" {
			httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: name is required", i))
			return
		}

		// Check for duplicate names within the batch (using kind/name key).
		key := string(kind) + "/" + name
		if batchNames[key] {
			httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: duplicate item %q in batch", i, key))
			return
		}
		batchNames[key] = true

		// Validate priority.
		priority := 5
		if raw.Priority != nil {
			if *raw.Priority < 1 || *raw.Priority > 10 {
				httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: priority must be between 1 and 10", i))
				return
			}
			priority = int(*raw.Priority)
		}

		// Check for conflicts with existing items on disk.
		itemDir := h.store.ItemDir(kind, name)
		if _, err := os.Stat(itemDir); err == nil {
			httputil.Conflict(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: %q already exists", i, key))
			return
		}

		tags := raw.Tags
		if tags == nil {
			tags = []string{}
		}

		researchTarget := ""
		if raw.ResearchTarget != nil && kind == KindResearch {
			normalized, err := normalizeResearchTarget(*raw.ResearchTarget)
			if err != nil {
				httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: %s", i, err.Error()))
				return
			}
			researchTarget = normalized
		}

		dependsOn := raw.DependsOn
		if dependsOn == nil {
			dependsOn = []string{}
		}

		effort := ""
		if raw.Effort != nil {
			normalized, err := validateEffort(*raw.Effort)
			if err != nil {
				httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: %s", i, err.Error()))
				return
			}
			effort = normalized
		}

		scope := ""
		if raw.Scope != nil {
			scope = strings.TrimSpace(*raw.Scope)
			if err := validateScope(scope); err != nil {
				httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: %s", i, err.Error()))
				return
			}
		}
		if err := validateGlobs(raw.AcceptanceAllow); err != nil {
			httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: acceptance_allow: %s", i, err.Error()))
			return
		}
		if err := validateGlobs(raw.AcceptanceDeny); err != nil {
			httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: acceptance_deny: %s", i, err.Error()))
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
			ResearchTarget:  researchTarget,
			DependsOn:       dependsOn,
			Initiative:      initiativeName,
			Effort:          effort,
			Scope:           scope,
			AcceptanceAllow: raw.AcceptanceAllow,
			AcceptanceDeny:  raw.AcceptanceDeny,
		}

		validated = append(validated, validatedItem{item: item, kind: kind})
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
				httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: %s", i, err.Error()))
				return
			}
			if _, loadErr := h.store.LoadItem(depKind, depName); errors.Is(loadErr, ErrNotFound) {
				httputil.BadRequest(w, "[backlog] batch-create", fmt.Sprintf("item[%d]: dependency %q does not exist", i, ref))
				return
			}
		}
	}

	// Phase 3: Build dependency graph (batch + existing items) and check for cycles.
	existingItems, err := h.store.LoadAll(nil)
	if err != nil {
		log.Printf("[backlog] batch-create: failed to load existing items: %v", err)
		httputil.InternalError(w, "[backlog] batch-create", "failed to load existing items for cycle check")
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
		httputil.BadRequest(w, "[backlog] batch-create",
			fmt.Sprintf("dependency cycle detected: %s", strings.Join(cycle, " -> ")))
		return
	}

	// Phase 4: If initiative specified, verify it exists or create it.
	if initiativeName != "" {
		if h.initiativeAssigner == nil {
			httputil.InternalError(w, "[backlog] batch-create", "initiative support not configured")
			return
		}
		if ensureErr := h.initiativeAssigner.EnsureExists(initiativeName); ensureErr != nil {
			log.Printf("[backlog] batch-create: failed to ensure initiative %q: %v", initiativeName, ensureErr)
			httputil.InternalError(w, "[backlog] batch-create", "failed to ensure initiative exists: "+ensureErr.Error())
			return
		}
	}

	// Phase 5: Create all items atomically. Track created directories for rollback.
	createdDirs := make([]string, 0, len(validated))
	createdItems := make([]BacklogItem, 0, len(validated))

	for _, v := range validated {
		itemDir := h.store.ItemDir(v.kind, v.item.Name)
		if mkErr := os.MkdirAll(itemDir, 0o755); mkErr != nil {
			// Rollback all created directories.
			for _, dir := range createdDirs {
				_ = os.RemoveAll(dir)
			}
			log.Printf("[backlog] batch-create: failed to create directory for %q: %v", v.item.Name, mkErr)
			httputil.InternalError(w, "[backlog] batch-create", "failed to create item directory")
			return
		}
		createdDirs = append(createdDirs, itemDir)

		if saveErr := h.store.SaveItem(v.item); saveErr != nil {
			// Rollback all created directories.
			for _, dir := range createdDirs {
				_ = os.RemoveAll(dir)
			}
			log.Printf("[backlog] batch-create: failed to save %q: %v", v.item.Name, saveErr)
			httputil.InternalError(w, "[backlog] batch-create", "failed to save item")
			return
		}

		createdItems = append(createdItems, v.item)
	}

	// Phase 6: If initiative specified, add all items to it.
	var warnings []string
	if initiativeName != "" && h.initiativeAssigner != nil {
		itemRefs := make([]string, 0, len(createdItems))
		for _, item := range createdItems {
			itemRefs = append(itemRefs, string(item.Kind)+"/"+item.Name)
		}
		if addErr := h.initiativeAssigner.AddItems(initiativeName, itemRefs); addErr != nil {
			log.Printf("[backlog] batch-create: failed to add items to initiative %q: %v", initiativeName, addErr)
			warnings = append(warnings, fmt.Sprintf("items created but initiative assignment failed: %s",
				httputil.TruncateErrorMessage(addErr, 240)))
		}
	}

	log.Printf("[backlog] batch-created %d items", len(createdItems))

	resp := batchCreateResponse{
		Items:      createdItems,
		Initiative: initiativeName,
		Count:      len(createdItems),
		Warnings:   warnings,
	}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[backlog] batch-create", "failed to encode response")
	}
}
