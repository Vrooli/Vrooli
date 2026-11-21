package certification

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// CollectionIssue explains why one owner result did not become durable
// evidence. It is not converted into a verdict; the cell remains pending.
type CollectionIssue struct {
	Cell  CellID `json:"cell"`
	Error string `json:"error"`
}

// RecordedDriverResult is one append-only receipt created from an owner result.
type RecordedDriverResult struct {
	Cell    CellID  `json:"cell"`
	Receipt string  `json:"receipt"`
	Verdict Verdict `json:"verdict"`
}

// CollectionReport accounts for all required target-ground cells after one
// batch of owner results. Missing, duplicate, invalid, and failed writes remain
// pending; only validated append-only receipts count as recorded.
type CollectionReport struct {
	MatrixRevision string                 `json:"matrix_revision"`
	Required       int                    `json:"required"`
	Recorded       []RecordedDriverResult `json:"recorded"`
	Pending        []CellID               `json:"pending"`
	Issues         []CollectionIssue      `json:"issues"`
}

// CollectDriverResults converts cell-specific non-passing owner outcomes into
// durable receipts. Passed results stay pending until the certification
// package can authenticate both the target-owner receipt and assertion bytes;
// matching strings, digests, and Test Genie status alone are insufficient.
// Missing or invalid outcomes are never inferred.
func CollectDriverResults(evidenceDir string, results []DriverResult) (CollectionReport, error) {
	m, err := LoadEmbedded()
	if err != nil {
		return CollectionReport{}, err
	}
	plan, err := AssertionDrivers(m)
	if err != nil {
		return CollectionReport{}, err
	}
	byCell := indexDriverResults(results)
	report := CollectionReport{MatrixRevision: m.MatrixRevision, Required: len(plan)}
	recorded := make(map[CellID]bool, len(plan))
	for _, driver := range plan {
		collectPlannedDriverResult(evidenceDir, m, driver.Cell, byCell[driver.Cell], recorded, &report)
	}
	completeCollectionReport(m, byCell, recorded, &report)
	return report, nil
}

func indexDriverResults(results []DriverResult) map[CellID][]DriverResult {
	byCell := make(map[CellID][]DriverResult, len(results))
	for _, result := range results {
		byCell[result.Cell] = append(byCell[result.Cell], result)
	}
	return byCell
}

func collectPlannedDriverResult(evidenceDir string, matrix *Matrix, cell CellID, matches []DriverResult, recorded map[CellID]bool, report *CollectionReport) {
	if len(matches) == 0 {
		return
	}
	if len(matches) != 1 {
		report.Issues = append(report.Issues, CollectionIssue{Cell: cell, Error: fmt.Sprintf("received %d owner results for one cell; exactly one is required", len(matches))})
		return
	}
	result := matches[0]
	if result.Verdict == VerdictPassed && !result.ownerEvidenceVerified {
		report.Issues = append(report.Issues, CollectionIssue{Cell: cell, Error: "passed result requires authenticated target-owner receipt and assertion-artifact bytes; structural collection cannot verify owner evidence"})
		return
	}
	receipt, err := ReceiptFromDriverResult(matrix, result)
	if err != nil {
		report.Issues = append(report.Issues, CollectionIssue{Cell: cell, Error: err.Error()})
		return
	}
	path, err := WriteReceipt(evidenceDir, receipt)
	if err != nil {
		report.Issues = append(report.Issues, CollectionIssue{Cell: cell, Error: err.Error()})
		return
	}
	recorded[cell] = true
	report.Recorded = append(report.Recorded, RecordedDriverResult{Cell: cell, Receipt: path, Verdict: receipt.Verdict})
}

func completeCollectionReport(matrix *Matrix, byCell map[CellID][]DriverResult, recorded map[CellID]bool, report *CollectionReport) {
	for _, cell := range matrix.RequiredCells() {
		if !recorded[cell] {
			report.Pending = append(report.Pending, cell)
		}
	}
	appendUnexpectedCellIssues(matrix, byCell, report)
	sortCollectionReport(report)
}

func appendUnexpectedCellIssues(matrix *Matrix, byCell map[CellID][]DriverResult, report *CollectionReport) {
	for resultCell, matches := range byCell {
		if _, err := driverRefForMatrix(matrix, resultCell); err == nil {
			continue
		}
		report.Issues = append(report.Issues, CollectionIssue{Cell: resultCell, Error: fmt.Sprintf("received %d result(s) for a cell outside the required matrix", len(matches))})
	}
}

func sortCollectionReport(report *CollectionReport) {
	sort.Slice(report.Recorded, func(i, j int) bool { return report.Recorded[i].Cell.String() < report.Recorded[j].Cell.String() })
	sort.Slice(report.Pending, func(i, j int) bool { return report.Pending[i].String() < report.Pending[j].String() })
	sortCollectionIssues(report.Issues)
}

// CollectFullyAuthenticatedDriverResults verifies Test Genie-owned assertion
// bytes and the fresh target-owner operation receipt before granting the
// in-process pass capability consumed by the append-only writer.
type AssertionOwnerReaders struct {
	Bridge BridgeAssertionOwner
	BAS    BASAssertionOwner
}

func CollectFullyAuthenticatedDriverResults(ctx context.Context, evidenceDir string, results []DriverResult, receipts ValidationReceiptReader, artifacts TestGenieArtifactOwner, targets TargetReceiptOwner, ownerReaders ...AssertionOwnerReaders) (CollectionReport, error) {
	verified := make([]DriverResult, 0, len(results))
	var issues []CollectionIssue
	counts := countDriverResults(results)
	duplicateReported := make(map[CellID]bool)
	for _, result := range results {
		verifiedResult, include, issue := verifyAuthenticatedDriverResult(ctx, result, counts, duplicateReported, receipts, artifacts, targets, ownerReaders)
		if issue != nil {
			issues = append(issues, *issue)
		}
		if include {
			verified = append(verified, verifiedResult)
		}
	}
	report, err := CollectDriverResults(evidenceDir, verified)
	if err != nil {
		return CollectionReport{}, err
	}
	report.Issues = append(report.Issues, issues...)
	sortCollectionIssues(report.Issues)
	return report, nil
}

func countDriverResults(results []DriverResult) map[CellID]int {
	counts := make(map[CellID]int, len(results))
	for _, result := range results {
		counts[result.Cell]++
	}
	return counts
}

func verifyAuthenticatedDriverResult(ctx context.Context, result DriverResult, counts map[CellID]int, duplicateReported map[CellID]bool, receipts ValidationReceiptReader, artifacts TestGenieArtifactOwner, targets TargetReceiptOwner, ownerReaders []AssertionOwnerReaders) (DriverResult, bool, *CollectionIssue) {
	if _, err := DriverRef(result.Cell); err != nil {
		return result, true, nil
	}
	if counts[result.Cell] != 1 {
		if duplicateReported[result.Cell] {
			return DriverResult{}, false, nil
		}
		duplicateReported[result.Cell] = true
		return DriverResult{}, false, &CollectionIssue{Cell: result.Cell, Error: fmt.Sprintf("received %d owner results for one cell; exactly one is required", counts[result.Cell])}
	}
	if result.Verdict != VerdictPassed {
		return verifyNonPassingDriverResult(ctx, result, receipts)
	}
	return verifyPassedDriverResult(ctx, result, receipts, artifacts, targets, ownerReaders)
}

func verifyNonPassingDriverResult(ctx context.Context, result DriverResult, receipts ValidationReceiptReader) (DriverResult, bool, *CollectionIssue) {
	if err := VerifyTestGenieReceipt(ctx, receipts, result.ValidationReceiptRef, result.Verdict); err != nil {
		return DriverResult{}, false, &CollectionIssue{Cell: result.Cell, Error: err.Error()}
	}
	return result, true, nil
}

func verifyPassedDriverResult(ctx context.Context, result DriverResult, receipts ValidationReceiptReader, artifacts TestGenieArtifactOwner, targets TargetReceiptOwner, ownerReaders []AssertionOwnerReaders) (DriverResult, bool, *CollectionIssue) {
	if !laneOwnerProofSupported(result.Cell.Lane) {
		return failedAuthenticatedResult(result, fmt.Sprintf("lane %s owner receipt adapter is not implemented; pass remains unavailable", result.Cell.Lane))
	}
	if issue := requireLaneOwnerReader(result, ownerReaders); issue != nil {
		return DriverResult{}, false, issue
	}
	assertionProof, err := VerifyTestGenieAssertionArtifact(ctx, receipts, artifacts, result)
	if err != nil {
		return failedAuthenticatedResult(result, err.Error())
	}
	result, err = verifyLaneAssertionOwner(ctx, result, assertionProof, ownerReaders)
	if err != nil {
		return failedAuthenticatedResult(result, err.Error())
	}
	targetProof, err := verifyDriverTargetReceipt(ctx, result, targets)
	if err != nil {
		return failedAuthenticatedResult(result, err.Error())
	}
	result.DriverRef, _ = DriverRef(result.Cell)
	result.TargetReceiptProof = &targetProof
	result.TestGenieProof = &assertionProof
	result.ownerEvidenceVerified = true
	return result, true, nil
}

func requireLaneOwnerReader(result DriverResult, readers []AssertionOwnerReaders) *CollectionIssue {
	if result.Cell.Lane == LaneRelay && (len(readers) != 1 || readers[0].Bridge == nil) {
		return &CollectionIssue{Cell: result.Cell, Error: "relay lane requires exactly one Bridge assertion owner reader"}
	}
	if result.Cell.Lane == LaneBASUI && (len(readers) != 1 || readers[0].BAS == nil) {
		return &CollectionIssue{Cell: result.Cell, Error: "BAS UI lane requires exactly one BAS assertion owner reader"}
	}
	return nil
}

func verifyLaneAssertionOwner(ctx context.Context, result DriverResult, assertion TestGenieAssertionProof, readers []AssertionOwnerReaders) (DriverResult, error) {
	switch result.Cell.Lane {
	case LaneRelay:
		proof, err := VerifyBridgeAssertionOwnerReceipt(ctx, readers[0].Bridge, result, assertion)
		if err != nil {
			return result, err
		}
		result.BridgeOwnerProof = &proof
	case LaneBASUI:
		proof, err := VerifyBASAssertionOwnerReceipt(ctx, readers[0].BAS, result, assertion)
		if err != nil {
			return result, err
		}
		result.BASOwnerProof = &proof
	}
	return result, nil
}

func verifyDriverTargetReceipt(ctx context.Context, result DriverResult, targets TargetReceiptOwner) (TargetReceiptProof, error) {
	cellDriver, err := DriverRef(result.Cell)
	if err != nil {
		return TargetReceiptProof{}, err
	}
	return VerifyTargetOwnerReceipt(ctx, targets, TargetReceiptExpectation{
		Cell: result.Cell, DriverRef: cellDriver, DeploymentID: result.TargetDeploymentID,
		OperationID: strings.TrimPrefix(firstOperationRef(result.OperationRefs), "operation:"),
		PlanDigest:  result.TargetPlanDigest, Step: result.TargetReceiptStep, Target: result.Target,
	})
}

func failedAuthenticatedResult(result DriverResult, message string) (DriverResult, bool, *CollectionIssue) {
	return DriverResult{}, false, &CollectionIssue{Cell: result.Cell, Error: message}
}

func sortCollectionIssues(issues []CollectionIssue) {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Cell != issues[j].Cell {
			return issues[i].Cell.String() < issues[j].Cell.String()
		}
		return issues[i].Error < issues[j].Error
	})
}

func firstOperationRef(refs []string) string {
	if len(refs) != 1 {
		return ""
	}
	return refs[0]
}
