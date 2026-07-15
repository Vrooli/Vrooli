package onboard

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// sshPasswordEnvVar is the ambient non-interactive path for the owner's SSH
// password. It is deliberately an env var, never a flag, so the single-use
// secret never lands in argv (where `ps` would expose it to any local user)
// or shell history.
const sshPasswordEnvVar = "BRIDGE_SSH_PASSWORD"

// credentialSource names where the resolved password came from. It is
// non-secret metadata: the start report echoes it so the operator can see
// which intake path won without ever seeing the secret itself.
type credentialSource string

const (
	credentialFromStdin  credentialSource = "--password-stdin"
	credentialFromPrompt credentialSource = "--prompt-password"
	credentialFromEnv    credentialSource = "$" + sshPasswordEnvVar
	credentialNone       credentialSource = "none"
)

// passwordSource resolves the SSH password without ever reading it from argv.
// The seams (env, isTerminal, readSecret, stdin) are injected so the
// resolution logic is unit-testable without a real TTY or process stdin.
type passwordSource struct {
	lookupEnv  func(string) (string, bool)
	isTerminal func() bool
	readSecret func() ([]byte, error)
	stdin      io.Reader
	prompt     io.Writer
}

// newPasswordSource wires passwordSource to the real process:
// $BRIDGE_SSH_PASSWORD, stdin's TTY state, a masked terminal read, and process
// stdin for --password-stdin. The prompt text goes to stderr so it never
// contaminates piped --json stdout.
func newPasswordSource() passwordSource {
	fd := int(os.Stdin.Fd())
	return passwordSource{
		lookupEnv:  os.LookupEnv,
		isTerminal: func() bool { return term.IsTerminal(fd) },
		readSecret: func() ([]byte, error) { return term.ReadPassword(fd) },
		stdin:      os.Stdin,
		prompt:     os.Stderr,
	}
}

// resolve returns the SSH password to send once in the StartOnboarding request
// body, plus the (non-secret) source it came from. `start` NEVER prompts unless
// explicitly asked: an unattended run cannot hang on a question, and an
// interactive one only sees a prompt it opted into. Precedence:
//
//  1. --password-stdin: read the whole of stdin (pipe it from a secret manager
//     or `read -s`); one trailing newline is stripped.
//  2. --prompt-password: a masked TTY prompt, explicit opt-in only. Errors
//     without a TTY rather than silently degrading.
//  3. $BRIDGE_SSH_PASSWORD, when set — honoured even if empty, which means
//     "the host already trusts the bridge key".
//  4. Empty: assume the host is already key-trusted. The op fails at ssh-setup
//     with guidance naming every intake path if that assumption is wrong.
//
// The two flags are mutually exclusive; passing both is a usage error. The
// returned string goes straight into the request body — never argv.
func (p passwordSource) resolve(user, host string, fromStdin, promptRequested bool) (string, credentialSource, error) {
	if fromStdin && promptRequested {
		return "", credentialNone, fmt.Errorf("--password-stdin and --prompt-password are mutually exclusive: choose one credential path")
	}
	if fromStdin {
		raw, err := io.ReadAll(p.stdin)
		if err != nil {
			return "", credentialNone, fmt.Errorf("read SSH password from stdin: %w", err)
		}
		return trimTrailingNewline(string(raw)), credentialFromStdin, nil
	}
	if promptRequested {
		if !p.isTerminal() {
			return "", credentialNone, fmt.Errorf("--prompt-password needs a TTY: pipe the password via --password-stdin or set $%s instead", sshPasswordEnvVar)
		}
		who := user
		if who == "" {
			who = "root"
		}
		fmt.Fprintf(p.prompt, "SSH password for %s@%s (leave blank if the host already trusts the bridge key): ", who, host)
		secret, err := p.readSecret()
		// ReadPassword consumes the trailing newline without echoing it; print
		// our own so the next line of output is not glued to the prompt.
		fmt.Fprintln(p.prompt)
		if err != nil {
			return "", credentialNone, fmt.Errorf("read SSH password: %w", err)
		}
		return string(secret), credentialFromPrompt, nil
	}
	if v, ok := p.lookupEnv(sshPasswordEnvVar); ok {
		return v, credentialFromEnv, nil
	}
	return "", credentialNone, nil
}

// trimTrailingNewline strips exactly one trailing LF or CRLF — the newline the
// shell pipe appends — while preserving any other byte that is genuinely part
// of the password (including a bare trailing CR).
func trimTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		s = strings.TrimSuffix(s, "\n")
		s = strings.TrimSuffix(s, "\r")
	}
	return s
}
