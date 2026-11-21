package main

import (
	"net/http"
	"strings"

	"web-console/session"
)

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#session-activity-contract

// hookNotificationRequest mirrors the Claude Code Notification-hook payload.
// notification_type arrived in later 2.x releases; older payloads carry only
// the message, so the message is the fallback signal.
type hookNotificationRequest struct {
	Message             string `json:"message"`
	NotificationType    string `json:"notification_type"`
	HookEventName       string `json:"hook_event_name"`
	SessionIDSnake      string `json:"session_id"`
	WebConsoleSessionID string `json:"web_console_session_id"`
}

// notificationHookKind maps a Claude notification onto an activity hook kind:
// a permission or question dialog means the agent is waiting on the user; the
// idle notification means the turn is over; anything else is only output.
func notificationHookKind(req hookNotificationRequest) string {
	switch strings.ToLower(strings.TrimSpace(req.NotificationType)) {
	case "permission_prompt", "elicitation_dialog":
		return session.HookWaiting
	case "idle_prompt":
		return session.HookIdle
	case "":
	default:
		return session.HookOutput
	}
	message := strings.ToLower(req.Message)
	switch {
	case strings.Contains(message, "needs your permission"), strings.Contains(message, "needs your approval"):
		return session.HookWaiting
	case strings.Contains(message, "waiting for your input"):
		return session.HookIdle
	default:
		return session.HookOutput
	}
}

// handleHookNotification is the Claude Code Notification-hook receiver. It
// feeds the session's activity detector so a permission prompt reads as
// "waiting on you" the moment Claude raises it.
// POST /api/v1/hooks/notification
func (s *Server) handleHookNotification(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Hook-Token")
	if token == "" || token != s.hookAuthToken {
		writeCatalogError(w, "unauthorized", "Invalid or missing hook token")
		return
	}
	var req hookNotificationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	kind := notificationHookKind(req)
	detector := s.activityDetector(req.WebConsoleSessionID)
	if detector != nil {
		detector.OnHook(kind)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "kind": kind, "routed": detector != nil})
}
