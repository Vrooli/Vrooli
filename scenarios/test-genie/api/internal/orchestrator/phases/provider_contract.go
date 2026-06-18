package phases

import (
	"errors"
	"fmt"
	"strings"

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

func requireProtoProviderAssessment(provider string, phase string, assessment *commonv1.MaturityAssessment) error {
	if err := assessmentpkgValidate(assessment); err != nil {
		return providerMaturityContractError("%s %s assessment is invalid: %v", provider, phase, err)
	}
	if got := strings.TrimSpace(assessment.GetProvider()); got != provider {
		return providerMaturityContractError("%s %s assessment.provider=%q, want %q", provider, phase, got, provider)
	}
	if got := strings.TrimSpace(assessment.GetPhase()); got != phase {
		return providerMaturityContractError("%s %s assessment.phase=%q, want %q", provider, phase, got, phase)
	}
	return nil
}

func assessmentpkgValidate(a *commonv1.MaturityAssessment) error {
	return assessment.ValidateAssessment(a)
}
