package releases

import (
	"testing"
	"time"

	"github.com/vrooli/api-core/targetmodel"
)

func candidateFixture() Candidate {
	return Candidate{
		SourceRevision: "commit-1", ProfileRevision: "profile-2",
		BuildInputs:          map[string]string{"go": "1.24", "template": "tpl-3"},
		DependencyLockDigest: "sha256:lock", PolicyDigest: "sha256:policy",
		Artifacts: []CandidateArtifact{
			{Target: TargetIdentity{ID: "win-x64", Platform: "windows", OS: "windows", Architecture: "amd64", Format: "msi"}, ImmutableRef: "candidate/win", Digest: "sha256:win", SizeBytes: 10, SignatureDigest: "sha256:sig-win", SignerRef: "signer:windows"},
			{Target: TargetIdentity{ID: "mac-arm64", Platform: "darwin", OS: "macos", Architecture: "arm64", Format: "dmg"}, ImmutableRef: "candidate/mac", Digest: "sha256:mac", SizeBytes: 11, SignatureDigest: "sha256:sig-mac", SignerRef: "signer:apple"},
		},
	}
}

func TestCandidateIdentityCanonicalizesArtifactOrder(t *testing.T) {
	first := candidateFixture()
	second := candidateFixture()
	second.Artifacts[0], second.Artifacts[1] = second.Artifacts[1], second.Artifacts[0]
	a, err := first.Identity()
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("equivalent candidate order changed identity: %s != %s", a, b)
	}
}

func TestCandidateIdentityChangesWhenFinalBytesOrArchitectureChanges(t *testing.T) {
	base, err := candidateFixture().Identity()
	if err != nil {
		t.Fatal(err)
	}
	changedBytes := candidateFixture()
	changedBytes.Artifacts[0].Digest = "sha256:other"
	changed, err := changedBytes.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if base == changed {
		t.Fatal("changed final bytes retained candidate identity")
	}
	changedTarget := candidateFixture()
	changedTarget.Artifacts[0].Target.Architecture = "arm64"
	changedTargetID, err := changedTarget.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if base == changedTargetID {
		t.Fatal("changed architecture retained candidate identity")
	}
}

func TestCandidateIdentityIncludesCapabilityDeclaration(t *testing.T) {
	base := candidateFixture()
	base.CapabilityDeclaration = &CapabilityDeclaration{
		SupportOwner: "team:support", IncidentOwner: "team:incident", CustomerContact: "mailto:support@example.test",
		ReleaseAuthority: "team:release", RollbackAuthority: "team:rollback", DegradedModeAuthority: "runbook:degraded-mode",
	}
	withDeclaration, err := base.Identity()
	if err != nil {
		t.Fatal(err)
	}
	changed := base
	declaration := *base.CapabilityDeclaration
	declaration.RollbackAuthority = "team:recovery"
	changed.CapabilityDeclaration = &declaration
	changedID, err := changed.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if withDeclaration == changedID {
		t.Fatal("changed operational authority retained candidate identity")
	}
	if _, err := (Candidate{SourceRevision: "commit", ProfileRevision: "profile", DependencyLockDigest: "lock", PolicyDigest: "policy", Artifacts: base.Artifacts, CapabilityDeclaration: &CapabilityDeclaration{SupportOwner: "support"}}).Identity(); err == nil {
		t.Fatal("incomplete capability declaration was accepted")
	}
}

func TestReviewBindingCanonicalizesTargetsAndRequiresAuthorityEpoch(t *testing.T) {
	binding := ReviewBinding{CandidateID: "candidate-1", DestinationRevisionID: "destination-1", Targets: []string{"win", "mac"}, Channel: "stable", EvidenceSetDigest: "sha256:evidence", PolicyDigest: "sha256:policy", AuthorizationEpoch: 2}
	first, err := binding.Identity()
	if err != nil {
		t.Fatal(err)
	}
	binding.Targets[0], binding.Targets[1] = binding.Targets[1], binding.Targets[0]
	second, err := binding.Identity()
	if err != nil || first != second {
		t.Fatalf("target order changed binding identity: %s %s %v", first, second, err)
	}
	binding.AuthorizationEpoch = 0
	if _, err := binding.Identity(); err == nil {
		t.Fatal("binding without authorization epoch was accepted")
	}
}

func TestReceiptsFailClosedUntilProducerEvidenceIsComplete(t *testing.T) {
	now := time.Now().UTC()
	publication := PublicationReceipt{CandidateID: "candidate-1", DestinationRevisionID: "destination-1", TargetID: "win", ArtifactDigest: "sha256:artifact", DestinationObject: "s3://bucket/key", Producer: "lpbs", ExternalReceipt: "upload-1", Outcome: ReceiptPublished, ObservedAt: now}
	if !publication.TrustedPublication() {
		t.Fatal("complete publication receipt was rejected")
	}
	publication.ExternalReceipt = ""
	if publication.TrustedPublication() {
		t.Fatal("publication without producer receipt was trusted")
	}
	update := ClientUpdateReceipt{CandidateID: "candidate-1", PredecessorRef: "installed:1.0.0", SuccessorDigest: "sha256:artifact", TargetID: "win", VerifiedVersion: "2.0.0", Outcome: ReceiptPublished, Producer: "scenario-to-desktop", ExternalReceipt: "update-1", ObservedAt: now}
	if update.TrustedUpdate() {
		t.Fatal("publication outcome was accepted as a verified client update")
	}
	update.Outcome = ReceiptVerified
	if !update.TrustedUpdate() {
		t.Fatal("complete verified client update was rejected")
	}
}

func TestCanonicalContractsRoundTripThroughGeneratedProto(t *testing.T) {
	candidate := candidateFixture()
	candidateMessage, err := candidate.Proto()
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := candidateFromProto(candidateMessage)
	if err != nil {
		t.Fatal(err)
	}
	originalID, err := candidate.Identity()
	if err != nil {
		t.Fatal(err)
	}
	roundTripID, err := roundTrip.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if originalID != roundTripID || candidateMessage.GetCandidateId() != originalID {
		t.Fatalf("candidate identity changed over proto round trip: %q, %q, %q", originalID, roundTripID, candidateMessage.GetCandidateId())
	}

	destination := DestinationRevision{Kind: "object-store", DestinationID: "staging", ConfigurationDigest: "sha256:config", Channel: "stable", ExpectedChannelRevision: "rev-4"}
	destinationMessage, err := destination.Proto()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := destinationRevisionFromProto(destinationMessage); err != nil {
		t.Fatal(err)
	}

	binding := ReviewBinding{CandidateID: originalID, DestinationRevisionID: destinationMessage.GetDestinationRevisionId(), Targets: []string{"win-x64", "mac-arm64"}, Channel: "stable", EvidenceSetDigest: "sha256:evidence", PolicyDigest: "sha256:policy", AuthorizationEpoch: 3}
	bindingMessage, err := binding.Proto()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reviewBindingFromProto(bindingMessage); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedProtoRejectsTamperedCanonicalIdentity(t *testing.T) {
	candidateMessage, err := candidateFixture().Proto()
	if err != nil {
		t.Fatal(err)
	}
	candidateMessage.CandidateId = "candidate-tampered"
	if _, err := candidateFromProto(candidateMessage); err == nil {
		t.Fatal("tampered candidate identity was accepted")
	}

	destinationMessage, err := (DestinationRevision{Kind: "vps", DestinationID: "staging", ConfigurationDigest: "sha256:config", Channel: "stable"}).Proto()
	if err != nil {
		t.Fatal(err)
	}
	destinationMessage.DestinationRevisionId = "destination-tampered"
	if _, err := destinationRevisionFromProto(destinationMessage); err == nil {
		t.Fatal("tampered destination identity was accepted")
	}
}

func TestTargetIdentityBridgesSharedTargetModelWithoutDroppingArchitecture(t *testing.T) {
	identity, err := TargetIdentityFromTarget(targetmodel.Target{ID: "bridge:mac-1", Platform: "darwin", OS: "macos", Architecture: "arm64"}, "dmg")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Architecture != "arm64" || identity.Format != "dmg" {
		t.Fatalf("target bridge lost release dimensions: %+v", identity)
	}
	projected := identity.ExecutionTarget()
	if projected.ID != identity.ID || projected.Architecture != identity.Architecture || projected.Platform != identity.Platform || projected.OS != identity.OS {
		t.Fatalf("shared target projection changed identity: %+v", projected)
	}
}
