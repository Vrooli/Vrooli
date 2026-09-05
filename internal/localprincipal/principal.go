// Package localprincipal is the control-plane compatibility seam over the
// shared api-core peer-credential implementation.
package localprincipal

import (
	"net"

	shared "github.com/vrooli/api-core/localprincipal"
)

type Principal = shared.Principal

var ErrUnsupported = shared.ErrUnsupported

func Peer(conn *net.UnixConn) (Principal, error) { return shared.Peer(conn) }

func Current() (Principal, error) { return shared.Current() }
