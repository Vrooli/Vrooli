package releases

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	apiidentity "github.com/vrooli/api-core/identity"
)

type identityRepoFake struct {
	candidate   *CandidateRecord
	destination *DestinationRevisionRecord
	review      *ReviewRecord
}

func (f *identityRepoFake) RegisterCandidate(context.Context, Candidate) (*CandidateRecord, error) {
	return f.candidate, nil
}
func (f *identityRepoFake) GetCandidate(context.Context, string) (*CandidateRecord, error) {
	if f.candidate == nil {
		return nil, errors.New("candidate not found")
	}
	return f.candidate, nil
}
func (f *identityRepoFake) RegisterDestinationRevision(context.Context, DestinationRevision) (*DestinationRevisionRecord, error) {
	return f.destination, nil
}
func (f *identityRepoFake) GetDestinationRevision(context.Context, string) (*DestinationRevisionRecord, error) {
	if f.destination == nil {
		return nil, errors.New("destination not found")
	}
	return f.destination, nil
}
func (f *identityRepoFake) RegisterReview(context.Context, string, ReviewBinding, string, *time.Time) (*ReviewRecord, error) {
	return f.review, nil
}
func (f *identityRepoFake) GetReview(context.Context, string) (*ReviewRecord, error) {
	if f.review == nil {
		return nil, errors.New("review not found")
	}
	return f.review, nil
}

func TestStartRequiresDurableIdentityBindingBeforeOrchestration(t *testing.T) {
	candidate := Candidate{
		SourceRevision:       "commit-1",
		ProfileRevision:      "profile-1",
		BuildInputs:          map[string]string{"builder": "v1"},
		DependencyLockDigest: "lock-1",
		PolicyDigest:         "policy-1",
		Artifacts: []CandidateArtifact{{
			Target:          TargetIdentity{ID: "linux-x64", Platform: "desktop", OS: "linux", Architecture: "amd64", Format: "AppImage"},
			ImmutableRef:    "object://candidate/linux",
			Digest:          "sha256:artifact",
			SizeBytes:       1,
			SignatureDigest: "sha256:signature",
			SignerRef:       "authority://desktop",
		}},
	}
	candidateID, err := candidate.Identity()
	if err != nil {
		t.Fatal(err)
	}
	manifestDigest, err := candidate.ArtifactManifestDigest()
	if err != nil {
		t.Fatal(err)
	}
	destination := DestinationRevision{Kind: "object-store", DestinationID: "staging", ConfigurationDigest: "destination-1", Channel: "stable"}
	destinationID, err := destination.Identity()
	if err != nil {
		t.Fatal(err)
	}
	approvedAt := time.Now().UTC()
	identity := &identityRepoFake{
		candidate:   &CandidateRecord{ID: candidateID, Candidate: candidate, ArtifactManifestDigest: manifestDigest},
		destination: &DestinationRevisionRecord{ID: destinationID, Revision: destination},
		review: &ReviewRecord{
			ID:      "review-1",
			Binding: ReviewBinding{CandidateID: candidateID, DestinationRevisionID: destinationID, Targets: []string{"linux-x64"}, Channel: "stable", EvidenceSetDigest: "evidence-1", PolicyDigest: "policy-1", AuthorizationEpoch: 1},
			Status:  "approved", ApprovedAt: &approvedAt,
		},
	}
	approval := &ReadinessApproval{Key: "review-1", ProfileID: "p1", CandidateCommit: "commit-1", ArtifactDigest: manifestDigest, CandidateID: candidateID, DestinationRevisionID: destinationID, AuthorizationEpoch: 1, EvidenceSetDigest: "evidence-1", PolicyDigest: "policy-1", Targets: []string{"linux-x64"}, Channel: "stable", Status: "approved", ApprovedAt: &approvedAt}
	orch := &mockOrch{}
	handler := NewHandler(newMockRepo(), newMockLPBSConfig(), nil, orch, nil).
		WithAuthorization(func(context.Context) error { return nil }).
		WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) { return approval, nil }).
		WithIdentityRepository(identity)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", strings.NewReader(`{"git_commit_hash":"commit-1","artifact_digest":"`+manifestDigest+`","candidate_id":"`+candidateID+`","destination_revision_id":"`+destinationID+`","authorization_epoch":1,"readiness_review_key":"review-1","release_version":"1.0.0","channel":"stable","platforms":["linux-x64"]}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || orch.called != 1 {
		t.Fatalf("valid durable binding status=%d calls=%d body=%s", response.Code, orch.called, response.Body.String())
	}

	identity.review.Binding.EvidenceSetDigest = "evidence-2"
	request = httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", strings.NewReader(`{"git_commit_hash":"commit-1","artifact_digest":"`+manifestDigest+`","candidate_id":"`+candidateID+`","destination_revision_id":"`+destinationID+`","authorization_epoch":1,"readiness_review_key":"review-1","release_version":"1.0.1","channel":"stable","platforms":["linux-x64"]}`))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusPreconditionFailed || orch.called != 1 {
		t.Fatalf("stale evidence binding status=%d calls=%d body=%s", response.Code, orch.called, response.Body.String())
	}

	identity.candidate = nil
	request = httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", strings.NewReader(`{"git_commit_hash":"commit-1","artifact_digest":"`+manifestDigest+`","candidate_id":"missing","destination_revision_id":"`+destinationID+`","authorization_epoch":1,"readiness_review_key":"review-1","release_version":"1.0.1","channel":"stable","platforms":["linux-x64"]}`))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusPreconditionFailed || orch.called != 1 {
		t.Fatalf("unregistered candidate status=%d calls=%d body=%s", response.Code, orch.called, response.Body.String())
	}
}

type recoveryExecutorFake struct{}

func (recoveryExecutorFake) Recover(_ context.Context, request *RecoveryRequest, deploymentID, _ string) (*RecoveryReceipt, error) {
	receipt := &RecoveryReceipt{ReceiptID: "rr-1", ReleaseID: request.ReleaseID, CandidateID: request.CandidateID, DestinationRevisionID: request.DestinationRevisionID, DeploymentID: deploymentID, Action: request.Action, Outcome: "halted", Health: "stopped", ExternalReceipt: "cloud-recovery-1", ObservedAt: time.Now().UTC()}
	if request.DryRun {
		receipt.Outcome, receipt.Health, receipt.DryRun = "preview", "unknown", true
	}
	return receipt, nil
}

type countingRecoveryExecutor struct {
	calls int
}

func (f *countingRecoveryExecutor) Recover(_ context.Context, request *RecoveryRequest, deploymentID, _ string) (*RecoveryReceipt, error) {
	f.calls++
	return &RecoveryReceipt{ReceiptID: "rr-counted", ReleaseID: request.ReleaseID, CandidateID: request.CandidateID, DestinationRevisionID: request.DestinationRevisionID, DeploymentID: deploymentID, Action: request.Action, Outcome: "halted", Health: "stopped", ExternalReceipt: "cloud-recovery-counted", ObservedAt: time.Now().UTC()}, nil
}

func TestRecoveryRequiresExactIdentityAndPersistsOwnerReceipt(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-1"] = &Release{ID: "release-1", CandidateID: "candidate-1", DestinationRevisionID: "destination-1", ReadinessReviewKey: "review-1", DeploymentID: "deployment-1", ArtifactDigest: "sha256:bundle"}
	repo.active = []*Operation{{ID: operationIDForRelease("release-1"), ReleaseID: "release-1", ProfileID: "p1", Status: OperationAmbiguous}}
	identity := &identityRepoFake{destination: &DestinationRevisionRecord{ID: "destination-1", Revision: DestinationRevision{Kind: "cloud", DestinationID: "deployment-1", Channel: "stable"}}, review: &ReviewRecord{ID: "review-1", Status: "approved", Binding: ReviewBinding{CandidateID: "candidate-1", DestinationRevisionID: "destination-1"}}}
	handler := NewHandler(repo, nil, nil, nil, nil).WithRecoveryAuthorization(func(context.Context) error { return nil }).WithIdentityRepository(identity).WithRecoveryExecutor(recoveryExecutorFake{}).WithDurableOperations(true)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/releases/{release_id}/recover", handler.Recover).Methods(http.MethodPost)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/releases/release-1/recover", strings.NewReader(`{"review_key":"review-1","candidate_id":"candidate-1","destination_revision_id":"destination-1","action":"halt","confirmation":"halt release-1"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(repo.recovery) != 1 || !repo.recovery[0].Valid() {
		t.Fatalf("recovery status=%d receipts=%#v body=%s", response.Code, repo.recovery, response.Body.String())
	}
	if repo.active[0].Status != OperationComplete || repo.active[0].ActiveStage != "recover" {
		t.Fatalf("recovery operation = %#v, want complete/recover", repo.active[0])
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/releases/release-1/recover", strings.NewReader(`{"review_key":"review-1","candidate_id":"wrong","destination_revision_id":"destination-1","action":"halt","confirmation":"halt release-1"}`))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusPreconditionFailed || len(repo.recovery) != 1 {
		t.Fatalf("mismatched recovery status=%d receipts=%#v body=%s", response.Code, repo.recovery, response.Body.String())
	}
}

func TestLPBSRecoveryDerivesOwnerIdentityFromDestinationRevision(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-lpbs"] = &Release{
		ID:                    "release-lpbs",
		CandidateID:           "candidate-lpbs",
		DestinationRevisionID: "destination-lpbs",
		ReadinessReviewKey:    "review-lpbs",
	}
	identity := &identityRepoFake{
		destination: &DestinationRevisionRecord{ID: "destination-lpbs", Revision: DestinationRevision{Kind: "lpbs", DestinationID: "lpbs-staging", Channel: "stable"}},
		review:      &ReviewRecord{ID: "review-lpbs", Status: "approved", Binding: ReviewBinding{CandidateID: "candidate-lpbs", DestinationRevisionID: "destination-lpbs"}},
	}
	handler := NewHandler(repo, nil, nil, nil, nil).
		WithRecoveryAuthorization(func(context.Context) error { return nil }).
		WithIdentityRepository(identity).
		WithRecoveryExecutor(recoveryExecutorFake{})
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/releases/{release_id}/recover", handler.Recover).Methods(http.MethodPost)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/releases/release-lpbs/recover", strings.NewReader(`{"review_key":"review-lpbs","candidate_id":"candidate-lpbs","destination_revision_id":"destination-lpbs","action":"halt","confirmation":"halt release-lpbs"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(repo.recovery) != 1 || repo.recovery[0].DeploymentID != "lpbs-staging" {
		t.Fatalf("LPBS recovery status=%d receipts=%#v body=%s", response.Code, repo.recovery, response.Body.String())
	}
}

func TestCloudRecoveryRequiresDurableBundleIdentityBeforeRepair(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-cloud-repair"] = &Release{ID: "release-cloud-repair", CandidateID: "candidate-cloud", DestinationRevisionID: "destination-cloud", ReadinessReviewKey: "review-cloud", DeploymentID: "deployment-cloud"}
	identity := &identityRepoFake{
		destination: &DestinationRevisionRecord{ID: "destination-cloud", Revision: DestinationRevision{Kind: "cloud", DestinationID: "deployment-cloud", Channel: "stable"}},
		review:      &ReviewRecord{ID: "review-cloud", Status: "approved", Binding: ReviewBinding{CandidateID: "candidate-cloud", DestinationRevisionID: "destination-cloud"}},
	}
	executor := &countingRecoveryExecutor{}
	handler := NewHandler(repo, nil, nil, nil, nil).
		WithRecoveryAuthorization(func(context.Context) error { return nil }).
		WithIdentityRepository(identity).
		WithRecoveryExecutor(executor)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/releases/{release_id}/recover", handler.Recover).Methods(http.MethodPost)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/releases/release-cloud-repair/recover", strings.NewReader(`{"review_key":"review-cloud","candidate_id":"candidate-cloud","destination_revision_id":"destination-cloud","action":"rollback","confirmation":"rollback release-cloud-repair"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusPreconditionFailed || executor.calls != 0 {
		t.Fatalf("cloud repair status=%d owner calls=%d body=%s", response.Code, executor.calls, response.Body.String())
	}
}

func TestRecoveryReauthorizesImmediatelyBeforeOwnerCall(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-reauth"] = &Release{ID: "release-reauth", CandidateID: "candidate-1", DestinationRevisionID: "destination-1", ReadinessReviewKey: "review-reauth", DeploymentID: "deployment-1", ArtifactDigest: "sha256:bundle"}
	identity := &identityRepoFake{destination: &DestinationRevisionRecord{ID: "destination-1", Revision: DestinationRevision{Kind: "cloud", DestinationID: "deployment-1", Channel: "stable"}}, review: &ReviewRecord{ID: "review-reauth", Status: "approved", Binding: ReviewBinding{CandidateID: "candidate-1", DestinationRevisionID: "destination-1"}}}
	executor := &countingRecoveryExecutor{}
	authorizationCalls := 0
	handler := NewHandler(repo, nil, nil, nil, nil).
		WithRecoveryAuthorization(func(context.Context) error {
			authorizationCalls++
			if authorizationCalls > 1 {
				return errors.New("approval revoked")
			}
			return nil
		}).WithIdentityRepository(identity).WithRecoveryExecutor(executor)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/releases/{release_id}/recover", handler.Recover).Methods(http.MethodPost)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/releases/release-reauth/recover", strings.NewReader(`{"review_key":"review-reauth","candidate_id":"candidate-1","destination_revision_id":"destination-1","action":"halt","confirmation":"halt release-reauth"}`))
	request = request.WithContext(apiidentity.WithPrincipal(request.Context(), apiidentity.Principal{Kind: apiidentity.ActorHuman, Subject: "operator", Verified: true}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("revoked recovery status=%d body=%s", response.Code, response.Body.String())
	}
	if authorizationCalls != 2 || executor.calls != 0 || len(repo.recovery) != 0 {
		t.Fatalf("reauthorization calls=%d owner calls=%d receipts=%d", authorizationCalls, executor.calls, len(repo.recovery))
	}
}
