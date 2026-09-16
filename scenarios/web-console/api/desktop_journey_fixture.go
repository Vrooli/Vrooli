package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"web-console/internal/backend"
	consoleSession "web-console/session"
)

// desktopTerminalFixture is a loopback-only, metadata-only journey seam. It
// exercises the same session manager and PTY path as the UI, but never exposes
// terminal bytes beyond the fixed assertion result.
func (s *Server) desktopTerminalFixture(w http.ResponseWriter, r *http.Request) {
	if r.RemoteAddr != "" && !strings.HasPrefix(r.RemoteAddr, "127.0.0.1:") && !strings.HasPrefix(r.RemoteAddr, "[::1]:") {
		http.Error(w, "loopback only", http.StatusForbidden)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	sess, err := s.sessions.Create(ctx, "/bin/sh", 80, 24, backend.Standard, nil)
	if err != nil {
		http.Error(w, "terminal session unavailable", http.StatusServiceUnavailable)
		return
	}
	defer func() { _ = s.sessions.Delete(context.Background(), sess.ID) }()
	sub := sess.Subscribe("desktop-journey", "desktop journey", "validation")
	defer sess.Unsubscribe(sub.OutputCh)
	if err := sess.SendInput(consoleSession.InputText("printf 'desktop-terminal-fixture\\n'").AsPaste().WithSource("desktop-journey")); err != nil {
		http.Error(w, "terminal command unavailable", http.StatusServiceUnavailable)
		return
	}
	for {
		select {
		case data := <-sub.OutputCh:
			if strings.Contains(string(data), "desktop-terminal-fixture") {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"observed":"terminal_output=desktop-terminal-fixture","route":"bundled-private"}`))
				return
			}
		case <-ctx.Done():
			http.Error(w, "terminal output timed out", http.StatusGatewayTimeout)
			return
		}
	}
}
