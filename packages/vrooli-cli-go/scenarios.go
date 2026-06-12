package vroolicli

import (
	"context"
	"fmt"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ListScenarios returns the typed `vrooli scenario list --json` response.
func (c *Client) ListScenarios(ctx context.Context) (*cliv1.ScenarioListResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "scenario", "list", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ScenarioListResponse{})
	if err != nil {
		return nil, fmt.Errorf("scenario list: %w", err)
	}
	return resp, nil
}

// ScenarioStatuses returns the typed list form of `vrooli scenario status
// --json` (summary + every scenario's runtime status).
func (c *Client) ScenarioStatuses(ctx context.Context) (*cliv1.ScenarioStatusListResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "scenario", "status", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ScenarioStatusListResponse{})
	if err != nil {
		return nil, fmt.Errorf("scenario status: %w", err)
	}
	return resp, nil
}

// ScenarioStatus returns the typed single form of `vrooli scenario status <name>
// --json`.
func (c *Client) ScenarioStatus(ctx context.Context, name string) (*cliv1.ScenarioStatusSingle, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("scenario status: name is required")
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "scenario", "status", name, "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ScenarioStatusSingle{})
	if err != nil {
		return nil, fmt.Errorf("scenario status %s: %w", name, err)
	}
	return resp, nil
}
