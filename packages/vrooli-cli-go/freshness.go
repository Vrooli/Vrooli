package vroolicli

import (
	"context"
	"fmt"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ScenarioFreshnessOption customizes a ScenarioFreshness call.
type ScenarioFreshnessOption func(args *[]string)

// WithFreshnessPath resolves the scenario from a custom path rather than the
// default scenarios/ tree (adds --path). Use it when the scenario lives outside
// the conventional location.
func WithFreshnessPath(path string) ScenarioFreshnessOption {
	return func(args *[]string) {
		if strings.TrimSpace(path) != "" {
			*args = append(*args, "--path", path)
		}
	}
}

// ScenarioFreshness returns the typed `vrooli scenario freshness <name> --json`
// report — the canonical content-hash freshness verdict for a scenario's build
// artifacts (binaries + ui-bundle) plus the resolved freshness_policy of each
// declared scenario dependency. Callers branch on the typed Stale fields rather
// than parsing stdout. This is the single authority for "is the build fresh?";
// it folds in file: workspace deps, keyed build inputs (NODE_ENV, toolchain),
// and per-file content hashing that an mtime heuristic misses.
func (c *Client) ScenarioFreshness(ctx context.Context, name string, opts ...ScenarioFreshnessOption) (*cliv1.ScenarioFreshnessResponse, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("scenario freshness: name is required")
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	args := []string{"scenario", "freshness", name, "--json"}
	for _, opt := range opts {
		opt(&args)
	}

	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.ScenarioFreshnessResponse{})
	if err != nil {
		return nil, fmt.Errorf("scenario freshness %s: %w", name, err)
	}
	return resp, nil
}
