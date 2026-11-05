package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const DefaultPrivilegeBrokerSocketPath = "/run/vrooli/privilege-broker.sock"

const TemporaryActivityObserveAction = "storage.temporary-activity.observe"

// PrivilegeBrokerTransport carries one bounded JSON request over the local
// setup-owned socket. Action validation stays in the control-plane registry;
// this transport never runs a command or attempts elevation.
type PrivilegeBrokerTransport struct {
	SocketPath       string
	Timeout          time.Duration
	MaxResponseBytes int64
	verifyPeer       func(net.Conn) error
}

func (c PrivilegeBrokerTransport) Exchange(ctx context.Context, request, response any) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("setup-managed privilege broker transport requires Linux")
	}
	raw, err := json.Marshal(request)
	if err != nil || len(raw) > 16<<10 {
		return fmt.Errorf("privilege broker request exceeds its bound or cannot be encoded")
	}
	limit := c.MaxResponseBytes
	if limit <= 0 {
		limit = 64 << 10
	}
	if limit > 8<<20 {
		return fmt.Errorf("privilege broker response limit exceeds its bound")
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	socket := c.SocketPath
	if socket == "" {
		socket = PrivilegeBrokerSocketPath()
	}
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", socket)
	if err != nil {
		return fmt.Errorf("setup-managed privilege broker unavailable; run vrooli setup --sudo-mode=ask: %w", err)
	}
	defer conn.Close()
	verify := c.verifyPeer
	if verify == nil {
		verify = verifyPrivilegeBrokerPeer
	}
	if err := verify(conn); err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	if _, err := io.Copy(conn, bytes.NewReader(append(raw, '\n'))); err != nil {
		return fmt.Errorf("send privilege broker request: %w", err)
	}
	// The broker closes after its one response. Reading limit+1 distinguishes
	// complete JSON from a truncated or unbounded response.
	data, err := io.ReadAll(io.LimitReader(conn, limit+1))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("read privilege broker response: %w", err)
	}
	if int64(len(data)) > limit {
		return fmt.Errorf("privilege broker response exceeds its bound")
	}
	if err := json.Unmarshal(data, response); err != nil {
		return fmt.Errorf("decode privilege broker response: %w", err)
	}
	return nil
}

// PrivilegeBrokerSocketPath is the single cross-module socket policy. The
// Linux broker is a system service, so its endpoint must not depend on the
// caller's user-session environment (in particular XDG_RUNTIME_DIR).
func PrivilegeBrokerSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\vrooli-privilege-broker`
	}
	if configured := strings.TrimSpace(os.Getenv("VROOLI_PRIVILEGE_BROKER_SOCKET")); configured != "" {
		return configured
	}
	if runtime.GOOS == "linux" {
		return DefaultPrivilegeBrokerSocketPath
	}
	if runtimeRoot := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeRoot != "" {
		return filepath.Join(runtimeRoot, "vrooli", "privilege-broker.sock")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.TempDir(), "vrooli", "privilege-broker.sock")
	}
	return DefaultPrivilegeBrokerSocketPath
}
