package certification

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// AssertionDriver is the frozen owner contract for one target-ground cell.
// DriverRef is deliberately cell-specific: evidence produced for one case,
// lane, or ground cannot be replayed as another cell's assertion.
type AssertionDriver struct {
	Cell               CellID `json:"cell"`
	Ref                string `json:"driver_ref"`
	Trigger            string `json:"trigger"`
	Expected           string `json:"expected_assertion"`
	Owner              string `json:"owner"`
	OwnerReceiptScheme string `json:"owner_receipt_scheme"`
	OwnerAdapterStatus string `json:"owner_adapter_status"`
	EvidenceSource     string `json:"evidence_source"`
	Status             string `json:"status"`
}

// DriverResult is returned by an assertion owner after it has exercised the
// named cell. A missing or unavailable result stays missing/unavailable; this
// constructor never infers a pass from deployment or health observations.
type DriverResult struct {
	Cell                  CellID
	DriverRef             string
	Verdict               Verdict
	Candidate             Candidate
	Target                Target
	TargetDeploymentID    string
	TargetPlanDigest      string
	TargetReceiptStep     string
	ValidationReceiptRef  string
	OperationRefs         []string
	TargetReceiptRefs     []string
	AssertionArtifacts    []string
	Assertion             string
	Limitations           []string
	ObservedAt            time.Time
	TargetReceiptProof    *TargetReceiptProof
	TestGenieProof        *TestGenieAssertionProof
	BridgeOwnerProof      *BridgeAssertionOwnerProof
	BASOwnerProof         *BASAssertionOwnerProof
	ownerEvidenceVerified bool
}

// DriverRef returns the canonical assertion-driver identity for a required
// matrix cell. It rejects non-required or inapplicable cells.
func DriverRef(cell CellID) (string, error) {
	m, err := LoadEmbedded()
	if err != nil {
		return "", err
	}
	return driverRefForMatrix(m, cell)
}

func driverRefForMatrix(m *Matrix, cell CellID) (string, error) {
	caseRow, ok := m.Case(cell.CaseID)
	if !ok || !caseRow.Required || !containsLane(caseRow.Lanes, cell.Lane) {
		return "", fmt.Errorf("cell %s is not a required matrix cell", cell)
	}
	if !containsGround(m.TargetGroundsFor(cell.CaseID, cell.Lane), cell.TargetGround) {
		return "", fmt.Errorf("target ground %s is not applicable to %s", cell.TargetGround, cell)
	}
	return strings.Join([]string{"certification-assertion", "v1", cell.CaseID, string(cell.Lane), string(cell.TargetGround)}, ":"), nil
}

// AssertionDrivers enumerates the explicit assertion obligation for each
// required target-ground cell. The status accurately distinguishes the
// existing QEMU/VPS journey readers from the absent per-cell assertion owner.
func AssertionDrivers(m *Matrix) ([]AssertionDriver, error) {
	if m == nil {
		return nil, fmt.Errorf("matrix is required")
	}
	out := make([]AssertionDriver, 0, len(m.RequiredCells()))
	for _, cell := range m.RequiredCells() {
		caseRow, ok := m.Case(cell.CaseID)
		if !ok {
			return nil, fmt.Errorf("required cell %s has no case row", cell)
		}
		ref, err := driverRefForMatrix(m, cell)
		if err != nil {
			return nil, err
		}
		owner, source := laneEvidenceOwner(cell.Lane)
		adapterStatus := "missing"
		if laneOwnerProofSupported(cell.Lane) {
			adapterStatus = "implemented"
		}
		out = append(out, AssertionDriver{
			Cell: cell, Ref: ref, Trigger: caseRow.Trigger, Expected: caseRow.Assertion,
			Owner: owner, OwnerReceiptScheme: laneOwnerReceiptScheme(cell.Lane), OwnerAdapterStatus: adapterStatus,
			EvidenceSource: source, Status: "assertion_owner_required",
		})
	}
	return out, nil
}

func laneEvidenceOwner(lane Lane) (string, string) {
	switch lane {
	case LanePackage, LaneAPI, LaneWorkflow:
		return "scenario-to-cloud", "Test Genie owner receipt plus case-specific assertion artifact"
	case LaneRelay:
		return "Vrooli Bridge", "Bridge target owner receipt plus case-specific assertion artifact"
	case LaneQEMU:
		return "scenario-to-cloud QEMU lane", "QEMU target owner receipt plus case-specific assertion artifact"
	case LaneRealVPS:
		return "scenario-to-cloud VPS lane", "Bridge target owner receipt plus case-specific assertion artifact"
	case LaneBASUI:
		return "Browser Automation Studio", "BAS owner receipt plus case-specific assertion artifact"
	case LaneIndependentReview:
		return "independent reviewer", "signed reviewer receipt plus case-specific artifact"
	case LaneSoak:
		return "scenario-to-cloud soak owner", "soak owner receipt plus case-specific metrics artifact"
	default:
		return "unassigned", "owner evidence required"
	}
}

func laneOwnerReceiptScheme(lane Lane) string {
	switch lane {
	case LanePackage, LaneAPI, LaneWorkflow:
		return "test-genie"
	case LaneRelay:
		return "bridge-run"
	case LaneQEMU, LaneRealVPS:
		return "target-receipt"
	case LaneBASUI:
		return "bas-execution"
	case LaneIndependentReview:
		return "reviewer-attestation"
	case LaneSoak:
		return "soak-run"
	default:
		return ""
	}
}

func laneOwnerProofSupported(lane Lane) bool {
	switch lane {
	case LanePackage, LaneAPI, LaneWorkflow, LaneRelay, LaneQEMU, LaneRealVPS, LaneBASUI:
		return true
	default:
		return false
	}
}

func validateOwnerReceiptScheme(lane Lane, ref string) error {
	want := laneOwnerReceiptScheme(lane)
	u, err := url.Parse(strings.TrimSpace(ref))
	if want == "" || err != nil || u.Scheme != want || strings.TrimSpace(u.Opaque+u.Host+u.Path) == "" {
		return fmt.Errorf("lane %s owner receipt reference must use %s scheme", lane, want)
	}
	return nil
}

// ReceiptFromDriverResult binds a typed owner result to exactly one matrix
// cell. It intentionally requires the expected assertion text and leaves
// external identity/evidence validation to Receipt.Validate and the owners of
// Test Genie, Bridge, BAS, QEMU, VPS, and soak receipts.
func ReceiptFromDriverResult(m *Matrix, result DriverResult) (Receipt, error) {
	caseRow, driverRef, err := validateDriverResult(m, result)
	if err != nil {
		return Receipt{}, err
	}
	receipt := receiptFromDriverResult(caseRow, driverRef, result)
	if result.Verdict == VerdictPassed {
		if err := validatePassedDriverResult(result, driverRef); err != nil {
			return Receipt{}, err
		}
		receipt.ArtifactRefs = passedDriverArtifactRefs(result)
	}
	if err := receipt.Validate(m); err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

func validateDriverResult(m *Matrix, result DriverResult) (Case, string, error) {
	if m == nil {
		return Case{}, "", fmt.Errorf("matrix is required")
	}
	caseRow, ok := m.Case(result.Cell.CaseID)
	if !ok || !caseRow.Required || !containsLane(caseRow.Lanes, result.Cell.Lane) {
		return Case{}, "", fmt.Errorf("cell %s is not required by the matrix", result.Cell)
	}
	if !containsGround(m.TargetGroundsFor(result.Cell.CaseID, result.Cell.Lane), result.Cell.TargetGround) {
		return Case{}, "", fmt.Errorf("target ground %s is not applicable to %s", result.Cell.TargetGround, result.Cell)
	}
	wantDriver, err := driverRefForMatrix(m, result.Cell)
	if err != nil {
		return Case{}, "", err
	}
	if result.DriverRef != wantDriver {
		return Case{}, "", fmt.Errorf("driver_ref must be %q for %s", wantDriver, result.Cell)
	}
	if strings.TrimSpace(result.Assertion) != caseRow.Assertion {
		return Case{}, "", fmt.Errorf("assertion result does not match the frozen matrix assertion for %s", result.Cell)
	}
	return caseRow, wantDriver, nil
}

func receiptFromDriverResult(caseRow Case, driverRef string, result DriverResult) Receipt {
	return Receipt{
		SchemaVersion: 2, CaseID: result.Cell.CaseID, DriverRef: driverRef,
		Verdict: result.Verdict, Lane: result.Cell.Lane,
		RequirementRefs: append([]string(nil), caseRow.RequirementRefs...),
		Candidate:       result.Candidate, Target: result.Target,
		OperationRefs:        append([]string(nil), result.OperationRefs...),
		ValidationReceiptRef: result.ValidationReceiptRef,
		ArtifactRefs:         append(append([]string(nil), result.TargetReceiptRefs...), result.AssertionArtifacts...),
		Assertions:           []string{caseRow.Assertion}, ObservedAt: result.ObservedAt,
		Limitations:        append([]string(nil), result.Limitations...),
		TargetReceiptProof: result.TargetReceiptProof, TestGenieAssertionProof: result.TestGenieProof,
		BridgeAssertionOwnerProof: result.BridgeOwnerProof,
		BASAssertionOwnerProof:    result.BASOwnerProof,
		ownerEvidenceVerified:     result.ownerEvidenceVerified,
	}
}

func validatePassedDriverResult(result DriverResult, driverRef string) error {
	if result.TargetReceiptProof == nil || result.TestGenieProof == nil {
		return fmt.Errorf("passed result requires authenticated target receipt and Test Genie assertion proofs")
	}
	if err := validateTargetReceiptProof(result.TargetReceiptProof, result.Cell, driverRef, result.Target, result.OperationRefs); err != nil {
		return err
	}
	if err := validateTestGenieAssertionProof(result.TestGenieProof, result.Cell, driverRef, result.ValidationReceiptRef, result.Candidate, result.Target); err != nil {
		return err
	}
	if err := validateLaneOwnerReceiptRef(result.Cell.Lane, result.TestGenieProof.Envelope.OwnerReceiptRef, result.ValidationReceiptRef, result.TargetReceiptProof); err != nil {
		return err
	}
	if err := validateOptionalBridgeProof(result); err != nil {
		return err
	}
	return validateOptionalBASProof(result)
}

func validateOptionalBridgeProof(result DriverResult) error {
	if result.Cell.Lane == LaneRelay {
		return validateBridgeAssertionOwnerProof(result.BridgeOwnerProof, result.Target, result.TestGenieProof)
	}
	if result.BridgeOwnerProof != nil {
		return fmt.Errorf("Bridge owner proof is only valid for relay lane cells")
	}
	return nil
}

func validateOptionalBASProof(result DriverResult) error {
	if result.Cell.Lane == LaneBASUI {
		return validateBASAssertionOwnerProof(result.BASOwnerProof, result.Target, result.TestGenieProof)
	}
	if result.BASOwnerProof != nil {
		return fmt.Errorf("BAS owner proof is only valid for bas_ui lane cells")
	}
	return nil
}

func passedDriverArtifactRefs(result DriverResult) []string {
	refs := []string{result.TargetReceiptProof.ArtifactRef, result.TestGenieProof.ArtifactRef, result.TestGenieProof.AssertionArtifactRef}
	if result.BridgeOwnerProof != nil {
		refs = append(refs, result.BridgeOwnerProof.ArtifactRef)
	}
	if result.BASOwnerProof != nil {
		refs = append(refs, result.BASOwnerProof.ArtifactRef)
	}
	return refs
}

func validateLaneOwnerReceiptRef(lane Lane, ref, testGenieReceipt string, targetProof *TargetReceiptProof) error {
	if !laneOwnerProofSupported(lane) {
		return fmt.Errorf("lane %s owner receipt adapter is not implemented; pass remains unavailable", lane)
	}
	if err := validateOwnerReceiptScheme(lane, ref); err != nil {
		return err
	}
	switch lane {
	case LanePackage, LaneAPI, LaneWorkflow:
		if ref != testGenieReceipt {
			return fmt.Errorf("lane %s owner receipt must equal its successful Test Genie receipt", lane)
		}
	case LaneQEMU, LaneRealVPS:
		if targetProof == nil || ref != targetProof.ArtifactRef {
			return fmt.Errorf("lane %s owner receipt must equal the exact target-owner receipt proof", lane)
		}
	}
	return nil
}
