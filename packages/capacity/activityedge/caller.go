package activityedge

import (
	"context"
	"net"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

// A model store's usage file says when a model was last requested; the caller
// identity says who requested it. The edge sees only a loopback TCP peer, so
// it resolves that peer to the process holding the other end of the
// connection and names it by its Vrooli scope (scenario service, resource
// component, agent session) and executable, using the shared callerid lookup.
// The scope is what stays stable across restarts, so "who uses this model"
// survives PID churn.
//
// Resolution runs once per connection, off the request path, the first time
// the connection carries model work; the sender is alive then because it is
// waiting for its response.

type peerKey struct{}

// connPeer resolves one connection's caller at most once and hands the
// result to every request on that connection that asked for it.
type connPeer struct {
	resolve func() string

	mu       sync.Mutex
	started  bool
	resolved bool
	caller   string
	waiters  []func(string)
}

// withPeer attaches a lazily resolved caller to a connection's context.
func withPeer(ctx context.Context, conn net.Conn, identify func(local, remote net.Addr) string) context.Context {
	if conn == nil || identify == nil {
		return ctx
	}
	local, remote := conn.LocalAddr(), conn.RemoteAddr()
	return context.WithValue(ctx, peerKey{}, &connPeer{resolve: func() string { return identify(local, remote) }})
}

func peerFrom(ctx context.Context) *connPeer {
	peer, _ := ctx.Value(peerKey{}).(*connPeer)
	return peer
}

// observe calls fn with the connection's caller, immediately when it is
// known and otherwise once the background resolution finishes.
func (p *connPeer) observe(fn func(string)) {
	if p == nil {
		fn(modelretention.UnknownCaller)
		return
	}
	p.mu.Lock()
	if p.resolved {
		caller := p.caller
		p.mu.Unlock()
		fn(caller)
		return
	}
	p.waiters = append(p.waiters, fn)
	start := !p.started
	p.started = true
	p.mu.Unlock()
	if !start {
		return
	}
	go func() {
		caller := strings.TrimSpace(p.resolve())
		if caller == "" {
			caller = modelretention.UnknownCaller
		}
		p.mu.Lock()
		p.resolved, p.caller = true, caller
		waiters := p.waiters
		p.waiters = nil
		p.mu.Unlock()
		for _, waiter := range waiters {
			waiter(caller)
		}
	}()
}
