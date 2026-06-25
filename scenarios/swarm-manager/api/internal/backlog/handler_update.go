package backlog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/workshop"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Update updates an existing backlog item.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "update")
	if !ok {
		return
	}

	existing, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] update", apierr.NotFound("backlog item not found"))
			return
		}
		slog.Error("failed to load item for update", "name", name, "err", err)
		apierr.MapError(w, "[backlog] update", apierr.Internal("%s", httputil.TruncateErrorMessage(err, 240)))
		return
	}

	update, fields, err := decodeUpdateBacklogPatch(r)
	if err != nil {
		apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
		return
	}
	if validationErr := validateUpdateBacklogItemRequest(update, fields, existing.Kind, existing.Status); validationErr != "" {
		apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", validationErr))
		return
	}

	oldStatus := existing.Status
	oldPriority := existing.Priority
	oldEffort := existing.Effort
	oldInitiative := existing.Initiative
	oldDependsOn := append([]string(nil), existing.DependsOn...)

	if fields.Has(updateFieldEffort) {
		normalized, err := validateEffort(update.GetEffort())
		if err != nil {
			apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
			return
		}
		update.Effort = &normalized
	}

	applyUpdateBacklogPatch(&existing, update, fields)
	existing.Updated = time.Now().UTC().Format(time.RFC3339)

	if fields.Has(updateFieldInitiative) {
		if err := h.validateInitiativeReference(existing.Initiative); err != nil {
			apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
			return
		}
	}

	if fields.Has(updateFieldDependsOn) && len(existing.DependsOn) > 0 {
		if err := h.store.ValidateDependencies(existing.DependsOn); err != nil {
			apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
			return
		}
		if err := h.checkDependencyCycles(existing); err != nil {
			apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
			return
		}
	}

	ref := string(kind) + "/" + name
	initiativeChanged := fields.Has(updateFieldInitiative) && oldInitiative != existing.Initiative
	if initiativeChanged && h.initiativeAssigner != nil && oldInitiative != "" {
		if err := h.initiativeAssigner.ForgetItem(oldInitiative, ref); err != nil {
			slog.Error("failed to detach item from old initiative", "ref", ref, "initiative", oldInitiative, "err", err)
			apierr.MapError(w, "[backlog] update", apierr.Internal("failed to update old initiative membership"))
			return
		}
	}

	if err := h.store.SaveItem(existing); err != nil {
		if initiativeChanged && h.initiativeAssigner != nil && oldInitiative != "" {
			if rErr := h.initiativeAssigner.RememberItem(oldInitiative, ref); rErr != nil {
				slog.Error("failed to re-attach to old initiative after save failure", "ref", ref, "err", rErr)
			}
		}
		slog.Error("failed to save item", "name", name, "err", err)
		apierr.MapError(w, "[backlog] update", apierr.Internal("failed to save backlog item"))
		return
	}

	if initiativeChanged && h.initiativeAssigner != nil && existing.Initiative != "" {
		if err := h.initiativeAssigner.RememberItem(existing.Initiative, ref); err != nil {
			slog.Error("failed to attach item to new initiative", "ref", ref, "initiative", existing.Initiative, "err", err)
			apierr.MapError(w, "[backlog] update", apierr.Internal("failed to update new initiative membership"))
			return
		}
	}

	h.logAndEmitUpdate(kind, name, oldStatus, existing.Status, oldPriority, existing.Priority, oldEffort, existing.Effort, oldInitiative, existing.Initiative, oldDependsOn, existing.DependsOn)
	h.maybeManuallyAcceptExecution(r.Context(), kind, name, oldStatus, existing.Status)
	h.maybeCascadeWorkshop(oldStatus, existing)

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(existing)}
	h.invalidateAllGraphLenses()
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] update", apierr.Internal("failed to encode response"))
	}
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

// maybeCascadeWorkshop triggers workshops for dependents when a status
// transition unblocks them.
func (h *Handler) maybeCascadeWorkshop(oldStatus BacklogStatus, item BacklogItem) {
	if oldStatus == item.Status {
		return
	}
	if !blockingDepStatuses[oldStatus] || blockingDepStatuses[item.Status] {
		return
	}
	cfg, cfgErr := settings.NewStore("").Load()
	if cfgErr != nil {
		slog.Warn("cascade settings load error, using defaults", "err", cfgErr)
		cfg = settings.DefaultSettings()
	}
	if workshop.ShouldCascade(cfg.AutoCascadeWorkshop) {
		go h.cascadeWorkshopTrigger(item)
	}
}
