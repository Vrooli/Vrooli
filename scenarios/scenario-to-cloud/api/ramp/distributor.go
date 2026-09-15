package ramp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/evidence"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// PublishCase is the ExecutionRequest case id for the activation effect.
const PublishCase = "PUBLISH"

// DistributionKind is the spine target kind for a cloud publication target.
const DistributionKind = "vps"

// Distributor activates an approved release on the target. The effect
// receipt is the target's own active-release pointer read back after the
// activation, never the pipeline's acceptance: an activation whose pointer
// cannot be read is unavailable, and a pointer naming a different release
// is failed.
type Distributor struct {
	Executor Executor
	Target   TargetReader
	Now      func() time.Time
}

// Distribute implements deliveryramp.Distributor.
func (d Distributor) Distribute(ctx context.Context, request deliveryramp.DistributionRequest) (result deliveryramp.DistributionResult, err error) {
	defer func() {
		if err != nil {
			return
		}
		if validationErr := request.ValidateResult(result); validationErr != nil {
			err = fmt.Errorf("validate cloud distribution result: %w", validationErr)
			result = deliveryramp.DistributionResult{}
		}
	}()
	targetID := strings.TrimSpace(request.Cell.Target.ID)
	deploymentID := strings.TrimSpace(request.Artifact.Metadata["deployment_id"])
	releaseDigest := ReleaseDigestFromRef(request.Artifact.ImmutableRef)
	targets := []deliveryramp.DistributionTarget{{ID: targetID, Kind: DistributionKind, Available: true}}
	if targetID == "" || deploymentID == "" || releaseDigest == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "cloud distribution requires a target, a deployment and a release digest"}, nil
	}
	if !request.Cell.Target.Available {
		reason := request.Cell.Target.Reason
		if reason == "" {
			reason = "target is unavailable"
		}
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: reason, Targets: []deliveryramp.DistributionTarget{{ID: targetID, Kind: DistributionKind, Available: false, Reason: reason}}}, nil
	}
	if d.Executor == nil || d.Target == nil {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "cloud distributor has no executor or target reader", Targets: targets}, nil
	}
	execution, execErr := d.Executor.Execute(ctx, ExecutionRequest{DeploymentID: deploymentID, ReleaseDigest: releaseDigest, CaseID: PublishCase, RequestKey: strings.TrimSpace(request.Cell.ID)})
	if execErr != nil {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "activation was not performed: " + execErr.Error(), Targets: targets}, nil
	}
	if execution.Outcome == evidence.DispositionFailed {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionFailed, Reason: firstNonEmpty(execution.Reason, "activation failed"), Targets: targets}, nil
	}
	receipt, readErr := d.Target.ReadActiveRelease(ctx, deploymentID)
	if readErr != nil {
		// Delivery acknowledgement without the target's pointer is not a pass.
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "activation was dispatched but the target receipt could not be read: " + readErr.Error(), Targets: targets}, nil
	}
	if !strings.EqualFold(strings.TrimPrefix(receipt.ActiveRelease, "sha256:"), strings.TrimPrefix(releaseDigest, "sha256:")) {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionFailed, Reason: fmt.Sprintf("target reports active release %s, not %s", receipt.ActiveRelease, releaseDigest), Targets: targets}, nil
	}
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	observedAt := now
	if parsed, err := time.Parse(time.RFC3339, receipt.ActivatedAt); err == nil {
		observedAt = parsed.UTC()
	}
	receipt.ReceiptDigest = receipt.Digest()
	references := []deliveryramp.EvidenceReference{{ID: receipt.ReceiptDigest, Kind: "target-active-release", URI: "cloud-target://" + deploymentID + "/active-release.json", Checksum: receipt.ReceiptDigest, Redacted: true}}
	for _, ref := range execution.ReceiptRefs {
		references = append(references, deliveryramp.EvidenceReference{ID: ref, Kind: "target-receipt", URI: "receipt://" + ref, Checksum: "sha256:" + strings.TrimPrefix(releaseDigest, "sha256:"), Redacted: true})
	}
	return deliveryramp.DistributionResult{
		Disposition: deliveryramp.DispositionPass, Targets: targets, References: references, CapabilityReady: true,
		EffectReceipt: &deliveryramp.DistributionEffectReceipt{
			TargetID: targetID, ArtifactRef: request.Artifact.ImmutableRef,
			ExternalReceipt: "cloud-target:active-release:" + receipt.ReceiptDigest, Outcome: "activated", ObservedAt: observedAt,
		},
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

var _ deliveryramp.Distributor = Distributor{}
