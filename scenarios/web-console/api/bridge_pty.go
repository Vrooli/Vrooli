package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	"web-console/internal/pty"
)

var errBridgePTYClosed = errors.New("bridge PTY is closed")

type bridgePTY struct {
	conn *websocket.Conn
	mu   sync.Mutex

	readCh  chan []byte
	doneCh  chan struct{}
	readyCh chan error
	pending []byte
	closed  atomic.Bool
	ready   atomic.Bool
	exit    atomic.Int32
	seq     atomic.Uint64
}

func bridgePTYFactory(spec pty.LaunchSpec) (pty.PTY, error) {
	base := strings.TrimRight(strings.TrimSpace(spec.RemoteURL), "/")
	if base == "" || strings.TrimSpace(spec.RemoteNodeID) == "" {
		return nil, errors.New("remote backend requires a Bridge URL and node id")
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("remote backend received an invalid Bridge URL")
	}
	u.Scheme = websocketScheme(u.Scheme)
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/channel/session"
	query := u.Query()
	query.Set("node", spec.RemoteNodeID)
	query.Set("session_id", spec.SessionID)
	query.Set("scopes", "vrooli-bridge:session")
	if spec.Shell != "" {
		query.Set("shell", spec.Shell)
	}
	if spec.WorkingDir != "" {
		query.Set("working_dir", spec.WorkingDir)
	}
	u.RawQuery = query.Encode()
	header := make(map[string][]string)
	header["Authorization"] = []string{currentOwnerAuthorization(spec.RemoteOwnerToken)}
	if strings.TrimSpace(spec.RemoteReauthToken) != "" {
		header["X-Bridge-Owner-Reauth"] = []string{spec.RemoteReauthToken}
	}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return nil, fmt.Errorf("dial Bridge session: %w", err)
	}
	p := &bridgePTY{conn: conn, readCh: make(chan []byte, 64), doneCh: make(chan struct{}), readyCh: make(chan error, 1)}
	p.exit.Store(-1)
	go p.readLoop()
	return p, nil
}

func (p *bridgePTY) readLoop() {
	defer close(p.doneCh)
	defer close(p.readCh)
	for {
		kind, payload, err := p.conn.ReadMessage()
		if err != nil {
			p.signalReady(err)
			return
		}
		if kind != websocket.BinaryMessage {
			continue
		}
		var frame sessionv1.Frame
		if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &frame); err != nil {
			p.signalReady(fmt.Errorf("decode Bridge frame: %w", err))
			return
		}
		switch payload := frame.Payload.(type) {
		case *sessionv1.Frame_Open:
			p.ready.Store(true)
			p.signalReady(nil)
		case *sessionv1.Frame_Data:
			if data := append([]byte(nil), payload.Data.GetData()...); len(data) > 0 {
				select {
				case p.readCh <- data:
				case <-p.doneCh:
					return
				}
			}
		case *sessionv1.Frame_Ack:
			if !payload.Ack.GetAccepted() {
				reason := payload.Ack.GetCode()
				if reason == "" {
					reason = payload.Ack.GetReason()
				}
				if reason == "" {
					reason = "Bridge rejected session data"
				}
				p.signalReady(errors.New(reason))
			}
		case *sessionv1.Frame_Close:
			p.signalReady(errors.New(payload.Close.GetReason()))
			return
		}
	}
}

func (p *bridgePTY) signalReady(err error) {
	if p.ready.Load() || err != nil {
		select {
		case p.readyCh <- err:
		default:
		}
	}
}

func (p *bridgePTY) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	if len(p.pending) > 0 {
		n := copy(dst, p.pending)
		p.pending = p.pending[n:]
		return n, nil
	}
	select {
	case data, ok := <-p.readCh:
		if !ok {
			return 0, io.EOF
		}
		n := copy(dst, data)
		p.pending = data[n:]
		return n, nil
	case <-p.doneCh:
		return 0, io.EOF
	}
}

func (p *bridgePTY) WriteInput(data []byte, _ pty.InputKind) error {
	if p.closed.Load() {
		return errBridgePTYClosed
	}
	return p.write(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: p.seq.Add(1) - 1, Data: append([]byte(nil), data...)}}})
}

func (p *bridgePTY) SetSize(cols, rows uint16) error {
	if p.closed.Load() {
		return errBridgePTYClosed
	}
	return p.write(&sessionv1.Frame{Payload: &sessionv1.Frame_Resize{Resize: &sessionv1.Resize{Columns: uint32(cols), Rows: uint32(rows)}}})
}

func (p *bridgePTY) write(frame *sessionv1.Frame) error {
	payload, err := proto.Marshal(frame)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return p.conn.WriteMessage(websocket.BinaryMessage, payload)
}

func (p *bridgePTY) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	_ = p.write(&sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Code: "close", Reason: "web-console session closed"}}})
	return p.conn.Close()
}

func (p *bridgePTY) Kill() error { return p.Close() }

func (p *bridgePTY) ExitCode() int { return int(p.exit.Load()) }

func (p *bridgePTY) ProbeReady(ctx context.Context) error {
	if p.ready.Load() {
		return nil
	}
	select {
	case err := <-p.readyCh:
		if err != nil {
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *bridgePTY) CurrentDir(context.Context) (string, error) {
	return "", pty.ErrUnsupported
}
