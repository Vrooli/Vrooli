package usagemodes

import (
	"context"
	"fmt"

	"agent-manager/internal/domain"

	"github.com/vrooli/cli-core/usagemode"
)

// PolicyFiles reads a runner's resource-owned model-policy file.
type PolicyFiles interface {
	PolicyFile(ctx context.Context, runner domain.RunnerType) ([]byte, error)
}

// ResourceTiers loads usage tiers from the resources' model-policy files.
type ResourceTiers struct{ Files PolicyFiles }

// Tiers parses the runner's policy file with the shared usage mode loader.
func (r ResourceTiers) Tiers(ctx context.Context, runner string) (usagemode.Tiers, error) {
	if r.Files == nil {
		return usagemode.Tiers{}, fmt.Errorf("resource policy files are not configured")
	}
	raw, err := r.Files.PolicyFile(ctx, domain.RunnerType(runner))
	if err != nil {
		return usagemode.Tiers{}, err
	}
	return usagemode.LoadTiers(raw)
}
