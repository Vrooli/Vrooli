//go:build !linux && !darwin

package privsep

import (
	"fmt"
	"net"
)

func peerUID(net.Conn) (int, error) {
	return -1, fmt.Errorf("Unix peer credentials are not available on this build")
}
