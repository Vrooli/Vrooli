package auth

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type passwordSource struct {
	isTerminal func() bool
	readSecret func() ([]byte, error)
	stdin      io.Reader
	prompt     io.Writer
}

func newPasswordSource() passwordSource {
	fd := int(os.Stdin.Fd())
	return passwordSource{
		isTerminal: func() bool { return term.IsTerminal(fd) },
		readSecret: func() ([]byte, error) { return term.ReadPassword(fd) },
		stdin:      os.Stdin,
		prompt:     os.Stderr,
	}
}

func (p passwordSource) one(fromStdin bool) ([]byte, error) {
	if fromStdin {
		raw, err := io.ReadAll(p.stdin)
		if err != nil {
			return nil, fmt.Errorf("read password from stdin: %w", err)
		}
		return []byte(strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")), nil
	}
	if !p.isTerminal() {
		return nil, fmt.Errorf("password input needs a TTY; use --password-stdin for unattended use")
	}
	fmt.Fprint(p.prompt, "Password: ")
	secret, err := p.readSecret()
	fmt.Fprintln(p.prompt)
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}
	return secret, nil
}

func (p passwordSource) pair(fromStdin bool) (current, next []byte, err error) {
	if fromStdin {
		raw, readErr := io.ReadAll(p.stdin)
		if readErr != nil {
			return nil, nil, fmt.Errorf("read passwords from stdin: %w", readErr)
		}
		lines := strings.Split(strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r"), "\n")
		if len(lines) != 2 {
			return nil, nil, fmt.Errorf("--password-stdin expects current and new passwords on two newline-delimited lines")
		}
		return []byte(lines[0]), []byte(lines[1]), nil
	}
	if !p.isTerminal() {
		return nil, nil, fmt.Errorf("password input needs a TTY; use --password-stdin for unattended use")
	}
	fmt.Fprint(p.prompt, "Current password: ")
	current, err = p.readSecret()
	fmt.Fprintln(p.prompt)
	if err != nil {
		return nil, nil, fmt.Errorf("read current password: %w", err)
	}
	fmt.Fprint(p.prompt, "New password: ")
	next, err = p.readSecret()
	fmt.Fprintln(p.prompt)
	if err != nil {
		clear(current)
		return nil, nil, fmt.Errorf("read new password: %w", err)
	}
	return current, next, nil
}

func clear(secret []byte) {
	for i := range secret {
		secret[i] = 0
	}
}
