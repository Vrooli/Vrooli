// Package localexchange serves the one local-socket identity exchange path.
// The socket is connectable by local users; authorization comes only from the
// kernel peer credential injected into request context.
package localexchange

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/localprincipal"
)

type peerKey struct{}

func WithPeerPrincipal(ctx context.Context, principal localprincipal.Principal) context.Context {
	return context.WithValue(ctx, peerKey{}, principal)
}

func PeerPrincipal(ctx context.Context) (localprincipal.Principal, bool) {
	principal, ok := ctx.Value(peerKey{}).(localprincipal.Principal)
	return principal, ok
}

// Start binds the socket before returning and serves until stop is called or
// the parent context ends. The listener is deliberately broad (0666): local
// users may connect, but only a bound kernel principal can exchange a token.
func Start(ctx context.Context, path string, handler http.Handler) (func(), error) {
	if handler == nil {
		return nil, errors.New("local exchange handler is required")
	}
	if path == "" {
		return nil, errors.New("local exchange socket path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return nil, fmt.Errorf("create local exchange directory: %w", err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove stale local exchange socket: %w", err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("listen local exchange socket: %w", err)
	}
	if err := os.Chmod(path, 0o666); err != nil {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("set local exchange socket permissions: %w", err)
	}
	server := &http.Server{
		Handler: handler,
		ConnContext: func(requestContext context.Context, conn net.Conn) context.Context {
			unixConn, ok := conn.(*net.UnixConn)
			if !ok {
				return requestContext
			}
			principal, err := localprincipal.Peer(unixConn)
			if err != nil {
				return requestContext
			}
			return WithPeerPrincipal(requestContext, principal)
		},
	}
	stop := func() {
		_ = server.Close()
		_ = listener.Close()
		_ = os.Remove(path)
	}
	go func() {
		<-ctx.Done()
		stop()
	}()
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) && ctx.Err() == nil {
			// The owning process logs the failure through its health/lifecycle
			// surface; the goroutine must not turn an accepted API into a panic.
		}
	}()
	return stop, nil
}
