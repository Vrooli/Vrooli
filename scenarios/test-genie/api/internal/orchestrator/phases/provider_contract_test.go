package phases

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"test-genie/internal/shared"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func testProviderAssessment(scenario, provider, phase, current, next string) *commonv1.MaturityAssessment {
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

func TestRequireProviderAssessmentJSONValidatesSharedContract(t *testing.T) {
	t.Run("accepts valid assessment", func(t *testing.T) {
		raw := json.RawMessage(`{
			"scenario":"demo",
			"provider":"cli-health",
			"phase":"contracts",
			"version":"1.0.0",
			"local":{"current_level":"L4","next_level":"L5"}
		}`)
		got, err := requireProviderAssessmentJSON("cli-health", "contracts", raw)
		if err != nil {
			t.Fatalf("requireProviderAssessmentJSON returned error: %v", err)
		}
		if got.GetProvider() != "cli-health" || got.GetPhase() != "contracts" {
			t.Fatalf("assessment identity = %s/%s", got.GetProvider(), got.GetPhase())
		}
	})

	t.Run("rejects wrong provider", func(t *testing.T) {
		raw := json.RawMessage(`{
			"scenario":"demo",
			"provider":"old-health",
			"phase":"contracts",
			"version":"1.0.0",
			"local":{"current_level":"L4"}
		}`)
		_, err := requireProviderAssessmentJSON("cli-health", "contracts", raw)
		if err == nil || !strings.Contains(err.Error(), `assessment.provider="old-health"`) {
			t.Fatalf("expected provider identity error, got %v", err)
		}
	})

	t.Run("rejects wrong phase", func(t *testing.T) {
		raw := json.RawMessage(`{
			"scenario":"demo",
			"provider":"cli-health",
			"phase":"quality",
			"version":"1.0.0",
			"local":{"current_level":"L4"}
		}`)
		_, err := requireProviderAssessmentJSON("cli-health", "contracts", raw)
		if err == nil || !strings.Contains(err.Error(), `assessment.phase="quality"`) {
			t.Fatalf("expected phase identity error, got %v", err)
		}
	})

	t.Run("rejects findings without maturity metadata", func(t *testing.T) {
		raw := json.RawMessage(`{
			"scenario":"demo",
			"provider":"cli-health",
			"phase":"contracts",
			"version":"1.0.0",
			"local":{"current_level":"L4"},
			"findings":[{"code":"cli.manifest.missing","severity":"SEVERITY_ERROR"}]
		}`)
		_, err := requireProviderAssessmentJSON("cli-health", "contracts", raw)
		if err == nil || !strings.Contains(err.Error(), "assessment.findings[0].maturity is required") {
			t.Fatalf("expected missing finding maturity error, got %v", err)
		}
	})
}

func TestRequireProtoProviderAssessmentChecksIdentityAndFindingMaturity(t *testing.T) {
	assessment := testProtoProviderAssessment("demo", "cli-health", "contracts", "L4", "L5")
	assessment.Findings = []*commonv1.AssessmentFinding{{Code: "cli.manifest.missing", Severity: "SEVERITY_ERROR"}}
	err := requireProtoProviderAssessment("cli-health", "contracts", assessment)
	if err == nil || !strings.Contains(err.Error(), "assessment.findings[0].maturity is required") {
		t.Fatalf("expected missing finding maturity error, got %v", err)
	}
}
