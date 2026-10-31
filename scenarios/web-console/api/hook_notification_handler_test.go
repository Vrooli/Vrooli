package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"web-console/session"
)

// [REQ:P0-017i] Claude's Notification hook drives the waiting state.
func TestNotificationHookKind(t *testing.T) {
	cases := []struct {
		name string
		req  hookNotificationRequest
		want string
	}{
		{"permission dialog", hookNotificationRequest{NotificationType: "permission_prompt", Message: "Claude needs your permission to use Bash"}, session.HookWaiting},
		{"question dialog", hookNotificationRequest{NotificationType: "elicitation_dialog"}, session.HookWaiting},
		{"idle prompt", hookNotificationRequest{NotificationType: "idle_prompt", Message: "Claude is waiting for your input"}, session.HookIdle},
		{"auth success is only output", hookNotificationRequest{NotificationType: "auth_success"}, session.HookOutput},
		{"legacy permission message", hookNotificationRequest{Message: "Claude needs your permission to use Write"}, session.HookWaiting},
		{"legacy idle message", hookNotificationRequest{Message: "Claude is waiting for your input"}, session.HookIdle},
		{"unrecognized message", hookNotificationRequest{Message: "Something happened"}, session.HookOutput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := notificationHookKind(tc.req); got != tc.want {
				t.Fatalf("notificationHookKind(%+v) = %q, want %q", tc.req, got, tc.want)
			}
		})
	}
}

func TestHandleHookNotification_RejectsMissingToken(t *testing.T) {
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hooks/notification", strings.NewReader(`{"notification_type":"permission_prompt"}`))
	rec := httptest.NewRecorder()
	srv.handleHookNotification(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandleHookNotification_UnknownSessionIsNotRouted(t *testing.T) {
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hooks/notification", strings.NewReader(`{"notification_type":"permission_prompt","web_console_session_id":"missing"}`))
	req.Header.Set("X-Hook-Token", "secret")
	rec := httptest.NewRecorder()
	srv.handleHookNotification(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Kind   string `json:"kind"`
		Routed bool   `json:"routed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Kind != session.HookWaiting || body.Routed {
		t.Fatalf("body = %+v, want kind waiting and not routed", body)
	}
}
