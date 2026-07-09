package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/planclient"
	"swarm-manager/internal/review"
	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
)

func (s *Server) registerExecutionRoutes(dataRoot, scenarioRoot string) *execution.Service {
	// Create archiver from scenarios handler for post-spec-sync archive
	var archiver execution.Archiver
	if s.scenariosHandler != nil {
		archiver = scenarios.NewArchiver(s.scenariosHandler)
	}

	storePath, err := runtimepaths.StatePath("execution-runs.json")
	if err != nil {
		panic(err)
	}

	// Baseline Modes engagement (plan P6 §200): opt-in via
	// SWARM_MANAGER_BASELINE_ENGAGEMENT so execution fronts each declared scenario
	// with a live-mode `baseline start` (git-free restore point) and
	// promotes-on-green / abandons-on-terminal-failure at finalization. Default
	// off — the mechanism is dormant until an operator flips the env, which keeps
	// the reflexive kernel unperturbed during rollout.
	finalizationCfg := execution.DefaultFinalizationConfig()
	finalizationCfg.BaselineEngagementEnabled = baselineEngagementEnabled()

	// Execution control endpoints
	cfg := execution.ServiceConfig{
		DataRoot:                 dataRoot,
		RepoRoot:                 scenarioRoot,
		StorePath:                storePath,
		SelfScenarioName:         "swarm-manager",
		PolicyProvider:           settings.NewPolicyAdapter(s.settingsStore),
		GovernanceProvider:       settings.NewGovernanceAdapter(s.settingsStore),
		ReviewThresholdsProvider: settings.NewReviewThresholdsAdapter(s.settingsStore),
		AgentService:             s.requireTrackedAgentService(),
		ScenarioLifecycle:        scenarios.NewCLILifecycle(),
		ScenarioHealthChecker:    scenarios.NewCLIHealthChecker(20 * time.Second),
		Archiver:                 archiver,
		ReviewClient:             execution.NewHTTPReviewClient(nil),
		BaselineClient:           execution.NewConnectBaselineClient(nil),
		BaselineEngagementRunner: &execution.GCTBaselineEngagementRunner{ProjectRoot: repoRootFromScenarioRoot(scenarioRoot)},
		PlanRenderer:             planclient.NewConnectClient(nil, nil),
		Finalization:             finalizationCfg,
	}
	s.executionSvc = execution.NewService(cfg)
	// Wire the agentactivity service in as the lane reader so
	// GovernanceStatus reports per-lane utilization for all four canonical
	// lanes (Execute is also visible via execution.Records, but the other
	// three only live in the activity store).
	if s.agentActivitySvc != nil {
		s.executionSvc.SetActivityLaneReader(s.agentActivitySvc)
	}
	// Wire the fix-before-feature discovery filer (Tier 2). It files fix items
	// via the canonical backlog creation path. backlogHandler is registered
	// before this route group, so its store is available here.
	if s.backlogHandler != nil {
		filerCfg := backlog.ServiceConfig{
			Store:    s.backlogHandler.Store(),
			Assigner: s.initiativeService,
		}
		if s.emitter != nil {
			filerCfg.Events = s.emitter
		}
		if filerSvc, filerErr := backlog.NewService(filerCfg); filerErr == nil {
			s.executionSvc.SetRemediationFiler(backlog.NewFixDiscoveryFiler(filerSvc))
		}
	}
	s.executionHandler = execution.NewHandlerFromService(s.executionSvc)
	s.executionHandler.RegisterRoutes(s.router)

	// Wire execution queuer back into scenarios handler for spec-sync-archive
	if s.scenariosHandler != nil {
		s.scenariosHandler.SetExecutionQueuer(scenarios.NewExecutionQueuer(s.executionSvc))
	}
	if s.backlogHandler != nil {
		s.backlogHandler.SetExecutionQueuer(s.executionSvc)

		// Baseline Modes engagement close (plan P-c): promote/abandon the owner's
		// whole engagement set at the review-decide terminal transition — the
		// atomic accept/reject — not at finalization. Chained via Add so it
		// coexists with the initiative-review trigger.
		execSvc := s.executionSvc
		s.backlogHandler.AddItemTerminalHandler(func(ctx context.Context, kind, name string, status backlog.BacklogStatus) {
			execSvc.CloseOwnerEngagements(ctx, kind, name, engagementCloseDecisionForStatus(status))
		})
	}
	return s.executionSvc
}

// engagementCloseDecisionForStatus maps a backlog terminal status onto the
// execution package's engagement-close decision. Accept (completed) promotes the
// set; reject (failed) abandons it; needs_followup leaves it open for the next
// run under the same owner.
func engagementCloseDecisionForStatus(status backlog.BacklogStatus) execution.EngagementCloseDecision {
	switch status {
	case backlog.StatusCompleted:
		return execution.EngagementPromote
	case backlog.StatusFailed:
		return execution.EngagementAbandon
	default:
		return execution.EngagementLeaveOpen
	}
}

// baselineEngagementEnabled reports whether Baseline Modes engagements are
// turned on for backlog execution (plan P6 §200). Opt-in via
// SWARM_MANAGER_BASELINE_ENGAGEMENT (1/true/yes/on, case-insensitive); default
// off so the reflexive kernel runs unchanged until an operator enables it.
func baselineEngagementEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SWARM_MANAGER_BASELINE_ENGAGEMENT"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// repoRootFromScenarioRoot derives the repo root from the scenario source path
// (`<repo>/scenarios/swarm-manager` → `<repo>`), used as the working dir for the
// git-control-tower engagement runner. Returns "" when the path does not look
// like a scenario root (the runner then inherits the process working dir; GCT
// baseline verbs resolve scenarios via the vrooli registry regardless).
func repoRootFromScenarioRoot(scenarioRoot string) string {
	root := strings.TrimSpace(scenarioRoot)
	if root == "" {
		return ""
	}
	parent := filepath.Dir(root)
	if filepath.Base(parent) == "scenarios" {
		return filepath.Dir(parent)
	}
	return ""
}

func (s *Server) registerReviewRoutes(scenarioRoot string, execSvc *execution.Service) {
	backlogStore := backlog.NewFileStore(s.dataRoot)
	reviewPlanClient := planclient.NewConnectClient(nil, nil)
	cfg := review.ServiceConfig{
		DataRoot:     s.dataRoot,
		AgentService: s.requireTrackedAgentService(),
		ItemDirFn:    func(kind, name string) string { return backlogStore.ItemDir(backlog.BacklogKind(kind), name) },
		LoadItemTitle: func(kind, name string) (string, error) {
			item, err := backlogStore.LoadItem(backlog.BacklogKind(kind), name)
			if err != nil {
				return "", err
			}
			return item.Title, nil
		},
		PlanContentResolver: func(ctx context.Context, kind, name, _ string) (string, error) {
			if strings.EqualFold(strings.TrimSpace(kind), string(backlog.KindResearch)) {
				return "", nil
			}
			item, err := backlogStore.LoadItem(backlog.BacklogKind(kind), name)
			if err != nil {
				return "", err
			}
			if item.PlanRef == nil {
				return "", fmt.Errorf("backlog item %s/%s has no plan_ref", kind, name)
			}
			planID := strings.TrimSpace(item.PlanRef.PlanID)
			if planID == "" {
				planID = strings.TrimSpace(item.PlanRef.Slug)
			}
			if planID == "" {
				return "", fmt.Errorf("backlog item %s/%s plan_ref has no plan id or slug", kind, name)
			}
			rendered, err := reviewPlanClient.RenderMarkdown(ctx, planID, true)
			if err != nil {
				return "", err
			}
			return rendered.Markdown, nil
		},
		// When a review round finishes (complete or failed), flip the backlog
		// item from in_review to review_pending so the user can assess and
		// decide the terminal status via the review-decide endpoint.
		OnRoundTerminal: func(ctx context.Context, kind, name string, round review.Round) {
			item, err := backlogStore.LoadItem(backlog.BacklogKind(kind), name)
			if err != nil {
				return
			}
			// Only transition from in_review; any other state means the user
			// (or another flow) has already taken over — don't overwrite.
			if item.Status != backlog.StatusInReview {
				return
			}
			item.Status = backlog.StatusReviewPending
			_ = backlogStore.SaveItem(item)
		},
	}
	if execSvc != nil {
		cfg.LoadExecutionContext = func(ctx context.Context, executionID string) (*review.ExecutionContext, error) {
			record, err := execSvc.Get(ctx, executionID)
			if err != nil {
				return nil, err
			}

			ctxOut := &review.ExecutionContext{
				BacklogKind:            record.BacklogKind,
				BacklogName:            record.BacklogName,
				ItemTitle:              record.BacklogName,
				AffectedScenarios:      []string{},
				ChangedPathsByScenario: map[string][]string{},
			}
			if item, err := backlogStore.LoadItem(backlog.BacklogKind(record.BacklogKind), record.BacklogName); err == nil {
				ctxOut.ItemTitle = item.Title
			}

			if record.Finalization == nil {
				return ctxOut, nil
			}

			ctxOut.AffectedScenarios = append([]string(nil), record.Finalization.AffectedScenarios...)
			resultsByScenario := make(map[string]any)
			baselineByScenario := make(map[string]execution.BaselineDiffResult)
			for _, scenario := range record.Finalization.Scenarios {
				if len(scenario.ChangedPaths) > 0 {
					ctxOut.ChangedPathsByScenario[scenario.ScenarioName] = append([]string(nil), scenario.ChangedPaths...)
				}
				if scenario.BaselineDiff != nil {
					baselineByScenario[scenario.ScenarioName] = *scenario.BaselineDiff
				}
				if scenario.Review.Result == nil {
					continue
				}
				resultsByScenario[scenario.ScenarioName] = map[string]any{
					"classification": scenario.Review.Result.Classification,
					"dimensions":     scenario.Review.Result.Dimensions,
					"raw_dimensions": scenario.Review.Result.RawDimensions,
					"summary":        scenario.Review.Result.Summary,
				}
			}
			ctxOut.GCTResultsJSON = review.MarshalScenarioGCTResults(resultsByScenario)
			ctxOut.BaselineDiffJSON = execution.MarshalBaselineDiffResults(baselineByScenario)

			return ctxOut, nil
		}
	}
	s.reviewSvc = review.NewService(cfg)
	s.reviewHandler = review.NewHandler(s.reviewSvc)
	s.reviewHandler.RegisterRoutes(s.router)

	// Wire review service into execution service for finalization integration.
	if execSvc != nil {
		execSvc.SetReviewService(s.reviewSvc)
	}

	// Wire the review-round liveness seam so the recover-review endpoint can
	// refuse to short-circuit an in-flight review.
	if s.backlogHandler != nil {
		s.backlogHandler.SetReviewRoundInspector(s.reviewSvc)

		// Orphaned-in_review safety net: a boot-time sweep recovers any items
		// stranded in in_review with no live review round (work done
		// out-of-band, a review run that died, or a premature mark), and the
		// ticker keeps the invariant for the process lifetime. Mirrors the
		// feedback sweeper. Recovery routes items to review_pending so a human
		// still decides the terminal state.
		sweeper := review.NewSweeper(s.reviewSvc, backlogStore, s.backlogHandler.RecoverOrphanedReview)
		if recovered, err := sweeper.RunOnce(context.Background()); err != nil {
			slog.Warn("review: boot-time orphaned-in_review sweep failed", "err", err)
		} else if recovered > 0 {
			slog.Info("review: boot-time orphaned-in_review sweep recovered items", "count", recovered)
		}
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				<-s.reviewSweeperStop
				cancel()
			}()
			sweeper.Start(ctx)
		}()
	}
}
