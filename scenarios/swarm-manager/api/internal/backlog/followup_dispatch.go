package backlog

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/followup"
	"swarm-manager/internal/httputil"
)

type authorFollowUpRequest struct {
	FollowUp *FollowUp `json:"follow_up"`
}

// AuthorFollowUp repairs a legacy needs_followup item by recording the same
// typed instruction used by review-decide. The next action then becomes the
// normal dispatch_followup action instead of leaving an operator dead end.
func (h *Handler) AuthorFollowUp(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "author-follow-up")
	if !ok {
		return
	}
	var req authorFollowUpRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil || validateFollowUp(req.FollowUp) != nil {
		message := "follow_up must include steering and a valid disposition"
		if err == nil {
			message = validateFollowUp(req.FollowUp).Error()
		}
		apierr.MapError(w, "[backlog] author-follow-up", apierr.BadRequest("%s", message))
		return
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		apierr.MapError(w, "[backlog] author-follow-up", apierr.NotFound("backlog item not found"))
		return
	}
	if item.Status != StatusNeedsFollowup {
		apierr.MapError(w, "[backlog] author-follow-up", apierr.BadRequest("item is not awaiting follow-up"))
		return
	}
	item.PendingFollowUp = req.FollowUp
	item.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := h.store.SaveItem(item); err != nil {
		apierr.MapError(w, "[backlog] author-follow-up", apierr.Internal("failed to save follow-up instruction"))
		return
	}
	_ = httputil.JSON(w, map[string]any{"item": item})
}

// DispatchFollowUp resolves the persisted recovery instruction on a
// needs_followup item. The instruction is consumed exactly once on successful
// dispatch so the next-action projection cannot offer a stale second launch.
func (h *Handler) DispatchFollowUp(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "dispatch-follow-up")
	if !ok {
		return
	}
	item, instruction, err := h.DispatchPendingFollowUp(r.Context(), kind, name)
	if err != nil {
		apierr.MapError(w, "[backlog] dispatch-follow-up", err)
		return
	}
	if err := httputil.JSON(w, map[string]any{"item": item, "dispatched": true, "disposition": instruction.Disposition}); err != nil {
		apierr.MapError(w, "[backlog] dispatch-follow-up", apierr.Internal("failed to encode response"))
	}
}

// DispatchPendingFollowUp is the domain mutation for follow_up.dispatch.
func (h *Handler) DispatchPendingFollowUp(ctx context.Context, kind BacklogKind, name string) (BacklogItem, FollowUp, error) {
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		return BacklogItem{}, FollowUp{}, apierr.NotFound("backlog item not found")
	}
	if item.Status != StatusNeedsFollowup || item.PendingFollowUp == nil {
		return BacklogItem{}, FollowUp{}, apierr.BadRequest("item has no pending follow-up instruction")
	}
	if err := validateFollowUp(item.PendingFollowUp); err != nil {
		return BacklogItem{}, FollowUp{}, apierr.BadRequest("invalid pending follow-up: %s", err)
	}
	instruction := *item.PendingFollowUp
	switch instruction.Disposition {
	case FollowUpRun:
		if h.followUpDispatcher == nil {
			return BacklogItem{}, FollowUp{}, apierr.Unavailable("execution follow-up is not available")
		}
		if err := h.followUpDispatcher.DispatchFollowUp(ctx, kind, name, instruction.Steering); err != nil {
			return BacklogItem{}, FollowUp{}, apierr.BadRequest("could not start follow-up: %s", err)
		}
		item.Status = StatusQueued
	case FollowUpReplan:
		item.Status = StatusBacklog
		item.PlanAcceptance = nil
		item.Note = appendNote(item.Note, "Follow-up replan: "+strings.TrimSpace(instruction.Steering))
	case FollowUpNewItems:
		if err := h.createFollowUpItems(ctx, item, instruction.Items); err != nil {
			return BacklogItem{}, FollowUp{}, apierr.BadRequest("could not create follow-up items: %s", err)
		}
		item.Status = StatusCompleted
	default:
		return BacklogItem{}, FollowUp{}, apierr.BadRequest("unknown follow-up disposition")
	}
	item.PendingFollowUp = nil
	item.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := h.store.SaveItem(item); err != nil {
		return BacklogItem{}, FollowUp{}, apierr.Internal("failed to persist dispatched follow-up")
	}
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogStatusChanged(string(kind)+"/"+name, string(StatusNeedsFollowup), string(item.Status))
	}
	return item, instruction, nil
}

func (h *Handler) createFollowUpItems(ctx context.Context, parent BacklogItem, specs []followup.ItemSpec) error {
	if h.lifecycleService == nil {
		return fmt.Errorf("backlog lifecycle is not available")
	}
	if len(specs) == 0 {
		return fmt.Errorf("new_items must not be empty")
	}
	for _, spec := range specs {
		kind, err := ParseBacklogKind(strings.TrimSpace(spec.Kind))
		if err != nil {
			return err
		}
		priority := spec.Priority
		if priority == 0 {
			priority = 5
		}
		child := BacklogItem{Name: strings.TrimSpace(spec.Name), Title: strings.TrimSpace(spec.Title), Description: strings.TrimSpace(spec.Description), Kind: kind, Status: StatusBacklog, Priority: priority, DependsOn: spec.DependsOn, Milestone: parent.Milestone, Note: "Follow-up from " + string(parent.Kind) + "/" + parent.Name + ": " + strings.TrimSpace(parent.PendingFollowUp.Steering)}
		if err := h.lifecycleService.Create(child, CreationContext{Context: ctx, Source: SourceProposal, DecidedBy: "operator-follow-up", Entrypoint: "review.follow_up"}); err != nil {
			return err
		}
	}
	return nil
}
