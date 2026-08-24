// Package session-core contains the transport-neutral contract for interactive
// sessions.  It is deliberately independent of WebSockets, protobufs and SSH
// implementations so desktop, bridge and cloud surfaces can share the same
// lifecycle and byte-stream semantics.
package sessioncore

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
)

// InputKind distinguishes ordinary terminal input, clipboard payloads, and
// synthetic terminal control bytes. Control bytes are best-effort and must
// never inherit a reliable delivery queue from the caller.
type InputKind uint8

const (
	KindKeystroke InputKind = iota
	KindPaste
	KindControl
)

// LaunchSpec is the backend-neutral process/session launch request.
type LaunchSpec struct {
	SessionID  string
	Shell      string
	Cols       uint16
	Rows       uint16
	WorkingDir string
	Env        map[string]string
}

// PTY is the common interactive stream contract. Implementations must not
// expose a raw shell command or a second untyped write path.
type PTY interface {
	Read([]byte) (int, error)
	WriteInput([]byte, InputKind) error
	SetSize(cols, rows uint16) error
	Close() error
	Kill() error
	ExitCode() int
	HasChildProcess() bool
	ProbeReady(context.Context) error
	CurrentDir(context.Context) (string, error)
}

// Factory opens a PTY for a typed launch request.
type Factory func(LaunchSpec) (PTY, error)

// Backend identifies the origin of a session stream.
type BackendKind string

const (
	BackendLocal BackendKind = "local"
	BackendAgent BackendKind = "node-agent"
	BackendSSH   BackendKind = "ssh"
)

// Backend is the interchangeable backend seam used by higher-level session
// managers. The returned PTY has identical read, write, resize and close
// semantics regardless of the selected transport.
type Backend interface {
	Kind() BackendKind
	Open(context.Context, LaunchSpec) (PTY, error)
}

// FactoryBackend adapts an existing PTY factory to the backend seam. It is
// useful for local PTYs and for deterministic tests.
type FactoryBackend struct {
	BackendKind BackendKind
	Factory     Factory
}

func (b FactoryBackend) Kind() BackendKind { return b.BackendKind }

func (b FactoryBackend) Open(_ context.Context, spec LaunchSpec) (PTY, error) {
	if b.Factory == nil {
		return nil, errors.New("session backend factory is not configured")
	}
	return b.Factory(spec)
}

// NewLocalBackend wraps the process/PTY factory used by the local desktop
// runtime. Keeping the factory injection here makes the exact same backend
// contract usable in tests without spawning a shell.
func NewLocalBackend(factory Factory) Backend {
	return FactoryBackend{BackendKind: BackendLocal, Factory: factory}
}

// AgentFrame is the typed node-agent stream envelope. Data is byte-transparent
// and resize/close are control messages; no field is interpreted as a shell
// command.
type AgentFrame struct {
	Data       []byte
	Columns    uint16
	Rows       uint16
	Resize     bool
	Close      bool
	CloseCause string
}

// AgentStream is implemented by the authenticated bridge channel client.
type AgentStream interface {
	Receive(context.Context) (AgentFrame, error)
	Send(context.Context, AgentFrame) error
	Close() error
}

// AgentOpener creates an authenticated, already-authorized agent stream.
type AgentOpener func(context.Context, LaunchSpec) (AgentStream, error)

type agentBackend struct{ open AgentOpener }

func NewAgentBackend(open AgentOpener) Backend { return agentBackend{open: open} }
func (agentBackend) Kind() BackendKind         { return BackendAgent }

func (b agentBackend) Open(ctx context.Context, spec LaunchSpec) (PTY, error) {
	if b.open == nil {
		return nil, errors.New("node-agent opener is not configured")
	}
	stream, err := b.open(ctx, spec)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, errors.New("node-agent opener returned a nil stream")
	}
	return &agentPTY{ctx: ctx, stream: stream}, nil
}

type agentPTY struct {
	ctx     context.Context
	stream  AgentStream
	pending []byte
	closed  bool
}

func (p *agentPTY) Read(dst []byte) (int, error) {
	if len(p.pending) == 0 {
		if p.closed {
			return 0, io.EOF
		}
		frame, err := p.stream.Receive(p.ctx)
		if err != nil {
			return 0, err
		}
		if frame.Close {
			p.closed = true
			return 0, io.EOF
		}
		p.pending = append(p.pending[:0], frame.Data...)
	}
	n := copy(dst, p.pending)
	p.pending = p.pending[n:]
	return n, nil
}

func (p *agentPTY) WriteInput(data []byte, kind InputKind) error {
	if p.closed {
		return io.ErrClosedPipe
	}
	return p.stream.Send(p.ctx, AgentFrame{Data: append([]byte(nil), data...)})
}

func (p *agentPTY) SetSize(cols, rows uint16) error {
	if p.closed {
		return io.ErrClosedPipe
	}
	return p.stream.Send(p.ctx, AgentFrame{Resize: true, Columns: cols, Rows: rows})
}

func (p *agentPTY) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	return p.stream.Close()
}
func (p *agentPTY) Kill() error                                { return p.Close() }
func (p *agentPTY) ExitCode() int                              { return -1 }
func (p *agentPTY) HasChildProcess() bool                      { return true }
func (p *agentPTY) ProbeReady(context.Context) error           { return nil }
func (p *agentPTY) CurrentDir(context.Context) (string, error) { return "", nil }

// SSHSession is the small capability surface needed by the SSH backend. The
// scenario-to-cloud adapter can implement it with x/crypto/ssh without making
// this shared package depend on a particular SSH library.
type SSHSession interface {
	io.Reader
	io.Writer
	RequestPTY(term string, rows, cols uint16) error
	Resize(rows, cols uint16) error
	Start() error
	Wait() error
	Close() error
}

type (
	SSHOpener  func(context.Context, LaunchSpec) (SSHSession, error)
	sshBackend struct{ open SSHOpener }
)

func NewSSHBackend(open SSHOpener) Backend { return sshBackend{open: open} }
func (sshBackend) Kind() BackendKind       { return BackendSSH }

func (b sshBackend) Open(ctx context.Context, spec LaunchSpec) (PTY, error) {
	if b.open == nil {
		return nil, errors.New("SSH opener is not configured")
	}
	s, err := b.open(ctx, spec)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errors.New("SSH opener returned a nil session")
	}
	term := spec.Shell
	if term == "" {
		term = "xterm-256color"
	}
	if err := s.RequestPTY(term, spec.Rows, spec.Cols); err != nil {
		_ = s.Close()
		return nil, err
	}
	if err := s.Start(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return &sshPTY{session: s}, nil
}

type sshPTY struct{ session SSHSession }

func (p *sshPTY) Read(dst []byte) (int, error) { return p.session.Read(dst) }
func (p *sshPTY) WriteInput(data []byte, _ InputKind) error {
	_, err := p.session.Write(data)
	return err
}
func (p *sshPTY) SetSize(cols, rows uint16) error            { return p.session.Resize(rows, cols) }
func (p *sshPTY) Close() error                               { return p.session.Close() }
func (p *sshPTY) Kill() error                                { return p.session.Close() }
func (p *sshPTY) ExitCode() int                              { return -1 }
func (p *sshPTY) HasChildProcess() bool                      { return true }
func (p *sshPTY) ProbeReady(context.Context) error           { return nil }
func (p *sshPTY) CurrentDir(context.Context) (string, error) { return "", nil }

// SameOrigin validates a browser Origin against the request host. It accepts
// only an explicit HTTP(S) origin with the same authority; callers may handle
// an absent Origin separately for non-browser clients.
func SameOrigin(origin, host string) bool {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, strings.TrimSpace(host))
}
