package platform

import (
	"errors"
	"net"
)

// ErrPeerProcessIdentityUnavailable means the connection transport does not
// expose a trustworthy operating-system process identity.
var ErrPeerProcessIdentityUnavailable = errors.New("platform: peer process identity is unavailable")

// PeerProcessID returns the kernel-authenticated PID on the other end of a
// local IPC connection. Network connections and unsupported platforms fail
// closed; caller-supplied PID headers are never an equivalent.
func PeerProcessID(conn net.Conn) (int, error) { return peerProcessID(conn) }
