package backlog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Update updates an existing backlog item.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "update")
	if !ok {
		return
	}
	updated, apiErr := h.doUpdate(r.Context(), kind, name, r)
	if apiErr != nil {
		apierr.MapError(w, "[backlog] update", apiErr)
		return
	}
	resp := &apipb.BacklogItemResponse{Item: backlogToProto(updated)}
	h.invalidateAllGraphLenses()
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] update", apierr.Internal("failed to encode response"))
	}
}

// doUpdate performs all mutation logic for Update, returning the saved item or
// a typed API error. Extracting this helper collapses many scattered
// apierr.MapError+return pairs into a single error-return path at the call
// site, which is the dominant source of cyclomatic complexity in Update.
func (h *Handler) doUpdate(ctx context.Context, kind BacklogKind, name string, r *http.Request) (BacklogItem, *apierr.DomainError) {
	existing, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return BacklogItem{}, apierr.NotFound("backlog item not found")
		}
		slog.Error("failed to load item for update", "name", name, "err", err)
		return BacklogItem{}, apierr.Internal("%s", httputil.TruncateErrorMessage(err, 240))
	}

	update, fields, err := decodeUpdateBacklogPatch(r)
	if err != nil {
		return BacklogItem{}, apierr.BadRequest("%s", err.Error())
	}
	if validationErr := validateUpdateBacklogItemRequest(update, fields, existing.Kind, existing.Status); validationErr != "" {
		return BacklogItem{}, apierr.BadRequest("%s", validationErr)
	}

	oldStatus := existing.Status
	oldPriority := existing.Priority
	oldEffort := existing.Effort
	oldInitiative := existing.Initiative
	oldDependsOn := append([]string(nil), existing.DependsOn...)

	if fields.Has(updateFieldEffort) {
		normalized, err := validateEffort(update.GetEffort())
		if err != nil {
			return BacklogItem{}, apierr.BadRequest("%s", err.Error())
		}
		update.Effort = &normalized
	}

	applyUpdateBacklogPatch(&existing, update, fields)
	existing.Updated = time.Now().UTC().Format(time.RFC3339)

	if fields.Has(updateFieldInitiative) {
		if err := h.validateInitiativeReference(existing.Initiative); err != nil {
			return BacklogItem{}, apierr.BadRequest("%s", err.Error())
		}
	}

	if fields.Has(updateFieldDependsOn) && len(existing.DependsOn) > 0 {
		if err := h.store.ValidateDependencies(existing.DependsOn); err != nil {
			return BacklogItem{}, apierr.BadRequest("%s", err.Error())
		}
		if err := h.checkDependencyCycles(existing); err != nil {
			return BacklogItem{}, apierr.BadRequest("%s", err.Error())
		}
	}

	if apiErr := h.saveWithInitiativeUpdate(existing, kind, name, oldInitiative, fields); apiErr != nil {
		return BacklogItem{}, apiErr
	}

	h.logAndEmitUpdate(kind, name, oldStatus, existing.Status, oldPriority, existing.Priority, oldEffort, existing.Effort, oldInitiative, existing.Initiative, oldDependsOn, existing.DependsOn)
	h.maybeManuallyAcceptExecution(ctx, kind, name, oldStatus, existing.Status)
	return existing, nil
}

// saveWithInitiativeUpdate detaches the item from the old initiative (when
// changed), saves the item, rolls back on save failure, and attaches to the
// new initiative. Consolidating these three dependent operations eliminates
// duplicate error-path branches in doUpdate.
func (h *Handler) saveWithInitiativeUpdate(existing BacklogItem, kind BacklogKind, name, oldInitiative string, fields backlogUpdateFieldSet) *apierr.DomainError {
	ref := string(kind) + "/" + name
	initiativeChanged := fields.Has(updateFieldInitiative) && oldInitiative != existing.Initiative

	if initiativeChanged && h.initiativeAssigner != nil && oldInitiative != "" {
		if err := h.initiativeAssigner.ForgetItem(oldInitiative, ref); err != nil {
			slog.Error("failed to detach item from old initiative", "ref", ref, "initiative", oldInitiative, "err", err)
			return apierr.Internal("failed to update old initiative membership")
		}
	}

	if err := h.store.SaveItem(existing); err != nil {
		if initiativeChanged && h.initiativeAssigner != nil && oldInitiative != "" {
			if rErr := h.initiativeAssigner.RememberItem(oldInitiative, ref); rErr != nil {
				slog.Error("failed to re-attach to old initiative after save failure", "ref", ref, "err", rErr)
			}
		}
		slog.Error("failed to save item", "name", name, "err", err)
		return apierr.Internal("failed to save backlog item")
	}

	if initiativeChanged && h.initiativeAssigner != nil && existing.Initiative != "" {
		if err := h.initiativeAssigner.RememberItem(existing.Initiative, ref); err != nil {
			slog.Error("failed to attach item to new initiative", "ref", ref, "initiative", existing.Initiative, "err", err)
			return apierr.Internal("failed to update new initiative membership")
		}
	}
	return nil
}

// logAndEmitUpdate logs the update and emits analytics events for changed fields.
func (h *Handler) logAndEmitUpdate(
	kind BacklogKind, name string,
	oldStatus, newStatus BacklogStatus,
	oldPriority, newPriority int,
	oldEffort, newEffort string,
	oldInitiative, newInitiative string,
	oldDeps, newDeps []string,
) {
	if oldStatus != newStatus || oldPriority != newPriority {
		slog.Info("item updated", "name", name, "old_status", oldStatus, "new_status", newStatus, "old_priority", oldPriority, "new_priority", newPriority)
	} else {
		slog.Info("item updated", "name", name)
	}

	if h.eventLogger == nil {
		return
	}
	entityID := string(kind) + "/" + name
	if oldStatus != newStatus {
		h.eventLogger.EmitBacklogStatusChanged(entityID, string(oldStatus), string(newStatus))
	}
	if oldPriority != newPriority {
		h.eventLogger.EmitBacklogPriorityChanged(entityID, oldPriority, newPriority)
	}
	if oldEffort != newEffort {
		h.eventLogger.EmitBacklogEffortChanged(entityID, oldEffort, newEffort)
	}
	if oldInitiative != newInitiative {
		h.eventLogger.EmitBacklogInitiativeChanged(entityID, oldInitiative, newInitiative)
	}
	h.emitDependencyChanges(entityID, oldDeps, newDeps)
}

// maybeManuallyAcceptExecution recognizes a user-initiated override of the
// agent's verdict: the user changed the backlog item from failed to completed
// without re-running it. Flip the latest failed/needs_fixup execution to
// Completed with ManuallyAccepted=true so Agent-tab stats count the run as a
// success and surface the human override separately.
func (h *Handler) maybeManuallyAcceptExecution(
	ctx context.Context,
	kind BacklogKind, name string,
	oldStatus, newStatus BacklogStatus,
) {
	if oldStatus != StatusFailed || newStatus != StatusCompleted {
		return
	}
	if h.executionQueuer == nil {
		return
	}
	ref := string(kind) + "/" + name
	execID, accepted, err := h.executionQueuer.ManuallyAcceptLatestForBacklog(ctx, string(kind), name, "user", "user accepted failed run via backlog status change")
	if err != nil {
		slog.Error("manual-accept failed", "ref", ref, "err", err)
		return
	}
	if accepted {
		slog.Info("execution manually accepted", "ref", ref, "execution_id", execID)
	}
}
