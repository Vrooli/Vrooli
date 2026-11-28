//go:build linux

package target

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type unixUpstreamBridge struct {
	listener   net.Listener
	socketPath string
	target     string
	done       chan struct{}
	mu         sync.Mutex
	active     map[net.Conn]struct{}
	wg         sync.WaitGroup
}

func startUnixUpstreamBridge(socketPath, hostURL string) (*unixUpstreamBridge, error) {
	parsed, err := validateOfflineUpstreamURL(hostURL)
	if err != nil {
		return nil, err
	}
	if len(socketPath) >= 104 {
		return nil, fmt.Errorf("private upstream socket path is too long")
	}
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("clear stale private upstream socket: %w", err)
	}
	if err := os.Remove(socketPath + ".ready"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("clear stale private upstream readiness marker: %w", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on private upstream socket: %w", err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = listener.Close()
		_ = os.Remove(socketPath)
		return nil, fmt.Errorf("protect private upstream socket: %w", err)
	}
	bridge := &unixUpstreamBridge{
		listener: listener, socketPath: socketPath,
		target: net.JoinHostPort(parsed.Hostname(), parsed.Port()),
		done:   make(chan struct{}), active: make(map[net.Conn]struct{}),
	}
	go bridge.serve()
	return bridge, nil
}

func (b *unixUpstreamBridge) serve() {
	defer close(b.done)
	for {
		connection, err := b.listener.Accept()
		if err != nil {
			return
		}
		b.mu.Lock()
		b.active[connection] = struct{}{}
		b.mu.Unlock()
		b.wg.Add(1)
		go b.forward(connection)
	}
}

func (b *unixUpstreamBridge) forward(client net.Conn) {
	defer b.wg.Done()
	defer func() {
		b.mu.Lock()
		delete(b.active, client)
		b.mu.Unlock()
		_ = client.Close()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	upstream, err := (&net.Dialer{}).DialContext(ctx, "tcp4", b.target)
	if err != nil {
		return
	}
	b.mu.Lock()
	b.active[upstream] = struct{}{}
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.active, upstream)
		b.mu.Unlock()
		_ = upstream.Close()
	}()
	finished := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(upstream, client)
		if c, ok := upstream.(interface{ CloseWrite() error }); ok {
			_ = c.CloseWrite()
		}
		finished <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, upstream)
		if c, ok := client.(interface{ CloseWrite() error }); ok {
			_ = c.CloseWrite()
		}
		finished <- struct{}{}
	}()
	<-finished
	_ = client.Close()
	_ = upstream.Close()
	<-finished
}

func (b *unixUpstreamBridge) Close() error {
	if b == nil {
		return nil
	}
	err := b.listener.Close()
	<-b.done
	b.mu.Lock()
	for connection := range b.active {
		_ = connection.Close()
	}
	b.mu.Unlock()
	b.wg.Wait()
	_ = os.Remove(b.socketPath)
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func offlineUpstreamSocketPath(root string, index int) string {
	return filepath.Join(root, fmt.Sprintf("offline-upstream-%d.sock", index))
}
