package initiatives

import (
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
)

// Delete removes an initiative and cascades referential integrity:
//   - Every member item has its `initiative` field cleared (orphaned, not deleted).
//   - Every other initiative that referenced this one via `depends_on` has the
//     reference removed.
//
// The cascade is best-effort atomic: side effects are captured up front so a
// failure mid-cascade can be rolled back. If the final store.Delete fails,
// prior cascades are reverted.
func (s *Service) Delete(name string) error {
	if !s.store.Exists(name) {
		return nil // idempotent
	}
	init, err := s.store.Load(name)
	if err != nil {
		return fmt.Errorf("load initiative before delete: %w", err)
	}

	refs := parseInitiativeItemRefs(init.Items)

	cleared, err := s.cascadeClearItemInitiatives(name, refs)
	if err != nil {
		return err
	}

	all, err := s.store.LoadAll()
	if err != nil {
		s.rollbackClearedInitiatives(name, cleared)
		return fmt.Errorf("cascade: load initiatives for depends_on scrub: %w", err)
	}

	scrubbed, err := s.cascadeScrubDependsOn(name, all)
	if err != nil {
		s.rollbackScrubbedDependsOn(scrubbed)
		s.rollbackClearedInitiatives(name, cleared)
		return err
	}

	if err := s.store.Delete(name); err != nil {
		s.rollbackScrubbedDependsOn(scrubbed)
		s.rollbackClearedInitiatives(name, cleared)
		return err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeArchived(name, init.Status, "")
	}
	s.invalidateTopologyGraph()
	return nil
}

// itemRef is a parsed "kind/name" member-item reference used during cascade.
type itemRef struct {
	kind      backlog.BacklogKind
	localName string
	ref       string
}

// parseInitiativeItemRefs parses raw "kind/name" item strings, silently
// skipping any that are malformed or reference an unknown kind.
func parseInitiativeItemRefs(items []string) []itemRef {
	refs := make([]itemRef, 0, len(items))
	for _, raw := range items {
		parts := strings.SplitN(raw, "/", 2)
		if len(parts) != 2 {
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			continue
		}
		refs = append(refs, itemRef{kind: kind, localName: parts[1], ref: raw})
	}
	return refs
}

// cascadeClearItemInitiatives clears the initiative field on every member item.
// On failure it rolls back any items already cleared and returns the error.
func (s *Service) cascadeClearItemInitiatives(name string, refs []itemRef) ([]itemRef, error) {
	cleared := make([]itemRef, 0, len(refs))
	if s.backlogLoader == nil {
		return cleared, nil
	}
	for _, r := range refs {
		_, changed, err := s.backlogLoader.ClearItemInitiative(r.kind, r.localName, name)
		if err != nil {
			s.rollbackClearedInitiatives(name, cleared)
			return nil, fmt.Errorf("cascade: clear initiative on %s: %w", r.ref, err)
		}
		if changed {
			cleared = append(cleared, r)
		}
	}
	return cleared, nil
}

// rollbackClearedInitiatives best-effort restores the initiative field on
// items previously cleared by cascadeClearItemInitiatives.
func (s *Service) rollbackClearedInitiatives(name string, cleared []itemRef) {
	for _, done := range cleared {
		if _, setErr := s.backlogLoader.SetItemInitiative(done.kind, done.localName, name); setErr != nil {
			// Keep rollback best-effort; failures here are non-fatal.
			_ = setErr
		}
	}
}

// depScrub captures an initiative's depends_on list prior to scrubbing so it
// can be restored on rollback.
type depScrub struct {
	initName string
	oldDeps  []string
}

// cascadeScrubDependsOn removes name from every other initiative's depends_on.
// On failure it returns the partially-scrubbed list so the caller can roll back.
func (s *Service) cascadeScrubDependsOn(name string, all []Initiative) ([]depScrub, error) {
	scrubbed := make([]depScrub, 0)
	for i := range all {
		other := &all[i]
		if other.Name == name {
			continue
		}
		if !stringSliceContains(other.DependsOn, name) {
			continue
		}
		oldDeps := append([]string(nil), other.DependsOn...)
		filtered := make([]string, 0, len(other.DependsOn))
		for _, d := range other.DependsOn {
			if d != name {
				filtered = append(filtered, d)
			}
		}
		other.DependsOn = filtered
		other.Updated = time.Now().UTC().Format(time.RFC3339)
		if saveErr := s.store.Save(other); saveErr != nil {
			return scrubbed, fmt.Errorf("cascade: scrub depends_on from %q: %w", other.Name, saveErr)
		}
		scrubbed = append(scrubbed, depScrub{initName: other.Name, oldDeps: oldDeps})
	}
	return scrubbed, nil
}

// rollbackScrubbedDependsOn best-effort restores depends_on lists scrubbed by
// cascadeScrubDependsOn.
func (s *Service) rollbackScrubbedDependsOn(scrubbed []depScrub) {
	for _, sc := range scrubbed {
		if rolled, rErr := s.store.Load(sc.initName); rErr == nil {
			rolled.DependsOn = sc.oldDeps
			_ = s.store.Save(rolled)
		}
	}
}

func stringSliceContains(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}

// Replace writes a full initiative snapshot, used for internal rollback flows.
func (s *Service) Replace(init Initiative) error {
	if strings.TrimSpace(init.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(init.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if !ValidateStatus(strings.TrimSpace(init.Status)) {
		return fmt.Errorf("invalid status %q: must be active or completed", init.Status)
	}
	if !ValidatePriority(init.Priority) {
		return fmt.Errorf("invalid priority %d: must be 0 (unset) or 1-10", init.Priority)
	}
	init.Name = strings.TrimSpace(init.Name)
	init.Title = strings.TrimSpace(init.Title)
	init.Description = strings.TrimSpace(init.Description)
	init.Status = strings.TrimSpace(init.Status)
	init.Mode = NormalizeMode(init.Mode)
	if !ValidateMode(init.Mode) {
		return fmt.Errorf("invalid operating mode %q: must be one of %s", init.Mode, OperatingModeList())
	}
	init.DependsOn = normalizeDependsOn(init.DependsOn)
	init.AcceptanceCriteria = normalizeStringList(init.AcceptanceCriteria)
	if err := s.validateDependsOn(init.Name, init.DependsOn); err != nil {
		return err
	}
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(&init); err != nil {
		return err
	}
	s.invalidateTopologyGraph()
	return nil
}
