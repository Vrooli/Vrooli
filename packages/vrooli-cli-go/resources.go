package vroolicli

import (
	"context"
	"fmt"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ListResources returns the typed `vrooli resource list --json` response. It
// prefers --verbose and falls back to plain output, but only while the
// operation's shared deadline is still alive — a timed-out first attempt is not
// retried (that would just wait again).
func (c *Client) ListResources(ctx context.Context) (*cliv1.ResourceListResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "resource", "list", "--json", "--verbose")
	if err != nil && ctx.Err() == nil {
		out, err = c.run(ctx, "resource", "list", "--json")
	}
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ResourceListResponse{})
	if err != nil {
		return nil, fmt.Errorf("resource list: %w", err)
	}
	return resp, nil
}

// ResourceStatuses returns the typed fleet form of `vrooli resource status
// --json` (every resource's runtime status).
func (c *Client) ResourceStatuses(ctx context.Context) (*cliv1.ResourceStatusesResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "resource", "status", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ResourceStatusesResponse{})
	if err != nil {
		return nil, fmt.Errorf("resource status: %w", err)
	}
	return resp, nil
}

// ResourceStatus returns the typed single form of `vrooli resource status <name>
// --json`.
func (c *Client) ResourceStatus(ctx context.Context, name string) (*cliv1.ResourceStatusResponse, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("resource status: name is required")
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "resource", "status", name, "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ResourceStatusResponse{})
	if err != nil {
		return nil, fmt.Errorf("resource status %s: %w", name, err)
	}
	return resp, nil
}
