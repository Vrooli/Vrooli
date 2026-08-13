package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
)

func (h *Handlers) HostInventory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	record, err := h.store.GetLatestHostInventorySnapshot(ctx)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("host_inventory", "retrieve host inventory", err))
		return
	}
	resp := hostinventory.InventoryResponse{Snapshot: record, Fresh: false, ProbeStatus: map[string]hostinventory.IntegrityProbeState{}}
	if record != nil {
		resp.AgeSeconds = int64(time.Since(record.CollectedAt) / time.Second)
		resp.Fresh = resp.AgeSeconds <= 300
		resp.ProbeStatus = record.Inventory.ProbeStatus
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		apierrors.LogError("host_inventory", "encode_response", err)
	}
}

func (h *Handlers) CollectHostInventory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	inv, err := h.hostCollector.Collect(ctx)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("host_inventory", "collect host inventory", err))
		return
	}
	record, changes, err := h.store.SaveHostInventorySnapshot(ctx, inv)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("host_inventory", "save host inventory", err))
		return
	}
	if changes == nil {
		changes = []hostinventory.Change{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"snapshot": record, "changes": changes}); err != nil {
		apierrors.LogError("host_inventory", "encode_collect_response", err)
	}
}

func (h *Handlers) HostInventoryChanges(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := parsePositiveInt(limitStr); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	changes, err := h.store.GetRecentHostInventoryChanges(ctx, limit)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("host_inventory", "retrieve host inventory changes", err))
		return
	}
	if changes == nil {
		changes = []hostinventory.Change{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"changes": changes, "total": len(changes)}); err != nil {
		apierrors.LogError("host_inventory", "encode_changes_response", err)
	}
}
