//go:build windows

package privsep

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const windowsPipePrefix = `\\.\pipe\`

type windowsPipeListener struct {
	name  string
	attrs windows.SecurityAttributes
	// Keep the descriptor alive for every pipe instance created by Accept.
	descriptor *windows.SECURITY_DESCRIPTOR
	closed     chan struct{}
	once       sync.Once
	mu         sync.Mutex
	pending    windows.Handle
}

func prepareIPCPath(socket string) error {
	if strings.TrimSpace(socket) == "" {
		return errors.New("provision IPC named pipe is required")
	}
	return nil
}

func listenIPC(socket, principal string) (net.Listener, error) {
	if strings.TrimSpace(principal) == "" {
		return nil, errors.New("Windows provisioning IPC requires an explicit client account")
	}
	name := windowsPipeName(socket)
	sid, _, _, err := windows.LookupSID("", principal)
	if err != nil {
		return nil, fmt.Errorf("resolve Windows provisioning IPC client account %q: %w", principal, err)
	}
	sddl := fmt.Sprintf("D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;%s)", sid.String())
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return nil, fmt.Errorf("build Windows provisioning IPC ACL: %w", err)
	}
	return &windowsPipeListener{
		name:       name,
		attrs:      windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: descriptor},
		descriptor: descriptor,
		closed:     make(chan struct{}),
	}, nil
}

func (l *windowsPipeListener) Accept() (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, net.ErrClosed
	default:
	}
	name, err := windows.UTF16PtrFromString(l.name)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateNamedPipe(
		name,
		windows.PIPE_ACCESS_DUPLEX,
		windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT|windows.PIPE_REJECT_REMOTE_CLIENTS,
		windows.PIPE_UNLIMITED_INSTANCES,
		64*1024,
		64*1024,
		0,
		&l.attrs,
	)
	if err != nil {
		return nil, fmt.Errorf("create Windows provisioning IPC pipe: %w", err)
	}
	l.mu.Lock()
	select {
	case <-l.closed:
		l.mu.Unlock()
		_ = windows.CloseHandle(handle)
		return nil, net.ErrClosed
	default:
		l.pending = handle
		l.mu.Unlock()
	}
	if err := windows.ConnectNamedPipe(handle, nil); err != nil && !errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
		l.mu.Lock()
		l.pending = 0
		closed := isPipeListenerClosed(l.closed)
		l.mu.Unlock()
		_ = windows.CloseHandle(handle)
		if closed {
			return nil, net.ErrClosed
		}
		return nil, fmt.Errorf("accept Windows provisioning IPC client: %w", err)
	}
	l.mu.Lock()
	l.pending = 0
	closed := isPipeListenerClosed(l.closed)
	l.mu.Unlock()
	if closed {
		_ = windows.CloseHandle(handle)
		return nil, net.ErrClosed
	}
	return &windowsPipeConn{file: os.NewFile(uintptr(handle), l.name)}, nil
}

func (l *windowsPipeListener) Close() error {
	l.once.Do(func() {
		close(l.closed)
		l.mu.Lock()
		pending := l.pending
		l.pending = 0
		l.mu.Unlock()
		if pending != 0 {
			_ = windows.CloseHandle(pending)
		}
	})
	return nil
}

func isPipeListenerClosed(closed <-chan struct{}) bool {
	select {
	case <-closed:
		return true
	default:
		return false
	}
}

func (l *windowsPipeListener) Addr() net.Addr { return pipeAddr(l.name) }

func removeIPC(string) error { return nil }

func dialIPC(ctx context.Context, socket string) (net.Conn, error) {
	name, err := windows.UTF16PtrFromString(windowsPipeName(socket))
	if err != nil {
		return nil, err
	}
	for {
		handle, openErr := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, 0, 0)
		if openErr == nil {
			return &windowsPipeConn{file: os.NewFile(uintptr(handle), windowsPipeName(socket))}, nil
		}
		if !errors.Is(openErr, windows.ERROR_PIPE_BUSY) && !errors.Is(openErr, windows.ERROR_FILE_NOT_FOUND) {
			return nil, fmt.Errorf("connect to Windows provisioning IPC pipe: %w", openErr)
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

type windowsPipeConn struct{ file *os.File }

func (c *windowsPipeConn) Read(p []byte) (int, error)  { return c.file.Read(p) }
func (c *windowsPipeConn) Write(p []byte) (int, error) { return c.file.Write(p) }
func (c *windowsPipeConn) Close() error                { return c.file.Close() }
func (c *windowsPipeConn) LocalAddr() net.Addr         { return pipeAddr(c.file.Name()) }
func (c *windowsPipeConn) RemoteAddr() net.Addr        { return pipeAddr(c.file.Name()) }
func (c *windowsPipeConn) SetDeadline(time.Time) error { return nil }
func (c *windowsPipeConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c *windowsPipeConn) SetWriteDeadline(time.Time) error {
	return nil
}

type pipeAddr string

func (a pipeAddr) Network() string { return "windows-named-pipe" }
func (a pipeAddr) String() string  { return string(a) }

func windowsPipeName(socket string) string {
	trimmed := strings.TrimSpace(socket)
	if strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(windowsPipePrefix)) {
		return trimmed
	}
	trimmed = strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(trimmed)
	trimmed = strings.Trim(trimmed, "-")
	return windowsPipePrefix + "vrooli-bridge-" + trimmed
}
