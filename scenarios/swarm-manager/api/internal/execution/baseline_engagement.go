package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Baseline Modes engagement — shadow-mode rework (plan P-b).
//
// When SWARM_MANAGER_BASELINE_ENGAGEMENT is set, backlog execution spawns the
// agent with agent-manager's ManualReview hold (overlay NOT merged) and, at the
// resulting needs_review hold, opens a `git-control-tower baseline start --mode
// shadow` engagement for each scenario the run *actually* touched (the real
// GetRunDiff, not the declared acceptance_allow). Only then is the merge
// approved. The ordering is the whole point of the isolation floor (plan P-a):
//
//	working tree (clean baseline)  ──baseline start──▶  restore-point copy
//	overlay (candidate)            ──ApproveRun──────▶  merged into working tree
//
// so live keeps serving the captured baseline from the copy while @shadow runs
// the candidate from the working tree. Engagements are owned by the backlog item
// (EngagementStore, ownerKeyFor) and held across the main run, every fixup, and
// the gap until review-decide promotes the whole set (accept) or abandons it
// (reject) — see engagement_owner.go and engagement_hold.go.
//
// This file owns the git-control-tower runner contract (Start/Promote/Abandon)
// and the owner-set open/promote/abandon helpers; the pre-merge orchestration
// lives in engagement_hold.go.

// BaselineEngagement is the runner's result for a started engagement.
type BaselineEngagement struct {
	// Scenario is the engaged target.
	Scenario string
	// Mode is the decision-tree outcome: "shadow" or "live".
	Mode string
	// AmbientVar is the scenario slug to route to its shadow (set only in shadow
	// mode); empty in live mode.
	AmbientVar string
	// Reflexive reports whether the target is in the reflexive/core set (advisory).
	Reflexive bool
}

// BaselinePromoteParams are the inputs to a promote, threading the self-deadlock
// guards into the GCT promote's quiesce drain.
type BaselinePromoteParams struct {
	Scenario string
	// ExcludeRun is the run excluded from the promote drain set — the run that
	// edited the scenario. For swarm-manager it has already completed by
	// finalization time, so it is a no-op for the drain, but it is threaded for
	// parity with the EM half and to harden against a future overlapping run.
	ExcludeRun string
	// TagPrefix scopes the drain to a set of spawned runs (empty ⇒ drain every
	// in-flight run targeting the scenario, the safe default before a live
	// re-point/restart).
	TagPrefix string
}

// BaselineEngagementRunner is the execution package's boundary to
// `git-control-tower baseline`. Production wires GCTBaselineEngagementRunner
// (shells the CLI); tests wire a fake. Start opens the engagement (returning the
// decided mode); Promote/Abandon close it.
type BaselineEngagementRunner interface {
	Start(ctx context.Context, scenario, mode string) (BaselineEngagement, error)
	Promote(ctx context.Context, p BaselinePromoteParams) error
	Abandon(ctx context.Context, scenario string) error
}

// GCTBaselineEngagementRunner shells the git-control-tower CLI's `baseline`
// verbs. It mirrors ecosystem-manager's GCTBaselineRunner: a project-root working
// dir, a PATH-resolved binary (overridable via $SWARM_MANAGER_GCT_BIN), and a
// per-call timeout.
type GCTBaselineEngagementRunner struct {
	// ProjectRoot is the repo root; the command runs with this as its working
	// dir. Empty ⇒ inherit the process working dir (GCT baseline verbs resolve
	// scenarios via the vrooli registry/home, so they are largely wd-independent).
	ProjectRoot string
	// Binary is the git-control-tower executable; defaults to "git-control-tower"
	// (or $SWARM_MANAGER_GCT_BIN) resolved on PATH.
	Binary string
	// StartTimeout bounds `baseline start`. Zero ⇒ defaultEngagementStartTimeout.
	StartTimeout time.Duration
	// PromoteTimeout bounds `baseline promote`/`abandon`. Zero ⇒
	// defaultEngagementPromoteTimeout.
	PromoteTimeout time.Duration
}

const (
	defaultEngagementStartTimeout   = 10 * time.Minute
	defaultEngagementPromoteTimeout = 15 * time.Minute
)

var _ BaselineEngagementRunner = (*GCTBaselineEngagementRunner)(nil)

func (r *GCTBaselineEngagementRunner) binary() string {
	if strings.TrimSpace(r.Binary) != "" {
		return r.Binary
	}
	if env := strings.TrimSpace(os.Getenv("SWARM_MANAGER_GCT_BIN")); env != "" {
		return env
	}
	return "git-control-tower"
}

func (r *GCTBaselineEngagementRunner) run(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.binary(), args...)
	if strings.TrimSpace(r.ProjectRoot) != "" {
		cmd.Dir = r.ProjectRoot
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return stdout.String(), fmt.Errorf("git-control-tower %s timed out after %s", args[0], timeout)
	}
	if err != nil {
		return stdout.String(), fmt.Errorf("git-control-tower %s: %w (stderr: %s)",
			strings.Join(args, " "), err, truncateEngagement(stderr.String(), 500))
	}
	return stdout.String(), nil
}

// engagementStartJSON is the subset of `baseline start --json` (GCT startResult)
// the runner reads to learn the decided mode + ambient routing var.
type engagementStartJSON struct {
	Scenario   string `json:"scenario"`
	Variant    string `json:"variant"`
	AmbientVar string `json:"ambientVar"`
	Decision   struct {
		Mode      string `json:"mode"`
		Reflexive bool   `json:"reflexive"`
	} `json:"decision"`
}

func (r *GCTBaselineEngagementRunner) Start(ctx context.Context, scenario, mode string) (BaselineEngagement, error) {
	timeout := r.StartTimeout
	if timeout <= 0 {
		timeout = defaultEngagementStartTimeout
	}
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "auto"
	}
	// --no-anchor: swarm-manager already pins its own pre-exec baselines for the
	// before/after diff (capturePreExecBaselinesLocked); the GCT anchor would
	// redundantly re-run test-genie surfaces and slow the engagement open.
	out, err := r.run(ctx, timeout, "baseline", "start", "--scenario", scenario, "--mode", mode, "--no-anchor", "--json")
	if err != nil {
		return BaselineEngagement{}, err
	}
	return parseEngagementStartJSON(scenario, out)
}

// parseEngagementStartJSON decodes `baseline start --json` (GCT startResult) into
// the runner's engagement view. Split out so the contract is unit-testable
// without a git-control-tower binary.
func parseEngagementStartJSON(scenario, out string) (BaselineEngagement, error) {
	var res engagementStartJSON
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
		return BaselineEngagement{}, fmt.Errorf("parse baseline start output for %q: %w", scenario, err)
	}
	mode := res.Variant
	if mode == "" {
		mode = res.Decision.Mode
	}
	return BaselineEngagement{
		Scenario:   scenario,
		Mode:       mode,
		AmbientVar: res.AmbientVar,
		Reflexive:  res.Decision.Reflexive,
	}, nil
}

func (r *GCTBaselineEngagementRunner) Promote(ctx context.Context, p BaselinePromoteParams) error {
	timeout := r.PromoteTimeout
	if timeout <= 0 {
		timeout = defaultEngagementPromoteTimeout
	}
	args := []string{"baseline", "promote", "--scenario", p.Scenario, "--json"}
	if strings.TrimSpace(p.ExcludeRun) != "" {
		args = append(args, "--exclude-run", p.ExcludeRun)
	}
	if strings.TrimSpace(p.TagPrefix) != "" {
		args = append(args, "--tag-prefix", p.TagPrefix)
	}
	_, err := r.run(ctx, timeout, args...)
	return err
}

func (r *GCTBaselineEngagementRunner) Abandon(ctx context.Context, scenario string) error {
	timeout := r.PromoteTimeout
	if timeout <= 0 {
		timeout = defaultEngagementPromoteTimeout
	}
	_, err := r.run(ctx, timeout, "baseline", "abandon", "--scenario", scenario, "--json")
	return err
}

func truncateEngagement(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// openEngagementsForOwner opens a Baseline Modes engagement for each scenario
// under the given owner, returning the scenario→decided-mode map. It requests
// shadow mode; git-control-tower's decision tree may downgrade a non-shadow-
// eligible scenario to live and report that back (the loud fallback, plan §11),
// in which case the returned mode is "live". Self is skipped (restarting or
// rolling back our own process is the deadlock the plan externalizes).
//
// Unlike the old best-effort live-mode open, a start ERROR is returned, not
// swallowed: at the pre-merge hold the candidate still sits in the agent's
// overlay, so failing to capture a restore point means we must NOT approve the
// merge (an unisolated live could rebuild the candidate on its next restart).
// The caller leaves the run held for a later retry.
func (s *Service) openEngagementsForOwner(ctx context.Context, scenarios []string) (map[string]string, error) {
	engagements := make(map[string]string)
	for _, scenario := range scenarios {
		scenario = strings.TrimSpace(scenario)
		if scenario == "" {
			continue
		}
		if s.selfScenarioName != "" && scenario == s.selfScenarioName {
			continue
		}
		eng, err := s.baselineEngagementRunner.Start(ctx, scenario, engagementMode)
		if err != nil {
			return nil, fmt.Errorf("baseline start --mode %s for %q: %w", engagementMode, scenario, err)
		}
		mode := strings.TrimSpace(eng.Mode)
		if mode == "" {
			mode = engagementMode
		}
		engagements[scenario] = mode
		slog.Info("baseline engagement: opened", "scenario", scenario, "mode", mode)
	}
	if len(engagements) == 0 {
		return nil, nil
	}
	return engagements, nil
}

// promoteOwnerSet promotes every engagement in an owner's set — the candidate
// becomes the new baseline (drop the restore point, re-point live to the merged
// working tree). Best-effort per scenario: one failure is logged and left for
// `baseline gc`, never blocking the rest. excludeRun threads the editing run(s)
// into the promote drain self-guard (memory part 12).
func (s *Service) promoteOwnerSet(ctx context.Context, set EngagementSet, excludeRun string) {
	if s.baselineEngagementRunner == nil {
		return
	}
	for _, scenario := range set.scenarios() {
		if err := s.baselineEngagementRunner.Promote(ctx, BaselinePromoteParams{
			Scenario:   scenario,
			ExcludeRun: excludeRun,
		}); err != nil {
			slog.Warn("baseline engagement: promote failed (reap via `baseline gc`)",
				"scenario", scenario, "owner", set.Owner, "err", err)
			continue
		}
		slog.Info("baseline engagement: promoted", "scenario", scenario, "owner", set.Owner)
	}
}

// abandonOwnerSet abandons every engagement in an owner's set — discard the
// candidate, restore the baseline over the working tree so a botched change
// never strands the scenario broken. Best-effort per scenario.
func (s *Service) abandonOwnerSet(ctx context.Context, set EngagementSet) {
	if s.baselineEngagementRunner == nil {
		return
	}
	for _, scenario := range set.scenarios() {
		if err := s.baselineEngagementRunner.Abandon(ctx, scenario); err != nil {
			slog.Warn("baseline engagement: abandon failed (reap via `baseline gc`)",
				"scenario", scenario, "owner", set.Owner, "err", err)
			continue
		}
		slog.Info("baseline engagement: abandoned (candidate discarded, baseline restored)",
			"scenario", scenario, "owner", set.Owner)
	}
}
