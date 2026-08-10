package validationmatrix

import (
	"context"
	"testing"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

// referenceRamp is intentionally test-only. It is a second implementer of
// the spine contract, proving that the exported adapters are sufficient
// without importing a scenario package or reaching into an internal package.
type referenceRamp struct{}

func (referenceRamp) Probe(context.Context, deliveryramp.ProbeRequest) (deliveryramp.Inventory, error) {
	return deliveryramp.Inventory{Observed: time.Now().UTC(), Targets: []deliveryramp.Target{
		{ID: "reference-host", Ramp: "reference-ramp", Label: "Reference host", Platform: "reference", OS: "linux", DeviceKind: "host", Available: true, Capabilities: []string{"reference.launch"}, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "reference-host", Available: true}},
		{ID: "reference-missing", Ramp: "reference-ramp", Label: "Missing capability", Platform: "reference", OS: "linux", DeviceKind: "host", Available: false, MissingCapability: "reference.capture", NextAction: "enable reference capture"},
	}}, nil
}

func (referenceRamp) Build(context.Context, deliveryramp.BuildRequest) (deliveryramp.Artifact, error) {
	return deliveryramp.Artifact{ImmutableRef: "artifact:reference", Kind: "bundle", Checksum: "sha256:reference", SizeBytes: 1, UsefulFrames: true, CreatedAt: time.Now().UTC()}, nil
}

func (referenceRamp) Execute(context.Context, deliveryramp.DriverRequest) (deliveryramp.JourneyResult, error) {
	return deliveryramp.JourneyResult{SchemaVersion: deliveryramp.JourneySchemaVersion, EvidenceVersion: deliveryramp.JourneyEvidenceVersion, SmokeTestID: "reference-run", Capability: "reference.launch", PlanID: "reference-plan", Profile: "reference", Disposition: deliveryramp.DispositionPass, Steps: []deliveryramp.JourneyStep{{ID: "launch", Name: "launch", Action: "launch", Disposition: deliveryramp.StepPassed, Evidence: []deliveryramp.EvidenceReference{{ID: "reference-journey", Kind: "journey", URI: "capture://reference-journey", Checksum: "sha256:journey", Redacted: true}}}}}, nil
}

func (referenceRamp) Distribute(context.Context, deliveryramp.DistributionRequest) (deliveryramp.DistributionResult, error) {
	return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionPass, Targets: []deliveryramp.DistributionTarget{{ID: "reference-local", Kind: "local", Available: true}}}, nil
}

type referenceTransport struct{ driver deliveryramp.Driver }

func (t referenceTransport) Execute(ctx context.Context, request CellRequest) CellResult {
	target := deliveryramp.Target{ID: request.Cell.GetTargetId(), Ramp: "reference-ramp", Platform: "reference", OS: "linux", DeviceKind: "host", Available: true, Capabilities: []string{"reference.launch"}, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: request.Cell.GetTargetId(), Available: true}}
	result := (deliveryramp.JourneyRunner{Driver: t.driver}).Run(ctx, deliveryramp.JourneyExecutionRequest{RunID: request.RunID, Cell: deliveryramp.Cell{ID: request.Cell.GetCellId(), Target: target, Capability: "reference.launch"}, Target: target, Plan: deliveryramp.JourneyPlan{ID: "reference-plan", Capability: "reference.launch", Profile: "reference"}})
	if result.Disposition != deliveryramp.DispositionPass {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: result.DegradedReason}
	}
	return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Reason: "reference driver passed", Evidence: validEvidence()}
}

func TestReferenceRampConformsThroughExportedSpineSeams(t *testing.T) {
	var ramp referenceRamp
	var _ deliveryramp.Prober = ramp
	var _ deliveryramp.Builder = ramp
	var _ deliveryramp.Driver = ramp
	var _ deliveryramp.Distributor = ramp

	inventory, err := ramp.Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil || len(inventory.Targets) != 2 || inventory.Targets[1].Available {
		t.Fatalf("reference probe = %+v, err=%v", inventory, err)
	}
	if _, err := ramp.Build(context.Background(), deliveryramp.BuildRequest{}); err != nil {
		t.Fatal(err)
	}
	if result, err := ramp.Distribute(context.Background(), deliveryramp.DistributionRequest{}); err != nil || result.Disposition != deliveryramp.DispositionPass {
		t.Fatalf("reference distribution = %+v, err=%v", result, err)
	}

	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, Executors{Local: referenceTransport{driver: ramp}})
	selection := MatrixSelection{ScenarioName: "reference-ramp", ArtifactDigest: "sha256:reference", Journeys: []JourneySelection{{JourneyID: "reference", DisplayName: "Reference", Required: true}}, Targets: []TargetSelection{{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "reference-host", DisplayName: "Reference host", Available: true}}}, EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL}}
	run, err := service.Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(run.RunID); err != nil {
		t.Fatal(err)
	}
	completed, err := service.Wait(context.Background(), run.RunID)
	if err != nil || completed.State != RunCompleted || !completed.Gate.GetPassed() {
		t.Fatalf("reference matrix = %+v, err=%v", completed, err)
	}
	priorEvidence := completed.Cells[0].Cell.GetEvidence()[0].GetSha256()
	rerun, err := service.Rerun(run.RunID, RerunSelector{Kind: RerunAll})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(rerun.RunID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Wait(context.Background(), rerun.RunID); err != nil {
		t.Fatal(err)
	}
	comparison, err := service.Compare(rerun.RunID, run.RunID)
	if err != nil || comparison.Changed {
		t.Fatalf("reference comparison = %+v, err=%v", comparison, err)
	}
	unchanged, _ := store.Get(run.RunID)
	if unchanged.Cells[0].Cell.GetEvidence()[0].GetSha256() != priorEvidence {
		t.Fatal("rerun mutated prior evidence")
	}

	blocking := &referenceBlockingTransport{started: make(chan struct{})}
	abortService := NewService(store, Executors{Local: blocking})
	abortRun, err := abortService.Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := abortService.Start(abortRun.RunID); err != nil {
		t.Fatal(err)
	}
	<-blocking.started
	if err := abortService.Abort(abortRun.RunID); err != nil {
		t.Fatal(err)
	}
	aborted, err := abortService.Wait(context.Background(), abortRun.RunID)
	if err != nil || aborted.State != RunCancelled {
		t.Fatalf("reference abort = %+v, err=%v", aborted, err)
	}

	for _, target := range []deliveryramp.Target{
		{ID: "host", Ramp: "reference-ramp", Platform: "reference", OS: "linux", DeviceKind: "host", Available: true, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal}},
		{ID: "emulator", Ramp: "reference-ramp", Platform: "reference", OS: "android", DeviceKind: "emulator", Available: true, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal}},
		{ID: "bridge-node", Ramp: "reference-ramp", Platform: "reference", OS: "linux", DeviceKind: "physical", NodeID: "node-1", Available: true, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportBridge}},
	} {
		verdict, err := deliveryramp.NewTargetVerdict(deliveryramp.TargetVerdictInput{Producer: "reference-ramp", Target: target, Disposition: deliveryramp.DispositionPass, RunID: "reference-run", CreatedAt: time.Now().UTC(), References: []deliveryramp.EvidenceReference{{ID: "reference-journey", Kind: "journey", Checksum: "sha256:journey", Redacted: true}}})
		if err != nil || verdict.GetDisposition() != commonv1.Disposition_DISPOSITION_PASSED || len(verdict.GetRefs()) != 1 {
			t.Fatalf("reference verdict = %+v, err=%v", verdict, err)
		}
	}
}

type referenceBlockingTransport struct{ started chan struct{} }

func (t *referenceBlockingTransport) Execute(ctx context.Context, _ CellRequest) CellResult {
	close(t.started)
	<-ctx.Done()
	return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "reference run cancelled"}
}
