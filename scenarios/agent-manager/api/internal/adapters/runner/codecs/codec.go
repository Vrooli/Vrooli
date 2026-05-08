// Package codecs defines the Codec seam: the runner-specific glue that
// translates a CLI agent's stdout JSON, CLI arguments, and environment
// shape into the shared event/result domain.
//
// The generic [agent-manager/internal/adapters/runner/core.Runner] owns
// process launching, stdout scanning, transcript writing, lifecycle
// tracking, and event emission. It defers to a Codec for everything that
// varies between agents (Claude Code, Codex, OpenCode, …): which CLI
// flags to build, how to interpret each stdout line, how to derive
// rolling metrics from events, and how to classify the final result.
//
// Adding a new agent means writing a new file in this package that
// implements [Codec] and registering it via the constructors in
// scenarios/agent-manager/api/main.go. No changes to core/ are required.
package codecs

import (
	"context"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"
)

// Codec is the runner-specific seam.
//
// Codec implementations MUST be safe for concurrent use across multiple
// runs — the generic core.Runner reuses one Codec instance for all calls.
// Per-run state lives behind [State] (created via [Codec.NewState] for
// every Execute/Continue call).
type Codec interface {
	// Static metadata -------------------------------------------------

	// Type returns the runner type identifier this codec handles.
	Type() domain.RunnerType

	// Capabilities reports what features the runner exposes.
	Capabilities() runner.Capabilities

	// BinaryPath returns the resolved CLI binary path, or "" when the
	// codec is not [Available].
	BinaryPath() string

	// BinaryDescription is a short human label used in error messages,
	// e.g. "claude CLI" or "resource-codex".
	BinaryDescription() string

	// TagEnvKey is the environment variable name that carries the
	// per-run tag the reconciler reads from /proc/<pid>/environ. The
	// generic launcher prepends "<key>=<tag>" to the env-wrapped exec
	// invocation so it appears in /proc/<pid>/cmdline.
	TagEnvKey() string

	// Available reports whether the codec is ready to run. Returns
	// (true, message) on success or (false, reason) otherwise. Codecs
	// SHOULD include any install hint in the failure message.
	Available(ctx context.Context) (bool, string)

	// ProbeModel performs a lightweight check that the given model id
	// is usable. Codecs that cannot cheaply tell SHOULD return nil.
	ProbeModel(ctx context.Context, modelID string) error

	// Per-call wiring -------------------------------------------------

	// BuildArgs constructs CLI arguments for an Execute call (excluding
	// the binary path itself; the launcher prepends that). Codecs MAY
	// stash request-derived per-run data on state here (e.g. Codex
	// captures req.GetConfig().Model so cost events emitted later can
	// label themselves correctly).
	BuildArgs(state State, req runner.ExecuteRequest) []string

	// BuildContinueArgs constructs CLI arguments for resuming an
	// existing session via Continue. Like BuildArgs, codecs MAY stash
	// request-derived per-run data on state here.
	BuildContinueArgs(state State, req runner.ContinueRequest) []string

	// BuildPrompt produces the bytes piped to the agent's stdin. An
	// empty string signals the codec does not pipe a prompt (e.g.
	// OpenCode passes the prompt as a CLI argument); the launcher then
	// closes stdin immediately.
	BuildPrompt(prompt string, attachments []runner.Attachment) string

	// BuildEnv produces the os.Environ()-shaped slice for the agent
	// process. The provided tag is the value the launcher should write
	// to TagEnvKey (the codec is responsible for adding both the tag
	// entry and any codec-specific env vars).
	BuildEnv(tag string, extras map[string]string) []string

	// ContinueTag returns the per-continuation tag for Continue calls.
	// The generic core distinguishes initial runs from continuations by
	// using a synthesized tag here (e.g. "claude-continue-<id>") so
	// /proc-based reconciliation can tell them apart.
	ContinueTag(req runner.ContinueRequest) string

	// Per-run state ---------------------------------------------------

	// NewState creates a fresh per-run state object (text buffers, tool
	// accumulators, captured session ID, etc.).
	NewState() State

	// DecodeStreamLine parses a single stdout line, mutating state as
	// needed. Returns zero or more events. A returned error is treated
	// as "log a warning and continue scanning" — runners do not abort
	// on per-line decode errors.
	DecodeStreamLine(state State, runID uuid.UUID, line string) ([]*domain.RunEvent, error)

	// ParseTranscriptLine parses one durable transcript line with a fresh
	// parser. Use [Codec.NewTranscriptParser] for any multi-line replay
	// so codec state is preserved across the transcript stream.
	ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult

	// NewTranscriptParser creates the stateful parser for one logical
	// durable transcript replay. Live tail plus final drain must share
	// this parser so result-line fallbacks can see content emitted by
	// earlier transcript lines.
	NewTranscriptParser() runner.TranscriptParser

	// UpdateMetrics applies an event's effect on rolling execution
	// metrics. Called once per event the codec emits.
	UpdateMetrics(event *domain.RunEvent, metrics *runner.ExecutionMetrics, lastAssistant *string)

	// OnEarlyTerminate is consulted on each non-empty stdout line
	// AFTER DecodeStreamLine has emitted its events. Codecs that
	// signal an early end through a sentinel line (e.g. OpenCode's
	// terminal step_finish) record "we just saw a terminator" on
	// state during DecodeStreamLine and return true here so the
	// scanner loop exits. Most codecs return false.
	OnEarlyTerminate(state State, line string) bool

	// Result classification ------------------------------------------

	// PostClassify lets the codec adjust the populated ExecuteResult
	// based on accumulated state. Common use: Claude Code surfaces
	// rate-limit conditions only after the full stream has been seen,
	// so the codec flips Success=false and ExitCode=429 here when
	// state holds a captured RateLimitEventData.
	PostClassify(state State, result *runner.ExecuteResult)

	// ClassifyTerminalError inspects a non-zero-exit run's accumulated
	// stderr and exit code and returns a typed *domain.RunnerError when
	// the codec recognises a known failure shape. Returning nil means
	// "no codec-specific classification — let core.Runner fall back to
	// the generic ErrCodeRunnerExecution path."
	//
	// Codecs typically distinguish:
	//   - session/thread genuinely gone   → NewRunnerSessionExpiredError
	//   - in-memory writer race mid-run   → NewRunnerSessionStateLostError
	// so operators see distinct ErrorCodes on the run timeline rather
	// than a single `INTERNAL` bucket.
	//
	// Implementations MUST be deterministic for the same (stderr,
	// exitCode) pair — the result feeds the run's typed-error event
	// and is therefore part of the public API surface.
	//
	// DOC: scenarios/agent-manager/docs/internal/SEAMS.md (Codec
	// Terminal-Error Classification).
	ClassifyTerminalError(stderr string, exitCode int) *domain.RunnerError

	// Classify converts a non-success Execute outcome into a typed
	// [*fallback.ClassifiedError]. Codecs should consult their own
	// structured signals first (HTTP status from streamed events, codec
	// error codes, captured rate-limit state surfaced via PostClassify
	// into the supplied stderr/ErrorMessage) and delegate to the
	// residual [fallback.TextClassifier] for the rest.
	//
	// Contract:
	//   - Returns nil ONLY when stderr is empty AND exitCode == 0 — i.e.
	//     the run succeeded and there is no signal to classify.
	//   - Returns a non-nil *ClassifiedError otherwise; ReasonUnknown is
	//     the explicit "I saw a failure but could not classify it"
	//     signal (raw text preserved on Message/Cause).
	//   - Implementations MUST be deterministic for the same (stderr,
	//     exitCode) pair.
	//
	// DOC: scenarios/agent-manager/docs/internal/EVENT_TAXONOMY.md
	// (model.fallback.attempted reason field).
	Classify(stderr string, exitCode int) *fallback.ClassifiedError

	// Labels supplies the human-readable status messages the runner
	// emits on Execute/Continue start and completion.
	Labels() Labels
}

// State is the opaque per-run state object owned by a Codec.
// Implementations are created via [Codec.NewState] and live for the
// duration of a single Execute or Continue call.
type State interface {
	// SessionID returns the captured session id, or "" when none has
	// been seen on the stream yet. The generic core reads this after
	// the scanner loop completes to populate ExecuteResult.SessionID.
	SessionID() string
}

// Labels carries the human-readable status messages a codec uses for
// Execute/Continue start and completion. Stored once per codec rather
// than re-derived per-call.
type Labels struct {
	StartMessage         string // emitted on Execute start
	EndMessage           string // emitted on Execute completion
	ContinueStartMessage string // emitted on Continue start
	ContinueEndMessage   string // emitted on Continue completion
}
