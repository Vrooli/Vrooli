package execution

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/pathutil"
)

// Pre-merge engagement hold (plan P-b.3 / P-b.4).
//
// With SWARM_MANAGER_BASELINE_ENGAGEMENT on, a backlog run is spawned with
// agent-manager's ManualReview hold, so it parks at needs_review with its edits
// still in the sandbox overlay (NOT merged). This file drives what swarm-manager
// does at that hold:
//
//  1. read the ACTUAL diff (GetRunDiff) → the scenarios the run really touched;
//  2. for each not already engaged under this owner, open a shadow restore point
//     (`baseline start --mode shadow`) — capturing the clean working tree BEFORE
//     the merge lands on it;
//  3. ApproveRun → merge the overlay into the working tree.
//
// Only after the restore points exist is the merge approved, so the isolation
// floor (plan P-a) holds: live serves the captured baseline from the copy while
// @shadow runs the just-merged candidate from the working tree. The work runs
// OUTSIDE the service mutex (baseline start is slow) under a per-execution
// in-flight guard, and is idempotent via Record.EngagementHoldAt.

// engagementHoldActive reports whether the pre-merge hold machinery should run:
// the feature flag is on and an approver seam is wired. When false, runs are
// never spawned with ManualReview, so they never reach needs_review and this
// path is dormant.
func (s *Service) engagementHoldActive() bool {
	return s.finalizationCfg.BaselineEngagementEnabled &&
		s.baselineEngagementRunner != nil &&
		s.approver != nil &&
		s.differ != nil &&
		s.engagementStore != nil
}

func (s *Service) beginHold(executionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processingHolds == nil {
		s.processingHolds = map[string]struct{}{}
	}
	if _, exists := s.processingHolds[executionID]; exists {
		return false
	}
	s.processingHolds[executionID] = struct{}{}
	return true
}

func (s *Service) endHold(executionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.processingHolds, executionID)
}

// processEngagementHold opens shadow engagements from the actual diff and
// approves the merge for a run parked at needs_review. Idempotent: a record
// whose EngagementHoldAt is already set is skipped. Errors leave the run held
// (no merge without isolation) for a later poll/operator retry.
func (s *Service) processEngagementHold(ctx context.Context, executionID string) error {
	if !s.engagementHoldActive() {
		return nil
	}
	if !s.beginHold(executionID) {
		return nil
	}
	defer s.endHold(executionID)

	s.mu.Lock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	record := records[idx]
	s.mu.Unlock()

	if record.Status != StatusNeedsReview || strings.TrimSpace(record.EngagementHoldAt) != "" {
		return nil
	}
	if strings.TrimSpace(record.RunID) == "" {
		return nil
	}

	owner := ownerKeyForRecord(record)

	// 1. Actual diff → directly-touched scenarios.
	diff, err := s.differ.GetRunDiff(ctx, record.RunID)
	if err != nil {
		return fmt.Errorf("engagement hold: get run diff for %s: %w", executionID, err)
	}
	touched := scenariosFromRunDiff(diff)

	// 2. Decide which to open: skip self, skip scenarios already engaged under
	// this owner (a fixup re-touching the same scenario), and enforce exclusivity
	// against OTHER owners (diff-level defense behind the start-time gate).
	toOpen, err := s.engagementsToOpen(touched, owner, executionID)
	if err != nil {
		return err
	}

	// 3. Open shadow restore points (capture the clean working tree) BEFORE merge.
	if len(toOpen) > 0 {
		opened, openErr := s.openEngagementsForOwner(ctx, toOpen)
		if openErr != nil {
			return fmt.Errorf("engagement hold: open engagements for %s: %w", executionID, openErr)
		}
		if len(opened) > 0 {
			if _, addErr := s.engagementStore.Add(owner, opened, nowRFC3339()); addErr != nil {
				return fmt.Errorf("engagement hold: persist engagement set for %s: %w", owner, addErr)
			}
		}
	}

	// 4. Approve the merge: overlay → working tree.
	if approveErr := s.approver.ApproveRun(ctx, record.RunID, "swarm-manager", "baseline-shadow: merge candidate after restore-point capture"); approveErr != nil {
		return fmt.Errorf("engagement hold: approve merge for %s: %w", executionID, approveErr)
	}

	// 5. Mark the hold done (idempotency) under the lock.
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err = s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	records[idx].EngagementHoldAt = nowRFC3339()
	records[idx].UpdatedAt = records[idx].EngagementHoldAt
	if saveErr := s.store.Save(records); saveErr != nil {
		return saveErr
	}
	slog.Info("baseline engagement: pre-merge hold processed",
		"execution_id", executionID, "owner", owner, "engaged", toOpen)
	return nil
}

// engagementsToOpen filters touched scenarios down to those needing a fresh
// shadow engagement under owner: it skips self and scenarios already held by
// this owner, and fails closed when a scenario is engaged under a different owner.
func (s *Service) engagementsToOpen(touched []string, owner, executionID string) ([]string, error) {
	toOpen := make([]string, 0, len(touched))
	for _, scenario := range touched {
		if s.selfScenarioName != "" && scenario == s.selfScenarioName {
			continue
		}
		holder, _, held, holderErr := s.engagementStore.HolderOf(scenario)
		if holderErr != nil {
			return nil, fmt.Errorf("engagement hold: exclusivity lookup for %q: %w", scenario, holderErr)
		}
		if held {
			if holder == owner {
				continue // already ours — fixup inheritance, nothing to open
			}
			return nil, fmt.Errorf("engagement hold: scenario %q is engaged under a different owner %q; "+
				"cannot merge run %s safely (run left held for operator)", scenario, holder, executionID)
		}
		toOpen = append(toOpen, scenario)
	}
	return toOpen, nil
}

// checkExclusivityAtStart enforces the block-at-start exclusivity policy: an
// owner may not start if its PROJECTED scope (acceptance_allow scenarios)
// intersects a scenario with an open engagement under a different owner. Returns
// an actionable BadRequest error on conflict; never queues (the locked policy is
// ExclusivityBlockAtStart). A no-op when the hold machinery is inactive.
func (s *Service) checkExclusivityAtStart(item backlogItem, owner string) error {
	if !s.engagementHoldActive() || engagementExclusivityPolicy != ExclusivityBlockAtStart {
		return nil
	}
	projected := pathutil.UniqueSortedStrings(pathutil.ScenariosFromGlobs(item.AcceptanceAllow))
	for _, scenario := range projected {
		if s.selfScenarioName != "" && scenario == s.selfScenarioName {
			continue
		}
		holder, _, held, err := s.engagementStore.HolderOf(scenario)
		if err != nil {
			return apierr.Internal("exclusivity check failed for %q: %v", scenario, err)
		}
		if held && holder != owner {
			return apierr.Wrapf(errEngagementConflict, http.StatusConflict,
				"scenario %q already has an open baseline engagement under %q; "+
					"finish or abandon that work before starting %q", scenario, holder, owner)
		}
	}
	return nil
}

// shadowTargetFor returns the lifecycle target for a scenario during
// finalization (plan P-b.5). When the scenario is shadow-engaged under this
// run's owner, validation must restart/health-check the @shadow instance — the
// one running the just-merged candidate — not live (which still serves the
// captured baseline). Returns "<scenario>@shadow" in that case, the bare name
// otherwise (no engagement, or a live-mode downgrade where live IS the target).
// CLILifecycle/CLIHealthChecker forward the @variant suffix to the Vrooli CLI.
func (s *Service) shadowTargetFor(record Record, scenarioName string) string {
	if s.engagementStore == nil {
		return scenarioName
	}
	set, ok, err := s.engagementStore.Get(ownerKeyForRecord(record))
	if err != nil || !ok {
		return scenarioName
	}
	if set.Engagements[scenarioName] == engagementMode {
		return scenarioName + "@" + engagementMode
	}
	return scenarioName
}

// scenariosFromRunDiff extracts the directly-touched scenario names from a run
// diff. Shared/platform paths are deliberately excluded — they can't be isolated
// per-scenario by a shadow restore point, so engaging them would be meaningless.
func scenariosFromRunDiff(diff agentmanager.RunDiff) []string {
	if len(diff.Files) == 0 {
		return nil
	}
	paths := make([]string, 0, len(diff.Files))
	for _, file := range diff.Files {
		paths = append(paths, file.Path)
	}
	grouped := pathutil.GroupChangedPaths(paths)
	scenarios := make([]string, 0, len(grouped.DirectScenarioPaths))
	for scenario := range grouped.DirectScenarioPaths {
		scenarios = append(scenarios, scenario)
	}
	return pathutil.UniqueSortedStrings(scenarios)
}
