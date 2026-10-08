package releases

import (
	"fmt"
	"time"

	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Proto converts the domain target identity to the generated wire contract.
func (t TargetIdentity) Proto() *releasesv1.TargetIdentity {
	return &releasesv1.TargetIdentity{Id: t.ID, Platform: t.Platform, Os: t.OS, Architecture: t.Architecture, Format: t.Format}
}

func targetIdentityFromProto(value *releasesv1.TargetIdentity) (TargetIdentity, error) {
	if value == nil {
		return TargetIdentity{}, fmt.Errorf("target identity is required")
	}
	return (TargetIdentity{ID: value.GetId(), Platform: value.GetPlatform(), OS: value.GetOs(), Architecture: value.GetArchitecture(), Format: value.GetFormat()}).canonical()
}

// Proto converts the canonical candidate, including the computed candidate id.
func (c Candidate) Proto() (*releasesv1.Candidate, error) {
	canonical, err := c.Canonical()
	if err != nil {
		return nil, err
	}
	id, err := canonical.Identity()
	if err != nil {
		return nil, err
	}
	artifacts := make([]*releasesv1.CandidateArtifact, 0, len(canonical.Artifacts))
	for _, artifact := range canonical.Artifacts {
		artifacts = append(artifacts, &releasesv1.CandidateArtifact{
			Target:          artifact.Target.Proto(),
			ImmutableRef:    artifact.ImmutableRef,
			Digest:          artifact.Digest,
			SizeBytes:       artifact.SizeBytes,
			SignatureDigest: artifact.SignatureDigest,
			SignerRef:       artifact.SignerRef,
		})
	}
	var declaration *releasesv1.CapabilityDeclaration
	if canonical.CapabilityDeclaration != nil {
		declaration = &releasesv1.CapabilityDeclaration{
			SupportOwner: canonical.CapabilityDeclaration.SupportOwner, IncidentOwner: canonical.CapabilityDeclaration.IncidentOwner,
			CustomerContact: canonical.CapabilityDeclaration.CustomerContact, ReleaseAuthority: canonical.CapabilityDeclaration.ReleaseAuthority,
			RollbackAuthority: canonical.CapabilityDeclaration.RollbackAuthority, DegradedModeAuthority: canonical.CapabilityDeclaration.DegradedModeAuthority,
		}
	}
	return &releasesv1.Candidate{SourceRevision: canonical.SourceRevision, ProfileRevision: canonical.ProfileRevision, BuildInputs: canonical.BuildInputs, DependencyLockDigest: canonical.DependencyLockDigest, PolicyDigest: canonical.PolicyDigest, Artifacts: artifacts, CandidateId: id, CapabilityDeclaration: declaration}, nil
}

func candidateFromProto(value *releasesv1.Candidate) (Candidate, error) {
	if value == nil {
		return Candidate{}, fmt.Errorf("candidate is required")
	}
	candidate := Candidate{SourceRevision: value.GetSourceRevision(), ProfileRevision: value.GetProfileRevision(), BuildInputs: value.GetBuildInputs(), DependencyLockDigest: value.GetDependencyLockDigest(), PolicyDigest: value.GetPolicyDigest()}
	if declaration := value.GetCapabilityDeclaration(); declaration != nil {
		candidate.CapabilityDeclaration = &CapabilityDeclaration{
			SupportOwner: declaration.GetSupportOwner(), IncidentOwner: declaration.GetIncidentOwner(), CustomerContact: declaration.GetCustomerContact(),
			ReleaseAuthority: declaration.GetReleaseAuthority(), RollbackAuthority: declaration.GetRollbackAuthority(), DegradedModeAuthority: declaration.GetDegradedModeAuthority(),
		}
	}
	for _, artifact := range value.GetArtifacts() {
		if artifact == nil {
			return Candidate{}, fmt.Errorf("candidate contains a nil artifact")
		}
		target, err := targetIdentityFromProto(artifact.GetTarget())
		if err != nil {
			return Candidate{}, err
		}
		candidate.Artifacts = append(candidate.Artifacts, CandidateArtifact{Target: target, ImmutableRef: artifact.GetImmutableRef(), Digest: artifact.GetDigest(), SizeBytes: artifact.GetSizeBytes(), SignatureDigest: artifact.GetSignatureDigest(), SignerRef: artifact.GetSignerRef()})
	}
	canonical, err := candidate.Canonical()
	if err != nil {
		return Candidate{}, err
	}
	if value.GetCandidateId() != "" {
		id, err := canonical.Identity()
		if err != nil {
			return Candidate{}, err
		}
		if id != value.GetCandidateId() {
			return Candidate{}, fmt.Errorf("candidate identity mismatch: expected %q, got %q", id, value.GetCandidateId())
		}
	}
	return canonical, nil
}

// CandidateFromProto converts a producer registration request into the
// canonical domain identity. The candidate_id field is checked when present;
// the server derives the persisted ID from the canonical value.
func CandidateFromProto(value *releasesv1.Candidate) (Candidate, error) {
	return candidateFromProto(value)
}

// Proto converts a canonical destination revision and includes its computed id.
func (d DestinationRevision) Proto() (*releasesv1.DestinationRevision, error) {
	canonical, err := d.Canonical()
	if err != nil {
		return nil, err
	}
	id, err := canonical.Identity()
	if err != nil {
		return nil, err
	}
	return &releasesv1.DestinationRevision{Kind: canonical.Kind, DestinationId: canonical.DestinationID, ConfigurationDigest: canonical.ConfigurationDigest, Channel: canonical.Channel, ExpectedChannelRevision: canonical.ExpectedChannelRevision, DestinationRevisionId: id}, nil
}

func destinationRevisionFromProto(value *releasesv1.DestinationRevision) (DestinationRevision, error) {
	if value == nil {
		return DestinationRevision{}, fmt.Errorf("destination revision is required")
	}
	canonical, err := (DestinationRevision{Kind: value.GetKind(), DestinationID: value.GetDestinationId(), ConfigurationDigest: value.GetConfigurationDigest(), Channel: value.GetChannel(), ExpectedChannelRevision: value.GetExpectedChannelRevision()}).Canonical()
	if err != nil {
		return DestinationRevision{}, err
	}
	if value.GetDestinationRevisionId() != "" {
		id, err := canonical.Identity()
		if err != nil {
			return DestinationRevision{}, err
		}
		if id != value.GetDestinationRevisionId() {
			return DestinationRevision{}, fmt.Errorf("destination revision identity mismatch: expected %q, got %q", id, value.GetDestinationRevisionId())
		}
	}
	return canonical, nil
}

// DestinationRevisionFromProto converts a producer registration request into
// the canonical destination identity.
func DestinationRevisionFromProto(value *releasesv1.DestinationRevision) (DestinationRevision, error) {
	return destinationRevisionFromProto(value)
}

// Proto converts a review binding and includes its computed review id.
func (b ReviewBinding) Proto() (*releasesv1.ReviewBinding, error) {
	canonical, err := b.Canonical()
	if err != nil {
		return nil, err
	}
	id, err := canonical.Identity()
	if err != nil {
		return nil, err
	}
	return &releasesv1.ReviewBinding{CandidateId: canonical.CandidateID, DestinationRevisionId: canonical.DestinationRevisionID, Targets: canonical.Targets, Channel: canonical.Channel, EvidenceSetDigest: canonical.EvidenceSetDigest, PolicyDigest: canonical.PolicyDigest, AuthorizationEpoch: canonical.AuthorizationEpoch, ReviewId: id}, nil
}

func reviewBindingFromProto(value *releasesv1.ReviewBinding) (ReviewBinding, error) {
	if value == nil {
		return ReviewBinding{}, fmt.Errorf("review binding is required")
	}
	binding, err := (ReviewBinding{CandidateID: value.GetCandidateId(), DestinationRevisionID: value.GetDestinationRevisionId(), Targets: value.GetTargets(), Channel: value.GetChannel(), EvidenceSetDigest: value.GetEvidenceSetDigest(), PolicyDigest: value.GetPolicyDigest(), AuthorizationEpoch: value.GetAuthorizationEpoch()}).Canonical()
	if err != nil {
		return ReviewBinding{}, err
	}
	if value.GetReviewId() != "" {
		id, err := binding.Identity()
		if err != nil {
			return ReviewBinding{}, err
		}
		if id != value.GetReviewId() {
			return ReviewBinding{}, fmt.Errorf("review identity mismatch: expected %q, got %q", id, value.GetReviewId())
		}
	}
	return binding, nil
}

// Proto converts producer-attributed publication evidence.
func (r PublicationReceipt) Proto() *releasesv1.PublicationReceipt {
	return &releasesv1.PublicationReceipt{CandidateId: r.CandidateID, DestinationRevisionId: r.DestinationRevisionID, TargetId: r.TargetID, ArtifactDigest: r.ArtifactDigest, DestinationObject: r.DestinationObject, Producer: r.Producer, ExternalReceipt: r.ExternalReceipt, Outcome: receiptOutcomeToProto(r.Outcome), ObservedAt: protoTime(r.ObservedAt)}
}

func publicationReceiptFromProto(value *releasesv1.PublicationReceipt) PublicationReceipt {
	if value == nil {
		return PublicationReceipt{}
	}
	return PublicationReceipt{CandidateID: value.GetCandidateId(), DestinationRevisionID: value.GetDestinationRevisionId(), TargetID: value.GetTargetId(), ArtifactDigest: value.GetArtifactDigest(), DestinationObject: value.GetDestinationObject(), Producer: value.GetProducer(), ExternalReceipt: value.GetExternalReceipt(), Outcome: receiptOutcomeFromProto(value.GetOutcome()), ObservedAt: fromProtoTime(value.GetObservedAt())}
}

// Proto converts producer-attributed client-update evidence.
func (r ClientUpdateReceipt) Proto() *releasesv1.ClientUpdateReceipt {
	return &releasesv1.ClientUpdateReceipt{CandidateId: r.CandidateID, PredecessorRef: r.PredecessorRef, SuccessorDigest: r.SuccessorDigest, TargetId: r.TargetID, VerifiedVersion: r.VerifiedVersion, Outcome: receiptOutcomeToProto(r.Outcome), Producer: r.Producer, ExternalReceipt: r.ExternalReceipt, ObservedAt: protoTime(r.ObservedAt)}
}

// ClientUpdateReceiptFromProto converts the owner wire contract into the
// canonical domain receipt used by durable qualification and health checks.
func ClientUpdateReceiptFromProto(value *releasesv1.ClientUpdateReceipt) ClientUpdateReceipt {
	if value == nil {
		return ClientUpdateReceipt{}
	}
	return ClientUpdateReceipt{CandidateID: value.GetCandidateId(), PredecessorRef: value.GetPredecessorRef(), SuccessorDigest: value.GetSuccessorDigest(), TargetID: value.GetTargetId(), VerifiedVersion: value.GetVerifiedVersion(), Outcome: receiptOutcomeFromProto(value.GetOutcome()), Producer: value.GetProducer(), ExternalReceipt: value.GetExternalReceipt(), ObservedAt: fromProtoTime(value.GetObservedAt())}
}

func clientUpdateReceiptFromProto(value *releasesv1.ClientUpdateReceipt) ClientUpdateReceipt {
	return ClientUpdateReceiptFromProto(value)
}

// Proto converts durable owner recovery evidence to the typed wire contract.
func (r RecoveryReceipt) Proto() *releasesv1.RecoveryReceipt {
	return &releasesv1.RecoveryReceipt{
		ReleaseId: r.ReleaseID, CandidateId: r.CandidateID, DestinationRevisionId: r.DestinationRevisionID,
		DeploymentId: r.DeploymentID, Action: r.Action, Outcome: r.Outcome, Health: r.Health,
		BundleSha256: r.BundleSHA256, ExternalReceipt: r.ExternalReceipt, ObservedAt: protoTime(r.ObservedAt), DryRun: r.DryRun,
	}
}

func receiptOutcomeToProto(outcome ReceiptOutcome) releasesv1.ReceiptOutcome {
	values := map[ReceiptOutcome]releasesv1.ReceiptOutcome{
		ReceiptPrepared: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_PREPARED, ReceiptStaged: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_STAGED, ReceiptPublished: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_PUBLISHED, ReceiptVerified: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_VERIFIED, ReceiptFailed: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_FAILED, ReceiptAmbiguous: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_AMBIGUOUS, ReceiptUnavailable: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_UNAVAILABLE,
	}
	return values[outcome]
}

func receiptOutcomeFromProto(outcome releasesv1.ReceiptOutcome) ReceiptOutcome {
	values := map[releasesv1.ReceiptOutcome]ReceiptOutcome{
		releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_PREPARED: ReceiptPrepared, releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_STAGED: ReceiptStaged, releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_PUBLISHED: ReceiptPublished, releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_VERIFIED: ReceiptVerified, releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_FAILED: ReceiptFailed, releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_AMBIGUOUS: ReceiptAmbiguous, releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_UNAVAILABLE: ReceiptUnavailable,
	}
	return values[outcome]
}

func protoTime(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

func fromProtoTime(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
}

// Proto converts an operation standing to its typed wire contract.
func (o Operation) Proto() *releasesv1.ReleaseOperation {
	status := map[string]releasesv1.ReleaseOperationStatus{OperationQueued: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_QUEUED, OperationRunning: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_RUNNING, OperationComplete: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_COMPLETE, OperationFailed: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_FAILED, OperationCanceled: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_CANCELED, OperationAmbiguous: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_AMBIGUOUS}[o.Status]
	return &releasesv1.ReleaseOperation{OperationId: o.ID, ReleaseId: o.ReleaseID, ProfileId: o.ProfileID, IdempotencyKey: o.IdempotencyKey, Status: status, ActiveStage: o.ActiveStage, Error: o.Error, CreatedAt: protoTime(o.CreatedAt), UpdatedAt: protoTime(o.UpdatedAt), CompletedAt: protoTime(pointerTime(o.CompletedAt))}
}

func operationFromProto(value *releasesv1.ReleaseOperation) Operation {
	if value == nil {
		return Operation{}
	}
	status := map[releasesv1.ReleaseOperationStatus]string{releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_QUEUED: OperationQueued, releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_RUNNING: OperationRunning, releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_COMPLETE: OperationComplete, releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_FAILED: OperationFailed, releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_CANCELED: OperationCanceled, releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_AMBIGUOUS: OperationAmbiguous}[value.GetStatus()]
	completed := fromProtoTime(value.GetCompletedAt())
	var completedAt *time.Time
	if !completed.IsZero() {
		completedAt = &completed
	}
	return Operation{ID: value.GetOperationId(), ReleaseID: value.GetReleaseId(), ProfileID: value.GetProfileId(), IdempotencyKey: value.GetIdempotencyKey(), Status: status, ActiveStage: value.GetActiveStage(), Error: value.GetError(), CreatedAt: fromProtoTime(value.GetCreatedAt()), UpdatedAt: fromProtoTime(value.GetUpdatedAt()), CompletedAt: completedAt}
}

func pointerTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
