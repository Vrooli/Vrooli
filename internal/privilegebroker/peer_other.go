//go:build !linux

package privilegebroker

import (
	"net"

	"github.com/vrooli/vrooli/internal/localprincipal"
)

func peerUID(*net.UnixConn) (uint32, error) {
	return 0, localprincipal.ErrUnsupported
}
