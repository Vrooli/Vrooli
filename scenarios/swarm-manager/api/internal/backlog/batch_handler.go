// Batch operations for backlog items: atomic multi-item creation with
// dependency validation, initiative assignment, and rollback on failure.
package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"
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
	// Delete removes an initiative entirely. Implementations are expected to
	// cascade: clear the initiative field on every member item and remove the
	// deleted name from other initiatives' depends_on arrays.
	Delete(name string) error
	// AddItems appends item references ("kind/name") to the named initiative.
	// Implementations maintain symmetry with the item side: items already
	// attached to a different initiative are rejected; orphan items have their
	// initiative field set to this name.
	AddItems(name string, items []string) error
	// RememberItem appends a single ref to the initiative's items[] list
	// without touching the item side. Used by cascade paths (single-item
	// create/patch) where the item's initiative field is already correct.
	RememberItem(initiativeName, ref string) error
	// ForgetItem removes a single ref from the initiative's items[] list
	// without touching the item side. Used by cascade paths (single-item
	// delete/patch) where the item file has already been deleted or its
	// initiative field is handled elsewhere.
	ForgetItem(initiativeName, ref string) error
}

// InitiativeSpec describes the canonical metadata for an initiative.
type InitiativeSpec struct {
	Name        string
	Title       string
	Description string
	Status      string
	Priority    int
	DependsOn   []string
	CreatedBy   *identity.Provenance
}

// InitiativeSnapshot captures the full persisted state of an initiative.
type InitiativeSnapshot struct {
	Name        string
	Title       string
	Description string
	Status      string
	Priority    int
	DependsOn   []string
	Items       []string
	CreatedBy   *identity.Provenance
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
	Creates         []string `json:"creates,omitempty"`
	// SpawnedFrom stamps provenance the way single-create already does, so
	// batch-landed items (e.g. plan imports) carry where they came from.
	SpawnedFrom string `json:"spawned_from,omitempty"`
}

// batchCreateInitiative describes initiative metadata supplied with a batch import.
type batchCreateInitiative struct {
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Priority    *int      `json:"priority,omitempty"`
	DependsOn   *[]string `json:"depends_on,omitempty"`
}

// batchCreateInitiativeResult reports what the batch import will do or did for
// initiative metadata.
type batchCreateInitiativeResult struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Action      string   `json:"action"`
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

type batchApplyResult struct {
	items       []BacklogItem
	initiatives []batchCreateInitiativeResult
	artifacts   []agentsessions.Artifact
}

// validatedItem pairs a validated BacklogItem with its parsed kind.
type validatedItem struct {
	item BacklogItem
	kind BacklogKind
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

	result, err := h.applyBatchCreateRequest(r.Context(), req, identity.FromContext(r.Context()), true, "http.batch_create")
	if err != nil {
		apierr.MapError(w, "[backlog] batch-create", err)
		return
	}

	resp := batchCreateResponse{
		Items:       result.items,
		Initiatives: result.initiatives,
		Count:       len(result.items),
		Preview:     req.Preview,
	}
	status := http.StatusCreated
	if req.Preview {
		status = http.StatusOK
	}
	if err := httputil.JSONWithStatus(w, status, resp); err != nil {
		apierr.MapError(w, "[backlog] batch-create", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) ApplyAgentSessionBacklogBatchImport(ctx context.Context, payloadJSON string, prov identity.Provenance) ([]agentsessions.Artifact, error) {
	var req batchCreateRequest
	decoder := json.NewDecoder(bytes.NewReader([]byte(payloadJSON)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return nil, apierr.BadRequest("invalid backlog batch proposal payload: %s", httputil.TruncateErrorMessage(err, 240))
	}
	req.Preview = false
	result, err := h.applyBatchCreateRequest(ctx, req, prov, false, "agent_sessions.apply.backlog_batch_import")
	if err != nil {
		return nil, err
	}
	return result.artifacts, nil
}

// ImportBatchItems lands a JSON batch payload atomically via the same path as
// batch-create and returns the created items. Used by the plan-import bridge to
// reuse the atomic multi-item create (dependency validation, cycle rejection,
// provenance) without going back out over HTTP.
func (h *Handler) ImportBatchItems(ctx context.Context, payloadJSON string, prov identity.Provenance) ([]BacklogItem, error) {
	var req batchCreateRequest
	decoder := json.NewDecoder(bytes.NewReader([]byte(payloadJSON)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return nil, apierr.BadRequest("invalid plan-import batch payload: %s", httputil.TruncateErrorMessage(err, 240))
	}
	req.Preview = false
	result, err := h.applyBatchCreateRequest(ctx, req, prov, true, "http.plan_import")
	if err != nil {
		return nil, err
	}
	return result.items, nil
}

func (h *Handler) applyBatchCreateRequest(
	ctx context.Context,
	req batchCreateRequest,
	prov identity.Provenance,
	triggerAutoWorkshop bool,
	mutationSource string,
) (batchApplyResult, error) {
	if len(req.Items) == 0 {
		return batchApplyResult{}, apierr.BadRequest("at least one item is required")
	}
	const maxBatchSize = 100
	if len(req.Items) > maxBatchSize {
		return batchApplyResult{}, apierr.BadRequest("batch size %d exceeds maximum of %d", len(req.Items), maxBatchSize)
	}

	providedInitiatives, err := h.validateBatchInitiatives(req.Initiatives)
	if err != nil {
		return batchApplyResult{}, apierr.BadRequest("%s", err.Error())
	}

	validated, batchNames, referencedInitiatives, err := h.validateBatchItems(req.Items, providedInitiatives)
	if err != nil {
		return batchApplyResult{}, err
	}

	initiativePlans, initiativeResults, err := h.planInitiativeChanges(referencedInitiatives, providedInitiatives)
	if err != nil {
		return batchApplyResult{}, err
	}

	if err := h.validateBatchDependencies(validated, batchNames); err != nil {
		return batchApplyResult{}, err
	}

	if err := h.checkBatchDependencyCycles(validated); err != nil {
		return batchApplyResult{}, err
	}

	if req.Preview {
		previewItems := make([]BacklogItem, 0, len(validated))
		for _, v := range validated {
			previewItems = append(previewItems, v.item)
		}
		return batchApplyResult{items: previewItems, initiatives: initiativeResults}, nil
	}

	stampBatchItemProvenance(validated, prov)
	stampInitiativePlanProvenance(initiativePlans, prov)

	appliedInitiatives, err := h.applyInitiativeChanges(initiativePlans)
	if err != nil {
		return batchApplyResult{}, err
	}

	createdItems, err := h.createBatchItems(ctx, validated, appliedInitiatives)
	if err != nil {
		return batchApplyResult{}, err
	}

	if err := h.assignItemsToInitiatives(createdItems, appliedInitiatives); err != nil {
		return batchApplyResult{}, err
	}

	artifacts, err := h.recordBatchSessionArtifacts(ctx, createdItems, appliedInitiatives, prov, mutationSource)
	if err != nil {
		rollbackBatchCreate(batchItemDirs(h.store, createdItems), appliedInitiatives, h.initiativeAssigner)
		return batchApplyResult{}, apierr.Internal("failed to record session artifacts")
	}

	if triggerAutoWorkshop {
		for _, item := range createdItems {
			h.maybeAutoWorkshop(item, false)
		}
	}

	slog.Info("batch-created items", "count", len(createdItems))
	h.invalidateAllGraphLenses()

	return batchApplyResult{items: createdItems, initiatives: initiativeResults, artifacts: artifacts}, nil
}

// validateBatchInitiatives validates and normalizes initiative metadata from
// the request. Returns a map keyed by initiative name.
func (h *Handler) validateBatchInitiatives(raw []batchCreateInitiative) (map[string]batchCreateInitiative, error) {
	result := make(map[string]batchCreateInitiative, len(raw))
	for i, init := range raw {
		name := strings.TrimSpace(init.Name)
		if name == "" {
			return nil, fmt.Errorf("initiatives[%d]: name is required", i)
		}
		if strings.TrimSpace(init.Title) == "" {
			return nil, fmt.Errorf("initiatives[%d]: title is required", i)
		}
		if init.Status != nil {
			status := strings.TrimSpace(*init.Status)
			if !isValidInitiativeStatus(status) {
				return nil, fmt.Errorf("initiatives[%d]: status must be active or completed", i)
			}
		}
		if init.Priority != nil {
			p := *init.Priority
			if p < 0 || p > 10 {
				return nil, fmt.Errorf("initiatives[%d]: priority must be 0 (unset) or 1-10", i)
			}
		}
		if init.DependsOn != nil {
			normalized, err := normalizeInitiativeDeps(*init.DependsOn, name)
			if err != nil {
				return nil, fmt.Errorf("initiatives[%d]: %s", i, err.Error())
			}
			init.DependsOn = &normalized
		}
		if _, exists := result[name]; exists {
			return nil, fmt.Errorf("initiatives[%d]: duplicate initiative %q", i, name)
		}
		init.Name = name
		init.Title = strings.TrimSpace(init.Title)
		if init.Description != nil {
			d := strings.TrimSpace(*init.Description)
			init.Description = &d
		}
		if init.Status != nil {
			s := strings.TrimSpace(*init.Status)
			init.Status = &s
		}
		result[name] = init
	}
	if err := h.validateInitiativeDepRefs(result); err != nil {
		return nil, err
	}
	return result, nil
}

// validateBatchItems validates each item in the batch and returns the validated
// items, a set of batch names for duplicate checking, and the set of referenced
// initiative names.
func (h *Handler) validateBatchItems(
	items []batchCreateItem,
	providedInitiatives map[string]batchCreateInitiative,
) ([]validatedItem, map[string]bool, map[string]bool, error) {
	validated := make([]validatedItem, 0, len(items))
	batchNames := make(map[string]bool, len(items))
	referencedInitiatives := make(map[string]bool)
	now := time.Now().UTC().Format(time.RFC3339)

	for i, raw := range items {
		v, err := h.validateSingleBatchItem(i, raw, batchNames, now)
		if err != nil {
			return nil, nil, nil, err
		}
		batchNames[string(v.kind)+"/"+v.item.Name] = true
		if v.item.Initiative != "" {
			referencedInitiatives[v.item.Initiative] = true
		}
		validated = append(validated, v)
	}

	for name := range providedInitiatives {
		if !referencedInitiatives[name] {
			return nil, nil, nil, apierr.BadRequest("initiative %q is not referenced by any item", name)
		}
	}

	return validated, batchNames, referencedInitiatives, nil
}

// validateSingleBatchItem validates and normalizes one batch item.
func (h *Handler) validateSingleBatchItem(
	i int,
	raw batchCreateItem,
	batchNames map[string]bool,
	now string,
) (validatedItem, error) {
	if strings.TrimSpace(raw.Kind) == "" {
		return validatedItem{}, apierr.BadRequest("item[%d]: kind is required", i)
	}
	kind, err := ParseBacklogKind(raw.Kind)
	if err != nil {
		return validatedItem{}, apierr.BadRequest("item[%d]: %s", i, err.Error())
	}

	if strings.TrimSpace(raw.Title) == "" {
		return validatedItem{}, apierr.BadRequest("item[%d]: title is required", i)
	}

	name := strings.TrimSpace(raw.Name)
	if name == "" {
		name = raw.Title
	}
	name = sanitizeName(name)
	if name == "" {
		return validatedItem{}, apierr.BadRequest("item[%d]: name is required", i)
	}

	key := string(kind) + "/" + name
	if batchNames[key] {
		return validatedItem{}, apierr.BadRequest("item[%d]: duplicate item %q in batch", i, key)
	}

	priority := 5
	if raw.Priority != nil {
		if *raw.Priority < 1 || *raw.Priority > 10 {
			return validatedItem{}, apierr.BadRequest("item[%d]: priority must be between 1 and 10", i)
		}
		priority = int(*raw.Priority)
	}

	itemDir := h.store.ItemDir(kind, name)
	if _, err := os.Stat(itemDir); err == nil {
		return validatedItem{}, apierr.Conflict("item[%d]: %q already exists", i, key)
	}

	tags := raw.Tags
	if tags == nil {
		tags = []string{}
	}
	dependsOn := raw.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}

	effort := ""
	if raw.Effort != nil {
		normalized, err := validateEffort(*raw.Effort)
		if err != nil {
			return validatedItem{}, apierr.BadRequest("item[%d]: %s", i, err.Error())
		}
		effort = normalized
	}

	if err := validateGlobs(raw.AcceptanceAllow); err != nil {
		return validatedItem{}, apierr.BadRequest("item[%d]: acceptance_allow: %s", i, err.Error())
	}
	if err := validateGlobs(raw.AcceptanceDeny); err != nil {
		return validatedItem{}, apierr.BadRequest("item[%d]: acceptance_deny: %s", i, err.Error())
	}
	if err := validateGlobs(raw.Creates); err != nil {
		return validatedItem{}, apierr.BadRequest("item[%d]: creates: %s", i, err.Error())
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
		Initiative:      strings.TrimSpace(raw.Initiative),
		Effort:          effort,
		AcceptanceAllow: raw.AcceptanceAllow,
		AcceptanceDeny:  raw.AcceptanceDeny,
		Creates:         raw.Creates,
		SpawnedFrom:     strings.TrimSpace(raw.SpawnedFrom),
	}

	return validatedItem{item: item, kind: kind}, nil
}

func stampBatchItemProvenance(validated []validatedItem, prov identity.Provenance) {
	for i := range validated {
		validated[i].item.CreatedBy = &prov
	}
}

func stampInitiativePlanProvenance(plans map[string]resolvedInitiativePlan, prov identity.Provenance) {
	for name, plan := range plans {
		if plan.action == "create" && plan.spec.CreatedBy == nil {
			plan.spec.CreatedBy = &prov
			plans[name] = plan
		}
	}
}

// planInitiativeChanges resolves what initiative operations are needed and
// returns both the execution plans and preview results.
func (h *Handler) planInitiativeChanges(
	referencedInitiatives map[string]bool,
	providedInitiatives map[string]batchCreateInitiative,
) (map[string]resolvedInitiativePlan, []batchCreateInitiativeResult, error) {
	if len(referencedInitiatives) == 0 {
		return nil, nil, nil
	}
	if h.initiativeAssigner == nil {
		return nil, nil, apierr.Internal("initiative support not configured")
	}
	plans, results, err := h.resolveInitiativePlans(referencedInitiatives, providedInitiatives)
	if err != nil {
		return nil, nil, apierr.BadRequest("%s", err.Error())
	}
	return plans, results, nil
}

// validateBatchDependencies checks that all depends_on references exist either
// in the batch or already on disk.
func (h *Handler) validateBatchDependencies(validated []validatedItem, batchNames map[string]bool) error {
	for i, v := range validated {
		for _, ref := range v.item.DependsOn {
			if batchNames[ref] {
				continue
			}
			depKind, depName, err := parseDependencyRef(ref)
			if err != nil {
				return apierr.BadRequest("item[%d]: %s", i, err.Error())
			}
			if _, loadErr := h.store.LoadItem(depKind, depName); errors.Is(loadErr, ErrNotFound) {
				return apierr.BadRequest("item[%d]: dependency %q does not exist", i, ref)
			}
		}
	}
	return nil
}

// checkBatchDependencyCycles builds a dependency graph from batch + existing
// items and checks for cycles.
func (h *Handler) checkBatchDependencyCycles(validated []validatedItem) error {
	existingItems, err := h.store.LoadAll(nil)
	if err != nil {
		slog.Error("failed to load existing items", "err", err)
		return apierr.Internal("failed to load existing items for cycle check")
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
		return apierr.BadRequest("dependency cycle detected: %s", strings.Join(cycle, " -> "))
	}
	return nil
}

// applyInitiativeChanges creates or updates initiatives as planned. Returns
// the list of applied plans for rollback tracking. Plans are applied in
// dependency order so cross-initiative depends_on references are always
// satisfied before the dependent is persisted.
func (h *Handler) applyInitiativeChanges(plans map[string]resolvedInitiativePlan) ([]resolvedInitiativePlan, error) {
	applied := make([]resolvedInitiativePlan, 0, len(plans))
	order, err := orderedInitiativePlans(plans)
	if err != nil {
		return nil, apierr.BadRequest("%s", err.Error())
	}
	for _, name := range order {
		plan := plans[name]
		switch plan.action {
		case "create":
			if err := h.initiativeAssigner.Create(plan.spec); err != nil {
				slog.Error("failed to create initiative", "initiative", name, "err", err)
				return nil, apierr.Internal("%s", "failed to create initiative: "+httputil.TruncateErrorMessage(err, 240))
			}
		case "update":
			if err := h.initiativeAssigner.Update(plan.spec); err != nil {
				slog.Error("failed to update initiative", "initiative", name, "err", err)
				return nil, apierr.Internal("%s", "failed to update initiative: "+httputil.TruncateErrorMessage(err, 240))
			}
		}
		applied = append(applied, plan)
	}
	return applied, nil
}

// createBatchItems creates all items on disk via the unified
// backlog.Service.Create chokepoint. SkipDuplicateCheck and
// SkipCycleCheck are set because validateBatchItems / checkBatchDependencyCycles
// already validated up front. SkipInitiativeAttach defers initiative
// membership writes to assignItemsToInitiatives, which uses bulk
// AddItems for one initiative.json write per initiative instead of N
// from per-item RememberItem. SkipWorkshopTrigger and SkipGraphInvalidation
// defer those side effects to the end of the batch where they fire once.
func (h *Handler) createBatchItems(ctx context.Context, validated []validatedItem, appliedInitiatives []resolvedInitiativePlan) ([]BacklogItem, error) {
	createdDirs := make([]string, 0, len(validated))
	createdItems := make([]BacklogItem, 0, len(validated))
	svc := h.creationService()

	for _, v := range validated {
		err := svc.Create(v.item, CreationContext{
			Context:               ctx,
			Source:                SourceBatch,
			Entrypoint:            "http.batch_create",
			SkipDuplicateCheck:    true,
			SkipCycleCheck:        true,
			SkipInitiativeAttach:  true,
			SkipWorkshopTrigger:   true,
			SkipGraphInvalidation: true,
			SkipSessionArtifact:   true,
		})
		if err != nil {
			rollbackBatchCreate(createdDirs, appliedInitiatives, h.initiativeAssigner)
			slog.Error("failed to create batch item", "item", v.item.Name, "err", err)
			return nil, apierr.Internal("failed to create item: %s", httputil.TruncateErrorMessage(err, 240))
		}
		createdDirs = append(createdDirs, h.store.ItemDir(v.kind, v.item.Name))
		createdItems = append(createdItems, v.item)
	}
	return createdItems, nil
}

// assignItemsToInitiatives adds created items to their respective initiatives.
// On failure, rolls back all item directories and initiative changes.
func (h *Handler) assignItemsToInitiatives(createdItems []BacklogItem, appliedInitiatives []resolvedInitiativePlan) error {
	createdDirs := make([]string, 0, len(createdItems))
	for _, item := range createdItems {
		createdDirs = append(createdDirs, h.store.ItemDir(item.Kind, item.Name))
	}

	for name, refs := range groupItemRefsByInitiative(createdItems) {
		if addErr := h.initiativeAssigner.AddItems(name, refs); addErr != nil {
			rollbackBatchCreate(createdDirs, appliedInitiatives, h.initiativeAssigner)
			slog.Error("failed to add items to initiative", "initiative", name, "err", addErr)
			return apierr.Internal("%s", "failed to assign items to initiative: "+httputil.TruncateErrorMessage(addErr, 240))
		}
	}
	return nil
}

func (h *Handler) recordBatchSessionArtifacts(ctx context.Context, items []BacklogItem, applied []resolvedInitiativePlan, prov identity.Provenance, mutationSource string) ([]agentsessions.Artifact, error) {
	if h.sessionArtifacts == nil || strings.TrimSpace(prov.SessionID) == "" {
		return nil, nil
	}
	attr := agentsessions.AttributionFromProvenance(prov)
	artifacts := make([]agentsessions.Artifact, 0, len(items)+len(applied))
	for _, item := range items {
		artifacts = append(artifacts, agentsessions.Artifact{
			SessionID:      prov.SessionID,
			ArtifactType:   agentsessions.ArtifactBacklogItem,
			Action:         agentsessions.ArtifactActionCreated,
			EntityRef:      string(item.Kind) + "/" + item.Name,
			Title:          item.Title,
			RunID:          prov.RunID,
			MutationSource: mutationSource,
			Attribution:    &attr,
		})
	}
	for _, plan := range applied {
		action := agentsessions.ArtifactActionUpdated
		if plan.action == "create" {
			action = agentsessions.ArtifactActionCreated
		}
		artifacts = append(artifacts, agentsessions.Artifact{
			SessionID:      prov.SessionID,
			ArtifactType:   agentsessions.ArtifactInitiative,
			Action:         action,
			EntityRef:      plan.spec.Name,
			Title:          plan.spec.Title,
			RunID:          prov.RunID,
			MutationSource: mutationSource,
			Attribution:    &attr,
		})
	}
	return h.sessionArtifacts.AttachArtifacts(ctx, artifacts)
}

func batchItemDirs(store Store, items []BacklogItem) []string {
	dirs := make([]string, 0, len(items))
	for _, item := range items {
		dirs = append(dirs, store.ItemDir(item.Kind, item.Name))
	}
	return dirs
}
