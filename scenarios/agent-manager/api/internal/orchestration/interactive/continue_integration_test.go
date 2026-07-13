//go:build integration

// Phase 5 live integration: drives multi-turn interactive Continue + human input
// + Stop against a LIVE web-console, end-to-end through the real client
// (SendPrompt paste+Enter, SendText), the real Coordinator turn-boundary debounce
// (the Continue mechanism — a typed follow-up grows the transcript, the debounce
// treats the growth as a new turn), the real Tailer + codec parser (event
// extraction), and the real Substrate.Stop teardown.
//
// The "agent" is a fake codex CLI that writes turn 1 to its rollout, then reads
// stdin and appends a NEW turn per line received — so a SendPrompt (agent
// Continue) and a SendText (human input on a different source) each produce a
// fresh turn in the structured events, exactly as a real interactive turn does.
//
// The orchestration wrapper (Orchestrator.ContinueRun/StopRun registry +
// status transitions) is covered by the unit tests in the orchestration package;
// this test proves the live web-console interaction the wrapper depends on.
//
// Run with: go test -tags=integration ./internal/orchestration/interactive/...
// Skips cleanly when web-console is unreachable.
package interactive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestLive_InteractiveContinueHumanInputAndStop(t *testing.T) {
	client := liveClient(t)

	workDir := t.TempDir()
	runDir := t.TempDir()

	// A fake codex CLI: writes turn 1 immediately, then appends a new turn for
	// every stdin line it receives (a typed Continue follow-up or human input).
	script := filepath.Join(t.TempDir(), "fake-codex-continue.sh")
	scriptBody := `#!/bin/sh
set -e
d="$CODEX_HOME/sessions/2026/07/13"
mkdir -p "$d"
f="$d/rollout-2026-07-13T10-00-00-p5.jsonl"
printf '%s\n' '{"type":"session_meta","payload":{"id":"live-p5"}}' > "$f"
printf '%s\n' '{"type":"event_msg","payload":{"type":"agent_message","message":"turn one ready"}}' >> "$f"
printf '%s\n' '{"type":"event_msg","payload":{"type":"task_complete","turn_id":"t1"}}' >> "$f"
n=1
while IFS= read -r line; do
  n=$((n+1))
  printf '%s\n' "{\"type\":\"event_msg\",\"payload\":{\"type\":\"agent_message\",\"message\":\"reply turn $n\"}}" >> "$f"
  printf '%s\n' "{\"type\":\"event_msg\",\"payload\":{\"type\":\"task_complete\",\"turn_id\":\"t$n\"}}" >> "$f"
done
`
	if err := os.WriteFile(script, []byte(scriptBody), 0o755); err != nil {
		t.Fatalf("write fake codex script: %v", err)
	}

	sub := NewSubstrate(client, fakeResolver(fakeLaunchInfo{
		rt:     domain.RunnerTypeCodex,
		tagKey: "CODEX_AGENT_TAG",
		binary: script,
	}), WithDiscoveryTimeout(15*time.Second), WithPollInterval(150*time.Millisecond), WithStopGrace(200*time.Millisecond))

	ctx := context.Background()
	res, err := sub.Launch(ctx, LaunchParams{
		RunID:      uuid.New(),
		RunnerType: domain.RunnerTypeCodex,
		Tag:        "phase5-live-continue",
		WorkingDir: workDir,
		RunDir:     runDir,
	})
	if res.SessionID != "" {
		t.Cleanup(func() { _ = client.DeleteSession(context.Background(), res.SessionID) })
	}
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Drive the real Coordinator turn-boundary loop in the background: it tails
	// turn 1, then the idle-debounce keeps it alive between turns so each typed
	// follow-up is picked up as a new turn (the Continue mechanism).
	sink := &collectSink{}
	store := &fakeRunStore{}
	coord := NewCoordinator(CoordinatorDeps{
		Tailer:       NewTailer(codecParserResolver, WithTailPollInterval(50*time.Millisecond)),
		Sessions:     client,
		Runs:         store,
		NewSink:      func(uuid.UUID) runner.EventSink { return sink },
		Debounce:     3 * time.Second,
		ActivityPoll: 100 * time.Millisecond,
		SessionPoll:  2 * time.Second,
		Heartbeat:    -1,
	})

	run := &domain.Run{
		ID:             uuid.New(),
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex},
		TranscriptPath: res.TranscriptPath,
	}

	type tailResult struct {
		terminal *runner.TranscriptTerminal
		err      error
	}
	done := make(chan tailResult, 1)
	tctx, cancelTail := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTail()
	go func() {
		term, terr := coord.TailToCompletion(tctx, run)
		done <- tailResult{terminal: term, err: terr}
	}()

	// Give the tailer a beat to consume turn 1 and enter the debounce.
	time.Sleep(1 * time.Second)

	// Continue: type a follow-up via the real SendPrompt (paste + Enter). The
	// fake agent appends turn 2.
	if err := client.SendPrompt(ctx, res.SessionID, "please do the second turn", "agent-manager:run-continue"); err != nil {
		t.Fatalf("SendPrompt (Continue): %v", err)
	}
	time.Sleep(1 * time.Second)

	// Human input on a DIFFERENT source into the same pane (free-for-all stdin).
	// The fake agent appends turn 3; event extraction must stay correct.
	if err := client.SendText(ctx, res.SessionID, "human follow-up here\n", "human:operator"); err != nil {
		t.Fatalf("SendText (human): %v", err)
	}

	// Let the final debounce elapse with no further input → run completes.
	var got tailResult
	select {
	case got = <-done:
	case <-time.After(28 * time.Second):
		cancelTail()
		t.Fatal("Coordinator.TailToCompletion did not finish")
	}
	if got.err != nil {
		t.Fatalf("TailToCompletion: %v", got.err)
	}
	if got.terminal == nil || !got.terminal.Success {
		t.Fatalf("expected a success terminal after the final turn, got %+v", got.terminal)
	}

	// Three turns' worth of assistant messages must have been extracted: the
	// initial turn, the Continue (SendPrompt) turn, and the human (SendText) turn.
	if msgs := sink.count(domain.EventTypeMessage); msgs < 3 {
		t.Fatalf("expected >=3 assistant messages across turns, got %d", msgs)
	}
	t.Logf("live continue: messages=%d terminal.success=%v", sink.count(domain.EventTypeMessage), got.terminal.Success)

	// Stop end-to-end: interrupt then delete; the session is gone afterward.
	if err := sub.Stop(ctx, res.SessionID, "agent-manager:run-continue"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, err := client.GetSession(ctx, res.SessionID); !errors.Is(err, webconsole.ErrSessionNotFound) {
		t.Fatalf("expected session gone after Stop, got %v", err)
	}
}
