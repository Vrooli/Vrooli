package callerid

import (
	"net"
	"sync"
)

// maxCachedSockets bounds the socket-to-caller memory. A keep-alive client
// reuses one socket for many requests, so the cache saves the process scan on
// every request after a connection's first; the bound only matters for a
// server with an unusual number of live connections, which then rescans.
const maxCachedSockets = 4096

// Resolver identifies loopback callers and remembers each live socket's
// caller. A socket inode is unique while the socket exists, so a cached answer
// can never name a different process; the inode lookup itself is one read of
// the kernel's TCP table, and only a new socket pays for the process scan.
type Resolver struct {
	procRoot string

	mu      sync.Mutex
	sockets map[string]string
}

// NewResolver returns a Resolver reading the live /proc.
func NewResolver() *Resolver { return &Resolver{procRoot: "/proc", sockets: map[string]string{}} }

// Identify names the process holding the client end of the loopback TCP
// connection whose server end is local and client end is remote. It returns
// Unknown for a non-loopback peer, on platforms without a peer-process
// lookup, and when the sender has already gone.
func (r *Resolver) Identify(local, remote net.Addr) string {
	if r == nil {
		return Unknown
	}
	return r.identify(local, remote)
}

func (r *Resolver) remembered(inode string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	label, ok := r.sockets[inode]
	return label, ok
}

func (r *Resolver) remember(inode, label string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sockets) >= maxCachedSockets {
		r.sockets = map[string]string{}
	}
	r.sockets[inode] = label
}
