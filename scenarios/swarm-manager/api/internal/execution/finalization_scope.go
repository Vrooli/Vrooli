package execution

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/pathutil"
)

func (s *Service) resolveFinalizationScope(ctx context.Context, record Record, item backlogItem) (finalizationScope, error) {
	scope := finalizationScope{
		source:                 FinalizationScopeNone,
		changedPathsByScenario: map[string][]string{},
	}

	acceptanceScenarios := pathutil.UniqueSortedStrings(pathutil.ScenariosFromGlobs(item.AcceptanceAllow))
	var sandboxDiff agentmanager.RunDiff
	if s.differ != nil && strings.TrimSpace(record.RunID) != "" {
		diff, err := s.differ.GetRunDiff(ctx, record.RunID)
		if err == nil {
			sandboxDiff = diff
			scope.sandboxID = strings.TrimSpace(diff.SandboxID)
		} else {
			scope.warnings = append(scope.warnings, newFinalizationWarning(
				finalizationWarningScopeDiffUnavailable,
				"",
				fmt.Sprintf("sandbox diff unavailable for %s: %v", record.RunID, err),
				false,
			))
		}
	}

	if len(sandboxDiff.Files) > 0 {
		paths := make([]string, 0, len(sandboxDiff.Files))
		for _, file := range sandboxDiff.Files {
			paths = append(paths, file.Path)
		}
		grouped := pathutil.GroupChangedPaths(paths)
		for scenarioName, changedPaths := range grouped.DirectScenarioPaths {
			scope.changedPathsByScenario[scenarioName] = append([]string(nil), changedPaths...)
		}

		directScenarios := mapKeysSorted(grouped.DirectScenarioPaths)
		switch {
		case len(grouped.SharedPaths) == 0 && len(directScenarios) > 0:
			scope.source = FinalizationScopeSandboxDiff
			scope.affectedScenarios = directScenarios
			return scope, nil
		case len(grouped.SharedPaths) > 0 && len(directScenarios) > 0:
			scope.source = FinalizationScopeSandboxDiffPlusAcceptance
			scope.affectedScenarios = unionSortedStrings(directScenarios, acceptanceScenarios)
			for _, scenarioName := range scope.affectedScenarios {
				if len(scope.changedPathsByScenario[scenarioName]) == 0 {
					scope.changedPathsByScenario[scenarioName] = append([]string(nil), grouped.SharedPaths...)
				}
			}
			scope.warnings = append(scope.warnings, newFinalizationWarning(
				finalizationWarningScopeSharedPathBroadening,
				"",
				fmt.Sprintf("shared repo changes required acceptance_allow broadening: %s", strings.Join(grouped.SharedPaths, ", ")),
				false,
			))
			return scope, nil
		case len(grouped.SharedPaths) > 0:
			scope.source = FinalizationScopeAcceptanceAllow
			scope.affectedScenarios = acceptanceScenarios
			for _, scenarioName := range scope.affectedScenarios {
				scope.changedPathsByScenario[scenarioName] = append([]string(nil), grouped.SharedPaths...)
			}
			scope.warnings = append(scope.warnings, newFinalizationWarning(
				finalizationWarningScopeSharedPathBroadening,
				"",
				fmt.Sprintf("sandbox diff only exposed shared paths; falling back to acceptance_allow: %s", strings.Join(grouped.SharedPaths, ", ")),
				false,
			))
			return scope, nil
		}
	}

	scope.source = FinalizationScopeAcceptanceAllow
	scope.affectedScenarios = acceptanceScenarios
	return scope, nil
}
