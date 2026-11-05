//go:build !linux && !darwin

package localprincipal

import "net"

func peer(*net.UnixConn) (Principal, error) { return "", ErrUnsupported }

func current() (Principal, error) { return "", ErrUnsupported }

// PeerPID is unavailable until this platform has a verified process credential adapter.
func PeerPID(*net.UnixConn) (uint32, error) { return 0, ErrUnsupported }
