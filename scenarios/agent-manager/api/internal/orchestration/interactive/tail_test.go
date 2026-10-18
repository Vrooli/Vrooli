package interactive

import (
	"bufio"
	"context"
	"errors"
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

// TestTailerStructuredResultCompletesGoalRun proves that a deterministic
// structured result is a valid engine-owned completion signal even when the
// interactive provider emits only its ordinary success marker. Goal-session
// workers return the workflow result contract; they are not required to emit a
// second runner-specific goal marker.
func TestTailerStructuredResultCompletesGoalRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	content := strings.Join(readCodecFixture(t, "codex_rollout_trace.jsonl"), "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	tailer := NewTailer(codecParserResolver, WithTailPollInterval(10*time.Millisecond))
	called := 0
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	term, err := tailer.Tail(ctx, TailParams{
		RunID:          uuid.New(),
		RunnerType:     domain.RunnerTypeCodex,
		TranscriptPath: path,
		Until:          "complete the engine-owned contract",
		StructuredResultSatisfied: func() bool {
			called++
			return true
		},
	})
	if err != nil {
		t.Fatalf("Tail returned error: %v", err)
	}
	if term == nil || !term.Success {
		t.Fatalf("terminal = %+v, want successful structured-result completion", term)
	}
	if term.TerminalReason != "structured_result" {
		t.Fatalf("terminal reason = %q, want structured_result", term.TerminalReason)
	}
	if called != 1 {
		t.Fatalf("structured result callback calls = %d, want 1", called)
	}
}

func TestTailerDrainsAcceptedGoalThroughSameProviderTurn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	goalLines := readCodecFixture(t, "codex_goal_accepted.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(goalLines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write goal transcript: %v", err)
	}

	sink := &collectSink{}
	goalObserved := make(chan struct{}, 1)
	tailer := NewTailer(codecParserResolver, WithTailPollInterval(5*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	runID := uuid.New()
	done := make(chan struct {
		terminal *runner.TranscriptTerminal
		err      error
	}, 1)
	go func() {
		terminal, err := tailer.Tail(ctx, TailParams{
			RunID:            runID,
			RunnerType:       domain.RunnerTypeCodex,
			TranscriptPath:   path,
			SessionID:        "fixture-thread",
			Until:            "complete the fixture",
			GoalDrainTimeout: 500 * time.Millisecond,
			Sink:             sink,
			OnGoalStatus: func(runner.GoalMarker) error {
				select {
				case goalObserved <- struct{}{}:
				default:
				}
				return nil
			},
		})
		done <- struct {
			terminal *runner.TranscriptTerminal
			err      error
		}{terminal, err}
	}()

	select {
	case <-goalObserved:
	case <-ctx.Done():
		t.Fatalf("native goal was not observed: %v", ctx.Err())
	}
	// D21's ordering has the final assistant response and provider boundary
	// after the matched goal-update result.
	time.Sleep(25 * time.Millisecond)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open transcript for append: %v", err)
	}
	for _, line := range []string{
		`{"type":"event_msg","payload":{"type":"agent_message","message":"final answer after accepted goal"}}`,
		`{"type":"event_msg","payload":{"type":"task_complete","turn_id":"fixture-turn"}}`,
	} {
		if _, err := f.WriteString(line + "\n"); err != nil {
			_ = f.Close()
			t.Fatalf("append transcript: %v", err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close transcript: %v", err)
	}

	var gotTerminal *runner.TranscriptTerminal
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Tail returned error: %v", got.err)
		}
		if got.terminal == nil || !got.terminal.Success || got.terminal.TerminalReason != "goal_complete" {
			t.Fatalf("terminal = %+v, want accepted goal_complete verdict", got.terminal)
		}
		gotTerminal = got.terminal
	case <-ctx.Done():
		t.Fatalf("Tail did not finish at provider turn boundary: %v", ctx.Err())
	}

	sink.mu.Lock()
	finalCaptured := false
	for _, event := range sink.events {
		if message, ok := event.Data.(*domain.MessageEventData); ok && message.Content == "final answer after accepted goal" {
			finalCaptured = true
			break
		}
	}
	sink.mu.Unlock()
	if !finalCaptured {
		t.Fatal("same-turn final assistant answer was not emitted before finalization")
	}

	run := newInteractiveRun(domain.RunnerTypeCodex, path)
	run.ID = runID
	store := &fakeRunStore{}
	broadcaster := &fakeBroadcaster{}
	resultBuilds := 0
	coord := NewCoordinator(CoordinatorDeps{
		Runs:        store,
		Broadcaster: broadcaster,
		Result: func(_ context.Context, _ uuid.UUID, success bool, exitCode int, reason string) (*domain.RunResult, *domain.RunSummary) {
			resultBuilds++
			sink.mu.Lock()
			defer sink.mu.Unlock()
			for _, event := range sink.events {
				if message, ok := event.Data.(*domain.MessageEventData); ok && message.Content == "final answer after accepted goal" {
					return &domain.RunResult{FinalOutput: message.Content, Success: success, ExitCode: exitCode, TerminalReason: reason}, &domain.RunSummary{Description: message.Content}
				}
			}
			return &domain.RunResult{Success: success, ExitCode: exitCode, TerminalReason: reason}, nil
		},
	})
	if err := coord.Finalize(ctx, run, gotTerminal, nil); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if resultBuilds != 1 || run.Result == nil || run.Result.FinalOutput != "final answer after accepted goal" || run.LastHandoff != "final answer after accepted goal" {
		t.Fatalf("finalized result did not select captured handoff: builds=%d result=%+v handoff=%q", resultBuilds, run.Result, run.LastHandoff)
	}
	broadcaster.mu.Lock()
	defer broadcaster.mu.Unlock()
	if len(broadcaster.statuses) != 1 || broadcaster.statuses[0] != domain.RunStatusComplete {
		t.Fatalf("completion was not broadcast exactly once: %v", broadcaster.statuses)
	}
}

func TestTailerGoalDrainStopsAtNextUserTurnOrSession(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{
			name: "next user turn",
			lines: []string{
				`{"type":"event_msg","payload":{"type":"user_message","message":"start another task"}}`,
				`{"type":"event_msg","payload":{"type":"agent_message","message":"answer from later turn"}}`,
				`{"type":"event_msg","payload":{"type":"task_complete","turn_id":"later-turn"}}`,
			},
		},
		{
			name: "session change",
			lines: []string{
				`{"type":"session_meta","payload":{"id":"different-session"}}`,
				`{"type":"event_msg","payload":{"type":"agent_message","message":"answer from another session"}}`,
				`{"type":"event_msg","payload":{"type":"task_complete","turn_id":"other-session-turn"}}`,
			},
		},
		{
			name: "next provider turn",
			lines: []string{
				`{"type":"turn_context","payload":{"turn_id":"later-turn"}}`,
				`{"type":"event_msg","payload":{"type":"agent_message","message":"answer from later provider turn"}}`,
				`{"type":"event_msg","payload":{"type":"task_complete","turn_id":"later-turn"}}`,
			},
		},
		{
			name: "replacement active goal",
			lines: []string{
				`{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"a replacement goal","status":"active"}}}`,
				`{"type":"event_msg","payload":{"type":"agent_message","message":"answer from replacement goal"}}`,
				`{"type":"event_msg","payload":{"type":"task_complete","turn_id":"replacement-turn"}}`,
			},
		},
		{
			name: "later goal status cannot override accepted verdict",
			lines: []string{
				`{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"complete the fixture","status":"usage_limited"}}}`,
				`{"type":"event_msg","payload":{"type":"agent_message","message":"answer after accepted goal"}}`,
				`{"type":"event_msg","payload":{"type":"task_complete","turn_id":"later-status-turn"}}`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "transcript.jsonl")
			lines := append(readCodecFixture(t, "codex_goal_accepted.jsonl"), tc.lines...)
			if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
				t.Fatalf("write transcript: %v", err)
			}
			sink := &collectSink{}
			tailer := NewTailer(codecParserResolver, WithTailPollInterval(5*time.Millisecond))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var goalStatuses []runner.GoalStatus
			term, err := tailer.Tail(ctx, TailParams{
				RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex,
				TranscriptPath: path, SessionID: "fixture-thread",
				Until: "complete the fixture", GoalDrainTimeout: 400 * time.Millisecond,
				Sink: sink,
				OnGoalStatus: func(marker runner.GoalMarker) error {
					goalStatuses = append(goalStatuses, marker.Status)
					return nil
				},
			})
			if err != nil || term == nil || term.TerminalReason != "goal_complete" {
				t.Fatalf("Tail terminal=%+v err=%v, want accepted goal verdict", term, err)
			}
			if got := sink.count(domain.EventTypeMessage); got != 0 {
				t.Fatalf("captured %d message(s) from after the invocation boundary", got)
			}
			if len(goalStatuses) != 2 || goalStatuses[0] != runner.GoalStatusActive || goalStatuses[1] != runner.GoalStatusComplete {
				t.Fatalf("processed goal statuses %v through the prior invocation boundary; want only original active+complete markers", goalStatuses)
			}
		})
	}
}

func TestTailerGoalOnlyCompletesOnBoundedDrainAndCancellationIsHonored(t *testing.T) {
	goalLines := readCodecFixture(t, "codex_goal_accepted.jsonl")
	for _, tc := range []struct {
		name   string
		cancel bool
	}{
		{name: "goal only"},
		{name: "cancel while draining", cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "transcript.jsonl")
			if err := os.WriteFile(path, []byte(strings.Join(goalLines, "\n")+"\n"), 0o644); err != nil {
				t.Fatalf("write transcript: %v", err)
			}
			tailer := NewTailer(codecParserResolver, WithTailPollInterval(5*time.Millisecond))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			observed := make(chan struct{}, 1)
			termCh := make(chan *runner.TranscriptTerminal, 1)
			errCh := make(chan error, 1)
			go func() {
				term, err := tailer.Tail(ctx, TailParams{
					RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex,
					TranscriptPath: path, SessionID: "fixture-thread",
					Until: "complete the fixture", GoalDrainTimeout: 80 * time.Millisecond,
					OnGoalStatus: func(runner.GoalMarker) error {
						select {
						case observed <- struct{}{}:
						default:
						}
						if tc.cancel {
							cancel()
						}
						return nil
					},
				})
				termCh <- term
				errCh <- err
			}()
			select {
			case <-observed:
			case <-time.After(time.Second):
				t.Fatal("native goal was not observed")
			}
			select {
			case term := <-termCh:
				err := <-errCh
				if tc.cancel {
					if err == nil || !errors.Is(err, context.Canceled) {
						t.Fatalf("Tail error=%v, want cancellation", err)
					}
					return
				}
				if err != nil || term == nil || term.TerminalReason != "goal_complete" {
					t.Fatalf("Tail terminal=%+v err=%v, want bounded goal completion", term, err)
				}
			case <-time.After(500 * time.Millisecond):
				t.Fatal("goal-only drain exceeded its configured bound")
			}
		})
	}
}

func TestTailerCarriesBillingIntoCodexTranscript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	content := strings.Join([]string{
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"output_tokens":5}}}}`,
		`{"type":"event_msg","payload":{"type":"task_complete"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	sink := &collectSink{}
	tailer := NewTailer(codecParserResolver, WithTailPollInterval(10*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	term, err := tailer.Tail(ctx, TailParams{
		RunID:          uuid.New(),
		RunnerType:     domain.RunnerTypeCodex,
		TranscriptPath: path,
		Billing:        domain.BillingSnapshot{Mode: domain.BillingModeSubscription},
		Sink:           sink,
	})
	if err != nil {
		t.Fatalf("Tail returned error: %v", err)
	}
	if term == nil || !term.Success {
		t.Fatalf("terminal = %+v, want success", term)
	}

	sink.mu.Lock()
	defer sink.mu.Unlock()
	for _, event := range sink.events {
		charge, ok := event.Data.(*domain.ChargeEventData)
		if ok && charge.Basis == domain.ChargeBasisSubscription && charge.AmountMicroUSD != nil && *charge.AmountMicroUSD == 0 {
			return
		}
	}
	t.Fatal("interactive Codex transcript did not emit an explicit zero subscription charge")
}

func TestTailerAcceptsPinnedCodexGetGoalRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "get-goal.jsonl")
	transcript := strings.Join([]string{
		`{"type":"response_item","payload":{"type":"custom_tool_call","name":"exec","call_id":"goal-read","input":"text(JSON.stringify(await tools.get_goal({})));"}}`,
		`{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"goal-read","output":[{"type":"input_text","text":"Script completed\nWall time 0.0 seconds\nOutput:\n"},{"type":"input_text","text":"{\"goal\":{\"threadId\":\"fixture-thread\",\"objective\":\"finish the route\\ninitial prompt context\",\"status\":\"complete\",\"tokensUsed\":12,\"timeUsedSeconds\":4,\"createdAt\":1,\"updatedAt\":2},\"remainingTokens\":null,\"completionBudgetReport\":null}"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	tailer := NewTailer(codecParserResolver, WithTailPollInterval(time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	terminal, err := tailer.Tail(ctx, TailParams{
		RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex,
		TranscriptPath: path, SessionID: "fixture-thread", Until: "finish the route",
		GoalDrainTimeout: 10 * time.Millisecond,
	})
	if err != nil || terminal == nil || !terminal.Success || terminal.TerminalReason != "goal_complete" {
		t.Fatalf("pinned native get_goal result was not accepted: terminal=%+v err=%v", terminal, err)
	}
}
