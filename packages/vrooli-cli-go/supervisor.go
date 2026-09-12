package vroolicli

import (
	"context"
	"fmt"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// RuntimeSupervisorStatus returns the typed `vrooli runtime supervisor status
// --json` report: the local runtime supervisor's identity, heartbeat freshness,
// supervised/unverified instance counts, effective tuning (durations in
// nanoseconds), and its last reconcile tick. Note this output has no `success`
// envelope — it is the bare status object.
func (c *Client) RuntimeSupervisorStatus(ctx context.Context) (*cliv1.CliSupervisorStatus, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "runtime", "supervisor", "status", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.CliSupervisorStatus{})
	if err != nil {
		return nil, fmt.Errorf("runtime supervisor status: %w", err)
	}
	return resp, nil
}
