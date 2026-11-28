package toolfacts_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/toolfacts"
)

// The harness captures committed beside the codecs are redacted copies of real
// provider transcripts; reading them with the real codecs proves each dialect
// reaches the same facts struct.
const captures = "../adapters/runner/codecs/testdata/"

func TestFactsFromRealHarnessCaptures(t *testing.T) {
	cases := []struct {
		name, file, harness string
		want                toolfacts.Facts
	}{
		{"claude code", "claude_trace.jsonl", "claude-code", toolfacts.Facts{ToolCalls: 1, EventsRead: 2}},
		{"codex rollout", "codex_rollout_trace.jsonl", "codex", toolfacts.Facts{ToolCalls: 2, EventsRead: 4}},
		{"codex code-mode with nested commands", "codex_command_executions.jsonl", "codex", toolfacts.Facts{
			ToolCalls: 3, Commands: 12, ToolFailures: 1, FailedCommands: 1, FailedTools: []string{"command_execution"},
			LastError: "command exited with code 1", EventsRead: 17}},
		{"codex exec polling loop", "corpus/poll-loop.jsonl", "codex", toolfacts.Facts{
			ToolCalls: 3, Commands: 3, RepeatedCommands: 2, UnchangedRepeats: 2, LongestUnchangedRun: 2, EventsRead: 6}},
		{"opencode", "opencode_trace.jsonl", "opencode", toolfacts.Facts{ToolCalls: 2, RepeatedPaths: 1, EventsRead: 3}},
	}
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := codecParsers(tc.harness)
			if err != nil {
				t.Fatal(err)
			}
			got, err := toolfacts.FromTranscript(context.Background(), rules, filepath.Join(captures, tc.file), parser)
			if err != nil {
				t.Fatal(err)
			}
			tc.want.Source = toolfacts.SourceTranscript
			if !sameFacts(got, tc.want) {
				t.Fatalf("facts = %+v\nwant   %+v", got, tc.want)
			}
		})
	}
}

func sameFacts(a, b toolfacts.Facts) bool {
	return a.Source == b.Source && a.ToolCalls == b.ToolCalls && a.Commands == b.Commands && a.ToolFailures == b.ToolFailures &&
		a.FailedCommands == b.FailedCommands && a.RepeatedPaths == b.RepeatedPaths && a.RepeatedCommands == b.RepeatedCommands &&
		a.UnchangedRepeats == b.UnchangedRepeats && a.LongestUnchangedRun == b.LongestUnchangedRun &&
		strings.Join(a.FailedTools, ",") == strings.Join(b.FailedTools, ",") && a.LastError == b.LastError &&
		a.EventsRead == b.EventsRead && a.ParseErrors == b.ParseErrors && a.SkippedLines == b.SkippedLines
}

func TestTranscriptLinesAreBoundedNotTrusted(t *testing.T) {
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	rules.MaxLineBytes = 4096
	body, err := os.ReadFile(filepath.Join(captures, "corpus/poll-loop.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	// A line longer than the bound (and than the reader's buffer), a line that
	// is not JSON, a blank line, and no newline at the end.
	huge := `{"type":"x","pad":"` + strings.Repeat("a", 600*1024) + `"}`
	text := string(body) + huge + "\nnot json at all\n\n" + strings.TrimSpace(string(body))
	parser, _ := codecParsers("codex")
	facts, err := toolfacts.ReadTranscript(context.Background(), rules, strings.NewReader(text), parser)
	if err != nil {
		t.Fatal(err)
	}
	if facts.SkippedLines != 1 {
		t.Fatalf("skipped lines = %d, want 1 (the oversized line)", facts.SkippedLines)
	}
	// The poll loop twice: the unterminated last line still counts.
	if facts.ToolCalls != 6 {
		t.Fatalf("calls = %d, want 6", facts.ToolCalls)
	}
}

func TestTranscriptReadStopsWhenCancelled(t *testing.T) {
	rules, _ := toolfacts.DefaultRules()
	parser, _ := codecParsers("codex")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	text := strings.Repeat(`{"type":"turn.started"}`+"\n", 5000)
	if _, err := toolfacts.ReadTranscript(ctx, rules, strings.NewReader(text), parser); err == nil {
		t.Fatal("a cancelled read returned no error")
	}
}

type countingTranscriptParser struct {
	runner.TranscriptParser
	calls int
}

func (p *countingTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	p.calls++
	return p.TranscriptParser.ParseTranscriptLine(runID, line)
}

func TestTranscriptReadDoesNotParseWhenAlreadyCancelled(t *testing.T) {
	rules, _ := toolfacts.DefaultRules()
	parser, err := codecParsers("codex")
	if err != nil {
		t.Fatal(err)
	}
	counted := &countingTranscriptParser{TranscriptParser: parser}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = toolfacts.ReadTranscript(ctx, rules, strings.NewReader(`{"type":"turn.started"}`+"\n"), counted)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("read error = %v, want context cancellation", err)
	}
	if counted.calls != 0 {
		t.Fatalf("parser calls = %d, want 0 after cancellation", counted.calls)
	}
}

func TestTranscriptReadStopsAfterCancellationDuringRead(t *testing.T) {
	rules, _ := toolfacts.DefaultRules()
	parser, err := codecParsers("codex")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	counted := &cancellingTranscriptParser{TranscriptParser: parser, cancel: cancel}
	text := "first line\nsecond line\n"

	_, err = toolfacts.ReadTranscript(ctx, rules, strings.NewReader(text), counted)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("read error = %v, want context cancellation", err)
	}
	if counted.calls != 1 {
		t.Fatalf("parser calls = %d, want 1 before cancellation stops the next line", counted.calls)
	}
}

type cancellingTranscriptParser struct {
	runner.TranscriptParser
	calls  int
	cancel context.CancelFunc
}

func (p *cancellingTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	p.calls++
	result := p.TranscriptParser.ParseTranscriptLine(runID, line)
	p.cancel()
	return result
}

func readerFixture(t *testing.T) (*toolfacts.Reader, *sqlx.DB, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(dir, "facts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE runs (id TEXT PRIMARY KEY, transcript_path TEXT, import_source_harness TEXT, resolved_config TEXT)`,
		`CREATE TABLE run_events (id TEXT PRIMARY KEY, run_id TEXT NOT NULL, sequence INTEGER NOT NULL, event_type TEXT NOT NULL, timestamp TEXT, data TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	return &toolfacts.Reader{DB: db, Parsers: codecParsers, Rules: rules}, db, dir
}

func TestReaderPrefersEventsThenTranscriptThenSaysNothingWasObservable(t *testing.T) {
	reader, db, dir := readerFixture(t)
	transcript := filepath.Join(dir, "poll.jsonl")
	body, err := os.ReadFile(filepath.Join(captures, "corpus/poll-loop.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transcript, body, 0o600); err != nil {
		t.Fatal(err)
	}
	insertRun := func(id, path, harness, config string) {
		if _, err := db.Exec(`INSERT INTO runs VALUES (?,?,?,?)`, id, path, harness, config); err != nil {
			t.Fatal(err)
		}
	}
	insertRun("with-events", transcript, "codex", "")
	insertRun("imported", transcript, "codex", "")
	insertRun("native", transcript, "", `{"runnerType":"codex"}`)
	insertRun("deleted", filepath.Join(dir, "gone.jsonl"), "codex", "")
	insertRun("no-transcript", "", "", "")
	for i, event := range []struct{ kind, data string }{
		{"tool_call", `{"toolName":"bash","toolCallId":"c1","input":{"command":"ls /a"}}`},
		{"tool_result", `{"toolName":"bash","toolCallId":"c1","output":"a","success":true}`},
		{"message", `{"role":"user","content":"hi"}`},
	} {
		if _, err := db.Exec(`INSERT INTO run_events VALUES (?,?,?,?,?,?)`, "e"+string(rune('a'+i)), "with-events", i+1, event.kind, "2026-11-10T16:08:27.603818176Z", event.data); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		run, source string
		calls       int
	}{
		{"with-events", toolfacts.SourceRunEvents, 1},
		{"imported", toolfacts.SourceTranscript, 3},
		{"native", toolfacts.SourceTranscript, 3},
		{"deleted", toolfacts.SourceNone, 0},
		{"no-transcript", toolfacts.SourceNone, 0},
		{"unknown-run", toolfacts.SourceNone, 0},
	}
	for _, tc := range cases {
		facts, err := reader.Facts(context.Background(), tc.run)
		if err != nil {
			t.Fatalf("%s: %v", tc.run, err)
		}
		if facts.Source != tc.source || facts.ToolCalls != tc.calls {
			t.Errorf("%s: source %q with %d calls, want %q with %d", tc.run, facts.Source, facts.ToolCalls, tc.source, tc.calls)
		}
	}
}

// toolEventShape is what runsignal consumes: typed call and result events with
// distinct ids and an increasing sequence.
func TestEventStreamYieldsTheShapeTheInvocationReadersConsume(t *testing.T) {
	reader, db, dir := readerFixture(t)
	body, err := os.ReadFile(filepath.Join(captures, "codex_command_executions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "codemode.jsonl")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO runs VALUES ('imported', ?, 'codex', '')`, path); err != nil {
		t.Fatal(err)
	}
	stream, err := reader.EventStream(context.Background(), "imported")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Source != toolfacts.SourceTranscript {
		t.Fatalf("source = %q", stream.Source)
	}
	seen := map[string]bool{}
	var last int64
	calls := 0
	for event, err := range stream.Events {
		if err != nil {
			t.Fatal(err)
		}
		switch event.Data.(type) {
		case *domain.ToolCallEventData:
			calls++
		case *domain.ToolResultEventData:
		default:
			t.Fatalf("event %d is %T, want a tool call or result", event.Sequence, event.Data)
		}
		if seen[event.ID.String()] || event.Sequence <= last || event.Timestamp.IsZero() {
			t.Fatalf("event %d has a repeated id, a sequence out of order or no timestamp", event.Sequence)
		}
		seen[event.ID.String()] = true
		last = event.Sequence
	}
	if calls != 3 || last != 17 {
		t.Fatalf("calls = %d, events = %d, want 3 and 17", calls, last)
	}
}

func TestEventsCanBeAbandonedEarly(t *testing.T) {
	reader, db, dir := readerFixture(t)
	body, _ := os.ReadFile(filepath.Join(captures, "corpus/poll-loop.jsonl"))
	path := filepath.Join(dir, "poll.jsonl")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO runs VALUES ('imported', ?, 'codex', '')`, path); err != nil {
		t.Fatal(err)
	}
	stream, err := reader.EventStream(context.Background(), "imported")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, err := range stream.Events {
		if err != nil {
			t.Fatal(err)
		}
		if n++; n == 2 {
			break
		}
	}
	if n != 2 {
		t.Fatalf("read %d events before stopping, want 2", n)
	}
}
