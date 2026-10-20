package hostapp

import (
	"context"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// InspectDesktopSession observes one desktop session and its authenticated
// peer. Argument parsing and rendering belong to the CLI handler package.
func (Service) InspectDesktopSession(ctx context.Context, sessionID string, peerPID int) (hostinventory.DesktopSessionFacts, error) {
	if ctx == nil {
		return hostinventory.DesktopSessionFacts{}, context.Canceled
	}
	return hostinventory.SystemCollector().InspectDesktopSession(ctx, sessionID, peerPID)
}

// InspectDesktopSessionDisplay obtains the local X-server peer for an explicit
// display and verifies it against the requested logind session.
func (Service) InspectDesktopSessionDisplay(ctx context.Context, sessionID string, display uint16) (hostinventory.DesktopSessionFacts, error) {
	if ctx == nil {
		return hostinventory.DesktopSessionFacts{}, context.Canceled
	}
	return hostinventory.SystemCollector().InspectDesktopSessionOnDisplay(ctx, sessionID, display)
}
