package privilegebroker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	clientParameterA = 10
)

const (
	clientParameterB = 64
)

// Client sends one validated request per connection to the setup-installed
// broker. It is the only supported way for control-plane code to reach a
// privileged host action; nothing else in the tree may spawn sudo.
type Client struct {
	SocketPath string
	Timeout    time.Duration
	dial       func(ctx context.Context, socket string) (net.Conn, error)
}

// NewClient constructs a client for the default broker socket.
func NewClient() *Client {
	return &Client{
		SocketPath: DefaultSocketPath,
		Timeout:    tuning.PrivilegeBrokerOperationTimeout(),
		dial: func(ctx context.Context, socket string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socket)
		},
	}
}

// Available reports whether the broker socket accepts connections. A false
// result means the capability is absent, not that the request was denied.
func (c *Client) Available() bool {
	if c == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), tuning.PrivilegeBrokerUnlockTimeout())
	defer cancel()
	conn, err := c.dialer()(ctx, c.socket())
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Do sends one request and returns the broker's typed result. Validation runs
// client-side first so an obviously malformed request never reaches the root
// service, but the broker validates again and is the authority.
func (c *Client) Do(ctx context.Context, req Request) (Result, error) {
	if c == nil {
		return Result{}, fmt.Errorf("privilege broker client is not configured")
	}
	if err := Validate(req); err != nil {
		return Result{}, fmt.Errorf("invalid broker request: %w", err)
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = tuning.PrivilegeBrokerOperationTimeout()
	}
	ctx, cancel := context.WithTimeout(ctx, tuning.PrivilegeBrokerRequestTimeout(timeout))
	defer cancel()

	conn, err := c.dialer()(ctx, c.socket())
	if err != nil {
		return Result{}, fmt.Errorf("dial privilege broker: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Result{}, fmt.Errorf("send broker request: %w", err)
	}
	var result Result
	if err := json.NewDecoder(io.LimitReader(conn, clientParameterB<<clientParameterA)).Decode(&result); err != nil {
		return Result{}, fmt.Errorf("read broker response: %w", err)
	}
	return result, nil
}

func (c *Client) socket() string {
	if c.SocketPath == "" {
		return DefaultSocketPath
	}
	return c.SocketPath
}

func (c *Client) dialer() func(context.Context, string) (net.Conn, error) {
	if c.dial != nil {
		return c.dial
	}
	return func(ctx context.Context, socket string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, "unix", socket)
	}
}
