package validationbroker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	sharedruns "test-genie/internal/shared/runs"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestGCTAttachmentTimeoutRetainsChildAndReattachesWithoutRecapture(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(testsqllite(t))
	intent := validIntent("plan-manager", "attachment")
	intent.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_REGRESSION_BEFORE
	intent.BehavioralPrior = "prior"
	intent.EvidencePolicy = &validationv1.EvidencePolicy{RequireBehavioralBefore: true, RequiredEvidenceKinds: []string{"gct-baseline-collection"}}
	admitted, err := repo.Admit(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	gct := &fakeGCTEvidence{waitErr: context.DeadlineExceeded}
	producer := NewRunProducer(&fakeSuiteRuns{}).WithGCTEvidence(gct)
	err = producer.ExecuteValidation(ctx, admitted.Receipt, intent, repo.Transition)
	if !errors.Is(err, errEvidencePending) {
		t.Fatalf("attachment failure = %v", err)
	}
	retained, err := repo.Get(ctx, admitted.Receipt.GetReceiptId())
	if err != nil || terminal(retained.GetState()) || len(retained.GetChildren()) != 1 {
		t.Fatalf("lost active evidence: %v %v", retained, err)
	}
	gct.waitErr = nil
	gct.result = GCTEvidenceResult{Passed: true, Evidence: []*validationv1.EvidenceReference{{EvidenceId: "prior", Kind: "gct-baseline-collection"}}}
	if err := producer.ExecuteValidation(ctx, retained, intent, repo.Transition); err != nil {
		t.Fatal(err)
	}
	final, err := repo.Get(ctx, retained.GetReceiptId())
	if err != nil || final.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(gct.captures) != 1 || len(gct.waits) != 2 {
		t.Fatalf("reattachment = %v %v captures=%d waits=%d", final, err, len(gct.captures), len(gct.waits))
	}
}

func TestBehavioralCaptureChecksIdentityAfterGCTCompletes(t *testing.T) {
	runs := &fakeSuiteRuns{}
	intent := validIntent("plan-manager", "moving-before")
	intent.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_REGRESSION_BEFORE
	intent.BehavioralPrior = "before"
	intent.EvidencePolicy = &validationv1.EvidencePolicy{RequireBehavioralBefore: true, RequiredEvidenceKinds: []string{"gct-baseline-collection"}}
	changed := proto.Clone(intent.ExpectedIdentity).(*validationv1.SourceIdentity)
	changed.Identity = "ci:v1:changed"
	gct := &fakeGCTEvidence{result: GCTEvidenceResult{Passed: true, Evidence: []*validationv1.EvidenceReference{{EvidenceId: "before", Kind: "gct-baseline-collection"}}}}
	producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{intent.ExpectedIdentity, changed}}).WithGCTEvidence(gct)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	created, err := service.CreateValidation(context.Background(), connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
	if err != nil {
		t.Fatal(err)
	}
	final := waitForTerminalReceipt(t, service, created.Msg.GetReceipt())
	if final.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_IDENTITY_CHANGED || len(final.GetEvidence()) != 1 {
		t.Fatalf("capture incorrectly certified: %v", final)
	}
}

func TestExplicitPhaseSelectionReachesSuiteOwner(t *testing.T) {
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	producer := NewRunProducer(runs)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	intent := validIntent("plan-manager", "selected-checks")
	intent.Phases = []string{"unit", "contracts"}
	created, err := service.CreateValidation(context.Background(), connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
	if err != nil {
		t.Fatal(err)
	}
	final := waitForTerminalReceipt(t, service, created.Msg.GetReceipt())
	if final.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(runs.starts) != 1 || fmt.Sprint(runs.starts[0].Input.Request.Phases) != "[contracts unit]" {
		t.Fatalf("phase selection lost: %v %#v", final, runs.starts)
	}
	intent.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_CERTIFICATION
	if _, err := normalizeIntent(intent); !errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("narrow certification accepted: %v", err)
	}
	intent.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_PHASE
	for _, strength := range []validationv1.ValidationStrength{validationv1.ValidationStrength_VALIDATION_STRENGTH_COMPREHENSIVE, validationv1.ValidationStrength_VALIDATION_STRENGTH_CERTIFICATION} {
		intent.RequiredStrength = strength
		if _, err := normalizeIntent(intent); !errors.Is(err, ErrInvalidIntent) {
			t.Fatalf("selected checks claimed %s: %v", strength, err)
		}
	}
}

type fakeSuiteRuns struct {
	mu          sync.Mutex
	starts      []runmanager.StartOptions
	aborts      []string
	waits       []string
	wait        runmanager.LiveStatus
	results     []runmanager.LiveStatus
	waitErr     error
	startErrors []error
}

type sequenceIdentityResolver struct {
	values []*validationv1.SourceIdentity
	next   atomic.Int32
}

type abortRaceRuns struct {
	waiting chan struct{}
	done    chan struct{}
	once    sync.Once
}

type fakeGCTEvidence struct {
	captures []GCTCaptureRequest
	diffs    []GCTDiffRequest
	waits    []string
	result   GCTEvidenceResult
	startErr error
	waitErr  error
}

func (f *fakeGCTEvidence) StartCapture(_ context.Context, req GCTCaptureRequest) (string, error) {
	f.captures = append(f.captures, req)
	return req.Collection, f.startErr
}

func (f *fakeGCTEvidence) WaitCapture(_ context.Context, req GCTCaptureRequest) (GCTEvidenceResult, error) {
	f.waits = append(f.waits, "capture:"+req.Collection)
	return f.result, f.waitErr
}

func (f *fakeGCTEvidence) StartDiff(_ context.Context, req GCTDiffRequest) (string, error) {
	f.diffs = append(f.diffs, req)
	return req.OperationID, f.startErr
}

func (f *fakeGCTEvidence) WaitDiff(_ context.Context, req GCTDiffRequest) (GCTEvidenceResult, error) {
	f.waits = append(f.waits, "diff:"+req.OperationID)
	return f.result, f.waitErr
}

func (r *abortRaceRuns) Start(runmanager.StartOptions) (runmanager.StartResult, error) {
	return runmanager.StartResult{RunID: "abort-run"}, nil
}

func (r *abortRaceRuns) Wait(context.Context, string, string) (runmanager.LiveStatus, error) {
	close(r.waiting)
	<-r.done
	return runmanager.LiveStatus{Status: sharedruns.StatusAborted}, nil
}

func (r *abortRaceRuns) Abort(string, string) (runmanager.LiveStatus, error) {
	r.once.Do(func() { close(r.done) })
	return runmanager.LiveStatus{Status: sharedruns.StatusAborted}, nil
}

func (r *sequenceIdentityResolver) Resolve(context.Context, *validationv1.ValidationIntent) (*validationv1.SourceIdentity, error) {
	index := int(r.next.Add(1)) - 1
	if index >= len(r.values) {
		index = len(r.values) - 1
	}
	return proto.Clone(r.values[index]).(*validationv1.SourceIdentity), nil
}

func (f *fakeSuiteRuns) Start(options runmanager.StartOptions) (runmanager.StartResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts = append(f.starts, options)
	if index := len(f.starts) - 1; index < len(f.startErrors) && f.startErrors[index] != nil {
		return runmanager.StartResult{}, f.startErrors[index]
	}
	return runmanager.StartResult{RunID: fmt.Sprintf("run-%d", len(f.starts))}, nil
}

func (f *fakeSuiteRuns) Wait(context.Context, string, string) (runmanager.LiveStatus, error) {
	f.mu.Lock()
	index := len(f.waits)
	f.waits = append(f.waits, "wait")
	status := f.wait
	if index < len(f.results) {
		status = f.results[index]
	}
	f.mu.Unlock()
	return status, f.waitErr
}

func (f *fakeSuiteRuns) Abort(scenario, runID string) (runmanager.LiveStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.aborts = append(f.aborts, scenario+":"+runID)
	return runmanager.LiveStatus{Status: sharedruns.StatusAborted}, nil
}

func TestCreateValidationDrivesExactlyOneDurableSuiteProducer(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	producer := NewRunProducer(runs)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)

	created := createThroughService(t, service, "driven-work")
	receipt := created
	for sequence := 0; sequence < 6 && !terminal(receipt.GetState()); sequence++ {
		response, err := service.WaitValidation(context.Background(), connect.NewRequest(&validationv1.WaitValidationRequest{
			ReceiptId: receipt.GetReceiptId(), WaitId: fmt.Sprintf("producer-observer-%d", sequence), AfterRevision: receipt.GetRevision(), Timeout: durationpb.New(time.Second),
		}))
		if err != nil {
			t.Fatal(err)
		}
		receipt = response.Msg.GetReceipt()
	}
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(receipt.GetChildren()) != 1 || len(receipt.GetEvidence()) != 1 {
		t.Fatalf("terminal receipt = %#v", receipt)
	}
	if len(runs.starts) != 1 || runs.starts[0].Input.Request.Preset != "quick" || !runs.starts[0].Input.Request.RetainForEvidence {
		t.Fatalf("suite starts = %#v", runs.starts)
	}
}

func TestReceiptProducerResolvesAdaptivePresetThroughSuitePlanner(t *testing.T) {
	runs := &fakeSuiteRuns{}
	planner := identityPlannerFunc(func(request orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error) {
		if request.Preset != "quick" {
			t.Fatalf("unexpected preset: %s", request.Preset)
		}
		return &execution.ExecutionPlanPreview{ConfigurationFingerprint: "owner-config", Phases: []execution.PlannedPhase{{Name: "unit"}}}, nil
	})
	producer := NewRunProducer(runs).WithExecutionPlanner(planner)
	if _, err := producer.startAfterCapacity(context.Background(), "receipt", "demo", validIntent("plan-manager", "planned-child")); err != nil {
		t.Fatal(err)
	}
	if len(runs.starts) != 1 || len(runs.starts[0].Input.Request.ResolvedPhases) != 1 || runs.starts[0].Input.Request.ResolvedPhases[0] != "unit" {
		t.Fatalf("adaptive preset reached executor unresolved: %+v", runs.starts)
	}
}

func TestRegressionBeforeReceiptOwnsGCTCaptureAndSkipsCurrentSuite(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	gct := &fakeGCTEvidence{result: GCTEvidenceResult{Passed: true, Detail: "complete", Evidence: []*validationv1.EvidenceReference{{EvidenceId: "before", Kind: "gct-baseline-collection", Owner: "git-control-tower"}, {EvidenceId: "paths-before", Kind: "gct-source-snapshot", Owner: "git-control-tower"}}}}
	producer := NewRunProducer(runs).WithGCTEvidence(gct)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	intent := validIntent("plan-manager", "gct-before")
	intent.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_REGRESSION_BEFORE
	intent.EvidencePolicy = &validationv1.EvidencePolicy{RequireBehavioralBefore: true, RequireSourceSnapshot: true, RequiredEvidenceKinds: []string{"gct-baseline-collection", "gct-source-snapshot"}}
	intent.CallerAttributes = map[string]string{"baseline_name": "before"}
	intent.ContentInputs = append(intent.ContentInputs, &validationv1.ContentInputRoot{Name: "plan-boundary", Root: ".", Dependency: true, Selections: []*validationv1.InputSelection{{Glob: "packages/proto/**"}}})

	response, err := service.CreateValidation(context.Background(), connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
	if err != nil {
		t.Fatal(err)
	}
	receipt := waitForTerminalReceipt(t, service, response.Msg.GetReceipt())
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(receipt.GetChildren()) != 2 || receipt.GetChildren()[0].GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_GCT_BASELINE_COLLECTION || receipt.GetChildren()[1].GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_SOURCE_SNAPSHOT {
		t.Fatalf("before receipt = %#v", receipt)
	}
	if len(runs.starts) != 0 || len(gct.captures) != 1 || len(gct.waits) != 1 || len(receipt.GetEvidence()) != 2 {
		t.Fatalf("suite starts=%d captures=%#v waits=%v evidence=%#v", len(runs.starts), gct.captures, gct.waits, receipt.GetEvidence())
	}
	if got := gct.captures[0].Paths; len(got) != 1 || got[0] != "packages/proto/**" {
		t.Fatalf("source selections = %v", got)
	}
	if gct.captures[0].ParentReceiptID != response.Msg.GetReceipt().GetReceiptId() {
		t.Fatalf("capture parent receipt = %q, want %q", gct.captures[0].ParentReceiptID, response.Msg.GetReceipt().GetReceiptId())
	}
}

func TestRegressionBeforeReceiptRetainsDistinctEvidenceKindsWithSameID(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	providerID := "shared-provider-id"
	gct := &fakeGCTEvidence{result: GCTEvidenceResult{
		Passed: true,
		Detail: "complete",
		Evidence: []*validationv1.EvidenceReference{
			{EvidenceId: providerID, Kind: "gct-baseline-collection", Owner: "git-control-tower"},
			{EvidenceId: providerID, Kind: "gct-source-snapshot", Owner: "git-control-tower"},
		},
	}}
	producer := NewRunProducer(&fakeSuiteRuns{}).WithGCTEvidence(gct)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	intent := validIntent("plan-manager", "gct-same-evidence-id")
	intent.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_REGRESSION_BEFORE
	intent.EvidencePolicy = &validationv1.EvidencePolicy{RequireBehavioralBefore: true, RequireSourceSnapshot: true, RequiredEvidenceKinds: []string{"gct-baseline-collection", "gct-source-snapshot"}}
	intent.CallerAttributes = map[string]string{"baseline_name": providerID}

	response, err := service.CreateValidation(context.Background(), connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
	if err != nil {
		t.Fatal(err)
	}
	receipt := waitForTerminalReceipt(t, service, response.Msg.GetReceipt())
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(receipt.GetChildren()) != 2 || len(receipt.GetEvidence()) != 2 {
		t.Fatalf("same-ID evidence receipt = %#v", receipt)
	}
	if receipt.GetEvidence()[0].GetKind() != "gct-baseline-collection" || receipt.GetEvidence()[1].GetKind() != "gct-source-snapshot" {
		t.Fatalf("same-ID evidence kinds = %#v", receipt.GetEvidence())
	}
}

func TestPhaseReceiptOwnsGCTDiffBeforeCurrentSuite(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	gct := &fakeGCTEvidence{result: GCTEvidenceResult{Passed: true, Detail: "clean", Evidence: []*validationv1.EvidenceReference{{EvidenceId: "diff-1", Kind: "gct-collection-diff", Owner: "git-control-tower"}}}}
	producer := NewRunProducer(runs).WithGCTEvidence(gct)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	intent := validIntent("plan-manager", "gct-current")
	intent.EvidencePolicy = &validationv1.EvidencePolicy{RequireBehavioralBefore: true, RequiredEvidenceKinds: []string{"gct-collection-diff", "test-genie-run"}}
	intent.CallerAttributes = map[string]string{"baseline_name": "before"}

	response, err := service.CreateValidation(context.Background(), connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
	if err != nil {
		t.Fatal(err)
	}
	receipt := waitForTerminalReceipt(t, service, response.Msg.GetReceipt())
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(receipt.GetChildren()) != 2 {
		t.Fatalf("current receipt = %#v", receipt)
	}
	if receipt.GetChildren()[0].GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_GCT_COLLECTION_DIFF || receipt.GetChildren()[1].GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN {
		t.Fatalf("child ordering = %#v", receipt.GetChildren())
	}
	if len(gct.diffs) != 1 || len(runs.starts) != 1 || len(receipt.GetEvidence()) != 2 {
		t.Fatalf("diffs=%#v suite starts=%d evidence=%#v", gct.diffs, len(runs.starts), receipt.GetEvidence())
	}
	if gct.diffs[0].ParentReceiptID != response.Msg.GetReceipt().GetReceiptId() {
		t.Fatalf("diff parent receipt = %q, want %q", gct.diffs[0].ParentReceiptID, response.Msg.GetReceipt().GetReceiptId())
	}
}

func TestMissingGCTEvidenceFailsClosedUnlessExplicitlyDegraded(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	for _, test := range []struct {
		name       string
		authorized bool
		want       validationv1.ReceiptState
	}{{"required", false, validationv1.ReceiptState_RECEIPT_STATE_FAILED}, {"authorized", true, validationv1.ReceiptState_RECEIPT_STATE_DEGRADED}} {
		t.Run(test.name, func(t *testing.T) {
			producer := NewRunProducer(&fakeSuiteRuns{})
			service := NewService(NewRepository(testsqllite(t)), producer)
			service.SetProducer(producer)
			intent := validIntent("plan-manager", "missing-gct-"+test.name)
			intent.EvidencePolicy = &validationv1.EvidencePolicy{RequireBehavioralBefore: true, AllowAuthorizedDegradation: test.authorized}
			intent.CallerAttributes = map[string]string{"baseline_name": "before"}
			response, err := service.CreateValidation(context.Background(), connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
			if err != nil {
				t.Fatal(err)
			}
			receipt := waitForTerminalReceipt(t, service, response.Msg.GetReceipt())
			if receipt.GetState() != test.want || receipt.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_MISSING {
				t.Fatalf("receipt = %#v", receipt)
			}
		})
	}
}

func TestAbortValidationRoutesToActiveSuiteChildren(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	runs := &fakeSuiteRuns{}
	producer := NewRunProducer(runs)
	err := producer.AbortValidation(context.Background(), &validationv1.ValidationReceipt{Children: []*validationv1.ChildOperation{{
		ChildId: "scenario:demo:1", Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, OperationId: "run-7",
	}}}, "operator", "reason")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs.aborts) != 1 || runs.aborts[0] != "demo:run-7" {
		t.Fatalf("aborts = %v", runs.aborts)
	}
}

func TestMidRunIdentityChangeRetainsEvidenceButFailsReceipt(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0] [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	expected := validIntent("plan-manager", "identity-moved").GetExpectedIdentity()
	changed := proto.Clone(expected).(*validationv1.SourceIdentity)
	changed.Identity = "ci:v1:changed"
	producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{expected, changed}})
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	receipt := createThroughService(t, service, "identity-moved")
	receipt = waitForTerminalReceipt(t, service, receipt)
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED || receipt.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_IDENTITY_CHANGED || receipt.GetObservedIdentity().GetIdentity() != "ci:v1:changed" || len(receipt.GetEvidence()) != 1 {
		t.Fatalf("identity-change receipt = %#v", receipt)
	}
}

func waitForTerminalReceipt(t *testing.T, service *Service, receipt *validationv1.ValidationReceipt) *validationv1.ValidationReceipt {
	t.Helper()
	for sequence := 0; sequence < 20 && !terminal(receipt.GetState()); sequence++ {
		response, err := service.WaitValidation(context.Background(), connect.NewRequest(&validationv1.WaitValidationRequest{ReceiptId: receipt.GetReceiptId(), WaitId: fmt.Sprintf("terminal-observer-%d", sequence), AfterRevision: receipt.GetRevision(), Timeout: durationpb.New(time.Second)}))
		if err != nil {
			t.Fatal(err)
		}
		receipt = response.Msg.GetReceipt()
	}
	return receipt
}

func TestStartupRecoveryReattachesRecordedChildWithoutDuplicateStart(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	repo := NewRepository(testsqllite(t))
	admission, err := repo.Admit(context.Background(), validIntent("plan-manager", "recover"))
	if err != nil {
		t.Fatal(err)
	}
	running, err := repo.Transition(context.Background(), admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
		receipt.Children = []*validationv1.ChildOperation{{ChildId: "scenario:demo:1", Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, Owner: "test-genie", OperationId: "existing-run"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	producer := NewRunProducer(runs)
	restarted := NewService(repo, producer)
	restarted.SetProducer(producer)
	count, err := restarted.Recover(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("recover count=%d err=%v", count, err)
	}
	terminalReceipt := waitForTerminalReceipt(t, restarted, running)
	if terminalReceipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(runs.starts) != 0 || len(runs.waits) != 1 {
		t.Fatalf("recovered receipt=%#v starts=%d waits=%d", terminalReceipt, len(runs.starts), len(runs.waits))
	}
}

func TestStartupRecoveryReattachesRecordedGCTChildWithoutDuplicateStart(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	repo := NewRepository(testsqllite(t))
	intent := validIntent("plan-manager", "recover-gct")
	intent.CallerAttributes = map[string]string{"baseline_name": "before"}
	intent.EvidencePolicy = &validationv1.EvidencePolicy{RequireBehavioralBefore: true, RequiredEvidenceKinds: []string{"gct-collection-diff", "test-genie-run"}}
	admission, err := repo.Admit(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	operationID := "receipt-" + admission.Receipt.GetReceiptId()
	running, err := repo.Transition(context.Background(), admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
		receipt.Children = []*validationv1.ChildOperation{{ChildId: "gct-diff:before", Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_GCT_COLLECTION_DIFF, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, Owner: "git-control-tower", OperationId: operationID}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	gct := &fakeGCTEvidence{result: GCTEvidenceResult{Passed: true, Detail: "clean", Evidence: []*validationv1.EvidenceReference{{EvidenceId: operationID, Kind: "gct-collection-diff", Owner: "git-control-tower"}}}}
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	producer := NewRunProducer(runs).WithGCTEvidence(gct)
	restarted := NewService(repo, producer)
	restarted.SetProducer(producer)
	count, err := restarted.Recover(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("recover count=%d err=%v", count, err)
	}
	terminalReceipt := waitForTerminalReceipt(t, restarted, running)
	if terminalReceipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(gct.diffs) != 0 || len(gct.waits) != 1 || len(runs.starts) != 1 {
		t.Fatalf("recovered receipt=%#v gct starts=%d waits=%v suite starts=%d", terminalReceipt, len(gct.diffs), gct.waits, len(runs.starts))
	}
}

func TestFailedChildRetriesInsideServerOwnedProducerBudget(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	runs := &fakeSuiteRuns{results: []runmanager.LiveStatus{{Status: sharedruns.StatusFailed, Error: "transient"}, {Status: sharedruns.StatusPassed}}}
	producer := NewRunProducer(runs)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	receipt := waitForTerminalReceipt(t, service, createThroughService(t, service, "retry"))
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(receipt.GetChildren()) != 2 || len(receipt.GetEvidence()) != 2 || len(runs.starts) != 2 {
		t.Fatalf("retried receipt=%#v starts=%d", receipt, len(runs.starts))
	}
	history, err := service.repo.History(context.Background(), receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	foundRetry := false
	for _, record := range history {
		foundRetry = foundRetry || record.To == validationv1.ReceiptState_RECEIPT_STATE_RETRY_PENDING
	}
	if !foundRetry {
		t.Fatalf("transition history has no retry: %#v", history)
	}
}

func TestIncompatibleBusyWorkQueuesBehindExistingRunWithoutCallerRetry(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	runs := &fakeSuiteRuns{startErrors: []error{&runmanager.BusyError{Scenario: "demo", RunID: "existing"}}, results: []runmanager.LiveStatus{{Status: sharedruns.StatusPassed}, {Status: sharedruns.StatusPassed}}}
	producer := NewRunProducer(runs)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	receipt := waitForTerminalReceipt(t, service, createThroughService(t, service, "busy-queue"))
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || len(runs.starts) != 2 || len(runs.waits) != 2 {
		t.Fatalf("queued receipt=%#v starts=%d waits=%d", receipt, len(runs.starts), len(runs.waits))
	}
}

func TestAbortRaceTerminalizesOnceAsCancelled(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	runs := &abortRaceRuns{waiting: make(chan struct{}), done: make(chan struct{})}
	producer := NewRunProducer(runs)
	service := NewService(NewRepository(testsqllite(t)), producer)
	service.SetProducer(producer)
	created := createThroughService(t, service, "abort-race")
	select {
	case <-runs.waiting:
	case <-time.After(time.Second):
		t.Fatal("producer did not reach server-owned wait")
	}
	response, err := service.AbortValidationWork(context.Background(), connect.NewRequest(&validationv1.AbortValidationWorkRequest{ReceiptId: created.GetReceiptId(), Reason: "test race", RequestedBy: "test"}))
	if err != nil {
		t.Fatal(err)
	}
	receipt := response.Msg.GetReceipt()
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_CANCELLED {
		t.Fatalf("abort receipt = %#v", receipt)
	}
	history, err := service.repo.History(context.Background(), receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	terminalCount := 0
	for _, record := range history {
		if terminal(record.To) {
			terminalCount++
		}
	}
	if terminalCount != 1 {
		t.Fatalf("terminal transitions = %d history=%#v", terminalCount, history)
	}
}
