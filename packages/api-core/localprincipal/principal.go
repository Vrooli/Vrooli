// Package localprincipal abstracts operating-system peer credentials for
// Unix-domain identity exchange. The principal is opaque so Windows can later
// use a SID without changing the authenticator contract.
package localprincipal

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var ErrUnsupported = errors.New("local peer credentials unsupported; use the token-file fallback")

// Principal is an opaque, platform-qualified local identity.
type Principal string

func (p Principal) String() string { return string(p) }

// UnixUID is the canonical opaque representation used by Linux and macOS.
func UnixUID(uid uint32) Principal { return Principal("unix:" + strconv.FormatUint(uint64(uid), 10)) }

// ParseUnixUID is used only by the privilege broker's legacy numeric gate.
func ParseUnixUID(p Principal) (uint32, error) {
	value := strings.TrimPrefix(string(p), "unix:")
	if value == string(p) || value == "" {
		return 0, fmt.Errorf("invalid unix principal %q", p)
	}
	uid, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid unix principal %q: %w", p, err)
	}
	return uint32(uid), nil
}

// Peer returns the authenticated OS principal for conn. It never trusts a
// principal supplied in the request body.
func Peer(conn *net.UnixConn) (Principal, error) { return peer(conn) }

// Current returns the caller's own local principal for explicit machine
// linking. It is separate from Peer so linking never substitutes caller input
// for a socket's kernel-authenticated peer.
func Current() (Principal, error) { return current() }
