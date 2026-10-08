//go:build linux

package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type authorizationPeer struct {
	pid, uid int
	err      error
}
type authorizationPeerKey struct{}

// StartAuthorizationBroker exposes credentials only to kernel-identified
// descendants of an approved process incarnation. It shares this API owner's
// lifecycle and signer; it is not an independent identity daemon.
func (o *Orchestrator) StartAuthorizationBroker() (func() error, error) {
	path := cliutil.AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Getuid() || info.Mode().Perm() != 0700 || !info.IsDir() {
		return nil, errors.New("unsafe agent credential socket directory")
	}
	if _, err := os.Lstat(path); err == nil {
		conn, err := net.DialTimeout("unix", path, time.Second)
		if err == nil {
			conn.Close()
			return nil, errors.New("agent credential broker already running")
		}
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSocket == 0 {
			return nil, errors.New("unsafe agent credential socket path")
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		listener.Close()
		return nil, err
	}
	server := &http.Server{ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 5 * time.Second,
		ConnContext: func(ctx context.Context, conn net.Conn) context.Context {
			peer := authorizationPeer{err: errors.New("peer credentials unavailable")}
			if unix, ok := conn.(*net.UnixConn); ok {
				raw, err := unix.SyscallConn()
				if err == nil {
					err = raw.Control(func(fd uintptr) {
						credentials, e := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
						peer.err = e
						if e == nil {
							peer.pid, peer.uid = int(credentials.Pid), int(credentials.Uid)
						}
					})
					if err != nil {
						peer.err = err
					}
				}
			}
			return context.WithValue(ctx, authorizationPeerKey{}, peer)
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if r.Method != http.MethodGet || r.URL.Path != "/credential" {
				http.NotFound(w, r)
				return
			}
			peer, ok := r.Context().Value(authorizationPeerKey{}).(authorizationPeer)
			if !ok || peer.err != nil || peer.uid != os.Getuid() {
				http.Error(w, "verified local peer required", http.StatusUnauthorized)
				return
			}
			token, found, err := o.ResolveAuthorizationCredential(r.Context(), peer.pid)
			if err != nil {
				http.Error(w, "authorization expired, revoked or unavailable", http.StatusForbidden)
				return
			}
			if !found {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"token": token})
		})}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			o.logAuthorizationBrokerFailure(err)
		}
	}()
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}, nil
}
func (o *Orchestrator) logAuthorizationBrokerFailure(err error) {
	fmt.Fprintln(os.Stderr, "agent authorization broker stopped:", err)
}
