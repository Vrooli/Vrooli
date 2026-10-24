package vroolicli

import (
	"context"
	"fmt"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// Status returns the typed summary view of the unified `vrooli status --json`
// (fleet rollup counts for resources, scenarios, and maintenance).
func (c *Client) Status(ctx context.Context) (*cliv1.StatusResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "status", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.StatusResponse{})
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}
	return resp, nil
}
