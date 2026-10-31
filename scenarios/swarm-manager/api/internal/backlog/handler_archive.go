package backlog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Delete deletes a backlog item and cascades referential integrity:
//   - Removes the item's "kind/name" ref from every other item's depends_on.
//   - Removes the ref from its enclosing milestone's items[] list.
//
// Cascade runs before the item file is deleted so that a partial failure
// leaves a consistent "item still exists, references intact" state. After
// the item file is removed, the depends_on sweep is run as a best-effort
// cleanup of refs that now point at a non-existent item.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "delete")
	if !ok {
		return
	}
	_, apiErr := h.deleteItem(kind, name)
	if apiErr != nil {
		apierr.MapError(w, "[backlog] delete", apiErr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteItem removes one item and its dependent references. A missing item is
// intentionally a successful no-op, matching DELETE's idempotent REST contract.
func (h *Handler) deleteItem(kind BacklogKind, name string) (bool, *apierr.DomainError) {
	existing, err := h.store.LoadItem(kind, name)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		slog.Error("failed to load item for delete", "name", name, "err", err)
		return false, apierr.Internal("failed to load backlog item")
	}

	ref := string(kind) + "/" + name
	if strings.TrimSpace(existing.Milestone) != "" && h.milestoneAssigner != nil {
		if err := h.milestoneAssigner.ForgetItem(existing.Milestone, ref); err != nil {
			slog.Error("failed to forget item from milestone", "ref", ref, "milestone", existing.Milestone, "err", err)
			return false, apierr.Internal("failed to update milestone membership")
		}
	}

	if err := h.store.DeleteItem(kind, name); err != nil {
		if existing.Milestone != "" && h.milestoneAssigner != nil {
			if rollbackErr := h.milestoneAssigner.RememberItem(existing.Milestone, ref); rollbackErr != nil {
				slog.Error("failed to roll back milestone membership after delete failure", "ref", ref, "err", rollbackErr)
			}
		}
		slog.Error("failed to delete item", "name", name, "err", err)
		return false, apierr.Internal("failed to delete backlog item")
	}

	if n, err := h.store.RemoveDependencyRef(ref); err != nil {
		slog.Error("failed to clean up dependency references", "ref", ref, "err", err)
	} else if n > 0 {
		slog.Info("cleaned up dependency references", "ref", ref, "updated_items", n)
	}

	slog.Info("item deleted", "name", name, "kind", kind)
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogDeleted(ref)
	}
	h.invalidateAllGraphLenses()
	return true, nil
}

// Archive sets archived_at on a backlog item, and settles its status first:
// archiving unfinished work means the operator is not going to do it, so the
// item transitions to `dropped` before archived_at is stamped. The status
// change is emitted so listeners see the transition.
//
// Archiving used to leave a non-review item's status untouched, which is how
// items ended up archived while still reading `backlog` — permanently blocking
// their dependents, since only a resolved status satisfies a dependency gate.
// Items already in a terminal status keep it: archiving completed work does
// not retroactively turn it into abandoned work.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "archive")
	if !ok {
		return
	}
	if !h.requireBookkeeping(w, r, kind, name) {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] archive", apierr.Internal("%s", err.Error()))
		return
	}

	if item.ArchivedAt != nil {
		resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
		if err := httputil.ProtoJSON(w, resp); err != nil {
			apierr.MapError(w, "[backlog] archive", apierr.Internal("failed to encode response"))
		}
		return
	}
	item, apiErr := h.archiveLoadedItem(r.Context(), item, nil)
	if apiErr != nil {
		apierr.MapError(w, "[backlog] archive", apiErr)
		return
	}
	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] archive", apierr.Internal("failed to encode response"))
	}
}

// archiveLoadedItem is the domain mutation shared by REST and typed archival.
// Callers establish object authority and any snapshot preconditions first.
func (h *Handler) archiveLoadedItem(ctx context.Context, item BacklogItem, expected *apipb.TerminalArchiveTarget) (BacklogItem, *apierr.DomainError) {
	if item.ArchivedAt != nil {
		return item, nil
	}
	kind, name := item.Kind, item.Name

	now := time.Now().UTC().Format(time.RFC3339)
	priorStatus := item.Status
	statusChanged := false
	if !IsTerminalStatus(priorStatus) {
		item.Status = StatusDropped
		statusChanged = true
	}
	item.ArchivedAt = &now
	item.Updated = now

	if expected != nil {
		writer, ok := h.store.(interface {
			ArchiveTerminalSnapshot(BacklogKind, string, string, string, string) (BacklogItem, error)
		})
		if !ok {
			return BacklogItem{}, apierr.Unavailable("conditional archive storage is unavailable")
		}
		stored, err := writer.ArchiveTerminalSnapshot(kind, name, expected.ExpectedStatus, expected.ExpectedUpdated, now)
		if errors.Is(err, ErrArchiveSnapshotChanged) {
			return BacklogItem{}, &apierr.DomainError{Status: http.StatusConflict, Message: "item changed before archival"}
		}
		if err != nil {
			return BacklogItem{}, apierr.Internal("conditional archive failed")
		}
		item = stored
	} else if err := h.store.SaveItem(item); err != nil {
		return BacklogItem{}, apierr.Internal("failed to save item")
	}

	entityID := string(kind) + "/" + name
	if h.eventLogger != nil {
		if statusChanged {
			h.eventLogger.EmitBacklogStatusChanged(ctx, entityID, string(priorStatus), string(item.Status))
		}
		if emitter, ok := h.eventLogger.(interface {
			EmitBacklogArchivedFromContext(context.Context, string, string, string)
		}); ok {
			emitter.EmitBacklogArchivedFromContext(ctx, entityID, string(priorStatus), now)
		} else {
			h.eventLogger.EmitBacklogArchived(entityID, string(priorStatus), now)
		}
	}
	if statusChanged && h.itemTerminalHandler != nil {
		h.itemTerminalHandler(ctx, string(kind), name, item.Status)
	}
	h.invalidateAllGraphLenses()

	return item, nil
}

// Unarchive clears archived_at on a backlog item.
func (h *Handler) Unarchive(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "unarchive")
	if !ok {
		return
	}
	if !h.requireBookkeeping(w, r, kind, name) {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] unarchive", apierr.Internal("%s", err.Error()))
		return
	}

	if item.ArchivedAt == nil {
		resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
		if err := httputil.ProtoJSON(w, resp); err != nil {
			apierr.MapError(w, "[backlog] unarchive", apierr.Internal("failed to encode response"))
		}
		return
	}

	prevArchivedAt := *item.ArchivedAt
	item.ArchivedAt = nil
	item.Updated = time.Now().UTC().Format(time.RFC3339)

	if err := h.store.SaveItem(item); err != nil {
		apierr.MapError(w, "[backlog] unarchive", apierr.Internal("failed to save item"))
		return
	}

	if h.eventLogger != nil {
		if emitter, ok := h.eventLogger.(interface {
			EmitBacklogUnarchivedFromContext(context.Context, string, string)
		}); ok {
			emitter.EmitBacklogUnarchivedFromContext(r.Context(), string(kind)+"/"+name, prevArchivedAt)
		} else {
			h.eventLogger.EmitBacklogUnarchived(string(kind)+"/"+name, prevArchivedAt)
		}
	}
	h.invalidateAllGraphLenses()

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] unarchive", apierr.Internal("failed to encode response"))
	}
}
