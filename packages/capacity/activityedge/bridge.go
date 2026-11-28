package activityedge

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// BridgeCommand runs the only TCP listener for a private native backend. The
// listener is in the backend's rootless network namespace; the host activity
// edge reaches it over a protected Unix socket.
func BridgeCommand() cliapp.Command {
	return cliapp.Command{Name: "upstream-bridge", Description: "Expose a private backend only to its owning activity edge", Usage: "upstream-bridge --socket path --upstream loopback:port --peer-pid pid", Run: runBridge}
}

func runBridge(args []string) error {
	fs := flag.NewFlagSet("upstream-bridge", flag.ContinueOnError)
	socket := fs.String("socket", "", "host-shared private Unix socket")
	upstream := fs.String("upstream", "", "private loopback backend")
	peerPID := fs.Int("peer-pid", 0, "exact host activity-edge PID allowed to connect")
	parentPID := fs.Int("parent-pid", 0, "managed RootlessKit owner PID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !filepath.IsAbs(*socket) || filepath.Clean(*socket) != *socket || len(*socket) > 100 {
		return fmt.Errorf("bridge socket path must be absolute, clean, and shorter than 101 bytes")
	}
	host, port, err := net.SplitHostPort(*upstream)
	if err != nil || (host != "127.0.0.1" && host != "localhost") {
		return fmt.Errorf("bridge upstream must be a loopback TCP address")
	}
	if p, e := strconv.Atoi(port); e != nil || p < 1 || p > 65535 {
		return fmt.Errorf("bridge upstream port is invalid")
	}
	if *peerPID <= 1 {
		return fmt.Errorf("bridge peer PID must be greater than one")
	}
	if err := os.MkdirAll(filepath.Dir(*socket), 0700); err != nil {
		return err
	}
	if info, err := os.Lstat(*socket); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("bridge socket path exists and is not a socket")
		}
		conn, dialErr := net.DialTimeout("unix", *socket, 150*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			return fmt.Errorf("bridge socket is already served by a live process")
		}
		if err := os.Remove(*socket); err != nil {
			return fmt.Errorf("remove stale bridge socket: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	ln, err := net.Listen("unix", *socket)
	if err != nil {
		return err
	}
	defer func() { _ = ln.Close(); _ = os.Remove(*socket) }()
	if err := os.Chmod(*socket, 0600); err != nil {
		return err
	}
	if *parentPID > 1 && runtime.GOOS == "linux" {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(*parentPID))); err != nil {
					_ = ln.Close()
					return
				}
			}
		}()
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			if *parentPID > 1 && runtime.GOOS == "linux" {
				if _, statErr := os.Stat(filepath.Join("/proc", strconv.Itoa(*parentPID))); os.IsNotExist(statErr) {
					return nil
				}
			}
			return err
		}
		go func(client net.Conn) {
			defer client.Close()
			if err := authorizeUnixPeer(client, *peerPID, os.Getuid()); err != nil {
				return
			}
			_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
			reader := bufio.NewReader(client)
			if _, err := reader.Peek(1); err != nil {
				return
			}
			_ = client.SetReadDeadline(time.Time{})
			backend, err := net.DialTimeout("tcp", *upstream, 2*time.Second)
			if err != nil {
				return
			}
			defer backend.Close()
			copyDone := make(chan struct{}, 1)
			go func() {
				_, _ = io.Copy(backend, reader)
				if c, ok := backend.(*net.TCPConn); ok {
					_ = c.CloseWrite()
				}
				copyDone <- struct{}{}
			}()
			_, _ = io.Copy(client, backend)
			<-copyDone
		}(conn)
	}
}

func authorizeUnixPeer(conn net.Conn, expectedPID, expectedUID int) error {
	if expectedPID <= 1 || expectedUID < 0 {
		return fmt.Errorf("bridge peer identity is invalid")
	}
	pid, uid, err := unixPeerCredentials(conn)
	if err != nil {
		return err
	}
	if pid != expectedPID || uid != expectedUID {
		return fmt.Errorf("Unix bridge peer identity does not match the activity-edge owner")
	}
	return nil
}
