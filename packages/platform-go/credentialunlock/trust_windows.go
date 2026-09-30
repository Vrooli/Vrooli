//go:build windows

package credentialunlock

import (
	"errors"
	"net"
)

// Windows opens its store unattended through DPAPI; the socket hand-off is not
// used there, and without peer credentials it could not be authorized.
func trustedSocket(string) error {
	return errors.New("the credential unlock socket is not supported on Windows")
}

// PeerUID is unsupported on Windows.
func PeerUID(net.Conn) (int, error) {
	return -1, errors.New("peer credentials are not supported on Windows")
}
