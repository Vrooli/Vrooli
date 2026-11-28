package agenttranscript

import (
	"reflect"
	"testing"
)

func TestClaudeTextPartsPreservesNativeTextOnlyOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want []string
	}{
		{name: "legacy string", raw: `"hello"`, want: []string{"hello"}},
		{name: "content blocks", raw: `[{"type":"text","text":"first"},{"type":"tool_use","name":"Read"},{"type":"text","text":" second "},{"type":"text","text":""}]`, want: []string{"first", " second "}},
		{name: "malformed", raw: `{`, want: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClaudeTextParts([]byte(tc.raw)); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ClaudeTextParts() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestClaudeToolSelectorsUseSharedNativeContentDecoder(t *testing.T) {
	raw := []byte(`[{"type":"text","text":"before"},{"type":"tool_use","id":"call-1","name":"Read","input":{"path":"/workspace/a"}},{"type":"tool_result","tool_use_id":"call-1","content":"done","is_error":true}]`)
	if got, want := ClaudeToolUses(raw), []ClaudeToolBlock{{Type: "tool_use", ID: "call-1", Name: "Read", Input: []byte(`{"path":"/workspace/a"}`)}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ClaudeToolUses() = %#v, want %#v", got, want)
	}
	if got, want := ClaudeToolResults(raw), []ClaudeToolBlock{{Type: "tool_result", ToolUseID: "call-1", Content: "done", IsError: true}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ClaudeToolResults() = %#v, want %#v", got, want)
	}
	if got := ClaudeToolUses([]byte(`"plain text"`)); len(got) != 0 {
		t.Fatalf("ClaudeToolUses(string content) = %#v, want none", got)
	}
}
