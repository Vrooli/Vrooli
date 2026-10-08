package codecs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

const interruptedSession = "01a116b2-513b-7492-bb3f-bcbb2f784da8"

// writeInterruptedRollout writes a run-scoped rollout whose first turn closed
// before the invocation under test, followed by that invocation's turn lines.
func writeInterruptedRollout(t *testing.T, turn ...string) string {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, "sessions", "2026", "10", "07")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	lines := append([]string{
		`{"timestamp":"2026-10-07T14:00:00.000Z","type":"session_meta","payload":{"id":"` + interruptedSession + `"}}`,
		`{"timestamp":"2026-10-07T14:00:01.000Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-1"}}`,
		`{"timestamp":"2026-10-07T14:00:02.000Z","type":"token_usage_record","payload":{"turn_id":"turn-1","turn_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":10}}}`,
		`{"timestamp":"2026-10-07T14:00:02.100Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":10}}}}`,
		`{"timestamp":"2026-10-07T14:00:03.000Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-1"}}`,
	}, turn...)
	path := filepath.Join(dir, "rollout-2026-10-07T14-00-00-"+interruptedSession+".jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

var (
	turnTwoStart  = `{"timestamp":"2026-10-07T14:30:00.500Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-2"}}`
	turnTwoPrompt = `{"timestamp":"2026-10-07T14:30:01.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"wake"}]}}`
	turnTwoCall   = `{"timestamp":"2026-10-07T14:30:05.000Z","type":"response_item","payload":{"type":"custom_tool_call","name":"exec","call_id":"c1","input":"agent-manager run park"}}`
	turnTwoRecord = `{"timestamp":"2026-10-07T14:30:05.010Z","type":"token_usage_record","payload":{"turn_id":"turn-2","turn_token_usage":{"input_tokens":300,"cached_input_tokens":200,"output_tokens":30}}}`
	turnTwoOutput = `{"timestamp":"2026-10-07T14:30:06.000Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"c1","output":"PARKED"}}`
	turnTwoCount  = `{"timestamp":"2026-10-07T14:30:06.010Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":400,"cached_input_tokens":240,"output_tokens":40}}}}`
	turnTwoDone   = `{"timestamp":"2026-10-07T14:30:09.000Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-2"}}`
)

func TestCodexRecoversInterruptedTurnUsageFromRollout(t *testing.T) {
	started := time.Date(2026, 10, 7, 14, 30, 0, 0, time.UTC)
	subscription := domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription, Mode: domain.BillingModeSubscription}
	for _, tc := range []struct {
		name      string
		turn      []string
		authority bool
		source    string
	}{
		{
			// A park ends the process while the model answers the park
			// command's output: only the completed calls are known.
			name: "request in flight is a lower bound",
			turn: []string{turnTwoStart, turnTwoPrompt, turnTwoCall, turnTwoRecord, turnTwoOutput, turnTwoCount},
		},
		{
			name:      "stopped while a tool ran has no outstanding request",
			turn:      []string{turnTwoStart, turnTwoPrompt, turnTwoCall, turnTwoRecord},
			authority: true,
			source:    domain.ReconciliationSourceInterruptedTurn,
		},
		{
			name:      "turn closed in the rollout",
			turn:      []string{turnTwoStart, turnTwoPrompt, turnTwoCall, turnTwoRecord, turnTwoOutput, turnTwoCount, turnTwoDone},
			authority: true,
			source:    domain.ReconciliationSourceTranscriptRecovery,
		},
		{
			// Older rollouts have no token_usage_record; the thread total
			// since the previous turn is the fallback, never authoritative.
			name: "token_count fallback",
			turn: []string{turnTwoStart, turnTwoPrompt, turnTwoCall, turnTwoOutput, turnTwoCount},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := writeInterruptedRollout(t, tc.turn...)
			runID := uuid.New()
			events, note := NewCodexForTest().RecoverInterruptedUsage(InterruptedUsageRequest{
				RunID: runID, SessionID: interruptedSession, Env: []string{"PATH=/usr/bin", "CODEX_HOME=" + home},
				StartedAt: started, EndedAt: started.Add(time.Minute), Model: "gpt-6-luna", Billing: subscription,
			})
			if note == "" {
				t.Fatal("recovery must explain itself on the run timeline")
			}
			var usage *domain.UsageEventData
			charges := 0
			for _, event := range events {
				switch data := event.Data.(type) {
				case *domain.UsageEventData:
					if usage != nil {
						t.Fatalf("expected one usage event, got %+v", events)
					}
					usage = data
				case *domain.ChargeEventData:
					charges++
				}
				if event.RunID != runID {
					t.Fatalf("event run = %s, want %s", event.RunID, runID)
				}
			}
			if usage == nil || charges != 1 {
				t.Fatalf("usage=%+v charges=%d", usage, charges)
			}
			// Only this invocation's turn counts; turn 1 belonged to an earlier one.
			if usage.InputTokens != 100 || usage.CacheReadTokens != 200 || usage.OutputTokens != 30 || usage.Model != "gpt-6-luna" {
				t.Fatalf("recovered usage = %+v, want input 100 cache 200 output 30", usage)
			}
			if usage.ReconciliationAuthority != tc.authority || usage.ReconciliationSource != tc.source {
				t.Fatalf("authority=%v source=%q, want %v %q", usage.ReconciliationAuthority, usage.ReconciliationSource, tc.authority, tc.source)
			}
			if tc.authority && (usage.Charge == nil || usage.Charge.AmountMicroUSD == nil || *usage.Charge.AmountMicroUSD != 0) {
				t.Fatalf("authoritative receipt must carry its subscription charge: %+v", usage.Charge)
			}
		})
	}
}

func TestCodexInterruptedUsageNeverRecountsEarlierTurns(t *testing.T) {
	home := writeInterruptedRollout(t)
	for name, req := range map[string]InterruptedUsageRequest{
		"no turn since the invocation started": {SessionID: interruptedSession, Env: []string{"CODEX_HOME=" + home}, StartedAt: time.Date(2026, 10, 7, 14, 30, 0, 0, time.UTC)},
		"no session home":                      {SessionID: interruptedSession, StartedAt: time.Date(2026, 10, 7, 13, 0, 0, 0, time.UTC)},
		"another session":                      {SessionID: uuid.NewString(), Env: []string{"CODEX_HOME=" + home}, StartedAt: time.Date(2026, 10, 7, 13, 0, 0, 0, time.UTC)},
	} {
		t.Run(name, func(t *testing.T) {
			req.RunID = uuid.New()
			if events, _ := NewCodexForTest().RecoverInterruptedUsage(req); len(events) != 0 {
				t.Fatalf("recovered usage that belongs to no stopped turn: %+v", events)
			}
		})
	}
}
