package opencode

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2ClientProjectsSessionsAndMessages(t *testing.T) {
	var sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, password, ok := r.BasicAuth(); !ok || user != "opencode" || password != "secret" {
			t.Errorf("basic auth = %q/%q/%v", user, password, ok)
		}
		sawAuth = true
		switch r.URL.Path {
		case "/api/session":
			if r.URL.Query().Get("directory") != "/work" {
				t.Errorf("directory = %q", r.URL.Query().Get("directory"))
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"ses_a","title":"T","location":{"directory":"/work"},"time":{"created":10,"updated":20}}],"cursor":{"previous":null,"next":null}}`))
		case "/api/session/ses_a/message":
			if r.URL.Query().Get("order") != "asc" {
				t.Errorf("order = %q", r.URL.Query().Get("order"))
			}
			_, _ = w.Write([]byte(`{"data":[
				{"id":"msg_u","type":"user","text":"hi","time":{"created":11}},
				{"id":"msg_a","type":"assistant","time":{"created":12,"completed":13},"content":[{"type":"reasoning","text":"thinking"},{"type":"text","text":"hello"},{"type":"text","text":"there"}]}
			],"cursor":{"previous":null,"next":null}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewV2HTTPClient(server.URL, "secret")
	sessions, err := client.ListSessions(context.Background(), "/work")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "ses_a" || sessions[0].Directory != "/work" || sessions[0].Time.Created != 10 {
		t.Fatalf("sessions = %+v", sessions)
	}
	messages, err := client.SessionMessages(context.Background(), "ses_a")
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Info.Role != "user" || messages[0].Parts[0].Text != "hi" {
		t.Fatalf("messages = %+v", messages)
	}
	if messages[1].Info.Role != "assistant" || messages[1].Info.Time.Completed != 13 || len(messages[1].Parts) != 2 {
		t.Fatalf("assistant = %+v", messages[1])
	}
	if !sawAuth {
		t.Fatal("client did not authenticate")
	}
}

func TestV2ClientPermissionRevertStatusAndEvents(t *testing.T) {
	var got map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		switch r.URL.Path {
		case "/api/session/active":
			_, _ = w.Write([]byte(`{"data":{"ses_a":{"type":"running"},"ses_b":{"type":"idle"}}}`))
		case "/api/session/ses_a/interrupt":
			got = map[string]string{"method": r.Method, "resume": r.URL.Query().Get("resume")}
			_, _ = w.Write([]byte(`{}`))
		case "/api/session/ses_a/revert/stage":
			got = map[string]string{"method": r.Method, "messageID": payload["messageID"].(string)}
			_, _ = w.Write([]byte(`{"data":{}}`))
		case "/api/session/ses_a/permission/per_1/reply":
			got = map[string]string{"method": r.Method, "decision": payload["decision"].(string)}
			w.WriteHeader(http.StatusNoContent)
		case "/api/event":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"type\":\"session.execution.started\",\"data\":{\"sessionID\":\"ses_a\"}}\n\n"))
			_, _ = w.Write([]byte("data: {\"type\":\"permission.asked\",\"data\":{\"sessionID\":\"ses_a\",\"id\":\"per_1\",\"action\":\"bash\",\"resources\":[\"git push *\"]}}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewV2HTTPClient(server.URL, "")
	ctx := context.Background()

	status, err := client.SessionStatus(ctx)
	if err != nil || status["ses_a"].Type != "busy" {
		t.Fatalf("status = %+v, err=%v", status, err)
	}
	if err := client.AbortSession(ctx, "ses_a"); err != nil || got["resume"] != "false" {
		t.Fatalf("abort = %v %v", got, err)
	}
	if err := client.RevertMessage(ctx, "ses_a", "msg_1", ""); err != nil || got["messageID"] != "msg_1" {
		t.Fatalf("revert = %v %v", got, err)
	}
	if err := client.ReplyPermission(ctx, "ses_a", "per_1", "once"); err != nil || got["decision"] != "once" {
		t.Fatalf("reply = %v %v", got, err)
	}

	var events []Event
	if err := client.Events(ctx, func(event Event) {
		events = append(events, event)
		if len(events) == 2 {
			return
		}
	}); err != nil && !strings.Contains(err.Error(), "200") {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Type != "session.status" || events[0].SessionID() != "ses_a" {
		t.Fatalf("execution event = %+v", events[0])
	}
	if events[1].Type != "permission.asked" || events[1].Properties["permission"] != "bash" {
		t.Fatalf("permission event = %+v", events[1])
	}
}
