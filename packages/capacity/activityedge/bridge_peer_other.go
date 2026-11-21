//go:build !linux

package activityedge

import (
	"fmt"
	"net"
)

func unixPeerCredentials(net.Conn) (int, int, error) {
	return 0, 0, fmt.Errorf("upstream bridge is supported only on Linux")
}
