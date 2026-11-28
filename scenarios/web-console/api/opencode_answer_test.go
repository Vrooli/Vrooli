package main

import (
	"context"
	"testing"

	"web-console/backends/opencode"
)

func TestOpenCodeWatcherRepliesToThePendingPermission(t *testing.T) {
	// [REQ:P0-017i] An answer for an OpenCode pane goes to the permission
	// request that pane is waiting on, and only while it is pending.
	client := &fakeOpenCodeClient{}
	w := &OpenCodeWatcher{server: &Server{}, claimed: map[string]string{"ses_a": "wc1"}, client: client}
	if err := w.ReplyPermission(context.Background(), "wc1", "once"); err == nil {
		t.Fatal("a reply with nothing pending succeeded")
	}
	w.applyActivity(opencode.Event{Type: "permission.asked", Properties: map[string]any{"id": "per_1", "sessionID": "ses_a", "permission": "bash"}})
	if err := w.ReplyPermission(context.Background(), "wc1", "once"); err != nil {
		t.Fatal(err)
	}
	if len(client.replies) != 1 || client.replies[0] != "per_1:once" {
		t.Fatalf("replies = %v, want per_1:once", client.replies)
	}
	w.applyActivity(opencode.Event{Type: "permission.replied", Properties: map[string]any{"sessionID": "ses_a", "requestID": "per_1", "reply": "once"}})
	if err := w.ReplyPermission(context.Background(), "wc1", "once"); err == nil {
		t.Fatal("a reply after the permission resolved succeeded")
	}
}
