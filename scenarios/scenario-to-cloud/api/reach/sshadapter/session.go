package sshadapter

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// OpenSession opens an interactive PTY over the resolved SSH connection.
// Authentication uses the key the connection config names (resolved through
// the credential binding) and, when present, the operator's local agent;
// host keys follow the same trust-on-first-use store as every other SSH
// command.
func (a *Adapter) OpenSession(ctx context.Context, target identity.TargetRef, spec reach.SessionSpec) (reach.Session, error) {
	cfg, err := a.connect(ctx, target)
	if err != nil {
		return nil, err
	}
	dial := a.Dial
	if dial == nil {
		dial = dialSSH
	}
	client, err := dial(cfg)
	if err != nil {
		return nil, &reach.Error{Kind: reach.KindTargetOffline, Transport: identity.TransportSSH, Target: target.Key(), Detail: "ssh session dial failed", Err: err}
	}
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportSSH, Target: target.Key(), Detail: "ssh session open failed", Err: err}
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportSSH, Target: target.Key(), Err: err}
	}
	pr, pw := io.Pipe()
	session.Stdout = pw
	session.Stderr = pw
	term := spec.Term
	if term == "" {
		term = "xterm-256color"
	}
	cols, rows := spec.Cols, spec.Rows
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	modes := gossh.TerminalModes{gossh.ECHO: 1, gossh.TTY_OP_ISPEED: 14400, gossh.TTY_OP_OSPEED: 14400}
	if err := session.RequestPty(term, rows, cols, modes); err != nil {
		session.Close()
		client.Close()
		return nil, &reach.Error{Kind: reach.KindProtocolUnsupported, Transport: identity.TransportSSH, Target: target.Key(), Detail: "pty request refused", Err: err}
	}
	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return nil, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportSSH, Target: target.Key(), Detail: "shell start failed", Err: err}
	}
	s := &ptySession{client: client, session: session, stdin: stdin, out: pr, done: make(chan struct{})}
	go func() {
		s.waitErr = session.Wait()
		_ = pw.Close()
		close(s.done)
	}()
	return s, nil
}

// Dialer opens the SSH client a session rides on. Tests substitute an
// in-memory server; production uses dialSSH.
type Dialer func(cfg ConnectionConfig) (*gossh.Client, error)

type ptySession struct {
	client  *gossh.Client
	session *gossh.Session
	stdin   io.WriteCloser
	out     *io.PipeReader
	done    chan struct{}
	waitErr error
	once    sync.Once
}

func (s *ptySession) Read(p []byte) (int, error)  { return s.out.Read(p) }
func (s *ptySession) Write(p []byte) (int, error) { return s.stdin.Write(p) }
func (s *ptySession) Resize(rows, cols int) error { return s.session.WindowChange(rows, cols) }

func (s *ptySession) Wait() error {
	<-s.done
	return s.waitErr
}

func (s *ptySession) Close() error {
	s.once.Do(func() {
		_ = s.session.Close()
		_ = s.client.Close()
		_ = s.out.Close()
	})
	return nil
}

// dialSSH builds the client from the connection config: the bound key (when
// readable) and the local agent (when present); TOFU host key trust.
func dialSSH(cfg ConnectionConfig) (*gossh.Client, error) {
	var auth []gossh.AuthMethod
	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		if conn, err := net.Dial("unix", socket); err == nil { // #nosec G704 -- local operator agent socket, never a network target.
			auth = append(auth, gossh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}
	var keyErr error
	if cfg.KeyPath != "" {
		raw, err := os.ReadFile(ExpandPath(cfg.KeyPath)) //nolint:gosec // key path resolved by the credential binding
		if err == nil {
			if signer, perr := gossh.ParsePrivateKey(raw); perr == nil {
				auth = append(auth, gossh.PublicKeys(signer))
			} else {
				keyErr = perr
			}
		} else {
			keyErr = err
		}
	}
	if len(auth) == 0 {
		if keyErr != nil {
			return nil, fmt.Errorf("no usable ssh authentication: %w", keyErr)
		}
		return nil, fmt.Errorf("no usable ssh authentication: no key bound and no agent")
	}
	port := cfg.Port
	if port == 0 {
		port = DefaultPort
	}
	hostKey, err := NewTOFUHostKeyCallback(cfg.Host, port)
	if err != nil {
		return nil, err
	}
	user := cfg.User
	if user == "" {
		user = DefaultUser
	}
	return gossh.Dial("tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(port)), &gossh.ClientConfig{User: user, Auth: auth, Timeout: 30 * time.Second, HostKeyCallback: hostKey})
}
