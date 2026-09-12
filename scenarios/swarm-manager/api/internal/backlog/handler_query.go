package backlog

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

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

	filters := ListFilters{
		Kinds:       kinds,
		Statuses:    statusFilter,
		Archived:    parseArchivedQuery(r),
		Scenarios:   parseScenariosQuery(r),
		SpawnedFrom: strings.TrimSpace(r.URL.Query().Get("spawned_from")),
		PlanRef:     strings.TrimSpace(r.URL.Query().Get("plan_ref")),
		ActorID:     strings.TrimSpace(r.URL.Query().Get("actor_id")),
	}
	if filters.PlanRef == "" {
		filters.PlanRef = strings.TrimSpace(r.URL.Query().Get("plan_ref_slug"))
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("has_plan_ref")); raw != "" {
		value := raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
		filters.HasPlanRef = &value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("stale")); raw != "" {
		value := raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
		filters.Stale = &value
	}
	resp, err := h.listItems(filters)
	if err != nil {
		apierr.MapError(w, "[backlog] list", apierr.Internal("%s", err.Error()))
		return
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] list", apierr.Internal("failed to encode response"))
	}
}

// ListFilters is the transport-neutral filter set for the backlog projection.
// Both REST and Connect delegate here so their result semantics cannot drift.
type ListFilters struct {
	Kinds       []BacklogKind
	Statuses    []BacklogStatus
	Archived    archivedFilter
	Scenarios   []string
	SpawnedFrom string
	HasPlanRef  *bool
	PlanRef     string
	Stale       *bool
	ActorID     string
}

func (h *Handler) listItems(filters ListFilters) (*apipb.ListBacklogItemsResponse, error) {
	items, err := h.store.LoadAll(filters.Kinds)
	if err != nil {
		return nil, err
	}
	items = filterByStatus(items, filters.Statuses)
	items = filterByArchived(items, filters.Archived)
	items = filterByScenario(items, filters.Scenarios)
	if filters.ActorID != "" {
		filtered := items[:0]
		for _, item := range items {
			if item.CreatedBy != nil && item.CreatedBy.IsVerifiedAgent() && item.CreatedBy.ProfileKey == filters.ActorID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if filters.SpawnedFrom != "" {
		filtered := items[:0]
		for _, item := range items {
			if item.SpawnedFrom == filters.SpawnedFrom {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	items = filterByPlanRefValues(items, filters.HasPlanRef, filters.PlanRef)
	if filters.Stale != nil {
		filtered := items[:0]
		for _, item := range items {
			if IsStale(item, h.repoRoot, time.Now().UTC()) == *filters.Stale {
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
		proto := backlogToProto(item)
		stale := IsStale(item, h.repoRoot, time.Now().UTC())
		proto.Stale = &stale
		protoItems = append(protoItems, proto)
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

	return &apipb.ListBacklogItemsResponse{Items: protoItems, Blocking: protoBlocking}, nil
}

func filterByPlanRefValues(items []BacklogItem, hasPlanRef *bool, planRef string) []BacklogItem {
	if hasPlanRef != nil {
		filtered := items[:0]
		for _, item := range items {
			if (item.PlanRef != nil) == *hasPlanRef {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if planRef != "" {
		filtered := items[:0]
		for _, item := range items {
			if item.PlanRef != nil && item.PlanRef.Slug == planRef {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return items
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

	proto := backlogToProto(item)
	stale := IsStale(item, h.repoRoot, time.Now().UTC())
	proto.Stale = &stale
	resp := &apipb.BacklogItemResponse{Item: proto}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] get", apierr.Internal("failed to encode response"))
	}
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogViewed(string(kind)+"/"+name, string(kind))
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
