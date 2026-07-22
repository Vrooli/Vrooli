package proposals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
)

func (a *Applier) applyAddItem(ctx context.Context, spec ItemSpec, source Source) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, err := backlog.ParseBacklogKind(spec.Kind)
	if err != nil {
		return err
	}

	now := a.clock().UTC().Format(time.RFC3339)
	item := backlog.BacklogItem{
		Name:            spec.Name,
		Title:           strings.TrimSpace(spec.Title),
		Description:     spec.Description,
		Kind:            kind,
		Status:          backlog.StatusBacklog,
		Priority:        spec.Priority,
		Tags:            append([]string(nil), spec.Tags...),
		DependsOn:       append([]string(nil), spec.DependsOn...),
		Effort:          strings.ToUpper(strings.TrimSpace(spec.Effort)),
		Milestone:      source.MilestoneName,
		AcceptanceAllow: append([]string(nil), spec.AcceptanceAllow...),
		AcceptanceDeny:  append([]string(nil), spec.AcceptanceDeny...),
		Note:            spec.Note,
		Created:         now,
		Updated:         now,
	}
	if item.Priority == 0 {
		item.Priority = DefaultItemPriority
	}

	cc := backlog.CreationContext{
		Source:          backlog.SourceProposal,
		DecidedBy:       source.DecidedBy,
		FeedbackRoundID: source.FeedbackRoundID,
		ReviewRoundID:   source.ReviewRoundID,
		RoundNumber:     source.RoundNumber,
		RoundSlug:       source.RoundSlug,
		Entrypoint:      source.Entrypoint,
		// Apply re-validates depends_on against ValidateDependencies via
		// CurrentState; the per-item cycle check inside Service is
		// redundant here (and would require a full LoadAll on a path
		// that already paid the cost in Validate). Skip it.
		SkipCycleCheck: true,
		// The Applier batches graph invalidation across the whole
		// mutation set in Apply (see ScheduleAll at the bottom of
		// the loop) so individual add_item ops should not fire one
		// re-projection each.
		SkipGraphInvalidation: true,
	}
	if err := a.creator.Create(item, cc); err != nil {
		if errors.Is(err, backlog.ErrItemExists) {
			return fmt.Errorf("%w: %s", ErrDuplicateItem, spec.Ref())
		}
		return err
	}
	return nil
}

func (a *Applier) applyUpdateItem(ctx context.Context, ref string, patch *ItemPatch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	// Dependency cycles are a business-level concern the applier enforces
	// before delegating to the shared patch helper — ApplyItemPatch is
	// intentionally semantics-free about validity.
	if patch.DependsOn != nil && len(*patch.DependsOn) > 0 {
		if err := a.store.ValidateDependencies(*patch.DependsOn); err != nil {
			return fmt.Errorf("depends_on: %w", err)
		}
	}
	backlog.ApplyItemPatch(&item, toBacklogPatch(patch))
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

// toBacklogPatch converts proposals' ItemPatch (the wire shape an agent
// produces) into backlog.ItemPatch (the shared mutation primitive). Ops
// that touch status/milestone/spawned_from are excluded here — those
// have dedicated mutations so misuse surfaces as an unknown op instead
// of a silent field smuggle.
func toBacklogPatch(p *ItemPatch) backlog.ItemPatch {
	if p == nil {
		return backlog.ItemPatch{}
	}
	out := backlog.ItemPatch{
		Title:       p.Title,
		Description: p.Description,
		Priority:    p.Priority,
		Effort:      p.Effort,
		Note:        p.Note,
	}
	if p.Tags != nil {
		cp := append([]string(nil), *p.Tags...)
		out.Tags = &cp
	}
	if p.DependsOn != nil {
		cp := append([]string(nil), *p.DependsOn...)
		out.DependsOn = &cp
	}
	if p.AcceptanceAllow != nil {
		cp := append([]string(nil), *p.AcceptanceAllow...)
		out.AcceptanceAllow = &cp
	}
	if p.AcceptanceDeny != nil {
		cp := append([]string(nil), *p.AcceptanceDeny...)
		out.AcceptanceDeny = &cp
	}
	return out
}

func (a *Applier) applyChangeStatus(ctx context.Context, ref, status string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	newStatus := backlog.BacklogStatus(strings.ToLower(strings.TrimSpace(status)))
	if backlog.IsTerminalStatus(newStatus) {
		return fmt.Errorf("%w: %s", ErrTerminalStatusWrite, newStatus)
	}
	item.Status = newStatus
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyChangePriority(ctx context.Context, ref string, priority int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	item.Priority = priority
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyArchive(ctx context.Context, ref string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	if item.ArchivedAt != nil && *item.ArchivedAt != "" {
		return nil
	}
	ts := a.clock().UTC().Format(time.RFC3339)
	item.ArchivedAt = &ts
	item.Updated = ts
	return a.store.SaveItem(item)
}

func (a *Applier) applyInterrupt(ctx context.Context, ref string) error {
	if a.cancel == nil {
		return errors.New("interrupt_in_progress requires an ExecutionCanceller; none wired")
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	return a.cancel.CancelForBacklog(ctx, string(kind), name)
}
