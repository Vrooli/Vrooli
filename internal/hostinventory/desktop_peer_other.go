//go:build !linux

package hostinventory

import (
	"context"
	"fmt"
)

type desktopPeer struct {
	PID int
	UID uint32
}

func desktopPeerForDisplay(context.Context, uint16) (desktopPeer, error) {
	return desktopPeer{}, fmt.Errorf("desktop display peer inspection is unsupported on this platform")
}
