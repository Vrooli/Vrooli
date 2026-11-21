package conflicts

import (
	"context"
	"errors"
	"os"

	"architecture-cartographer/internal/domains"

	intent "intent-go"
)

// ClaimProvider is the seam between intent extractors and detectors. Production
// uses intent-go over the scenario root; tests pass a fake claim provider.
type ClaimProvider interface {
	Claims(ctx context.Context, scenario string) ([]intent.CapabilityClaim, error)
}

// FileClaimProvider adapts the shared intent-go PRD and requirements
// extractors to cartographer's scenario locator.
type FileClaimProvider struct {
	Locator               domains.ScenarioLocator
	PRDExtractor          intent.PRDExtractor
	RequirementsExtractor intent.RequirementsExtractor
}

func NewFileClaimProvider(locator domains.ScenarioLocator) *FileClaimProvider {
	return &FileClaimProvider{
		Locator:               locator,
		PRDExtractor:          intent.FilePRDExtractor{},
		RequirementsExtractor: intent.FileRequirementsExtractor{},
	}
}

func (p *FileClaimProvider) Claims(_ context.Context, scenario string) ([]intent.CapabilityClaim, error) {
	if p == nil || p.Locator == nil {
		return nil, nil
	}
	root, err := p.Locator.Locate(scenario)
	if err != nil {
		return nil, err
	}
	prdExtractor := p.PRDExtractor
	if prdExtractor == nil {
		prdExtractor = intent.FilePRDExtractor{}
	}
	requirementsExtractor := p.RequirementsExtractor
	if requirementsExtractor == nil {
		requirementsExtractor = intent.FileRequirementsExtractor{}
	}
	outcomes, err := prdExtractor.ExtractPRDClaims(root)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	requirements, err := requirementsExtractor.ExtractRequirementClaims(root)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	claims := make([]intent.CapabilityClaim, 0, len(outcomes)+len(requirements))
	claims = append(claims, outcomes...)
	claims = append(claims, requirements...)
	return claims, nil
}

// StaticClaimProvider is a test seam for detectors that consume intent claims.
type StaticClaimProvider struct {
	ClaimsOut []intent.CapabilityClaim
	Err       error
}

func (p StaticClaimProvider) Claims(context.Context, string) ([]intent.CapabilityClaim, error) {
	return p.ClaimsOut, p.Err
}
