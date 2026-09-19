// Silent-launch failure detection for sandboxed runs.
//
// When a protected-sandbox run returns Success=true after sub-2s with zero
// assistant message events, the runner almost certainly never started — the
// classic shape is bwrap chdir failing before claude launches. Demote it
// to a structured failure so it does NOT auto-accept.
//
// The heuristic is intentionally narrow:
//   - mode must be PROTECTED (in-place runs don't traverse bwrap)
//   - duration must be < Diagnostics.LaunchFailedMaxDuration
//   - zero RUN_EVENT_TYPE_MESSAGE events
//
// DOC: see scenarios/agent-manager/docs/internal/ERROR-SEMANTICS.md
// (LAUNCH_FAILED row).

package phases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ValidateOutcomeInput is the explicit input to ValidateRunOutcome.
type ValidateOutcomeInput struct {
	Deps    Deps
	Run     *domain.Run
	Result  *runner.ExecuteResult
	ExecErr error
}

// ValidateOutcomeOutput is the explicit output. Result and ExecErr may have
// been rewritten — callers must use the returned values, not the input pointers.
type ValidateOutcomeOutput struct {
	Result  *runner.ExecuteResult
	ExecErr error
}

// ValidateRunOutcome inspects the post-Execute state and rewrites the result
// + execErr when the run shape indicates a silent launch failure. On entry,
// in.Result/in.ExecErr are the runner's verdict; on return, out carries the
// possibly-demoted versions ready for classifyOutcome.
//
// No-ops in three cases:
//   - Result is nil / Success=false (failures already route to handleFailure)
//   - Run mode is not protected sandboxed
//   - Duration ≥ LaunchFailedMaxDuration OR there are RUN_EVENT_TYPE_MESSAGE events
func ValidateRunOutcome(ctx context.Context, in ValidateOutcomeInput) ValidateOutcomeOutput {
	out := ValidateOutcomeOutput{Result: in.Result, ExecErr: in.ExecErr}
	if in.Result == nil || !in.Result.Success {
		return out
	}
	if !isProtectedSandboxRun(in.Run) {
		return out
	}
	if in.Result.Duration >= in.Deps.Levers.Diagnostics.LaunchFailedMaxDuration {
		return out
	}
	msgCount, err := countMessageEvents(ctx, in.Deps.Events, in.Run.ID)
	if err != nil {
		// Don't demote on a bookkeeping failure — better to let a real
		// run through than to false-positive on event-store hiccups.
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
			"validateRunOutcome: count message events: "+err.Error())
		return out
	}
	if msgCount > 0 {
		return out
	}

	stderrSnippet := truncateForLogPayload(in.Result.ErrorMessage, 4096)
	EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn", fmt.Sprintf(
		"silent launch failure detected (mode=protected, duration=%s, message_events=0); demoting to FAILED — inspect workspace-sandbox logs for bwrap stderr",
		in.Result.Duration.Truncate(time.Millisecond),
	))

	msg := stderrSnippet
	if strings.TrimSpace(msg) == "" {
		msg = "sandbox-routed run produced no output before exit; likely bwrap launch failure (see workspace-sandbox process logs)"
	}
	in.Result.Success = false
	if in.Result.ExitCode == 0 {
		in.Result.ExitCode = -1
	}
	in.Result.ErrorMessage = msg
	out.Result = in.Result
	out.ExecErr = domain.NewSandboxLaunchFailedError(msg)
	return out
}

// isProtectedSandboxRun returns true when the resolved config picks
// SandboxModeProtected. Other modes (in-place, tracking) don't traverse
// bwrap so they cannot exhibit the launch-failure shape.
func isProtectedSandboxRun(run *domain.Run) bool {
	if run == nil || run.RunMode != domain.RunModeSandboxed {
		return false
	}
	cfg := run.ResolvedConfig
	if cfg == nil || cfg.SandboxConfig == nil {
		return false
	}
	return cfg.SandboxConfig.Mode.Effective() == domain.SandboxModeProtected
}

// countMessageEvents returns how many RUN_EVENT_TYPE_MESSAGE events were
// emitted for this run so far.
func countMessageEvents(ctx context.Context, store event.Store, runID uuid.UUID) (int, error) {
	if store == nil {
		return 0, nil
	}
	got, err := store.Get(ctx, runID, event.GetOptions{
		EventTypes: []domain.RunEventType{domain.EventTypeMessage},
		Limit:      1,
	})
	if err != nil {
		return 0, err
	}
	return len(got), nil
}

// truncateForLogPayload bounds a string for inclusion in run-event payloads.
func truncateForLogPayload(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n…[truncated]"
}
