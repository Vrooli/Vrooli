package codecs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptredact"

	"github.com/google/uuid"
)

func TestReplayCorpusParsesWithoutSecrets(t *testing.T) {
	cases := []struct {
		name                         string
		newCodec                     func(t *testing.T) Codec
		session                      string
		messages, tools, costMetrics int
	}{
		{"claude-stdout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewClaudeForTest() }, "751b5a53-bc44-4484-943d-8851ccfdfda1", 2, 2, 3},
		{"claude-ondisk.jsonl", func(t *testing.T) Codec { t.Helper(); return NewClaudeForTest() }, "a94bac9c-0365-4246-9a92-2fb4f1f5b67c", 1, 2, 0},
		{"codex-stdout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() }, "019b3906-b365-7403-b3d1-70d60f6f06c4", 2, 3, 1},
		{"codex-rollout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() }, "019f46b3-b60d-7373-bf64-b8748b8d5992", 2, 4, 0},
		{"codex-live-identity-guard.jsonl", func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() }, "019fa64c-e19a-72b1-bf39-bf6ec9a028ab", 2, 0, 1},
		{"codex-live-investigation-both-1.jsonl", func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() }, "019fac3d-becf-7212-9a87-bd05c57f4b1f", 2, 0, 1},
		{"codex-live-investigation-both-2.jsonl", func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() }, "019fac41-00fb-7210-b264-0d19dfc1dbdf", 2, 0, 1},
		{"poll-loop.jsonl", func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() }, "poll-loop-fixture", 0, 6, 1},
		{"grok-stdout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewGrokForTest() }, "019f10db-c647-7d93-9278-8bd0ab1e7528", 1, 0, 0},
		{"grok-updates.jsonl", func(t *testing.T) Codec { t.Helper(); return NewGrokForTest() }, "019f1023-e563-7513-a64e-96165fba4be6", 1, 2, 0},
		{"opencode-stdout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewOpenCodeForTest() }, "sess-1", 2, 3, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("testdata", "corpus", tc.name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lower := strings.ToLower(string(raw))
			for _, forbidden := range []string{"/home/", "api_key=", "authorization=", "bearer "} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("corpus contains unredacted %q", forbidden)
				}
			}
			parser := tc.newCodec(t).NewTranscriptParser()
			messages := 0
			tools := 0
			costs := 0
			session := ""
			var terminal *runner.TranscriptTerminal
			for i, line := range strings.Split(string(raw), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				result := parser.ParseTranscriptLine(uuid.New(), line)
				if result.Err != nil {
					t.Fatalf("line %d: %v", i+1, result.Err)
				}
				if result.SessionID != "" {
					session = result.SessionID
				}
				if result.Terminal != nil {
					terminal = result.Terminal
				}
				for _, event := range result.Events {
					if event != nil && event.EventType == domain.EventTypeMessage {
						messages++
					}
					if event != nil && (event.EventType == domain.EventTypeToolCall || event.EventType == domain.EventTypeToolResult) {
						tools++
					}
					if event != nil && event.EventType == domain.EventTypeMetric {
						if _, usage := event.Data.(*domain.UsageEventData); !usage {
							continue
						}
						costs++
					}
				}
			}
			if session != tc.session || messages != tc.messages || tools != tc.tools || costs != tc.costMetrics {
				t.Fatalf("replay = session=%q messages=%d tools=%d costs=%d, want session=%q messages=%d tools=%d costs=%d", session, messages, tools, costs, tc.session, tc.messages, tc.tools, tc.costMetrics)
			}
			if terminal == nil || !terminal.Success {
				t.Fatalf("replay terminal = %#v, want successful terminal", terminal)
			}
		})
	}
}

func TestCommittedTranscriptFixturesContainNoSecretsOrAbsoluteHomePaths(t *testing.T) {
	violations, err := transcriptredact.ScanDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("committed transcript fixture gate failed:\n%s", strings.Join(violations, "\n"))
	}
}

func TestTranscriptRedactionPreservesCodecClassification(t *testing.T) {
	path := filepath.Join("testdata", "claude_ondisk_trace.jsonl")
	redacted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	unsafe := strings.Replace(string(redacted), "<HOME>", "/home/alice", 1)
	if unsafe == string(redacted) {
		t.Fatal("fixture did not contain a redacted home path")
	}
	if got := transcriptredact.Redact(unsafe); got != string(redacted) {
		t.Fatalf("redaction did not restore canonical fixture:\n%s", got)
	}
	before := replayClassification(t, unsafe)
	after := replayClassification(t, string(redacted))
	if before != after {
		t.Fatalf("classification changed after transcript redaction: before=%+v after=%+v", before, after)
	}
}

type replayOutcome struct {
	messages, tools, metrics int
	terminalSuccess          bool
}

func replayClassification(t *testing.T, transcript string) replayOutcome {
	t.Helper()
	parser := NewClaudeForTest().NewTranscriptParser()
	outcome := replayOutcome{}
	for lineNumber, line := range strings.Split(transcript, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		result := parser.ParseTranscriptLine(uuid.New(), line)
		if result.Err != nil {
			t.Fatalf("line %d: %v", lineNumber+1, result.Err)
		}
		if result.Terminal != nil {
			outcome.terminalSuccess = result.Terminal.Success
		}
		for _, event := range result.Events {
			switch event.EventType {
			case domain.EventTypeMessage:
				outcome.messages++
			case domain.EventTypeToolCall, domain.EventTypeToolResult:
				outcome.tools++
			case domain.EventTypeMetric:
				outcome.metrics++
			}
		}
	}
	return outcome
}

func TestReplayCorpusResumesWithItsRecordedSession(t *testing.T) {
	// A second parser represents the continuation process. Each representative
	// capture contains the provider session identity needed to resume it.
	for _, tc := range []struct {
		name    string
		codec   func() Codec
		session string
	}{
		{"claude", func() Codec { return NewClaudeForTest() }, "751b5a53-bc44-4484-943d-8851ccfdfda1"},
		{"codex", func() Codec { return NewCodexForTest() }, "019b3906-b365-7403-b3d1-70d60f6f06c4"},
		{"grok", func() Codec { return NewGrokForTest() }, "019f1023-e563-7513-a64e-96165fba4be6"},
		{"opencode", func() Codec { return NewOpenCodeForTest() }, "sess-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := map[string]string{"claude": "claude-stdout.jsonl", "codex": "codex-stdout.jsonl", "grok": "grok-updates.jsonl", "opencode": "opencode-stdout.jsonl"}[tc.name]
			raw, err := os.ReadFile(filepath.Join("testdata", "corpus", file))
			if err != nil {
				t.Fatal(err)
			}
			parser := tc.codec().NewTranscriptParser()
			got := ""
			for _, line := range strings.Split(string(raw), "\n") {
				if result := parser.ParseTranscriptLine(uuid.New(), line); result.SessionID != "" {
					got = result.SessionID
				}
			}
			if got != tc.session {
				t.Fatalf("resume session = %q, want %q", got, tc.session)
			}
		})
	}
}

func TestReplayCorpusFailureTerminals(t *testing.T) {
	for _, tc := range []struct {
		name, line, want string
		codec            func() Codec
	}{
		{"claude", `{"type":"result","session_id":"claude-failure","is_error":true,"result":"provider failure"}`, "provider failure", func() Codec { return NewClaudeForTest() }},
		{"codex", `{"type":"error","thread_id":"codex-failure","error":{"message":"provider failure"}}`, "provider failure", func() Codec { return NewCodexForTest() }},
		{"grok", `{"type":"error","message":"provider failure"}`, "provider failure", func() Codec { return NewGrokForTest() }},
		{"opencode", `{"type":"step_finish","sessionID":"opencode-failure","part":{"type":"step-finish","reason":"error","output":"provider failure"}}`, "provider failure", func() Codec { return NewOpenCodeForTest() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.codec().NewTranscriptParser().ParseTranscriptLine(uuid.New(), tc.line)
			if result.Err != nil {
				t.Fatal(result.Err)
			}
			if result.Terminal == nil || result.Terminal.Success || result.Terminal.ErrorMessage != tc.want {
				t.Fatalf("failure terminal = %#v, want error %q", result.Terminal, tc.want)
			}
		})
	}
}
