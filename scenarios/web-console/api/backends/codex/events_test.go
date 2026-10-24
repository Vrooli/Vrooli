package codex

import (
	"encoding/json"
	"testing"
)

func TestMapNotificationPreservesNativeIdentity(t *testing.T) {
	event, ok := MapNotification(Message{Method: "item/completed", Params: []byte(`{"thread":{"id":"thread-1"},"turn":{"id":"turn-2"},"item":{"id":"item-3","type":"agentMessage","text":"done"}}`)})
	if !ok {
		t.Fatal("notification was not mapped")
	}
	if event.ThreadID != "thread-1" || event.TurnID != "turn-2" || event.ItemID != "item-3" || event.Text != "done" {
		t.Fatalf("event identity = %+v", event)
	}
	if !event.Controllable || event.BoundaryID != "item-3" {
		t.Fatalf("event control metadata = %+v", event)
	}
}

func TestMapNotificationIgnoresMalformedAndUnknownShape(t *testing.T) {
	if _, ok := MapNotification(Message{Method: "item/completed", Params: []byte("not-json")}); ok {
		t.Fatal("malformed notification was accepted")
	}
	if _, ok := MapNotification(Message{Method: "thread/started", Params: []byte(`{"thread":{"id":"thread-1"}}`)}); !ok {
		t.Fatal("lifecycle notification should remain observable")
	}
	if _, ok := MapNotification(Message{Method: "provider/private", Params: []byte(`{"text":"ignore"}`)}); ok {
		t.Fatal("unknown notification was accepted")
	}
}

func TestMapNotificationReadsUserInputContentArray(t *testing.T) {
	event, ok := MapNotification(Message{Method: "item/completed", Params: []byte(`{"threadId":"thread-1","turnId":"turn-1","item":{"id":"item-1","type":"userMessage","content":[{"type":"text","text":"hello"}]}}`)})
	if !ok || event.Text != "hello" || event.Role != "user" {
		t.Fatalf("event = %+v, ok=%v", event, ok)
	}
}

func TestDisplayVoiceDelegationWithoutProviderScaffolding(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{`<realtime_delegation><input>Inspect &lt;main&gt; &amp; tests</input><transcript_delta>private context</transcript_delta></realtime_delegation>`, `Inspect <main> & tests`},
		{`<realtime_delegation><source>transcript_tail_flush</source><input>Internal acknowledgement</input></realtime_delegation>`, ""},
		{`<realtime_delegation>user-authored malformed text</realtime_delegation>`, `<realtime_delegation>user-authored malformed text</realtime_delegation>`},
		{"Ordinary user message", "Ordinary user message"},
	} {
		if got := DisplayUserText(tc.input); got != tc.want {
			t.Fatalf("got %q, want %q", got, tc.want)
		}
	}
}

func TestVoiceHandoffRequiresCodingUserDelegation(t *testing.T) {
	for _, tc := range []struct {
		name, role, text string
		want             bool
	}{
		{"delegated input", "userMessage", `<realtime_delegation><input>Inspect the tests</input><transcript_delta>context</transcript_delta></realtime_delegation>`, true},
		{"ordinary coding request", "userMessage", "Inspect the tests", false},
		{"ordinary transcript", "userMessage", "Sent to coding agent", false},
		{"assistant quotation", "agentMessage", `<realtime_delegation><input>Inspect the tests</input></realtime_delegation>`, false},
		{"malformed envelope", "userMessage", `<realtime_delegation><input>incomplete`, false},
		{"empty envelope", "userMessage", `<realtime_delegation><input> </input></realtime_delegation>`, false},
		{"synthetic tail", "userMessage", `<realtime_delegation><source>transcript_tail_flush</source><input>Internal acknowledgement</input></realtime_delegation>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params, err := json.Marshal(map[string]any{"threadId": "thread", "turnId": "turn", "item": map[string]any{"id": "item", "type": tc.role, "text": tc.text}})
			if err != nil {
				t.Fatal(err)
			}
			event, ok := MapNotification(Message{Method: "item/completed", Params: params})
			if !ok || event.VoiceHandoff != tc.want {
				t.Fatalf("handoff = %v, want %v: %+v", event.VoiceHandoff, tc.want, event)
			}
		})
	}
}
