//go:build linux

package securestore

import (
	"context"
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/shell"
)

// UnlockLoginKeyring streams the operator's passphrase to the running GNOME
// keyring daemon. It deliberately accepts a reader rather than a string: the
// passphrase never becomes an argument, environment variable, temporary file,
// log message, or returned command output.
func UnlockLoginKeyring(ctx context.Context, input io.Reader) error {
	cmd := shell.NewCommandContext(ctx, "gnome-keyring-daemon", "--unlock")
	cmd.Env = sessionEnviron()
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
