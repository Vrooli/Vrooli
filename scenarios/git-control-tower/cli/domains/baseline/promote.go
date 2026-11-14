// promote.go implements `git-control-tower baseline promote` — the terminal
// "keep" engagement verb (Baseline Modes, plan phase P2 / Contract Decision §8).
//
// promote is the one engagement verb that mutates LIVE, so it is the most
// safety-critical. It stays a thin orchestration over the trusted-base floor
// (`vrooli recovery`/`vrooli scenario`), the data substrate
// (`data-backup-manager safety backup-now`), and the agent-run drain
// (`agent-manager run quiesce`, the P6 promote-quiesce primitive). GCT owns no
// recovery state here either — auto-rollback restores from the floor-owned
// restore point.
//
// Shadow-mode promote sequence (§8):
//
//	quiesce/drain live  →  pre-promote data snapshot (secondary location)
//	  →  apply managed schema migrations to live (dry-run against a throwaway copy
//	      first; bounce on failure with live untouched; no scripts ⇒ shape-
//	      unchanged fast path; SQLite engine in v1 — `vrooli recovery migrate`)
//	  →  RE-POINT live from the frozen baseline copy to the blessed working tree by
//	      collapsing the shadow split (`recovery set-mode --mode live`): while the
//	      engagement is a shadow split the lifecycle EngagementResolver routes a
//	      live restart to the restore-point copy, so a plain restart would relaunch
//	      the OLD code; flipping to live mode makes the resolver stop redirecting so
//	      live resolves to the working tree. The restore-point copy is preserved as
//	      the rollback source (dropped only after the probe passes).
//	  →  restart live  →  health+smoke probe  →  caller's --probe-cmd (optional)
//	  →  (probe fails ⇒ re-open the split [`set-mode --mode shadow`] + restart =
//	      auto-rollback: live resolves back to the baseline copy, the working tree
//	      keeps the candidate, the shadow is left standing, the engagement stays
//	      open for retry — no working-tree restore needed)
//	  →  tear down the shadow  →  clean the engagement (drop the restore point)
//
// The clean closes the engagement; `baseline cycle` (cycle.go) follows it with a
// fresh shadow under the same name. A promote interrupted between the re-point
// and the clean (mode live, variant shadow) is resumed by a retry, not accepted
// in place.
//
// Live-mode promote is "accept": the working tree was edited+validated in place,
// so promote just drops the restore point + manifest (the safety net is no
// longer needed). The shadow's copied DB/namespaces are validation-only and are
// NEVER promoted.
package baseline

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	impactv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/impact"
)

// defaultDrainTimeout bounds the quiesce wait before promote aborts (or, with
// --force, cancels survivors). Mirrors the agent-manager quiesce default.
const defaultDrainTimeout = 5 * time.Minute

// promoteParams are the resolved inputs to a promote run.
type promoteParams struct {
	scenario     string
	slug         string
	excludeRun   string // the promoting orchestrator run's own ID — excluded from the drain set (self-deadlock guard)
	tagPrefix    string // also drain whole-repo runs by this tag (EM tags its runs; task ScopePath = vrooliRoot, not scenarios/<X>)
	scopePrefix  string // override the working-tree scope used to enumerate runs
	drainTimeout time.Duration
	force        bool          // on drain timeout, cancel survivors instead of aborting
	noDrain      bool          // skip the drain entirely (e.g. no agent-manager reachable / no in-flight runs expected)
	probeCmd     string        // optional live probe (`sh -c`) run after the restart and before the shadow teardown; failure rolls back
	probeTimeout time.Duration // bound on probeCmd (default defaultProbeTimeout)
}

// promoteResult is the structured outcome of a promote (also the --json shape).
type promoteResult struct {
	Scenario     string   `json:"scenario"`
	Slug         string   `json:"slug"`
	Mode         string   `json:"mode"`
	Promoted     bool     `json:"promoted"`
	RolledBack   bool     `json:"rolledBack,omitempty"`
	Drained      bool     `json:"drained"`
	DataSnapshot string   `json:"dataSnapshot,omitempty"` // pre-promote backup run ID, if one was taken
	Steps        []string `json:"steps"`
	Message      string   `json:"message"`
	// Next names the command that re-creates the shadow after a shadow promote
	// closed the engagement (same --name, so the goal keeps one engagement).
	Next string `json:"next,omitempty"`
}

// promoteFlags are the inputs `baseline promote` and `baseline cycle` share, so
// both gate, drain and probe live the same way.
type promoteFlags struct {
	scenario, slug                       string
	excludeRun, tagPrefix, scopePrefix   string
	drainTimeout, probeCmd, probeTimeout string
	force, noDrain, jsonOut              bool
}

func (f *promoteFlags) bind(fs *flag.FlagSet, defaultSlug, nameUsage string) {
	fs.StringVar(&f.scenario, "scenario", "", "Scenario slug (required)")
	fs.StringVar(&f.slug, "name", defaultSlug, nameUsage)
	fs.StringVar(&f.excludeRun, "exclude-run", "", "The promoting orchestrator run's own ID, excluded from the drain set (self-deadlock guard)")
	fs.StringVar(&f.tagPrefix, "tag-prefix", "", "Also drain in-flight runs by this tag prefix (whole-repo orchestrator runs)")
	fs.StringVar(&f.scopePrefix, "scope-prefix", "", "Override the working-tree scope used to enumerate runs (default scenarios/<scenario>)")
	fs.StringVar(&f.drainTimeout, "drain-timeout", "", "Max wait for in-flight runs to terminate before abort/--force (e.g. 5m; default 5m)")
	fs.StringVar(&f.probeCmd, "probe-cmd", "", "Live probe, run with `sh -c` against live after the restart and before the shadow teardown (e.g. one journey); a non-zero exit rolls back")
	fs.StringVar(&f.probeTimeout, "probe-timeout", "", "Max run time for --probe-cmd (default 15m); a timeout rolls back")
	fs.BoolVar(&f.force, "force", false, "Bypass the proto impact gate, and on drain timeout cancel survivors instead of aborting the promote")
	fs.BoolVar(&f.noDrain, "no-drain", false, "Skip the in-flight-run drain (no agent-manager / no concurrent runs expected)")
	fs.BoolVar(&f.jsonOut, "json", false, "Emit JSON")
}

func (f promoteFlags) params() (promoteParams, error) {
	p := promoteParams{
		scenario: f.scenario, slug: f.slug, excludeRun: f.excludeRun, tagPrefix: f.tagPrefix,
		scopePrefix: f.scopePrefix, force: f.force, noDrain: f.noDrain, probeCmd: strings.TrimSpace(f.probeCmd),
	}
	if s := strings.TrimSpace(f.drainTimeout); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return promoteParams{}, fmt.Errorf("invalid --drain-timeout %q: %w", f.drainTimeout, err)
		}
		p.drainTimeout = d
	}
	if s := strings.TrimSpace(f.probeTimeout); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return promoteParams{}, fmt.Errorf("invalid --probe-timeout %q: %w", f.probeTimeout, err)
		}
		p.probeTimeout = d
	}
	return p, nil
}

func runPromoteCmd(core *cliapp.ScenarioApp, args []string) error {
	var f promoteFlags
	fs := newFlagSet("baseline promote")
	f.bind(fs, defaultEngagementSlug, "Engagement slug (pass the engagement's --name; default wip)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := f.params()
	if err != nil {
		return err
	}

	res, err := promoteEngagement(core, p)
	if err != nil {
		return err
	}
	if f.jsonOut {
		return printJSON(res)
	}
	printPromote(res)
	return nil
}

// promoteEngagement runs the full promote sequence. Split from runPromoteCmd so
// it is unit-testable with an injected runCommand (no os.Exit, no flag parsing).
func promoteEngagement(core *cliapp.ScenarioApp, p promoteParams) (promoteResult, error) {
	scenario := strings.TrimSpace(p.scenario)
	if scenario == "" {
		return promoteResult{}, fmt.Errorf("--scenario is required")
	}
	slug := strings.TrimSpace(p.slug)
	if slug == "" {
		slug = defaultEngagementSlug
	}
	ctx := context.Background()

	eng, err := readEngagement(ctx, scenario, slug)
	if err != nil {
		return promoteResult{}, err
	}
	res := promoteResult{Scenario: scenario, Slug: slug, Mode: eng.Mode}
	if p.force {
		res.Steps = append(res.Steps, "proto impact gate bypassed (--force)")
	} else if err := checkProtoImpactGate(ctx, scenario); err != nil {
		res.Message = "promote aborted: proto impact gate failed"
		return res, err
	} else {
		res.Steps = append(res.Steps, "proto impact gate passed")
	}

	// A retry of a promote interrupted after its re-point must finish it —
	// restart, probe, tear down, clean — never "accept in place", which would
	// drop the restore point while live may still run unprobed code. Every step
	// below is safe to repeat.
	resumed := interruptedPromote(eng)

	// Live-mode promote = accept in place: the working tree was edited+validated
	// live, so there is nothing to swap — just drop the safety net (after the
	// optional live probe, whose failure keeps the restore point for abandon).
	if eng.Mode != modeShadow && !resumed {
		if p.probeCmd != "" {
			detail, ok := runLiveProbe(ctx, scenario, p.probeCmd, p.probeTimeout)
			if !ok {
				res.Steps = append(res.Steps, "✗ "+detail+" — engagement kept open; `baseline abandon` restores the restore point")
				res.Message = "promote aborted: " + detail
				return res, fmt.Errorf("promote aborted: %s", detail)
			}
			res.Steps = append(res.Steps, "live probe passed: "+p.probeCmd)
		}
		if _, err := runCommand(ctx, "vrooli", "recovery", "clean", "--scenario", scenario, "--slug", slug); err != nil {
			return promoteResult{}, fmt.Errorf("clean engagement: %w", err)
		}
		res.Promoted = true
		res.Steps = append(res.Steps, "live engagement accepted in place; restore point + manifest dropped")
		res.Message = "promoted (live, accepted in place)"
		return res, nil
	}

	// ---- shadow → live promote -------------------------------------------

	if resumed {
		res.Mode = modeShadow
		res.Steps = append(res.Steps, "resuming a promote interrupted after its re-point (engagement mode live, variant "+eng.Variant+")")
	}

	// 1. Drain in-flight agent runs targeting live so the restart can't kill
	//    work mid-flight. Default policy is abort-on-timeout (never destroy
	//    others' in-flight work — promote is re-runnable); --force cancels.
	if !p.noDrain {
		drained, msg, drainErr := drainLiveRuns(ctx, scenario, p)
		if drainErr != nil {
			return promoteResult{}, drainErr
		}
		res.Drained = drained
		if !drained {
			// Aborted: leave live and the shadow exactly as they are; the
			// engagement stays open so promote can be retried.
			res.Message = msg
			res.Steps = append(res.Steps, "drain aborted — promote not started")
			return res, fmt.Errorf("promote aborted: %s", msg)
		}
		res.Steps = append(res.Steps, "drained in-flight runs targeting live")
	} else {
		res.Steps = append(res.Steps, "drain skipped (--no-drain)")
	}

	// 2. Pre-promote data snapshot to the secondary safety location. Best-effort:
	//    a code-only scenario has no registered stateful targets, and the
	//    DB-shape-unchanged fast path never mutates live data — so a missing
	//    snapshot must not block promote, but its absence and its cause are
	//    surfaced (it is the data-rollback net if a migration ever does mutate
	//    live). The run is awaited (bounded) so the migrations below never race
	//    the capture.
	if snapID, note := safetyBackupNow(ctx, scenario); snapID == "" {
		res.Steps = append(res.Steps, "pre-promote data snapshot skipped — "+note)
	} else {
		res.DataSnapshot = snapID
		switch status := awaitSafetyRun(ctx, snapID); status {
		case "RUN_STATUS_COMPLETED":
			res.Steps = append(res.Steps, "pre-promote data snapshot completed: "+snapID)
		case "":
			res.Steps = append(res.Steps, fmt.Sprintf("⚠ pre-promote data snapshot %s still running after the poll budget — promote continues; it may not be a complete rollback point", snapID))
		default:
			res.Steps = append(res.Steps, fmt.Sprintf("⚠ pre-promote data snapshot %s ended %s — promote continues; inspect `data-backup-manager runs get %s`", snapID, status, snapID))
		}
	}

	// 3. Apply any managed schema migrations to live BEFORE the restart picks up
	//    the new code. The runner dry-runs the scripts against a throwaway copy of
	//    the current database first, so an incompatible script becomes a detected
	//    bounce (live untouched) rather than silent corruption; the universal case
	//    — no scripts authored — is the shape-unchanged fast path (a no-op). v1
	//    applies SQLite only; an authored non-SQLite script bounces the promote
	//    here rather than guessing.
	migNote, migOK := applyMigrations(ctx, scenario, slug)
	if !migOK {
		// Nothing has touched live code yet (restart is below), so a bounce just
		// leaves live as-is and keeps the engagement open for a retry.
		res.Steps = append(res.Steps, "✗ migration bounced (live untouched): "+migNote)
		res.Message = "promote aborted: migration bounced: " + migNote
		return res, fmt.Errorf("promote aborted: migration bounced: %s", migNote)
	}
	res.Steps = append(res.Steps, "migrations: "+migNote)

	// 4. Re-point live from the frozen baseline copy to the blessed working tree.
	//    While the engagement is a shadow split, the lifecycle EngagementResolver
	//    routes a live (re)start to the restore-point copy (the old baseline), so a
	//    plain restart would relaunch the OLD code, not the validated edits.
	//    Collapsing the split (flip the engagement to live mode) makes the resolver
	//    stop redirecting, so live resolves to the working tree. The restore-point
	//    copy is preserved on disk as the rollback source — it is dropped only after
	//    the probe passes (step 7). The re-point is now an explicit, observable step
	//    rather than a no-op restart.
	if _, err := runCommand(ctx, "vrooli", "recovery", "set-mode", "--scenario", scenario, "--slug", slug, "--mode", modeLive); err != nil {
		// Nothing has restarted yet; live still serves the baseline from the copy.
		// Leave the engagement open (still a shadow split) for a retry.
		res.Steps = append(res.Steps, fmt.Sprintf("✗ re-point failed (live untouched): %v", err))
		res.Message = "promote aborted: re-point failed"
		return res, fmt.Errorf("promote aborted: re-point: %w", err)
	}
	res.Steps = append(res.Steps, "live re-pointed from the baseline copy to the blessed working tree (split collapsed)")
	if _, err := runCommand(ctx, "vrooli", "scenario", "restart", scenario); err != nil {
		// Restart failed after the re-point → roll back (re-open the split + restart).
		return rollback(ctx, res, scenario, slug, fmt.Sprintf("restart failed: %v", err))
	}
	res.Steps = append(res.Steps, "live restarted on the working tree")

	// 5. Health + smoke probe. On failure, auto-rollback (swap code back from the
	//    restore point + restart).
	if healthy, detail := probeLiveHealth(ctx, scenario); !healthy {
		return rollback(ctx, res, scenario, slug, "health probe failed: "+detail)
	}
	res.Steps = append(res.Steps, "live health+smoke probe passed")

	// 5b. Caller's live probe (--probe-cmd, e.g. one journey). It runs while the
	//     restore point and the shadow still exist, so a failure takes the same
	//     auto-rollback as a failed restart.
	if p.probeCmd != "" {
		if detail, ok := runLiveProbe(ctx, scenario, p.probeCmd, p.probeTimeout); !ok {
			return rollback(ctx, res, scenario, slug, detail)
		}
		res.Steps = append(res.Steps, "live probe passed: "+p.probeCmd)
	}

	// 6. Tear down the shadow (validation-only; never promoted) — best-effort.
	variant := eng.Variant
	if variant == "" {
		variant = modeShadow
	}
	if _, err := runCommand(ctx, "vrooli", "scenario", "stop", scenario, "--instance", variant); err != nil {
		// The promote already succeeded; a lingering shadow is reaper/gc fodder,
		// not a promote failure. Surface it without failing.
		res.Steps = append(res.Steps, fmt.Sprintf("⚠ shadow teardown failed (reap via `baseline gc`): %v", err))
	} else {
		res.Steps = append(res.Steps, "shadow torn down")
	}

	// 7. Close the engagement (drop restore point + manifest).
	if _, err := runCommand(ctx, "vrooli", "recovery", "clean", "--scenario", scenario, "--slug", slug); err != nil {
		return promoteResult{}, fmt.Errorf("promote succeeded but cleaning the engagement failed: %w", err)
	}
	res.Steps = append(res.Steps, "engagement closed (restore point + manifest dropped)")

	res.Promoted = true
	res.Message = "promoted (shadow → live)"
	res.Next = recreateShadowCommand(scenario, slug, eng)
	return res, nil
}

// interruptedPromote reports a shadow engagement whose promote stopped between
// the re-point (step 4) and the clean (step 7): set-mode recorded mode live, but
// the variant is still the shadow's. A real live engagement has variant live.
func interruptedPromote(eng engagementView) bool {
	return eng.Mode == modeLive && eng.Variant != "" && eng.Variant != modeLive
}

// recreateShadowCommand is the `baseline start` that opens a fresh shadow like
// the engagement a promote just closed: the same --name (a bare start would
// open "wip", a second engagement), the same idle TTL, and an anchor only when
// the closed engagement had one.
func recreateShadowCommand(scenario, slug string, eng engagementView) string {
	cmd := fmt.Sprintf("git-control-tower baseline start --scenario %s --name %s --mode shadow", scenario, slug)
	if d, err := time.ParseDuration(strings.TrimSpace(eng.TTL)); err == nil && d > 0 {
		cmd += " --ttl " + d.String()
	}
	if strings.TrimSpace(eng.AnchorBaselineName) == "" {
		cmd += " --no-anchor"
	}
	return cmd
}

func checkProtoImpactGate(ctx context.Context, scenario string) error {
	out, err := runCommand(ctx, "proto-health", "impact", "scenario", scenario, "--json")
	if err != nil {
		return fmt.Errorf("proto impact gate: %w", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		return nil
	}
	var resp impactv1.GetImpactResponse
	if err := protojson.Unmarshal(out, &resp); err != nil {
		return fmt.Errorf("proto impact gate: parse impact report: %w", err)
	}
	report := resp.GetReport()
	if report == nil || report.GetStableUnreconciledBreakingCount() == 0 {
		return nil
	}
	return fmt.Errorf("promote blocked: %d stable proto breaking change(s) still have unreconciled consumers; rerun `proto-health impact scenario %s` or use --force with a recorded reason", report.GetStableUnreconciledBreakingCount(), scenario)
}

// drainLiveRuns shells the agent-manager promote-quiesce primitive (P6) and
// reports whether live is now quiet. Returns (drained, human-message, error).
// A non-nil error is a hard failure (e.g. the self-deadlock guard rejecting a
// run that is promoting its own scenario); (false, msg, nil) is a clean abort.
func drainLiveRuns(ctx context.Context, scenario string, p promoteParams) (bool, string, error) {
	args := []string{"run", "quiesce", "--scenario", scenario, "--json"}
	if p.scopePrefix != "" {
		args = append(args, "--scope-prefix", p.scopePrefix)
	}
	if p.tagPrefix != "" {
		args = append(args, "--tag-prefix", p.tagPrefix)
	}
	if p.excludeRun != "" {
		args = append(args, "--exclude-run", p.excludeRun)
	}
	timeout := p.drainTimeout
	if timeout <= 0 {
		timeout = defaultDrainTimeout
	}
	args = append(args, "--timeout", timeout.String())
	if p.force {
		args = append(args, "--force")
	}

	out, err := runCommand(ctx, "agent-manager", args...)
	if err != nil {
		// The drain command failing (incl. the self-deadlock guard's validation
		// rejection) is a hard stop — promote must not proceed blind.
		return false, "", fmt.Errorf("drain in-flight runs (promote-quiesce): %w", err)
	}
	q := parseQuiesce(out)
	if q.drained {
		return true, "live drained", nil
	}
	msg := q.reason
	if msg == "" {
		msg = fmt.Sprintf("%d run(s) still in-flight against %s — retry after they finish, or pass --force", len(q.inFlight), scenario)
	}
	return false, msg, nil
}

// applyMigrations runs the engagement's managed schema migrations against live
// via the trusted-base runner (`vrooli recovery migrate`), which dry-runs the
// scripts against a throwaway copy of the current database first. The universal
// case — no scripts authored — is the shape-unchanged fast path (a no-op).
// Returns a human note for the step log and ok=false when the migration bounced
// (an incompatible/invalid script, or a non-SQLite engine in v1) so promote can
// abort with live untouched.
func applyMigrations(ctx context.Context, scenario, slug string) (string, bool) {
	out, err := runCommand(ctx, "vrooli", "recovery", "migrate", "--scenario", scenario, "--slug", slug, "--json")
	if err != nil {
		note := strings.TrimSpace(string(out))
		if note == "" {
			note = err.Error()
		}
		return note, false
	}
	// Decode the typed vrooli.cli.v1 recovery-migrate contract. runCommand
	// (not the typed client) stays the seam here so the exec-failure path above
	// is distinct from a decode miss below — the runner exited 0, so a parse miss
	// must not block an otherwise-clean promote (mirrors probeLiveHealth).
	var r cliv1.RecoveryMigrateOutput
	if (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(out, &r) != nil {
		return "migrations applied (unparseable runner output)", true
	}
	if r.GetFastPath() {
		return "no migration scripts — shape-unchanged fast path", true
	}
	return fmt.Sprintf("applied %d migration script(s); %d already-applied", len(r.GetApplied()), len(r.GetSkipped())), true
}

// quiesceResult is the subset of the agent-manager QuiesceResult JSON promote reads.
type quiesceResult struct {
	drained  bool
	aborted  bool
	reason   string
	inFlight []string
}

// parseQuiesce decodes the QuiesceScenarioResponse the CLI prints. The result
// nests under "result"; agent-manager writes proto field names (in_flight), and
// the lowerCamel spelling (inFlight) is accepted too.
func parseQuiesce(out []byte) quiesceResult {
	type runRef struct {
		ID string `json:"id"`
	}
	var resp struct {
		Result struct {
			Drained       bool     `json:"drained"`
			Aborted       bool     `json:"aborted"`
			Reason        string   `json:"reason"`
			InFlight      []runRef `json:"in_flight"`
			InFlightCamel []runRef `json:"inFlight"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return quiesceResult{}
	}
	q := quiesceResult{drained: resp.Result.Drained, aborted: resp.Result.Aborted, reason: resp.Result.Reason}
	for _, r := range append(resp.Result.InFlight, resp.Result.InFlightCamel...) {
		q.inFlight = append(q.inFlight, r.ID)
	}
	return q
}

// probeLiveHealth asks the lifecycle whether live is running after the restart.
// It parses `vrooli scenario status <s> --json` defensively across the shapes
// the CLI may emit (top-level / scenario.* / details.*). When no status string
// can be extracted, the restart having succeeded is treated as sufficient (the
// probe never false-fails a healthy promote on a parse miss).
func probeLiveHealth(ctx context.Context, scenario string) (bool, string) {
	out, err := runCommand(ctx, "vrooli", "scenario", "status", scenario, "--json")
	if err != nil {
		return false, fmt.Sprintf("status query failed: %v", err)
	}
	status := extractStatus(out)
	switch status {
	case "":
		// Unparseable but restart succeeded — do not block a healthy promote.
		return true, ""
	case "running":
		return true, ""
	default:
		return false, "live status is " + status + " after restart"
	}
}

// extractStatus pulls a lifecycle status string out of the scenario-status JSON,
// tolerating the top-level, scenario-wrapped, and details-wrapped shapes.
func extractStatus(out []byte) string {
	var v struct {
		Status   string `json:"status"`
		Scenario struct {
			Status string `json:"status"`
		} `json:"scenario"`
		Details struct {
			Status string `json:"status"`
		} `json:"details"`
	}
	if json.Unmarshal(out, &v) != nil {
		return ""
	}
	for _, s := range []string{v.Scenario.Status, v.Details.Status, v.Status} {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// rollback performs the auto-rollback after a failed shadow promote: it re-opens
// the shadow split (flip the engagement back to shadow mode) and restarts live,
// which the resolver then routes back to the frozen restore-point copy — the old
// baseline. No working-tree restore is needed: unlike the pre-floor model live
// never ran from the working tree, so the working tree is left holding the
// candidate (the in-progress code, untouched for a retry). The shadow instance is
// left standing for diagnosis; `baseline abandon`/`gc` reap it. Returns the
// result annotated as rolled-back plus a non-nil error describing the cause.
func rollback(ctx context.Context, res promoteResult, scenario, slug, cause string) (promoteResult, error) {
	res.Steps = append(res.Steps, "✗ "+cause+" — auto-rolling back")
	if _, err := runCommand(ctx, "vrooli", "recovery", "set-mode", "--scenario", scenario, "--slug", slug, "--mode", modeShadow); err != nil {
		res.Steps = append(res.Steps, fmt.Sprintf("⚠ re-point back to the baseline copy FAILED: %v — see docs/operations/manual-scenario-recovery.md", err))
		return res, fmt.Errorf("promote failed (%s) AND auto-rollback re-point failed: %w", cause, err)
	}
	if _, err := runCommand(ctx, "vrooli", "scenario", "restart", scenario); err != nil {
		res.Steps = append(res.Steps, fmt.Sprintf("⚠ restart after rollback FAILED: %v", err))
		return res, fmt.Errorf("promote failed (%s); re-pointed back to baseline but restart failed: %w", cause, err)
	}
	res.RolledBack = true
	res.Steps = append(res.Steps, "live re-pointed back to the baseline copy and restarted; shadow left standing, engagement open for retry")
	dataNote := ""
	if res.DataSnapshot != "" {
		dataNote = fmt.Sprintf(" (data snapshot %s available for manual restore)", res.DataSnapshot)
	}
	res.Message = "promote rolled back: " + cause + dataNote
	return res, fmt.Errorf("promote rolled back: %s%s", cause, dataNote)
}
