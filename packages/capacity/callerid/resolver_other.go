//go:build !linux

package callerid

import "net"

// identify has no portable peer-process lookup outside Linux; callers are
// still served, under the unknown identity.
func (r *Resolver) identify(_, _ net.Addr) string { return Unknown }
