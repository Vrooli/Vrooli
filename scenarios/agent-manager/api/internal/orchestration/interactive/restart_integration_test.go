//go:build integration

// Phase 4 live restart drill: launches a real interactive session, tails it with
// a first coordinator, simulates an agent-manager restart by cancelling that
// coordinator mid-tail, then reattaches a FRESH coordinator built only from the
// persisted run row (session id + transcript path + cursor). The reattached
// coordinator must resume from the cursor (no duplicated events), observe the
// terminal marker the fake agent writes after the "restart", and finalize the
// run Complete — proving an interactive run survives an agent-manager restart.
//
// Run with: go test -tags=integration ./internal/orchestration/interactive/...
// Skips cleanly when web-console is unreachable.
package interactive

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestLive_InteractiveRestartReattachCompletes(t *testing.T) {
	client := liveClient(t)

	workDir := t.TempDir()
	runDir := t.TempDir()

	// A fake codex CLI that writes its rollout up to (but not including) the
	// terminal, then waits a few seconds — the window during which we simulate an
	// agent-manager restart — before writing task_complete.
	script := filepath.Join(t.TempDir(), "fake-codex-restart.sh")
	scriptBody := `#!/bin/sh
set -e
d="$CODEX_HOME/sessions/2026/07/13"
mkdir -p "$d"
f="$d/rollout-2026-07-13T10-00-00-restart.jsonl"
printf '%s\n' '{"type":"session_meta","payload":{"id":"live-restart"}}' > "$f"
sleep 0.3
printf '%s\n' '{"type":"response_item","payload":{"type":"function_call","name":"exec","arguments":"{\"cmd\":\"ls\"}","call_id":"c1"}}' >> "$f"
printf '%s\n' '{"type":"response_item","payload":{"type":"function_call_output","call_id":"c1","output":"file.txt"}}' >> "$f"
printf '%s\n' '{"type":"event_msg","payload":{"type":"agent_message","message":"working before restart"}}' >> "$f"
sleep 3
printf '%s\n' '{"type":"event_msg","payload":{"type":"task_complete","turn_id":"t1"}}' >> "$f"
exec sh
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
		Tag:        "phase4-restart-drill",
		WorkingDir: workDir,
		RunDir:     runDir,
	})
	if res.SessionID != "" {
		t.Cleanup(func() { _ = sub.Stop(context.Background(), res.SessionID, "agent-manager:run-test") })
	}
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// The durable run row a restarted agent-manager would recover from.
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType = domain.RunnerTypeCodex
	run := &domain.Run{
		ID:                  uuid.New(),
		Status:              domain.RunStatusRunning,
		ExecutionMode:       domain.ExecutionModeInteractive,
		WebConsoleSessionID: res.SessionID,
		TranscriptPath:      res.TranscriptPath,
		ResolvedConfig:      cfg,
	}
	store := &fakeRunStore{}

	// Coordinator #1 tails the live session until we simulate the restart.
	sink1 := &collectSink{}
	coord1 := NewCoordinator(CoordinatorDeps{
		Tailer:      NewTailer(codecParserResolver, WithTailPollInterval(100*time.Millisecond)),
		Sessions:    client,
		Runs:        store,
		NewSink:     func(uuid.UUID) runner.EventSink { return sink1 },
		Debounce:    500 * time.Millisecond,
		SessionPoll: 500 * time.Millisecond,
		Heartbeat:   -1,
	})
	ctx1, cancel1 := context.WithCancel(ctx)
	done1 := make(chan struct{})
	go func() {
		defer close(done1)
		_, _ = coord1.TailToCompletion(ctx1, run)
	}()

	// Wait until the pre-restart output has been consumed, then "restart" by
	// cancelling coordinator #1 and waiting for it to stop mutating the run.
	waitFor(t, 8*time.Second, func() bool { return sink1.count(domain.EventTypeMessage) > 0 })
	cancel1()
	<-done1
	cursorAtRestart := run.TranscriptCursor
	if cursorAtRestart <= 0 {
		t.Fatalf("cursor did not advance before restart: %d", cursorAtRestart)
	}

	// Coordinator #2 is a fresh process: it reattaches from the persisted run row
	// only. It must resume from the cursor and complete once task_complete lands.
	sink2 := &collectSink{}
	coord2 := NewCoordinator(CoordinatorDeps{
		Tailer:      NewTailer(codecParserResolver, WithTailPollInterval(100*time.Millisecond)),
		Sessions:    client,
		Runs:        store,
		NewSink:     func(uuid.UUID) runner.EventSink { return sink2 },
		Debounce:    500 * time.Millisecond,
		SessionPoll: 500 * time.Millisecond,
		Heartbeat:   -1,
	})
	tctx, tcancel := context.WithTimeout(ctx, 20*time.Second)
	defer tcancel()
	term, err := coord2.TailToCompletion(tctx, run)
	if err != nil {
		t.Fatalf("reattached TailToCompletion: %v", err)
	}
	if term == nil || !term.Success {
		t.Fatalf("expected success terminal after reattach, got %+v", term)
	}
	if err := coord2.Finalize(tctx, run, term, err); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if run.Status != domain.RunStatusComplete {
		t.Fatalf("status = %s, want complete", run.Status)
	}
	// The reattach resumed from the cursor: it must not have re-consumed the
	// pre-restart output (cursor only advanced).
	if run.TranscriptCursor <= cursorAtRestart {
		t.Errorf("cursor did not advance across reattach: %d <= %d", run.TranscriptCursor, cursorAtRestart)
	}
	t.Logf("restart drill: cursor before=%d after=%d, reattach terminal.success=%v",
		cursorAtRestart, run.TranscriptCursor, term.Success)
}
