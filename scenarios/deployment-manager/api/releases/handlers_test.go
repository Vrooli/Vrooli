package releases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"deployment-manager/profiles"
	"github.com/vrooli/api-core/eventbus"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

func TestDeployRequestSnapshotRoundTripPreservesCloudInputs(t *testing.T) {
	request := DeployRequest{
		ProfileID: "profile-1", ReleaseID: "release-1", Channel: "stable",
		Platforms: []string{"linux-x64", "cloud"}, GitCommitHash: "commit-1",
		ArtifactDigest: "sha256:artifact", ReleaseVersion: "1.2.3", ReleaseNotes: "notes",
		CandidateID: "candidate-1", DestinationRevisionID: "destination-1", AuthorizationEpoch: 7,
		IdempotencyKey: "idem-1", ReadinessReviewKey: "review-1",
		CloudManifest:       json.RawMessage(`{"service":"demo","version":"1.2.3"}`),
		CloudDeploymentName: "demo-prod", CloudBundlePath: "/tmp/demo.tar.gz",
		CloudBundleSHA256: "bundle-sha", CloudBundleSizeBytes: 1234, CloudRunPreflight: true,
	}

	snapshot, err := snapshotForDeployRequest(request)
	if err != nil {
		t.Fatalf("snapshot request: %v", err)
	}
	restored, err := deployRequestFromSnapshot(snapshot, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("restore request: %v", err)
	}
	if restored.AuthorizationCheck == nil || restored.ReadinessReviewKey != request.ReadinessReviewKey {
		t.Fatal("process-local authorization or readiness binding was not restored")
	}
	restored.AuthorizationCheck = nil
	request.AuthorizationCheck = nil
	if !reflect.DeepEqual(restored, request) {
		t.Fatalf("restored request differs from original:\nrestored=%#v\noriginal=%#v", restored, request)
	}
}

func TestDeployRequestSnapshotRejectsMissingOrMalformedData(t *testing.T) {
	if _, err := deployRequestFromSnapshot(nil, nil); err == nil {
		t.Fatal("missing snapshot was accepted")
	}
	if _, err := deployRequestFromSnapshot(json.RawMessage(`{"release_id":`), nil); err == nil {
		t.Fatal("malformed snapshot was accepted")
	}
}

func TestReleaseMutationAuthorizationDistinguishesIdentityFromCapability(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil).WithAuthorization(func(ctx context.Context) error {
		if _, ok := identity.PrincipalFromContext(ctx); !ok {
			return errors.New("missing principal")
		}
		return errors.New("missing write capability")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/profile-1/releases/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "profile-1"})
	req = req.WithContext(identity.WithPrincipal(req.Context(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true}))
	rec := httptest.NewRecorder()
	h.Start(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("verified principal status = %d, want forbidden", rec.Code)
	}
}

func TestReleaseMutationsFailClosedWithoutAuthorization(t *testing.T) {
	handler := NewHandler(newMockRepo(), newMockLPBSConfig(), nil, &mockOrch{}, nil)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", bytes.NewBufferString(`{"git_commit_hash":"commit-1","release_version":"1.0.0"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unconfigured release authorization status=%d body=%s", response.Code, response.Body.String())
	}
}

// --- mocks ---

type mockRepo struct {
	mu           sync.Mutex
	store        map[string]*Release
	locked       map[string]bool
	insertErr    error
	updateErr    error
	recovery     []RecoveryReceipt
	publications []PublicationReceipt
	active       []*Operation
}

func newMockRepo() *mockRepo {
	return &mockRepo{store: map[string]*Release{}, locked: map[string]bool{}}
}

func (m *mockRepo) Insert(_ context.Context, rel *Release) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertErr != nil {
		return m.insertErr
	}
	m.store[rel.ID] = rel
	return nil
}

func (m *mockRepo) Get(_ context.Context, id string) (*Release, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rel, ok := m.store[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return rel, nil
}

func (m *mockRepo) GetByIdempotencyKey(_ context.Context, profileID, key string) (*Release, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rel := range m.store {
		if rel.ProfileID == profileID && rel.IdempotencyKey == key {
			return rel, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) InsertOperation(_ context.Context, _ *Operation) error { return nil }
func (m *mockRepo) GetOperation(_ context.Context, _ string) (*Operation, error) {
	return nil, errors.New("not found")
}
func (m *mockRepo) ListActiveOperations(_ context.Context) ([]*Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*Operation(nil), m.active...), nil
}
func (m *mockRepo) UpdateOperation(_ context.Context, id, status, stage, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, operation := range m.active {
		if operation != nil && operation.ID == id {
			operation.Status = status
			operation.ActiveStage = stage
			operation.Error = errMsg
		}
	}
	return nil
}

func (m *mockRepo) ListByProfile(_ context.Context, profileID string, _ int) ([]*Release, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Release
	for _, r := range m.store {
		if r.ProfileID == profileID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, id, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	if rel, ok := m.store[id]; ok {
		rel.Status = status
	}
	return nil
}

func (m *mockRepo) SetDeploymentID(_ context.Context, id, deploymentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rel, ok := m.store[id]; ok {
		rel.DeploymentID = deploymentID
	}
	return nil
}

func (m *mockRepo) SetVerificationEvidence(_ context.Context, id string, items []VerificationItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rel, ok := m.store[id]; ok {
		rel.VerificationEvidence = items
	}
	return nil
}

func (m *mockRepo) MarkPlatformPublished(_ context.Context, id, platform string, artifactID int64) error {
	return nil
}

func (m *mockRepo) MarkPlatformStatus(_ context.Context, id, platform, status, errMsg string) error {
	return nil
}

func (m *mockRepo) MarkSuperseded(_ context.Context, profileID, channel, exceptID string) error {
	return nil
}

func (m *mockRepo) AcquireProfileLock(_ context.Context, profileID string) (bool, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.locked[profileID] {
		return false, func() {}, nil
	}
	m.locked[profileID] = true
	return true, func() {
		m.mu.Lock()
		delete(m.locked, profileID)
		m.mu.Unlock()
	}, nil
}

func (m *mockRepo) RecordReadinessWaiver(_ context.Context, profileID, commit, reason, actor string) error {
	if profileID == "" || commit == "" || reason == "" || actor == "" {
		return errors.New("waiver fields required")
	}
	return nil
}

func (m *mockRepo) GetLatestReadiness(_ context.Context, _ string) (*ReadinessRecord, error) {
	return &ReadinessRecord{}, nil
}

func (m *mockRepo) RecordRecoveryReceipt(_ context.Context, receipt RecoveryReceipt) error {
	m.recovery = append(m.recovery, receipt)
	return nil
}

func (m *mockRepo) ListRecoveryReceipts(_ context.Context, _ string) ([]RecoveryReceipt, error) {
	return append([]RecoveryReceipt(nil), m.recovery...), nil
}

func (m *mockRepo) ListPublicationReceipts(_ context.Context, _ string) ([]PublicationReceipt, error) {
	return append([]PublicationReceipt(nil), m.publications...), nil
}

func (m *mockRepo) RecordPublicationReceipt(_ context.Context, _ string, receipt PublicationReceipt) error {
	m.publications = append(m.publications, receipt)
	return nil
}

type mockLPBSConfig struct {
	mu      sync.Mutex
	configs map[string]*profiles.LPBSReleaseConfig
	err     error
}

type alertPublisherFake struct {
	events []eventbus.DomainEvent
	err    error
}

func (p *alertPublisherFake) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	return p.err
}

type signingExpiryFake struct {
	expiresAt time.Time
}

func (s signingExpiryFake) LookupSigningExpiry(context.Context, string, string) (*SigningExpiry, error) {
	return &SigningExpiry{ExpiresAt: s.expiresAt}, nil
}

func newMockLPBSConfig() *mockLPBSConfig {
	return &mockLPBSConfig{configs: map[string]*profiles.LPBSReleaseConfig{}}
}

func (m *mockLPBSConfig) Get(_ context.Context, profileID string) (*profiles.LPBSReleaseConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	return m.configs[profileID], nil
}

func (m *mockLPBSConfig) Upsert(_ context.Context, cfg *profiles.LPBSReleaseConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[cfg.ProfileID] = cfg
	return nil
}

func (m *mockLPBSConfig) Delete(_ context.Context, profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.configs, profileID)
	return nil
}

func TestHealthEndpointUsesPublicationAndRecoveryLedgers(t *testing.T) {
	repo := newMockRepo()
	created := time.Now().UTC().Add(-2 * time.Minute)
	repo.store["release-health"] = &Release{
		ID: "release-health", Status: StatusPublished, CreatedAt: created,
		PublishedAt:          timePtr(created.Add(time.Minute)),
		Platforms:            []ReleasePlatform{{Platform: "linux", Status: PlatformStatusPublished}},
		VerificationEvidence: []VerificationItem{{Platform: "linux", Match: true, SHA512Match: true, CheckedAt: created.Add(time.Minute)}},
	}
	repo.publications = []PublicationReceipt{{CandidateID: "candidate-1", DestinationRevisionID: "destination-1", TargetID: "linux", ArtifactDigest: "sha512:artifact", DestinationObject: "lpbs://stable/linux/1", Producer: "lpbs", ExternalReceipt: "publish-1", Outcome: ReceiptVerified, ObservedAt: created.Add(time.Minute)}}
	handler := NewHandler(repo, newMockLPBSConfig(), nil, nil, nil).WithReadAuthorization(func(context.Context) error { return nil })
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/releases/release-health/health", nil), map[string]string{"release_id": "release-health"})
	response := httptest.NewRecorder()
	handler.Health(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
	var health ReleaseHealth
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if health.Status != "healthy" || !health.PublicationVerified {
		t.Fatalf("health=%+v", health)
	}
}

func TestCompatibilityReadRoutesFailClosedWithoutReadAuthorization(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-read"] = &Release{ID: "release-read", Status: StatusPublished}
	handler := NewHandler(repo, nil, nil, nil, nil)
	request := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/releases/release-read/health", nil), map[string]string{"release_id": "release-read"})
	response := httptest.NewRecorder()
	handler.Health(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("health status=%d body=%s, want unauthorized", response.Code, response.Body.String())
	}
}

type recoveryControlsFake struct {
	controls []string
}

func (recoveryControlsFake) Recover(context.Context, *RecoveryRequest, string, string) (*RecoveryReceipt, error) {
	return nil, errors.New("recovery is not used by this projection test")
}

func (f recoveryControlsFake) RecoveryControls(context.Context, string) ([]string, error) {
	return append([]string(nil), f.controls...), nil
}

func TestHealthProjectionUsesOwnerRecoveryControls(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-controls"] = &Release{ID: "release-controls", DestinationRevisionID: "destination-lpbs"}
	handler := NewHandler(repo, nil, nil, nil, nil).WithReadAuthorization(func(context.Context) error { return nil }).WithRecoveryExecutor(recoveryControlsFake{controls: []string{"halt", "withdraw", "rollback", "forward_repair"}})
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/releases/release-controls/health", nil), map[string]string{"release_id": "release-controls"})
	response := httptest.NewRecorder()
	handler.Health(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
	var health ReleaseHealth
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(health.SupportedControls, []string{"halt", "withdraw", "rollback", "forward_repair"}) || !reflect.DeepEqual(health.UnsupportedControls, []string{"cohort_rollout"}) {
		t.Fatalf("recovery controls = supported=%v unsupported=%v", health.SupportedControls, health.UnsupportedControls)
	}
}

type ownerObservationFake struct{}

func (ownerObservationFake) RunDeploy(context.Context, DeployRequest) (*DeployResult, error) {
	return nil, errors.New("deployment is not used by this reconciliation test")
}

func (ownerObservationFake) ObserveRelease(context.Context, *Release) (*OwnerObservation, error) {
	return &OwnerObservation{
		DeploymentID:          "cloud-deployment-1",
		TargetID:              "cloud:cloud-deployment-1",
		ExpectedReleaseDigest: "sha256:bundle",
		ObservedReleaseDigest: "sha256:bundle",
		Healthy:               true,
		ObservedAt:            time.Now().UTC(),
	}, nil
}

func TestReconcileRoutesCloudOwnerObservation(t *testing.T) {
	repo := newMockRepo()
	repo.store["cloud-release"] = &Release{ID: "cloud-release", Status: StatusAmbiguous, ProfileID: "p1", Channel: "stable", ReleaseVersion: "1.0.0", DeploymentID: "cloud-deployment-1", DestinationRevisionID: "destination-cloud"}
	identity := &identityRepoFake{destination: &DestinationRevisionRecord{ID: "destination-cloud", Revision: DestinationRevision{Kind: "cloud", DestinationID: "cloud-deployment-1", Channel: "stable"}}}
	handler := NewHandler(repo, nil, nil, ownerObservationFake{}, nil).WithAuthorization(func(context.Context) error { return nil }).WithIdentityRepository(identity)
	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/api/v1/releases/cloud-release/reconcile", nil), map[string]string{"release_id": "cloud-release"})
	response := httptest.NewRecorder()
	handler.Reconcile(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("reconcile status=%d body=%s", response.Code, response.Body.String())
	}
	got, err := repo.Get(context.Background(), "cloud-release")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusAmbiguous || len(got.VerificationEvidence) != 1 || !got.VerificationEvidence[0].Match || !got.VerificationEvidence[0].SHA512Match {
		t.Fatalf("cloud reconciliation standing = %#v", got)
	}
}

func TestDossierMakesMissingIdentityProofExplicit(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-dossier"] = &Release{
		ID: "release-dossier", Status: StatusPublished, CreatedAt: time.Now().UTC(),
	}
	handler := NewHandler(repo, newMockLPBSConfig(), nil, nil, nil).WithReadAuthorization(func(context.Context) error { return nil })
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/releases/release-dossier/dossier", nil), map[string]string{"release_id": "release-dossier"})
	response := httptest.NewRecorder()
	handler.Dossier(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("dossier status=%d body=%s", response.Code, response.Body.String())
	}
	var dossier ReleaseDossier
	if err := json.NewDecoder(response.Body).Decode(&dossier); err != nil {
		t.Fatal(err)
	}
	if len(dossier.MissingProof) != 4 || dossier.MissingProof[0] != "identity_repository" {
		t.Fatalf("missing proof=%v", dossier.MissingProof)
	}
}

func TestHealthEndpointPublishesCanonicalAlertsWithoutChangingHealthResponse(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-alert"] = &Release{
		ID: "release-alert", Status: StatusFailed, CreatedAt: time.Now().UTC(),
	}
	publisher := &alertPublisherFake{}
	handler := NewHandler(repo, newMockLPBSConfig(), nil, nil, nil).WithReadAuthorization(func(context.Context) error { return nil }).WithAlertPublisher(publisher)
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/releases/release-alert/health", nil), map[string]string{"release_id": "release-alert"})
	response := httptest.NewRecorder()
	handler.Health(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events=%d, want 1", len(publisher.events))
	}
	event := publisher.events[0]
	if event.EventType != "deployment-manager.release.alert.v1" || event.Source != "deployment-manager" {
		t.Fatalf("event identity=%+v", event)
	}
	if event.Payload["check_id"] != "failed_verification" || event.Payload["release_id"] != "release-alert" {
		t.Fatalf("event payload=%+v", event.Payload)
	}
	if event.Payload["message"] != "Release verification or publication failed. Next action: Inspect the failed stage and use an exact owner recovery action." {
		t.Fatalf("event message=%q", event.Payload["message"])
	}
}

func TestHealthEndpointDoesNotFailWhenAlertPublicationFails(t *testing.T) {
	repo := newMockRepo()
	repo.store["release-alert-error"] = &Release{
		ID: "release-alert-error", Status: StatusFailed, CreatedAt: time.Now().UTC(),
	}
	publisher := &alertPublisherFake{err: errors.New("events unavailable")}
	handler := NewHandler(repo, newMockLPBSConfig(), nil, nil, nil).WithReadAuthorization(func(context.Context) error { return nil }).WithAlertPublisher(publisher)
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/releases/release-alert-error/health", nil), map[string]string{"release_id": "release-alert-error"})
	response := httptest.NewRecorder()
	handler.Health(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHealthEndpointAlertsOnSigningExpiry(t *testing.T) {
	repo := newMockRepo()
	created := time.Now().UTC().Add(-2 * time.Minute)
	repo.store["release-signing-expiry"] = &Release{
		ID: "release-signing-expiry", ProfileID: "profile-1", Status: StatusPublished, CreatedAt: created,
		PublishedAt: timePtr(created.Add(time.Minute)),
		Platforms:   []ReleasePlatform{{Platform: "windows-x64", Status: PlatformStatusPublished}},
		VerificationEvidence: []VerificationItem{{
			Platform: "windows-x64", Match: true, SHA512Match: true, CheckedAt: created.Add(time.Minute),
		}},
	}
	repo.publications = []PublicationReceipt{{CandidateID: "candidate-1", DestinationRevisionID: "destination-1", TargetID: "windows-x64", ArtifactDigest: "sha512:artifact", DestinationObject: "lpbs://stable/windows-x64/1", Producer: "lpbs", ExternalReceipt: "publish-1", Outcome: ReceiptVerified, ObservedAt: created.Add(time.Minute)}}
	handler := NewHandler(repo, newMockLPBSConfig(), nil, nil, nil).WithReadAuthorization(func(context.Context) error { return nil }).WithSigningExpiryLookup(signingExpiryFake{expiresAt: time.Now().UTC().Add(5 * 24 * time.Hour)})
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/releases/release-signing-expiry/health", nil), map[string]string{"release_id": "release-signing-expiry"})
	response := httptest.NewRecorder()
	handler.Health(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
	var health ReleaseHealth
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, alert := range health.Alerts {
		if alert.Code == "signing_expiry" && alert.Target == "windows-x64" && alert.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Fatalf("signing expiry alert missing: %+v", health.Alerts)
	}
}

type mockOrch struct {
	called    int
	lastReq   DeployRequest
	result    *DeployResult
	err       error
	returnNil bool
}

func (m *mockOrch) RunDeploy(_ context.Context, req DeployRequest) (*DeployResult, error) {
	m.called++
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.returnNil {
		return nil, nil
	}
	if m.result != nil {
		return m.result, nil
	}
	return &DeployResult{Status: "complete"}, nil
}

func TestStartRefusesNilOrchestratorResultAsAmbiguous(t *testing.T) {
	repo := newMockRepo()
	orch := &mockOrch{returnNil: true}
	handler := NewHandler(repo, newMockLPBSConfig(), nil, orch, nil).WithAuthorization(func(context.Context) error { return nil })
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", bytes.NewBufferString(`{"git_commit_hash":"deadbeef","release_version":"1.0.0","platforms":["linux-x64"]}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s, want service unavailable", response.Code, response.Body.String())
	}
	for _, release := range repo.store {
		if release.Status != StatusAmbiguous {
			t.Fatalf("release status=%q, want ambiguous", release.Status)
		}
		return
	}
	t.Fatal("release was not persisted")
}

type mockVerifier struct {
	outcomes  map[string]*VerifyOutcome
	err       error
	returnNil bool
	requests  []*VerifyCall
}

func (m *mockVerifier) Verify(_ context.Context, req *VerifyCall) (*VerifyOutcome, error) {
	m.requests = append(m.requests, req)
	if m.err != nil {
		return nil, m.err
	}
	if m.returnNil {
		return nil, nil
	}
	key := req.Platform + ":" + req.ExpectedVersion
	if v, ok := m.outcomes[key]; ok {
		return v, nil
	}
	return &VerifyOutcome{
		Match: true, SHA512Match: true, Platform: req.Platform, Channel: req.Channel,
		ExpectedVersion: req.ExpectedVersion, ObservedVersion: req.ExpectedVersion,
	}, nil
}

func TestVerifyUsesDurablePublicationDigestForIdentityBoundRelease(t *testing.T) {
	repo := newMockRepo()
	_ = repo.Insert(context.Background(), &Release{
		ID: "identity-release", ProfileID: "p1", CandidateID: "candidate-1", DestinationRevisionID: "destination-1",
		Channel: "stable", ReleaseVersion: "1.0.0", Status: StatusPublished,
		Platforms: []ReleasePlatform{{Platform: "linux-x64"}},
	})
	repo.publications = []PublicationReceipt{{
		CandidateID: "candidate-1", DestinationRevisionID: "destination-1", TargetID: "linux-x64",
		ArtifactDigest: "sha512:published-bytes", DestinationObject: "lpbs://stable/linux-x64/1",
		Producer: "lpbs", ExternalReceipt: "upload-1", Outcome: ReceiptVerified, ObservedAt: time.Now().UTC(),
	}}
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "myapp"})
	verifier := &mockVerifier{}
	handler := NewHandler(repo, cfg, verifier, nil, nil).WithAuthorization(func(context.Context) error { return nil })
	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/api/v1/releases/identity-release/verify", nil), map[string]string{"release_id": "identity-release"})
	response := httptest.NewRecorder()
	handler.Verify(response, req)
	if response.Code != http.StatusOK || len(verifier.requests) != 1 || verifier.requests[0].ExpectedSHA512 != "published-bytes" {
		t.Fatalf("status=%d requests=%+v body=%s", response.Code, verifier.requests, response.Body.String())
	}
}

// --- helpers ---

func newServer(repo Repository, cfg profiles.LPBSReleaseConfigRepository, ver LPBSVerifier, orch Orchestrator) http.Handler {
	h := NewHandler(repo, cfg, ver, orch, nil).WithAuthorization(func(context.Context) error { return nil }).WithReadAuthorization(func(context.Context) error { return nil })
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/profiles/{id}/releases", h.ListByProfile).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{id}/releases/start", h.Start).Methods("POST")
	r.HandleFunc("/api/v1/releases/{release_id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/releases/{release_id}/verify", h.Verify).Methods("POST")
	r.HandleFunc("/api/v1/releases/{release_id}/reconcile", h.Reconcile).Methods("POST")
	return r
}

func TestStartRequiresExactApprovedReadinessIdentity(t *testing.T) {
	tests := []struct {
		name       string
		approval   *ReadinessApproval
		wantStatus int
		wantCalls  int
	}{
		{name: "artifact mismatch", approval: &ReadinessApproval{Key: "rr-1", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:other", Targets: []string{"linux-x64"}, Channel: "stable", Status: "approved", ApprovedAt: timePointer(time.Now())}, wantStatus: http.StatusPreconditionFailed},
		{name: "exact approved identity", approval: &ReadinessApproval{Key: "rr-1", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Targets: []string{"linux-x64"}, Channel: "stable", Status: "approved", ApprovedAt: timePointer(time.Now())}, wantStatus: http.StatusOK, wantCalls: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, cfg, orch := newMockRepo(), newMockLPBSConfig(), &mockOrch{}
			handler := NewHandler(repo, cfg, nil, orch, nil).WithAuthorization(func(context.Context) error { return nil }).WithReadinessLookup(func(context.Context, string) (*ReadinessApproval, error) { return tc.approval, nil })
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
			body := bytes.NewBufferString(`{"git_commit_hash":"abc","artifact_digest":"sha256:candidate","candidate_id":"candidate-1","destination_revision_id":"destination-1","authorization_epoch":1,"readiness_review_key":"rr-1","release_version":"1.0.0","channel":"stable","platforms":["linux-x64"]}`)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", body)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.wantStatus || orch.called != tc.wantCalls {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, orch.called, response.Body.String())
			}
		})
	}
}

func timePointer(value time.Time) *time.Time { return &value }

// --- tests ---

func TestStartRequiresCommitAndVersion(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	srv := httptest.NewServer(newServer(repo, cfg, nil, &mockOrch{}))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestStartHappyPath(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "myapp", DefaultChannel: "stable"})
	orch := &mockOrch{}
	srv := httptest.NewServer(newServer(repo, cfg, nil, orch))
	defer srv.Close()

	body := `{"git_commit_hash":"deadbeef","release_version":"1.0.0","platforms":["linux-x64"]}`
	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if orch.called != 1 {
		t.Errorf("expected orch called once, got %d", orch.called)
	}
	if orch.lastReq.Channel != "stable" {
		t.Errorf("expected channel=stable from default, got %q", orch.lastReq.Channel)
	}
	if orch.lastReq.GitCommitHash != "deadbeef" {
		t.Errorf("expected commit=deadbeef, got %q", orch.lastReq.GitCommitHash)
	}
	// Release inserted
	if len(repo.store) != 1 {
		t.Errorf("expected 1 release in store, got %d", len(repo.store))
	}
}

func TestStartFailsClosedWhenReleaseConfigurationCannotBeRead(t *testing.T) {
	repo := newMockRepo()
	cfg := &mockLPBSConfig{configs: map[string]*profiles.LPBSReleaseConfig{}, err: errors.New("database unavailable")}
	orch := &mockOrch{}
	srv := httptest.NewServer(newServer(repo, cfg, nil, orch))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(`{"git_commit_hash":"deadbeef","release_version":"1.0.0"}`)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable || orch.called != 0 || len(repo.store) != 0 {
		t.Fatalf("configuration failure status=%d orch_calls=%d releases=%d", resp.StatusCode, orch.called, len(repo.store))
	}
}

func TestStartPersistsVerifiedActorInsteadOfRequestLabel(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	handler := NewHandler(repo, cfg, nil, &mockOrch{}, nil).WithAuthorization(func(context.Context) error { return nil }).WithActorResolver(func(context.Context) (string, error) {
		return "verified-reviewer", nil
	})
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", bytes.NewBufferString(`{"git_commit_hash":"deadbeef","release_version":"1.0.0","released_by":"forged-request-label"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	for _, release := range repo.store {
		if release.ReleasedBy != "verified-reviewer" {
			t.Fatalf("released_by=%q, want verified-reviewer", release.ReleasedBy)
		}
		return
	}
	t.Fatal("release was not persisted")
}

func TestStartIdempotencyReplaysExistingRelease(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	orch := &mockOrch{}
	handler := NewHandler(repo, cfg, nil, orch, nil).WithAuthorization(func(context.Context) error { return nil })
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)
	body := []byte(`{"git_commit_hash":"deadbeef","release_version":"1.0.0","platforms":["linux-x64"],"idempotency_key":"release-request-1"}`)
	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", bytes.NewReader(body))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("attempt %d: status=%d body=%s", attempt, response.Code, response.Body.String())
		}
	}
	if orch.called != 1 {
		t.Fatalf("expected one orchestration for two identical requests, got %d", orch.called)
	}
}

func TestStartLockContentionReturns409(t *testing.T) {
	repo := newMockRepo()
	repo.locked["p1"] = true // pre-acquired by something else
	cfg := newMockLPBSConfig()
	srv := httptest.NewServer(newServer(repo, cfg, nil, &mockOrch{}))
	defer srv.Close()

	body := `{"git_commit_hash":"d","release_version":"1.0.0"}`
	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	var raw map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	if raw["error"] != "release_in_flight" {
		t.Errorf("expected release_in_flight error, got %v", raw)
	}
}

func TestDurableStartBlocksProfileWithActiveOperation(t *testing.T) {
	repo := newMockRepo()
	repo.active = []*Operation{{ID: "op-1", ReleaseID: "release-1", ProfileID: "p1", Status: OperationRunning}}
	handler := NewHandler(repo, newMockLPBSConfig(), nil, &mockOrch{}, nil).WithAuthorization(func(context.Context) error { return nil }).WithDurableOperations(true)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/profiles/{id}/releases/start", handler.Start).Methods(http.MethodPost)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/p1/releases/start", bytes.NewBufferString(`{"git_commit_hash":"deadbeef","release_version":"1.0.0"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s, want conflict", response.Code, response.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["operation_id"] != "op-1" || body["release_id"] != "release-1" {
		t.Fatalf("active operation identity = %#v", body)
	}
	if len(repo.store) != 0 {
		t.Fatalf("blocked start inserted releases: %#v", repo.store)
	}
}

func TestResumeOperationsDoesNotRetryAmbiguousOwnerEffect(t *testing.T) {
	repo := newMockRepo()
	repo.active = []*Operation{
		{ID: "op-ambiguous", ReleaseID: "release-1", ProfileID: "p1", Status: OperationAmbiguous},
		{ID: "op-running", ReleaseID: "release-2", ProfileID: "p2", Status: OperationRunning},
		{ID: "op-corrupt", ReleaseID: "release-3", ProfileID: "p3", Status: OperationQueued, RequestSnapshot: json.RawMessage(`{"release_id":`)},
	}
	repo.store["release-2"] = &Release{ID: "release-2", Status: StatusPublishing}
	repo.store["release-3"] = &Release{ID: "release-3", Status: StatusPending}
	orch := &mockOrch{}
	handler := NewHandler(repo, newMockLPBSConfig(), nil, orch, nil).WithDurableOperations(true)
	if err := handler.ResumeOperations(context.Background()); err != nil {
		t.Fatalf("ResumeOperations() error = %v", err)
	}
	if orch.called != 0 {
		t.Fatalf("uncertain operation was retried %d time(s)", orch.called)
	}
	if repo.active[1].Status != OperationAmbiguous || repo.active[1].ActiveStage != "reconcile" {
		t.Fatalf("running operation standing = %#v, want ambiguous/reconcile", repo.active[1])
	}
	if repo.active[2].Status != OperationAmbiguous || repo.store["release-3"].Status != StatusAmbiguous {
		t.Fatalf("corrupt snapshot standing = operation=%#v release=%#v, want both ambiguous", repo.active[2], repo.store["release-3"])
	}
}

func TestStartUsesExplicitChannel(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k", DefaultChannel: "stable"})
	orch := &mockOrch{}
	srv := httptest.NewServer(newServer(repo, cfg, nil, orch))
	defer srv.Close()

	body := `{"git_commit_hash":"d","release_version":"1.0.0","channel":"beta"}`
	resp, err := http.Post(srv.URL+"/api/v1/profiles/p1/releases/start", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if orch.lastReq.Channel != "beta" {
		t.Errorf("expected channel=beta override, got %q", orch.lastReq.Channel)
	}
}

func TestVerifyMarksFailedOnMismatch(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "myapp"})
	rel := &Release{
		ID: "r1", ProfileID: "p1", Channel: "stable", ReleaseVersion: "1.0.0", Status: StatusPublished,
		Platforms: []ReleasePlatform{{Platform: "linux-x64"}, {Platform: "darwin-arm64"}},
	}
	_ = repo.Insert(context.Background(), rel)

	ver := &mockVerifier{outcomes: map[string]*VerifyOutcome{
		"linux-x64:1.0.0":    {Match: true, SHA512Match: true, ObservedVersion: "1.0.0"},
		"darwin-arm64:1.0.0": {Match: false, ObservedVersion: "0.9.9"},
	}}
	srv := httptest.NewServer(newServer(repo, cfg, ver, nil))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/releases/r1/verify", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (verify succeeds even when outcome failed), got %d", resp.StatusCode)
	}
	got, _ := repo.Get(context.Background(), "r1")
	if got.Status != StatusVerifyFailed {
		t.Errorf("expected status=verify_failed, got %q", got.Status)
	}
	if len(got.VerificationEvidence) != 2 {
		t.Errorf("expected 2 evidence items, got %d", len(got.VerificationEvidence))
	}
}

func TestVerifyRefusesSuccessWhenStatusPersistenceFails(t *testing.T) {
	repo := newMockRepo()
	repo.updateErr = errors.New("database unavailable")
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "myapp"})
	_ = repo.Insert(context.Background(), &Release{ID: "r-status-error", ProfileID: "p1", Channel: "stable", ReleaseVersion: "1.0.0", Status: StatusPublished, Platforms: []ReleasePlatform{{Platform: "linux-x64"}}})
	srv := httptest.NewServer(newServer(repo, cfg, &mockVerifier{}, nil))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/releases/r-status-error/verify", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected service unavailable when status persistence fails, got %d", resp.StatusCode)
	}
}

func TestVerifyMarksFailedWhenVerifierReturnsNil(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "myapp"})
	_ = repo.Insert(context.Background(), &Release{ID: "r-nil-result", ProfileID: "p1", Channel: "stable", ReleaseVersion: "1.0.0", Status: StatusPublished, Platforms: []ReleasePlatform{{Platform: "linux-x64"}}})
	ver := &mockVerifier{returnNil: true}
	srv := httptest.NewServer(newServer(repo, cfg, ver, nil))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/releases/r-nil-result/verify", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want recorded failed verification", resp.StatusCode)
	}
	got, _ := repo.Get(context.Background(), "r-nil-result")
	if got.Status != StatusVerifyFailed || len(got.VerificationEvidence) != 1 || got.VerificationEvidence[0].Error == "" {
		t.Fatalf("nil verifier result standing = %#v", got)
	}
}

func TestReconcilePreservesAmbiguousStandingWhenOwnerObservationMatches(t *testing.T) {
	repo := newMockRepo()
	_ = repo.Insert(context.Background(), &Release{
		ID: "ambiguous-release", ProfileID: "p1", Channel: "stable", ReleaseVersion: "1.0.0", Status: StatusAmbiguous,
		Platforms: []ReleasePlatform{{Platform: "linux-x64"}},
	})
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "myapp"})
	srv := httptest.NewServer(newServer(repo, cfg, &mockVerifier{}, nil))
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/v1/releases/ambiguous-release/reconcile", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reconcile status=%d", resp.StatusCode)
	}
	got, _ := repo.Get(context.Background(), "ambiguous-release")
	if got.Status != StatusAmbiguous {
		t.Fatalf("reconcile cleared ambiguous standing: %q", got.Status)
	}
	if len(got.VerificationEvidence) != 1 || !got.VerificationEvidence[0].Match {
		t.Fatalf("reconcile evidence=%+v", got.VerificationEvidence)
	}
}

func TestListByProfileEmpty(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	srv := httptest.NewServer(newServer(repo, cfg, nil, nil))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/profiles/p1/releases")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var raw map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	if releases, ok := raw["releases"]; ok && releases != nil {
		// An empty repo lists nil slice; ok if the field is missing or null.
		if list, isList := releases.([]interface{}); isList && len(list) > 0 {
			t.Errorf("expected empty list, got %v", releases)
		}
	}
}

func TestVerifyReturns412WithoutAppKey(t *testing.T) {
	repo := newMockRepo()
	cfg := newMockLPBSConfig()
	_ = cfg.Upsert(context.Background(), &profiles.LPBSReleaseConfig{ProfileID: "p1"}) // missing app key
	rel := &Release{ID: "r1", ProfileID: "p1", Channel: "stable", ReleaseVersion: "1"}
	_ = repo.Insert(context.Background(), rel)

	ver := &mockVerifier{}
	srv := httptest.NewServer(newServer(repo, cfg, ver, nil))
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/v1/releases/r1/verify", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d", resp.StatusCode)
	}
}

func TestNewReleaseIDIsHex32(t *testing.T) {
	id := newReleaseID()
	if len(id) != 32 {
		t.Errorf("expected 32-char hex id, got %q (len=%d)", id, len(id))
	}
	for _, ch := range id {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("non-hex char in id: %c", ch)
		}
	}
}

// Compile-time check that mockRepo satisfies Repository.
var _ Repository = (*mockRepo)(nil)

// Sentinel to prevent unused-import errors in case fmt drops out.
var _ = fmt.Sprintf
