package deployments

import (
	"context"

	"deployment-manager/codesigning"
)

// signingValidatorAdapter adapts codesigning components to the SigningValidator interface.
type signingValidatorAdapter struct {
	repo      codesigning.Repository
	validator codesigning.Validator
	checker   codesigning.PrerequisiteChecker
}

// NewSigningValidatorAdapter creates an adapter that implements SigningValidator.
func NewSigningValidatorAdapter(
	repo codesigning.Repository,
	validator codesigning.Validator,
	checker codesigning.PrerequisiteChecker,
) SigningValidator {
	return &signingValidatorAdapter{
		repo:      repo,
		validator: validator,
		checker:   checker,
	}
}

// GetSigningConfig retrieves the signing config for a profile.
func (a *signingValidatorAdapter) GetSigningConfig(ctx context.Context, profileID string) (*codesigning.SigningConfig, error) {
	return a.repo.Get(ctx, profileID)
}

// ValidateConfig checks structural validity of a SigningConfig.
func (a *signingValidatorAdapter) ValidateConfig(config *codesigning.SigningConfig) *codesigning.ValidationResult {
	return a.validator.ValidateConfig(config)
}

// CheckPrerequisites validates tools and certificates are available.
func (a *signingValidatorAdapter) CheckPrerequisites(ctx context.Context, config *codesigning.SigningConfig) *codesigning.ValidationResult {
	return a.checker.CheckPrerequisites(ctx, config)
}
