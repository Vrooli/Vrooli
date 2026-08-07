package validationmatrix

import (
	"testing"

	"github.com/stretchr/testify/require"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestEvaluateReleaseGateBlocksMissingRequiredCoverage(t *testing.T) {
	gate := EvaluateReleaseGate(&domainv1.ValidationMatrix{MatrixId: "matrix-1"})
	require.False(t, gate.GetPassed())
	require.Equal(t, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, gate.GetDisposition())
}

func TestEvaluateReleaseGateRequiresRedactedChecksummedEvidence(t *testing.T) {
	cell := passingCell()
	cell.Evidence[0].Redacted = false
	gate := EvaluateReleaseGate(&domainv1.ValidationMatrix{MatrixId: "matrix-1", Cells: []*domainv1.ValidationCell{cell}})
	require.False(t, gate.GetPassed())
	require.Equal(t, []string{"cell-1"}, gate.GetFailedCellIds())
}

func TestEvaluateReleaseGatePassesCompleteRequiredApplicableCells(t *testing.T) {
	cell := passingCell()
	gate := EvaluateReleaseGate(&domainv1.ValidationMatrix{MatrixId: "matrix-1", Cells: []*domainv1.ValidationCell{cell}})
	require.True(t, gate.GetPassed())
	require.Equal(t, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, gate.GetDisposition())
	require.Equal(t, int32(1), gate.GetPassingCellCount())
}

func passingCell() *domainv1.ValidationCell {
	return &domainv1.ValidationCell{
		CellId: "cell-1", Required: true, Applicable: true,
		Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS,
		Evidence:    []*domainv1.LayeredEvidence{{EvidenceId: "evidence-1", Uri: "captures/evidence.json", Sha256: "sha256:one", Redacted: true}},
	}
}
