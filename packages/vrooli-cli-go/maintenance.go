package vroolicli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// Orphans returns the typed `vrooli orphans --json` response: Vrooli processes
// running on the host with no live supervising parent. This is the read-only
// inspection list; killing them is a separate mutation (see CleanupOrphans).
func (c *Client) Orphans(ctx context.Context) (*cliv1.ProjectOrphansResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "orphans", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ProjectOrphansResponse{})
	if err != nil {
		return nil, fmt.Errorf("orphans: %w", err)
	}
	return resp, nil
}

// DiagnosePort returns the typed `vrooli diagnose-port <port> [scenario] --json`
// response: listener evidence, registry claims, legacy lock artifacts, and
// port-ownership recommendations for one port. scenario is optional context for
// the diagnosis and may be empty.
func (c *Client) DiagnosePort(ctx context.Context, port int, scenario string) (*cliv1.ProjectPortDiagnosticResponse, error) {
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("diagnose-port: port %d out of range [1, 65535]", port)
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	args := []string{"diagnose-port", strconv.Itoa(port)}
	if s := strings.TrimSpace(scenario); s != "" {
		args = append(args, s)
	}
	args = append(args, "--json")

	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ProjectPortDiagnosticResponse{})
	if err != nil {
		return nil, fmt.Errorf("diagnose-port %d: %w", port, err)
	}
	return resp, nil
}

// CleanupOrphans runs `vrooli cleanup orphans --json` (a MUTATION: it SIGTERMs,
// then SIGKILLs, orphaned Vrooli processes) and returns the typed stop report of
// what was stopped and what failed.
func (c *Client) CleanupOrphans(ctx context.Context) (*cliv1.ProjectStopResponse, error) {
	return c.cleanup(ctx, "orphans")
}

// CleanupLocks runs `vrooli cleanup locks --json` (a MUTATION: it expires stale
// registry claims and prunes legacy port-lock artifacts) and returns the typed
// stop report.
func (c *Client) CleanupLocks(ctx context.Context) (*cliv1.ProjectStopResponse, error) {
	return c.cleanup(ctx, "locks")
}

// cleanup runs `vrooli cleanup <target> --json` and decodes the shared stop
// report envelope both cleanup subcommands emit.
func (c *Client) cleanup(ctx context.Context, target string) (*cliv1.ProjectStopResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "cleanup", target, "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ProjectStopResponse{})
	if err != nil {
		return nil, fmt.Errorf("cleanup %s: %w", target, err)
	}
	return resp, nil
}
