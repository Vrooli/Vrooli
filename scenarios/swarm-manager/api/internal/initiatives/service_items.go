package initiatives

import (
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
)

// parsedItem is a parsed "kind/name" backlog reference used by the AddItems /
// RemoveItems cascade paths.
type parsedItem struct {
	kind      backlog.BacklogKind
	localName string
	ref       string
}

// parseStrictItemRefs parses "kind/name" references, rejecting any malformed or
// unknown-kind reference. Used by AddItems where bad input must fail loudly.
func parseStrictItemRefs(items []string) ([]parsedItem, error) {
	parsed := make([]parsedItem, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, "/", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("invalid item reference %q: expected format kind/name", item)
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid item reference %q: %w", item, err)
		}
		parsed = append(parsed, parsedItem{kind: kind, localName: parts[1], ref: item})
	}
	return parsed, nil
}

// assertItemsUnclaimed rejects any parsed item that already belongs to a
// different initiative. Items that fail to load (e.g. not yet created) and
// orphan items are tolerated.
func (s *Service) assertItemsUnclaimed(name string, parsed []parsedItem) error {
	if s.backlogLoader == nil {
		return nil
	}
	for _, p := range parsed {
		item, err := s.backlogLoader.LoadItem(p.kind, p.localName)
		if err != nil {
			continue
		}
		current := strings.TrimSpace(item.Initiative)
		if current != "" && current != name {
			return fmt.Errorf("item %q already belongs to initiative %q; use PATCH on the item to move it", p.ref, current)
		}
	}
	return nil
}

// AddItems appends items to an initiative, deduplicating. Each item must be
// in "kind/name" format (e.g., "idea/my-feature"). Maintains symmetry with
// the item side: items already attached to a different initiative are
// rejected; orphan items (with an empty initiative field) have their
// initiative field set to this name so the two references stay in sync.
func (s *Service) AddItems(name string, items []string) error {
	parsed, err := parseStrictItemRefs(items)
	if err != nil {
		return err
	}

	if err := s.assertItemsUnclaimed(name, parsed); err != nil {
		return err
	}

	init, err := s.store.Load(name)
	if err != nil {
		return err
	}
	existing := make(map[string]bool, len(init.Items))
	for _, item := range init.Items {
		existing[item] = true
	}
	added := make([]parsedItem, 0, len(parsed))
	for _, p := range parsed {
		if existing[p.ref] {
			continue
		}
		init.Items = append(init.Items, p.ref)
		existing[p.ref] = true
		added = append(added, p)
	}
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}

	if s.backlogLoader != nil {
		for _, p := range added {
			if _, err := s.backlogLoader.SetItemInitiative(p.kind, p.localName, name); err != nil {
				// Item may not exist yet (e.g., batch create writes items
				// separately); not-found is not an error here.
				continue
			}
		}
	}

	if s.eventLogger != nil {
		for _, p := range added {
			s.eventLogger.EmitInitiativeItemAdded(name, p.ref)
		}
	}
	s.invalidateTopologyGraph()
	return nil
}

// RemoveItems removes items from an initiative and clears the item's
// initiative field if it currently equals this initiative, maintaining
// two-way referential integrity.
func (s *Service) RemoveItems(name string, items []string) error {
	parsed := make([]parsedItem, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, "/", 2)
		if len(parts) != 2 {
			parsed = append(parsed, parsedItem{ref: item})
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			parsed = append(parsed, parsedItem{ref: item})
			continue
		}
		parsed = append(parsed, parsedItem{kind: kind, localName: parts[1], ref: item})
	}

	init, err := s.store.Load(name)
	if err != nil {
		return err
	}
	remove := make(map[string]bool, len(parsed))
	for _, p := range parsed {
		remove[p.ref] = true
	}
	filtered := make([]string, 0, len(init.Items))
	for _, item := range init.Items {
		if !remove[item] {
			filtered = append(filtered, item)
		}
	}
	init.Items = filtered
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}

	if s.backlogLoader != nil {
		for _, p := range parsed {
			if p.localName == "" || p.kind == "" {
				continue
			}
			if _, _, err := s.backlogLoader.ClearItemInitiative(p.kind, p.localName, name); err != nil {
				continue
			}
		}
	}

	if s.eventLogger != nil {
		for _, p := range parsed {
			if remove[p.ref] {
				s.eventLogger.EmitInitiativeItemRemoved(name, p.ref)
			}
		}
	}
	s.invalidateTopologyGraph()
	return nil
}

// RememberItem appends a single ref to the initiative's items[] list if not
// already present. This is a one-way helper: it does not modify the item's
// initiative field. Used by single-item create/patch cascade, which writes
// the item's initiative field itself via SaveItem.
func (s *Service) RememberItem(initiativeName, ref string) error {
	init, err := s.store.Load(initiativeName)
	if err != nil {
		return err
	}
	for _, existing := range init.Items {
		if existing == ref {
			return nil
		}
	}
	init.Items = append(init.Items, ref)
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeItemAdded(initiativeName, ref)
	}
	s.invalidateTopologyGraph()
	return nil
}

// ForgetItem removes a single ref from the initiative's items[] list. This
// is a one-way helper: it does not modify the item's initiative field. Used
// by single-item delete/patch cascade, where the item file is already gone
// or its initiative field is written separately.
func (s *Service) ForgetItem(initiativeName, ref string) error {
	if !s.store.Exists(initiativeName) {
		return nil
	}
	init, err := s.store.Load(initiativeName)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(init.Items))
	removed := false
	for _, existing := range init.Items {
		if existing == ref {
			removed = true
			continue
		}
		filtered = append(filtered, existing)
	}
	if !removed {
		return nil
	}
	init.Items = filtered
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeItemRemoved(initiativeName, ref)
	}
	s.invalidateTopologyGraph()
	return nil
}
