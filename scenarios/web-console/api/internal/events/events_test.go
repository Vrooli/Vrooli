package events

import "testing"

func TestLoggerHistoryAndSubscription(t *testing.T) {
	el := NewLogger(2)
	ch := make(chan Event, 1)
	unsub := el.Subscribe(ch)
	el.Emit(SessionCreated, "s1", map[string]string{"shell": "bash"})
	el.Emit(SessionConnected, "s1", nil)
	el.Emit(SessionDeleted, "s1", nil)
	if el.Count() != 2 || len(el.Recent(10)) != 2 || len(el.Recent(1)) != 1 {
		t.Fatalf("history count=%d recent=%d", el.Count(), len(el.Recent(10)))
	}
	select {
	case <-ch:
	default:
		t.Fatal("subscriber did not receive first event")
	}
	unsub()
	el.Emit(SessionCreated, "s2", nil)
	if len(el.Recent(0)) != 2 {
		t.Fatal("Recent(0) did not return history")
	}
}
