// This file tails interactive run output for connected clients.
package interactive

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// defaultTailPoll is the re-open cadence for the interactive tail loop when no
// override is supplied.
const defaultTailPoll = 250 * time.Millisecond

// defaultGoalDrainWindow matches the coordinator's existing interactive idle
// debounce. A native goal verdict is already final; this window only captures
// provider output emitted before the same turn closes.
const defaultGoalDrainWindow = 5 * time.Second

// ParserResolver resolves a fresh codec transcript parser for a runner type.
// A registry-backed resolver ([RegistryParser]) is used in production; tests
// supply a codec's real parser directly.
type ParserResolver func(domain.RunnerType) (runner.TranscriptParser, error)

// RegistryParser adapts a runner.Registry to a ParserResolver: it resolves the
// runner and asserts it exposes a fresh stateful transcript parser
// (runner.TranscriptParserFactory) — the same resolve-then-assert pattern
// crash recovery uses. A fresh parser per Tail call keeps replay state (grok
// text accumulation, claude on-disk latch, codex thread id) across poll cycles.
func RegistryParser(reg runner.Registry) ParserResolver {
	return func(rt domain.RunnerType) (runner.TranscriptParser, error) {
		if reg == nil {
			return nil, fmt.Errorf("runner registry is not configured")
		}
		r, err := reg.Get(rt)
		if err != nil {
			return nil, err
		}
		factory, ok := r.(runner.TranscriptParserFactory)
		if !ok {
			return nil, fmt.Errorf("runner %s does not expose a transcript parser", rt)
		}
		return factory.NewTranscriptParser(), nil
	}
}

// Tailer follows an interactive run's agent-owned transcript and emits the same
// structured events a codec pipe run produces, by feeding each transcript line
// through the codec's transcript parser (design §1/§3). An accepted native goal
// is reported immediately and followed by a bounded drain to the same provider
// turn boundary before the terminal is returned. It is the sibling of
// core.runDurable's live tail: it does not launch a process, scan stdout, or
// wait on process exit — it only tails the file the real CLI writes inside the
// web-console session.
type Tailer struct {
	resolveParser ParserResolver

	// homeDir overrides the user home for codex rotation re-discovery (tests).
	homeDir      string
	pollInterval time.Duration
}

// TailerOption configures a Tailer.
type TailerOption func(*Tailer)

// WithTailHomeDir overrides the home directory used for transcript
// re-discovery (tests).
func WithTailHomeDir(dir string) TailerOption { return func(t *Tailer) { t.homeDir = dir } }

// WithTailPollInterval sets the re-open cadence of the tail loop.
func WithTailPollInterval(d time.Duration) TailerOption {
	return func(t *Tailer) {
		if d > 0 {
			t.pollInterval = d
		}
	}
}

// NewTailer builds a Tailer from a parser resolver.
func NewTailer(resolveParser ParserResolver, opts ...TailerOption) *Tailer {
	t := &Tailer{
		resolveParser: resolveParser,
		pollInterval:  defaultTailPoll,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// TailParams is the input to Tail.
type TailParams struct {
	RunID      uuid.UUID
	RunnerType domain.RunnerType
	Model      string
	// TranscriptPath is the agent-owned path the substrate resolved
	// (LaunchResult.TranscriptPath). For codex it is the seed for
	// rotation-aware re-discovery; for claude/grok it is pinned.
	TranscriptPath string
	SessionID      string
	Until          string
	Billing        domain.BillingSnapshot
	// MaxTurns caps assistant turn boundaries in goal mode. Zero disables the
	// ceiling.
	MaxTurns int
	// GoalMarkerObserved reports whether a runner-native goal marker has been
	// seen for this run. After defaultNativeGoalFallbackTurns success turn
	// boundaries with no marker, the harness is treated as non-native and a
	// validated structured terminal is accepted. Nil means "never observed".
	GoalMarkerObserved func() bool
	// RunDir / WorkingDir / LaunchedAt mirror the substrate's DiscoverParams,
	// used only to re-glob the newest codex rollout on rotation.
	RunDir     string
	WorkingDir string
	LaunchedAt time.Time
	// StartCursor resumes a tail from a persisted byte offset (0 = start).
	StartCursor int64
	// GoalDrainTimeout bounds capture after an accepted native goal verdict.
	// The verdict remains authoritative if the provider turn does not close in
	// time. Zero uses the existing five-second interactive debounce.
	GoalDrainTimeout time.Duration

	Sink      runner.EventSink
	OnAdvance func(cursor, lastSeq int64) error
	// OnTranscriptPath commits a newly selected file before any bytes from that
	// file are consumed. The cursor is file-local, so callers that persist both
	// fields must reset it atomically when the path changes.
	OnTranscriptPath func(path string) error
	OnSessionID      func(sessionID string) error
	OnGoalStatus     func(marker runner.GoalMarker) error
	// StructuredResultSatisfied reports whether the current persisted transcript
	// satisfies the run's deterministic structured-result contract. It is
	// checked at a provider success boundary when Until is non-empty, because a
	// goal-session worker may finish with its structured result without emitting
	// a runner-native goal marker.
	StructuredResultSatisfied func() bool
	// CompleteAfterFirstSuccessfulTurnWithoutGoalMarker closes a continuation
	// after its first successful provider turn only when that new invocation
	// emitted no native goal marker. It is reserved for continuing an already
	// terminal native goal (complete or blocked) without reinstalling it.
	CompleteAfterFirstSuccessfulTurnWithoutGoalMarker bool
}

// Tail follows the transcript, emitting codec events to the sink until the
// terminal boundary appears (returned for Phase 4 to consume) or ctx is
// cancelled. It reuses runner.Consume(Live=false) in a re-open-from-cursor
// poll loop, which uniformly handles the three interactive-tail hazards:
//
//   - transcript appears late — the resolved path/file is absent, so the loop
//     waits and re-resolves rather than erroring;
//   - half-written trailing line — Consume returns at EOF without consuming the
//     fragment (cursor stays at the last complete line), and the next drain
//     re-reads it once the newline lands, so a partial JSON line is never
//     emitted;
//   - codex rollout rotation — between drains the loop re-globs the newest
//     rollout under the run-scoped CODEX_HOME; on a strictly-newer file it
//     drains the current file to EOF first, then switches with cursor=0.
//
// claude and grok write a single pinned transcript, so their re-discovery
// returns the same path and no rotation occurs.
func (t *Tailer) Tail(ctx context.Context, p TailParams) (*runner.TranscriptTerminal, error) {
	parser, err := t.resolveParser(p.RunnerType)
	if err != nil {
		return nil, fmt.Errorf("resolve transcript parser for %s: %w", p.RunnerType, err)
	}
	if setter, ok := parser.(runner.TranscriptModelSetter); ok {
		setter.SetTranscriptModel(p.Model)
	}
	if setter, ok := parser.(runner.TranscriptBillingSetter); ok {
		setter.SetTranscriptBilling(p.Billing)
	}
	// Each tail starts at a persisted cursor, usually past the session's first
	// line; usage reports still name the session whose counters they carry.
	if setter, ok := parser.(runner.TranscriptSessionSetter); ok {
		setter.SetTranscriptSession(p.SessionID, false)
	}
	if setter, ok := parser.(interface {
		SetGoalContext(uuid.UUID, string, string)
	}); ok {
		setter.SetGoalContext(p.RunID, p.SessionID, p.Until)
	}

	path := p.TranscriptPath
	cursor := p.StartCursor
	selectPath := func(next string) error {
		if next == "" || next == path {
			return nil
		}
		if p.OnTranscriptPath != nil {
			if err := p.OnTranscriptPath(next); err != nil {
				return fmt.Errorf("persist selected transcript path %q: %w", next, err)
			}
		}
		path = next
		cursor = 0
		return nil
	}
	turnBoundaries := 0
	var acceptedGoal *runner.TranscriptTerminal
	var lastGoalMarker *runner.GoalMarker
	var acceptedGoalMarker *runner.GoalMarker
	var goalDrainDeadline time.Time
	goalMarkerSeen := false
	goalDrainEnabled := strings.TrimSpace(p.Until) != ""
	providerBoundary := false
	userBoundary := false
	goalBoundary := false
	sessionChanged := false
	expectedSessionID := strings.TrimSpace(p.SessionID)
	goalDrainTimeout := p.GoalDrainTimeout
	if goalDrainTimeout <= 0 {
		goalDrainTimeout = defaultGoalDrainWindow
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if acceptedGoal != nil && !time.Now().Before(goalDrainDeadline) {
			return acceptedGoal, nil
		}

		if path == "" {
			newest, derr := t.rediscover(p)
			if derr != nil {
				return nil, derr
			}
			if newest == "" {
				if !sleepCtx(ctx, t.pollInterval) {
					return nil, ctx.Err()
				}
				continue
			}
			if err := selectPath(newest); err != nil {
				return nil, err
			}
		}

		parseLine := func(runID uuid.UUID, line string) runner.TranscriptParseResult {
			result := parser.ParseTranscriptLine(runID, line)
			if result.SessionID != "" {
				if expectedSessionID == "" {
					expectedSessionID = result.SessionID
				} else if result.SessionID != expectedSessionID && goalMarkerSeen {
					sessionChanged = true
				}
			}
			return result
		}
		nextCursor, terminal, cerr := runner.Consume(ctx, runner.ConsumeArgs{
			RunID:      p.RunID,
			Transcript: path,
			StartAt:    cursor,
			Live:       false,
			ParseFn:    parseLine,
			EventSink:  p.Sink,
			OnAdvance:  p.OnAdvance,
			OnSessionID: func(sessionID string) error {
				if p.OnSessionID == nil {
					return nil
				}
				return p.OnSessionID(sessionID)
			},
			OnGoalStatus: func(marker runner.GoalMarker) error {
				markerCopy := marker
				lastGoalMarker = &markerCopy
				if p.OnGoalStatus != nil {
					return p.OnGoalStatus(marker)
				}
				return nil
			},
			OnTerminal: func(terminal *runner.TranscriptTerminal) error {
				if goalDrainEnabled && acceptedGoal == nil && terminal != nil && terminal.Success &&
					(terminal.TerminalReason == "goal_complete" || terminal.TerminalReason == "goal_blocked") {
					// Preserve the canonical terminal constructed by runner.Consume.
					acceptedGoal = terminal
					acceptedGoalMarker = lastGoalMarker
					goalMarkerSeen = true
					goalDrainDeadline = time.Now().Add(goalDrainTimeout)
				}
				return nil
			},
			StopBeforeTranscriptResult: func(result runner.TranscriptParseResult) bool {
				if !goalMarkerSeen {
					return false
				}
				if result.Goal != nil && acceptedGoalMarker != nil {
					marker := result.Goal
					if marker.Status == runner.GoalStatusActive || marker.Status == runner.GoalStatusPaused ||
						marker.Objective != acceptedGoalMarker.Objective || marker.Status != acceptedGoalMarker.Status {
						goalBoundary = true
						return true
					}
				}
				if result.UserTurnStarted || result.ProviderTurnStarted {
					userBoundary = true
					return true
				}
				return sessionChanged
			},
			StopAfterTranscriptResult: func(result runner.TranscriptParseResult) bool {
				if goalMarkerSeen && result.Terminal != nil {
					providerBoundary = true
					return true
				}
				return false
			},
		})
		cursor = nextCursor
		if cerr != nil {
			if ctx.Err() != nil {
				return terminal, ctx.Err()
			}
			if errors.Is(cerr, fs.ErrNotExist) {
				// The file vanished (rotation race) or was never created —
				// re-discover without losing the offset if the same file returns.
				newest, derr := t.rediscover(p)
				if derr != nil {
					return nil, derr
				}
				if err := selectPath(newest); err != nil {
					return nil, err
				}
				if !sleepCtx(ctx, t.pollInterval) {
					return terminal, ctx.Err()
				}
				continue
			}
			return terminal, cerr
		}
		if acceptedGoal != nil {
			if providerBoundary || userBoundary || goalBoundary || sessionChanged {
				return acceptedGoal, nil
			}
			if !time.Now().Before(goalDrainDeadline) {
				return acceptedGoal, nil
			}
			// The native verdict is already accepted. Keep consuming only this
			// provider turn so a delayed final answer is persisted before the
			// coordinator finalizes and wakes its parent.
			if !sleepCtx(ctx, t.pollInterval) {
				return nil, ctx.Err()
			}
			continue
		}
		if terminal != nil {
			// In goal mode a provider success marker is only a turn boundary.
			// Keep following the same live transcript until a native goal
			// terminal or a fully validated structured result is observed.
			if terminal.Success && strings.TrimSpace(p.Until) != "" && !strings.HasPrefix(terminal.TerminalReason, "goal_") {
				// A validated deterministic structured result is terminal
				// immediately: the worker returned its workflow result contract
				// without a runner-specific goal marker.
				if p.StructuredResultSatisfied != nil && p.StructuredResultSatisfied() {
					terminal.TerminalReason = terminalReasonStructuredResult
					return terminal, nil
				}
				if p.CompleteAfterFirstSuccessfulTurnWithoutGoalMarker && lastGoalMarker == nil && (p.GoalMarkerObserved == nil || !p.GoalMarkerObserved()) {
					terminal.TerminalReason = "completed_goal_continuation"
					return terminal, nil
				}
				turnBoundaries++
				if p.MaxTurns > 0 && turnBoundaries >= p.MaxTurns {
					terminal.TerminalReason = terminalReasonMaxTurns
					return terminal, nil
				}
				// No goal marker was ever observed and the fallback window
				// elapsed: treat the harness as non-native and accept the turn
				// boundary as a run boundary.
				if (p.GoalMarkerObserved == nil || !p.GoalMarkerObserved()) && turnBoundaries >= defaultNativeGoalFallbackTurns {
					terminal.TerminalReason = "turn_boundary_idle"
					return terminal, nil
				}
				terminal = nil
			}
		}
		if terminal != nil {
			return terminal, nil
		}

		// Current file drained to EOF; check for a codex rollout rotation
		// before waiting, so a switch is followed immediately by a drain.
		newest, derr := t.rediscover(p)
		if derr != nil {
			return nil, derr
		}
		if newest != "" && newest != path {
			if err := selectPath(newest); err != nil {
				return nil, err
			}
			continue
		}

		if !sleepCtx(ctx, t.pollInterval) {
			return nil, ctx.Err()
		}
	}
}

// rediscover returns the current newest transcript path. codex rollout files
// rotate, so its path is re-globbed under the run-scoped CODEX_HOME; claude and
// grok write a single pinned file, so their seed path is returned unchanged
// (claude is deliberately pinned to avoid a concurrent-session race, design R3).
func (t *Tailer) rediscover(p TailParams) (string, error) {
	if p.RunnerType != domain.RunnerTypeCodex {
		return p.TranscriptPath, nil
	}
	if strings.TrimSpace(p.RunDir) == "" {
		return "", fmt.Errorf("codex transcript rediscovery requires a run-scoped run directory")
	}
	return findTranscript(DiscoverParams{
		RunnerType: p.RunnerType,
		WorkingDir: p.WorkingDir,
		RunDir:     p.RunDir,
		LaunchedAt: p.LaunchedAt,
		HomeDir:    t.homeDir,
		SessionID:  p.SessionID,
	})
}

// DiscoverCurrentTranscript returns the path the tailer would select now for
// this exact run. Recovery uses it before replaying retained goal evidence so
// an older persisted rollout cannot finalize a run after Codex has rotated to
// a newer rollout in the same run-scoped home.
func (t *Tailer) DiscoverCurrentTranscript(p TailParams) (string, error) {
	return t.rediscover(p)
}

// sleepCtx waits d or until ctx is done; returns false if ctx was cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
