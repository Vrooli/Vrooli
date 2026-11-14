//go:build linux

package target

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const isolatedPortRelay = `
import os
import select
import socket
import sys
import threading

socket_path = sys.argv[1]
try:
    os.unlink(socket_path)
except FileNotFoundError:
    pass

listener = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
listener.bind(socket_path)
os.chmod(socket_path, 0o600)
listener.listen(16)

def relay(client):
    upstream = None
    target_port = 0
    try:
        port_bytes = b''
        while len(port_bytes) < 2:
            part = client.recv(2 - len(port_bytes))
            if not part:
                return
            port_bytes += part
        target_port = int.from_bytes(port_bytes, 'big')
        if target_port < 1 or target_port > 65535:
            return
        upstream = socket.create_connection(('127.0.0.1', target_port), timeout=10)
        client.settimeout(None)
        upstream.settimeout(None)
        while True:
            ready, _, _ = select.select([client, upstream], [], [])
            for source in ready:
                data = source.recv(65536)
                destination = upstream if source is client else client
                if not data:
                    try:
                        destination.shutdown(socket.SHUT_WR)
                    except OSError:
                        pass
                    return
                destination.sendall(data)
    except OSError:
        try:
            with open(socket_path + '.errors', 'a', encoding='utf-8') as diagnostics:
                diagnostics.write('target_port=%s errno=%s\n' % (target_port, getattr(sys.exc_info()[1], 'errno', 'unknown')))
                diagnostics.flush()
            os.chmod(socket_path + '.errors', 0o600)
        except OSError:
            pass
    finally:
        client.close()
        if upstream is not None:
            upstream.close()

while True:
    connection, _ = listener.accept()
    threading.Thread(target=relay, args=(connection,), daemon=True).start()
`

const isolatedElectronShell = `
set -u
"$1" -c "$2" "$3" "$4" &
cdp_bridge_pid=$!
"$1" -c "$2" "$5" &
renderer_bridge_pid=$!
shift 5
"$@" &
app_pid=$!
wait "$app_pid"
app_status=$?
kill "$cdp_bridge_pid" "$renderer_bridge_pid" 2>/dev/null || true
wait "$cdp_bridge_pid" 2>/dev/null || true
wait "$renderer_bridge_pid" 2>/dev/null || true
exit "$app_status"
`

type unixCDPBridge struct {
	listener   net.Listener
	socketPath string
	targetPort int
	done       chan struct{}
	mu         sync.Mutex
	active     map[net.Conn]struct{}
	wg         sync.WaitGroup
}

func prepareOfflineNetworkRoot(existing string) (string, bool, error) {
	created := false
	root := existing
	if root == "" {
		var err error
		root, err = os.MkdirTemp("", "vrooli-electron-offline-")
		if err != nil {
			return "", false, fmt.Errorf("create private offline session root: %w", err)
		}
		created = true
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		if created {
			_ = os.RemoveAll(root)
		}
		return "", false, fmt.Errorf("private offline session root is not a private directory")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		if created {
			_ = os.RemoveAll(root)
		}
		return "", false, fmt.Errorf("private offline session root must be absolute and clean")
	}
	for _, name := range []string{"tmp", "cache", "profile", "test-database"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o700); err != nil {
			if created {
				_ = os.RemoveAll(root)
			}
			return "", false, fmt.Errorf("create private offline %s directory: %w", name, err)
		}
	}
	return root, created, nil
}

func pathWithin(root, path string) bool {
	root, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func startUnixCDPBridge(socketPath string, requestedPort int) (*unixCDPBridge, error) {
	targetPort := requestedPort
	if targetPort == 0 {
		// CDP's inner port is selected from the host listener below so Electron
		// and the namespace relay agree on one port.
	}
	return startUnixPortBridge(socketPath, requestedPort, targetPort)
}

func startUnixPortBridge(socketPath string, requestedPort, targetPort int) (*unixCDPBridge, error) {
	if len(socketPath) >= 104 {
		return nil, fmt.Errorf("private CDP socket path is too long")
	}
	address := "127.0.0.1:0"
	if requestedPort > 0 {
		address = fmt.Sprintf("127.0.0.1:%d", requestedPort)
	}
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return nil, fmt.Errorf("listen on private CDP loopback: %w", err)
	}
	if targetPort == 0 {
		targetPort = listener.Addr().(*net.TCPAddr).Port
	}
	if targetPort < 1 || targetPort > 65535 {
		_ = listener.Close()
		return nil, fmt.Errorf("private bridge target port is invalid")
	}
	bridge := &unixCDPBridge{listener: listener, socketPath: socketPath, targetPort: targetPort, done: make(chan struct{}), active: make(map[net.Conn]struct{})}
	go bridge.serve()
	return bridge, nil
}

func (b *unixCDPBridge) Port() int {
	return b.listener.Addr().(*net.TCPAddr).Port
}

func (b *unixCDPBridge) Endpoint() string {
	return "http://" + b.listener.Addr().String()
}

func (b *unixCDPBridge) serve() {
	defer close(b.done)
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			return
		}
		b.mu.Lock()
		b.active[conn] = struct{}{}
		b.mu.Unlock()
		b.wg.Add(1)
		go b.forward(conn)
	}
}

func (b *unixCDPBridge) forward(client net.Conn) {
	defer b.wg.Done()
	defer func() {
		b.mu.Lock()
		delete(b.active, client)
		b.mu.Unlock()
		_ = client.Close()
	}()
	upstream, err := net.DialTimeout("unix", b.socketPath, 2*time.Second)
	if err != nil {
		return
	}
	var portHeader [2]byte
	binary.BigEndian.PutUint16(portHeader[:], uint16(b.targetPort))
	if _, err := upstream.Write(portHeader[:]); err != nil {
		_ = upstream.Close()
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

func (b *unixCDPBridge) Close() error {
	if b == nil {
		return nil
	}
	err := b.listener.Close()
	<-b.done
	b.mu.Lock()
	for conn := range b.active {
		_ = conn.Close()
	}
	b.mu.Unlock()
	b.wg.Wait()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func isolatedElectronCommand(root, profile, cdpSocketPath, rendererSocketPath string, port int, artifact string, electronArgs []string) (string, []string, error) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return "", nil, fmt.Errorf("offline Electron validation requires bubblewrap: %w", err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		return "", nil, fmt.Errorf("offline Electron CDP bridge requires python3: %w", err)
	}
	if !pathWithin(root, profile) || !pathWithin(root, cdpSocketPath) || !pathWithin(root, rendererSocketPath) || port < 1 || port > 65535 {
		return "", nil, fmt.Errorf("offline Electron launch has an invalid private root, socket, or CDP port")
	}
	args := []string{
		"--unshare-net", "--ro-bind", "/", "/", "--proc", "/proc", "--dev", "/dev",
		"--tmpfs", "/dev/shm", "--bind", root, root,
		"--setenv", "HOME", profile, "--setenv", "TMPDIR", filepath.Join(root, "tmp"),
		"--setenv", "XDG_CACHE_HOME", filepath.Join(root, "cache"),
		"--", "/bin/sh", "-c", isolatedElectronShell, "vrooli-offline-electron",
		python, isolatedPortRelay, cdpSocketPath, fmt.Sprint(port), rendererSocketPath, artifact,
	}
	args = append(args, electronArgs...)
	return bwrap, args, nil
}
