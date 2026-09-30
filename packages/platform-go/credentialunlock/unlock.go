// Package credentialunlock is the local hand-off that lets a Bridge-managed
// node open its encrypted credential store after a reboot with nobody present.
//
// A headless machine often has no unattended key wrap: a Mac's login Keychain
// stays locked until someone logs in, and macOS has no session-scoped cache to
// remember an unlock between commands. The node's Bridge agent receives the
// store passphrase from the control plane sealed to the node's own encryption
// key, keeps it only in memory, and serves it here to local Vrooli processes
// running as the same user. Nothing secret is written to the node's disk.
//
// Both ends are deliberately strict: the client only talks to a socket owned by
// its own user in a directory nobody else can write, and the server only
// answers a peer whose kernel-reported UID is its own.
package credentialunlock

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

// SocketName is the socket's file name inside the runtime state directory.
const SocketName = "credential-unlock.sock"

// NodeStoreLogicalIDPrefix is the credential-authority identity prefix under
// which the control plane escrows each node's store passphrase
// ("vrooli-bridge/node-credential-store/<machineID>", field "passphrase"). The
// node agent recognizes a pushed grant for this address as its own store's
// unlock material rather than an ordinary job credential.
const NodeStoreLogicalIDPrefix = "vrooli-bridge/node-credential-store/"

// protocolVersion guards the one-line JSON exchange.
const protocolVersion = 1

// DefaultTimeout bounds one request. A missing or silent agent must cost a
// store open milliseconds, never a hang.
const DefaultTimeout = 750 * time.Millisecond

// ErrUnavailable means no trustworthy unlock agent answered. Callers treat it
// exactly like "no passphrase was supplied".
var ErrUnavailable = errors.New("credential unlock agent is unavailable")

// SocketPath returns the unlock socket for the user whose home is home.
func SocketPath(home string) (string, error) {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	return filepath.Join(state, SocketName), nil
}

type request struct {
	Version int    `json:"version"`
	Op      string `json:"op"`
}

type response struct {
	Version    int    `json:"version"`
	Passphrase string `json:"passphrase,omitempty"`
	Error      string `json:"error,omitempty"`
}

const opStorePassphrase = "store-passphrase"

// RequestPassphrase asks the local unlock agent for the store passphrase. It
// refuses a socket it cannot prove belongs to this user.
func RequestPassphrase(ctx context.Context, socket string) (string, error) {
	if err := trustedSocket(socket); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := json.NewEncoder(conn).Encode(request{Version: protocolVersion, Op: opStorePassphrase}); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	var reply response
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&reply); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if reply.Version != protocolVersion {
		return "", fmt.Errorf("%w: unexpected protocol version %d", ErrUnavailable, reply.Version)
	}
	if reply.Error != "" {
		return "", fmt.Errorf("%w: %s", ErrUnavailable, reply.Error)
	}
	if strings.TrimSpace(reply.Passphrase) == "" {
		return "", fmt.Errorf("%w: the agent holds no passphrase yet", ErrUnavailable)
	}
	return reply.Passphrase, nil
}

// Listen prepares an owner-only socket at socket, replacing a stale socket but
// never another kind of file.
func Listen(socket string) (net.Listener, error) {
	dir := filepath.Dir(socket)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create credential unlock directory: %w", err)
	}
	if info, err := os.Lstat(socket); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("refusing to replace non-socket path %q", socket)
		}
		if err := os.Remove(socket); err != nil {
			return nil, fmt.Errorf("remove stale credential unlock socket: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect credential unlock socket: %w", err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(socket, 0o600); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("restrict credential unlock socket: %w", err)
	}
	return listener, nil
}

// Serve answers requests until ctx ends or the listener closes. passphrase
// returns the held value and whether one is held; the returned slice is
// copied into the reply and never retained.
func Serve(ctx context.Context, listener net.Listener, passphrase func() ([]byte, bool)) error {
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	self := os.Getuid()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go serveConn(conn, self, passphrase)
	}
}

func serveConn(conn net.Conn, self int, passphrase func() ([]byte, bool)) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	uid, err := PeerUID(conn)
	if err != nil || uid != self {
		_ = json.NewEncoder(conn).Encode(response{Version: protocolVersion, Error: "caller is not this node's user"})
		return
	}
	var req request
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&req); err != nil || req.Version != protocolVersion || req.Op != opStorePassphrase {
		_ = json.NewEncoder(conn).Encode(response{Version: protocolVersion, Error: "invalid credential unlock request"})
		return
	}
	value, ok := passphrase()
	if !ok || len(value) == 0 {
		_ = json.NewEncoder(conn).Encode(response{Version: protocolVersion, Error: "the control plane has not delivered this node's store passphrase yet"})
		return
	}
	_ = json.NewEncoder(conn).Encode(response{Version: protocolVersion, Passphrase: string(value)})
}
