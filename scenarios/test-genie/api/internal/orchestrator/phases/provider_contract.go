package phases

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"test-genie/internal/shared"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
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

func requireProviderAssessmentJSON(provider string, phase string, raw json.RawMessage) (*commonv1.MaturityAssessment, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, providerMaturityContractError("%s %s output is missing required assessment", provider, phase)
	}
	var msg commonv1.MaturityAssessment
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &msg); err != nil {
		return nil, providerMaturityContractError("%s %s assessment is malformed: %v", provider, phase, err)
	}
	if err := requireProtoProviderAssessment(provider, phase, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
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
