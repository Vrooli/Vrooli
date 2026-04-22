// DOC: docs/internal/SEAMS.md#workshop-auto-trigger
//
// Pending advance file mechanism for deferred workshop auto-advance.
// When auto_advance_delay_seconds > 0, WorkshopSave writes a pending advance
// file instead of spawning immediately. A background ticker fires the spawn
// when the delay expires. Users can cancel via a DELETE endpoint.
package backlog

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/storage"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

const pendingAdvanceFile = ".workshop-pending-advance.json"

// PendingAdvance describes a deferred auto-advance spawn.
type PendingAdvance struct {
	CreatedAt  time.Time `json:"created_at"`
	AdvanceAt  time.Time `json:"advance_at"`
	NextMode   string    `json:"next_mode"` // "workshop" | "finalize"
	RoundCount int       `json:"round_count"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
}

// writePendingAdvance atomically writes a pending advance file to the item dir.
func writePendingAdvance(itemDir string, pa PendingAdvance) error {
	path := itemDir + "/" + pendingAdvanceFile
	return storage.WriteJSONAtomic(path, pa)
}

// readPendingAdvance loads the pending advance file, returning nil if not found.
func readPendingAdvance(itemDir string) (*PendingAdvance, error) {
	path := itemDir + "/" + pendingAdvanceFile
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var pa PendingAdvance
	if err := json.Unmarshal(data, &pa); err != nil {
		return nil, err
	}
	return &pa, nil
}

// deletePendingAdvance removes the pending advance file. Returns true if a file was deleted.
func deletePendingAdvance(itemDir string) bool {
	path := itemDir + "/" + pendingAdvanceFile
	err := os.Remove(path)
	return err == nil
}

// hasPendingAdvance checks whether a pending advance file exists.
func hasPendingAdvance(itemDir string) bool {
	path := itemDir + "/" + pendingAdvanceFile
	_, err := os.Stat(path)
	return err == nil
}

// WorkshopCancelPendingAdvance cancels a pending auto-advance for a backlog item.
func (h *Handler) WorkshopCancelPendingAdvance(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "workshop-cancel-pending")
	if !ok {
		return
	}

	itemDir := h.store.ItemDir(kind, name)
	cancelled := deletePendingAdvance(itemDir)

	// Also unregister from ticker if running.
	if h.workshopTicker != nil {
		h.workshopTicker.Unregister(string(kind), name)
	}

	if cancelled {
		slog.Info("workshop pending advance cancelled", "kind", kind, "name", name)
	}

	resp := &apipb.WorkshopCancelPendingAdvanceResponse{Cancelled: cancelled}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] workshop-cancel-pending", apierr.Internal("failed to encode response"))
	}
}
