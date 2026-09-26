package transport

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"deployment-manager/releases"

	"connectrpc.com/connect"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type releaseIdentityFake struct {
	candidate   *releases.CandidateRecord
	destination *releases.DestinationRevisionRecord
	review      *releases.ReviewRecord
	updates     []releases.ClientUpdateReceipt
}

func (f *releaseIdentityFake) RecordClientUpdateReceipt(_ context.Context, _ string, receipt releases.ClientUpdateReceipt) error {
	f.updates = append(f.updates, receipt)
	return nil
}

func (f *releaseIdentityFake) ListClientUpdateReceipts(context.Context, string) ([]releases.ClientUpdateReceipt, error) {
	return f.updates, nil
}

func (f *releaseIdentityFake) RegisterCandidate(context.Context, releases.Candidate) (*releases.CandidateRecord, error) {
	return f.candidate, nil
}
func (f *releaseIdentityFake) GetCandidate(context.Context, string) (*releases.CandidateRecord, error) {
	if f.candidate == nil {
		return nil, errors.New("candidate not found")
	}
	return f.candidate, nil
}
func (f *releaseIdentityFake) RegisterDestinationRevision(context.Context, releases.DestinationRevision) (*releases.DestinationRevisionRecord, error) {
	return f.destination, nil
}
func (f *releaseIdentityFake) GetDestinationRevision(context.Context, string) (*releases.DestinationRevisionRecord, error) {
	if f.destination == nil {
		return nil, errors.New("destination not found")
	}
	return f.destination, nil
}
func (f *releaseIdentityFake) RegisterReview(context.Context, string, releases.ReviewBinding, string, *time.Time) (*releases.ReviewRecord, error) {
	return f.review, nil
}
func (f *releaseIdentityFake) GetReview(context.Context, string) (*releases.ReviewRecord, error) {
	if f.review == nil {
		return nil, errors.New("review not found")
	}
	return f.review, nil
}

func TestTypedReleaseViewIncludesDurableIdentityObjects(t *testing.T) {
	candidate := releases.Candidate{
		SourceRevision: "commit-1", ProfileRevision: "profile-1", BuildInputs: map[string]string{"builder": "v1"},
		DependencyLockDigest: "lock-1", PolicyDigest: "policy-1",
		Artifacts: []releases.CandidateArtifact{{
			Target:       releases.TargetIdentity{ID: "linux-x64", Platform: "desktop", OS: "linux", Architecture: "amd64", Format: "AppImage"},
			ImmutableRef: "object://candidate/linux", Digest: "sha256:artifact", SizeBytes: 1,
			SignatureDigest: "sha256:signature", SignerRef: "authority://desktop",
		}},
	}
	candidateID, err := candidate.Identity()
	if err != nil {
		t.Fatal(err)
	}
	destination := releases.DestinationRevision{Kind: "object-store", DestinationID: "staging", ConfigurationDigest: "destination-1", Channel: "stable"}
	destinationID, err := destination.Identity()
	if err != nil {
		t.Fatal(err)
	}
	identity := &releaseIdentityFake{
		candidate:   &releases.CandidateRecord{ID: candidateID, Candidate: candidate},
		destination: &releases.DestinationRevisionRecord{ID: destinationID, Revision: destination},
		review:      &releases.ReviewRecord{ID: "review-1", Binding: releases.ReviewBinding{CandidateID: candidateID, DestinationRevisionID: destinationID, Targets: []string{"linux-x64"}, Channel: "stable", EvidenceSetDigest: "evidence", PolicyDigest: "policy", AuthorizationEpoch: 1}},
	}
	h := (&Handler{releaseList: func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"releases":[{"id":"r1","profile_id":"p1","git_commit_hash":"commit-1","candidate_id":"` + candidateID + `","destination_revision_id":"` + destinationID + `","readiness_review_key":"review-1","release_version":"1.0.0","channel":"stable","status":"pending"}]}`))
	}}).WithReleaseIdentityRepository(identity).WithReadAuthorization(func(context.Context) error { return nil })
	response, err := (releasesService{h}).List(context.Background(), connect.NewRequest(&releasesv1.ListReleasesRequest{ProfileId: "p1", Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	view := response.Msg.Releases[0]
	if view.GetCandidate() == nil || view.GetDestinationRevision() == nil || view.GetReviewBinding() == nil {
		t.Fatalf("typed release omitted durable identity objects: %s", view)
	}
}

func TestTypedReleaseServiceRecordsOwnerClientUpdateReceipt(t *testing.T) {
	identity := &releaseIdentityFake{}
	h := (&Handler{}).WithReleaseIdentityRepository(identity).WithClientUpdateReceiptAuthorization(func(context.Context) error { return nil })
	observedAt := time.Now().UTC().Truncate(time.Second)
	response, err := (releasesService{h}).RecordClientUpdateReceipt(context.Background(), connect.NewRequest(&releasesv1.RecordClientUpdateReceiptRequest{
		ReleaseId: "release-1",
		Receipt: &releasesv1.ClientUpdateReceipt{
			CandidateId: "candidate-1", PredecessorRef: "desktop-1.0", SuccessorDigest: "sha256:successor",
			TargetId: "linux-x64", VerifiedVersion: "2.0.0", Outcome: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_VERIFIED,
			Producer: "scenario-to-desktop", ExternalReceipt: "update-1", ObservedAt: timestamppb.New(observedAt),
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !response.Msg.GetAccepted() || response.Msg.GetReleaseId() != "release-1" {
		t.Fatalf("unexpected response: %v", response.Msg)
	}
	if len(identity.updates) != 1 || !identity.updates[0].TrustedUpdate() {
		t.Fatalf("receipt was not routed to the repository: %#v", identity.updates)
	}
}

func TestTypedReleaseOwnerMutationsFailClosedWithoutAuthorizationBoundary(t *testing.T) {
	candidate := releases.Candidate{
		SourceRevision: "commit-1", ProfileRevision: "profile-1", DependencyLockDigest: "lock-1", PolicyDigest: "policy-1",
		Artifacts: []releases.CandidateArtifact{{
			Target:       releases.TargetIdentity{ID: "linux-x64", Platform: "desktop", OS: "linux", Architecture: "amd64", Format: "AppImage"},
			ImmutableRef: "object://candidate/linux", Digest: "sha256:artifact", SizeBytes: 1,
			SignatureDigest: "sha256:signature", SignerRef: "authority://desktop",
		}},
	}
	candidateProto, err := candidate.Proto()
	if err != nil {
		t.Fatal(err)
	}
	identity := &releaseIdentityFake{candidate: &releases.CandidateRecord{ID: "candidate-1", Candidate: candidate}}
	h := (&Handler{}).WithReleaseIdentityRepository(identity)

	_, err = (releasesService{h}).RegisterCandidate(context.Background(), connect.NewRequest(&releasesv1.RegisterCandidateRequest{Candidate: candidateProto}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("candidate registration error code = %s, want unauthenticated: %v", connect.CodeOf(err), err)
	}

	_, err = (releasesService{h}).RecordClientUpdateReceipt(context.Background(), connect.NewRequest(&releasesv1.RecordClientUpdateReceiptRequest{
		ReleaseId: "release-1",
		Receipt: &releasesv1.ClientUpdateReceipt{
			CandidateId: "candidate-1", PredecessorRef: "desktop-1.0", SuccessorDigest: "sha256:successor",
			TargetId: "linux-x64", VerifiedVersion: "2.0.0", Outcome: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_VERIFIED,
			Producer: "scenario-to-desktop", ExternalReceipt: "update-1", ObservedAt: timestamppb.New(time.Now().UTC()),
		},
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("client receipt error code = %s, want unauthenticated: %v", connect.CodeOf(err), err)
	}
}

func TestTypedReleaseServiceRegistersCanonicalReleaseIdentities(t *testing.T) {
	candidate := releases.Candidate{
		SourceRevision: "commit-1", ProfileRevision: "profile-1", BuildInputs: map[string]string{"builder": "v1"},
		DependencyLockDigest: "lock-1", PolicyDigest: "policy-1",
		Artifacts: []releases.CandidateArtifact{{
			Target:       releases.TargetIdentity{ID: "linux-x64", Platform: "desktop", OS: "linux", Architecture: "amd64", Format: "AppImage"},
			ImmutableRef: "object://candidate/linux", Digest: "sha256:artifact", SizeBytes: 1,
			SignatureDigest: "sha256:signature", SignerRef: "authority://desktop",
		}},
	}
	candidateID, err := candidate.Identity()
	if err != nil {
		t.Fatal(err)
	}
	destination := releases.DestinationRevision{Kind: "object-store", DestinationID: "staging", ConfigurationDigest: "destination-1", Channel: "stable"}
	destinationID, err := destination.Identity()
	if err != nil {
		t.Fatal(err)
	}
	identity := &releaseIdentityFake{
		candidate:   &releases.CandidateRecord{ID: candidateID, Candidate: candidate},
		destination: &releases.DestinationRevisionRecord{ID: destinationID, Revision: destination},
	}
	authorized := false
	h := (&Handler{}).WithReleaseIdentityRepository(identity).WithReleasePreparationAuthorization(func(context.Context) error {
		authorized = true
		return nil
	})
	candidateProto, err := candidate.Proto()
	if err != nil {
		t.Fatal(err)
	}
	candidateProto.CandidateId = ""
	candidateResponse, err := (releasesService{h}).RegisterCandidate(context.Background(), connect.NewRequest(&releasesv1.RegisterCandidateRequest{Candidate: candidateProto}))
	if err != nil {
		t.Fatal(err)
	}
	if !authorized || candidateResponse.Msg.GetCandidate().GetCandidateId() != candidateID {
		t.Fatalf("candidate registration did not canonicalize and return the server identity: %v", candidateResponse.Msg)
	}

	destinationProto, err := destination.Proto()
	if err != nil {
		t.Fatal(err)
	}
	destinationProto.DestinationRevisionId = ""
	destinationResponse, err := (releasesService{h}).RegisterDestinationRevision(context.Background(), connect.NewRequest(&releasesv1.RegisterDestinationRevisionRequest{Revision: destinationProto}))
	if err != nil {
		t.Fatal(err)
	}
	if destinationResponse.Msg.GetDestination().GetDestinationRevisionId() != destinationID {
		t.Fatalf("destination registration did not canonicalize and return the server identity: %v", destinationResponse.Msg)
	}
}
