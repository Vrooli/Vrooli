package initiatives

import (
	"context"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
)

// RecreateInitiative creates an active lineage-preserving successor, moves
// members to it, and archives the source. Each member write is restored and
// the successor archived if a later write fails.
func (s *Service) RecreateInitiative(ctx context.Context, name string) error {
	source, err := s.store.Load(strings.TrimSpace(name))
	if err != nil {
		return err
	}
	if source.ArchivedAt != nil {
		return fmt.Errorf("cannot recreate archived initiative %q", name)
	}
	if s.activityChecker != nil {
		for _, ref := range source.Items {
			kind, itemName, parseErr := initiativeItemRef(ref)
			if parseErr != nil {
				return parseErr
			}
			if s.activityChecker.HasActiveAgent(ctx, string(kind), itemName) {
				return fmt.Errorf("an agent is currently working on item %s; try again after it finishes", ref)
			}
		}
	}
	all, err := s.store.LoadAll()
	if err != nil {
		return err
	}
	cloneName := nextRecreatedInitiativeName(source.Name, all)
	now := time.Now().UTC().Format(time.RFC3339)
	clone := *source
	clone.Name, clone.Status, clone.Created, clone.Updated = cloneName, InitiativeStatusActive, now, now
	clone.ArchivedAt = nil
	clone.SpawnedFrom = source.Name
	if _, err := s.Create(CreateRequest{Name: clone.Name, Title: clone.Title, Description: clone.Description, Status: InitiativeStatusActive, Priority: clone.Priority, DependsOn: append([]string(nil), clone.DependsOn...), Items: append([]string(nil), clone.Items...), AcceptanceCriteria: append([]string(nil), clone.AcceptanceCriteria...), CreatedBy: clone.CreatedBy, PlanRef: clone.PlanRef}); err != nil {
		return fmt.Errorf("create initiative recreation clone: %w", err)
	}
	createdClone, err := s.store.Load(cloneName)
	if err != nil {
		return fmt.Errorf("load initiative recreation clone: %w", err)
	}
	createdClone.SpawnedFrom = source.Name
	if err := s.store.Save(createdClone); err != nil {
		return fmt.Errorf("stamp initiative recreation lineage: %w", err)
	}
	moved := make([]itemRef, 0, len(source.Items))
	rollback := func() {
		for _, ref := range moved {
			_, _ = s.backlogLoader.SetItemInitiative(ref.kind, ref.localName, source.Name)
		}
		if cloned, loadErr := s.store.Load(cloneName); loadErr == nil {
			archivedAt := time.Now().UTC().Format(time.RFC3339)
			cloned.ArchivedAt, cloned.Updated = &archivedAt, archivedAt
			_ = s.store.Save(cloned)
		}
	}
	if s.backlogLoader == nil && len(source.Items) > 0 {
		rollback()
		return fmt.Errorf("recreate initiative requires backlog loader")
	}
	for _, raw := range source.Items {
		kind, itemName, parseErr := initiativeItemRef(raw)
		if parseErr != nil {
			rollback()
			return parseErr
		}
		if _, err := s.backlogLoader.SetItemInitiative(kind, itemName, cloneName); err != nil {
			rollback()
			return fmt.Errorf("move member %s: %w", raw, err)
		}
		moved = append(moved, itemRef{kind: kind, localName: itemName, ref: raw})
	}
	archivedAt := time.Now().UTC().Format(time.RFC3339)
	source.ArchivedAt, source.Updated = &archivedAt, archivedAt
	if err := s.store.Save(source); err != nil {
		rollback()
		return fmt.Errorf("archive source initiative: %w", err)
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeArchived(source.Name, source.Status, archivedAt)
	}
	s.invalidateTopologyGraph()
	return nil
}

func initiativeItemRef(raw string) (backlog.BacklogKind, string, error) {
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid initiative item ref %q", raw)
	}
	kind, err := backlog.ParseBacklogKind(parts[0])
	if err != nil {
		return "", "", err
	}
	return kind, parts[1], nil
}

func nextRecreatedInitiativeName(base string, all []Initiative) string {
	used := make(map[string]struct{}, len(all))
	for _, init := range all {
		used[init.Name] = struct{}{}
	}
	for n := 1; ; n++ {
		candidate := base + "-recreated"
		if n > 1 {
			candidate += fmt.Sprintf("-%d", n)
		}
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}
