package onboard

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// sshPasswordEnvVar is the non-interactive path for the owner's SSH password.
// It is the ONLY way to supply the password without a TTY prompt — deliberately
// an env var, never a flag, so the single-use secret never lands in argv (where
// `ps` would expose it to any local user) or shell history.
const sshPasswordEnvVar = "BRIDGE_SSH_PASSWORD"

// passwordSource resolves the SSH password without ever reading it from argv.
// The seams (env, isTerminal, readSecret) are injected so the resolution logic
// is unit-testable without a real TTY.
type passwordSource struct {
	lookupEnv  func(string) (string, bool)
	isTerminal func() bool
	readSecret func() ([]byte, error)
	prompt     io.Writer
}

// newPasswordSource wires passwordSource to the real process: $BRIDGE_SSH_PASSWORD,
// stdin's TTY state, and a masked terminal read. The prompt text goes to stderr
// so it never contaminates piped --json stdout.
func newPasswordSource() passwordSource {
	fd := int(os.Stdin.Fd())
	return passwordSource{
		lookupEnv:  os.LookupEnv,
		isTerminal: func() bool { return term.IsTerminal(fd) },
		readSecret: func() ([]byte, error) { return term.ReadPassword(fd) },
		prompt:     os.Stderr,
	}
}

// resolve returns the SSH password to send once in the StartOnboarding request.
// Precedence:
//  1. $BRIDGE_SSH_PASSWORD, when set (the non-interactive/programmatic path) —
//     honoured even if empty, which means "the host already trusts the bridge key".
//  2. An interactive masked prompt, when stdin is a TTY. A blank entry is valid
//     and means the same "already key-trusted" case.
//  3. Empty, when there is neither an env var nor a TTY — assume the host is
//     already key-trusted rather than blocking a non-interactive run.
//
// The returned string is passed straight into the request body; it is never
// placed on argv. An empty result is legitimate (the API treats it as "no
// first-touch needed").
func (p passwordSource) resolve(user, host string) (string, error) {
	if v, ok := p.lookupEnv(sshPasswordEnvVar); ok {
		return v, nil
	}
	if !p.isTerminal() {
		return "", nil
	}
	who := user
	if who == "" {
		who = "root"
	}
	fmt.Fprintf(p.prompt, "SSH password for %s@%s (leave blank if the host already trusts the bridge key): ", who, host)
	secret, err := p.readSecret()
	// ReadPassword consumes the trailing newline without echoing it; print our
	// own so the next line of output is not glued to the prompt.
	fmt.Fprintln(p.prompt)
	if err != nil {
		return "", fmt.Errorf("read SSH password: %w", err)
	}
	return string(secret), nil
}
