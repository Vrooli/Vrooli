package codex

import (
	"encoding/json"
	"testing"

	"github.com/vrooli/cli-core/agenttranscript"
)

func TestSharedCodexCodecRetainsVoiceHandoffReceipt(t *testing.T) {
	for _, tc := range []struct {
		text    string
		handoff bool
		display string
	}{
		{`<realtime_delegation><input>Run tests</input></realtime_delegation>`, true, "Run tests"},
		{"Run tests", false, "Run tests"},
		{`<realtime_delegation><source>transcript_tail_flush</source><input>Tail</input></realtime_delegation>`, false, ""},
	} {
		line, err := json.Marshal(map[string]any{
			"type": "response_item",
			"payload": map[string]any{
				"role":    "user",
				"content": []map[string]string{{"type": "input_text", "text": tc.text}},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		messages := agenttranscript.ParseMessages(agenttranscript.HarnessCodex, line)
		if len(messages) != 1 || messages[0].Role != "user" {
			t.Fatalf("shared codec messages = %#v", messages)
		}
		input := ParseUserInput(messages[0].Text)
		if input.Text != tc.display || input.VoiceHandoff != tc.handoff {
			t.Fatalf("input = %+v, want %q handoff=%v", input, tc.display, tc.handoff)
		}
	}
}
