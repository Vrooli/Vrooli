package audit

import (
	"path/filepath"
	"testing"

	internalaudit "quality-health/internal/audit"
	"quality-health/internal/contracts"
	"quality-health/internal/surfaces"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
)

func TestContractToProtoPreservesRuleIDs(t *testing.T) {
	contract, ok := contracts.ByRule(contracts.RuleTSConfigStrict)
	require.True(t, ok)

	got := contractToProto(contract)

	require.Equal(t, contract.ID, got.GetId())
	require.Contains(t, got.GetRuleIds(), contracts.RuleTSConfigStrict)
	require.True(t, got.GetAutofixAvailable())
}

func TestResponseToProtoIncludesValidSharedMaturityAssessment(t *testing.T) {
	spec := testMaturitySpec(t)
	resp, err := ResponseToProto(internalaudit.Response{
		RunID:   "qh-test",
		Status:  "failed",
		Summary: "quality failed",
		Inventory: surfaces.Inventory{
			Scenario:   "demo",
			TargetKind: "scenario",
			RootPath:   "/repo/scenarios/demo",
		},
		Maturity: internalaudit.Maturity{Rung: 2, Label: "L2"},
		Findings: []internalaudit.Finding{{
			RuleID:      contracts.RuleTSConfigStrict,
			Severity:    "error",
			Message:     "strict mode not enabled",
			FilePath:    "ui/tsconfig.json",
			Remediation: "Restore strict TypeScript settings.",
		}},
	}, spec, nil)
	require.NoError(t, err)
	require.NotNil(t, resp.GetAssessment())
	require.Equal(t, "quality-health", resp.GetAssessment().GetProvider())
	require.Equal(t, "quality", resp.GetAssessment().GetPhase())
	require.Equal(t, "L1", resp.GetAssessment().GetLocal().GetCurrentLevel())
	require.Equal(t, []string{contracts.RuleTSConfigStrict}, resp.GetAssessment().GetLocal().GetBlockingFindingCodes())
	require.NoError(t, assessment.ValidateAssessment(resp.GetAssessment()))
}

func TestResponseToProtoRequiresMaturitySpec(t *testing.T) {
	_, err := ResponseToProto(internalaudit.Response{
		Inventory: surfaces.Inventory{Scenario: "demo"},
	}, nil, nil)
	require.ErrorContains(t, err, "maturity spec is required")
}

func testMaturitySpec(t *testing.T) *assessment.Spec {
	t.Helper()
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	return spec
}
