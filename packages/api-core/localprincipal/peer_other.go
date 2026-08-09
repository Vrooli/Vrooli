//go:build !linux && !darwin

package localprincipal

import "net"

func peer(*net.UnixConn) (Principal, error) { return "", ErrUnsupported }

func current() (Principal, error) { return "", ErrUnsupported }
