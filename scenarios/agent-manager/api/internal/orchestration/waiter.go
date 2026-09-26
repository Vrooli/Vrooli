// This file waits for run completion and delivers terminal snapshots.
package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"agent-manager/internal/domain"
	"github.com/vrooli/envkit-go"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Per-producer Waiter seam (durable park/resume, Phase 3).
//
// When a run parks on externally-owned async work, agent-manager — which owns
// the (now-exited) agent process — performs the blocking wait the agent could
// not. That wait is producer-specific: a test-genie suite is awaited via
// `test-genie runs wait`, a git-control-tower baseline diff via the gct CLI's
// blocking diff. Each producer contributes a Waiter; the await-handle registry
// (await_registry.go) dispatches a parked run's handle to the Waiter matching
// its Producer and wakes the run with whatever the Waiter returns.
//
// The Waiter is the single seam producers extend: adding a new parkable async
// source is a one-Waiter change (plus the producer-side park trigger in
// Phase 5). Production Waiters shell the producer's own blocking-wait CLI
// through the CommandRunner seam so they stay testable without real binaries;
// because agent-manager's own process is NOT an agent-controlled context, those
// commands block normally (server-owned) rather than re-parking.

// Producer keys identify which Waiter resolves an await-handle. They are the
// stable values written to AwaitHandle.Producer by the producer-side park
// trigger and matched here.
const (
	ProducerTestGenie   = "test-genie"
	ProducerGCT         = "git-control-tower"
	ProducerLifecycle   = "lifecycle"
	ProducerSupervision = "supervision"
)

// Waiter blocks until a producer's externally-owned async work, identified by a
// producer-scoped key, reaches a terminal state — returning a human-readable
// result string to inject into the woken turn.
//
// Contract:
//   - Wait MUST honour ctx cancellation/deadline and return promptly when the
//     context is done (the registry uses ctx both for the park deadline and to
//     abort the wait when the run is stopped/woken externally).
//   - On producer-side failure Wait returns a non-nil error; the registry wakes
//     the run with the error framed as the result (the agent decides what to do).
//   - The returned string is surfaced verbatim to the agent, so it should be the
//     producer's result payload (e.g. the `--json` output of the wait command).
type Waiter interface {
	// Producer returns the stable key matched against AwaitHandle.Producer.
	Producer() string
	// Wait blocks until the work identified by key resolves or ctx is done.
	Wait(ctx context.Context, key string) (string, error)
}

type cohortWatchWaiter interface {
	WaitTerminal(context.Context, string) (*domainpb.CohortWatch, error)
}

type supervisionWaiter struct{ watches cohortWatchWaiter }

func NewSupervisionWaiter(watches cohortWatchWaiter) Waiter {
	return &supervisionWaiter{watches: watches}
}

func (w *supervisionWaiter) Producer() string { return ProducerSupervision }

func (w *supervisionWaiter) Wait(ctx context.Context, key string) (string, error) {
	if w.watches == nil {
		return "", fmt.Errorf("supervision watch service is unavailable")
	}
	watch, err := w.watches.WaitTerminal(ctx, strings.TrimSpace(key))
	if err != nil {
		return "", err
	}
	evidence := []string{}
	classification := ""
	if decision := watch.GetLastDecision(); decision != nil {
		evidence = append(evidence, decision.GetEvidenceIds()...)
		if len(evidence) > 20 {
			evidence = evidence[:20]
		}
		classification = decision.GetClassification()
	}
	payload, err := json.Marshal(map[string]any{
		"kind": "cohort_supervision_result", "watch_id": watch.GetWatchId(),
		"status": watch.GetStatus().String(), "revision": watch.GetRevision(),
		"classification": classification, "evidence_ids": evidence,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// CommandRunner executes an external CLI command and returns its combined
// stdout+stderr. It is a seam so production Waiters can shell real producer
// CLIs while tests inject a fake that returns canned output without spawning
// processes. Mirrors the shelling discipline of CommandWorkspaceSandboxEnsurer
// (delegate to the producer's own command; never re-implement its wait).
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// execCommandRunner is the production CommandRunner backed by os/exec. The
// context bounds the wait (park deadline / cancellation) — exec.CommandContext
// kills the subprocess when ctx is done.
type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = []string(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.ForeignScenario, nil))
	return cmd.CombinedOutput()
}

// splitProducerKey parses an await-handle key of the form "<scenario>/<id>"
// into its two parts. Both producers below address work by (scenario, work-id),
// so they share this encoding; the producer-side park trigger (Phase 5) builds
// the key the same way.
func splitProducerKey(key string) (scenario, id string, err error) {
	key = strings.TrimSpace(key)
	scenario, id, ok := strings.Cut(key, "/")
	scenario = strings.TrimSpace(scenario)
	id = strings.TrimSpace(id)
	if !ok || scenario == "" || id == "" {
		return "", "", domain.NewValidationError("key",
			fmt.Sprintf("await key %q must be \"<scenario>/<id>\"", key))
	}
	return scenario, id, nil
}

// testGenieWaiter awaits a test-genie run by shelling the blocking
// `test-genie runs wait --json <scenario> <run-id>` verb. That command is
// server-owned and blocks until the run is terminal; run from agent-manager's
// (non-agent) process it waits normally rather than re-parking.
type testGenieWaiter struct{ runner CommandRunner }

// NewTestGenieWaiter builds the production test-genie Waiter. A nil runner uses
// the os/exec-backed runner.
func NewTestGenieWaiter(runner CommandRunner) Waiter {
	if runner == nil {
		runner = execCommandRunner{}
	}
	return &testGenieWaiter{runner: runner}
}

func (w *testGenieWaiter) Producer() string { return ProducerTestGenie }

func (w *testGenieWaiter) Wait(ctx context.Context, key string) (string, error) {
	scenario, runID, err := splitProducerKey(key)
	if err != nil {
		return "", err
	}
	out, err := w.runner.Run(ctx, "test-genie", "runs", "wait", "--json", scenario, runID)
	if err != nil {
		// A failing suite exits 1 (exitRegression) but the wait itself succeeded —
		// the `--json` snapshot IS the result the agent wants. Treat that as a
		// resolved wait (the agent reasons about the verdict on wake); only a
		// genuine wait failure (RPC error, timeout) is surfaced as an error.
		if isProducerVerdictExit(err, 1) {
			return string(out), nil
		}
		return string(out), fmt.Errorf("test-genie runs wait %s/%s: %w: %s",
			scenario, runID, err, trimCommandOutput(string(out)))
	}
	return string(out), nil
}

// gctBaselineWaiter awaits a git-control-tower baseline diff by shelling the
// blocking `git-control-tower baseline diff --scenario <s> --name <n> --wait
// --json` verb. `--wait` is what makes the diff resolve the verdict server-side
// (GetDiffResult(Wait=true)) rather than returning a fast start-banner; without
// it the command coalesces onto the in-flight run and returns immediately, which
// would wake the agent with a banner instead of the verdict.
type gctBaselineWaiter struct{ runner CommandRunner }

// NewGCTBaselineWaiter builds the production git-control-tower Waiter. A nil
// runner uses the os/exec-backed runner.
func NewGCTBaselineWaiter(runner CommandRunner) Waiter {
	if runner == nil {
		runner = execCommandRunner{}
	}
	return &gctBaselineWaiter{runner: runner}
}

func (w *gctBaselineWaiter) Producer() string { return ProducerGCT }

func (w *gctBaselineWaiter) Wait(ctx context.Context, key string) (string, error) {
	scenario, name, err := splitProducerKey(key)
	if err != nil {
		return "", err
	}
	out, err := w.runner.Run(ctx, "git-control-tower", "baseline", "diff",
		"--scenario", scenario, "--name", name, "--wait", "--json")
	if err != nil {
		// `baseline diff` exits 1 (regression) / 2 (not-comparable) by verdict —
		// the diff resolved, this IS the result. Only a genuine wait failure (run
		// busy, test-genie unreachable, still in-flight at deadline) is an error.
		if isProducerVerdictExit(err, 1, 2) {
			return string(out), nil
		}
		return string(out), fmt.Errorf("git-control-tower baseline diff %s/%s: %w: %s",
			scenario, name, err, trimCommandOutput(string(out)))
	}
	return string(out), nil
}

// lifecycleWaiter awaits a scenario start/restart by shelling the blocking
// `vrooli scenario wait --json <scenario>` verb. The wait attaches to the
// in-flight start operation (or evaluates current health when none) and
// returns exactly one JSON verdict document; run from agent-manager's
// (non-agent) process it blocks normally rather than re-parking.
type lifecycleWaiter struct{ runner CommandRunner }

// NewLifecycleWaiter builds the production scenario-lifecycle Waiter. A nil
// runner uses the os/exec-backed runner.
func NewLifecycleWaiter(runner CommandRunner) Waiter {
	if runner == nil {
		runner = execCommandRunner{}
	}
	return &lifecycleWaiter{runner: runner}
}

func (w *lifecycleWaiter) Producer() string { return ProducerLifecycle }

func (w *lifecycleWaiter) Wait(ctx context.Context, key string) (string, error) {
	scenario, variant, err := splitProducerKey(key)
	if err != nil {
		return "", err
	}
	args := []string{"scenario", "wait", scenario, "--json"}
	if variant != "" && variant != "live" {
		args = append(args, "--instance", variant)
	}
	out, err := w.runner.Run(ctx, "vrooli", args...)
	if err != nil {
		// The wait verb exits with the VERDICT: 1 failed/not-running, 2
		// degraded, 124 timeout ceiling. Those are resolved waits whose JSON
		// document is the result the agent wants; only a genuine wait failure
		// (CLI missing, registry error) is surfaced as an error.
		if isProducerVerdictExit(err, 1, 2, 124) {
			return string(out), nil
		}
		return string(out), fmt.Errorf("vrooli scenario wait %s: %w: %s",
			key, err, trimCommandOutput(string(out)))
	}
	return string(out), nil
}

// isProducerVerdictExit reports whether err is a producer process that exited
// with one of the given verdict codes — i.e. the wait completed and the non-zero
// exit encodes a (legitimate) verdict the agent should see, not a wait failure.
// Satisfied by *exec.ExitError in production and by any error exposing
// ExitCode() int in tests.
func isProducerVerdictExit(err error, codes ...int) bool {
	var ec interface{ ExitCode() int }
	if !errors.As(err, &ec) {
		return false
	}
	got := ec.ExitCode()
	for _, c := range codes {
		if got == c {
			return true
		}
	}
	return false
}
