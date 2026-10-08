package codex

// This file contains the deliberately small transport seam used by the
// managed Codex owner. Codex app-server speaks newline-delimited JSON-RPC over
// an owned stdio stream. The owner above this package is responsible for
// policy, provenance, receipts, and projection reconciliation.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

var ErrClosed = errors.New("codex app-server client is closed")

type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type Client struct {
	stdin         io.WriteCloser
	stdout        io.Reader
	cmd           *exec.Cmd
	writeMu       sync.Mutex
	pendingMu     sync.Mutex
	pending       map[string]chan rpcResponse
	orphan        map[string]Message
	notifications chan Message
	requests      chan Message
	readDone      chan struct{}
	readOnce      sync.Once
	closeOnce     sync.Once
	nextID        atomic.Uint64
	closed        bool
	closeErr      error
	serverVersion string
}

type rpcResponse struct {
	message Message
	err     error
}

// NewClient creates a client over an already-owned stream. Tests and Unix
// socket adapters can use this without spawning a provider process.
func NewClient(stdin io.WriteCloser, stdout io.Reader) *Client {
	c := &Client{stdin: stdin, stdout: stdout, pending: make(map[string]chan rpcResponse), orphan: make(map[string]Message), notifications: make(chan Message, 128), requests: make(chan Message, 128), readDone: make(chan struct{})}
	return c
}

func (c *Client) startReader() { c.readOnce.Do(func() { go c.readLoop() }) }

// Start launches exactly one app-server owner over stdio. The caller owns the
// returned client and must Close it before allowing the session to recover.
func Start(ctx context.Context, executable string, args ...string) (*Client, error) {
	return StartInDir(ctx, executable, "", args...)
}

func StartInDir(ctx context.Context, executable, dir string, args ...string) (*Client, error) {
	return StartInDirWithEnv(ctx, executable, dir, nil, args...)
}

func StartInDirWithEnv(ctx context.Context, executable, dir string, env []string, args ...string) (*Client, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("app-server stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("app-server stdout: %w", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start app-server: %w", err)
	}
	c := NewClient(stdin, stdout)
	c.cmd = cmd
	return c, nil
}

func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
	if c == nil || c.stdin == nil || c.stdout == nil {
		return ErrClosed
	}
	id := c.nextID.Add(1)
	payload, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal %s params: %w", method, err)
	}
	req := Message{JSONRPC: "2.0", ID: json.RawMessage(fmt.Sprintf("%d", id)), Method: method, Params: payload}
	line, err := json.Marshal(req)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	key := fmt.Sprintf("%d", id)
	responseCh := make(chan rpcResponse, 1)
	c.pendingMu.Lock()
	if c.closed {
		c.pendingMu.Unlock()
		return ErrClosed
	}
	if message, ok := c.orphan[key]; ok {
		delete(c.orphan, key)
		responseCh <- rpcResponse{message: message}
	} else {
		c.pending[key] = responseCh
	}
	c.pendingMu.Unlock()
	c.startReader()
	c.writeMu.Lock()
	if _, err := c.stdin.Write(line); err != nil {
		c.writeMu.Unlock()
		c.pendingMu.Lock()
		delete(c.pending, key)
		c.pendingMu.Unlock()
		return fmt.Errorf("write %s: %w", method, err)
	}
	c.writeMu.Unlock()
	select {
	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, key)
		c.pendingMu.Unlock()
		return ctx.Err()
	case response := <-responseCh:
		if response.err != nil {
			return response.err
		}
		if response.message.Error != nil {
			return fmt.Errorf("%s (%d): %s", method, response.message.Error.Code, response.message.Error.Message)
		}
		if result != nil && len(response.message.Result) > 0 {
			if err := json.Unmarshal(response.message.Result, result); err != nil {
				return fmt.Errorf("decode %s result: %w", method, err)
			}
		}
		return nil
	}
}

func (c *Client) Notify(method string, params any) error {
	if c == nil || c.stdin == nil {
		return ErrClosed
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.startReader()
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	line, err := json.Marshal(Message{JSONRPC: "2.0", Method: method, Params: payload})
	if err != nil {
		return err
	}
	line = append(line, '\n')
	_, err = c.stdin.Write(line)
	return err
}

func (c *Client) readLoop() {
	defer func() {
		close(c.notifications)
		close(c.requests)
		close(c.readDone)
	}()
	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Method != "" && len(msg.ID) > 0 {
			select {
			case c.requests <- msg:
			default:
			}
			continue
		}
		if len(msg.ID) == 0 {
			select {
			case c.notifications <- msg:
			default:
			}
			continue
		}
		key := string(msg.ID)
		c.pendingMu.Lock()
		ch := c.pending[key]
		delete(c.pending, key)
		if ch == nil {
			if len(c.orphan) >= 128 {
				for orphanID := range c.orphan {
					delete(c.orphan, orphanID)
					break
				}
			}
			c.orphan[key] = msg
		}
		c.pendingMu.Unlock()
		if ch != nil {
			ch <- rpcResponse{message: msg}
		}
	}
	err := scanner.Err()
	if err == nil {
		err = io.EOF
	}
	c.pendingMu.Lock()
	c.closed = true
	for key, ch := range c.pending {
		delete(c.pending, key)
		ch <- rpcResponse{err: func() error {
			if errors.Is(err, io.EOF) {
				return ErrClosed
			}
			return err
		}()}
	}
	c.pendingMu.Unlock()
}

func (c *Client) Notifications() <-chan Message {
	if c == nil {
		return nil
	}
	return c.notifications
}

// Done closes when the owned app-server stream reaches EOF or fails. Owners
// use it to withdraw native capability without confusing a dead provider with
// a live, controllable session.
func (c *Client) Done() <-chan struct{} {
	if c == nil {
		return nil
	}
	return c.readDone
}

// Requests exposes provider-initiated JSON-RPC requests such as approval
// prompts. They require a response and must not be mistaken for events.
func (c *Client) Requests() <-chan Message {
	if c == nil {
		return nil
	}
	return c.requests
}

func (c *Client) Respond(id json.RawMessage, result any, rpcErr *RPCError) error {
	if c == nil || c.stdin == nil {
		return ErrClosed
	}
	message := Message{JSONRPC: "2.0", ID: id, Error: rpcErr}
	if rpcErr == nil {
		payload, err := json.Marshal(result)
		if err != nil {
			return err
		}
		message.Result = payload
	}
	line, err := json.Marshal(message)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err = c.stdin.Write(line)
	return err
}

func (c *Client) Initialize(ctx context.Context, clientName, version string) error {
	_, err := c.InitializeWithInfo(ctx, clientName, version)
	return err
}

type InitializeResult struct {
	UserAgent    string         `json:"userAgent"`
	Capabilities map[string]any `json:"capabilities,omitempty"`
}

func (c *Client) InitializeWithInfo(ctx context.Context, clientName, version string) (InitializeResult, error) {
	var info InitializeResult
	if err := c.Call(ctx, "initialize", map[string]any{"clientInfo": map[string]string{"name": clientName, "version": version}}, &info); err != nil {
		return info, err
	}
	c.serverVersion = info.UserAgent
	if err := c.Notify("initialized", map[string]any{}); err != nil {
		return info, err
	}
	return info, nil
}

func (c *Client) ServerVersion() string {
	if c == nil {
		return ""
	}
	return c.serverVersion
}

type ThreadRef struct {
	ID    string    `json:"id"`
	Turns []TurnRef `json:"turns,omitempty"`
}

type ThreadReadResult struct {
	Thread ThreadRef `json:"thread"`
}

type ForkResult struct {
	ThreadID string    `json:"threadId"`
	Thread   ThreadRef `json:"thread"`
}

func (r ForkResult) ID() string {
	if r.ThreadID != "" {
		return r.ThreadID
	}
	return r.Thread.ID
}

type ThreadStartResult struct {
	ThreadID string    `json:"threadId"`
	Thread   ThreadRef `json:"thread"`
}

type TurnRef struct {
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
}

type TurnStartResult struct {
	TurnID string  `json:"turnId"`
	Turn   TurnRef `json:"turn"`
}

func (r TurnStartResult) ID() string {
	if r.TurnID != "" {
		return r.TurnID
	}
	return r.Turn.ID
}

func (r ThreadStartResult) ID() string {
	if r.ThreadID != "" {
		return r.ThreadID
	}
	return r.Thread.ID
}

func (c *Client) StartThread(ctx context.Context, cwd string) (ThreadStartResult, error) {
	var out ThreadStartResult
	err := c.Call(ctx, "thread/start", map[string]string{"cwd": cwd}, &out)
	return out, err
}

func (c *Client) ResumeThread(ctx context.Context, threadID string) (ThreadStartResult, error) {
	var out ThreadStartResult
	err := c.Call(ctx, "thread/resume", map[string]string{"threadId": threadID}, &out)
	return out, err
}

func (c *Client) ReadThread(ctx context.Context, threadID string) (ThreadReadResult, error) {
	var out ThreadReadResult
	err := c.Call(ctx, "thread/read", map[string]any{"threadId": threadID, "includeTurns": true}, &out)
	return out, err
}

func (c *Client) ForkThread(ctx context.Context, threadID, lastTurnID string) (ForkResult, error) {
	var out ForkResult
	err := c.Call(ctx, "thread/fork", map[string]any{"threadId": threadID, "lastTurnId": lastTurnID}, &out)
	return out, err
}

func (c *Client) InterruptTurn(ctx context.Context, threadID, turnID string) error {
	return c.Call(ctx, "turn/interrupt", map[string]string{"threadId": threadID, "turnId": turnID}, nil)
}

func (c *Client) StartTurn(ctx context.Context, threadID, text string) (TurnStartResult, error) {
	var out TurnStartResult
	err := c.Call(ctx, "turn/start", map[string]any{
		"threadId": threadID,
		"input":    []map[string]string{{"type": "text", "text": text}},
	}, &out)
	return out, err
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		c.startReader()
		c.writeMu.Lock()
		c.closeErr = c.stdin.Close()
		c.writeMu.Unlock()
		<-c.readDone
		if c.cmd != nil {
			_ = c.cmd.Wait()
		}
	})
	return c.closeErr
}
