package workflows

import (
	"context"

	"github.com/vrooli/browser-automation-studio/services/testgenie"
)

// NewDefaultSeedRunner adapts a testgenie.Client to the SeedRunner seam.
func NewDefaultSeedRunner(client *testgenie.Client) SeedRunner {
	return &defaultSeedRunner{client: client}
}

type defaultSeedRunner struct {
	client *testgenie.Client
}

func (d *defaultSeedRunner) ApplySeed(ctx context.Context, scenario string, retain bool) (string, map[string]any, error) {
	if d == nil || d.client == nil {
		return "", nil, errSeedCleanupUnavail
	}
	resp, err := d.client.ApplySeed(ctx, scenario, retain)
	if err != nil {
		return "", nil, err
	}
	return resp.CleanupToken, resp.SeedState, nil
}

func (d *defaultSeedRunner) CleanupSeed(ctx context.Context, scenario, cleanupToken string) error {
	if d == nil || d.client == nil {
		return errSeedCleanupUnavail
	}
	_, err := d.client.CleanupSeed(ctx, scenario, cleanupToken)
	return err
}
