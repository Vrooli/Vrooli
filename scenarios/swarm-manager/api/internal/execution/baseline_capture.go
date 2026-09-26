package execution

import (
	"context"
	"log/slog"

	"swarm-manager/internal/pathutil"
)

// capturePreExecBaselinesLocked plans pre-execution baselines for the scenarios
// a backlog item declares it may touch (acceptance_allow), and kicks off their
// capture. It is called from startLocked while the service lock is held, so it
// only does cheap synchronous work:
//
//  1. Resolve declared scenarios (the sole pre-execution scope signal — the
//     sandbox diff does not exist yet).
//  2. Compute each scenario's working-tree-state hash and derive a
//     content-addressed baseline name. The hash is captured here, before the
//     agent spawns, so the name pins the true pre-execution state.
//  3. Launch a detached, best-effort goroutine to ensure each snapshot exists.
//     Snapshotting re-runs test-genie surfaces (minutes), so it must never run
//     under the lock or block execution start.
//
// Returns the scenario→baseline-name map to persist on the Record so
// finalization knows which baselines to diff. Returns nil when the feature is
// disabled, no scenarios are declared, or scope resolution fails — in every
// case execution proceeds unaffected.
func (s *Service) capturePreExecBaselinesLocked(ctx context.Context, item backlogItem) map[string]string {
	if !s.finalizationCfg.BaselineDiffEnabled || s.baselineClient == nil {
		return nil
	}

	scenarios := pathutil.UniqueSortedStrings(pathutil.ScenariosFromGlobs(item.AcceptanceAllow))
	if len(scenarios) == 0 {
		return nil
	}

	gitRoot, err := gitRepoRoot(ctx, s.repoRoot)
	if err != nil {
		slog.Warn("baseline: cannot resolve git root; skipping pre-exec capture", "err", err)
		return nil
	}

	type captureJob struct{ scenario, name string }
	var jobs []captureJob
	names := make(map[string]string)
	for _, scenario := range scenarios {
		// Never baseline self: finalization skips restarting swarm-manager and
		// a snapshot of our own in-flight tree would be meaningless.
		if s.selfScenarioName != "" && scenario == s.selfScenarioName {
			continue
		}
		hash, hashErr := workingTreeStateHash(ctx, gitRoot, scenario)
		if hashErr != nil {
			slog.Warn("baseline: working-tree state hash failed; skipping scenario",
				"scenario", scenario, "err", hashErr)
			continue
		}
		name := baselineNameFor(scenario, hash)
		names[scenario] = name
		jobs = append(jobs, captureJob{scenario: scenario, name: name})
	}
	if len(jobs) == 0 {
		return nil
	}

	// Detached capture: the request context is canceled when Start returns, so
	// use a background context bounded by the baseline client's own timeout.
	// A cache hit (state unchanged since a prior run) returns immediately.
	client := s.baselineClient
	// #nosec G118 -- intentional: per the comment above, this detached capture
	// must outlive the request context (canceled when Start returns); it is
	// bounded by the baseline client's own timeout.
	go func() {
		bgCtx := context.Background()
		for _, job := range jobs {
			cached, ensureErr := client.EnsureSnapshot(bgCtx, job.scenario, job.name)
			if ensureErr != nil {
				slog.Warn("baseline: pre-exec snapshot failed (finalization will mark scenario not_comparable)",
					"scenario", job.scenario, "name", job.name, "err", ensureErr)
				continue
			}
			slog.Info("baseline: pre-exec snapshot ready",
				"scenario", job.scenario, "name", job.name, "cached", cached)
		}
	}()

	return names
}
