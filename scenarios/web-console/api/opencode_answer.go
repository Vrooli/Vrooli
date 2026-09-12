package main

import (
	"context"
	"errors"
	"fmt"
)

// setPendingPermission records (or, with "", clears) the OpenCode permission
// request a pane is waiting on: the request an answer from Messages replies to.
func (w *OpenCodeWatcher) setPendingPermission(sessionID, requestID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if requestID == "" {
		delete(w.pendingPermission, sessionID)
		return
	}
	if w.pendingPermission == nil {
		w.pendingPermission = make(map[string]string)
	}
	w.pendingPermission[sessionID] = requestID
}

// ReplyPermission answers the OpenCode permission request the pane is waiting
// on with "once", "always", or "reject".
func (w *OpenCodeWatcher) ReplyPermission(ctx context.Context, sessionID, reply string) error {
	w.mu.Lock()
	requestID := w.pendingPermission[sessionID]
	client := w.client
	w.mu.Unlock()
	if requestID == "" {
		return fmt.Errorf("no OpenCode permission request is pending for session %s", sessionID)
	}
	if client == nil {
		return errors.New("the OpenCode server is not connected")
	}
	return client.ReplyPermission(ctx, requestID, reply)
}

// opencodeReplier answers OpenCode permissions for the terminal handler. The
// watcher starts after the routes are mounted, so it is looked up per answer.
type opencodeReplier struct {
	server *Server
}

func (r opencodeReplier) ReplyPermission(ctx context.Context, sessionID, reply string) error {
	if r.server == nil || r.server.opencodeWatcher == nil {
		return errors.New("OpenCode capture is not running")
	}
	return r.server.opencodeWatcher.ReplyPermission(ctx, sessionID, reply)
}
