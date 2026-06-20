package phases

import (
	"errors"
	"fmt"

	"test-genie/internal/shared"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

var errProviderMaturityContract = errors.New("provider maturity contract violation")

func providerMaturityContractError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errProviderMaturityContract, fmt.Sprintf(format, args...))
}

func classifyProviderParseFailure(err error) shared.FailureClass {
	if errors.Is(err, errProviderMaturityContract) {
		return shared.FailureClassMaturityContract
	}
	return shared.FailureClassSystem
}

func localMaturitySummary(assessment *commonv1.MaturityAssessment) (current string, next string) {
	if assessment == nil {
		return "", ""
	}
	return assessment.GetLocal().GetCurrentLevel(), assessment.GetLocal().GetNextLevel()
}

func requireProtoProviderAssessment(provider string, phase string, a *commonv1.MaturityAssessment) error {
	if err := assessment.RequireIdentity(provider, phase, a); err != nil {
		return providerMaturityContractError("%s %s %v", provider, phase, err)
	}
	return nil
}
