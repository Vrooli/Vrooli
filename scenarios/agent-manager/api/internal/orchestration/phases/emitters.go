// Event emission helpers used by every phase.
//
// All phase events flow through these helpers, which in turn route through
// the per-run emit.Gate when one is wired into Deps. This is the single
// choke point for run-event emission: future invariants (dedupe, audit
// hooks, ordering) attach inside the Gate so phases never need to know.
//
// The helpers also fall back to writing directly to the event store when no
// gate is wired (e.g. unit tests that drive a phase function in isolation).
// This keeps the test surface honest: phase functions don't silently no-op
// on missing gates — events still land in the store the test inspects.

package phases

import (
	"context"
	"errors"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// EmitSystemEvent records a system-level log/status event for the run.
// Level is one of "info", "warn", "error".
func EmitSystemEvent(ctx context.Context, deps Deps, runID uuid.UUID, level, message string) {
	evt := domain.NewLogEvent(runID, level, message)
	publishEvent(ctx, deps, runID, evt)
}

// EmitFailureEvent captures a typed domain error as a structured run event.
func EmitFailureEvent(ctx context.Context, deps Deps, runID uuid.UUID, err domain.DomainError) {
	evt := domain.NewErrorEventFromDomainError(runID, err)
	publishEvent(ctx, deps, runID, evt)
}

// EmitGenericFailureEvent captures a non-domain error as a run event.
//
// The error chain is inspected for known typed sentinels so the run
// timeline carries a useful retryable code instead of a blanket INTERNAL
// — the difference between "the agent CLI couldn't be reached because the
// home overlay didn't mount" (retryable) and "something failed".
func EmitGenericFailureEvent(ctx context.Context, deps Deps, runID uuid.UUID, err error) {
	code := domain.ErrCodeInternal
	retryable := false
	if coded := errorCode(err); coded != "" {
		code = domain.ErrorCode(coded)
		if code == domain.ErrCodeSandboxHomeOverlayUnavailable {
			retryable = true
		}
	}
	evt := domain.NewErrorEvent(runID, string(code), err.Error(), retryable)
	publishEvent(ctx, deps, runID, evt)
}

// publishEvent routes a constructed event through the Gate (if wired) or
// falls back to the event store. It never panics on a nil-shaped Deps so
// phase functions can be invoked from tests with partial wiring.
func publishEvent(ctx context.Context, deps Deps, runID uuid.UUID, evt *domain.RunEvent) {
	if deps.Gate != nil {
		_ = deps.Gate.Emit(evt)
		return
	}
	if deps.Events != nil {
		if err := deps.Events.Append(ctx, runID, evt); err != nil {
			return
		}
	} else {
		return
	}
	if deps.Broadcaster != nil {
		deps.Broadcaster.BroadcastEvent(evt)
	}
}

// errorCode extracts a stable string error code from err if any error in
// its chain implements `Code() string`. Returns "" otherwise.
func errorCode(err error) string {
	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
		if c, ok := cur.(interface{ Code() string }); ok {
			if code := c.Code(); code != "" {
				return code
			}
		}
	}
	return ""
}
