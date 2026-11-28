package toolfacts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
	"agent-manager/internal/sqlcompat"
)

// Reader is the one source of a stored run's tool events and tool facts,
// wherever the run's tool activity was recorded: its run_events rows when it
// has tool calls there, else the transcript file the run names. An imported
// session whose recall layer was never retained, or was evicted for the
// Transcript Indexer's byte budget, has only its transcript; reading it lazily
// costs a read per annotation and no database growth.
type Reader struct {
	DB      sqlcompat.DB
	Parsers ParserResolver
	Rules   Rules
}

// evidence is where a run's tool activity was found: its tool call count in
// run_events, and the transcript path and harness to read when there are none.
type evidence struct {
	calls          int
	path, harness  string
	transcriptOpen bool
}

func (r *Reader) locate(ctx context.Context, runID string) (evidence, error) {
	var found evidence
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM run_events WHERE run_id = ? AND event_type = ?`,
		runID, string(domain.EventTypeToolCall)).Scan(&found.calls); err != nil {
		return found, fmt.Errorf("count tool calls: %w", err)
	}
	if found.calls > 0 {
		return found, nil
	}
	var path, harness sql.NullString
	err := r.DB.QueryRowContext(ctx, `
SELECT transcript_path,
       COALESCE(NULLIF(import_source_harness, ''), CASE WHEN json_valid(resolved_config) THEN json_extract(resolved_config, '$.runnerType') END)
FROM runs WHERE id = ?`, runID).Scan(&path, &harness)
	if errors.Is(err, sql.ErrNoRows) {
		return found, nil
	}
	if err != nil {
		return found, fmt.Errorf("read run transcript: %w", err)
	}
	found.path, found.harness = strings.TrimSpace(path.String), strings.TrimSpace(harness.String)
	if found.path == "" || found.harness == "" || r.Parsers == nil {
		return found, nil
	}
	if _, err := os.Stat(found.path); err != nil {
		return found, nil
	}
	found.transcriptOpen = true
	return found, nil
}

// events streams the whole run's tool events, oldest first. A run with no tool
// events and no readable transcript yields Source none and no events: that is
// a fact about the evidence, not a failure.
func (r *Reader) events(ctx context.Context, runID string) (eventStream, error) {
	found, err := r.locate(ctx, runID)
	if err != nil {
		return eventStream{}, err
	}
	switch {
	case found.calls > 0:
		return r.storedEvents(ctx, runID), nil
	case found.transcriptOpen:
		return r.transcriptStream(ctx, found)
	default:
		return noEvents(), nil
	}
}

func (r *Reader) transcriptStream(ctx context.Context, found evidence) (eventStream, error) {
	parser, err := r.Parsers(found.harness)
	if err != nil || parser == nil {
		// A harness with no transcript parser has nothing observable.
		return noEvents(), nil
	}
	file, err := os.Open(found.path)
	if err != nil {
		return noEvents(), nil
	}
	es := transcriptEvents(ctx, r.Rules, file, parser)
	inner := es.Events
	es.Events = func(yield func(*domain.RunEvent, error) bool) {
		defer file.Close()
		inner(yield)
	}
	return es, nil
}

// noEvents is the empty, diagnostic-free stream used when no tool evidence is
// observable. A missing transcript or an unsupported/vanished source says
// nothing about the run's tool activity.
func noEvents() eventStream {
	return eventStream{
		Source:      SourceNone,
		Events:      func(func(*domain.RunEvent, error) bool) {},
		Diagnostics: func() (int, int) { return 0, 0 },
	}
}

// storedEvents streams the run's tool rows in order, one row at a time.
func (r *Reader) storedEvents(ctx context.Context, runID string) eventStream {
	return eventStream{
		Source:      SourceRunEvents,
		Diagnostics: func() (int, int) { return 0, 0 },
		Events: func(yield func(*domain.RunEvent, error) bool) {
			rows, err := r.DB.QueryxContext(ctx, `
SELECT id, sequence, event_type, timestamp, data FROM run_events
WHERE run_id = ? AND event_type IN (?, ?) ORDER BY sequence`,
				runID, string(domain.EventTypeToolCall), string(domain.EventTypeToolResult))
			if err != nil {
				yield(nil, fmt.Errorf("read tool events: %w", err))
				return
			}
			defer rows.Close()
			for rows.Next() {
				event, err := scanEvent(rows)
				if err != nil {
					yield(nil, err)
					return
				}
				if event != nil && !yield(event, nil) {
					return
				}
			}
			if err := rows.Err(); err != nil {
				yield(nil, err)
			}
		},
	}
}

type eventRow interface{ Scan(dest ...any) error }

// scanEvent decodes one stored tool row; a row whose payload no longer decodes
// is skipped (nil event), as every reader of run_events does.
func scanEvent(rows eventRow) (*domain.RunEvent, error) {
	var (
		id, eventType, timestamp, data string
		sequence                       int64
	)
	if err := rows.Scan(&id, &sequence, &eventType, &timestamp, &data); err != nil {
		return nil, err
	}
	payload, err := domain.DecodeEventPayload(domain.RunEventType(eventType), []byte(data))
	if err != nil {
		return nil, nil
	}
	return &domain.RunEvent{
		ID: eventUUID(id), Sequence: sequence, EventType: domain.RunEventType(eventType),
		Timestamp: parseDBTime(timestamp), Data: payload,
	}, nil
}

// Facts returns the tool facts of the whole run, from the same event stream.
func (r *Reader) Facts(ctx context.Context, runID string) (Facts, error) {
	es, err := r.events(ctx, runID)
	if err != nil {
		return Facts{}, err
	}
	extractor := New(r.Rules)
	for event, err := range es.Events {
		if err != nil {
			return Facts{}, err
		}
		extractor.Add(event)
	}
	facts := extractor.Facts()
	facts.Source = es.Source
	facts.ParseErrors, facts.SkippedLines = es.Diagnostics()
	return facts, nil
}

// eventUUID is a stored event id as the invocation readers key it. Stored ids
// are UUIDs; a fixture's short id maps to a stable one so a call still pairs
// with its result.
func eventUUID(id string) uuid.UUID {
	if parsed, err := uuid.Parse(id); err == nil {
		return parsed
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(id))
}

func parseDBTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
		if at, err := time.Parse(layout, value); err == nil {
			return at.UTC()
		}
	}
	return time.Time{}
}
