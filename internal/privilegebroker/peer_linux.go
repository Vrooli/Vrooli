//go:build linux

package privilegebroker

import (
	"fmt"
	shared "github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/vrooli/internal/localprincipal"
	"net"
)

func peerUID(conn *net.UnixConn) (uint32, error) {
	principal, err := localprincipal.Peer(conn)
	if err != nil {
		return 0, err
	}
	uid, err := shared.ParseUnixUID(principal)
	if err != nil {
		return 0, fmt.Errorf("parse local principal %q: %w", principal, err)
	}
	return uint32(uid), nil
}
