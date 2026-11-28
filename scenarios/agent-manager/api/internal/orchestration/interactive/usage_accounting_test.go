package interactive

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

// The interactive coordinator builds a fresh Tail (and parser) per turn from
// the persisted cursor. Before the one usage account, each mid-file parser
// counted the session's whole cumulative counter as its first delta, inflating
// interactive Codex runs by up to 17x (EVIDENCE E-5). This replays a trimmed
// real rollout through the real Tailer one turn at a time, re-tails the last
// turn as a restart would, and requires the account to equal the rollout's
// final counter.
func TestTailPerTurnFromCursorsCountsRolloutUsageOnce(t *testing.T) {
	lines := readCodecFixture(t, "codex_usage_rollout.jsonl")
	var session string
	var final int64
	for _, line := range lines {
		var record struct {
			Type    string `json:"type"`
			Payload struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Info *struct {
					Total struct {
						Total int64 `json:"total_tokens"`
					} `json:"total_token_usage"`
				} `json:"info"`
			} `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record.Type == "session_meta" {
			session = record.Payload.ID
		}
		if record.Payload.Type == "token_count" && record.Payload.Info != nil {
			final = record.Payload.Info.Total.Total
		}
	}
	// The harness writes one turn at a time; after each turn boundary the
	// coordinator starts a fresh Tail at the persisted cursor.
	var turns []string
	begin, closed := 0, false
	for index := 0; index <= len(lines); index++ {
		if index == len(lines) || (closed && strings.Contains(lines[index], `"type":"task_started"`)) {
			turns = append(turns, strings.Join(lines[begin:index], "\n")+"\n")
			begin, closed = index, false
		}
		if index < len(lines) && strings.Contains(lines[index], `"type":"task_complete"`) {
			closed = true
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout.jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	sink := &collectSink{}
	tailer := NewTailer(codecParserResolver, WithTailPollInterval(5*time.Millisecond))
	runID := uuid.New()
	tailFrom := func(start int64) int64 {
		cursor := start
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		terminal, err := tailer.Tail(ctx, TailParams{
			RunID: runID, RunnerType: domain.RunnerTypeCodex, TranscriptPath: path, RunDir: dir,
			SessionID: session, StartCursor: start, Sink: sink,
			OnAdvance: func(next, _ int64) error { cursor = next; return nil },
		})
		if err != nil || terminal == nil {
			t.Fatalf("tail from %d: terminal=%+v err=%v", start, terminal, err)
		}
		return cursor
	}
	var turnStarts []int64
	cursor := int64(0)
	for _, turn := range turns {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(turn); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
		turnStarts = append(turnStarts, cursor)
		cursor = tailFrom(cursor)
	}
	if len(turnStarts) < 3 {
		t.Fatalf("tailed %d turns, want several", len(turnStarts))
	}
	if got := runsignal.AccountUsage(sink.events).TotalTokens(); got != final {
		t.Fatalf("per-turn tails account = %d, want the rollout's final counter %d", got, final)
	}
	tailFrom(turnStarts[len(turnStarts)-1])
	if got := runsignal.AccountUsage(sink.events).TotalTokens(); got != final {
		t.Fatalf("after re-tailing the last turn account = %d, want %d", got, final)
	}
}
