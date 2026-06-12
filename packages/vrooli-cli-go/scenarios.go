package vroolicli

import (
	"context"
	"fmt"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ListScenariosOption customizes a ListScenarios call.
type ListScenariosOption func(args *[]string)

// WithPorts includes resolved port bindings in the scenario list (adds
// --include-ports). Off by default: resolving ports walks every scenario's
// runtime and is slower, so callers opt in only when they need port data.
func WithPorts() ListScenariosOption {
	return func(args *[]string) { *args = append(*args, "--include-ports") }
}

// ListScenarios returns the typed `vrooli scenario list --json` response.
func (c *Client) ListScenarios(ctx context.Context, opts ...ListScenariosOption) (*cliv1.ScenarioListResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	args := []string{"scenario", "list", "--json"}
	for _, opt := range opts {
		opt(&args)
	}

	out, err := c.run(ctx, args...)
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

// ScenarioPort returns the typed payload of `vrooli scenario port <name>
// <port_name> --json` — the resolved port for one scenario's lifecycle role.
// The returned message carries success/error and the int32 port (0 when
// unresolved), so callers branch on the typed fields rather than parsing stdout.
func (c *Client) ScenarioPort(ctx context.Context, name, portName string) (*cliv1.ScenarioPortSingle, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("scenario port: name is required")
	}
	if strings.TrimSpace(portName) == "" {
		return nil, fmt.Errorf("scenario port: port name is required")
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "scenario", "port", name, portName, "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ScenarioPortSingle{})
	if err != nil {
		return nil, fmt.Errorf("scenario port %s %s: %w", name, portName, err)
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
