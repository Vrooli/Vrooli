package codex

import "testing"

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
