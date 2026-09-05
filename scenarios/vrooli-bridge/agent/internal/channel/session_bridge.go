package channel

// This file is the node-side implementation of the session.Frame transport.
// It intentionally uses argv-shaped exec (no shell command string) and keeps
// the process handle behind the channel package. The control plane authenticates
// the request before it reaches this file; the agent authenticates every
// response with its existing node proof.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty/v2"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"

	"connectrpc.com/connect"
)

type nodeSession struct {
	id       string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	reader   io.Reader
	terminal *os.File
	cancel   context.CancelFunc
	mu       sync.Mutex
}

const sessionReportRetryWindow = 2 * time.Minute

func (c *Client) handleSessionFrame(envelope *sharedv1.SessionFrame) {
	if envelope == nil || envelope.GetSessionId() == "" || envelope.GetFrame() == nil {
		return
	}
	id := envelope.GetSessionId()
	frame := envelope.GetFrame()
	switch payload := frame.Payload.(type) {
	case *sessionv1.Frame_Open:
		c.openNodeSession(id, payload.Open)
	case *sessionv1.Frame_Data:
		c.writeNodeSession(id, payload.Data.GetData())
	case *sessionv1.Frame_Resize:
		c.resizeNodeSession(id, payload.Resize)
	case *sessionv1.Frame_Close:
		c.closeNodeSession(id, payload.Close.GetReason())
	}
}

func (c *Client) openNodeSession(id string, open *sessionv1.Open) {
	if open == nil {
		return
	}
	// A browser reconnect re-attaches to the existing Bridge session. Keep the
	// original PTY alive instead of replacing it, so input and scrollback retain
	// their session identity while the control-plane transport is repaired.
	c.mu.Lock()
	_, alreadyOpen := c.sessions[id]
	c.mu.Unlock()
	if alreadyOpen {
		return
	}
	shell := interactiveShell(open.GetShell())
	if shell == "" {
		_ = c.reportSessionFrame(id, &sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Reason: "shell_not_allowed"}}})
		return
	}
	c.closeNodeSession(id, "replaced")
	ctx, cancel := context.WithCancel(c.baseCtxOrBackground())
	cmd := exec.CommandContext(ctx, shell) // #nosec G204 -- interactiveShell allowlists the executable; no user-supplied argv or shell string reaches exec.
	cmd.Env = interactiveCommandEnv(c.cfg.VrooliBin, os.Environ())
	if dir := strings.TrimSpace(open.GetWorkingDir()); dir != "" {
		cmd.Dir = filepath.Clean(dir)
	}
	terminal, ptyErr := pty.StartWithSize(cmd, &pty.Winsize{Cols: 80, Rows: 24})
	if ptyErr != nil {
		c.logger.Printf("channel: session %q native pty start failed: %v; using pipe fallback", id, ptyErr)
		// pty.StartWithSize may have partially initialized the command before
		// returning an error. Do not reuse that Cmd: StdinPipe/StdoutPipe can
		// then report a misleading "already started" error and leave a blank
		// remote session. Recreate the command and start the bounded pipe
		// fallback from a clean process state.
		fallbackShell := shell
		if runtime.GOOS != "windows" {
			// /bin/sh is the POSIX baseline and avoids repeating a shell
			// implementation that just failed under the host's PTY layer.
			fallbackShell = "/bin/sh"
		}
		cmd = exec.CommandContext(ctx, fallbackShell, interactiveShellArgs()...) // #nosec G204 -- fallbackShell is a fixed platform shell or the previously validated shell.
		cmd.Env = interactiveCommandEnv(c.cfg.VrooliBin, os.Environ())
		if dir := strings.TrimSpace(open.GetWorkingDir()); dir != "" {
			cmd.Dir = filepath.Clean(dir)
		}
	}
	var stdin io.WriteCloser
	var reader io.Reader
	if ptyErr == nil {
		stdin, reader = terminal, terminal
	} else {
		// Windows and other unsupported targets retain a bounded pipe fallback.
		// The session contract stays identical, but resize is rejected instead
		// of being silently claimed to work.
		var err error
		stdin, err = cmd.StdinPipe()
		if err != nil {
			c.logger.Printf("channel: session %q pipe stdin setup failed: %v", id, err)
			cancel()
			return
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			c.logger.Printf("channel: session %q pipe stdout setup failed: %v", id, err)
			_ = stdin.Close()
			cancel()
			return
		}
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			c.logger.Printf("channel: session %q pipe start failed: %v", id, err)
			_ = stdin.Close()
			cancel()
			return
		}
		reader = stdout
	}
	s := &nodeSession{id: id, cmd: cmd, stdin: stdin, reader: reader, terminal: terminal, cancel: cancel}
	c.mu.Lock()
	if c.sessions == nil {
		c.sessions = make(map[string]*nodeSession)
	}
	c.sessions[id] = s
	c.mu.Unlock()
	go func() {
		buf := make([]byte, 32*1024)
		var seq uint64
		for {
			n, readErr := s.reader.Read(buf)
			if n > 0 {
				data := append([]byte(nil), buf[:n]...)
				frame := &sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: seq, Data: data}}}
				if err := c.reportSessionFrameWithRetry(id, frame); err != nil {
					c.logger.Printf("channel: session %q output report failed: %v", id, err)
					break
				}
				seq++
			}
			if readErr != nil {
				break
			}
		}
		_ = cmd.Wait()
		_ = c.reportSessionFrame(id, &sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Reason: "process_exit"}}})
		c.closeNodeSession(id, "process_exit")
	}()
}

// reportSessionFrameWithRetry keeps the PTY alive across a transient control
// plane/channel outage. Retrying the same sequence is safe because the Bridge
// manager treats an already-recorded identical frame as an idempotent replay.
// A bounded grace window still tears down a genuinely lost session rather than
// leaving a process orphaned forever.
func (c *Client) reportSessionFrameWithRetry(id string, frame *sessionv1.Frame) error {
	ctx, cancel := context.WithTimeout(c.baseCtxOrBackground(), sessionReportRetryWindow)
	defer cancel()
	backoff := 250 * time.Millisecond
	for {
		if err := c.reportSessionFrameContext(ctx, id, frame); err == nil {
			return nil
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("session frame report grace expired: %w", ctx.Err())
			}
			return ctx.Err()
		case <-timer.C:
		}
		if backoff < 5*time.Second {
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
	}
}

func (c *Client) resizeNodeSession(id string, resize *sessionv1.Resize) {
	if resize == nil || resize.GetColumns() == 0 || resize.GetRows() == 0 {
		return
	}
	c.mu.Lock()
	s := c.sessions[id]
	c.mu.Unlock()
	if s == nil || s.terminal == nil {
		return
	}
	cols, colsOK := ptyDimension(resize.GetColumns())
	rows, rowsOK := ptyDimension(resize.GetRows())
	if !colsOK || !rowsOK {
		c.logger.Printf("channel: session %q resize rejected: dimensions exceed PTY limit", id)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := pty.Setsize(s.terminal, &pty.Winsize{Cols: cols, Rows: rows}); err != nil {
		c.logger.Printf("channel: session %q resize failed: %v", id, err)
	}
}

func ptyDimension(value uint32) (uint16, bool) {
	if value == 0 || value > 65535 {
		return 0, false
	}
	return uint16(value), true // #nosec G115 -- the explicit upper bound is the uint16 maximum.
}

func (c *Client) writeNodeSession(id string, data []byte) {
	c.mu.Lock()
	s := c.sessions[id]
	c.mu.Unlock()
	if s == nil || len(data) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.stdin.Write(data)
}

func (c *Client) closeNodeSession(id, _ string) {
	c.mu.Lock()
	s := c.sessions[id]
	delete(c.sessions, id)
	c.mu.Unlock()
	if s == nil {
		return
	}
	s.cancel()
	_ = s.stdin.Close()
	_ = s.cmd.Process.Kill()
}

func (c *Client) reportSessionFrame(id string, frame *sessionv1.Frame) error {
	return c.reportSessionFrameContext(c.baseCtxOrBackground(), id, frame)
}

func (c *Client) reportSessionFrameContext(ctx context.Context, id string, frame *sessionv1.Frame) error {
	if c.rpc == nil {
		return errors.New("presence RPC is not configured")
	}
	req := connect.NewRequest(&presencev1.ReportSessionFrameRequest{Frame: &sharedv1.SessionFrame{SessionId: id, Frame: frame}})
	if c.cred != nil {
		for k, v := range c.cred.Headers(c.cfg.NodeID, c.now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	_, err := c.rpc.ReportSessionFrame(ctx, req)
	return err
}

func allowedInteractiveShell(shell string) bool {
	return interactiveShell(shell) != ""
}

func interactiveShell(shell string) string {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = os.Getenv("COMSPEC")
		} else {
			shell = os.Getenv("SHELL")
		}
		// A service account can inherit a stale or non-existent SHELL value
		// (for example, a macOS image without /bin/zsh). Remote sessions must
		// still get a usable host-owned shell, so validate the default and fall
		// back through the platform's conventional locations.
		if !shellExecutable(shell) {
			for _, candidate := range defaultInteractiveShells() {
				if shellExecutable(candidate) {
					shell = candidate
					break
				}
			}
		}
	}
	shell = filepath.Clean(shell)
	base := filepath.Base(shell)
	switch base {
	case "sh", "bash", "zsh", "fish", "cmd.exe", "powershell.exe", "pwsh.exe":
		return shell
	default:
		return ""
	}
}

func shellExecutable(shell string) bool {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		return false
	}
	if filepath.IsAbs(shell) {
		info, err := os.Stat(shell)
		return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
	}
	_, err := exec.LookPath(shell)
	return err == nil
}

func defaultInteractiveShells() []string {
	if runtime.GOOS == "windows" {
		return []string{"C:\\Windows\\System32\\cmd.exe", "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"}
	}
	return []string{"/bin/sh", "/bin/bash", "/usr/bin/sh", "/usr/bin/bash"}
}

func interactiveShellArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"/Q"}
	}
	// A pipe is not a terminal. POSIX shells otherwise exit immediately
	// instead of accepting the session's stdin stream.
	return []string{"-i"}
}

func interactiveCommandEnv(vrooliBin string, base []string) []string {
	env := append([]string(nil), base...)
	bin := strings.TrimSpace(vrooliBin)
	if bin == "" || !filepath.IsAbs(bin) {
		return env
	}
	dir := filepath.Dir(bin)
	if dir == "." || dir == string(filepath.Separator) {
		return env
	}

	pathValue := ""
	pathIndex := -1
	for i, value := range env {
		if strings.HasPrefix(value, "PATH=") {
			pathIndex = i
			pathValue = strings.TrimPrefix(value, "PATH=")
			break
		}
	}
	entries := strings.Split(pathValue, string(os.PathListSeparator))
	for _, entry := range entries {
		if entry == dir {
			return env
		}
	}
	pathValue = dir + string(os.PathListSeparator) + pathValue
	pathEntry := "PATH=" + pathValue
	if pathIndex >= 0 {
		env[pathIndex] = pathEntry
	} else {
		env = append(env, pathEntry)
	}
	return env
}
