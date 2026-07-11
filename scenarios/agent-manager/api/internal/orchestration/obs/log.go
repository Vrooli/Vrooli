// Package obs is the agent-manager orchestration observability surface.
//
// All structured logging in the orchestration / runner / phases layers
// flows through this package. Two design rules make the surface stable:
//
//  1. **One choke point.** A single package-level [*slog.Logger] is
//     installed by [Init] at server startup; downstream code calls
//     [Logger] (or the context-scoped helper [L]) and never constructs
//     its own logger. Replacing the handler in one place changes the
//     entire layer's output format.
//
//  2. **Stable keys.** Every dynamic field is logged via the constants
//     declared in this file (KeyRunID, KeyPhase, …). New keys are added
//     here, not inlined at the call site, so the lever between
//     "loggable" and "noise" lives in one file. This is what lets a
//     log aggregator's queries survive a refactor.
//
// Lifecycle events on the run timeline use a parallel surface in
// [obs/events.go]; the two surfaces share keys (same name, same shape)
// so a log line and its corresponding lifecycle event correlate by
// runID + phase.
//
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md
// (the obs.Logger seam, decision-boundary table).
package obs

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
)

// =============================================================================
// STABLE KEYS
// =============================================================================
//
// These are the only field names the orchestration layer should attach
// to slog records. Adding a new key is a contract change — it requires a
// constant here and (typically) a column in the lifecycle event payload.

const (
	// Run identity.
	KeyRunID          = "runID"
	KeyTaskID         = "taskID"
	KeyConversationID = "conversationID"

	// Run shape.
	KeyRunMode      = "runMode"
	KeyRunnerType   = "runnerType"
	KeyLauncherType = "launcherType"
	KeySandboxID    = "sandboxID"

	// Lifecycle / phase.
	KeyPhase    = "phase"
	KeyDuration = "durationMs"

	// Outcomes.
	KeyExitCode     = "exitCode"
	KeyTerminalCode = "terminalCode"
	KeyOutcome      = "outcome"

	// Spawn dispatcher signals (Phase 3 wires producers).
	KeyQueueDepth    = "queueDepth"
	KeyActiveCount   = "activeCount"
	KeyStartingCount = "startingCount"

	// Generic observability.
	KeyError     = "error"
	KeyMessage   = "message"
	KeyComponent = "component"

	// Permission-policy control-plane evidence. These describe whole-document
	// lifecycle outcomes only; rule patterns and native config content are never
	// emitted through structured logs.
	KeyPermissionPolicyDigest           = "permissionPolicyDigest"
	KeyPermissionPolicyResourceCount    = "permissionPolicyResourceCount"
	KeyPermissionPolicyDriftCount       = "permissionPolicyDriftCount"
	KeyPermissionPolicyUnsupportedCount = "permissionPolicyUnsupportedCount"
	KeyHardEnforcementSatisfied         = "hardEnforcementSatisfied"
	KeyMissingHardEnforcementRuleIDs    = "missingHardEnforcementRuleIDs"
	KeyPermissionPolicyPartialFailure   = "permissionPolicyPartialFailure"
)

// =============================================================================
// LOGGER WIRING
// =============================================================================

// loggerSlot holds the package-level [*slog.Logger]. atomic.Pointer
// keeps reads cheap (no mutex) and lets [Init] safely swap the handler
// during a hot reload. Production code never observes a nil pointer:
// init() seeds a sane default before any caller can reach [Logger].
var loggerSlot atomic.Pointer[slog.Logger]

func init() {
	loggerSlot.Store(buildLogger("text", slog.LevelInfo, os.Stderr))
}

// Init installs the package-level logger from the supplied format and
// level strings. format must be "text" or "json"; level must be one of
// debug|info|warn|error. Invalid values fall back to ("text", "info")
// and are reported via the existing logger — Init never panics so a
// startup misconfig cannot kill the server.
func Init(format, level string) {
	loggerSlot.Store(buildLogger(format, parseLevel(level), os.Stderr))
}

// InitWithWriter is the test seam for [Init]; it directs output to an
// arbitrary writer (typically a [*bytes.Buffer]) so assertions can read
// the emitted log lines back.
func InitWithWriter(format, level string, w io.Writer) {
	loggerSlot.Store(buildLogger(format, parseLevel(level), w))
}

// Logger returns the package-level logger. Always non-nil.
func Logger() *slog.Logger {
	return loggerSlot.Load()
}

// loggerKey is the unexported type used to attach a per-run logger to
// a context. The unexported type guarantees no other package can
// collide on this key.
type loggerKey struct{}

// WithLogger returns a child context carrying the supplied logger. Use
// [L] to retrieve it.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey{}, l)
}

// L returns the run-scoped logger attached to ctx (via [RunCtx] or
// [WithLogger]); falls back to the package-level [Logger] when no
// scoped logger is present.
func L(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Logger()
	}
	if v := ctx.Value(loggerKey{}); v != nil {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return Logger()
}

// RunCtx returns a child context carrying a logger pre-tagged with
// run-scoped fields. The expectation is that the orchestration entry
// point calls this once per run and downstream code uses [L] to get
// the scoped logger — no caller needs to remember to thread the runID
// through every log site.
//
// Empty/zero values are skipped so a nil sandbox ID doesn't pollute
// every log line with `sandboxID=00000000-…`.
func RunCtx(ctx context.Context, runID uuid.UUID, runMode domain.RunMode, runnerType domain.RunnerType, sandboxID *uuid.UUID) context.Context {
	attrs := []any{KeyRunID, runID.String()}
	if runMode != "" {
		attrs = append(attrs, KeyRunMode, string(runMode))
	}
	if runnerType != "" {
		attrs = append(attrs, KeyRunnerType, string(runnerType))
	}
	if sandboxID != nil && *sandboxID != uuid.Nil {
		attrs = append(attrs, KeySandboxID, sandboxID.String())
	}
	return WithLogger(ctx, Logger().With(attrs...))
}

// Component returns a derived logger tagged with a component name. Use
// for long-running background workers (reconciler, recommendation
// worker, …) that don't have a runID.
func Component(name string) *slog.Logger {
	return Logger().With(KeyComponent, name)
}

// =============================================================================
// INTERNAL HELPERS
// =============================================================================

// parseLevel maps a string level to the slog enum. Unknown values fall
// back to Info — this keeps a misconfigured lever from disabling all
// observability.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// buildLogger constructs a [*slog.Logger] for the requested format and
// level. Centralised so tests share the exact production wiring.
func buildLogger(format string, level slog.Level, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts))
	default:
		return slog.New(slog.NewTextHandler(w, opts))
	}
}
