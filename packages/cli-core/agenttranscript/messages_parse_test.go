package agenttranscript

import "testing"

func TestClaudeMessagesSkipMalformedToolUseButKeepSelectableCalls(t *testing.T) {
	malformed := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use"}]}}`)
	if got := ParseMessages(HarnessClaude, malformed); len(got) != 0 {
		t.Fatalf("malformed Claude tool_use = %#v, want no selectable message", got)
	}

	valid := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"call-1","name":"Read","input":{"path":"/tmp/a"}}]}}`)
	got := ParseMessages(HarnessClaude, valid)
	if len(got) != 1 || got[0].Role != "assistant" || got[0].Text == "" {
		t.Fatalf("valid Claude tool_use = %#v, want one selectable tool message", got)
	}
}

func TestCodexResponseItemsUseRoleSpecificTextTypes(t *testing.T) {
	for _, tc := range []struct {
		name, role, kind string
		want             string
	}{
		{"user input", "user", "input_text", "question"},
		{"user plain text", "user", "text", "question"},
		{"user rejects assistant output", "user", "output_text", ""},
		{"assistant output", "assistant", "output_text", "answer"},
		{"assistant plain text", "assistant", "text", "answer"},
		{"assistant rejects user input", "assistant", "input_text", ""},
		{"assistant rejects tool block", "assistant", "tool_call", ""},
	} {
		line := []byte(`{"type":"response_item","payload":{"role":"` + tc.role + `","content":[{"type":"` + tc.kind + `","text":"` + tc.want + `"}]}}`)
		got := ParseMessages(HarnessCodex, line)
		if tc.want == "" {
			if len(got) != 0 {
				t.Errorf("%s: ParseMessages = %#v, want no message", tc.name, got)
			}
			continue
		}
		if len(got) != 1 || got[0].Role != tc.role || got[0].Text != tc.want {
			t.Errorf("%s: ParseMessages = %#v", tc.name, got)
		}
	}
}
