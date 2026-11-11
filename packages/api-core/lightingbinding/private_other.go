//go:build !linux

package lightingbinding

import (
	"errors"
	"net"
)

func Listen(string, Identity) (net.Listener, error) {
	return nil, errors.New("private native lighting sockets are unsupported on this platform")
}
