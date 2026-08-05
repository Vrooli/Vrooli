package deployments

import (
	"context"
	"errors"
	"testing"

	"deployment-manager/profiles"
	"deployment-manager/releases"
)

// --- fakes for the new clients ---

type fakeCloudClient struct {
	healthy bool
	details string
	err     error
	calls   int
}

func (f *fakeCloudClient) CheckLPBSHealth(_ context.Context) (*CloudHealthResult, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &CloudHealthResult{Healthy: f.healthy, Details: f.details}, nil
}

type fakeLPBSClient struct {
	readiness        *LPBSReadinessResult
	readinessErr     error
	verifyOutcomes   map[string]*LPBSVerifyResult
	verifyErr        error
	readinessCalls   int
	verifyCallsCount int
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
	supersedeArg string
}

func newFakeReleasesRepo() *fakeReleasesRepo {
	return &fakeReleasesRepo{statuses: map[string]string{}, evidence: map[string][]releases.VerificationItem{}}
}

func (f *fakeReleasesRepo) Insert(_ context.Context, rel *releases.Release) error {
	f.statuses[rel.ID] = rel.Status
	return nil
}

func (f *fakeReleasesRepo) Get(_ context.Context, _ string) (*releases.Release, error) {
	return nil, errors.New("not used")
}

func (f *fakeReleasesRepo) ListByProfile(_ context.Context, _ string, _ int) ([]*releases.Release, error) {
	return nil, nil
}

func (f *fakeReleasesRepo) UpdateStatus(_ context.Context, id, status string) error {
	f.statuses[id] = status
	return nil
}

func (f *fakeReleasesRepo) SetVerificationEvidence(_ context.Context, id string, items []releases.VerificationItem) error {
	f.evidence[id] = items
	return nil
}

func (f *fakeReleasesRepo) MarkPlatformPublished(_ context.Context, _, _ string, _ int64) error {
	return nil
}
func (f *fakeReleasesRepo) MarkPlatformStatus(_ context.Context, _, _, _, _ string) error { return nil }
func (f *fakeReleasesRepo) MarkSuperseded(_ context.Context, _, _, exceptID string) error {
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

// --- tests ---

func TestDeployCheckCloudHealth_HealthyAdvancesStep(t *testing.T) {
	cloud := &fakeCloudClient{healthy: true}
	o := newOrch(cloud, nil, nil, nil)
	ds := newDeployState("p1", "", "stable", "1.0.0", []string{"linux-x64"})
	o.deployCheckCloudHealth(ds)
	if cloud.calls != 1 {
		t.Fatalf("expected cloud client called once, got %d", cloud.calls)
	}
	if len(ds.response.Steps) != 1 || ds.response.Steps[0].Status != "success" {
		t.Fatalf("expected one success step, got %+v", ds.response.Steps)
	}
}

func TestDeployCheckCloudHealth_UnhealthyMarksFailedRelease(t *testing.T) {
	cloud := &fakeCloudClient{healthy: false, details: "down"}
	relRepo := newFakeReleasesRepo()
	o := newOrch(cloud, nil, nil, relRepo)
	ds := newDeployState("p1", "rel-1", "stable", "1.0.0", []string{"linux-x64"})

	o.deployCheckCloudHealth(ds)

	if relRepo.statuses["rel-1"] != releases.StatusFailed {
		t.Errorf("expected release marked failed; got %v", relRepo.statuses)
	}
	if ds.response.Steps[0].Status != "failed" {
		t.Errorf("expected step failed; got %q", ds.response.Steps[0].Status)
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
		{Platform: "linux-x64", Version: "1.0.0"},
		{Platform: "darwin-arm64", Version: "1.0.0"},
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
		{Platform: "linux-x64", Version: "1.0.0"},
		{Platform: "darwin-arm64", Version: "1.0.0"},
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
