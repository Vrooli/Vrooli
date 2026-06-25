package initiatives

import (
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/pathutil"
)

// ComputeRollup loads each referenced backlog item and aggregates status
// counts. Items that fail to load are counted as pending.
func (s *Service) ComputeRollup(init *Initiative) (*RollupStatus, error) {
	rollup, _ := s.aggregateInitiativeData(init)
	return rollup, nil
}

// aggregateInitiativeData loads each referenced backlog item once and returns
// both the rollup and the deduped list of scenarios targeted by the item's
// acceptance_allow globs. Items that fail to load are counted as pending and
// contribute no scenarios.
func (s *Service) aggregateInitiativeData(init *Initiative) (*RollupStatus, []string) {
	rollup := &RollupStatus{
		Total: len(init.Items),
	}
	seen := make(map[string]struct{})
	var scenarios []string
	for _, ref := range init.Items {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			rollup.Pending++
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			rollup.Pending++
			continue
		}
		item, loadErr := s.backlogLoader.LoadItem(kind, parts[1])
		if loadErr != nil {
			rollup.Pending++
			continue
		}
		if backlog.IsArchived(item) {
			rollup.Archived++
			if item.Status == backlog.StatusCompleted {
				rollup.Completed++
			}
			continue
		}
		switch item.Status {
		case backlog.StatusCompleted:
			rollup.Completed++
		case backlog.StatusFailed:
			rollup.Failed++
		case backlog.StatusInProgress, backlog.StatusQueued, backlog.StatusResearching:
			rollup.InProgress++
		default:
			rollup.Pending++
		}
		for _, name := range pathutil.ScenariosFromGlobs(item.AcceptanceAllow) {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				scenarios = append(scenarios, name)
			}
		}
	}
	return rollup, scenarios
}
