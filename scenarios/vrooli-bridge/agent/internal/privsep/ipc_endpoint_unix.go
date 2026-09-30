//go:build !windows

package privsep

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func prepareIPCPath(socket string) error {
	if err := os.MkdirAll(filepath.Dir(socket), 0o750); err != nil {
		return fmt.Errorf("create provisioning IPC directory: %w", err)
	}
	info, err := os.Lstat(socket)
	if err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("refusing to replace non-socket provisioning IPC path %q", socket)
		}
		if err := os.Remove(socket); err != nil {
			return fmt.Errorf("remove stale provisioning IPC socket: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect provisioning IPC path: %w", err)
	}
	return nil
}

func listenIPC(socket, _ string) (net.Listener, error) {
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, err
	}
	// The socket mode permits the separately-owned runner to connect; peer UID
	// verification remains the authorization boundary on Unix.
	if err := os.Chmod(socket, 0o666); err != nil { // #nosec G302 -- peer credentials authorize the runner.
		_ = listener.Close()
		return nil, fmt.Errorf("secure provisioning IPC socket: %w", err)
	}
	return listener, nil
}

func removeIPC(socket string) error { return os.Remove(socket) }

func dialIPC(ctx context.Context, socket string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", socket)
}
