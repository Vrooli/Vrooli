package main

// Recovery floor.
//
// This file is why the loop exists as more than a restarter. On 2026-09-01 a
// new import in packages/api-core added two transitive dependencies that 98
// scenario api modules did not carry in their go.sum. Every affected scenario
// then failed to build, which made `vrooli scenario restart` a one-way door:
// it stopped the scenario and could not start it again. vrooli-autoheal was
// one of the 98. Its API -- which holds all the signature detection and every
// language recovery strategy -- could not build, so the component that knew
// how to fix "missing go.sum entry" was the one component that could not run.
//
// The loop, meanwhile, was alive the entire time (systemd, Restart=always, up
// 6d23h) and was calling `vrooli scenario start`, capturing output that
// literally contained the healable signature, and throwing it away.
//
// The fix is not more recovery actions. It is putting the minimum viable
// detector somewhere that survives the failure it repairs. langrecover is a
// standalone stdlib-only module for exactly this reason: the loop can depend
// on it without inheriting api-core's dependency graph, so dependency drift
// can never break the thing that repairs dependency drift.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	langrecover "vrooli-autoheal-langrecover"
)

// Circuit-breaker policy for the recovery floor.
//
// The loop has none of the API registry's heal-tracker machinery (cooldowns,
// exponential backoff, suspension, PID TOCTOU checks), and it runs unattended
// on a timer. A systematically broken repo must not produce an unbounded
// series of `go mod tidy` runs, so the floor is deliberately stingy: a few
// attempts in a short window, then a long suspension that survives process
// restart.
const (
	// selfHealWindow is the rolling window over which attempts are counted.
	selfHealWindow = time.Hour
	// selfHealMaxAttempts is the number of recovery runs allowed per window.
	selfHealMaxAttempts = 3
	// selfHealSuspension is how long the floor stays closed after tripping.
	selfHealSuspension = 6 * time.Hour
	// selfHealTimeout bounds a single recovery command. `go mod tidy` on a
	// cold module cache can be slow; beyond this it is not going to finish.
	selfHealTimeout = 10 * time.Minute
)

// selfHealState is the circuit-breaker state, persisted so that a systemd
// restart of the loop does not silently reset the attempt budget. Without
// persistence, a crash-restart loop would get a fresh 3 attempts every cycle,
// which is precisely the unbounded behaviour the breaker exists to prevent.
type selfHealState struct {
	Attempts        int       `json:"attempts"`
	WindowStartedAt time.Time `json:"windowStartedAt"`
	LastAttemptAt   time.Time `json:"lastAttemptAt"`
	SuspendedUntil  time.Time `json:"suspendedUntil"`
	LastStrategy    string    `json:"lastStrategy"`
	LastOutcome     string    `json:"lastOutcome"`
}

// selfHealStatePath returns the on-disk location of the breaker state.
func selfHealStatePath() string {
	stateRoot := runtimeHomeEntry(repocontract.HomeKeyState)
	if stateRoot == "" {
		return ""
	}
	return filepath.Join(stateRoot, "vrooli-autoheal", "recovery-floor.json")
}

func loadSelfHealState(path string) selfHealState {
	var state selfHealState
	if path == "" {
		return state
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return state
	}
	// A corrupt state file must fail open to "no history", never block
	// recovery forever.
	_ = json.Unmarshal(data, &state)
	return state
}

func saveSelfHealState(path string, state selfHealState) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// selfHealAllowed reports whether the breaker permits another attempt now, and
// returns a human-readable reason when it does not.
func selfHealAllowed(state selfHealState, now time.Time) (bool, string) {
	if !state.SuspendedUntil.IsZero() && now.Before(state.SuspendedUntil) {
		return false, fmt.Sprintf(
			"recovery floor suspended for another %s after %d attempts (last outcome: %s)",
			now.Sub(state.SuspendedUntil).Abs().Round(time.Minute), state.Attempts, state.LastOutcome,
		)
	}
	if state.WindowStartedAt.IsZero() || now.Sub(state.WindowStartedAt) > selfHealWindow {
		return true, ""
	}
	if state.Attempts >= selfHealMaxAttempts {
		return false, fmt.Sprintf("recovery floor budget exhausted (%d attempts within %s)",
			state.Attempts, selfHealWindow)
	}
	return true, ""
}

// recordSelfHealAttempt advances the breaker and returns the updated state.
func recordSelfHealAttempt(state selfHealState, now time.Time, strategy, outcome string) selfHealState {
	if state.WindowStartedAt.IsZero() || now.Sub(state.WindowStartedAt) > selfHealWindow {
		state.WindowStartedAt = now
		state.Attempts = 0
	}
	state.Attempts++
	state.LastAttemptAt = now
	state.LastStrategy = strategy
	state.LastOutcome = outcome
	if state.Attempts >= selfHealMaxAttempts {
		state.SuspendedUntil = now.Add(selfHealSuspension)
	}
	return state
}

// resetSelfHealState clears the breaker after a confirmed good start, so a
// healthy scenario does not carry stale attempt history into the next window.
func resetSelfHealState(path string) {
	saveSelfHealState(path, selfHealState{})
}

// selfHealRunner is the command seam used by the recovery floor. Production
// uses langrecover.DefaultRunner; tests substitute a fake so the floor can be
// exercised without spawning real `go mod tidy` subprocesses.
var selfHealRunner = langrecover.DefaultRunner

// selfHealOutcome describes what the floor did, for logging and for the
// caller's decision about whether to retry the start.
type selfHealOutcome struct {
	// Attempted is true when a recovery command actually ran.
	Attempted bool
	// Healed is true when recovery ran without error and changed files.
	// Only then is retrying the start worthwhile.
	Healed bool
	// Detail is a human-readable summary for the log.
	Detail string
	// VersionDeltas carries any module version movements caused by recovery.
	VersionDeltas []langrecover.VersionDelta
}

// attemptSelfHeal inspects a failed lifecycle command's output for a healable
// dependency-drift signature and, if one is present and the breaker allows,
// runs the matching recovery strategy.
//
// Signature source: the lifecycle CLI prints only a summary to stdout ("build
// component api: exit status 1") and writes the actual compiler error to
// ~/.vrooli/logs/<scenario>.log. So a failure whose stdout carries no signature
// is not evidence of no drift -- the floor must read the log tail too. This was
// found by live test on 2026-09-01: the unit tests passed because they were fed
// the raw compiler text, while the real loop saw only the summary and declined
// to act.
//
// Blast radius is deliberately narrow: the floor only ever repairs autoheal's
// OWN scenario directory or the repo root. It never sweeps other scenarios --
// the API registry owns those, with per-check heal trackers and cooldowns. A
// watchdog that could rewrite dependency files across the whole ecosystem
// unattended is a much worse failure mode than the outage it prevents.
func attemptSelfHeal(config *Config, failureOutput string) selfHealOutcome {
	if strings.TrimSpace(failureOutput) == "" {
		return selfHealOutcome{Detail: "no failure output to classify"}
	}
	root := config.VrooliRoot
	if root == "" {
		return selfHealOutcome{Detail: "vrooli root unknown; cannot locate scenario directory"}
	}

	scenarioDir := filepath.Join(root, "scenarios", config.ScenarioName)
	decision, source := decideFromSources(failureOutput, scenarioDir, root, config.ScenarioName)
	if !decision.Has() {
		return selfHealOutcome{Detail: "no healable dependency-drift signature in command output or lifecycle log"}
	}

	statePath := selfHealStatePath()
	state := loadSelfHealState(statePath)
	now := time.Now()
	if allowed, reason := selfHealAllowed(state, now); !allowed {
		return selfHealOutcome{Detail: reason}
	}

	ctx, cancel := context.WithTimeout(context.Background(), selfHealTimeout)
	defer cancel()

	var (
		result langrecover.Result
		err    error
	)
	switch decision.Kind {
	case langrecover.KindGo:
		result, err = langrecover.RecoverGo(ctx, selfHealRunner, decision.ScenarioDir, decision.GoSig)
	case langrecover.KindRepoRoot:
		result, err = langrecover.RecoverRepoRootGo(ctx, selfHealRunner, root, decision.GoSig)
	case langrecover.KindPnpm:
		result, err = langrecover.RecoverPnpm(ctx, selfHealRunner, decision.ScenarioDir, decision.PnpmSig)
	default:
		return selfHealOutcome{Detail: fmt.Sprintf("unsupported strategy kind %q", decision.Kind)}
	}

	outcome := "healed"
	healed := true
	switch {
	case err != nil:
		outcome, healed = "strategy-error: "+err.Error(), false
	case result.Err != nil:
		outcome, healed = "command-failed: "+result.Err.Error(), false
	case !result.ModifiedTrackedFiles:
		// The command succeeded but changed nothing, so the failure has a
		// different cause. Retrying the start would just fail identically.
		outcome, healed = "no-op (recovery changed no tracked files)", false
	}

	state = recordSelfHealAttempt(state, now, string(decision.Kind), outcome)
	saveSelfHealState(statePath, state)

	detail := fmt.Sprintf("%s in %s (signature from %s): %s", result.Command, result.WorkingDir, source, outcome)
	if len(result.ModifiedPaths) > 0 {
		detail += " [modified: " + strings.Join(result.ModifiedPaths, ", ") + "]"
	}
	changed := langrecover.ChangedVersionDeltas(result.VersionDeltas)
	if len(changed) > 0 {
		// Surfaced loudly: an unattended heal that moves a dependency version
		// is exactly the thing an operator must be able to see after the fact.
		detail += " [VERSION CHANGES: " + langrecover.FormatVersionDeltas(changed) + "]"
	}

	return selfHealOutcome{
		Attempted:     true,
		Healed:        healed,
		Detail:        detail,
		VersionDeltas: result.VersionDeltas,
	}
}

// titleVerb capitalises a lifecycle verb for log output. strings.Title is
// deprecated and locale-aware; this only ever sees ASCII verbs.
func titleVerb(verb string) string {
	if verb == "" {
		return verb
	}
	return strings.ToUpper(verb[:1]) + verb[1:]
}

// lifecycleLogTailBytes bounds how much of the run log the floor reads. Build
// failure tails are small; the cap keeps the scan cheap and bounded.
const lifecycleLogTailBytes = 32 * 1024

// readLifecycleLogTail returns the last n bytes of ~/.vrooli/logs/<scenario>.log,
// or "" when unavailable. An unreadable log is treated as "no signature", never
// as an error: the floor must degrade to doing nothing, not to guessing.
func readLifecycleLogTail(scenarioName string, n int64) string {
	logsRoot := runtimeHomeEntry(repocontract.HomeKeyLogs)
	if logsRoot == "" {
		return ""
	}
	path := filepath.Join(logsRoot, scenarioName+".log")
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.Size() == 0 {
		return ""
	}
	if info.Size() > n {
		if _, err := file.Seek(info.Size()-n, io.SeekStart); err != nil {
			return ""
		}
	}
	buf := make([]byte, n)
	read, err := file.Read(buf)
	if err != nil && read == 0 {
		return ""
	}
	return string(buf[:read])
}

// decideFromSources looks for a healable signature first in the command output
// and then in the lifecycle log, reporting which source matched.
//
// Order matters: command output is the most direct evidence and is always
// current, while the log tail can contain an older failure. The log is
// consulted only when stdout yields nothing.
func decideFromSources(commandOutput, scenarioDir, root, scenarioName string) (langrecover.Decision, string) {
	for _, candidate := range []struct {
		text   string
		source string
	}{
		{commandOutput, "command-output"},
		{readLifecycleLogTail(scenarioName, lifecycleLogTailBytes), "lifecycle-log"},
	} {
		if strings.TrimSpace(candidate.text) == "" {
			continue
		}
		if decision := langrecover.Decide(candidate.text, scenarioDir); decision.Has() {
			return decision, candidate.source
		}
		// A shared-package change can break the top-level module rather than
		// (or as well as) the scenario module.
		if decision := langrecover.DecideRepoRoot(candidate.text, root); decision.Has() {
			return decision, candidate.source
		}
	}
	return langrecover.Decision{}, ""
}
