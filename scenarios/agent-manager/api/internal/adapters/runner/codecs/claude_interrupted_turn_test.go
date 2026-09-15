package codecs

import (
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

const (
	claudeOpenTurnMode      = `{"type":"mode","mode":"normal","sessionId":"claude-session"}`
	claudeOpenTurnToolUse   = `{"type":"assistant","sessionId":"claude-session","message":{"model":"claude-fable-5","id":"msg_1","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file_path":"/x"}}],"stop_reason":"tool_use","usage":{"input_tokens":120,"cache_creation_input_tokens":30,"cache_read_input_tokens":5000,"output_tokens":40}}}`
	claudeOpenTurnResult    = `{"type":"user","sessionId":"claude-session","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}}`
	claudeOpenTurnEndOfTurn = `{"type":"assistant","sessionId":"claude-session","message":{"model":"claude-fable-5","id":"msg_2","type":"message","role":"assistant","content":[{"type":"text","text":"done"}],"stop_reason":"end_turn","usage":{"input_tokens":150,"cache_read_input_tokens":5100,"output_tokens":12}}}`
)

func parseClaudeLines(t *testing.T, lines ...string) (*claudeTranscriptParser, []*domain.UsageEventData) {
	t.Helper()
	parser, ok := NewClaudeForTest().NewTranscriptParser().(*claudeTranscriptParser)
	if !ok {
		t.Fatal("claude transcript parser type changed")
	}
	var usage []*domain.UsageEventData
	for _, line := range lines {
		result := parser.ParseTranscriptLine(uuid.New(), line)
		if result.Err != nil {
			t.Fatalf("parse %q: %v", line, result.Err)
		}
		for _, event := range result.Events {
			if data, ok := event.Data.(*domain.UsageEventData); ok {
				usage = append(usage, data)
			}
		}
	}
	return parser, usage
}

func TestClaudeFinalizesTurnKilledWhileAToolRan(t *testing.T) {
	parser, usage := parseClaudeLines(t, claudeOpenTurnMode, claudeOpenTurnToolUse)
	if len(usage) != 1 || usage[0].ReconciliationAuthority {
		t.Fatalf("live usage = %+v, want one non-authoritative reading", usage)
	}
	var finalizer runner.TranscriptInterruptedTurnFinalizer = parser
	at := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	events, ok := finalizer.FinalizeInterruptedTurn(uuid.New(), at)
	if !ok || len(events) == 0 {
		t.Fatal("a transcript ending on a completed response was not closed")
	}
	receipt, isUsage := events[0].Data.(*domain.UsageEventData)
	if !isUsage || !receipt.ReconciliationAuthority || receipt.ReconciliationSource != domain.ReconciliationSourceInterruptedTurn {
		t.Fatalf("receipt = %+v, want an authoritative interrupted-turn receipt", events[0].Data)
	}
	if receipt.InputTokens != usage[0].InputTokens || receipt.OutputTokens != usage[0].OutputTokens || receipt.CacheReadTokens != usage[0].CacheReadTokens || receipt.CacheCreationTokens != usage[0].CacheCreationTokens {
		t.Fatalf("receipt %+v does not carry the response's usage %+v", receipt, usage[0])
	}
	if !events[0].Timestamp.Equal(at) {
		t.Fatalf("receipt time = %v, want the run's end", events[0].Timestamp)
	}
	if usage[0].ReconciliationAuthority {
		t.Fatal("finalizing rewrote the live reading")
	}
}

func TestClaudeLeavesTurnUnknownWhenARequestMayBeOutstanding(t *testing.T) {
	for name, lines := range map[string][]string{
		"tool result sent":   {claudeOpenTurnMode, claudeOpenTurnToolUse, claudeOpenTurnResult},
		"turn already ended": {claudeOpenTurnMode, claudeOpenTurnToolUse, claudeOpenTurnResult, claudeOpenTurnEndOfTurn},
		"no response yet":    {claudeOpenTurnMode},
		"pipe dialect":       {strings.ReplaceAll(claudeOpenTurnToolUse, `"sessionId":"claude-session",`, "")},
	} {
		t.Run(name, func(t *testing.T) {
			parser, _ := parseClaudeLines(t, lines...)
			if events, ok := parser.FinalizeInterruptedTurn(uuid.New(), time.Now()); ok || len(events) != 0 {
				t.Fatalf("closed a turn it cannot prove complete: %+v", events)
			}
		})
	}
}
