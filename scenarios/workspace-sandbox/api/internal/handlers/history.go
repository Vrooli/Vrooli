package handlers

import (
	"net/http"
	"time"

	"workspace-sandbox/internal/types"
)

// ListHistoryResponse is the wire shape for GET /sandboxes/history.
type ListHistoryResponse struct {
	Archives   []*types.DiffArchive `json:"archives"`
	TotalCount int                  `json:"totalCount"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
}

// ListHistory returns archived (terminal-state) sandboxes for the
// History tab in the workspace-sandbox UI.
//
// Query parameters (all optional):
//   - status (repeatable): filter by sandbox_status; subset of
//     {approved, rejected, deleted}. Empty means all three.
//   - projectRoot, owner, agentManagerRunId: exact-match filters.
//   - search: free-text substring across owner / agent-manager run id /
//     sandbox id. Case-sensitive (predictable behavior).
//   - snapshotAtFrom, snapshotAtTo: RFC3339 bounds on snapshot_at.
//   - sortBy: "snapshot_at" (default) or "total_blob_bytes".
//   - sortDesc: "true" / "1" toggles descending order. Default is
//     descending for snapshot_at (newest first); the repository
//     normalizes when SortDesc is unset.
//   - limit, offset: pagination. Repository clamps to a sane upper
//     bound.
//
// The Active tab continues to use GET /sandboxes (status filter); this
// endpoint is purely for terminal-state listing.
func (h *Handlers) ListHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := types.ArchiveListFilter{
		ProjectRoot:       q.Get("projectRoot"),
		Owner:             q.Get("owner"),
		AgentManagerRunID: q.Get("agentManagerRunId"),
		Search:            q.Get("search"),
		SortBy:            q.Get("sortBy"),
	}

	for _, st := range q["status"] {
		filter.Statuses = append(filter.Statuses, types.Status(st))
	}

	if v := q.Get("snapshotAtFrom"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			h.JSONError(w, "snapshotAtFrom must be RFC3339 (e.g. 2026-04-29T00:00:00Z)", http.StatusBadRequest)
			return
		}
		filter.SnapshotAtFrom = t
	}
	if v := q.Get("snapshotAtTo"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			h.JSONError(w, "snapshotAtTo must be RFC3339", http.StatusBadRequest)
			return
		}
		filter.SnapshotAtTo = t
	}

	switch q.Get("sortDesc") {
	case "true", "1":
		filter.SortDesc = true
	}

	if limit := q.Get("limit"); limit != "" {
		var l int
		if _, err := parsePositiveInt(limit, &l); err == nil {
			filter.Limit = l
		}
	}
	if offset := q.Get("offset"); offset != "" {
		var o int
		if _, err := parsePositiveInt(offset, &o); err == nil {
			filter.Offset = o
		}
	}

	archives, total, err := h.Service.ListHistory(r.Context(), filter)
	if h.HandleDomainError(w, err) {
		return
	}
	if archives == nil {
		archives = []*types.DiffArchive{}
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = len(archives)
	}

	h.JSONSuccess(w, ListHistoryResponse{
		Archives:   archives,
		TotalCount: total,
		Limit:      limit,
		Offset:     filter.Offset,
	})
}
