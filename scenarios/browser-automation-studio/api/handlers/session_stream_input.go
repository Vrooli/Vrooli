package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/vrooli/browser-automation-studio/websocket"
)

// CreateInputStateReleaser returns an owner-only cleanup hook for stream
// control receipt rotation/revocation. It resolves the exact active session
// and lease before sending release-only events through the existing driver.
func (h *Handler) CreateInputStateReleaser() websocket.InputStateReleaser {
	return func(sessionID, executionID, leaseID string) error {
		if h == nil || h.sessionBroker == nil {
			return errors.New("BAS session owner unavailable for input release")
		}
		session, ok := h.sessionBroker.Get(sessionID)
		if !ok || session == nil {
			return errors.New("BAS stream session no longer exists")
		}
		actualExecution, actualLease := session.OwnerIdentity()
		if actualExecution != executionID || actualLease != leaseID {
			return errors.New("stream revoke lease does not match the current BAS session")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return session.ReleaseInput(ctx)
	}
}
