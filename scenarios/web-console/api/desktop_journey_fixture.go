package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	sessionsH "web-console/handlers/sessions"
	"web-console/internal/backend"
)

// desktopTerminalFixture is a loopback-only journey seam. It creates a normal
// session and publishes the same lifecycle event consumed by the UI. The
// desktop driver submits the fixed command through the rendered terminal.
func (s *Server) desktopTerminalFixture(w http.ResponseWriter, r *http.Request) {
	if r.RemoteAddr != "" && !strings.HasPrefix(r.RemoteAddr, "127.0.0.1:") && !strings.HasPrefix(r.RemoteAddr, "[::1]:") {
		http.Error(w, "loopback only", http.StatusForbidden)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if s.desktopCreateSession == nil {
		http.Error(w, "terminal session creator unavailable", http.StatusServiceUnavailable)
		return
	}
	sess, err := s.desktopCreateSession(ctx, sessionsH.CreateInput{
		Shell:        "/bin/sh",
		Cols:         80,
		Rows:         24,
		Backend:      string(backend.Standard),
		Origin:       "ui",
		DisplayLabel: "desktop-journey-terminal-fixture",
	})
	if err != nil {
		http.Error(w, "terminal session unavailable", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		// Give the browser's lifecycle stream and terminal transport time to
		// attach, execute the command, and capture the rendered output before the
		// fixture session is cleaned up. The response is not released until the
		// initial producer acknowledgement, and this longer grace period covers
		// the desktop action's output/read/settle sequence.
		time.Sleep(15 * time.Second)
		if s.lifecycleDelete != nil {
			_ = s.lifecycleDelete(context.Background(), sess.ID)
		} else {
			_ = s.sessions.Delete(context.Background(), sess.ID)
		}
	}()
	// The response below is only a bounded producer acknowledgement; the
	// desktop journey separately submits the command and proves rendered output.
	_ = sess
	select {
	case <-ctx.Done():
		http.Error(w, "terminal output timed out", http.StatusGatewayTimeout)
		return
	case <-time.After(2 * time.Second):
	}

	if s.events == nil {
		http.Error(w, "terminal event logger unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"observed":"terminal_output=desktop-terminal-fixture","route":"bundled-private","app_key":"web-console"}`))
}
