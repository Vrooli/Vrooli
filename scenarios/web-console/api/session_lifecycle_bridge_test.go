package main

import (
	"testing"
	"time"

	"web-console/internal/events"
)

// recvEnvelope reads one envelope from a hub subscriber within d, reporting
// whether one arrived (false = nothing published in time).
func recvEnvelope(t *testing.T, sub *hubSubscriber, d time.Duration) (HubEnvelope, bool) {
	t.Helper()
	select {
	case env := <-sub.events:
		return env, true
	case <-time.After(d):
		return HubEnvelope{}, false
	}
}

func TestPublishSessionLifecycleEvent_CreatedCarriesProvenance(t *testing.T) {
	srv := &Server{hub: NewConversationHub()}
	sub, _, _ := srv.hub.Subscribe(0)

	ts := time.Date(2026, 7, 12, 1, 2, 3, 0, time.UTC)
	srv.publishSessionLifecycleEvent(events.Event{
		Type:      events.SessionCreated,
		SessionID: "s1",
		Timestamp: ts,
		Details: map[string]string{
			"shell":     "/bin/zsh",
			"cols":      "120",
			"rows":      "40",
			"backend":   "persistent",
			"origin":    "programmatic",
			"owner":     "cli",
			"label":     "nightly",
			"agent":     "claude",
			"recovered": "true",
		},
	})

	env, ok := recvEnvelope(t, sub, time.Second)
	if !ok {
		t.Fatal("expected a session_status envelope, got none")
	}
	if env.Kind != HubKindSessionStatus {
		t.Fatalf("kind = %q, want %q", env.Kind, HubKindSessionStatus)
	}
	if env.SessionID != "s1" {
		t.Fatalf("session id = %q, want s1", env.SessionID)
	}
	p, ok := env.Payload.(sessionStatusPayload)
	if !ok {
		t.Fatalf("payload type = %T, want sessionStatusPayload", env.Payload)
	}
	if p.Action != "created" {
		t.Fatalf("action = %q, want created", p.Action)
	}
	if p.Shell != "/bin/zsh" || p.Cols != 120 || p.Rows != 40 {
		t.Fatalf("shell/cols/rows = %q/%d/%d", p.Shell, p.Cols, p.Rows)
	}
	if p.Backend != "persistent" || p.Origin != "programmatic" || p.Owner != "cli" || p.DisplayLabel != "nightly" {
		t.Fatalf("provenance mismatch: %+v", p)
	}
	if p.Agent != "claude" || !p.Recovered {
		t.Fatalf("agent/recovered mismatch: agent=%q recovered=%v", p.Agent, p.Recovered)
	}
	if p.CreatedAt != ts.Format(time.RFC3339) {
		t.Fatalf("created_at = %q, want %q", p.CreatedAt, ts.Format(time.RFC3339))
	}
}

func TestPublishSessionLifecycleEvent_DeletedAndTerminated(t *testing.T) {
	srv := &Server{hub: NewConversationHub()}
	sub, _, _ := srv.hub.Subscribe(0)

	srv.publishSessionLifecycleEvent(events.Event{Type: events.SessionDeleted, SessionID: "d1"})
	env, ok := recvEnvelope(t, sub, time.Second)
	if !ok {
		t.Fatal("expected deleted envelope")
	}
	if p := env.Payload.(sessionStatusPayload); p.Action != "deleted" {
		t.Fatalf("action = %q, want deleted", p.Action)
	}

	srv.publishSessionLifecycleEvent(events.Event{
		Type:      events.SessionTerminated,
		SessionID: "t1",
		Details:   map[string]string{"reason": "expired"},
	})
	env, ok = recvEnvelope(t, sub, time.Second)
	if !ok {
		t.Fatal("expected terminated envelope")
	}
	p := env.Payload.(sessionStatusPayload)
	if p.Action != "terminated" || p.Reason != "expired" {
		t.Fatalf("terminated payload = %+v", p)
	}
}

func TestPublishSessionLifecycleEvent_PublishesDeviceStatus(t *testing.T) {
	srv := &Server{hub: NewConversationHub()}
	sub, _, _ := srv.hub.Subscribe(0)

	srv.publishSessionLifecycleEvent(events.Event{
		Type: events.SessionConnected, SessionID: "s1",
		Details: map[string]string{"deviceId": "phone-1", "connId": "c1"},
	})
	env, ok := recvEnvelope(t, sub, time.Second)
	if !ok {
		t.Fatal("connect event should publish a device_status envelope")
	}
	if env.Kind != HubKindDeviceStatus || env.SessionID != "s1" {
		t.Fatalf("unexpected envelope: kind=%q session=%q", env.Kind, env.SessionID)
	}
	details, ok := env.Payload.(map[string]string)
	if !ok || details["action"] != "connected" || details["deviceId"] != "phone-1" {
		t.Fatalf("device payload = %#v", env.Payload)
	}
}

func TestStartSessionLifecycleBridge_EndToEnd(t *testing.T) {
	srv := &Server{hub: NewConversationHub(), events: events.NewLogger(100)}
	sub, _, _ := srv.hub.Subscribe(0)
	srv.startSessionLifecycleBridge()

	// Emit through the SAME logger the app uses; the bridge must fan it to the hub.
	srv.events.Emit(events.SessionCreated, "e2e-1", map[string]string{
		"shell":   "/bin/bash",
		"backend": "standard",
		"origin":  "ui",
	})

	env, ok := recvEnvelope(t, sub, time.Second)
	if !ok {
		t.Fatal("bridge did not fan the created event onto the hub")
	}
	if env.Kind != HubKindSessionStatus || env.SessionID != "e2e-1" {
		t.Fatalf("unexpected envelope: kind=%q session=%q", env.Kind, env.SessionID)
	}
	if p := env.Payload.(sessionStatusPayload); p.Action != "created" || p.Origin != "ui" {
		t.Fatalf("payload = %+v", p)
	}
}
