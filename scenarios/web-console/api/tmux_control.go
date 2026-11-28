//go:build !windows

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ErrControlChannelUnavailable means that a command was not submitted to the
// tmux control client. Callers must not replay the command implicitly: the
// next command may start a new client, but the failed command is lost.
var ErrControlChannelUnavailable = errors.New("tmux control channel unavailable")

type tmuxControlReply struct {
	body string
	err  error
}

// tmuxControl is one long-lived tmux -C client. tmux emits asynchronous
// notifications (for example %output) between tagged command replies; the
// reader consumes those notifications and only forwards %begin/%end replies.
type tmuxControl struct {
	sessionName string
	mu          sync.Mutex
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	replies     chan tmuxControlReply
	dead        chan struct{}
	startup     chan struct{}
	closed      bool
	closeOnce   sync.Once
	nextTag     uint64
}

func newTmuxControl(sessionName string) (*tmuxControl, error) {
	c := &tmuxControl{sessionName: sessionName}
	if err := c.start(context.Background()); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *tmuxControl) start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startLocked(ctx)
}

func (c *tmuxControl) startLocked(ctx context.Context) error {
	if c.closed {
		return ErrControlChannelUnavailable
	}
	if c.cmd != nil {
		select {
		case <-c.dead:
			c.cmd = nil
			c.stdin = nil
		default:
			return nil
		}
	}

	cmd := tmuxCmd("-C", "attach-session", "-t", c.sessionName)
	cmd.Env = ensureTermEnv(os.Environ())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("%w: stdin: %v", ErrControlChannelUnavailable, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("%w: stdout: %v", ErrControlChannelUnavailable, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("%w: stderr: %v", ErrControlChannelUnavailable, err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("%w: start: %v", ErrControlChannelUnavailable, err)
	}

	dead := make(chan struct{})
	startup := make(chan struct{})
	replies := make(chan tmuxControlReply, 8)
	c.cmd = cmd
	c.stdin = stdin
	c.dead = dead
	c.startup = startup
	c.replies = replies
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	go c.readReplies(stdout, replies, dead, startup)

	startupTimeout := time.NewTimer(2 * time.Second)
	defer startupTimeout.Stop()
	select {
	case <-startup:
		return nil
	case <-dead:
		return fmt.Errorf("%w: control client exited during attach", ErrControlChannelUnavailable)
	case <-startupTimeout.C:
		return fmt.Errorf("%w: attach handshake timed out", ErrControlChannelUnavailable)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *tmuxControl) readReplies(stdout io.Reader, replies chan<- tmuxControlReply, dead chan<- struct{}, startup chan<- struct{}) {
	defer close(dead)
	reader := bufio.NewReader(stdout)
	startupPending := true
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(line, "%begin ") {
			// %output, %session-changed, %layout-change and %exit are
			// notifications. The PTY attach stream is the source of pane
			// output, so this control client intentionally ignores them.
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		tag := parts[1]
		body := make([]string, 0, 4)
		failed := false
		for {
			bodyLine, bodyErr := reader.ReadString('\n')
			if bodyErr != nil {
				return
			}
			bodyLine = strings.TrimRight(bodyLine, "\r\n")
			switch {
			case strings.HasPrefix(bodyLine, "%end "):
				endParts := strings.Fields(bodyLine)
				if len(endParts) > 1 && endParts[1] == tag {
					goto replyComplete
				}
			case strings.HasPrefix(bodyLine, "%error "):
				errorParts := strings.Fields(bodyLine)
				if len(errorParts) > 1 && errorParts[1] == tag {
					failed = true
					body = append(body, strings.TrimSpace(strings.TrimPrefix(bodyLine, "%error ")))
					goto replyComplete
				}
			default:
				body = append(body, bodyLine)
			}
		}

	replyComplete:
		reply := tmuxControlReply{body: strings.Join(body, "\n")}
		if failed {
			reply.err = fmt.Errorf("tmux control command failed: %s", reply.body)
		}
		if startupPending {
			startupPending = false
			close(startup)
			continue
		}
		select {
		case replies <- reply:
		}
	}
}

// tmuxControlQuote encodes one argument for tmux's control-mode command
// parser. Single quotes preserve spaces, backslashes, control bytes and tmux
// format markers. Newlines are handled by deliverControlText because the
// control protocol itself is line-oriented.
func tmuxControlQuote(value string) string {
	var b strings.Builder
	b.Grow(len(value) + 2)
	b.WriteByte('\'')
	for _, r := range value {
		switch r {
		case '\'':
			b.WriteString(`'\''`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

func (c *tmuxControl) Exec(ctx context.Context, args ...string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return "", ErrControlChannelUnavailable
	}
	if err := c.startLocked(ctx); err != nil {
		return "", err
	}
	c.nextTag++
	command := make([]string, 0, len(args))
	for _, arg := range args {
		command = append(command, tmuxControlQuote(arg))
	}
	commandLine := strings.Join(command, " ") + "\n"
	if _, err := io.WriteString(c.stdin, commandLine); err != nil {
		return "", fmt.Errorf("%w: write: %v", ErrControlChannelUnavailable, err)
	}
	select {
	case reply := <-c.replies:
		if reply.err != nil {
			return "", reply.err
		}
		return reply.body, nil
	case <-c.dead:
		return "", ErrControlChannelUnavailable
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *tmuxControl) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		stdin := c.stdin
		cmd := c.cmd
		c.mu.Unlock()
		if stdin != nil {
			_ = stdin.Close()
		}
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	return nil
}
