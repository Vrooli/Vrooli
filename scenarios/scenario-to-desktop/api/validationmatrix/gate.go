package validationmatrix

import (
	"fmt"
	"strings"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

// EvaluateReleaseGate applies the fail-closed matrix rule. Non-applicable
// cells are not required coverage; every required applicable cell must be a
// pass with at least one linked, checksummed, redacted evidence item.
func EvaluateReleaseGate(matrix *domainv1.ValidationMatrix) *domainv1.ReleaseGate {
	gate := &domainv1.ReleaseGate{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED}
	if matrix != nil {
		gate.MatrixId = matrix.GetMatrixId()
	}
	if matrix == nil || strings.TrimSpace(matrix.GetMatrixId()) == "" {
		gate.Reason = stringPtr("validation matrix identity is missing")
		return gate
	}
	for _, cell := range matrix.GetCells() {
		if cell == nil || !cell.GetRequired() || !cell.GetApplicable() {
			continue
		}
		gate.RequiredCellCount++
		if cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS {
			gate.FailedCellIds = append(gate.FailedCellIds, cell.GetCellId())
			continue
		}
		if !completeEvidence(cell) {
			gate.FailedCellIds = append(gate.FailedCellIds, cell.GetCellId())
			continue
		}
		gate.PassingCellCount++
	}
	if gate.RequiredCellCount == 0 {
		gate.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN
		gate.Reason = stringPtr("no required applicable validation cells exist")
		return gate
	}
	seen := make(map[string]struct{}, len(matrix.GetCells()))
	for _, cell := range matrix.GetCells() {
		if cell == nil || !cell.GetRequired() || !cell.GetApplicable() {
			continue
		}
		seen[cell.GetCellId()] = struct{}{}
	}
	// The matrix contract requires a cell for every selected journey/target/
	// profile combination. The explicit cell list is authoritative; an empty
	// or duplicate identity is incomplete coverage, never a pass.
	if int32(len(seen)) != gate.RequiredCellCount {
		gate.Reason = stringPtr("required validation cell identities are incomplete")
		return gate
	}
	if len(gate.FailedCellIds) > 0 || gate.PassingCellCount != gate.RequiredCellCount {
		gate.Reason = stringPtr(fmt.Sprintf("%d required validation cell(s) failed or lack complete evidence", len(gate.FailedCellIds)))
		return gate
	}
	gate.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS
	gate.Passed = true
	return gate
}

func completeEvidence(cell *domainv1.ValidationCell) bool {
	if len(cell.GetEvidence()) == 0 {
		return false
	}
	for _, evidence := range cell.GetEvidence() {
		if evidence == nil || strings.TrimSpace(evidence.GetEvidenceId()) == "" || strings.TrimSpace(evidence.GetUri()) == "" || strings.TrimSpace(evidence.GetSha256()) == "" || !evidence.GetRedacted() {
			return false
		}
	}
	return true
}

func stringPtr(value string) *string { return &value }
