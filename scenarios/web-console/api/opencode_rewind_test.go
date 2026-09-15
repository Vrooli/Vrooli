package main

import (
	"context"
	"testing"

	"web-console/backends/opencode"
)

type fakeRewindClient struct {
	status  map[string]opencode.SessionStatus
	seen    *opencode.Session
	message string
}

func (f *fakeRewindClient) ListSessions(context.Context) ([]opencode.Session, error) {
	if f.seen == nil {
		return nil, nil
	}
	return []opencode.Session{*f.seen}, nil
}
func (f *fakeRewindClient) SessionMessages(context.Context, string) ([]opencode.MessageWithParts, error) {
	return nil, nil
}
func (f *fakeRewindClient) SessionStatus(context.Context) (map[string]opencode.SessionStatus, error) {
	return f.status, nil
}
func (f *fakeRewindClient) AbortSession(context.Context, string) error { return nil }
func (f *fakeRewindClient) RevertMessage(_ context.Context, sessionID, messageID, partID string) error {
	f.message = sessionID + ":" + messageID + ":" + partID
	f.seen = &opencode.Session{ID: sessionID, Revert: &struct {
		MessageID string `json:"messageID"`
		PartID    string `json:"partID"`
	}{MessageID: messageID, PartID: partID}}
	return nil
}
func (f *fakeRewindClient) Events(context.Context, func(opencode.Event)) error    { return nil }
func (f *fakeRewindClient) ReplyPermission(context.Context, string, string) error { return nil }

func TestOpenCodeRewindAdapterVerifiesNativeTarget(t *testing.T) {
	client := &fakeRewindClient{status: map[string]opencode.SessionStatus{"oc-1": {Type: "idle"}}}
	adapter := openCodeRewindAdapter{client: client}
	event := ConversationEvent{NativeProvenance: &NativeProvenance{Provider: "opencode", SessionID: "oc-1", MessageID: "msg-1", BoundaryID: "part-1"}}
	preflight, err := adapter.Preflight(context.Background(), event, ConversationControlPreflight{Capability: conversationCapabilities("opencode")})
	if err != nil || preflight.Decision != ControlSupported || preflight.ConfirmationKey == "" {
		t.Fatalf("preflight = %+v, err = %v", preflight, err)
	}
	target, err := adapter.Execute(context.Background(), event, true)
	if err != nil || target != "msg-1" || client.message != "oc-1:msg-1:part-1" {
		t.Fatalf("execute target=%q message=%q err=%v", target, client.message, err)
	}
	verified, err := adapter.Verify(context.Background(), event, target)
	if err != nil || !verified {
		t.Fatalf("verified=%v err=%v", verified, err)
	}
}

func TestOpenCodeRewindAdapterPreparesBusySession(t *testing.T) {
	adapter := openCodeRewindAdapter{client: &fakeRewindClient{status: map[string]opencode.SessionStatus{"oc-1": {Type: "busy"}}}}
	preflight, err := adapter.Preflight(context.Background(), ConversationEvent{NativeProvenance: &NativeProvenance{SessionID: "oc-1", MessageID: "msg-1"}}, ConversationControlPreflight{Capability: conversationCapabilities("opencode")})
	if err != nil || preflight.Decision != ControlSupported {
		t.Fatalf("preflight = %+v, err = %v", preflight, err)
	}
}

func TestOpenCodeRewindAdapterRefusesCompactionLineage(t *testing.T) {
	adapter := openCodeRewindAdapter{client: &fakeRewindClient{status: map[string]opencode.SessionStatus{"oc-1": {Type: "idle"}}}}
	preflight, err := adapter.Preflight(context.Background(), ConversationEvent{NativeProvenance: &NativeProvenance{
		SessionID:         "oc-1",
		MessageID:         "msg-1",
		CompactionLineage: "compacted-from-turn-3",
	}}, ConversationControlPreflight{Capability: conversationCapabilities("opencode")})
	if err != nil {
		t.Fatalf("preflight error = %v", err)
	}
	if preflight.Decision != ControlUnavailable || preflight.Capability.Available {
		t.Fatalf("preflight = %+v, want unavailable and unavailable capability", preflight)
	}
}
