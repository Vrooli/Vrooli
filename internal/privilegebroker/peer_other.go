//go:build !linux

package privilegebroker

import (
	"net"

	shared "github.com/vrooli/api-core/localprincipal"
)

func peerUID(*net.UnixConn) (uint32, error) {
	return 0, shared.ErrUnsupported
}
