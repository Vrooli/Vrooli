package backlog

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// List returns all backlog items.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	kinds, err := parseKindsQuery(r)
	if err != nil {
		apierr.MapError(w, "[backlog] list", apierr.BadRequest("%s", err.Error()))
		return
	}

	statusFilter, err := parseStatusesQuery(r)
	if err != nil {
		apierr.MapError(w, "[backlog] list", apierr.BadRequest("%s", err.Error()))
		return
	}

	archivedFilter := parseArchivedQuery(r)

	items, err := h.store.LoadAll(kinds)
	if err != nil {
		apierr.MapError(w, "[backlog] list", apierr.Internal("%s", err.Error()))
		return
	}

	items = filterByStatus(items, statusFilter)
	items = filterByArchived(items, archivedFilter)
	items = filterByScenario(items, parseScenariosQuery(r))
	validationFilter := parseValidationStatusQuery(r)
	if validationFilter != "" {
		items = h.filterByValidationStatus(items, validationFilter)
	}

	if sf := r.URL.Query().Get("spawned_from"); sf != "" {
		filtered := items[:0]
		for _, item := range items {
			if item.SpawnedFrom == sf {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	// Sort by priority (ascending) then by updated (descending)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return items[i].Updated > items[j].Updated
	})

	protoItems := make([]*domainpb.BacklogItem, 0, len(items))
	for _, item := range items {
		itemDir := h.store.ItemDir(item.Kind, item.Name)
		protoItems = append(protoItems, backlogToProtoWithValidation(item, itemDir))
	}

	// Compute per-item blocking info from the full item set.
	blockingMap := ComputeListBlockingInfo(items)
	protoBlocking := make(map[string]*apipb.ItemBlockingInfo, len(blockingMap))
	for key, info := range blockingMap {
		protoBlocking[key] = &apipb.ItemBlockingInfo{
			Blocked:         info.Blocked,
			BlockingDepKeys: info.BlockingDepKeys,
			AllForceable:    info.AllForceable,
		}
	}

	resp := &apipb.ListBacklogItemsResponse{Items: protoItems, Blocking: protoBlocking}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] list", apierr.Internal("failed to encode response"))
	}
}

// Get returns a single backlog item by name.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "get")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] get", apierr.Internal("%s", err.Error()))
		return
	}

	itemDir := h.store.ItemDir(kind, item.Name)
	resp := &apipb.BacklogItemResponse{Item: backlogToProtoWithValidation(item, itemDir)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] get", apierr.Internal("failed to encode response"))
	}
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogViewed(string(kind)+"/"+name, string(kind))
	}
}

// GetValidation returns a fresh plan validation result for a backlog item.
func (h *Handler) GetValidation(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "validation")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] validation", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] validation", apierr.Internal("%s", err.Error()))
		return
	}

	if kind == KindResearch {
		result := ValidatePlanCompleteness("", KindResearch)
		if err := httputil.JSON(w, result); err != nil {
			apierr.MapError(w, "[backlog] validation", apierr.Internal("failed to encode response"))
		}
		return
	}

	itemDir := h.store.ItemDir(kind, item.Name)
	deliverable := DeliverableForKind(kind)
	planContent := LoadPlanContentByName(itemDir, deliverable)
	result := ValidatePlanCompleteness(planContent, kind)

	// Write the report so it's cached for future reads.
	if writeErr := WriteValidationReport(itemDir, result); writeErr != nil {
		slog.Warn("failed to write validation report", "kind", kind, "name", name, "err", writeErr)
	}

	if err := httputil.JSON(w, result); err != nil {
		apierr.MapError(w, "[backlog] validation", apierr.Internal("failed to encode response"))
	}
}

func parseKindsQuery(r *http.Request) ([]BacklogKind, error) {
	query := r.URL.Query()
	raw := strings.TrimSpace(query.Get("kinds"))
	if raw == "" {
		raw = strings.TrimSpace(query.Get("kind"))
	}
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	kinds := make([]BacklogKind, 0, len(parts))
	for _, part := range parts {
		kind, err := ParseBacklogKind(part)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

// parseStatusesQuery reads the "statuses" (or "status") query parameter.
// Returns nil when no filter is specified (caller should apply default).
// The special value "all" returns an empty slice signaling no filtering.
func parseStatusesQuery(r *http.Request) ([]BacklogStatus, error) {
	query := r.URL.Query()
	raw := strings.TrimSpace(query.Get("statuses"))
	if raw == "" {
		raw = strings.TrimSpace(query.Get("status"))
	}
	if raw == "" {
		return nil, nil
	}
	if strings.EqualFold(raw, "all") {
		return []BacklogStatus{}, nil
	}

	parts := strings.Split(raw, ",")
	statuses := make([]BacklogStatus, 0, len(parts))
	for _, part := range parts {
		s := BacklogStatus(strings.TrimSpace(part))
		if s == "" {
			continue
		}
		statuses = append(statuses, s)
	}
	return statuses, nil
}

// filterByStatus applies status filtering to a list of backlog items.
//   - nil (no query param): no status filtering
//   - empty slice (status=all): no filtering, return everything
//   - non-empty slice: include only items matching one of the given statuses
func filterByStatus(items []BacklogItem, statuses []BacklogStatus) []BacklogItem {
	if len(statuses) == 0 {
		return items
	}

	allow := make(map[BacklogStatus]bool, len(statuses))
	for _, s := range statuses {
		allow[s] = true
	}
	filtered := make([]BacklogItem, 0, len(items))
	for _, item := range items {
		if allow[item.Status] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// archivedFilter represents the ?archived query parameter.
type archivedFilter int

const (
	archivedExclude archivedFilter = iota // default: exclude archived
	archivedOnly                          // archived=true: only archived
	archivedAll                           // archived=all: include everything
)

// parseArchivedQuery reads the "archived" query parameter.
func parseArchivedQuery(r *http.Request) archivedFilter {
	raw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("archived")))
	switch raw {
	case "true", "1", "yes":
		return archivedOnly
	case "all":
		return archivedAll
	default:
		return archivedExclude
	}
}

// filterByArchived filters items based on their archived_at field.
func filterByArchived(items []BacklogItem, filter archivedFilter) []BacklogItem {
	if filter == archivedAll {
		return items
	}
	filtered := make([]BacklogItem, 0, len(items))
	for _, item := range items {
		isArchived := item.ArchivedAt != nil
		if filter == archivedExclude && !isArchived {
			filtered = append(filtered, item)
		} else if filter == archivedOnly && isArchived {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// parseScenariosQuery reads the "scenario" (or "scenarios") query parameter.
// Returns nil when no filter is specified.
func parseScenariosQuery(r *http.Request) []string {
	query := r.URL.Query()
	raw := strings.TrimSpace(query.Get("scenario"))
	if raw == "" {
		raw = strings.TrimSpace(query.Get("scenarios"))
	}
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// filterByScenario keeps items whose AcceptanceAllow targets at least one of the given scenarios.
func filterByScenario(items []BacklogItem, scenarios []string) []BacklogItem {
	if len(scenarios) == 0 {
		return items
	}
	allow := make(map[string]bool, len(scenarios))
	for _, s := range scenarios {
		allow[s] = true
	}
	filtered := make([]BacklogItem, 0, len(items))
	for _, item := range items {
		for _, s := range pathutil.ScenariosFromGlobs(item.AcceptanceAllow) {
			if allow[s] {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

// parseValidationStatusQuery reads the "validation_status" query parameter.
// Returns "" when not specified. Valid values: "passed", "failed", "none".
func parseValidationStatusQuery(r *http.Request) string {
	raw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("validation_status")))
	switch raw {
	case "passed", "failed", "none":
		return raw
	default:
		return ""
	}
}

// filterByValidationStatus filters items by their plan validation status.
func (h *Handler) filterByValidationStatus(items []BacklogItem, status string) []BacklogItem {
	filtered := make([]BacklogItem, 0, len(items))
	for _, item := range items {
		if item.Kind == KindResearch {
			if status == "passed" || status == "none" {
				filtered = append(filtered, item)
			}
			continue
		}
		itemDir := h.store.ItemDir(item.Kind, item.Name)
		report, err := LoadValidationReport(itemDir)
		if err != nil || report == nil {
			if status == "none" {
				filtered = append(filtered, item)
			}
			continue
		}
		if status == "passed" && report.Passed {
			filtered = append(filtered, item)
		} else if status == "failed" && !report.Passed {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// checkDependencyCycles builds a dependency graph from all existing items plus
// the given item and checks for cycles.
func (h *Handler) checkDependencyCycles(item BacklogItem) error {
	items, err := h.store.LoadAll(nil)
	if err != nil {
		return fmt.Errorf("failed to load items for cycle check: %w", err)
	}

	g := depgraph.New()
	itemKey := string(item.Kind) + "/" + item.Name
	g.AddNode(itemKey, item.DependsOn)

	for _, existing := range items {
		key := string(existing.Kind) + "/" + existing.Name
		if key == itemKey {
			continue // use the new/updated version
		}
		g.AddNode(key, existing.DependsOn)
	}

	if cycle, found := g.DetectCycle(); found {
		return fmt.Errorf("dependency cycle detected: %s", strings.Join(cycle, " -> "))
	}
	return nil
}
