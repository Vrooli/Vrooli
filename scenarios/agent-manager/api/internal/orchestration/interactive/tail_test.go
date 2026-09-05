package interactive

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// collectSink is a minimal runner.EventSink that records emitted events.
type collectSink struct {
	mu     sync.Mutex
	seq    int64
	events []*domain.RunEvent
}

func (s *collectSink) Emit(e *domain.RunEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	e.Sequence = s.seq
	s.events = append(s.events, e)
	return nil
}
func (s *collectSink) Close() error        { return nil }
func (s *collectSink) LastSequence() int64 { return s.seq }

func (s *collectSink) count(typ domain.RunEventType) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, e := range s.events {
		if e.EventType == typ {
			n++
		}
	}
	return n
}

// codecParserResolver maps runner types to their real codec transcript parser,
// so the tailer is exercised end-to-end against the shipped dialect branches.
func codecParserResolver(rt domain.RunnerType) (runner.TranscriptParser, error) {
	switch rt {
	case domain.RunnerTypeClaudeCode:
		return codecs.NewClaudeForTest().NewTranscriptParser(), nil
	case domain.RunnerTypeCodex:
		return codecs.NewCodexForTest().NewTranscriptParser(), nil
	case domain.RunnerTypeGrok:
		return codecs.NewGrokForTest().NewTranscriptParser(), nil
	default:
		return nil, os.ErrNotExist
	}
}

// readCodecFixture loads the sanitized real on-disk fixture the codec golden
// tests use, so the tailer replays the exact same per-agent samples.
func readCodecFixture(t *testing.T, name string) []string {
	t.Helper()
	path := filepath.Join("..", "..", "adapters", "runner", "codecs", "testdata", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %s: %v", path, err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		if line := strings.TrimRight(sc.Text(), "\r"); line != "" {
			out = append(out, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return out
}

// TestTailerLiveReplayPerAgent replays each agent's real on-disk fixture through
// the live tailer, writing the file incrementally and splitting the final
// (terminal) line across two writes to exercise the truncated-trailing-line
// path. It asserts the tailer surfaces the terminal marker and the same
// structured events the codec golden test sees.
func TestTailerLiveReplayPerAgent(t *testing.T) {
	cases := []struct {
		name        string
		runnerType  domain.RunnerType
		fixture     string
		wantTool    bool
		wantMessage bool
	}{
		{"claude", domain.RunnerTypeClaudeCode, "claude_ondisk_trace.jsonl", true, true},
		{"codex", domain.RunnerTypeCodex, "codex_rollout_trace.jsonl", true, true},
		{"grok", domain.RunnerTypeGrok, "grok_updates_trace.jsonl", true, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			lines := readCodecFixture(t, tc.fixture)
			if len(lines) < 2 {
				t.Fatalf("fixture %s too short", tc.fixture)
			}
			dir := t.TempDir()
			path := filepath.Join(dir, "transcript.jsonl")
			if err := os.WriteFile(path, nil, 0o644); err != nil {
				t.Fatalf("seed file: %v", err)
			}

			sink := &collectSink{}
			tailer := NewTailer(codecParserResolver, WithTailPollInterval(10*time.Millisecond))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			type outcome struct {
				terminal *runner.TranscriptTerminal
				err      error
			}
			done := make(chan outcome, 1)
			go func() {
				term, err := tailer.Tail(ctx, TailParams{
					RunID:          uuid.New(),
					RunnerType:     tc.runnerType,
					TranscriptPath: path,
					Sink:           sink,
				})
				done <- outcome{term, err}
			}()

			f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				t.Fatalf("open for append: %v", err)
			}
			// Write every line except the last with its newline.
			for _, l := range lines[:len(lines)-1] {
				if _, err := f.WriteString(l + "\n"); err != nil {
					t.Fatalf("write: %v", err)
				}
			}
			_ = f.Sync()
			time.Sleep(30 * time.Millisecond)
			// Truncated trailing line: write the terminal line in two
			// fragments so a poll observes it half-written.
			last := lines[len(lines)-1]
			mid := len(last) / 2
			if _, err := f.WriteString(last[:mid]); err != nil {
				t.Fatalf("write frag1: %v", err)
			}
			_ = f.Sync()
			time.Sleep(30 * time.Millisecond)
			if _, err := f.WriteString(last[mid:] + "\n"); err != nil {
				t.Fatalf("write frag2: %v", err)
			}
			_ = f.Sync()
			_ = f.Close()

			select {
			case res := <-done:
				if res.err != nil {
					t.Fatalf("Tail returned error: %v", res.err)
				}
				if res.terminal == nil {
					t.Fatalf("no terminal marker surfaced")
				}
				if !res.terminal.Success {
					t.Fatalf("terminal = failure %q, want success", res.terminal.ErrorMessage)
				}
			case <-ctx.Done():
				t.Fatalf("tail did not complete: %v", ctx.Err())
			}

			if tc.wantMessage && sink.count(domain.EventTypeMessage) == 0 {
				t.Errorf("no Message events emitted")
			}
			if tc.wantTool && sink.count(domain.EventTypeToolCall) == 0 {
				t.Errorf("no ToolCall events emitted")
			}
		})
	}
}

// TestTailerLateFile proves the tailer waits for a transcript that does not
// exist when the tail starts (claude discovery can resolve the path before the
// CLI writes the file, design R3).
func TestTailerLateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "late.jsonl")

	sink := &collectSink{}
	tailer := NewTailer(codecParserResolver, WithTailPollInterval(10*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan *runner.TranscriptTerminal, 1)
	go func() {
		term, err := tailer.Tail(ctx, TailParams{
			RunID:          uuid.New(),
			RunnerType:     domain.RunnerTypeClaudeCode,
			TranscriptPath: path,
			Sink:           sink,
		})
		if err != nil {
			t.Errorf("Tail error: %v", err)
		}
		done <- term
	}()

	// Create the file only after the tailer has had time to poll a missing path.
	time.Sleep(60 * time.Millisecond)
	content := strings.Join(readCodecFixture(t, "claude_ondisk_trace.jsonl"), "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write late file: %v", err)
	}

	select {
	case term := <-done:
		if term == nil || !term.Success {
			t.Fatalf("expected success terminal from late file, got %+v", term)
		}
	case <-ctx.Done():
		t.Fatalf("tail did not pick up late file: %v", ctx.Err())
	}
}

// TestTailerCodexRotation proves the tailer follows a codex rollout rotation:
// it drains the first rollout file, then switches to a newer rollout that
// appears under the run-scoped CODEX_HOME and reads its terminal marker
// (design R4).
func TestTailerCodexRotation(t *testing.T) {
	runDir := t.TempDir()
	sessDir := filepath.Join(runDir, "codex", "sessions", "2026", "07", "13")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fixture := readCodecFixture(t, "codex_rollout_trace.jsonl")
	// Split the fixture: first rollout gets the head (no terminal), the rotated
	// rollout gets the tail including task_complete.
	head := fixture[:len(fixture)-2]
	tail := fixture[len(fixture)-2:]

	rolloutA := filepath.Join(sessDir, "rollout-2026-07-13T10-00-00-aaaa.jsonl")
	if err := os.WriteFile(rolloutA, []byte(strings.Join(head, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write rollout A: %v", err)
	}
	old := time.Now().Add(-1 * time.Minute)
	_ = os.Chtimes(rolloutA, old, old)

	sink := &collectSink{}
	tailer := NewTailer(codecParserResolver, WithTailPollInterval(10*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan *runner.TranscriptTerminal, 1)
	go func() {
		term, err := tailer.Tail(ctx, TailParams{
			RunID:          uuid.New(),
			RunnerType:     domain.RunnerTypeCodex,
			TranscriptPath: rolloutA,
			RunDir:         runDir,
			Sink:           sink,
		})
		if err != nil {
			t.Errorf("Tail error: %v", err)
		}
		done <- term
	}()

	// After the head is drained, a newer rollout file appears (rotation).
	time.Sleep(60 * time.Millisecond)
	rolloutB := filepath.Join(sessDir, "rollout-2026-07-13T11-00-00-bbbb.jsonl")
	if err := os.WriteFile(rolloutB, []byte(strings.Join(tail, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write rollout B: %v", err)
	}

	select {
	case term := <-done:
		if term == nil || !term.Success {
			t.Fatalf("expected success terminal after rotation, got %+v", term)
		}
	case <-ctx.Done():
		t.Fatalf("tail did not follow rotation: %v", ctx.Err())
	}
	if sink.count(domain.EventTypeMessage) == 0 {
		t.Errorf("expected the agent_message from the rotated rollout")
	}
}
