package backlog

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/httputil"
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

	items, err := h.store.LoadAll(kinds)
	if err != nil {
		apierr.MapError(w, "[backlog] list", apierr.Internal("%s", err.Error()))
		return
	}

	items = filterByStatus(items, statusFilter)

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
		protoItems = append(protoItems, backlogToProto(item))
	}

	resp := &apipb.ListBacklogItemsResponse{Items: protoItems}
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

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
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
//   - nil (no query param): exclude archived items (default)
//   - empty slice (status=all): no filtering, return everything
//   - non-empty slice: include only items matching one of the given statuses
func filterByStatus(items []BacklogItem, statuses []BacklogStatus) []BacklogItem {
	if statuses != nil && len(statuses) == 0 {
		return items
	}

	filtered := make([]BacklogItem, 0, len(items))
	if statuses == nil {
		for _, item := range items {
			if item.Status != StatusArchived {
				filtered = append(filtered, item)
			}
		}
		return filtered
	}

	allow := make(map[BacklogStatus]bool, len(statuses))
	for _, s := range statuses {
		allow[s] = true
	}
	for _, item := range items {
		if allow[item.Status] {
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
