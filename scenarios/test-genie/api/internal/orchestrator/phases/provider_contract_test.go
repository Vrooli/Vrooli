package phases

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"test-genie/internal/shared"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func testProviderAssessment(scenario, provider, phase, current, next string) *providerMaturityAssessment {
	assessment := &providerMaturityAssessment{
		Scenario: scenario,
		Provider: provider,
		Phase:    phase,
		Version:  "1.0.0",
	}
	assessment.Local.CurrentLevel = current
	assessment.Local.NextLevel = next
	return assessment
}

func testProviderAssessmentJSON(scenario, provider, phase, current, next string) string {
	return fmt.Sprintf(`"assessment":{"scenario":%q,"provider":%q,"phase":%q,"version":"1.0.0","local":{"current_level":%q,"next_level":%q}}`,
		scenario, provider, phase, current, next)
}

func testProtoProviderAssessmentJSON(scenario, provider, phase, current, next string) string {
	return fmt.Sprintf(`"assessment":{"scenario":%q,"provider":%q,"phase":%q,"version":"1.0.0","local":{"currentLevel":%q,"nextLevel":%q}}`,
		scenario, provider, phase, current, next)
}

func testProtoProviderAssessment(scenario, provider, phase, current, next string) *commonv1.MaturityAssessment {
	return &commonv1.MaturityAssessment{
		Scenario: scenario,
		Provider: provider,
		Phase:    phase,
		Version:  "1.0.0",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel: current,
			NextLevel:    next,
		},
	}
}

func TestClassifyProviderParseFailure(t *testing.T) {
	t.Run("maturity contract", func(t *testing.T) {
		err := providerMaturityContractError("assessment.local.current_level is required")
		if got := classifyProviderParseFailure(err); got != shared.FailureClassMaturityContract {
			t.Fatalf("class = %q, want %q", got, shared.FailureClassMaturityContract)
		}
	})

	t.Run("malformed payload", func(t *testing.T) {
		err := &json.SyntaxError{}
		if got := classifyProviderParseFailure(err); got != shared.FailureClassSystem {
			t.Fatalf("class = %q, want %q", got, shared.FailureClassSystem)
		}
	})

	t.Run("provider unavailable", func(t *testing.T) {
		err := errors.New("locate cli-health CLI: not found")
		if got := classifyProviderParseFailure(err); got != shared.FailureClassSystem {
			t.Fatalf("parse classifier should not classify availability errors, got %q", got)
		}
	})
}
