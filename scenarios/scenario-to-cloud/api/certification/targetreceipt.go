package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
)

// TargetReceiptOwner reads one fresh target-owner receipt through the
// authenticated scenario-to-cloud operation boundary.
type TargetReceiptOwner interface {
	GetTargetReceipt(context.Context, string, string, string) (*operationsv1.TargetReceiptRead, error)
}

// TargetReceiptExpectation is the immutable operation and target identity
// recorded by the ground-specific qualification runner.
type TargetReceiptExpectation struct {
	Cell         CellID
	DriverRef    string
	DeploymentID string
	OperationID  string
	PlanDigest   string
	Step         string
	Target       Target
}

// TargetReceiptProof is the minimized, redacted projection of a fresh
// owner-authenticated receipt. Digest covers every field except itself and
// the generated ArtifactRef; sensitive receipt detail/error text is omitted.
type TargetReceiptProof struct {
	Cell                 CellID       `json:"cell"`
	DriverRef            string       `json:"driver_ref"`
	DeploymentID         string       `json:"deployment_id"`
	OperationID          string       `json:"operation_id"`
	PlanDigest           string       `json:"plan_digest"`
	Step                 string       `json:"step"`
	TargetGround         TargetGround `json:"target_ground"`
	MachineID            string       `json:"machine_id"`
	NodeID               string       `json:"node_id"`
	EnrollmentGeneration uint64       `json:"enrollment_generation"`
	Transport            string       `json:"transport"`
	Outcome              string       `json:"outcome"`
	Fence                uint64       `json:"fence"`
	SHA256               string       `json:"sha256"`
	ArtifactRef          string       `json:"artifact_ref"`
}

type targetReceiptDigestFields struct {
	Cell                 CellID       `json:"cell"`
	DriverRef            string       `json:"driver_ref"`
	DeploymentID         string       `json:"deployment_id"`
	OperationID          string       `json:"operation_id"`
	PlanDigest           string       `json:"plan_digest"`
	Step                 string       `json:"step"`
	TargetGround         TargetGround `json:"target_ground"`
	MachineID            string       `json:"machine_id"`
	NodeID               string       `json:"node_id"`
	EnrollmentGeneration uint64       `json:"enrollment_generation"`
	Transport            string       `json:"transport"`
	Outcome              string       `json:"outcome"`
	Fence                uint64       `json:"fence"`
}

// VerifyTargetOwnerReceipt fetches and binds one fresh receipt to the exact
// matrix cell, operation plan, deployment and admitted target. The ground
// naming fence follows the QEMU/VPS runner contracts and prevents a response
// from one ground being relabeled as the other.
func VerifyTargetOwnerReceipt(ctx context.Context, owner TargetReceiptOwner, expected TargetReceiptExpectation) (TargetReceiptProof, error) {
	if err := validateTargetReceiptExpectation(expected); err != nil {
		return TargetReceiptProof{}, err
	}
	if owner == nil {
		return TargetReceiptProof{}, fmt.Errorf("target receipt owner is required")
	}
	read, err := owner.GetTargetReceipt(ctx, expected.DeploymentID, expected.OperationID, expected.Step)
	if err != nil {
		return TargetReceiptProof{}, fmt.Errorf("read target-owned operation receipt: %w", err)
	}
	if err := validateTargetReceiptRead(read, expected); err != nil {
		return TargetReceiptProof{}, err
	}
	return buildTargetReceiptProof(expected, read)
}

func validateTargetReceiptExpectation(expected TargetReceiptExpectation) error {
	if err := validateTargetReceiptCell(expected); err != nil {
		return err
	}
	if expected.DeploymentID == "" || expected.OperationID == "" || expected.PlanDigest == "" || expected.Step == "" {
		return fmt.Errorf("deployment, operation, plan digest and step are required")
	}
	if !isSHA256Digest(expected.PlanDigest) {
		return fmt.Errorf("plan digest must be sha256:<64 hex>")
	}
	return validateTargetReceiptIdentity(expected)
}

func validateTargetReceiptCell(expected TargetReceiptExpectation) error {
	matrix, err := LoadEmbedded()
	if err != nil {
		return err
	}
	if _, ok := matrix.Case(expected.Cell.CaseID); !ok || !containsGround(matrix.TargetGroundsFor(expected.Cell.CaseID, expected.Cell.Lane), expected.Cell.TargetGround) {
		return fmt.Errorf("cell %s is not required on target ground %s", expected.Cell, expected.Cell.TargetGround)
	}
	wantDriver, err := driverRefForMatrix(matrix, expected.Cell)
	if err != nil {
		return err
	}
	if expected.DriverRef != wantDriver {
		return fmt.Errorf("driver_ref must be %q for %s", wantDriver, expected.Cell)
	}
	return nil
}

func validateTargetReceiptIdentity(expected TargetReceiptExpectation) error {
	target := expected.Target
	if target.Ground != expected.Cell.TargetGround || target.MachineID == "" || target.NodeID == "" || target.Generation == 0 || target.Transport == "" || target.Architecture == "" || target.OS == "" {
		return fmt.Errorf("target identity is incomplete or does not match the matrix ground")
	}
	prefix, ok := targetGroundMachinePrefix(expected.Cell.TargetGround)
	if !ok || !strings.HasPrefix(target.MachineID, prefix) {
		return fmt.Errorf("machine id %q is not owned by target ground %s", target.MachineID, expected.Cell.TargetGround)
	}
	return nil
}

func validateTargetReceiptRead(read *operationsv1.TargetReceiptRead, expected TargetReceiptExpectation) error {
	if !matchesTargetReceiptEnvelope(read, expected) {
		return fmt.Errorf("target receipt envelope does not match deployment, operation, plan and step")
	}
	if !matchesTargetReceiptIdentity(read.GetTarget(), expected.Target) {
		return fmt.Errorf("target receipt identity does not match the admitted target")
	}
	if !isSuccessfulTargetReceipt(read.GetReceipt()) {
		return fmt.Errorf("target owner has no successful, fenced receipt for %s", expected.Step)
	}
	return nil
}

func matchesTargetReceiptEnvelope(read *operationsv1.TargetReceiptRead, expected TargetReceiptExpectation) bool {
	return read != nil && read.GetSchemaVersion() == "1" && read.GetDeploymentId() == expected.DeploymentID && read.GetOperationId() == expected.OperationID && read.GetPlanDigest() == expected.PlanDigest && read.GetStep() == expected.Step
}

func matchesTargetReceiptIdentity(target *operationsv1.TargetReceiptTarget, expected Target) bool {
	return target != nil && target.GetMachineId() == expected.MachineID && target.GetNodeId() == expected.NodeID && target.GetEnrollmentGeneration() == expected.Generation && target.GetTransport() == expected.Transport
}

func isSuccessfulTargetReceipt(receipt *operationsv1.TargetOwnerReceipt) bool {
	if receipt == nil || !receipt.GetFound() || receipt.GetFence() == 0 || receipt.GetError() != "" {
		return false
	}
	return receipt.GetOutcome() == "succeeded" || receipt.GetOutcome() == "unchanged"
}

func buildTargetReceiptProof(expected TargetReceiptExpectation, read *operationsv1.TargetReceiptRead) (TargetReceiptProof, error) {
	target, receipt := read.GetTarget(), read.GetReceipt()
	fields := targetReceiptDigestFields{
		Cell: expected.Cell, DriverRef: expected.DriverRef,
		DeploymentID: expected.DeploymentID, OperationID: expected.OperationID, PlanDigest: expected.PlanDigest,
		Step: expected.Step, TargetGround: expected.Cell.TargetGround, MachineID: target.GetMachineId(),
		NodeID: target.GetNodeId(), EnrollmentGeneration: target.GetEnrollmentGeneration(),
		Transport: target.GetTransport(), Outcome: receipt.GetOutcome(), Fence: receipt.GetFence(),
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return TargetReceiptProof{}, fmt.Errorf("encode target receipt proof: %w", err)
	}
	digest := sha256.Sum256(raw)
	sha := hex.EncodeToString(digest[:])
	artifactRef := targetReceiptArtifactRef(expected.OperationID, expected.Step, receipt.GetFence(), sha)
	return TargetReceiptProof{
		Cell: fields.Cell, DriverRef: fields.DriverRef,
		DeploymentID: fields.DeploymentID, OperationID: fields.OperationID, PlanDigest: fields.PlanDigest,
		Step: fields.Step, TargetGround: fields.TargetGround, MachineID: fields.MachineID, NodeID: fields.NodeID,
		EnrollmentGeneration: fields.EnrollmentGeneration, Transport: fields.Transport, Outcome: fields.Outcome,
		Fence: fields.Fence, SHA256: sha, ArtifactRef: artifactRef,
	}, nil
}

func targetReceiptArtifactRef(operationID, step string, fence uint64, digest string) string {
	u := url.URL{Scheme: "target-receipt", Host: operationID, Path: "/" + step}
	query := url.Values{}
	query.Set("fence", fmt.Sprint(fence))
	query.Set("sha256", digest)
	u.RawQuery = query.Encode()
	return u.String()
}

func validateTargetReceiptProof(proof *TargetReceiptProof, cell CellID, driverRef string, target Target, operationRefs []string) error {
	if proof == nil {
		return fmt.Errorf("passed receipt requires its authenticated target receipt proof")
	}
	if err := validateTargetReceiptProofIdentity(proof, cell, driverRef, target, operationRefs); err != nil {
		return err
	}
	if err := validateTargetReceiptProofOwnerFields(proof, cell.TargetGround); err != nil {
		return err
	}
	return validateTargetReceiptProofDigest(proof)
}

func validateTargetReceiptProofIdentity(proof *TargetReceiptProof, cell CellID, driverRef string, target Target, operationRefs []string) error {
	if proof.Cell != cell || proof.DriverRef != driverRef {
		return fmt.Errorf("target receipt proof does not match the exact cell driver")
	}
	if proof.TargetGround != cell.TargetGround || proof.MachineID != target.MachineID || proof.NodeID != target.NodeID || proof.EnrollmentGeneration != target.Generation || proof.Transport != target.Transport {
		return fmt.Errorf("target receipt proof identity does not match the certification target")
	}
	if len(operationRefs) != 1 || operationRefs[0] != "operation:"+proof.OperationID {
		return fmt.Errorf("target receipt proof must bind the exact single operation_ref")
	}
	return nil
}

func validateTargetReceiptProofOwnerFields(proof *TargetReceiptProof, ground TargetGround) error {
	if proof.DeploymentID == "" || proof.OperationID == "" || !isSHA256Digest(proof.PlanDigest) || proof.Step == "" || proof.Fence == 0 || (proof.Outcome != "succeeded" && proof.Outcome != "unchanged") {
		return fmt.Errorf("target receipt proof has incomplete or unsuccessful owner fields")
	}
	prefix, ok := targetGroundMachinePrefix(ground)
	if !ok || !strings.HasPrefix(proof.MachineID, prefix) {
		return fmt.Errorf("target receipt proof machine id does not belong to %s", ground)
	}
	return nil
}

func validateTargetReceiptProofDigest(proof *TargetReceiptProof) error {
	fields := targetReceiptDigestFields{
		Cell: proof.Cell, DriverRef: proof.DriverRef, DeploymentID: proof.DeploymentID, OperationID: proof.OperationID,
		PlanDigest: proof.PlanDigest, Step: proof.Step, TargetGround: proof.TargetGround, MachineID: proof.MachineID,
		NodeID: proof.NodeID, EnrollmentGeneration: proof.EnrollmentGeneration, Transport: proof.Transport,
		Outcome: proof.Outcome, Fence: proof.Fence,
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("encode target receipt proof: %w", err)
	}
	digest := sha256.Sum256(raw)
	if proof.SHA256 != hex.EncodeToString(digest[:]) {
		return fmt.Errorf("target receipt proof digest does not match its owner fields")
	}
	wantURL := targetReceiptArtifactRef(proof.OperationID, proof.Step, proof.Fence, proof.SHA256)
	if proof.ArtifactRef != wantURL {
		return fmt.Errorf("target receipt artifact ref does not match its authenticated proof")
	}
	return nil
}

func targetGroundMachinePrefix(ground TargetGround) (string, bool) {
	switch ground {
	case TargetGroundQEMU:
		return "local-qemu:", true
	case TargetGroundRealVPS:
		return "disposable-vps:", true
	default:
		return "", false
	}
}
