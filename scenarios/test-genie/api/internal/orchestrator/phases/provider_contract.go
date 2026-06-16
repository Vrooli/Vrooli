package phases

import (
	"errors"
	"fmt"
	"strings"

	"test-genie/internal/shared"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

var errProviderMaturityContract = errors.New("provider maturity contract violation")

type providerMaturityAssessment struct {
	Scenario string `json:"scenario"`
	Provider string `json:"provider"`
	Phase    string `json:"phase"`
	Version  string `json:"version"`
	Local    struct {
		CurrentLevel string   `json:"current_level"`
		NextLevel    string   `json:"next_level"`
		Blocking     []string `json:"blocking_finding_codes"`
	} `json:"local"`
}

func providerMaturityContractError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errProviderMaturityContract, fmt.Sprintf(format, args...))
}

func classifyProviderParseFailure(err error) shared.FailureClass {
	if errors.Is(err, errProviderMaturityContract) {
		return shared.FailureClassMaturityContract
	}
	return shared.FailureClassSystem
}

func requireProviderAssessment(provider string, phase string, assessment *providerMaturityAssessment) error {
	if assessment == nil {
		return providerMaturityContractError("%s %s output is missing required assessment", provider, phase)
	}
	if strings.TrimSpace(assessment.Scenario) == "" {
		return providerMaturityContractError("%s assessment.scenario is required", provider)
	}
	if strings.TrimSpace(assessment.Provider) == "" {
		return providerMaturityContractError("%s assessment.provider is required", provider)
	}
	if strings.TrimSpace(assessment.Phase) == "" {
		return providerMaturityContractError("%s assessment.phase is required", provider)
	}
	if strings.TrimSpace(assessment.Version) == "" {
		return providerMaturityContractError("%s assessment.version is required", provider)
	}
	if strings.TrimSpace(assessment.Local.CurrentLevel) == "" {
		return providerMaturityContractError("%s assessment.local.current_level is required", provider)
	}
	return nil
}

func localMaturitySummary(assessment *providerMaturityAssessment) (current string, next string) {
	if assessment == nil {
		return "", ""
	}
	return assessment.Local.CurrentLevel, assessment.Local.NextLevel
}

func requireProtoProviderAssessment(provider string, phase string, assessment *commonv1.MaturityAssessment) error {
	if assessment == nil {
		return providerMaturityContractError("%s %s output is missing required assessment", provider, phase)
	}
	if strings.TrimSpace(assessment.GetScenario()) == "" {
		return providerMaturityContractError("%s assessment.scenario is required", provider)
	}
	if strings.TrimSpace(assessment.GetProvider()) == "" {
		return providerMaturityContractError("%s assessment.provider is required", provider)
	}
	if strings.TrimSpace(assessment.GetPhase()) == "" {
		return providerMaturityContractError("%s assessment.phase is required", provider)
	}
	if strings.TrimSpace(assessment.GetVersion()) == "" {
		return providerMaturityContractError("%s assessment.version is required", provider)
	}
	if strings.TrimSpace(assessment.GetLocal().GetCurrentLevel()) == "" {
		return providerMaturityContractError("%s assessment.local.current_level is required", provider)
	}
	return nil
}

func protoLocalMaturitySummary(assessment *commonv1.MaturityAssessment) (current string, next string) {
	if assessment == nil {
		return "", ""
	}
	return assessment.GetLocal().GetCurrentLevel(), assessment.GetLocal().GetNextLevel()
}
