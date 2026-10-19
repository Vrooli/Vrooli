//go:build linux

package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"agent-manager/internal/domain"
	"github.com/vrooli/cli-core/cliutil"
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
	// Only the canonical authority owns the host credential socket. Isolated
	// scenario variants must not intercept live sessions or stop the owner.
	if variant := os.Getenv("VROOLI_VARIANT"); variant != "" && variant != "live" && variant != "default" {
		return func() error { return nil }, nil
	}
	return o.startAuthorizationBrokerAt(cliutil.AgentCredentialSocket())
}

func (o *Orchestrator) startAuthorizationBrokerAt(path string) (func() error, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Getuid() || info.Mode().Perm() != 0o700 || !info.IsDir() {
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
	if err := os.Chmod(path, 0o600); err != nil {
		listener.Close()
		return nil, err
	}
	server := &http.Server{
		ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 5 * time.Second,
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
			baselineToken := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
			purpose := ""
			if values, present := r.URL.Query()["purpose"]; present {
				if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
					http.Error(w, "invalid credential purpose", http.StatusBadRequest)
					return
				}
				purpose = values[0]
			}
			var credentials AuthorizationCredentials
			var err error
			switch purpose {
			case "":
				credentials, err = o.ResolveAuthorizationCredentials(r.Context(), peer.pid, baselineToken)
			case "ordinary_identity":
				credentials, err = o.resolveOrdinaryRunIdentityCredential(r.Context(), peer.pid, baselineToken)
			default:
				http.Error(w, "unknown credential purpose", http.StatusBadRequest)
				return
			}
			if err != nil {
				code := "unavailable"
				var refused *domain.AuthorizationError
				if errors.As(err, &refused) {
					code = refused.Code
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"code": code, "recovery": "vrooli agent authorization <session-or-run>"})
				return
			}
			if credentials.BaselineToken == "" && credentials.LegacyToken == "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(credentials)
		}),
	}
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
