package vroolicli

import (
	"context"
	"fmt"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// RecoveryShow returns the typed `vrooli recovery show --scenario <s> --slug
// <slug> --json` engagement view (the recovery-floor engagement a baseline verb
// reads back: mode/variant/anchor/ambient-var/TTL/expiry).
func (c *Client) RecoveryShow(ctx context.Context, scenario, slug string) (*cliv1.RecoveryEngagementView, error) {
	if strings.TrimSpace(scenario) == "" {
		return nil, fmt.Errorf("recovery show: scenario is required")
	}
	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("recovery show: slug is required")
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "recovery", "show", "--scenario", scenario, "--slug", slug, "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.RecoveryEngagementView{})
	if err != nil {
		return nil, fmt.Errorf("recovery show %s/%s: %w", scenario, slug, err)
	}
	return resp, nil
}

// RecoveryList returns the typed `vrooli recovery list --json` output — every
// open engagement across the floor.
func (c *Client) RecoveryList(ctx context.Context) (*cliv1.RecoveryListOutput, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	out, err := c.run(ctx, "recovery", "list", "--json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.RecoveryListOutput{})
	if err != nil {
		return nil, fmt.Errorf("recovery list: %w", err)
	}
	return resp, nil
}

// RecoveryNamespace returns the typed `vrooli recovery namespace --scenario <s>
// --variant <v> --json` output — the resolved SSOT storage namespaces (postgres
// DB, data dir, storage namespace) for a scenario instance.
func (c *Client) RecoveryNamespace(ctx context.Context, scenario, variant string) (*cliv1.RecoveryNamespaceOutput, error) {
	if strings.TrimSpace(scenario) == "" {
		return nil, fmt.Errorf("recovery namespace: scenario is required")
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	args := []string{"recovery", "namespace", "--scenario", scenario}
	if strings.TrimSpace(variant) != "" {
		args = append(args, "--variant", variant)
	}
	args = append(args, "--json")

	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.RecoveryNamespaceOutput{})
	if err != nil {
		return nil, fmt.Errorf("recovery namespace %s: %w", scenario, err)
	}
	return resp, nil
}
