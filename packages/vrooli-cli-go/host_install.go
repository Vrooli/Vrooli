package vroolicli

import (
	"context"
	"fmt"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// HostInstall ensures a single host tool by name through `vrooli host install
// <tool> --json` and returns the typed status. Unlike most client methods it
// does NOT apply the default timeout: a url/release fetch can take minutes, so
// the caller's context owns the deadline (a durable server-side job, typically).
//
// `host install` exits non-zero for failed / manual / unsupported tools but
// still emits the typed status on stdout; HostInstall decodes that status
// regardless of exit code and only surfaces the exec error when no parseable
// status was produced.
func (c *Client) HostInstall(ctx context.Context, tool string, dryRun bool) (*cliv1.CliHostInstallStatus, error) {
	args := []string{"host", "install", tool, "--json"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	if !c.staleCheck {
		args = append([]string{"--no-stale-check"}, args...)
	}

	out, runErr := c.runner.Run(ctx, c.bin, args...)
	resp, decodeErr := decode(out, &cliv1.CliHostInstallStatus{})
	if decodeErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf("host install %q: %w", tool, runErr)
		}
		return nil, fmt.Errorf("host install %q: decode status: %w", tool, decodeErr)
	}
	return resp, nil
}
