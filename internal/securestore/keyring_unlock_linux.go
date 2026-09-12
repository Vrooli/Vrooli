//go:build linux

package securestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/vrooli/vrooli/internal/shell"
)

// ErrNoKeyringDaemon reports that no gnome-keyring-daemon is listening in the
// session this process can reach.
var ErrNoKeyringDaemon = errors.New("no keyring daemon is running in this session")

// keyringControlSocket is where gnome-keyring-daemon listens for control
// requests (`--control-directory=%t/keyring` in its systemd unit).
const keyringControlSocket = "keyring/control"

// UnlockLoginKeyring streams the operator's passphrase to the running GNOME
// keyring daemon. It deliberately accepts a reader rather than a string: the
// passphrase never becomes an argument, environment variable, temporary file,
// log message, or returned command output.
//
// It refuses to run when no daemon is listening in the session it would
// address. `gnome-keyring-daemon --unlock` with no daemon to talk to starts a
// new one; from an SSH session that new daemon claims the control directory
// under /run/user/<uid> and competes with the console session's daemon for
// every later client, which is one of the ways a host ends up with two
// daemons and a sign-in that hangs. Refusing is always recoverable; a second
// daemon is not, short of a re-login.
func UnlockLoginKeyring(ctx context.Context, input io.Reader) error {
	environ := sessionEnviron()
	if !keyringDaemonListening(environ) {
		return fmt.Errorf("unlock login keyring: %w; unlocking from here would start a second daemon, so sign in to the graphical session (or start gnome-keyring-daemon.service in the user manager) first", ErrNoKeyringDaemon)
	}
	cmd := shell.NewCommandContext(ctx, "gnome-keyring-daemon", "--unlock")
	cmd.Env = environ
	cmd.Stdin = input
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("unlock login keyring: %w", ctx.Err())
		}
		return fmt.Errorf("unlock login keyring: gnome-keyring-daemon failed")
	}
	return nil
}

// keyringDaemonListening reports whether the daemon's control socket exists in
// the runtime directory the subprocess will use. environ is the repaired
// environment (nil means "inherit unchanged"), so the check looks where the
// daemon invocation itself will look.
func keyringDaemonListening(environ []string) bool {
	runtimeDir := envValue(environ, "XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = filepath.Join(runtimeRoot, strconv.Itoa(os.Getuid()))
	}
	info, err := os.Stat(filepath.Join(runtimeDir, keyringControlSocket))
	return err == nil && info.Mode()&os.ModeSocket != 0
}

func envValue(environ []string, name string) string {
	if environ == nil {
		return os.Getenv(name)
	}
	prefix := name + "="
	for _, entry := range environ {
		if len(entry) > len(prefix) && entry[:len(prefix)] == prefix {
			return entry[len(prefix):]
		}
	}
	return ""
}
