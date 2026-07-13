package interactive

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// defaultTailPoll is the re-open cadence for the interactive tail loop when no
// override is supplied.
const defaultTailPoll = 250 * time.Millisecond

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
// through the codec's transcript parser (design §1/§3). It is the sibling of
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
	// TranscriptPath is the agent-owned path the substrate resolved
	// (LaunchResult.TranscriptPath). For codex it is the seed for
	// rotation-aware re-discovery; for claude/grok it is pinned.
	TranscriptPath string
	// RunDir / WorkingDir / LaunchedAt mirror the substrate's DiscoverParams,
	// used only to re-glob the newest codex rollout on rotation.
	RunDir     string
	WorkingDir string
	LaunchedAt time.Time
	// StartCursor resumes a tail from a persisted byte offset (0 = start).
	StartCursor int64

	Sink        runner.EventSink
	OnAdvance   func(cursor, lastSeq int64) error
	OnSessionID func(sessionID string) error
}

// Tail follows the transcript, emitting codec events to the sink until the
// terminal marker appears (returned for Phase 4 to consume) or ctx is
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

	path := p.TranscriptPath
	cursor := p.StartCursor
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
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
			path = newest
			cursor = 0
		}

		nextCursor, terminal, cerr := runner.Consume(ctx, runner.ConsumeArgs{
			RunID:      p.RunID,
			Transcript: path,
			StartAt:    cursor,
			Live:       false,
			ParseFn:    parser.ParseTranscriptLine,
			EventSink:  p.Sink,
			OnAdvance:  p.OnAdvance,
			OnSessionID: func(sessionID string) error {
				if p.OnSessionID == nil {
					return nil
				}
				return p.OnSessionID(sessionID)
			},
		})
		cursor = nextCursor
		if cerr != nil {
			if ctx.Err() != nil {
				return terminal, ctx.Err()
			}
			if errors.Is(cerr, fs.ErrNotExist) {
				// The file vanished (rotation race) or was never created —
				// drop back to re-discovery.
				path = ""
				if !sleepCtx(ctx, t.pollInterval) {
					return terminal, ctx.Err()
				}
				continue
			}
			return terminal, cerr
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
			path = newest
			cursor = 0
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
	return findTranscript(DiscoverParams{
		RunnerType: p.RunnerType,
		WorkingDir: p.WorkingDir,
		RunDir:     p.RunDir,
		LaunchedAt: p.LaunchedAt,
		HomeDir:    t.homeDir,
	})
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
