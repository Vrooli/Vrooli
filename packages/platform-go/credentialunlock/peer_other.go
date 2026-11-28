//go:build !linux && !darwin && !windows

package credentialunlock

import (
	"errors"
	"net"
)

// PeerUID is unsupported off Linux and Darwin, so Serve refuses every caller.
func PeerUID(net.Conn) (int, error) {
	return -1, errors.New("peer credentials are not supported on this platform")
}
