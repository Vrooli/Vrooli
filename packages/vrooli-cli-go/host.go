package vroolicli

import (
	"context"
	"fmt"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// HostInventory returns the typed subset of `vrooli host inventory --json`
// (memory and swap totals). Byte counts arrive as JSON strings (the proto3
// 64-bit-integer convention) and decode into uint64 via protojson.
func (c *Client) HostInventory(ctx context.Context) (*cliv1.HostInventoryResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "host", "inventory", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.HostInventoryResponse{})
	if err != nil {
		return nil, fmt.Errorf("host inventory: %w", err)
	}
	return resp, nil
}
