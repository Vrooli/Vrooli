package proposals

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

func (a *Applier) applyAddEdge(ctx context.Context, from, to string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(from)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	for _, dep := range item.DependsOn {
		if dep == to {
			return fmt.Errorf("edge already exists: %s -> %s", from, to)
		}
	}
	deps := append(append([]string(nil), item.DependsOn...), to)
	if err := a.store.ValidateDependencies(deps); err != nil {
		return fmt.Errorf("depends_on: %w", err)
	}
	item.DependsOn = deps
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyRemoveEdge(ctx context.Context, from, to string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(from)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(item.DependsOn))
	removed := false
	for _, dep := range item.DependsOn {
		if dep == to {
			removed = true
			continue
		}
		filtered = append(filtered, dep)
	}
	if !removed {
		return fmt.Errorf("edge does not exist: %s -> %s", from, to)
	}
	item.DependsOn = filtered
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyMoveMilestone(ctx context.Context, ref, currentInit, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	dest := strings.TrimSpace(destination)

	if currentInit != "" {
		if err := a.assigner.ForgetItem(currentInit, ref); err != nil {
			return fmt.Errorf("detach from %s: %w", currentInit, err)
		}
	}

	if dest == "" {
		if _, _, err := a.store.ClearItemMilestone(kind, name, currentInit); err != nil {
			return fmt.Errorf("clear milestone field: %w", err)
		}
		item, loadErr := a.store.LoadItem(kind, name)
		if loadErr == nil {
			item.Updated = a.clock().UTC().Format(time.RFC3339)
			if saveErr := a.store.SaveItem(item); saveErr != nil {
				slog.Warn("proposals: persist updated timestamp failed", "err", saveErr, "kind", kind, "name", name)
			}
		}
		return nil
	}

	if _, err := a.store.SetItemMilestone(kind, name, dest); err != nil {
		// Roll back detach so the item isn't orphaned.
		if currentInit != "" {
			if remErr := a.assigner.RememberItem(currentInit, ref); remErr != nil {
				slog.Warn("proposals: rollback re-attach to original milestone failed", "err", remErr, "milestone", currentInit, "ref", ref)
			}
		}
		return fmt.Errorf("set milestone field: %w", err)
	}
	if err := a.assigner.RememberItem(dest, ref); err != nil {
		// Full rollback: the item now claims `dest` on disk but neither
		// milestone lists it. Restore the item field and re-attach to
		// the original milestone so the user sees pre-mutation state.
		if _, _, clearErr := a.store.ClearItemMilestone(kind, name, dest); clearErr != nil {
			return fmt.Errorf("attach to %s: %w; rollback clear failed: %v", dest, err, clearErr)
		}
		if currentInit != "" {
			if _, restoreErr := a.store.SetItemMilestone(kind, name, currentInit); restoreErr != nil {
				return fmt.Errorf("attach to %s: %w; rollback restore failed: %v", dest, err, restoreErr)
			}
			if remErr := a.assigner.RememberItem(currentInit, ref); remErr != nil {
				return fmt.Errorf("attach to %s: %w; rollback remember failed: %v", dest, err, remErr)
			}
		}
		return fmt.Errorf("attach to %s: %w", dest, err)
	}
	item, loadErr := a.store.LoadItem(kind, name)
	if loadErr == nil {
		item.Updated = a.clock().UTC().Format(time.RFC3339)
		if saveErr := a.store.SaveItem(item); saveErr != nil {
			slog.Warn("proposals: persist updated timestamp failed", "err", saveErr, "kind", kind, "name", name)
		}
	}
	return nil
}
