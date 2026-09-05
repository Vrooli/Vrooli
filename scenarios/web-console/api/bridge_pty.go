package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/nodereach"
	sharedsession "github.com/vrooli/api-core/operatorsession"
	"web-console/internal/pty"
)

var errBridgePTYClosed = errors.New("bridge PTY is closed")

// bridgePTY is the web-console PTY adapter for the shared node client. The
// client owns URL discovery, authentication, framing, reconnect semantics, and
// typed resize/data operations; this adapter only satisfies the local session
// manager's PTY interface.
type bridgePTY struct {
	session *nodereach.Session
	closed  atomic.Bool
}

func bridgePTYFactory(spec pty.LaunchSpec) (pty.PTY, error) {
	token := spec.RemoteOwnerToken
	var tokenProvider func(context.Context) (string, error)
	if strings.HasPrefix(strings.TrimSpace(token), sharedsession.LocalSessionScheme+" ") {
		token = ""
		tokenProvider = resolveLocalOwnerToken
	}
	client := nodereach.New(nodereach.Config{
		BridgeURL:     baseRemoteURL(spec.RemoteURL),
		Token:         token,
		TokenProvider: tokenProvider,
		ReauthToken:   spec.RemoteReauthToken,
	})
	session, err := client.Open(context.Background(), nodereach.OpenRequest{
		NodeID: spec.RemoteNodeID, SessionID: spec.SessionID, Shell: spec.Shell,
		WorkingDir: spec.WorkingDir, Width: uint32(spec.Cols), Height: uint32(spec.Rows),
	}, 30*time.Second)
	if err != nil {
		return nil, err
	}
	return &bridgePTY{session: session}, nil
}

func baseRemoteURL(raw string) string { return strings.TrimRight(strings.TrimSpace(raw), "/") }

func (p *bridgePTY) Read(dst []byte) (int, error) {
	if p.closed.Load() {
		return 0, io.EOF
	}
	return p.session.Read(dst)
}

func (p *bridgePTY) WriteInput(data []byte, _ pty.InputKind) error {
	if p.closed.Load() {
		return errBridgePTYClosed
	}
	_, err := p.session.Write(data)
	return err
}

func (p *bridgePTY) SetSize(cols, rows uint16) error {
	if p.closed.Load() {
		return errBridgePTYClosed
	}
	return p.session.Resize(uint32(cols), uint32(rows))
}

func (p *bridgePTY) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	return p.session.Close()
}

func (p *bridgePTY) Kill() error { return p.Close() }

func (p *bridgePTY) ExitCode() int { return -1 }

func (p *bridgePTY) ProbeReady(context.Context) error { return nil }

func (p *bridgePTY) CurrentDir(context.Context) (string, error) {
	return "", pty.ErrUnsupported
}
