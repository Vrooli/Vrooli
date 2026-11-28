package toolfacts_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/toolfacts"
)

func TestExtractorAcceptsRulesWithoutTailWindows(t *testing.T) {
	const raw = `{
		"unchanged_window": 8,
		"chunk_min_events": 1,
		"chunk_max_events": 16,
		"max_line_bytes": 4096,
		"result_head_bytes": 64,
		"result_tail_bytes": 16,
		"tail_calls": []
	}`
	rules, err := toolfacts.ParseRules([]byte(raw))
	if err != nil {
		t.Fatalf("ParseRules rejected an empty tail window list: %v", err)
	}
	if len(rules.TailCalls) != 0 {
		t.Fatalf("parsed tail windows = %v, want none", rules.TailCalls)
	}

	extractor := toolfacts.New(rules)
	extractor.Add(domain.NewToolCallEvent(uuid.New(), "Read", "call-1", map[string]any{"file_path": "/repo/file.go"}))
	facts := extractor.Facts()
	if facts.ToolCalls != 1 {
		t.Fatalf("tool calls = %d, want 1", facts.ToolCalls)
	}
	if len(facts.Tails) != 0 {
		t.Fatalf("tail facts = %v, want none", facts.Tails)
	}
}

func TestReaderFallsBackToNoSourceWhenTranscriptCannotBeOpened(t *testing.T) {
	cases := []struct {
		name    string
		resolve func(path string) toolfacts.ParserResolver
	}{
		{
			name: "parser unavailable for existing transcript",
			resolve: func(_ string) toolfacts.ParserResolver {
				return func(string) (runner.TranscriptParser, error) {
					return nil, errors.New("no parser for harness")
				}
			},
		},
		{
			name: "resolver returns no parser",
			resolve: func(_ string) toolfacts.ParserResolver {
				return func(string) (runner.TranscriptParser, error) { return nil, nil }
			},
		},
		{
			name: "transcript removed after locate",
			resolve: func(path string) toolfacts.ParserResolver {
				return func(string) (runner.TranscriptParser, error) {
					if err := os.Remove(path); err != nil {
						return nil, err
					}
					return nil, nil
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader, db, dir := readerFixture(t)
			path := filepath.Join(dir, "run.jsonl")
			if err := os.WriteFile(path, []byte("transcript content\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			reader.Parsers = tc.resolve(path)
			if _, err := db.Exec(`INSERT INTO runs VALUES ('transcript-run', ?, 'test-harness', '')`, path); err != nil {
				t.Fatal(err)
			}

			facts, err := reader.Facts(context.Background(), "transcript-run")
			if err != nil {
				t.Fatalf("Reader.Facts returned an error: %v", err)
			}
			if facts.Source != toolfacts.SourceNone || facts.ToolCalls != 0 {
				t.Fatalf("source/calls = %q/%d, want %q/0", facts.Source, facts.ToolCalls, toolfacts.SourceNone)
			}
			if facts.ParseErrors != 0 || facts.SkippedLines != 0 {
				t.Fatalf("parse/skip diagnostics = %d/%d, want 0/0", facts.ParseErrors, facts.SkippedLines)
			}
		})
	}
}
