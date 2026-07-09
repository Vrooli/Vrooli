package execution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/workshop"
)

// ProcessPreflight evaluates whether a backlog item is ready for processing.
func (s *Service) ProcessPreflight(_ context.Context, backlogKind, backlogName string) (ProcessPreflight, error) {
	item, err := s.loadBacklogItem(backlogKind, backlogName)
	if err != nil {
		return ProcessPreflight{}, err
	}
	return s.processPreflightForItem(item, true), nil
}

func (s *Service) processPreflightForItem(item backlogItem, checkQueueable bool) ProcessPreflight {
	targetScenarioID, archivedRevival := resolveTargetScenario(item)
	targetScenarioExists := false
	if strings.TrimSpace(targetScenarioID) != "" {
		targetScenarioExists = scenarioExists(filepath.Join(s.scenariosRootDir(), targetScenarioID))
	}

	preflight := ProcessPreflight{
		BacklogKind:              strings.TrimSpace(item.Kind),
		BacklogName:              strings.TrimSpace(item.Name),
		Ready:                    true,
		ArchivedRevival:          archivedRevival,
		ResolvedTargetScenarioID: targetScenarioID,
		TargetScenarioExists:     targetScenarioExists,
		SuggestedOperation:       "generator",
		SuggestedSteerProfileID:  "rapid-mvp",
	}
	if targetScenarioExists {
		preflight.SuggestedOperation = "improver"
		preflight.SuggestedSteerProfileID = "production-ready"
	}

	isArchived := item.ArchivedAt != nil
	if checkQueueable && !isQueueableStatus(item.Kind, item.Status) && !(isArchived && strings.ToLower(strings.TrimSpace(item.Kind)) == "idea") {
		preflight.BlockingReasons = append(preflight.BlockingReasons, fmt.Sprintf("backlog item cannot be queued from current status: %s", item.Status))
	}

	// Check workshop readiness instead of clarify questions.
	itemDir := s.itemDir(item.Kind, item.Name)
	deliverablePath := deliverableForKind(item.Kind)
	hasDeliverable := false
	if deliverablePath == "conclusion.md" {
		hasDeliverable = workshop.HasPlanByName(itemDir, deliverablePath)
	} else {
		hasDeliverable = hasExecutionPlanRef(item)
	}
	if !hasDeliverable {
		preflight.BlockingReasons = append(preflight.BlockingReasons, missingDeliverableReason(item.Kind, deliverablePath))
	}
	rounds, _ := workshop.LoadRounds(itemDir)
	if len(rounds) > 0 {
		latest := rounds[len(rounds)-1]
		rawScores := make(map[string]int, len(workshop.ReadinessDimensions))
		for _, dim := range workshop.ReadinessDimensions {
			if v, ok := latest.Readiness[dim]; ok {
				rawScores[dim] = v
			}
		}
		effective := workshop.ComputeEffectiveScores(rawScores, len(rounds), item.Kind)
		for _, dim := range workshop.ReadinessDimensions {
			if effective[dim] < 3 {
				preflight.BlockingReasons = append(preflight.BlockingReasons, fmt.Sprintf("readiness dimension %q is %d/3 — needs more workshop refinement", dim, effective[dim]))
			}
		}
	} else if hasDeliverable {
		// Primary deliverable exists but no workshop rounds — allow execution.
	} else {
		preflight.BlockingReasons = append(preflight.BlockingReasons, "no workshop rounds completed — run workshop or initialize first")
	}

	// Fix-before-feature gate applies only at queue time (checkQueueable),
	// never on start/retry/followup of an already-running item.
	if checkQueueable {
		s.applyFixBeforeFeatureGate(item, &preflight)
	}

	preflight.Ready = len(preflight.BlockingReasons) == 0 && len(preflight.ForceableBlockingReasons) == 0
	return preflight
}

func hasExecutionPlanRef(item backlogItem) bool {
	if item.PlanRef == nil {
		return false
	}
	return strings.TrimSpace(item.PlanRef.Provider) == planRefProviderPlanManager &&
		strings.TrimSpace(item.PlanRef.Role) == planRefRoleExecutionSpec &&
		(strings.TrimSpace(item.PlanRef.PlanID) != "" || strings.TrimSpace(item.PlanRef.Slug) != "")
}

// Old clarify-based blocking question types and loading have been removed.
// The execution preflight now uses workshop readiness from backlog.LoadWorkshopRounds
// and backlog.ComputeEffectiveScores instead.

func resolveTargetScenario(item backlogItem) (string, bool) {
	source := strings.TrimSpace(item.SourceScenarioName)
	if source != "" {
		return source, true
	}
	return strings.TrimSpace(item.Name), item.ArchivedAt != nil
}

// allBlockingReasons returns every reason that makes a preflight not-ready —
// both structural (non-forceable) and forceable — for display in error
// messages. The Ready flag already accounts for both slices.
func allBlockingReasons(preflight ProcessPreflight) []string {
	if len(preflight.ForceableBlockingReasons) == 0 {
		return preflight.BlockingReasons
	}
	combined := make([]string, 0, len(preflight.BlockingReasons)+len(preflight.ForceableBlockingReasons))
	combined = append(combined, preflight.BlockingReasons...)
	combined = append(combined, preflight.ForceableBlockingReasons...)
	return combined
}

func hasNonForceableExecutionReasons(reasons []string) bool {
	for _, reason := range reasons {
		normalized := strings.ToLower(strings.TrimSpace(reason))
		if normalized == "" {
			continue
		}
		if strings.Contains(normalized, "workshop decision") || strings.Contains(normalized, "pending decision") {
			continue
		}
		return true
	}
	return false
}

func scenarioExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
