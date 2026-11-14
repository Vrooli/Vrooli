//go:build !linux

package platform

import (
	"fmt"
	"net"
)

func verifyPrivilegeBrokerPeer(net.Conn) error {
	return fmt.Errorf("setup-managed privilege broker peer verification requires Linux")
}
