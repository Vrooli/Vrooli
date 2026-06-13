package vroolicli

import (
	"context"
	"fmt"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// Locks returns the typed `vrooli locks --json` response: the runtime
// registry's port claims (each with reconciliation state and a port-ownership
// recommendation). Health and recovery tooling uses this to reason about who
// owns which port.
func (c *Client) Locks(ctx context.Context) (*cliv1.ProjectLocksResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "locks", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ProjectLocksResponse{})
	if err != nil {
		return nil, fmt.Errorf("locks: %w", err)
	}
	return resp, nil
}
