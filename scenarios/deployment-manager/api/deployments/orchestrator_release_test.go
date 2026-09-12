package deployments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"deployment-manager/profiles"
	"deployment-manager/releases"
)

type candidateIdentityFake struct {
	candidate   *releases.CandidateRecord
	destination *releases.DestinationRevisionRecord
}

func (f *candidateIdentityFake) RegisterCandidate(context.Context, releases.Candidate) (*releases.CandidateRecord, error) {
	return f.candidate, nil
}

func (f *candidateIdentityFake) GetCandidate(context.Context, string) (*releases.CandidateRecord, error) {
	if f.candidate == nil {
		return nil, errors.New("candidate not found")
	}
	return f.candidate, nil
}

func (f *candidateIdentityFake) RegisterDestinationRevision(context.Context, releases.DestinationRevision) (*releases.DestinationRevisionRecord, error) {
	return nil, errors.New("not used")
}

func (f *candidateIdentityFake) GetDestinationRevision(context.Context, string) (*releases.DestinationRevisionRecord, error) {
	if f.destination != nil {
		return f.destination, nil
	}
	return nil, errors.New("not used")
}

func (f *candidateIdentityFake) RegisterReview(context.Context, string, releases.ReviewBinding, string, *time.Time) (*releases.ReviewRecord, error) {
	return nil, errors.New("not used")
}

func (f *candidateIdentityFake) GetReview(context.Context, string) (*releases.ReviewRecord, error) {
	return nil, errors.New("not used")
}

// --- fakes for the new clients ---

type fakeCloudClient struct {
	healthy  bool
	details  string
	reason   string
	err      error
	calls    int
	requests []DeploymentHealthRequest
}

func (f *fakeCloudClient) CheckDeploymentHealth(_ context.Context, request DeploymentHealthRequest) (*CloudHealthResult, error) {
	f.calls++
	f.requests = append(f.requests, request)
	if f.err != nil {
		return nil, f.err
	}
	return &CloudHealthResult{Healthy: f.healthy, ReasonCode: f.reason, Details: f.details, DeploymentID: request.DeploymentID, ObservedAt: time.Now().UTC()}, nil
}

type fakeCloudDeploymentClient struct {
	fakeCloudClient
	receipt     *CloudDeploymentReceipt
	deployErr   error
	deployCalls int
}

func (f *fakeCloudDeploymentClient) DeployCloud(_ context.Context, _ *CloudDeploymentRequest) (*CloudDeploymentReceipt, error) {
	f.deployCalls++
	if f.deployErr != nil {
		return nil, f.deployErr
	}
	return f.receipt, nil
}

type fakeLPBSClient struct {
	readiness        *LPBSReadinessResult
	readinessErr     error
	verifyOutcomes   map[string]*LPBSVerifyResult
	verifyErr        error
	readinessCalls   int
	verifyCallsCount int
	verifyRequests   []*LPBSVerifyRequest
	haltReceipt      *LPBSRecoveryReceipt
	haltRequest      *LPBSRecoveryRequest
	recoveryReceipt  *LPBSRecoveryReceipt
	recoveryRequest  *LPBSRecoveryRequest
}

type fakePublishPipelineRunner struct {
	status *PipelineStatus
	calls  int
}

func (f *fakePublishPipelineRunner) RunPublishPipelineConnect(context.Context, *PublishPipelineRequest) (*PublishPipelineResponse, error) {
	f.calls++
	return &PublishPipelineResponse{PipelineID: "pipeline-1"}, nil
}

func (f *fakePublishPipelineRunner) WaitForPipelineConnect(context.Context, string) (*PipelineStatus, error) {
	return f.status, nil
}

type failingPublishedVersionRepo struct{}

func (failingPublishedVersionRepo) RecordPublish(context.Context, *PublishedVersion) error {
	return errors.New("published-version ledger unavailable")
}

func (failingPublishedVersionRepo) GetLatestByProfile(context.Context, string) ([]PublishedVersion, error) {
	return nil, nil
}

func (failingPublishedVersionRepo) GetHistory(context.Context, string, string, int) ([]PublishedVersion, error) {
	return nil, nil
}

func (f *fakeLPBSClient) HaltChannel(_ context.Context, req *LPBSRecoveryRequest) (*LPBSRecoveryReceipt, error) {
	f.haltRequest = req
	return f.haltReceipt, nil
}

func (f *fakeLPBSClient) RecoverChannel(_ context.Context, req *LPBSRecoveryRequest) (*LPBSRecoveryReceipt, error) {
	f.recoveryRequest = req
	if f.recoveryReceipt != nil {
		f.recoveryReceipt.CandidateID = req.CandidateID
		f.recoveryReceipt.DestinationRevisionID = req.DestinationRevisionID
	}
	return f.recoveryReceipt, nil
}

func (f *fakeLPBSClient) CheckDeployReadiness(_ context.Context, _ *LPBSReadinessRequest) (*LPBSReadinessResult, error) {
	f.readinessCalls++
	if f.readinessErr != nil {
		return nil, f.readinessErr
	}
	if f.readiness == nil {
		return &LPBSReadinessResult{Ready: true}, nil
	}
	return f.readiness, nil
}

func (f *fakeLPBSClient) Verify(_ context.Context, req *LPBSVerifyRequest) (*LPBSVerifyResult, error) {
	f.verifyCallsCount++
	f.verifyRequests = append(f.verifyRequests, req)
	if f.verifyErr != nil {
		return nil, f.verifyErr
	}
	if r, ok := f.verifyOutcomes[req.Platform]; ok {
		return r, nil
	}
	return &LPBSVerifyResult{Match: true, SHA512Match: true, ObservedVersion: req.ExpectedVersion}, nil
}

type fakeReleasesRepo struct {
	statuses     map[string]string
	evidence     map[string][]releases.VerificationItem
	receipts     []releases.PublicationReceipt
	deployments  map[string]string
	supersedeArg string
	statusErr    error
	receiptErr   error
	supersedeErr error
}

func newFakeReleasesRepo() *fakeReleasesRepo {
	return &fakeReleasesRepo{statuses: map[string]string{}, evidence: map[string][]releases.VerificationItem{}, deployments: map[string]string{}}
}

func (f *fakeReleasesRepo) Insert(_ context.Context, rel *releases.Release) error {
	f.statuses[rel.ID] = rel.Status
	return nil
}

func (f *fakeReleasesRepo) Get(_ context.Context, _ string) (*releases.Release, error) {
	return nil, errors.New("not used")
}

func (f *fakeReleasesRepo) GetByIdempotencyKey(_ context.Context, _, _ string) (*releases.Release, error) {
	return nil, nil
}

func (f *fakeReleasesRepo) InsertOperation(_ context.Context, _ *releases.Operation) error {
	return nil
}

func (f *fakeReleasesRepo) GetOperation(_ context.Context, _ string) (*releases.Operation, error) {
	return nil, errors.New("not used")
}

func (f *fakeReleasesRepo) ListActiveOperations(_ context.Context) ([]*releases.Operation, error) {
	return nil, nil
}
func (f *fakeReleasesRepo) UpdateOperation(_ context.Context, _, _, _, _ string) error { return nil }

func (f *fakeReleasesRepo) ListByProfile(_ context.Context, _ string, _ int) ([]*releases.Release, error) {
	return nil, nil
}

func (f *fakeReleasesRepo) GetLatestReadiness(_ context.Context, _ string) (*releases.ReadinessRecord, error) {
	return nil, nil
}

func (f *fakeReleasesRepo) RecordReadinessWaiver(_ context.Context, _, _, _, _ string) error {
	return nil
}

func (f *fakeReleasesRepo) UpdateStatus(_ context.Context, id, status string) error {
	if f.statusErr != nil {
		return f.statusErr
	}
	f.statuses[id] = status
	return nil
}

func (f *fakeReleasesRepo) SetDeploymentID(_ context.Context, releaseID, deploymentID string) error {
	f.deployments[releaseID] = deploymentID
	return nil
}

func (f *fakeReleasesRepo) SetVerificationEvidence(_ context.Context, id string, items []releases.VerificationItem) error {
	f.evidence[id] = items
	return nil
}

func (f *fakeReleasesRepo) RecordPublicationReceipt(_ context.Context, _ string, receipt releases.PublicationReceipt) error {
	if f.receiptErr != nil {
		return f.receiptErr
	}
	f.receipts = append(f.receipts, receipt)
	return nil
}

func (f *fakeReleasesRepo) ListPublicationReceipts(_ context.Context, _ string) ([]releases.PublicationReceipt, error) {
	return f.receipts, nil
}

func (f *fakeReleasesRepo) MarkPlatformPublished(_ context.Context, _, _ string, _ int64) error {
	return nil
}
func (f *fakeReleasesRepo) MarkPlatformStatus(_ context.Context, _, _, _, _ string) error { return nil }
func (f *fakeReleasesRepo) MarkSuperseded(_ context.Context, _, _, exceptID string) error {
	if f.supersedeErr != nil {
		return f.supersedeErr
	}
	f.supersedeArg = exceptID
	return nil
}

func (f *fakeReleasesRepo) AcquireProfileLock(_ context.Context, _ string) (bool, func(), error) {
	return true, func() {}, nil
}

type fakeLPBSConfigRepo struct {
	cfg *profiles.LPBSReleaseConfig
}

func (f *fakeLPBSConfigRepo) Get(_ context.Context, _ string) (*profiles.LPBSReleaseConfig, error) {
	return f.cfg, nil
}

func (f *fakeLPBSConfigRepo) Upsert(_ context.Context, cfg *profiles.LPBSReleaseConfig) error {
	f.cfg = cfg
	return nil
}
func (f *fakeLPBSConfigRepo) Delete(_ context.Context, _ string) error { f.cfg = nil; return nil }

// --- helpers ---

func newDeployState(profileID, releaseID, channel, version string, platforms []string) *deployState {
	return &deployState{
		req: DeployDesktopRequest{
			ProfileID:      profileID,
			ReleaseID:      releaseID,
			Channel:        channel,
			ReleaseVersion: version,
			Platforms:      platforms,
		},
		response: &DeployDesktopResponse{
			ProfileID: profileID,
			Steps:     []OrchestrationStep{},
		},
		ctx: context.Background(),
	}
}

func newOrch(cloud CloudHealthClient, lpbs LPBSReleaseClient, cfgRepo profiles.LPBSReleaseConfigRepository, relRepo releases.Repository) *Orchestrator {
	return &Orchestrator{
		cloudClient:    cloud,
		lpbsClient:     lpbs,
		lpbsConfigRepo: cfgRepo,
		releasesRepo:   relRepo,
		log:            func(string, map[string]interface{}) {},
	}
}

func TestValidateCandidateTargetsRefusesUnreviewedTargetSet(t *testing.T) {
	candidate := releases.Candidate{
		SourceRevision: "source", ProfileRevision: "profile", DependencyLockDigest: "lock", PolicyDigest: "policy",
		Artifacts: []releases.CandidateArtifact{{
			Target:       releases.TargetIdentity{ID: "linux-x64", Platform: "linux", OS: "linux", Architecture: "amd64", Format: "appimage"},
			ImmutableRef: "artifact://linux", Digest: "sha256:artifact", SizeBytes: 1, SignatureDigest: "sha256:signature", SignerRef: "signer:test",
		}},
	}
	manifestDigest, err := candidate.ArtifactManifestDigest()
	if err != nil {
		t.Fatalf("candidate manifest digest: %v", err)
	}
	o := &Orchestrator{releaseIdentity: &candidateIdentityFake{candidate: &releases.CandidateRecord{
		ID: "candidate-1", Candidate: candidate, ArtifactManifestDigest: manifestDigest,
	}}}
	req := releases.DeployRequest{ReleaseID: "release-1", CandidateID: "candidate-1", Platforms: []string{"windows-x64"}}
	if _, err := o.validateCandidateTargets(context.Background(), req); err == nil {
		t.Fatal("candidate target drift was accepted")
	}
	req.Platforms = []string{"linux-x64"}
	req.ArtifactDigest = manifestDigest
	if _, err := o.validateCandidateTargets(context.Background(), req); err != nil {
		t.Fatalf("candidate target set was rejected: %v", err)
	}
}

func TestValidateReleaseDestinationRequiresExactRegisteredChannel(t *testing.T) {
	identity := &candidateIdentityFake{destination: &releases.DestinationRevisionRecord{
		ID:       "destination-1",
		Revision: releases.DestinationRevision{Kind: "lpbs", DestinationID: "staging", Channel: "stable"},
	}}
	o := &Orchestrator{releaseIdentity: identity}
	request := releases.DeployRequest{ReleaseID: "release-1", DestinationRevisionID: "destination-1", Channel: "stable"}
	if err := o.validateReleaseDestination(context.Background(), request); err != nil {
		t.Fatalf("exact destination was rejected: %v", err)
	}
	request.Channel = "beta"
	if err := o.validateReleaseDestination(context.Background(), request); err == nil {
		t.Fatal("destination channel drift was accepted")
	}
	request.DestinationRevisionID = "missing"
	if err := o.validateReleaseDestination(context.Background(), request); err == nil {
		t.Fatal("unregistered destination was accepted")
	}
}

func TestRecoverRoutesLPBSHaltToOwnerWithFencedDestination(t *testing.T) {
	owner := &fakeLPBSClient{haltReceipt: &LPBSRecoveryReceipt{AppKey: "desktop-app", VariantKey: "default", Revision: 7, Halted: true, Outcome: "halted", Health: "stopped", ExternalReceipt: "lpbs-halt-7", ObservedAt: time.Now().UTC()}}
	identity := &candidateIdentityFake{destination: &releases.DestinationRevisionRecord{ID: "destination-1", Revision: releases.DestinationRevision{Kind: "lpbs", DestinationID: "staging", Channel: "stable", ExpectedChannelRevision: "7"}}}
	o := &Orchestrator{releaseIdentity: identity, lpbsClient: owner, lpbsConfigRepo: &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "profile-1", LPBSAppKey: "desktop-app"}}}
	receipt, err := o.Recover(context.Background(), &releases.RecoveryRequest{ReleaseID: "release-1", ProfileID: "profile-1", Channel: "stable", DestinationRevisionID: "destination-1", CandidateID: "candidate-1", Action: "halt", Confirmation: "halt release-1"}, "deployment-1", "")
	if err != nil || receipt == nil {
		t.Fatalf("recover = %#v, %v", receipt, err)
	}
	if owner.haltRequest == nil || owner.haltRequest.AppKey != "desktop-app" || owner.haltRequest.ExpectedRevision != 7 || owner.haltRequest.VariantKey != "default" {
		t.Fatalf("owner request = %#v", owner.haltRequest)
	}
	if receipt.Outcome != "halted" || receipt.Health != "stopped" || receipt.ExternalReceipt != "lpbs-halt-7" {
		t.Fatalf("recovery receipt = %#v", receipt)
	}
}

func TestRecoverRoutesLPBSRollbackWithCompatibilityAndPredecessor(t *testing.T) {
	owner := &fakeLPBSClient{recoveryReceipt: &LPBSRecoveryReceipt{AppKey: "desktop-app", VariantKey: "default", Revision: 8, PredecessorRevision: 7, Action: "rollback", Outcome: "rolled_back", Health: "channel_restored", ExternalReceipt: "lpbs-rollback-8", ObservedAt: time.Now().UTC()}}
	identity := &candidateIdentityFake{destination: &releases.DestinationRevisionRecord{ID: "destination-1", Revision: releases.DestinationRevision{Kind: "lpbs", DestinationID: "staging", Channel: "stable", ExpectedChannelRevision: "7"}}}
	o := &Orchestrator{releaseIdentity: identity, lpbsClient: owner, lpbsConfigRepo: &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "profile-1", LPBSAppKey: "desktop-app"}}}
	receipt, err := o.Recover(context.Background(), &releases.RecoveryRequest{ReleaseID: "release-1", ProfileID: "profile-1", Channel: "stable", DestinationRevisionID: "destination-1", CandidateID: "candidate-1", Action: "rollback", ExpectedPredecessor: 6, DataCompatibility: "compatible", Confirmation: "rollback release-1"}, "deployment-1", "")
	if err != nil || receipt == nil {
		t.Fatalf("recover = %#v, %v", receipt, err)
	}
	if owner.recoveryRequest == nil || owner.recoveryRequest.ExpectedRevision != 7 || owner.recoveryRequest.ExpectedPredecessor != 6 || owner.recoveryRequest.DataCompatibility != "compatible" {
		t.Fatalf("owner request = %#v", owner.recoveryRequest)
	}
	if receipt.Outcome != "rolled_back" || receipt.Health != "channel_restored" {
		t.Fatalf("recovery receipt = %#v", receipt)
	}
}

func TestRecoveryControlsMatchConfiguredLPBSOwner(t *testing.T) {
	identity := &candidateIdentityFake{destination: &releases.DestinationRevisionRecord{ID: "destination-1", Revision: releases.DestinationRevision{Kind: "lpbs", DestinationID: "staging", Channel: "stable"}}}
	o := &Orchestrator{releaseIdentity: identity, lpbsClient: &fakeLPBSClient{}}
	controls, err := o.RecoveryControls(context.Background(), "destination-1")
	if err != nil {
		t.Fatalf("recovery controls: %v", err)
	}
	want := []string{"halt", "withdraw", "rollback", "forward_repair"}
	if strings.Join(controls, ",") != strings.Join(want, ",") {
		t.Fatalf("recovery controls = %v, want %v", controls, want)
	}
}

type cloudObservationFake struct {
	result *CloudHealthResult
}

func (f cloudObservationFake) CheckDeploymentHealth(context.Context, DeploymentHealthRequest) (*CloudHealthResult, error) {
	return f.result, nil
}

func TestObserveReleaseRequiresExactCloudDeploymentObservation(t *testing.T) {
	identity := &candidateIdentityFake{destination: &releases.DestinationRevisionRecord{ID: "destination-cloud", Revision: releases.DestinationRevision{Kind: "cloud", DestinationID: "staging", Channel: "stable"}}}
	o := &Orchestrator{
		releaseIdentity: identity,
		cloudClient: cloudObservationFake{result: &CloudHealthResult{
			DeploymentID: "cloud-deployment-1", ObservedReleaseDigest: "sha256:bundle", Healthy: true, ObservedAt: time.Now().UTC(),
		}},
	}
	observation, err := o.ObserveRelease(context.Background(), &releases.Release{DeploymentID: "cloud-deployment-1", ArtifactDigest: "sha256:bundle", DestinationRevisionID: "destination-cloud"})
	if err != nil || observation == nil || !observation.Healthy || observation.ObservedReleaseDigest != "sha256:bundle" {
		t.Fatalf("cloud observation = %#v, %v", observation, err)
	}
	o.cloudClient = cloudObservationFake{result: &CloudHealthResult{DeploymentID: "other-deployment", Healthy: true, ObservedAt: time.Now().UTC()}}
	if _, err := o.ObserveRelease(context.Background(), &releases.Release{DeploymentID: "cloud-deployment-1", ArtifactDigest: "sha256:bundle", DestinationRevisionID: "destination-cloud"}); err == nil {
		t.Fatal("observation for a different deployment was accepted")
	}
}

// --- tests ---

// TestDeployCheckCloudHealth_HealthyAdvancesStep [REQ:STC-P0-034] proves the
// selector is derived from the cloud manifest (scenario + environment +
// domain), never from a product slug, and the expected bundle digest is
// passed to the gate.
func TestDeployCheckCloudHealth_HealthyAdvancesStep(t *testing.T) {
	cloud := &fakeCloudClient{healthy: true}
	relRepo := newFakeReleasesRepo()
	o := newOrch(cloud, nil, nil, relRepo)
	ds := newDeployState("p1", "", "stable", "1.0.0", []string{"linux-x64"})
	ds.req.CloudManifest = json.RawMessage(`{"environment":"staging","scenario":{"id":"demo-app"},"edge":{"domain":"demo.example"},"target":{"vps":{"host":"203.0.113.10"}}}`)
	ds.req.CloudBundleSHA256 = strings.Repeat("b", 64)
	o.deployCheckCloudHealth(ds)
	if cloud.calls != 1 {
		t.Fatalf("expected cloud client called once, got %d", cloud.calls)
	}
	got := cloud.requests[0]
	if got.DeploymentID != "" || got.Selector.ScenarioID != "demo-app" || got.Selector.Environment != "staging" || got.Selector.Domain != "demo.example" || got.Selector.Host != "203.0.113.10" {
		t.Fatalf("health request selector = %+v", got)
	}
	if got.ExpectedReleaseDigest != strings.Repeat("b", 64) {
		t.Fatalf("expected release digest not passed: %q", got.ExpectedReleaseDigest)
	}
	if len(ds.response.Steps) != 1 || ds.response.Steps[0].Status != "success" {
		t.Fatalf("expected one success step, got %+v", ds.response.Steps)
	}
}

// TestDeployCheckCloudHealth_UsesReceiptIdentityWhenPresent [REQ:STC-P0-034]
// proves the exact deployment id from the cloud receipt wins over the
// selector (P16-A05: the id is not the scenario name).
func TestDeployCheckCloudHealth_UsesReceiptIdentityWhenPresent(t *testing.T) {
	cloud := &fakeCloudClient{healthy: true}
	o := newOrch(cloud, nil, nil, newFakeReleasesRepo())
	ds := newDeployState("p1", "", "stable", "1.0.0", []string{"linux-x64"})
	ds.req.CloudManifest = json.RawMessage(`{"scenario":{"id":"demo-app"},"edge":{"domain":"demo.example"}}`)
	ds.response.CloudReceipt = &CloudDeploymentReceipt{DeploymentID: "3c1c9a1e-5f3a-4a4d-9b1f-0d9d9f5b2f11", BundleSHA256: strings.Repeat("c", 64)}
	o.deployCheckCloudHealth(ds)
	got := cloud.requests[0]
	if got.DeploymentID != "3c1c9a1e-5f3a-4a4d-9b1f-0d9d9f5b2f11" || got.ExpectedReleaseDigest != strings.Repeat("c", 64) {
		t.Fatalf("health request = %+v", got)
	}
	if ds.response.Steps[0].Status != "success" || !strings.Contains(ds.response.Steps[0].Message, "3c1c9a1e-5f3a-4a4d-9b1f-0d9d9f5b2f11") {
		t.Fatalf("step = %+v", ds.response.Steps[0])
	}
}

// TestDeployCheckCloudHealth_UnhealthyMarksFailedRelease [REQ:STC-P0-034]
// proves an unhealthy verdict fails the release and surfaces the typed
// reason in the step message.
func TestDeployCheckCloudHealth_UnhealthyMarksFailedRelease(t *testing.T) {
	cloud := &fakeCloudClient{healthy: false, reason: "status_not_healthy", details: "application_readiness=FAILED(process_not_running)"}
	relRepo := newFakeReleasesRepo()
	o := newOrch(cloud, nil, nil, relRepo)
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})
	ds.req.CloudManifest = json.RawMessage(`{"scenario":{"id":"demo-app"},"edge":{"domain":"demo.example"}}`)

	o.deployCheckCloudHealth(ds)

	if relRepo.statuses["rel-1"] != releases.StatusFailed {
		t.Errorf("expected release marked failed; got %v", relRepo.statuses)
	}
	if ds.response.Steps[0].Status != "failed" || !strings.Contains(ds.response.Steps[0].Error, "status_not_healthy") {
		t.Errorf("expected failed step naming the reason; got %+v", ds.response.Steps[0])
	}
}

// TestDeployCheckCloudHealth_SkipsWithoutCloudIdentity proves a release
// with no cloud manifest and no receipt does not probe anything by name.
func TestDeployCheckCloudHealth_SkipsWithoutCloudIdentity(t *testing.T) {
	cloud := &fakeCloudClient{healthy: true}
	o := newOrch(cloud, nil, nil, newFakeReleasesRepo())
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})
	o.deployCheckCloudHealth(ds)
	if cloud.calls != 0 {
		t.Fatalf("cloud client was called %d times without a cloud identity", cloud.calls)
	}
	if len(ds.response.Steps) != 1 || ds.response.Steps[0].Status != "skipped" {
		t.Fatalf("steps = %+v", ds.response.Steps)
	}
}

func TestDeployCloudRequiresAndRecordsDurableReceipt(t *testing.T) {
	cloud := &fakeCloudDeploymentClient{
		fakeCloudClient: fakeCloudClient{healthy: true},
		receipt: &CloudDeploymentReceipt{
			SchemaVersion: 1, DeploymentID: "dep-1", ScenarioID: "demo", TargetKind: "vps",
			DestinationID: "sha256:target", DestinationHost: "vps.example", DestinationWorkdir: "/srv/demo", DestinationDomain: "demo.example",
			BundleSHA256: strings.Repeat("a", 64), Outcome: "deployed", Health: "healthy",
			ExternalReceipt: "scenario-to-cloud:dep-1", ObservedAt: time.Now().UTC(),
			ProducerRef: cloudReceiptProducerRef, TargetKey: "host:vps.example", ReleaseDigest: strings.Repeat("a", 64),
			Evidence: passingCloudEvidence(strings.Repeat("a", 64), "host:vps.example"),
		},
	}
	relRepo := newFakeReleasesRepo()
	o := newOrch(cloud, nil, nil, relRepo)
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})
	ds.req.CandidateID = "candidate-1"
	ds.req.DestinationRevisionID = "destination-1"
	ds.req.AuthorizationCheck = func(context.Context) error { return nil }
	ds.req.CloudManifest = json.RawMessage(`{"scenario":{"id":"demo"}}`)
	if code := o.deployCloud(ds); code != 0 {
		t.Fatalf("deployCloud() status = %d", code)
	}
	if cloud.deployCalls != 1 || ds.response.CloudReceipt != cloud.receipt {
		t.Fatalf("cloud calls/receipt = %d/%#v", cloud.deployCalls, ds.response.CloudReceipt)
	}
	if len(o.releasesRepo.(*fakeReleasesRepo).receipts) != 1 {
		t.Fatalf("persisted cloud receipts = %#v", o.releasesRepo.(*fakeReleasesRepo).receipts)
	}
	if got := o.releasesRepo.(*fakeReleasesRepo).deployments["rel-1"]; got != "dep-1" {
		t.Fatalf("persisted cloud deployment identity = %q", got)
	}
	if len(ds.response.Steps) != 1 || ds.response.Steps[0].Status != "success" {
		t.Fatalf("steps = %#v", ds.response.Steps)
	}
}

func TestDeployCloudReceiptPersistenceFailureIsAmbiguous(t *testing.T) {
	cloud := &fakeCloudDeploymentClient{
		fakeCloudClient: fakeCloudClient{healthy: true},
		receipt: &CloudDeploymentReceipt{
			SchemaVersion: 1, DeploymentID: "dep-ambiguous", ScenarioID: "demo", TargetKind: "vps",
			DestinationID: "sha256:target", DestinationHost: "vps.example", DestinationWorkdir: "/srv/demo", DestinationDomain: "demo.example",
			BundleSHA256: strings.Repeat("b", 64), Outcome: "deployed", Health: "healthy",
			ExternalReceipt: "scenario-to-cloud:dep-ambiguous", ObservedAt: time.Now().UTC(),
			ProducerRef: cloudReceiptProducerRef, TargetKey: "host:vps.example", ReleaseDigest: strings.Repeat("a", 64),
			Evidence: passingCloudEvidence(strings.Repeat("a", 64), "host:vps.example"),
		},
	}
	relRepo := newFakeReleasesRepo()
	relRepo.receiptErr = errors.New("receipt store unavailable")
	o := newOrch(cloud, nil, nil, relRepo)
	ds := newDeployState("p1", "rel-ambiguous", "stable", "1.0.0", []string{"linux-x64"})
	ds.req.CandidateID = "candidate-1"
	ds.req.DestinationRevisionID = "destination-1"
	ds.req.AuthorizationCheck = func(context.Context) error { return nil }
	ds.req.CloudManifest = json.RawMessage(`{"scenario":{"id":"demo"}}`)

	if code := o.deployCloud(ds); code != http.StatusBadGateway {
		t.Fatalf("deployCloud() status = %d", code)
	}
	if ds.response.Status != "ambiguous" || ds.response.Steps[0].Status != "failed" {
		t.Fatalf("response = %#v; want ambiguous response with failed step", ds.response)
	}
	if relRepo.statuses["rel-ambiguous"] != releases.StatusAmbiguous {
		t.Fatalf("durable release status = %q; want ambiguous", relRepo.statuses["rel-ambiguous"])
	}
}

func TestDeployCloudReauthorizesBeforeOwnerCall(t *testing.T) {
	cloud := &fakeCloudDeploymentClient{
		fakeCloudClient: fakeCloudClient{healthy: true},
		receipt: &CloudDeploymentReceipt{
			SchemaVersion: 1, DeploymentID: "dep-revoked", ScenarioID: "demo", TargetKind: "vps",
			DestinationID: "sha256:target", DestinationHost: "vps.example", DestinationWorkdir: "/srv/demo", DestinationDomain: "demo.example",
			BundleSHA256: strings.Repeat("c", 64), Outcome: "deployed", Health: "healthy",
			ExternalReceipt: "scenario-to-cloud:dep-revoked", ObservedAt: time.Now().UTC(),
			ProducerRef: cloudReceiptProducerRef, TargetKey: "host:vps.example", ReleaseDigest: strings.Repeat("a", 64),
			Evidence: passingCloudEvidence(strings.Repeat("a", 64), "host:vps.example"),
		},
	}
	relRepo := newFakeReleasesRepo()
	o := newOrch(cloud, nil, nil, relRepo)
	ds := newDeployState("p1", "rel-revoked", "stable", "1.0.0", []string{"linux-x64"})
	ds.req.CandidateID = "candidate-1"
	ds.req.DestinationRevisionID = "destination-1"
	ds.req.AuthorizationCheck = func(context.Context) error { return errors.New("approval revoked") }
	ds.req.CloudManifest = json.RawMessage(`{"scenario":{"id":"demo"}}`)

	if code := o.deployCloud(ds); code != http.StatusPreconditionFailed {
		t.Fatalf("deployCloud() status = %d", code)
	}
	if cloud.deployCalls != 0 {
		t.Fatalf("cloud owner was called %d times after authorization refusal", cloud.deployCalls)
	}
	if ds.response.Status != "blocked" || ds.response.Steps[0].Status != "failed" {
		t.Fatalf("response = %#v; want blocked response with failed step", ds.response)
	}
}

func TestDeployCloudFailsClosedWithoutLifecycleClient(t *testing.T) {
	o := newOrch(&fakeCloudClient{healthy: true}, nil, nil, newFakeReleasesRepo())
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})
	ds.req.CandidateID = "candidate-1"
	ds.req.DestinationRevisionID = "destination-1"
	ds.req.AuthorizationCheck = func(context.Context) error { return nil }
	ds.req.CloudManifest = json.RawMessage(`{"scenario":{"id":"demo"}}`)
	if code := o.deployCloud(ds); code != http.StatusServiceUnavailable {
		t.Fatalf("deployCloud() status = %d", code)
	}
	if ds.response.Status != "failed" || ds.response.Steps[0].Status != "failed" {
		t.Fatalf("response = %#v", ds.response)
	}
}

func TestDeployCheckLPBSReadiness_SkipsWhenNoConfig(t *testing.T) {
	lpbs := &fakeLPBSClient{}
	cfgRepo := &fakeLPBSConfigRepo{cfg: nil}
	o := newOrch(nil, lpbs, cfgRepo, nil)
	ds := newDeployState("p1", "", "stable", "1", []string{"linux-x64"})
	o.deployCheckLPBSReadiness(ds)

	if lpbs.readinessCalls != 0 {
		t.Errorf("expected readiness not called when no config; got %d", lpbs.readinessCalls)
	}
	if len(ds.response.Steps) != 1 || ds.response.Steps[0].Status != "skipped" {
		t.Errorf("expected one skipped step, got %+v", ds.response.Steps)
	}
}

func TestPublishToLPBSFailsClosedWithoutExactCoordinates(t *testing.T) {
	o := newOrch(nil, nil, &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "app"}}, nil)
	step := OrchestrationStep{Name: "Publish desktop artifacts"}
	response := &DeployDesktopResponse{}
	o.publishToLPBS(context.Background(), &profiles.Profile{Scenario: "demo"}, DeployDesktopRequest{ProfileID: "p1", ReleaseID: "release-1"}, response, &step)
	if step.Status != "failed" || !strings.Contains(step.Error, "exact LPBS app key and remote profile") {
		t.Fatalf("publish step = %+v", step)
	}
}

func TestPublishToLPBSRefusesCurrentProfileOutsideApprovedDestination(t *testing.T) {
	runner := &fakePublishPipelineRunner{status: &PipelineStatus{CurrentState: "completed"}}
	o := newOrch(nil, nil, &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "app", LPBSRemoteProfile: "current"}}, newFakeReleasesRepo())
	o.releaseIdentity = &candidateIdentityFake{destination: &releases.DestinationRevisionRecord{
		ID: "destination-1", Revision: releases.DestinationRevision{DestinationID: "approved", Channel: "stable"},
	}}
	o.publishPipelineRunner = runner
	step := OrchestrationStep{Name: "Publish desktop artifacts"}
	response := &DeployDesktopResponse{}
	o.publishToLPBS(context.Background(), &profiles.Profile{Scenario: "demo"}, DeployDesktopRequest{
		ProfileID: "p1", ReleaseID: "release-1", DestinationRevisionID: "destination-1", Channel: "stable",
	}, response, &step)
	if step.Status != "failed" || !strings.Contains(step.Error, "does not match") {
		t.Fatalf("publish step = %+v", step)
	}
	if runner.calls != 0 {
		t.Fatalf("owner pipeline calls = %d, want 0", runner.calls)
	}
}

func TestPublishToLPBSPersistenceFailureAfterOwnerStartIsAmbiguous(t *testing.T) {
	relRepo := newFakeReleasesRepo()
	o := newOrch(nil, nil, &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "app", LPBSRemoteProfile: "staging"}}, relRepo)
	o.publishedVersionsRepo = failingPublishedVersionRepo{}
	o.publishPipelineRunner = &fakePublishPipelineRunner{status: &PipelineStatus{
		CurrentState: "completed",
		Provenance:   &PipelineBuildProvenance{Version: "1.0.0", GitCommitHash: "commit-1"},
		Stages:       map[string]*PipelineStageResult{"deploy": {Details: map[string]any{"artifacts": []any{map[string]any{"artifact_id": 7, "platform": "linux-x64", "sha512": "artifact-digest", "destination_object": "lpbs://staging/linux-x64/1"}}}}},
	}}
	response := &DeployDesktopResponse{}
	step := OrchestrationStep{Name: "Publish desktop artifacts"}
	o.publishToLPBS(context.Background(), &profiles.Profile{Scenario: "demo"}, DeployDesktopRequest{
		ProfileID: "p1", ReleaseID: "release-1", Platforms: []string{"linux-x64"}, Channel: "stable", ReleaseVersion: "1.0.0", GitCommitHash: "commit-1", ArtifactDigest: "sha256:manifest",
	}, response, &step)
	if response.Status != "ambiguous" || step.Status != "failed" {
		t.Fatalf("owner effect followed by persistence failure = status %q, step %+v; want ambiguous/failed", response.Status, step)
	}
	if relRepo.statuses["release-1"] != releases.StatusAmbiguous {
		t.Fatalf("durable release status = %q, want ambiguous", relRepo.statuses["release-1"])
	}
}

func TestDeployCheckLPBSReadiness_NotReadyFails(t *testing.T) {
	lpbs := &fakeLPBSClient{readiness: &LPBSReadinessResult{Ready: false, Error: "missing storage"}}
	cfgRepo := &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k", DefaultChannel: "stable"}}
	relRepo := newFakeReleasesRepo()
	o := newOrch(nil, lpbs, cfgRepo, relRepo)
	ds := newDeployState("p1", "rel-1", "stable", "1", []string{"linux-x64"})

	o.deployCheckLPBSReadiness(ds)

	if relRepo.statuses["rel-1"] != releases.StatusFailed {
		t.Errorf("expected release failed; got %v", relRepo.statuses)
	}
	if ds.response.Steps[0].Status != "failed" {
		t.Errorf("expected failed step; got %q", ds.response.Steps[0].Status)
	}
}

func TestDeployVerifyUpdateEndpoints_AllMatchPublishes(t *testing.T) {
	lpbs := &fakeLPBSClient{verifyOutcomes: map[string]*LPBSVerifyResult{
		"linux-x64":    {Match: true, SHA512Match: true, ObservedVersion: "1.0.0"},
		"darwin-arm64": {Match: true, SHA512Match: true, ObservedVersion: "1.0.0"},
	}}
	cfgRepo := &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k", DefaultChannel: "stable"}}
	relRepo := newFakeReleasesRepo()
	o := newOrch(nil, lpbs, cfgRepo, relRepo)
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64", "darwin-arm64"})
	ds.response.PublishedVersions = []PublishedVersion{
		{Platform: "linux-x64", Version: "1.0.0", SHA512: "digest-linux"},
		{Platform: "darwin-arm64", Version: "1.0.0", SHA512: "digest-darwin"},
	}

	o.deployVerifyUpdateEndpoints(ds)

	if relRepo.statuses["rel-1"] != releases.StatusPublished {
		t.Errorf("expected status=published; got %v", relRepo.statuses)
	}
	if relRepo.supersedeArg != "rel-1" {
		t.Errorf("expected MarkSuperseded with except=rel-1; got %q", relRepo.supersedeArg)
	}
	if got := relRepo.evidence["rel-1"]; len(got) != 2 {
		t.Errorf("expected 2 evidence items; got %d", len(got))
	}
	if ds.response.Steps[0].Status != "success" {
		t.Errorf("expected verify step success; got %q", ds.response.Steps[0].Status)
	}
	if len(lpbs.verifyRequests) != 2 {
		t.Fatalf("expected two verification requests; got %d", len(lpbs.verifyRequests))
	}
	for _, request := range lpbs.verifyRequests {
		if !request.Deep {
			t.Errorf("publication verification must request deep byte verification: %#v", request)
		}
	}
}

func TestFinalizeAndVerifyReleaseStopsAfterPublicationFailure(t *testing.T) {
	lpbs := &fakeLPBSClient{}
	o := newOrch(nil, lpbs, &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k"}}, newFakeReleasesRepo())
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})
	ds.profile = &profiles.Profile{Scenario: "demo"}

	// A release-bound finalization without the durable publication repository
	// must fail before verification can inspect the remote destination.
	o.finalizeAndVerifyRelease(ds)

	if ds.response.Status != "failed" {
		t.Fatalf("finalization status = %q, want failed", ds.response.Status)
	}
	if lpbs.verifyCallsCount != 0 {
		t.Fatalf("verification calls = %d, want 0 after publication failure", lpbs.verifyCallsCount)
	}
}

func TestDeployVerifyUpdateEndpointsFailsClosedWhenVerifierUnavailable(t *testing.T) {
	relRepo := newFakeReleasesRepo()
	o := newOrch(nil, nil, &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k"}}, relRepo)
	ds := newDeployState("p1", "rel-verify", "stable", "1.0.0", []string{"linux-x64"})
	ds.response.PublishedVersions = []PublishedVersion{{Platform: "linux-x64", Version: "1.0.0", SHA512: "digest"}}

	o.deployVerifyUpdateEndpoints(ds)

	if ds.response.Status != "verify_failed" || len(ds.response.Steps) != 1 || ds.response.Steps[0].Status != "failed" {
		t.Fatalf("verification standing = %#v", ds.response)
	}
	if relRepo.statuses["rel-verify"] != releases.StatusVerifyFailed {
		t.Fatalf("release status = %q, want verify_failed", relRepo.statuses["rel-verify"])
	}
}

func TestDeployVerifyUpdateEndpoints_MismatchMarksVerifyFailed(t *testing.T) {
	lpbs := &fakeLPBSClient{verifyOutcomes: map[string]*LPBSVerifyResult{
		"linux-x64":    {Match: true, SHA512Match: true, ObservedVersion: "1.0.0"},
		"darwin-arm64": {Match: false, ObservedVersion: "0.9.9"},
	}}
	cfgRepo := &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k", DefaultChannel: "stable"}}
	relRepo := newFakeReleasesRepo()
	o := newOrch(nil, lpbs, cfgRepo, relRepo)
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64", "darwin-arm64"})
	ds.response.PublishedVersions = []PublishedVersion{
		{Platform: "linux-x64", Version: "1.0.0", SHA512: "digest-linux"},
		{Platform: "darwin-arm64", Version: "1.0.0", SHA512: "digest-darwin"},
	}

	o.deployVerifyUpdateEndpoints(ds)

	if relRepo.statuses["rel-1"] != releases.StatusVerifyFailed {
		t.Errorf("expected verify_failed status; got %v", relRepo.statuses)
	}
	if ds.response.Status != "verify_failed" {
		t.Errorf("expected response.Status verify_failed; got %q", ds.response.Status)
	}
	if ds.response.Steps[0].Status != "failed" {
		t.Errorf("expected step failed; got %q", ds.response.Steps[0].Status)
	}
}

func TestDeployVerifyUpdateEndpoints_SupersedePersistenceFailureIsAmbiguous(t *testing.T) {
	lpbs := &fakeLPBSClient{verifyOutcomes: map[string]*LPBSVerifyResult{
		"linux-x64": {Match: true, SHA512Match: true, ObservedVersion: "1.0.0"},
	}}
	cfgRepo := &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k", DefaultChannel: "stable"}}
	relRepo := newFakeReleasesRepo()
	relRepo.supersedeErr = errors.New("ledger unavailable")
	o := newOrch(nil, lpbs, cfgRepo, relRepo)
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})
	ds.response.PublishedVersions = []PublishedVersion{{Platform: "linux-x64", Version: "1.0.0", SHA512: "digest-linux"}}

	o.deployVerifyUpdateEndpoints(ds)

	if ds.response.Status != "ambiguous" {
		t.Fatalf("expected ambiguous response after remote verification and failed supersession persistence, got %q", ds.response.Status)
	}
	if ds.response.Steps[0].Status != "failed" {
		t.Fatalf("expected failed verification step, got %q", ds.response.Steps[0].Status)
	}
}

func TestDeployVerifyUpdateEndpoints_StatusPersistenceFailureIsAmbiguous(t *testing.T) {
	lpbs := &fakeLPBSClient{verifyOutcomes: map[string]*LPBSVerifyResult{
		"linux-x64": {Match: true, SHA512Match: true, ObservedVersion: "1.0.0"},
	}}
	cfgRepo := &fakeLPBSConfigRepo{cfg: &profiles.LPBSReleaseConfig{ProfileID: "p1", LPBSAppKey: "k", DefaultChannel: "stable"}}
	relRepo := newFakeReleasesRepo()
	relRepo.statusErr = errors.New("ledger unavailable")
	o := newOrch(nil, lpbs, cfgRepo, relRepo)
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})
	ds.response.PublishedVersions = []PublishedVersion{{Platform: "linux-x64", Version: "1.0.0", SHA512: "digest-linux"}}

	o.deployVerifyUpdateEndpoints(ds)

	if ds.response.Status != "ambiguous" {
		t.Fatalf("expected ambiguous response after status persistence failure, got %q", ds.response.Status)
	}
}

func TestEffectiveChannel_RequestedWins(t *testing.T) {
	if got := effectiveChannel("beta", "stable"); got != "beta" {
		t.Errorf("expected beta; got %q", got)
	}
	if got := effectiveChannel("", "nightly"); got != "nightly" {
		t.Errorf("expected nightly default; got %q", got)
	}
	if got := effectiveChannel("", ""); got != "stable" {
		t.Errorf("expected stable fallback; got %q", got)
	}
}

func TestSummarizeResult_CopiesStepsAndPublished(t *testing.T) {
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})
	ds.response.Status = "success"
	ds.response.Steps = []OrchestrationStep{{Name: "s1", Status: "success"}}
	ds.response.PublishedVersions = []PublishedVersion{{Platform: "linux-x64", Version: "1.0.0", ArtifactID: 42}}

	got := summarizeResult(ds)
	if got.ReleaseID != "rel-1" || got.Status != "success" {
		t.Errorf("unexpected result: %+v", got)
	}
	if len(got.Steps) != 1 || got.Steps[0].Name != "s1" {
		t.Errorf("expected one step copied; got %+v", got.Steps)
	}
	if len(got.PublishedVersions) != 1 || got.PublishedVersions[0].ArtifactID != 42 {
		t.Errorf("expected published copied; got %+v", got.PublishedVersions)
	}
}

// passingCloudEvidence is a complete owner summary for one release/target.
func passingCloudEvidence(release, target string) *CloudEvidenceSummary {
	cells := []CloudEvidenceCell{}
	for _, id := range []string{"GOV-01/api", "GOV-05/api", "RELEASE-07/package"} {
		parts := strings.SplitN(id, "/", 2)
		cells = append(cells, CloudEvidenceCell{CaseID: parts[0], Lane: parts[1], Disposition: "passed", Required: true, RecordID: "rec-" + parts[0], OperationID: "op-1", ReceiptRefs: []string{"cloud-target:op-1:" + parts[0]}})
	}
	return &CloudEvidenceSummary{SchemaVersion: "1", ProfileID: "cloud-launch-v1", ReleaseDigest: release, TargetKey: target, Cells: cells, RequiredCells: len(cells), Passed: true, ProducerRef: cloudReceiptProducerRef, EvaluatedAt: time.Now().UTC()}
}
