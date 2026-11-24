package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
)

// uncertainApplyOutcome means the journal lacks a success receipt and the
// operation may have changed the target. Retrying it requires fresh evidence.
func uncertainApplyOutcome(outcome applyItemOutcome) bool {
	switch outcome {
	case applyItemApplying, applyItemIndeterminate, applyItemFailed, applyItemTimedOut:
		return true
	default:
		return false
	}
}

// applyReplayBlockers implements the replay rule against a fresh plan:
// unresolved effects may be retried only when they are no longer selected, or
// when the plan has positively observed them as satisfied/not applicable.
func applyReplayBlockers(priorRuns []applyRun, target string, planItems []applydomain.Item) []string {
	selected := make(map[string]string, len(planItems))
	for _, item := range planItems {
		selected[item.ID] = item.ObservedState
	}
	blockers := map[string]struct{}{}
	for _, prior := range priorRuns {
		if prior.Target != target {
			continue
		}
		for _, item := range prior.Items {
			if !uncertainApplyOutcome(item.Outcome) {
				continue
			}
			state, isSelected := selected[item.ID]
			if isSelected && state != applyStateSatisfied && state != "not_applicable" {
				blockers[item.ID] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(blockers))
	for id := range blockers {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func checkApplyReplayEligibility(target string, planItems []applydomain.Item) error {
	prior, err := readApplyRunJournals()
	if err != nil {
		return fmt.Errorf("retry blocked: %w", err)
	}
	for _, run := range prior {
		if run.Target == target && !run.Status.Terminal() {
			if applyRunnerAlive(run) {
				return fmt.Errorf("retry blocked: apply run %s still owns this target; reconnect to it before starting another run", run.ID)
			}
		}
	}
	blockers := applyReplayBlockers(prior, target, planItems)
	if len(blockers) > 0 {
		return fmt.Errorf("retry blocked: target state is unknown for prior in-flight effect(s) %s; recheck the target, then retry only after each is observed satisfied or removed from the selection", strings.Join(blockers, ", "))
	}
	return nil
}

func priorUncertainApplyItemIDs(target string) (map[string]struct{}, error) {
	runs, err := readApplyRunJournals()
	if err != nil {
		return nil, err
	}
	ids := map[string]struct{}{}
	for _, run := range runs {
		if run.Target != target {
			continue
		}
		for _, item := range run.Items {
			if uncertainApplyOutcome(item.Outcome) {
				ids[item.ID] = struct{}{}
			}
		}
	}
	return ids, nil
}

func readApplyRunJournals() ([]applyRun, error) {
	statePath, err := operatorStatePath()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(filepath.Dir(statePath), "apply-runs")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read apply recovery journals: %w", err)
	}
	prior := make([]applyRun, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !validApplyRunID(id) {
			return nil, fmt.Errorf("apply recovery journal %s has an invalid filename; reconcile it before retrying", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read apply recovery journal %s: %w", entry.Name(), err)
		}
		run, err := decodeApplyRunJournal(id, data)
		if err != nil {
			return nil, fmt.Errorf("apply recovery journal %s is unreadable; reconcile it before retrying", entry.Name())
		}
		prior = append(prior, run)
	}
	return prior, nil
}

