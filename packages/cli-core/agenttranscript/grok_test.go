package agenttranscript

import "testing"

func TestParseGrokStreamEvent(t *testing.T) {
	tests := []struct {
		name string
		line string
		want GrokStreamEvent
		ok   bool
	}{
		{
			name: "text delta",
			line: ` {"type":"text","data":"hello","sessionId":"s1","requestId":"r1"} `,
			want: GrokStreamEvent{Type: "text", Data: "hello", SessionID: "s1", RequestID: "r1"},
			ok:   true,
		},
		{
			name: "terminal metadata",
			line: `{"type":"end","stopReason":"completed","sessionId":"s2"}`,
			want: GrokStreamEvent{Type: "end", StopReason: "completed", SessionID: "s2"},
			ok:   true,
		},
		{name: "blank", line: " \t"},
		{name: "banner", line: "Starting Grok"},
		{name: "malformed object", line: `{"type":`},
		{name: "array", line: `[]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseGrokStreamEvent([]byte(tt.line))
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("event=%+v want %+v", got, tt.want)
			}
		})
	}
}
