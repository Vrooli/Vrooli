package releases

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-ios/internal/builds"
	"scenario-to-ios/internal/distribution"
	"scenario-to-ios/internal/journeys"
	"scenario-to-ios/internal/targets"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeProber struct {
	inventory deliveryramp.Inventory
}

func (f fakeProber) Probe(context.Context, deliveryramp.ProbeRequest) (deliveryramp.Inventory, error) {
	return f.inventory, nil
}

func TestCatalogMapsProviderTargetsAndConformanceJourney(t *testing.T) {
	journeyPlan := journeys.Plan()
	catalog, err := (Catalog{Probe: fakeProber{inventory: deliveryramp.Inventory{Targets: []deliveryramp.Target{
		{ID: "ios:simulator:linux", Label: "Linux simulator", Available: false, Reason: "Apple tooling is unsupported", Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal}},
		{ID: "ios:macos:bridge-unavailable", Label: "macOS bridge", Available: false, Reason: "no bridge", Transport: deliveryramp.Transport{Kind: deliveryramp.TransportBridge}},
	}}}, Journey: validationmatrix.JourneySelection{JourneyID: journeyPlan.ID, DisplayName: "iOS generated-app conformance", Required: true}}).Resolve(context.Background(), "hello-mobile")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Journeys) != 1 || catalog.Journeys[0].JourneyID != journeys.Plan().ID {
		t.Fatalf("journey catalog = %+v", catalog.Journeys)
	}
	if len(catalog.Targets) != 2 || catalog.Targets[1].Kind != validationmatrix.TargetBridge {
		t.Fatalf("target catalog = %+v", catalog.Targets)
	}
	if catalog.Targets[0].Descriptor.GetAvailable() || catalog.Targets[0].Descriptor.GetReason() == "" {
		t.Fatalf("unsupported target lost its terminal reason: %+v", catalog.Targets[0].Descriptor)
	}
}

func TestExecutorPreservesUnavailableAppleBoundary(t *testing.T) {
	cell := &domainv1.ValidationCell{CellId: "cell-1", TargetId: "ios:simulator:linux", JourneyId: journeys.Plan().ID}
	result := (Executor{JourneyPlan: journeys.Plan()}).Execute(context.Background(), validationmatrix.CellRequest{
		RunID: "run-1", ArtifactDigest: "sha256:artifact", Cell: cell,
		Target: &domainv1.ValidationTargetDescriptor{TargetId: cell.TargetId, Available: false, Reason: stringPtr("iOS Simulator requires Apple tooling")},
	})
	if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE {
		t.Fatalf("disposition = %s, want unavailable", result.Disposition)
	}
	if result.Reason == "" {
		t.Fatal("unavailable result must carry a reason")
	}
}

func TestExecutorUsesInjectedJourneyRunnerForAvailableTarget(t *testing.T) {
	cell := &domainv1.ValidationCell{CellId: "cell-2", TargetId: "ios:macos:local", JourneyId: journeys.Plan().ID}
	executed := false
	result := (Executor{JourneyPlan: journeys.Plan(), RunJourney: func(_ context.Context, request deliveryramp.DriverRequest) (deliveryramp.JourneyResult, error) {
		executed = request.RunID == "run-2"
		return deliveryramp.JourneyResult{Disposition: deliveryramp.DispositionPass}, nil
	}}).Execute(context.Background(), validationmatrix.CellRequest{
		RunID: "run-2", ArtifactDigest: "sha256:artifact", Cell: cell,
		Target: &domainv1.ValidationTargetDescriptor{TargetId: cell.TargetId, DisplayName: "macOS", Available: true},
	})
	if !executed || result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS {
		t.Fatalf("executed=%t result=%+v", executed, result)
	}
	if len(result.Evidence) != 1 || result.Evidence[0].GetSha256() == "" {
		t.Fatalf("result evidence = %+v", result.Evidence)
	}
}

func TestIOSRampConformsAndFailsClosedWithoutAppleRuntime(t *testing.T) {
	var (
		_ deliveryramp.Prober      = targets.Prober{}
		_ deliveryramp.Builder     = builds.Builder{}
		_ deliveryramp.Driver      = journeys.Driver{}
		_ deliveryramp.Distributor = distribution.Distributor{}
	)

	prober := targets.Prober{GOOS: "linux", LookPath: func(string) (string, error) { return "", os.ErrNotExist }}
	inventory, err := prober.Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil || len(inventory.Targets) != 2 || inventory.Targets[0].Available {
		t.Fatalf("iOS probe = %+v, err=%v", inventory, err)
	}

	webRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<main>ios</main>"), 0o600))
	_, err = (builds.Builder{GOOS: "linux", BuildRoot: t.TempDir()}).Build(context.Background(), deliveryramp.BuildRequest{SourceRef: webRoot})
	var unavailable builds.UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("iOS build error = %v, want explicit Apple capability error", err)
	}

	journeyPlan := journeys.Plan()
	journey, err := (journeys.Driver{GOOS: "linux"}).Execute(context.Background(), deliveryramp.DriverRequest{RunID: "ios-run", Plan: journeyPlan})
	if err != nil || journey.Disposition != deliveryramp.DispositionUnavailable {
		t.Fatalf("iOS journey = %+v, err=%v", journey, err)
	}
	distributed, err := (distribution.Distributor{}).Distribute(context.Background(), deliveryramp.DistributionRequest{Artifact: deliveryramp.Artifact{ImmutableRef: "ios-artifact"}})
	if err != nil || distributed.Disposition != deliveryramp.DispositionUnavailable {
		t.Fatalf("iOS distribution = %+v, err=%v", distributed, err)
	}

	store, err := validationmatrix.NewFileStore(t.TempDir())
	require.NoError(t, err)
	service := validationmatrix.NewService(store, validationmatrix.Executors{Local: Executor{JourneyPlan: journeyPlan, RunJourney: (journeys.Driver{GOOS: "linux"}).Execute}})
	run, err := service.Create(validationmatrix.MatrixSelection{
		ScenarioName:        "hello-mobile",
		ArtifactDigest:      "sha256:ios-artifact",
		Journeys:            []validationmatrix.JourneySelection{{JourneyID: journeyPlan.ID, DisplayName: "iOS conformance", Required: true}},
		Targets:             []validationmatrix.TargetSelection{{Kind: validationmatrix.TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "ios:simulator:linux", DisplayName: "Linux iOS boundary", Available: true}}},
		EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL},
	})
	require.NoError(t, err)
	_, err = service.Start(run.RunID)
	require.NoError(t, err)
	completed, err := service.Wait(context.Background(), run.RunID)
	require.NoError(t, err)
	if completed.State != validationmatrix.RunFailed || completed.Gate.GetPassed() || completed.Cells[0].Cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE {
		t.Fatalf("iOS matrix must fail closed without Apple runtime: state=%s gate=%v cells=%+v", completed.State, completed.Gate, completed.Cells)
	}
}

func stringPtr(value string) *string { return &value }
