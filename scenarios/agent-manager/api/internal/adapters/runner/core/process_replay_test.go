package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// TestProcessReplay_AllSupportedCodecs drives a real local process through the
// host launcher, stdout/transcript writer, codec parser, and terminal result.
// The executable is cmd/fake-agent, never a real coding-agent binary.
func TestProcessReplay_AllSupportedCodecs(t *testing.T) {
	fakeAgent := testutil.BuildFakeAgent(t)
	cases := []struct {
		name   string
		corpus string
		codec  codecs.Codec
	}{
		{"claude", "claude-stdout.jsonl", codecs.NewClaudeForTestWithBinary(fakeAgent)},
		{"codex", "codex-stdout.jsonl", codecs.NewCodexForTestWithBinary(fakeAgent)},
		{"grok", "grok-stdout.jsonl", codecs.NewGrokForTestWithBinary(fakeAgent)},
		{"opencode", "opencode-stdout.jsonl", codecs.NewOpenCodeForTestWithBinary(fakeAgent)},
		{"antigravity", "antigravity-stdout.jsonl", codecs.NewAntigravityForTestWithBinary(fakeAgent)},
	}
	if len(cases) != len(domain.ValidRunnerTypes()) {
		t.Fatalf("process replay cases = %d, supported runners = %d", len(cases), len(domain.ValidRunnerTypes()))
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workDir := t.TempDir()
			transcript, err := os.CreateTemp(workDir, "runner-transcript-*.jsonl")
			if err != nil {
				t.Fatal(err)
			}
			defer transcript.Close()
			marker := filepath.Join(workDir, "tag-marker")
			sink := &recordingSink{}
			corpus, err := filepath.Abs(filepath.Join("..", "codecs", "testdata", "corpus", tc.corpus))
			if err != nil {
				t.Fatal(err)
			}
			r := NewRunner(tc.codec, runner.NewHostLauncher(), nil)
			result, err := r.Execute(context.Background(), runner.ExecuteRequest{
				RunID: uuid.New(), Tag: "replay-" + tc.name, Prompt: "replay corpus", WorkingDir: workDir,
				ResolvedConfig: &domain.RunConfig{RunnerType: tc.codec.Type()}, EventSink: sink,
				Environment: map[string]string{
					"FAKE_AGENT_CORPUS":     corpus,
					"FAKE_AGENT_TAG_MARKER": marker,
				},
				Transcript: &runner.TranscriptConfig{TranscriptPath: transcript.Name(), StdoutFile: transcript},
			})
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if !result.Success {
				t.Fatalf("result = %+v", result)
			}
			if result.SessionID == "" {
				t.Fatal("process replay did not capture a session id")
			}
			contents, err := os.ReadFile(transcript.Name())
			if err != nil || len(contents) == 0 {
				t.Fatalf("transcript missing or empty: %v", err)
			}
			tag, err := os.ReadFile(marker)
			if err != nil {
				t.Fatalf("tag marker: %v", err)
			}
			if !strings.Contains(string(tag), "replay-"+tc.name) {
				t.Fatalf("tag did not reach fake process: %q", tag)
			}
			assertReplayEvents(t, tc.name, sink.snapshot())
		})
	}
}

func assertReplayEvents(t *testing.T, name string, events []*domain.RunEvent) {
	t.Helper()
	var message, tool, metric bool
	for _, event := range events {
		switch event.EventType {
		case domain.EventTypeMessage:
			message = true
		case domain.EventTypeToolCall:
			tool = true
		case domain.EventTypeMetric:
			metric = true
		}
	}
	if !message {
		t.Fatalf("%s replay emitted no message events: %+v", name, events)
	}
	if name != "grok" && name != "antigravity" && !tool {
		t.Fatalf("%s replay emitted no tool events: %+v", name, events)
	}
	if name != "grok" && name != "antigravity" && !metric {
		t.Fatalf("%s replay emitted no metric events: %+v", name, events)
	}
}

func TestProcessReplay_FailureCorpusProducesFailedResult(t *testing.T) {
	fakeAgent := testutil.BuildFakeAgent(t)
	workDir := t.TempDir()
	corpus := filepath.Join(workDir, "failure.jsonl")
	if err := os.WriteFile(corpus, []byte(`{"type":"result","is_error":true,"result":"failed","session_id":"failure-session"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(codecs.NewClaudeForTestWithBinary(fakeAgent), runner.NewHostLauncher(), nil)
	result, err := r.Execute(context.Background(), runner.ExecuteRequest{RunID: uuid.New(), Tag: "failure-replay", Prompt: "fail", WorkingDir: workDir, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode}, EventSink: &recordingSink{}, Environment: map[string]string{"FAKE_AGENT_CORPUS": corpus}})
	if err == nil && (result == nil || result.Success) {
		t.Fatalf("failure corpus unexpectedly succeeded: result=%+v err=%v", result, err)
	}
}

// TestProcessReplay_ContinuationAllSupportedCodecs proves that a durable
// follow-up turn uses the same real launcher/transcript path as an initial
// turn. This catches regressions where a codec's continue argv or tag reaches
// a different execution seam than the initial replay.
func TestProcessReplay_ContinuationAllSupportedCodecs(t *testing.T) {
	fakeAgent := testutil.BuildFakeAgent(t)
	cases := []struct {
		name   string
		corpus string
		codec  codecs.Codec
	}{
		{"claude", "claude-stdout.jsonl", codecs.NewClaudeForTestWithBinary(fakeAgent)},
		{"codex", "codex-stdout.jsonl", codecs.NewCodexForTestWithBinary(fakeAgent)},
		{"grok", "grok-stdout.jsonl", codecs.NewGrokForTestWithBinary(fakeAgent)},
		{"opencode", "opencode-stdout.jsonl", codecs.NewOpenCodeForTestWithBinary(fakeAgent)},
		{"antigravity", "antigravity-stdout.jsonl", codecs.NewAntigravityForTestWithBinary(fakeAgent)},
	}
	if len(cases) != len(domain.ValidRunnerTypes()) {
		t.Fatalf("continuation replay cases = %d, supported runners = %d", len(cases), len(domain.ValidRunnerTypes()))
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workDir := t.TempDir()
			transcript, err := os.CreateTemp(workDir, "continue-transcript-*.jsonl")
			if err != nil {
				t.Fatal(err)
			}
			defer transcript.Close()
			marker := filepath.Join(workDir, "continue-tag-marker")
			corpus, err := filepath.Abs(filepath.Join("..", "codecs", "testdata", "corpus", tc.corpus))
			if err != nil {
				t.Fatal(err)
			}
			r := NewRunner(tc.codec, runner.NewHostLauncher(), nil)
			result, err := r.Continue(context.Background(), runner.ContinueRequest{
				RunID: uuid.New(), SessionID: "prior-session", Prompt: "follow-up", WorkingDir: workDir,
				ResolvedConfig: &domain.RunConfig{RunnerType: tc.codec.Type()}, EventSink: &recordingSink{},
				Environment: map[string]string{
					"FAKE_AGENT_CORPUS":     corpus,
					"FAKE_AGENT_TAG_MARKER": marker,
				},
				Transcript: &runner.TranscriptConfig{TranscriptPath: transcript.Name(), StdoutFile: transcript},
			})
			if err != nil {
				t.Fatalf("continue: %v", err)
			}
			if !result.Success || result.SessionID == "" {
				t.Fatalf("continue result = %+v", result)
			}
			if contents, err := os.ReadFile(transcript.Name()); err != nil || len(contents) == 0 {
				t.Fatalf("continuation transcript missing or empty: %v", err)
			}
			tag, err := os.ReadFile(marker)
			if err != nil {
				t.Fatalf("continuation tag marker: %v", err)
			}
			if !strings.Contains(string(tag), "continue-") {
				t.Fatalf("continuation tag did not reach fake process: %q", tag)
			}
		})
	}
}
