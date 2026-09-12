package ramp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/evidence"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/release"
	"scenario-to-cloud/releasesvc"

	"github.com/vrooli/api-core/targetmodel"
	"github.com/vrooli/vrooli/packages/cloudrelease"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	fixtureNow    = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	fixtureDigest = strings.Repeat("a", 64)
	fixtureOther  = strings.Repeat("b", 64)
)

type fakeDeployments struct{ deployments []*domain.Deployment }

func (f fakeDeployments) ListDeployments(context.Context, domain.ListFilter) ([]*domain.Deployment, error) {
	return f.deployments, nil
}

func (f fakeDeployments) GetDeployment(_ context.Context, id string) (*domain.Deployment, error) {
	for _, dep := range f.deployments {
		if dep.ID == id {
			return dep, nil
		}
	}
	return nil, nil
}

type fakeObserver struct {
	observations map[string]*healthv1.HealthObservation
	errs         map[string]error
}

func (f fakeObserver) Observe(_ context.Context, id string) (*healthv1.HealthObservation, error) {
	if err := f.errs[id]; err != nil {
		return nil, err
	}
	return f.observations[id], nil
}

type fakeReleases struct{ rel release.Release }

func (f fakeReleases) Build(context.Context, releasesvc.BuildRequest) (release.Release, error) {
	return f.rel, nil
}

type fakeExecutor struct {
	results  map[string]ExecutionResult
	errs     map[string]error
	requests []ExecutionRequest
}

func (f *fakeExecutor) Execute(_ context.Context, req ExecutionRequest) (ExecutionResult, error) {
	f.requests = append(f.requests, req)
	if err := f.errs[req.CaseID]; err != nil {
		return ExecutionResult{}, err
	}
	return f.results[req.CaseID], nil
}

type fakeTarget struct {
	receipt evidence.TargetReceipt
	err     error
}

func (f fakeTarget) ReadActiveRelease(context.Context, string) (evidence.TargetReceipt, error) {
	return f.receipt, f.err
}

type fakeRecorder struct{ records []evidence.Record }

func (f *fakeRecorder) AppendEvidenceRecord(_ context.Context, rec evidence.Record) error {
	f.records = append(f.records, rec)
	return nil
}

func fixtureDeployments() []*domain.Deployment {
	manifest, _ := json.Marshal(domain.CloudManifest{Version: "1", Scenario: domain.ManifestScenario{ID: "demo"}, Target: domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}}, Edge: domain.ManifestEdge{Domain: "demo.example"}})
	return []*domain.Deployment{
		{ID: "dep-bridge", Name: "demo @ bridge", ScenarioID: "demo", Environment: "staging", Status: domain.StatusDeployed, Manifest: manifest, Fence: 3, Target: identity.TargetRef{MachineID: "fixture-a", NodeID: "node-a", Transport: identity.TransportBridge, Locator: identity.TargetLocator{Host: "203.0.113.10"}}},
		{ID: "dep-ssh", Name: "demo @ ssh", ScenarioID: "demo", Environment: "staging", Status: domain.StatusDeployed, Manifest: manifest, Target: identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.11"}}},
		{ID: "dep-dark", Name: "demo @ dark", ScenarioID: "demo", Environment: "staging", Status: domain.StatusDeployed, Manifest: manifest, Target: identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.12"}}},
	}
}

func healthy(id string) *healthv1.HealthObservation {
	return &healthv1.HealthObservation{DeploymentId: id, Status: healthv1.HealthStatus_HEALTH_STATUS_HEALTHY, Freshness: healthv1.Freshness_FRESHNESS_CURRENT, ObservedAt: timestamppb.New(fixtureNow), Checks: []*healthv1.HealthCheck{{Id: "application_readiness", Status: healthv1.CheckStatus_CHECK_STATUS_PASSED}}}
}

func fixtureRelease(t *testing.T) release.Release {
	t.Helper()
	return release.Release{Digest: fixtureDigest, Dir: t.TempDir(), Complete: true, Manifest: cloudrelease.Manifest{BundleSHA256: strings.Repeat("c", 64), NativeCLI: cloudrelease.NativeCLI{GOOS: "linux", GOARCH: "amd64", SHA256: strings.Repeat("d", 64)}, ClosureDigest: "sha256:" + strings.Repeat("e", 64), ConfigurationDigest: strings.Repeat("f", 64)}, Inputs: domain.ReleaseInputs{BuiltAt: fixtureNow.Format(time.RFC3339Nano)}}
}

func fixtureArtifact(t *testing.T) deliveryramp.Artifact {
	t.Helper()
	artifact, err := Builder{Deployments: fakeDeployments{fixtureDeployments()}, Releases: fakeReleases{fixtureRelease(t)}}.Build(context.Background(), deliveryramp.BuildRequest{SourceRef: "dep-bridge"})
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

// TestCloudRampConformsThroughExportedSpineSeams [REQ:STC-P0-035] follows
// packages/delivery-ramp-go/validationmatrix/reference_ramp_test.go: the
// four adapters satisfy the exported seams, the Prober carries the cause of
// unavailability, the Driver runs through JourneyRunner, the Distributor's
// passing result validates with a target-owned effect receipt, and the
// target verdict contract accepts every cloud target shape (P17-A06).
func TestCloudRampConformsThroughExportedSpineSeams(t *testing.T) {
	var _ deliveryramp.Prober = Prober{}
	var _ deliveryramp.Builder = Builder{}
	var _ deliveryramp.Driver = Driver{}
	var _ deliveryramp.Distributor = Distributor{}

	observer := fakeObserver{
		observations: map[string]*healthv1.HealthObservation{"dep-bridge": healthy("dep-bridge"), "dep-ssh": {DeploymentId: "dep-ssh", Status: healthv1.HealthStatus_HEALTH_STATUS_HEALTHY, Freshness: healthv1.Freshness_FRESHNESS_STALE, ObservedAt: timestamppb.New(fixtureNow)}},
		errs:         map[string]error{"dep-dark": errors.New("ssh: connection refused")},
	}
	prober := Prober{Deployments: fakeDeployments{fixtureDeployments()}, Observer: observer, Now: func() time.Time { return fixtureNow }}
	inventory, err := prober.Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil || len(inventory.Targets) != 3 {
		t.Fatalf("probe = %+v err=%v", inventory, err)
	}
	bridge, ssh, dark := inventory.Targets[0], inventory.Targets[1], inventory.Targets[2]
	if bridge.ID != "machine:fixture-a" || bridge.Transport.Kind != deliveryramp.TransportBridge || !bridge.Available || bridge.NodeID != "node-a" || bridge.Health.Status != "healthy" {
		t.Fatalf("bridge target = %+v", bridge)
	}
	if ssh.ID != "host:203.0.113.11" || ssh.Transport.Kind != targetmodel.TransportSSH || ssh.Available || !strings.Contains(ssh.Reason, "STALE") || ssh.NextAction == "" {
		t.Fatalf("stale ssh target must be unavailable with cause: %+v", ssh)
	}
	if dark.Available || !strings.Contains(dark.Reason, "connection refused") || dark.MissingCapability != "target_reach" {
		t.Fatalf("unreachable target must carry the cause: %+v", dark)
	}
	bridgeOnly, err := prober.Probe(context.Background(), deliveryramp.ProbeRequest{TransportKinds: []deliveryramp.TransportKind{deliveryramp.TransportBridge}})
	if err != nil || len(bridgeOnly.Targets) != 1 {
		t.Fatalf("bridge-only probe = %+v err=%v", bridgeOnly, err)
	}

	artifact := fixtureArtifact(t)
	if artifact.ImmutableRef != "release:"+fixtureDigest || artifact.Kind != ArtifactKind || artifact.Checksum != "sha256:"+fixtureDigest || artifact.Metadata["bundle_sha256"] == "" || artifact.Metadata["native_cli"] == "" || artifact.Metadata["closure_digest"] == "" || artifact.Metadata["configuration_digest"] == "" {
		t.Fatalf("artifact = %+v", artifact)
	}

	profile, err := evidence.LoadProfile(evidence.ProfileCloudLaunchV1)
	if err != nil {
		t.Fatal(err)
	}
	cell := profile.Cells[0]
	executor := &fakeExecutor{results: map[string]ExecutionResult{cell.CaseID: {OperationID: "op-1", CompletedAt: fixtureNow, Outcome: evidence.DispositionPassed, ReceiptRefs: []string{"cloud-target:op-1:release-activate"}, Assertions: []string{"external endpoint serves the exact release"}}}}
	recorder := &fakeRecorder{}
	driver := Driver{Executor: executor, Recorder: recorder, Profile: profile, Now: func() time.Time { return fixtureNow }}
	plan := deliveryramp.JourneyPlan{ID: "cloud-launch-v1/" + cell.String(), Capability: PlanCapability, Profile: profile.ID}
	result := (deliveryramp.JourneyRunner{Driver: driver}).Run(context.Background(), deliveryramp.JourneyExecutionRequest{RunID: "run-1", Cell: deliveryramp.Cell{ID: cell.String(), Target: bridge, ProfileID: profile.ID, Capability: PlanCapability, Required: true}, Target: bridge, Artifact: artifact, Plan: plan})
	if result.Disposition != deliveryramp.DispositionPass || len(result.Steps) != 1 || result.Steps[0].Disposition != deliveryramp.StepPassed || len(result.Steps[0].Evidence) != 2 {
		t.Fatalf("journey = %+v", result)
	}
	if len(recorder.records) != 1 || recorder.records[0].Disposition != evidence.DispositionPassed || recorder.records[0].Binding.OperationID != "op-1" || recorder.records[0].Binding.TargetKey != bridge.ID || recorder.records[0].Binding.ReleaseDigest != fixtureDigest {
		t.Fatalf("recorded evidence = %+v", recorder.records)
	}
	if err := recorder.records[0].Validate(profile); err != nil {
		t.Fatalf("recorded evidence must validate: %v", err)
	}

	target := fakeTarget{receipt: evidence.TargetReceipt{ActiveRelease: fixtureDigest, PreviousRelease: fixtureOther, ActivatedAt: fixtureNow.Format(time.RFC3339), OperationID: "op-pub", Fence: 4}}
	distributor := Distributor{Executor: &fakeExecutor{results: map[string]ExecutionResult{PublishCase: {OperationID: "op-pub", Outcome: evidence.DispositionPassed, ReceiptRefs: []string{"cloud-target:op-pub:release-activate"}}}}, Target: target, Now: func() time.Time { return fixtureNow }}
	distribution, err := distributor.Distribute(context.Background(), deliveryramp.DistributionRequest{Cell: deliveryramp.Cell{ID: "publish-1", Target: bridge}, Artifact: artifact})
	if err != nil || distribution.Disposition != deliveryramp.DispositionPass || distribution.EffectReceipt == nil || distribution.EffectReceipt.TargetID != bridge.ID || distribution.EffectReceipt.ArtifactRef != artifact.ImmutableRef || !strings.HasPrefix(distribution.EffectReceipt.ExternalReceipt, "cloud-target:active-release:sha256:") {
		t.Fatalf("distribution = %+v err=%v", distribution, err)
	}
	if err := distribution.Validate(); err != nil {
		t.Fatal(err)
	}

	for _, shape := range []deliveryramp.Target{bridge, {ID: "host:203.0.113.11", Ramp: RampID, Platform: "vps", OS: "linux", DeviceKind: "host", Available: true, Transport: deliveryramp.Transport{Kind: targetmodel.TransportSSH}}} {
		verdict, err := deliveryramp.NewTargetVerdict(deliveryramp.TargetVerdictInput{Producer: RampID, Target: shape, Disposition: deliveryramp.DispositionPass, RunID: "run-1", CreatedAt: fixtureNow, References: result.Steps[0].Evidence})
		if err != nil || verdict.GetDisposition() != commonv1.Disposition_DISPOSITION_PASSED || len(verdict.GetRefs()) != 2 {
			t.Fatalf("verdict for %s = %+v err=%v", shape.ID, verdict, err)
		}
	}
}

// TestDriverKeepsDispatchAcceptanceDistinctFromTargetAssertion
// [REQ:STC-P0-035] proves P17 step 5: an accepted run without a target
// receipt is unavailable, a pipeline failure is unavailable, and a failed
// assertion is failed (never degraded into a pass).
func TestDriverKeepsDispatchAcceptanceDistinctFromTargetAssertion(t *testing.T) {
	profile, _ := evidence.LoadProfile(evidence.ProfileCloudLaunchV1)
	cell := profile.Cells[0]
	artifact := fixtureArtifact(t)
	target := deliveryramp.Target{ID: "machine:fixture-a", Ramp: RampID, Platform: "vps", OS: "linux", DeviceKind: "host", Available: true, Capabilities: []string{PlanCapability}, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportBridge, Available: true}}
	run := func(exec *fakeExecutor) (deliveryramp.JourneyResult, []evidence.Record) {
		recorder := &fakeRecorder{}
		driver := Driver{Executor: exec, Recorder: recorder, Profile: profile, Now: func() time.Time { return fixtureNow }}
		result := (deliveryramp.JourneyRunner{Driver: driver}).Run(context.Background(), deliveryramp.JourneyExecutionRequest{RunID: "run-2", Cell: deliveryramp.Cell{ID: cell.String(), Target: target}, Target: target, Artifact: artifact, Plan: deliveryramp.JourneyPlan{ID: "p", Capability: PlanCapability, Profile: profile.ID}})
		return result, recorder.records
	}
	accepted, records := run(&fakeExecutor{results: map[string]ExecutionResult{cell.CaseID: {OperationID: "op-2", AcceptedAt: fixtureNow, Outcome: evidence.DispositionPassed}}})
	if accepted.Disposition != deliveryramp.DispositionUnavailable || records[0].Disposition != evidence.DispositionUnavailable || !strings.Contains(records[0].Reason, "no target-owned receipt") {
		t.Fatalf("acceptance without receipts must be unavailable: %+v / %+v", accepted, records)
	}
	failed, records := run(&fakeExecutor{results: map[string]ExecutionResult{cell.CaseID: {OperationID: "op-3", Outcome: evidence.DispositionFailed, Reason: "endpoint served the predecessor", ReceiptRefs: []string{"cloud-target:op-3:verify"}}}})
	if failed.Disposition != deliveryramp.DispositionFailed || records[0].Disposition != evidence.DispositionFailed || failed.Steps[0].Error == "" {
		t.Fatalf("failed assertion must be failed: %+v", failed)
	}
	broken, records := run(&fakeExecutor{errs: map[string]error{cell.CaseID: errors.New("operation store unavailable")}})
	if broken.Disposition != deliveryramp.DispositionUnavailable || records[0].Disposition != evidence.DispositionUnavailable {
		t.Fatalf("pipeline failure must be unavailable: %+v", broken)
	}
	if err := records[0].Validate(profile); err != nil {
		t.Fatalf("unavailable record must still validate: %v", err)
	}
}

// TestDistributorRequiresTheTargetReceipt [REQ:STC-P0-035] proves the
// effect receipt is target-owned: an unreadable pointer is unavailable, a
// pointer naming another release is failed, and an unavailable target never
// activates (GOV-04, GOV-05 package lane).
func TestDistributorRequiresTheTargetReceipt(t *testing.T) {
	artifact := fixtureArtifact(t)
	target := deliveryramp.Target{ID: "machine:fixture-a", Available: true}
	executor := func() *fakeExecutor {
		return &fakeExecutor{results: map[string]ExecutionResult{PublishCase: {OperationID: "op-pub", Outcome: evidence.DispositionPassed}}}
	}
	unreadable, err := Distributor{Executor: executor(), Target: fakeTarget{err: errors.New("pointer unreadable")}}.Distribute(context.Background(), deliveryramp.DistributionRequest{Cell: deliveryramp.Cell{ID: "publish", Target: target}, Artifact: artifact})
	if err != nil || unreadable.Disposition != deliveryramp.DispositionUnavailable || unreadable.EffectReceipt != nil {
		t.Fatalf("unreadable pointer must be unavailable: %+v err=%v", unreadable, err)
	}
	mismatch, err := Distributor{Executor: executor(), Target: fakeTarget{receipt: evidence.TargetReceipt{ActiveRelease: fixtureOther}}}.Distribute(context.Background(), deliveryramp.DistributionRequest{Cell: deliveryramp.Cell{ID: "publish", Target: target}, Artifact: artifact})
	if err != nil || mismatch.Disposition != deliveryramp.DispositionFailed {
		t.Fatalf("mismatched pointer must be failed: %+v err=%v", mismatch, err)
	}
	exec := executor()
	unavailableTarget := target
	unavailableTarget.Available, unavailableTarget.Reason = false, "bridge enrollment revoked"
	skipped, err := Distributor{Executor: exec, Target: fakeTarget{}}.Distribute(context.Background(), deliveryramp.DistributionRequest{Cell: deliveryramp.Cell{ID: "publish", Target: unavailableTarget}, Artifact: artifact})
	if err != nil || skipped.Disposition != deliveryramp.DispositionUnavailable || len(exec.requests) != 0 {
		t.Fatalf("unavailable target must not activate: %+v err=%v requests=%d", skipped, err, len(exec.requests))
	}
}
