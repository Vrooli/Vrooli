package validationmatrix

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type localFunc func(context.Context, CellRequest) CellResult

func (f localFunc) Execute(ctx context.Context, request CellRequest) CellResult {
	return f(ctx, request)
}

type recordingTransport struct {
	calls int
}

const referenceCDPCapability = domainv1.ValidationTargetCapability(1)

func (t *recordingTransport) Execute(context.Context, CellRequest) CellResult {
	t.calls++
	return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Reason: "fake transport executed"}
}

func TestLocalAndBridgeCellsUseTheSameTransportEntryPoint(t *testing.T) {
	transport := &recordingTransport{}
	executors := Executors{Local: transport, Bridge: transport}
	for _, kind := range []TargetKind{TargetLocal, TargetBridge} {
		result := executors.Execute(context.Background(), kind, CellRequest{})
		if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS {
			t.Fatalf("%s result = %+v", kind, result)
		}
	}
	if transport.calls != 2 {
		t.Fatalf("transport calls = %d, want 2 through one Execute method", transport.calls)
	}
}

type recordingReporter struct {
	mu      sync.Mutex
	verdict ReleaseVerdict
	called  int
}

type catalogFunc func(context.Context, string) (CatalogSnapshot, error)

func (f catalogFunc) Resolve(ctx context.Context, scenario string) (CatalogSnapshot, error) {
	return f(ctx, scenario)
}

func TestCreateUsesProviderOwnedCatalogSnapshot(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selection := baseSelection()
	selection.Journeys = nil
	selection.Targets = nil
	service := NewService(store, Executors{}, WithCatalogResolver(catalogFunc(func(_ context.Context, scenario string) (CatalogSnapshot, error) {
		if scenario != "hello-desktop" {
			t.Fatalf("catalog resolved unexpected scenario %q", scenario)
		}
		return CatalogSnapshot{Journeys: baseSelection().Journeys, Targets: baseSelection().Targets}, nil
	})))
	run, err := service.Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Selection.Journeys) != 1 || len(run.Selection.Targets) != 1 || len(run.Cells) != 1 {
		t.Fatalf("provider catalog was not snapshotted: %+v", run.Selection)
	}
}

func TestCreatePreservesExplicitOperatorSelectionWhenCatalogIsAvailable(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selection := baseSelection()
	selection.Journeys = []JourneySelection{{JourneyID: "selected", DisplayName: "Selected journey", Required: true}}
	service := NewService(store, Executors{}, WithCatalogResolver(catalogFunc(func(_ context.Context, _ string) (CatalogSnapshot, error) {
		catalog := baseSelection()
		catalog.Journeys = []JourneySelection{
			{JourneyID: "selected", DisplayName: "Selected journey", Required: true},
			{JourneyID: "omitted", DisplayName: "Omitted provider journey", Required: false},
		}
		return CatalogSnapshot{Journeys: catalog.Journeys}, nil
	})))
	run, err := service.Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Selection.Journeys) != 1 || run.Selection.Journeys[0].JourneyID != "selected" {
		t.Fatalf("explicit selection was replaced by provider catalog: %+v", run.Selection.Journeys)
	}
}

func (r *recordingReporter) ReportValidationGate(_ context.Context, verdict ReleaseVerdict) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.called++
	r.verdict = verdict
	return nil
}

func TestCreatePersistsImmutableMatrixSelectionAndPreflightDispositions(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selection := MatrixSelection{
		ScenarioName:   "hello-desktop",
		ArtifactDigest: "sha256:artifact",
		Journeys:       []JourneySelection{{JourneyID: "journey", DisplayName: "Journey", Required: true, RequiredCapabilities: []domainv1.ValidationTargetCapability{referenceCDPCapability}}},
		Targets: []TargetSelection{
			{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "local", DisplayName: "Local", Available: true, Capabilities: []domainv1.ValidationTargetCapability{referenceCDPCapability}}},
			{Kind: TargetBridge, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "offline", DisplayName: "Offline", Available: false}},
		},
		EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL},
	}
	run, err := NewService(store, Executors{}).Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	if run.RunID == "" || run.Matrix.GetMatrixId() == "" {
		t.Fatal("expected durable run and matrix identities")
	}
	if run.Matrix.GetScenarioName() != selection.ScenarioName || run.Matrix.GetArtifactDigest() != selection.ArtifactDigest {
		t.Fatalf("immutable selection was not captured: %#v", run.Matrix)
	}
	if len(run.Cells) != 2 || len(run.Matrix.GetCells()) != 2 {
		t.Fatalf("expected one cell per journey/target/profile, got %d", len(run.Cells))
	}
	if run.Cells[0].Cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSPECIFIED {
		t.Fatalf("available cell should be queued: %v", run.Cells[0].Cell.GetDisposition())
	}
	if run.Cells[1].Cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE {
		t.Fatalf("unavailable target should be explicit: %v", run.Cells[1].Cell.GetDisposition())
	}
	reloaded, ok := store.Get(run.RunID)
	if !ok || reloaded.Matrix.GetArtifactDigest() != selection.ArtifactDigest {
		t.Fatal("matrix snapshot was not durable")
	}
}

func TestCreateIsIdempotentForSelectionKey(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selection := baseSelection()
	selection.IdempotencyKey = "same-request"
	service := NewService(store, Executors{})
	first, err := service.Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	if first.RunID != second.RunID || len(store.List()) != 1 {
		t.Fatalf("idempotency key created duplicate runs: first=%s second=%s count=%d", first.RunID, second.RunID, len(store.List()))
	}
}

func TestServiceRunsBoundedCellsRetriesAndReportsProvenance(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reporter := &recordingReporter{}
	var calls atomic.Int32
	service := NewService(store, Executors{Local: localFunc(func(_ context.Context, _ CellRequest) CellResult {
		if calls.Add(1) == 1 {
			return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: "transient", Retryable: true}
		}
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Evidence: validEvidence()}
	})}, WithReleaseReporter(reporter))
	run, err := service.Create(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(run.RunID); err != nil {
		t.Fatal(err)
	}
	completed, err := service.Wait(context.Background(), run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != RunCompleted || !completed.Gate.GetPassed() {
		t.Fatalf("expected passing terminal run, state=%s gate=%v", completed.State, completed.Gate)
	}
	if completed.Cells[0].Attempts != 2 || completed.Cells[0].State != CellCompleted {
		t.Fatalf("expected retry then completion: %+v", completed.Cells[0])
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	if reporter.called != 1 || reporter.verdict.RunID != run.RunID || reporter.verdict.MatrixID != run.Matrix.GetMatrixId() || reporter.verdict.ArtifactDigest != run.Matrix.GetArtifactDigest() {
		t.Fatalf("release report lost provenance: %+v", reporter.verdict)
	}
}

func TestServiceFailsClosedForUnsupportedAndUnavailableCells(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selection := baseSelection()
	selection.Targets = []TargetSelection{
		{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "unsupported", Available: true}},
		{Kind: TargetBridge, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "unavailable", Available: false}},
	}
	run, err := NewService(store, Executors{}).Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, Executors{})
	if _, err := service.Start(run.RunID); err != nil {
		t.Fatal(err)
	}
	finished, err := service.Wait(context.Background(), run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.State != RunFailed || finished.Gate.GetPassed() {
		t.Fatalf("incomplete required coverage passed: state=%s gate=%v", finished.State, finished.Gate)
	}
	if finished.Cells[0].Cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED || finished.Cells[1].Cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE {
		t.Fatalf("explicit non-pass dispositions were lost: %+v", finished.Cells)
	}
}

func TestComputeApplicabilityTruthTable(t *testing.T) {
	capability := referenceCDPCapability
	target := &domainv1.ValidationTargetDescriptor{TargetId: "local", Available: true, Capabilities: []domainv1.ValidationTargetCapability{capability, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_OFFLINE_NETWORK}}
	tests := []struct {
		name        string
		target      *domainv1.ValidationTargetDescriptor
		required    []domainv1.ValidationTargetCapability
		profile     domainv1.ValidationEnvironmentProfile
		disposition domainv1.ValidationDisposition
	}{
		{"eligible", target, []domainv1.ValidationTargetCapability{capability}, domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSPECIFIED},
		{"missing capability", &domainv1.ValidationTargetDescriptor{TargetId: "local", Available: true}, []domainv1.ValidationTargetCapability{capability}, domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED},
		{"unavailable", &domainv1.ValidationTargetDescriptor{TargetId: "remote", Available: false}, nil, domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE},
		{"offline unsupported until adapter exists", target, nil, domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_OFFLINE, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED},
		{"unspecified profile", target, nil, domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_UNSPECIFIED, domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			applicable, disposition, _ := ComputeApplicability(test.target, test.required, test.profile)
			if !applicable || disposition != test.disposition {
				t.Fatalf("ComputeApplicability() = %v, %v; want true, %v", applicable, disposition, test.disposition)
			}
		})
	}
}

func TestProfileContractsFailClosedForMissingCapabilities(t *testing.T) {
	contracts := ProfileContracts()
	if len(contracts) < 20 {
		t.Fatalf("profile contract catalog has %d entries; expected the complete profile inventory", len(contracts))
	}

	for _, contract := range contracts {
		if contract.Profile == domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL {
			continue
		}
		applicable, disposition, reason := ComputeApplicability(
			&domainv1.ValidationTargetDescriptor{TargetId: "local", Available: true},
			nil,
			contract.Profile,
		)
		if !applicable || disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED {
			t.Errorf("profile %s = applicable=%v disposition=%s; want applicable=true unsupported", contract.Profile, applicable, disposition)
		}
		if reason == nil || !strings.Contains(*reason, "required capability") {
			t.Errorf("profile %s reason = %v; want the missing capability", contract.Profile, reason)
		}
	}
}

func TestProfileContractsAcceptAdvertisedCapabilities(t *testing.T) {
	for _, contract := range ProfileContracts() {
		capabilities := append([]domainv1.ValidationTargetCapability(nil), contract.RequiredCapabilities...)
		applicable, disposition, reason := ComputeApplicability(
			&domainv1.ValidationTargetDescriptor{TargetId: "capable", Available: true, Capabilities: capabilities},
			nil,
			contract.Profile,
		)
		wantDisposition := domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED
		if contract.Executable {
			wantDisposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSPECIFIED
		}
		if !applicable || disposition != wantDisposition {
			t.Errorf("profile %s = applicable=%v disposition=%s; want applicable=true %s", contract.Profile, applicable, disposition, wantDisposition)
		}
		if contract.Executable && reason != nil {
			t.Errorf("profile %s reason = %v; want nil", contract.Profile, reason)
		} else if !contract.Executable && (reason == nil || !strings.Contains(*reason, "no executable adapter")) {
			t.Errorf("profile %s reason = %v; want no executable adapter", contract.Profile, reason)
		}
	}
}

func TestServicePersistsMixedTerminalDispositionsWithoutFalsePass(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selection := baseSelection()
	selection.Targets = []TargetSelection{
		{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "pass", Available: true, Capabilities: []domainv1.ValidationTargetCapability{referenceCDPCapability}}},
		{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "failed", Available: true, Capabilities: []domainv1.ValidationTargetCapability{referenceCDPCapability}}},
		{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "refused", Available: true, Capabilities: []domainv1.ValidationTargetCapability{referenceCDPCapability}}},
		{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "unsupported", Available: true}},
		{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "unavailable", Available: false}},
	}
	service := NewService(store, Executors{Local: localFunc(func(_ context.Context, request CellRequest) CellResult {
		switch request.Cell.GetTargetId() {
		case "pass":
			return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Evidence: validEvidence()}
		case "refused":
			return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED, Reason: "provider refused the requested profile"}
		default:
			return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: "cell failed"}
		}
	})})
	run, err := service.Create(selection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(run.RunID); err != nil {
		t.Fatal(err)
	}
	finished, err := service.Wait(context.Background(), run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.State != RunFailed || finished.Gate.GetPassed() || finished.Gate.GetPassingCellCount() != 1 || finished.Gate.GetRequiredCellCount() != 5 {
		for _, cell := range finished.Cells {
			t.Logf("cell target=%s disposition=%s state=%s evidence=%d", cell.Cell.GetTargetId(), cell.Cell.GetDisposition(), cell.State, len(cell.Cell.GetEvidence()))
		}
		t.Fatalf("mixed dispositions produced an untruthful gate: state=%s gate=%v", finished.State, finished.Gate)
	}
	byTarget := make(map[string]domainv1.ValidationDisposition, len(finished.Cells))
	for _, cell := range finished.Cells {
		byTarget[cell.Cell.GetTargetId()] = cell.Cell.GetDisposition()
	}
	if byTarget["refused"] != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED || byTarget["unsupported"] != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSUPPORTED || byTarget["unavailable"] != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE {
		t.Fatalf("typed dispositions were collapsed: %v", byTarget)
	}
}

func TestRerunSelectorCreatesIndependentRun(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, Executors{})
	original, err := service.Create(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	original.IdempotencyKey = "original-request"
	if err := store.Save(original); err != nil {
		t.Fatal(err)
	}
	original.Cells[0].State = CellFailed
	original.Cells[0].Cell.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED
	if err := store.Save(original); err != nil {
		t.Fatal(err)
	}
	rerun, err := service.Rerun(original.RunID, RerunSelector{Kind: RerunFailed})
	if err != nil {
		t.Fatal(err)
	}
	if rerun.RunID == original.RunID || rerun.ParentRunID != original.RunID {
		t.Fatalf("rerun did not get independent provenance: %#v", rerun)
	}
	if rerun.Cells[0].State != CellQueued || rerun.Cells[0].Cell.GetCellId() == original.Cells[0].Cell.GetCellId() {
		t.Fatal("failed selector did not reset only the selected cell")
	}
	unchanged, _ := store.Get(original.RunID)
	if unchanged.Cells[0].State != CellFailed {
		t.Fatal("rerun overwrote prior evidence")
	}
}

func TestCompareRunsByStableCellIdentity(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, Executors{})
	first, err := service.Create(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(MatrixSelection{
		ScenarioName:        "hello-desktop",
		ArtifactDigest:      "sha256:new-artifact",
		Journeys:            baseSelection().Journeys,
		Targets:             baseSelection().Targets,
		EnvironmentProfiles: baseSelection().EnvironmentProfiles,
	})
	if err != nil {
		t.Fatal(err)
	}
	second.Cells[0].Cell.Disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED
	if err := store.Save(second); err != nil {
		t.Fatal(err)
	}
	comparison, err := service.Compare(second.RunID, first.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !comparison.Changed || len(comparison.Cells) != 1 || comparison.CurrentRunID != second.RunID || comparison.PriorRunID != first.RunID {
		t.Fatalf("unexpected comparison: %+v", comparison)
	}
	if comparison.Cells[0].PriorDisposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNSPECIFIED || comparison.Cells[0].CurrentDisposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED {
		t.Fatalf("comparison lost dispositions: %+v", comparison.Cells[0])
	}
}

func TestAbortAndReattachRemainServerOwned(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	service := NewService(store, Executors{Local: localFunc(func(ctx context.Context, _ CellRequest) CellResult {
		close(started)
		<-ctx.Done()
		return CellResult{}
	})})
	run, err := service.Create(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(run.RunID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("cell did not start")
	}
	if err := service.Abort(run.RunID); err != nil {
		t.Fatal(err)
	}
	finished, err := service.Wait(context.Background(), run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.State != RunCancelled || finished.Cells[0].State != CellCancelled {
		t.Fatalf("abort did not reach durable terminal state: %+v", finished)
	}
	reattached := NewService(store, Executors{})
	reconnected, err := reattached.Wait(context.Background(), run.RunID)
	if err != nil || reconnected.State != RunCancelled {
		t.Fatalf("reconnect did not observe durable result: state=%s err=%v", reconnected.State, err)
	}
}

func TestClientDisconnectDoesNotCancelServerOwnedRun(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	service := NewService(store, Executors{Local: localFunc(func(_ context.Context, _ CellRequest) CellResult {
		close(started)
		<-release
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Evidence: validEvidence()}
	})})
	run, err := service.Create(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(run.RunID); err != nil {
		t.Fatal(err)
	}
	<-started
	clientCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Wait(clientCtx, run.RunID); err == nil {
		t.Fatal("disconnected client should stop waiting locally")
	}
	close(release)
	finished, err := service.Wait(context.Background(), run.RunID)
	if err != nil || finished.State != RunCompleted {
		t.Fatalf("server-owned run was cancelled by client disconnect: state=%s err=%v", finished.State, err)
	}
}

func TestAbortComputesFailClosedGateAndReports(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reporter := &recordingReporter{}
	started := make(chan struct{})
	service := NewService(store, Executors{Local: localFunc(func(ctx context.Context, _ CellRequest) CellResult {
		close(started)
		<-ctx.Done()
		return CellResult{}
	})}, WithReleaseReporter(reporter))
	run, err := service.Create(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(run.RunID); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := service.Abort(run.RunID); err != nil {
		t.Fatal(err)
	}
	finished, err := service.Wait(context.Background(), run.RunID)
	if err != nil || finished.Gate == nil || finished.Gate.GetPassed() {
		t.Fatalf("aborted run did not publish fail-closed gate: state=%s gate=%v err=%v", finished.State, finished.Gate, err)
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	if reporter.called != 1 || reporter.verdict.Gate == nil || reporter.verdict.Gate.GetPassed() {
		t.Fatalf("aborted run was not reported as failed: %+v", reporter.verdict)
	}
}

func TestMarkStaleFailsRunningCellsClosed(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, Executors{})
	run, err := service.Create(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	run.State = RunRunning
	run.Cells[0].State = CellRunning
	run.Cells[0].Attempts = 1
	if err := store.Save(run); err != nil {
		t.Fatal(err)
	}
	if err := service.MarkStale(run.RunID); err != nil {
		t.Fatal(err)
	}
	stale, _ := store.Get(run.RunID)
	if stale.State != RunFailed || stale.Cells[0].Cell.GetDisposition() != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED {
		t.Fatalf("stale run was not fail-closed: %+v", stale)
	}
}

func baseSelection() MatrixSelection {
	return MatrixSelection{
		ScenarioName:        "hello-desktop",
		ArtifactDigest:      "sha256:artifact",
		Journeys:            []JourneySelection{{JourneyID: "journey", DisplayName: "Journey", Required: true, RequiredCapabilities: []domainv1.ValidationTargetCapability{referenceCDPCapability}}},
		Targets:             []TargetSelection{{Kind: TargetLocal, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "local", DisplayName: "Local", Available: true, Capabilities: []domainv1.ValidationTargetCapability{referenceCDPCapability}}}},
		EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL},
	}
}

func validEvidence() []*domainv1.LayeredEvidence {
	return []*domainv1.LayeredEvidence{
		{Kind: domainv1.LayeredEvidence_KIND_BAS_WORKFLOW, EvidenceId: "workflow", Uri: "file:///workflow.json", Sha256: "sha256:workflow", Redacted: true},
		{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: "desktop", Uri: "file:///desktop.json", Sha256: "sha256:desktop", Redacted: true},
		{Kind: domainv1.LayeredEvidence_KIND_TARGET, EvidenceId: "target", Uri: "file:///target.json", Sha256: "sha256:target", Redacted: true},
		{Kind: domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, EvidenceId: "machine", Uri: "file:///machine.json", Sha256: "sha256:machine", Redacted: true},
	}
}
