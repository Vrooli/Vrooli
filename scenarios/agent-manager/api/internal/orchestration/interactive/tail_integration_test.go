//go:build integration

// Phase 3 live integration: drives Substrate.Launch against a LIVE web-console,
// then tails the discovered transcript with the real Tailer + codec parser. The
// "agent" is a small script that writes a codex-style rollout incrementally
// from inside the web-console PTY (proving the transcript is sourced from the
// agent-owned file, not web-console conversation events), and the Tailer must
// surface the same structured events + terminal a pipe run produces.
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

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestLive_Tailer_ConsumesInteractiveTranscript(t *testing.T) {
	client := liveClient(t)

	workDir := t.TempDir()
	runDir := t.TempDir()

	// A fake codex CLI writing its rollout incrementally into $CODEX_HOME (set
	// by the substrate's env prefix), then dropping to an interactive shell —
	// exactly the live-session shape a real codex TUI produces.
	script := filepath.Join(t.TempDir(), "fake-codex-live.sh")
	scriptBody := `#!/bin/sh
set -e
d="$CODEX_HOME/sessions/2026/07/13"
mkdir -p "$d"
f="$d/rollout-2026-07-13T10-00-00-livep3.jsonl"
printf '%s\n' '{"type":"session_meta","payload":{"id":"live-p3"}}' > "$f"
sleep 0.4
printf '%s\n' '{"type":"response_item","payload":{"type":"function_call","name":"exec","arguments":"{\"cmd\":\"ls\"}","call_id":"c1"}}' >> "$f"
printf '%s\n' '{"type":"response_item","payload":{"type":"function_call_output","call_id":"c1","output":"file.txt"}}' >> "$f"
sleep 0.4
printf '%s\n' '{"type":"event_msg","payload":{"type":"agent_message","message":"done listing"}}' >> "$f"
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
		Tag:        "phase3-live-tail",
		WorkingDir: workDir,
		RunDir:     runDir,
	})
	if res.SessionID != "" {
		t.Cleanup(func() { _ = sub.Stop(context.Background(), res.SessionID, "agent-manager:run-test") })
	}
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Tail the agent-owned transcript the substrate discovered.
	sink := &collectSink{}
	tailer := NewTailer(codecParserResolver, WithTailPollInterval(50*time.Millisecond))
	tctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	term, err := tailer.Tail(tctx, TailParams{
		RunID:          uuid.New(),
		RunnerType:     domain.RunnerTypeCodex,
		TranscriptPath: res.TranscriptPath,
		RunDir:         runDir,
		Sink:           sink,
	})
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if term == nil || !term.Success {
		t.Fatalf("expected success terminal from live interactive transcript, got %+v", term)
	}
	if sink.count(domain.EventTypeMessage) == 0 {
		t.Errorf("no assistant message from live transcript")
	}
	if sink.count(domain.EventTypeToolCall) == 0 || sink.count(domain.EventTypeToolResult) == 0 {
		t.Errorf("no tool events from live transcript (calls=%d results=%d)",
			sink.count(domain.EventTypeToolCall), sink.count(domain.EventTypeToolResult))
	}
	t.Logf("live tail: messages=%d tool_calls=%d tool_results=%d terminal.success=%v",
		sink.count(domain.EventTypeMessage), sink.count(domain.EventTypeToolCall),
		sink.count(domain.EventTypeToolResult), term.Success)
}
