package toolfacts

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"strings"

	"github.com/google/uuid"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// ParserResolver returns a fresh transcript parser for a harness: the runner
// type name Agent Manager stores for a run ("codex", "claude-code", "opencode",
// ...). It is the seam to the one transcript parser per harness.
type ParserResolver func(harness string) (runner.TranscriptParser, error)

// RegistryParsers resolves parsers from a runner registry: any registered
// runner that implements runner.TranscriptParserFactory, which is every harness
// the importer and the Transcript Indexer can read.
func RegistryParsers(registry runner.Registry) ParserResolver {
	return func(harness string) (runner.TranscriptParser, error) {
		owned, err := registry.Get(domain.RunnerType(harness))
		if err != nil {
			return nil, fmt.Errorf("no runner for harness %q: %w", harness, err)
		}
		factory, ok := owned.(runner.TranscriptParserFactory)
		if !ok {
			return nil, fmt.Errorf("runner %q has no transcript parser", harness)
		}
		parser := factory.NewTranscriptParser()
		// Imported evidence reads typed user turns too; the tool events do not
		// depend on it, but a parser configured differently for the same file
		// would give different events than the importer's.
		if setter, ok := parser.(runner.TranscriptRetentionSetter); ok {
			setter.SetTranscriptRetention(true)
		}
		return parser, nil
	}
}

// FromTranscript reads a transcript file with the harness's own parser and
// returns the tool facts of the whole file. The file is streamed line by line.
func FromTranscript(ctx context.Context, rules Rules, path string, parser runner.TranscriptParser) (Facts, error) {
	file, err := os.Open(path)
	if err != nil {
		return Facts{}, err
	}
	defer file.Close()
	return ReadTranscript(ctx, rules, file, parser)
}

// ReadTranscript is FromTranscript over any reader.
func ReadTranscript(ctx context.Context, rules Rules, source io.Reader, parser runner.TranscriptParser) (Facts, error) {
	events := transcriptEvents(ctx, rules, source, parser)
	extractor := New(rules)
	var err error
	for event, eventErr := range events.Events {
		if eventErr != nil {
			err = eventErr
			break
		}
		extractor.Add(event)
	}
	facts := extractor.Facts()
	facts.Source = SourceTranscript
	facts.ParseErrors, facts.SkippedLines = events.Diagnostics()
	return facts, err
}

// eventStream is the tool events of one run, oldest first, as the invocation
// readers consume them: *domain.RunEvent with a *domain.ToolCallEventData or
// *domain.ToolResultEventData. Events is single use. Diagnostics reports the
// transcript lines the parser refused and the over-long lines skipped; it is
// final once Events has been ranged to its end.
type eventStream struct {
	Source      string
	Events      iter.Seq2[*domain.RunEvent, error]
	Diagnostics func() (parseErrors, skippedLines int)
}

// transcriptEvents streams the tool events of a transcript, parsed line by line
// with the harness's own parser. Memory is bounded by the longest line allowed.
// Each event gets a fresh id (the runs' readers key events by id), its 1-based
// position as Sequence, and the line's timestamp.
func transcriptEvents(ctx context.Context, rules Rules, source io.Reader, parser runner.TranscriptParser) eventStream {
	var parseErrors, skipped int
	es := eventStream{Source: SourceTranscript, Diagnostics: func() (int, int) { return parseErrors, skipped }}
	es.Events = func(yield func(*domain.RunEvent, error) bool) {
		runID := uuid.New()
		var lines int
		var sequence int64
		stop := errors.New("stopped")
		skippedLines, err := forEachLine(source, rules.MaxLineBytes, func(line []byte) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			lines++
			text := string(line)
			if strings.TrimSpace(text) == "" {
				return nil
			}
			result := parser.ParseTranscriptLine(runID, text)
			if result.Err != nil {
				parseErrors++
				return nil
			}
			for _, event := range result.Events {
				if event == nil || (event.EventType != domain.EventTypeToolCall && event.EventType != domain.EventTypeToolResult) {
					continue
				}
				sequence++
				event.Sequence = sequence
				if event.ID == uuid.Nil {
					event.ID = uuid.New()
				}
				if event.Timestamp.IsZero() {
					event.Timestamp = result.Timestamp
				}
				if !yield(event, nil) {
					return stop
				}
			}
			return nil
		})
		skipped = skippedLines
		if err != nil && !errors.Is(err, stop) {
			yield(nil, err)
		}
	}
	return es
}

// forEachLine calls fn for each line of at most max bytes and returns how many
// longer lines it skipped. Memory is bounded by max.
func forEachLine(source io.Reader, max int, fn func(line []byte) error) (int, error) {
	reader := bufio.NewReaderSize(source, 256*1024)
	var long []byte
	overflow, skipped := false, 0
	for {
		chunk, err := reader.ReadSlice('\n')
		switch {
		case err == nil || errors.Is(err, io.EOF):
			if overflow || len(long)+len(chunk) > max {
				if len(chunk) > 0 || overflow {
					skipped++
				}
			} else {
				line := chunk
				if len(long) > 0 {
					line = append(long, chunk...)
				}
				if len(line) > 0 {
					if cbErr := fn(line); cbErr != nil {
						return skipped, cbErr
					}
				}
			}
			long, overflow = long[:0], false
			if err != nil {
				return skipped, nil
			}
		case errors.Is(err, bufio.ErrBufferFull):
			if !overflow {
				if len(long)+len(chunk) > max {
					overflow = true
					long = long[:0]
				} else {
					long = append(long, chunk...)
				}
			}
		default:
			return skipped, err
		}
	}
}
