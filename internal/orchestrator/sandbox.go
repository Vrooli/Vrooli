package orchestrator

import (
	"context"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// SandboxAffectedScenarios returns scenarios whose registry-known runtime
// instance has a WorkingDir rooted inside the given mergedPath. It is used by
// `vrooli scenario heal-from-sandbox` to relaunch only those scenarios whose
// processes are observably bound to a now-collapsed sandbox merge tree.
//
// The registry is authoritative: scenarios with no active instance are not
// considered affected, even if legacy process records exist on disk pointing
// into mergedPath.
func (s *Service) SandboxAffectedScenarios(ctx context.Context, mergedPath string) ([]string, error) {
	mergedPath = strings.TrimRight(mergedPath, "/")
	if mergedPath == "" {
		return nil, nil
	}
	store, err := s.openRuntimeRegistry(ctx)
	if err != nil {
		return nil, err
	}
	defer store.Close()

	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	affected := []string{}
	for _, instance := range instances {
		dir := strings.TrimRight(instance.WorkingDir, "/")
		if dir == "" {
			continue
		}
		if !strings.HasPrefix(dir, mergedPath) {
			continue
		}
		if _, ok := seen[instance.Scenario]; ok {
			continue
		}
		seen[instance.Scenario] = struct{}{}
		affected = append(affected, instance.Scenario)
	}
	slices.Sort(affected)
	return affected, nil
}
