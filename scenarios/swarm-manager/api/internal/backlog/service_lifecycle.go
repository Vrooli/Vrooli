package backlog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/workshop"
)

// ResetArtifactScope is the service-owned representation of a derived
// artifact collection. The proposal package mirrors these stable strings.
type ResetArtifactScope string

const (
	ResetScopeWorkshop          ResetArtifactScope = "workshop"
	ResetScopeClarifications    ResetArtifactScope = "clarifications"
	ResetScopeReview            ResetArtifactScope = "review"
	ResetScopeHandoffExecutions ResetArtifactScope = "handoff_executions"
	ResetScopePlanUnbind        ResetArtifactScope = "plan_unbind"
)

// ResetArtifactsResult describes exactly which derived data was removed.
type ResetArtifactsResult struct {
	Item           BacklogItem          `json:"item"`
	Scopes         []ResetArtifactScope `json:"scopes"`
	DeletedRounds  int                  `json:"deleted_rounds,omitempty"`
	StatusReverted bool                 `json:"status_reverted"`
}

func validResetArtifactScope(scope ResetArtifactScope) bool {
	switch scope {
	case ResetScopeWorkshop, ResetScopeClarifications, ResetScopeReview, ResetScopeHandoffExecutions, ResetScopePlanUnbind:
		return true
	default:
		return false
	}
}

// ResetArtifacts removes only selected generated/derived artifact groups.
// The item spec remains intact except plan_unbind and ready→backlog, both of
// which are necessary to avoid presenting an invalid plan as executable.
func (s *Service) ResetArtifacts(ctx context.Context, kind BacklogKind, name string, scopes []ResetArtifactScope) (ResetArtifactsResult, error) {
	if len(scopes) == 0 {
		return ResetArtifactsResult{}, fmt.Errorf("reset artifacts requires at least one scope")
	}
	seen := make(map[ResetArtifactScope]struct{}, len(scopes))
	for _, scope := range scopes {
		if !validResetArtifactScope(scope) {
			return ResetArtifactsResult{}, fmt.Errorf("unknown reset scope %q", scope)
		}
		if _, exists := seen[scope]; exists {
			return ResetArtifactsResult{}, fmt.Errorf("duplicate reset scope %q", scope)
		}
		seen[scope] = struct{}{}
	}
	if s.activityChecker != nil && s.activityChecker.HasActiveAgent(ctx, string(kind), name) {
		return ResetArtifactsResult{}, fmt.Errorf("an agent is currently working on this item; try again after it finishes")
	}
	item, err := s.store.LoadItem(kind, name)
	if err != nil {
		return ResetArtifactsResult{}, err
	}
	itemDir := s.store.ItemDir(kind, name)
	result := ResetArtifactsResult{Scopes: append([]ResetArtifactScope(nil), scopes...)}
	for _, scope := range scopes {
		switch scope {
		case ResetScopeWorkshop:
			deliverable := ""
			if kind == KindResearch {
				deliverable = "conclusion.md"
			}
			deleted, resetErr := workshop.ResetWorkshop(itemDir, deliverable)
			if resetErr != nil {
				return ResetArtifactsResult{}, fmt.Errorf("reset workshop: %w", resetErr)
			}
			result.DeletedRounds += deleted
		case ResetScopeClarifications:
			if err := removeArtifactDir(itemDir, "clarifications"); err != nil {
				return ResetArtifactsResult{}, err
			}
		case ResetScopeReview:
			if err := removeArtifactDir(itemDir, "review"); err != nil {
				return ResetArtifactsResult{}, err
			}
		case ResetScopeHandoffExecutions:
			for _, dir := range []string{"handoff", "executions"} {
				if err := removeArtifactDir(itemDir, dir); err != nil {
					return ResetArtifactsResult{}, err
				}
			}
		case ResetScopePlanUnbind:
			item.PlanRef = nil
			item.PlanAcceptance = nil
		}
	}
	if item.Status == StatusReady {
		item.Status = StatusBacklog
		result.StatusReverted = true
	}
	_, planUnbound := seen[ResetScopePlanUnbind]
	if result.StatusReverted || planUnbound {
		item.Updated = time.Now().UTC().Format(time.RFC3339)
		if err := s.store.SaveItem(item); err != nil {
			return ResetArtifactsResult{}, fmt.Errorf("save reset item: %w", err)
		}
	}
	result.Item = item
	if s.invalidator != nil {
		s.invalidator.ScheduleAll()
	}
	return result, nil
}

func removeArtifactDir(itemDir, name string) error {
	if err := os.RemoveAll(filepath.Join(itemDir, name)); err != nil {
		return fmt.Errorf("remove %s artifacts: %w", name, err)
	}
	return nil
}

type itemDetacher interface {
	ForgetItem(milestoneName, ref string) error
}

// RecreateItem makes an active backlog clone with lineage and archives the
// source only after dependents and milestone membership point at the clone.
// Every write after creation has compensating rollback, preserving the source
// graph when a filesystem failure occurs mid-operation.
func (s *Service) RecreateItem(ctx context.Context, kind BacklogKind, name string) (BacklogItem, error) {
	if s.activityChecker != nil && s.activityChecker.HasActiveAgent(ctx, string(kind), name) {
		return BacklogItem{}, fmt.Errorf("an agent is currently working on this item; try again after it finishes")
	}
	source, err := s.store.LoadItem(kind, name)
	if err != nil {
		return BacklogItem{}, err
	}
	if source.ArchivedAt != nil {
		return BacklogItem{}, fmt.Errorf("cannot recreate archived item %s/%s", kind, name)
	}
	allStore, ok := s.store.(interface {
		LoadAll([]BacklogKind) ([]BacklogItem, error)
	})
	if !ok {
		return BacklogItem{}, fmt.Errorf("recreate item requires a list-capable store")
	}
	all, err := allStore.LoadAll(nil)
	if err != nil {
		return BacklogItem{}, fmt.Errorf("list dependents: %w", err)
	}
	cloneName := nextRecreatedName(source.Name, all)
	ref := string(kind) + "/" + name
	cloneRef := string(kind) + "/" + cloneName
	now := time.Now().UTC().Format(time.RFC3339)
	clone := BacklogItem{Name: cloneName, Title: source.Title, Description: source.Description, Status: StatusBacklog, Priority: source.Priority, Tags: append([]string(nil), source.Tags...), Created: now, Updated: now, Kind: source.Kind, DependsOn: append([]string(nil), source.DependsOn...), Milestone: source.Milestone, Effort: source.Effort, AcceptanceAllow: append([]string(nil), source.AcceptanceAllow...), AcceptanceDeny: append([]string(nil), source.AcceptanceDeny...), Creates: append([]string(nil), source.Creates...), SpawnedFrom: ref, FindingRef: source.FindingRef, Note: source.Note, SuggestedSkills: append([]string(nil), source.SuggestedSkills...), CreatedBy: source.CreatedBy}
	if err := s.Create(clone, CreationContext{Context: ctx, Source: SourceProposal}); err != nil {
		return BacklogItem{}, fmt.Errorf("create recreation clone: %w", err)
	}
	dependents := make([]BacklogItem, 0)
	for _, item := range all {
		if item.Name == source.Name && item.Kind == source.Kind {
			continue
		}
		for _, dep := range item.DependsOn {
			if dep == ref {
				dependents = append(dependents, item)
				break
			}
		}
	}
	restored := func() {
		for _, previous := range dependents {
			_ = s.store.SaveItem(previous)
		}
		if source.Milestone != "" {
			if detacher, ok := s.assigner.(itemDetacher); ok {
				_ = detacher.ForgetItem(source.Milestone, cloneRef)
			}
			_ = s.assigner.RememberItem(source.Milestone, ref)
		}
		_, _ = s.ArchiveItem(ctx, kind, cloneName, "recreation rollback")
	}
	for _, previous := range dependents {
		updated := previous
		updated.DependsOn = replaceDependency(updated.DependsOn, ref, cloneRef)
		if err := s.store.ValidateDependencies(updated.DependsOn); err != nil {
			restored()
			return BacklogItem{}, fmt.Errorf("validate retargeted dependent %s/%s: %w", updated.Kind, updated.Name, err)
		}
		updated.Updated = time.Now().UTC().Format(time.RFC3339)
		if err := s.store.SaveItem(updated); err != nil {
			restored()
			return BacklogItem{}, fmt.Errorf("save retargeted dependent %s/%s: %w", updated.Kind, updated.Name, err)
		}
	}
	if source.Milestone != "" {
		detacher, ok := s.assigner.(itemDetacher)
		if !ok {
			restored()
			return BacklogItem{}, fmt.Errorf("recreate item requires milestone detacher")
		}
		if err := detacher.ForgetItem(source.Milestone, ref); err != nil {
			restored()
			return BacklogItem{}, fmt.Errorf("move milestone membership: %w", err)
		}
	}
	if _, err := s.ArchiveItem(ctx, kind, name, "recreated as "+cloneRef); err != nil {
		restored()
		return BacklogItem{}, fmt.Errorf("archive recreated source: %w", err)
	}
	return s.store.LoadItem(kind, cloneName)
}

func nextRecreatedName(base string, all []BacklogItem) string {
	used := make(map[string]struct{}, len(all))
	for _, item := range all {
		used[item.Name] = struct{}{}
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

func replaceDependency(deps []string, oldRef, newRef string) []string {
	out := make([]string, 0, len(deps))
	seen := make(map[string]struct{}, len(deps))
	for _, dep := range deps {
		if strings.TrimSpace(dep) == oldRef {
			dep = newRef
		}
		if _, exists := seen[dep]; exists {
			continue
		}
		seen[dep] = struct{}{}
		out = append(out, dep)
	}
	return out
}
