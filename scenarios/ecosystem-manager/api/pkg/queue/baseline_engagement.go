package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// Baseline Modes engagement (plan P6 §192). When an autosteer profile enables
// its BaselinePromote block, the orchestrator fronts an improvement run with a
// `git-control-tower baseline start` engagement: the target is stood up as a
// shadow instance (where the coding agent edits + the loop validates), and on
// the controller's terminal decision the shadow is promoted to live (objective
// met) or abandoned (not met). This file owns the queue-side engagement
// lifecycle — the runner that shells GCT, plus the per-task engagement state the
// AutoSteerIntegration drives. It lives in pkg/queue (not pkg/autosteer) because
// the agent-run spawn, tag, and run-ID it threads into the promote drain all
// live here.

// defaultSelfScenario is the scenario the autosteer loop itself runs as. Baseline
// Modes never engages on it: promoting/abandoning ecosystem-manager would restart
// the very loop doing the promoting (the self-deadlock the plan externalizes to a
// one-shot). For every OTHER scenario the engagement runs normally; the GCT
// promote's quiesce self-guard remains the backstop.
const defaultSelfScenario = "ecosystem-manager"

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
	// ExcludeRun is the orchestrator run excluded from the drain set (the run that
	// just edited the scenario); empty when unknown.
	ExcludeRun string
	// TagPrefix scopes the drain to this task's spawned runs.
	TagPrefix string
}

// BaselineEngagementRunner is the queue's boundary to `git-control-tower
// baseline`. Production wires GCTBaselineRunner (shells the CLI); tests wire a
// fake. Start returns the decided engagement; Promote/Abandon close it.
type BaselineEngagementRunner interface {
	Start(ctx context.Context, scenario string) (BaselineEngagement, error)
	Promote(ctx context.Context, p BaselinePromoteParams) error
	Abandon(ctx context.Context, scenario string) error
}

// GCTBaselineRunner shells the git-control-tower CLI's `baseline` verbs. It
// mirrors findings.TestGenieRunner: a project-root working dir, a PATH-resolved
// binary (overridable via $ECOSYSTEM_MANAGER_GCT_BIN), and a per-call timeout.
type GCTBaselineRunner struct {
	// ProjectRoot is the repo root; the command runs with this as its working dir.
	ProjectRoot string
	// Binary is the git-control-tower executable; defaults to "git-control-tower"
	// (or $ECOSYSTEM_MANAGER_GCT_BIN) resolved on PATH.
	Binary string
	// StartTimeout bounds `baseline start` (stands up a shadow instance). Zero ⇒
	// defaultEngagementStartTimeout.
	StartTimeout time.Duration
	// PromoteTimeout bounds `baseline promote`/`abandon`. Zero ⇒
	// defaultEngagementPromoteTimeout.
	PromoteTimeout time.Duration
}

const (
	defaultEngagementStartTimeout   = 10 * time.Minute
	defaultEngagementPromoteTimeout = 15 * time.Minute
)

var _ BaselineEngagementRunner = (*GCTBaselineRunner)(nil)

func (r *GCTBaselineRunner) binary() string {
	if strings.TrimSpace(r.Binary) != "" {
		return r.Binary
	}
	if env := strings.TrimSpace(os.Getenv("ECOSYSTEM_MANAGER_GCT_BIN")); env != "" {
		return env
	}
	return "git-control-tower"
}

func (r *GCTBaselineRunner) run(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
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

// startJSON is the subset of `baseline start --json` (GCT startResult) the runner
// reads to learn the decided mode + ambient routing var.
type startJSON struct {
	Scenario   string `json:"scenario"`
	Variant    string `json:"variant"`
	AmbientVar string `json:"ambientVar"`
	Decision   struct {
		Mode      string `json:"mode"`
		Reflexive bool   `json:"reflexive"`
	} `json:"decision"`
}

func (r *GCTBaselineRunner) Start(ctx context.Context, scenario string) (BaselineEngagement, error) {
	timeout := r.StartTimeout
	if timeout <= 0 {
		timeout = defaultEngagementStartTimeout
	}
	out, err := r.run(ctx, timeout, "baseline", "start", "--scenario", scenario, "--mode", "auto", "--json")
	if err != nil {
		return BaselineEngagement{}, err
	}
	return parseStartJSON(scenario, out)
}

// parseStartJSON decodes `baseline start --json` (GCT startResult) into the
// runner's engagement view. Split out so the contract is unit-testable without a
// git-control-tower binary.
func parseStartJSON(scenario, out string) (BaselineEngagement, error) {
	var res startJSON
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

func (r *GCTBaselineRunner) Promote(ctx context.Context, p BaselinePromoteParams) error {
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

func (r *GCTBaselineRunner) Abandon(ctx context.Context, scenario string) error {
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

// taskEngagement is a task's live Baseline Modes engagement state, held on the
// AutoSteerIntegration between EvaluateStart (open) and the controller's terminal
// decision (promote/abandon). active=false is a sentinel meaning "checked, not
// engaged" so the profile is loaded at most once per task.
type taskEngagement struct {
	scenario    string
	mode        string // "shadow" | "live"
	ambientVar  string // scenario routed to its shadow (shadow mode only)
	promoteMode string // end_of_engagement | checkpoint_on_green
	cadence     int    // checkpoint throttle (0 ⇒ every iteration)
	tagPrefix   string // this task's agent-run tag (promote drain scope)
	runID       string // most recent spawned run (promote --exclude-run guard)
	active      bool
}

// maybeStartEngagement opens a Baseline Modes engagement the first time a task is
// about to run an agent, when its profile's BaselinePromote block is enabled. It
// is idempotent (a sentinel after the first check loads the profile once) and
// best-effort (a runner error degrades to in-place editing — the loop still runs,
// just without a shadow + promote).
func (a *AutoSteerIntegration) maybeStartEngagement(task *tasks.TaskItem, scenarioName string) {
	if a == nil || a.baselineRunner == nil || task == nil {
		return
	}
	a.engMu.Lock()
	if _, seen := a.engagements[task.ID]; seen {
		a.engMu.Unlock()
		return
	}
	// Reserve a sentinel so a concurrent re-entry never double-starts.
	a.engagements[task.ID] = &taskEngagement{active: false}
	a.engMu.Unlock()

	profile, err := a.executionOrchestrator.ProfileForTask(task.ID)
	if err != nil || profile == nil || !profile.BaselinePromoteEnabled() {
		return // sentinel stays: no engagement
	}
	scenario := strings.TrimSpace(scenarioName)
	if scenario == "" {
		return
	}
	if scenario == a.selfScenario {
		log.Printf("Baseline Modes: task %s targets self (%s) — engagement externalized, editing in place", task.ID, scenario)
		systemlog.Infof("Baseline Modes: task %s self-improvement externalized (no shadow engagement)", task.ID)
		return
	}

	eng, err := a.baselineRunner.Start(context.Background(), scenario)
	if err != nil {
		log.Printf("Baseline Modes: baseline start failed for %s (task %s) — editing in place: %v", scenario, task.ID, err)
		systemlog.Errorf("Baseline Modes: engagement start failed for %s: %v", scenario, err)
		return
	}

	te := &taskEngagement{
		scenario:    scenario,
		mode:        eng.Mode,
		ambientVar:  eng.AmbientVar,
		promoteMode: profile.BaselinePromoteMode(),
		cadence:     profile.BaselinePromoteCadence(),
		tagPrefix:   makeAgentTag(task.ID),
		active:      true,
	}
	a.engMu.Lock()
	a.engagements[task.ID] = te
	a.engMu.Unlock()
	log.Printf("Baseline Modes: engagement open for %s (task %s) mode=%s promote=%s", scenario, task.ID, te.mode, te.promoteMode)
	systemlog.Infof("Baseline Modes: engagement open for %s — mode %s, promote %s", scenario, te.mode, te.promoteMode)
}

// ShadowScenarioForTask returns the scenario whose nested CLI calls a spawned
// agent should route to its shadow instance (empty unless a shadow engagement is
// active). The spawn site unions this into VROOLI_SHADOW_SCENARIOS.
func (a *AutoSteerIntegration) ShadowScenarioForTask(taskID string) string {
	if a == nil {
		return ""
	}
	a.engMu.Lock()
	defer a.engMu.Unlock()
	te := a.engagements[taskID]
	if te == nil || !te.active || te.mode != "shadow" {
		return ""
	}
	if strings.TrimSpace(te.ambientVar) != "" {
		return te.ambientVar
	}
	return te.scenario
}

// SetEngagementRunID records the most recent agent-manager run spawned for a
// task, threaded into the promote drain as --exclude-run (the self-deadlock
// guard). No-op when the task has no active engagement.
func (a *AutoSteerIntegration) SetEngagementRunID(taskID, runID string) {
	if a == nil || strings.TrimSpace(runID) == "" {
		return
	}
	a.engMu.Lock()
	defer a.engMu.Unlock()
	if te := a.engagements[taskID]; te != nil && te.active {
		te.runID = runID
	}
}

// maybeFinishEngagement closes a task's engagement at the controller's terminal
// decision: promote (objective met) or abandon (otherwise). Best-effort — a
// runner error is logged, never propagated (the controller already stopped).
func (a *AutoSteerIntegration) maybeFinishEngagement(taskID, reason string) {
	te := a.takeEngagement(taskID)
	if te == nil {
		return
	}
	a.closeEngagement(taskID, te, reason == autosteer.StopObjectiveMet, reason)
}

// maybeCheckpointPromote promotes a still-running engagement early when the
// checkpoint_on_green cadence observes an already-met objective. Returns true
// when it promoted (the caller must then stop the loop). No-op for
// end_of_engagement or a checkpoint that is not yet due / not yet green.
func (a *AutoSteerIntegration) maybeCheckpointPromote(taskID string, eval *autosteer.IterationEvaluation) bool {
	if eval == nil || !eval.ObjectiveMet {
		return false
	}
	a.engMu.Lock()
	te := a.engagements[taskID]
	if te == nil || !te.active || te.promoteMode != autosteer.BaselinePromoteCheckpointOnGreen {
		a.engMu.Unlock()
		return false
	}
	if te.cadence > 0 && (eval.Iteration <= 0 || eval.Iteration%te.cadence != 0) {
		a.engMu.Unlock()
		return false
	}
	delete(a.engagements, taskID)
	a.engMu.Unlock()
	a.closeEngagement(taskID, te, true, autosteer.BaselinePromoteCheckpointOnGreen)
	return true
}

// takeEngagement pops a task's engagement, returning nil for an absent task or a
// not-engaged sentinel (the entry is removed either way).
func (a *AutoSteerIntegration) takeEngagement(taskID string) *taskEngagement {
	a.engMu.Lock()
	defer a.engMu.Unlock()
	te := a.engagements[taskID]
	delete(a.engagements, taskID)
	if te == nil || !te.active {
		return nil
	}
	return te
}

// closeEngagement runs the terminal promote (green) or abandon (not green) for an
// engagement that has already been removed from the map. Network I/O runs outside
// the engagement lock.
func (a *AutoSteerIntegration) closeEngagement(taskID string, te *taskEngagement, green bool, reason string) {
	if a.baselineRunner == nil || te == nil {
		return
	}
	ctx := context.Background()
	// Anti-gaming promote-safety gate: a run that only went "green" by faking it
	// (weakened [REQ:] tests, deleted ledgers, suppression directives) must NOT be
	// promoted to live. The gameguard classifier flags such iterations on the
	// decision trace; RunGamed reports them. A gamed run is downgraded to abandon.
	if green && a.executionOrchestrator != nil {
		if gamed, err := a.executionOrchestrator.RunGamed(taskID); err != nil {
			log.Printf("Baseline Modes: gaming check failed for %s (task %s) — proceeding with promote: %v", te.scenario, taskID, err)
		} else if gamed {
			green = false
			reason = "blocked: gaming detected (faked green)"
			log.Printf("Baseline Modes: PROMOTE BLOCKED for %s (task %s) — gaming detected; abandoning the shadow instead", te.scenario, taskID)
			systemlog.Errorf("Baseline Modes: promote blocked for %s — gaming detected (faked green); shadow abandoned", te.scenario)
		}
	}
	if green {
		if err := a.baselineRunner.Promote(ctx, BaselinePromoteParams{
			Scenario: te.scenario, ExcludeRun: te.runID, TagPrefix: te.tagPrefix,
		}); err != nil {
			log.Printf("Baseline Modes: promote failed for %s (task %s, %s): %v", te.scenario, taskID, reason, err)
			systemlog.Errorf("Baseline Modes: promote failed for %s: %v", te.scenario, err)
			return
		}
		log.Printf("Baseline Modes: promoted %s (task %s, %s)", te.scenario, taskID, reason)
		systemlog.Infof("Baseline Modes: promoted %s — %s", te.scenario, reason)
		return
	}
	if err := a.baselineRunner.Abandon(ctx, te.scenario); err != nil {
		log.Printf("Baseline Modes: abandon failed for %s (task %s, %s): %v", te.scenario, taskID, reason, err)
		systemlog.Errorf("Baseline Modes: abandon failed for %s: %v", te.scenario, err)
		return
	}
	log.Printf("Baseline Modes: abandoned %s (task %s, %s)", te.scenario, taskID, reason)
	systemlog.Infof("Baseline Modes: abandoned %s — %s", te.scenario, reason)
}
