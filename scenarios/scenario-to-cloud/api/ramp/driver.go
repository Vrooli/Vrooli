package ramp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/evidence"

	"github.com/google/uuid"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// PlanCapability is the journey capability every cloud profile plan names.
const PlanCapability = CapabilityCloudLaunch

// Driver executes one journey cell: one qualification case of the
// capability profile against one deployment through the owner pipeline. The
// JourneyResult's step dispositions come from target-owned assertions in the
// execution result (operation step receipts, target receipts, observation
// ids). Dispatch acceptance without a target assertion is unavailable.
type Driver struct {
	Executor Executor
	Recorder Recorder
	Profile  *evidence.Profile
	Now      func() time.Time
}

// Execute implements deliveryramp.Driver.
func (d Driver) Execute(ctx context.Context, request deliveryramp.DriverRequest) (deliveryramp.JourneyResult, error) {
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	profile := d.Profile
	if profile == nil {
		loaded, err := evidence.LoadProfile(evidence.ProfileCloudLaunchV1)
		if err != nil {
			return deliveryramp.JourneyResult{}, err
		}
		profile = loaded
	}
	cell, err := evidence.ParseCellID(request.Cell.ID)
	if err != nil {
		return deliveryramp.JourneyResult{}, err
	}
	if !profile.Contains(cell) {
		return deliveryramp.JourneyResult{}, fmt.Errorf("cell %s is not part of profile %s", cell, profile.ID)
	}
	deploymentID := strings.TrimSpace(request.Artifact.Metadata["deployment_id"])
	if deploymentID == "" {
		return deliveryramp.JourneyResult{}, fmt.Errorf("driver request artifact does not name a deployment")
	}
	releaseDigest := ReleaseDigestFromRef(request.Artifact.ImmutableRef)
	if releaseDigest == "" {
		return deliveryramp.JourneyResult{}, fmt.Errorf("driver request artifact has no release digest")
	}
	targetKey := strings.TrimSpace(request.Cell.Target.ID)
	if targetKey == "" {
		return deliveryramp.JourneyResult{}, fmt.Errorf("driver request cell has no target")
	}
	result := deliveryramp.JourneyResult{
		SchemaVersion: deliveryramp.JourneySchemaVersion, EvidenceVersion: deliveryramp.JourneyEvidenceVersion,
		SmokeTestID: request.RunID, Capability: PlanCapability, PlanID: request.Plan.ID, Profile: profile.ID, Platform: "vps",
		TargetID: targetKey, CellID: request.Cell.ID, CreatedAt: now,
	}
	if d.Executor == nil {
		result.Disposition = deliveryramp.DispositionUnavailable
		result.DegradedReason = "no cell executor is configured"
		return result, nil
	}
	requestKey := strings.TrimSpace(request.RunID)
	if requestKey == "" {
		requestKey = uuid.New().String()
	}
	execution, execErr := d.Executor.Execute(ctx, ExecutionRequest{DeploymentID: deploymentID, ReleaseDigest: releaseDigest, CaseID: cell.CaseID, Lane: string(cell.Lane), RequestKey: requestKey + ":" + cell.String()})
	step := deliveryramp.JourneyStep{ID: cell.String(), Name: cell.CaseID, Action: "qualify", StartedAt: now}
	record := evidence.Record{
		SchemaVersion: evidence.RecordSchemaVersion, ID: uuid.New().String(), ProfileID: profile.ID, CaseID: cell.CaseID, Lane: cell.Lane,
		Binding:             evidence.Binding{ProducerRef: evidence.ProducerRef, DeploymentID: deploymentID, ReleaseDigest: releaseDigest, TargetKey: targetKey, OperationID: execution.OperationID},
		ConfigurationDigest: request.Artifact.Metadata["configuration_digest"],
		ReceiptRefs:         append([]string(nil), execution.ReceiptRefs...), Assertions: append([]string(nil), execution.Assertions...),
		ObservedAt: execution.CompletedAt, RecordedAt: now,
	}
	if record.ObservedAt.IsZero() {
		record.ObservedAt = now
	}
	switch {
	case execErr != nil:
		record.Disposition = evidence.DispositionUnavailable
		record.Reason = "owner pipeline did not run the cell: " + execErr.Error()
		record.Binding.ObservationID = "driver:" + requestKey
	case execution.OperationID == "" && len(execution.ReceiptRefs) == 0:
		record.Disposition = evidence.DispositionUnavailable
		record.Reason = "owner pipeline returned neither an operation nor target receipts"
		record.Binding.ObservationID = "driver:" + requestKey
	case execution.Outcome == evidence.DispositionPassed && len(execution.ReceiptRefs) == 0:
		// Dispatch acceptance is not a target assertion.
		record.Disposition = evidence.DispositionUnavailable
		record.Reason = "cell was accepted for execution but no target-owned receipt proves the assertion"
	case execution.Outcome.Recordable():
		record.Disposition = execution.Outcome
		record.Reason = execution.Reason
		if record.Disposition != evidence.DispositionPassed && record.Reason == "" {
			record.Reason = string(record.Disposition) + " reported by the owner pipeline"
		}
	default:
		record.Disposition = evidence.DispositionUnavailable
		record.Reason = fmt.Sprintf("owner pipeline reported unknown disposition %q", execution.Outcome)
	}
	if d.Recorder != nil {
		if err := d.Recorder.AppendEvidenceRecord(ctx, record); err != nil {
			return deliveryramp.JourneyResult{}, fmt.Errorf("record cell evidence: %w", err)
		}
	}
	step.CompletedAt = record.ObservedAt
	step.ObservedState = string(record.Disposition)
	step.ExpectedState = string(evidence.DispositionPassed)
	step.AssertionID = cell.CaseID
	step.AssertionStatus = string(record.Disposition)
	for _, ref := range record.ReceiptRefs {
		step.Evidence = append(step.Evidence, deliveryramp.EvidenceReference{ID: ref, Kind: "target-receipt", URI: "receipt://" + ref, Checksum: "sha256:" + strings.TrimPrefix(releaseDigest, "sha256:"), Redacted: true})
	}
	step.Evidence = append(step.Evidence, deliveryramp.EvidenceReference{ID: record.ID, Kind: "evidence-record", URI: "evidence://" + record.ID, Checksum: "sha256:" + strings.TrimPrefix(releaseDigest, "sha256:"), Redacted: true})
	switch record.Disposition {
	case evidence.DispositionPassed:
		step.Disposition = deliveryramp.StepPassed
	case evidence.DispositionFailed:
		step.Disposition = deliveryramp.StepFailed
		step.Error = record.Reason
	case evidence.DispositionSkipped:
		step.Disposition = deliveryramp.StepNotRun
		step.DegradedReason = record.Reason
	default:
		step.Disposition = deliveryramp.StepUnavailable
		step.DegradedReason = record.Reason
	}
	result.Steps = []deliveryramp.JourneyStep{step}
	result.Disposition = record.Disposition.ToRamp()
	if result.Disposition != deliveryramp.DispositionPass {
		result.DegradedReason = record.Reason
	}
	result.CompletedAt = record.ObservedAt
	return result, nil
}

var _ deliveryramp.Driver = Driver{}
