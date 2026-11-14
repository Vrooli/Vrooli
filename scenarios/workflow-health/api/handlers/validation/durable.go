package validation

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"workflow-health/internal/execution"
	internalvalidation "workflow-health/internal/validation"
	workflowrun "workflow-health/internal/validationrun"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/metrics"
	core "github.com/vrooli/api-core/validationrun"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *connectHandler) StartValidationRun(ctx context.Context, req *connect.Request[scenariovalidationv1.StartValidationRunRequest]) (*connect.Response[scenariovalidationv1.StartValidationRunResponse], error) {
	if req.Msg.BrowserCaseSelector != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("browser_case_selector requires StartSelectedValidationRun"))
	}
	return h.startValidationRun(ctx, req.Msg, nil)
}

func (h *connectHandler) StartSelectedValidationRun(ctx context.Context, req *connect.Request[scenariovalidationv1.StartValidationRunRequest]) (*connect.Response[scenariovalidationv1.StartValidationRunResponse], error) {
	if req.Msg.BrowserCaseSelector == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("browser_case_selector is required"))
	}
	paths, err := execution.NormalizeCasePaths(req.Msg.BrowserCaseSelector.CasePaths)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if strings.TrimSpace(req.Msg.GetDesktopBinding().GetWorkflowPath()) != "" || len(req.Msg.CapabilitySubset) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("selected browser cases do not support workflow-path bindings or capability_subset"))
	}
	if binding := req.Msg.DesktopBinding; binding != nil && (electronTarget(binding) == nil || strings.TrimSpace(binding.GetProfileId()) == "") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("selected desktop binding requires complete target and validation-cell identity"))
	}
	return h.startValidationRun(ctx, req.Msg, paths)
}

// Both public operations share one durable owner. Reattachment checks persisted
// identity before catalog validation so drift never admits another execution.
func (h *connectHandler) startValidationRun(ctx context.Context, req *scenariovalidationv1.StartValidationRunRequest, paths []string) (*connect.Response[scenariovalidationv1.StartValidationRunResponse], error) {
	if h.deps.Ledger.DB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("validation-run ledger is unavailable"))
	}
	key := strings.TrimSpace(req.GetIdempotencyKey())
	if key == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("idempotency_key is required"))
	}
	target := core.Target{Scenario: req.GetScenario(), Path: req.GetPath()}
	if err := target.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	variant := req.GetDesktopBinding().GetBrowserAutomationStudioVariant()
	if _, err := browserAutomationStudioAddress(variant); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if existing, err := h.deps.Ledger.FindByIdempotency(ctx, key); err == nil {
		if !matchesRequest(existing, target, paths, req.GetDesktopBinding()) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("idempotency_key belongs to a different target or browser case selection"))
		}
		return connect.NewResponse(&scenariovalidationv1.StartValidationRunResponse{Run: runProto(existing)}), nil
	} else if !core.IsCode(err, core.ErrorNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Replays of an already accepted request remain readable during maintenance.
	// A new durable reservation must acquire the provider's linearizable gate
	// before static analysis, persistence, or worker dispatch.
	releaseAdmission, err := h.admitValidation(ctx)
	if err != nil {
		if errors.Is(err, errValidationAdmissionClosed) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("check lifecycle admission: %w", err))
	}
	defer releaseAdmission()
	static, err := h.staticResponseWithSelector(ctx, target.Scenario, target.Path, execution.AssetSelector{CasesProvided: paths != nil, CasePaths: paths})
	if err != nil {
		return nil, err
	}
	record := workflowrun.Record{Run: core.Run{ID: uuid.NewString(), Target: target, IdempotencyKey: key, ParentRunID: req.GetParentRunId(), State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}, ETA: 2 * time.Minute, Preliminary: static.Msg, ExecutionBinding: req.GetDesktopBinding(), BrowserCasePaths: paths}
	if err := h.deps.Ledger.Create(ctx, record); err != nil {
		// Another handler or process may have won the unique-key insert. Only
		// its durable identity can permit replay; never launch the losing record.
		if existing, readErr := h.deps.Ledger.FindByIdempotency(ctx, key); readErr == nil {
			if !matchesRequest(existing, target, paths, req.GetDesktopBinding()) {
				return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("idempotency_key belongs to a different target or browser case selection"))
			}
			return connect.NewResponse(&scenariovalidationv1.StartValidationRunResponse{Run: runProto(existing)}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("persist validation run: %w", err))
	}
	h.launch(record.Run.ID)
	return connect.NewResponse(&scenariovalidationv1.StartValidationRunResponse{Run: runProto(record)}), nil
}

func matchesRequest(record workflowrun.Record, target core.Target, paths []string, binding *scenariovalidationv1.DesktopValidationBinding) bool {
	return record.Run.Target.Equal(target) && (record.BrowserCasePaths == nil) == (paths == nil) && slices.Equal(record.BrowserCasePaths, paths) && proto.Equal(record.ExecutionBinding, binding)
}

func (h *connectHandler) GetValidationRun(ctx context.Context, req *connect.Request[scenariovalidationv1.GetValidationRunRequest]) (*connect.Response[scenariovalidationv1.GetValidationRunResponse], error) {
	record, err := h.deps.Ledger.Get(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, runError(err)
	}
	return connect.NewResponse(&scenariovalidationv1.GetValidationRunResponse{Run: h.currentEvidence(ctx, runProto(record))}), nil
}

func (h *connectHandler) WaitValidationRun(ctx context.Context, req *connect.Request[scenariovalidationv1.WaitValidationRunRequest]) (*connect.Response[scenariovalidationv1.WaitValidationRunResponse], error) {
	if timeout := req.Msg.GetTimeout(); timeout != nil && timeout.AsDuration() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout.AsDuration())
		defer cancel()
	}
	for {
		// Subscribe before reading so a terminal transition between the read and
		// subscription cannot strand a waiter on the next notification channel.
		notice := h.notice(req.Msg.GetRunId())
		record, err := h.deps.Ledger.Get(ctx, req.Msg.GetRunId())
		if err != nil {
			return nil, runError(err)
		}
		if record.Run.State.Terminal() {
			return connect.NewResponse(&scenariovalidationv1.WaitValidationRunResponse{Run: h.currentEvidence(ctx, runProto(record))}), nil
		}
		select {
		case <-notice:
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("wait timed out; validation run remains active"))
		}
	}
}

func (h *connectHandler) AbortValidationRun(ctx context.Context, req *connect.Request[scenariovalidationv1.AbortValidationRunRequest]) (*connect.Response[scenariovalidationv1.AbortValidationRunResponse], error) {
	record, err := h.deps.Ledger.Get(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, runError(err)
	}
	if !record.Run.State.Terminal() {
		next, err := core.Transition(record.Run, core.EventRequestAbort, time.Now().UTC())
		if err != nil {
			return nil, runError(err)
		}
		next.Version = record.Run.Version + 1
		record.Run = next
		if err := h.deps.Ledger.Update(ctx, record, next.Version-1); err != nil {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		h.cancel(record.Run.ID)
		h.signal(record.Run.ID)
	}
	return connect.NewResponse(&scenariovalidationv1.AbortValidationRunResponse{Run: runProto(record)}), nil
}

func (h *connectHandler) launch(id string) {
	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.cancels[id] = cancel
	h.mu.Unlock()
	go func() {
		defer h.cancel(id)
		record, err := h.deps.Ledger.Get(ctx, id)
		if err != nil {
			return
		}
		claimed, err := core.Transition(record.Run, core.EventClaim, time.Now().UTC())
		if err != nil {
			return
		}
		claimed.Version = record.Run.Version + 1
		record.Run = claimed
		if h.deps.Ledger.Update(ctx, record, claimed.Version-1) != nil {
			return
		}
		h.signal(id)
		opts, err := executionOptions(record)
		if err != nil {
			h.finish(record, core.EventFail, nil, nil, err)
			return
		}

		// Measure the durable run. This path is the one that produces the
		// workflow phase records Test Genie costs, and it used to report an
		// empty &commonv1.ExecutionMetrics{} — non-nil, so the metrics_present
		// column read 1 over no data for every record. The static path beside
		// it has always collected properly; this simply does the same.
		collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
		evidence := h.snapshotEvidence(ctx, record.Run.Target.Scenario, record.Run.Target.Path, record.BrowserCasePaths)
		report, err := h.run(ctx, record.Run.Target.Scenario, record.Run.Target.Path, opts)
		if err != nil {
			collector.Stop()
			h.finish(record, core.EventFail, nil, nil, err)
			return
		}
		report.Evidence = h.qualifyEvidence(ctx, evidence)
		terminal, err := h.responseForReport(report, collector.Stop())
		if err != nil {
			h.finish(record, core.EventFail, nil, nil, err)
			return
		}
		h.finish(record, core.EventSucceed, terminal, artifactReferences(report), nil)
	}()
}

// executionOptions keeps the saved selector separate from renderer binding.
func executionOptions(record workflowrun.Record) (execution.Options, error) {
	if record.BrowserCasePaths != nil && strings.TrimSpace(record.ExecutionBinding.GetWorkflowPath()) != "" {
		return execution.Options{}, fmt.Errorf("saved browser case selection conflicts with workflow-path binding")
	}
	if record.BrowserCasePaths != nil && record.ExecutionBinding != nil && (electronTarget(record.ExecutionBinding) == nil || strings.TrimSpace(record.ExecutionBinding.GetProfileId()) == "") {
		return execution.Options{}, fmt.Errorf("saved selected desktop binding requires complete target and validation-cell identity")
	}
	opts := execution.Options{IncludeExecution: true, Selector: execution.AssetSelector{CasesProvided: record.BrowserCasePaths != nil, CasePaths: record.BrowserCasePaths}, RunID: record.Run.ID, Isolation: execution.NewRoutingIsolation(), ElectronTarget: electronTarget(record.ExecutionBinding), ValidationContext: electronValidationContext(record.ExecutionBinding, record.Run.ID)}
	if record.ExecutionBinding != nil {
		opts.BASVariant = record.ExecutionBinding.GetBrowserAutomationStudioVariant()
		opts.IsolationLeaseID = record.ExecutionBinding.GetIsolationLeaseId()
		opts.PrepareIsolationRestart = record.ExecutionBinding.GetPrepareIsolationRestart()
		opts.ResumeIsolationRestart = record.ExecutionBinding.GetResumeIsolationRestart()
		if (opts.PrepareIsolationRestart || opts.ResumeIsolationRestart) && (!record.ExecutionBinding.GetBundledRuntime() || opts.IsolationLeaseID == "" || (opts.PrepareIsolationRestart && opts.ResumeIsolationRestart)) {
			return execution.Options{}, fmt.Errorf("restart handoff requires one bundled target and a stable isolation lease identity")
		}
		if record.ExecutionBinding.GetBundledRuntime() {
			isolation, err := execution.NewElectronRoutingIsolation(opts.ElectronTarget)
			if err != nil {
				return execution.Options{}, err
			}
			isolation.PrepareRestart = opts.PrepareIsolationRestart
			isolation.ResumeRestart = opts.ResumeIsolationRestart
			opts.ElectronTarget.DownloadRoot = isolation.DownloadRoot
			opts.Isolation = isolation
		}
	}
	if workflowPath := strings.TrimSpace(record.ExecutionBinding.GetWorkflowPath()); workflowPath != "" {
		// Default desktop flow bindings preserve their existing narrow flow behavior.
		opts.Selector.CasePaths = []string{workflowPath}
		opts.Selector.FlowPaths = []string{workflowPath}
		opts.AllowFlowExecution = true
	}
	return opts, nil
}

func electronTarget(binding *scenariovalidationv1.DesktopValidationBinding) *execution.ElectronTarget {
	if binding == nil || strings.TrimSpace(binding.GetTargetId()) == "" || strings.TrimSpace(binding.GetCdpEndpoint()) == "" || strings.TrimSpace(binding.GetRendererId()) == "" || strings.TrimSpace(binding.GetScenarioName()) == "" || strings.TrimSpace(binding.GetArtifactDigest()) == "" || strings.TrimSpace(binding.GetContextId()) == "" || strings.TrimSpace(binding.GetCdpTransport()) == "" {
		return nil
	}
	return &execution.ElectronTarget{
		TargetID: binding.GetTargetId(), CDPEndpoint: binding.GetCdpEndpoint(), RendererID: binding.GetRendererId(),
		RendererURL: binding.GetRendererUrl(), RendererTitle: binding.GetRendererTitle(), ScenarioName: binding.GetScenarioName(),
		ArtifactDigest: binding.GetArtifactDigest(), ContextID: binding.GetContextId(), CDPTransport: binding.GetCdpTransport(),
		APIURL: binding.GetApiUrl(), TestDatabaseRoot: binding.GetTestDatabaseRoot(), BundledRuntime: binding.GetBundledRuntime(),
		IsolationLeaseID: binding.GetIsolationLeaseId(), PrepareIsolationRestart: binding.GetPrepareIsolationRestart(), ResumeIsolationRestart: binding.GetResumeIsolationRestart(),
	}
}

func electronValidationContext(binding *scenariovalidationv1.DesktopValidationBinding, runID string) *execution.ValidationContext {
	if binding == nil {
		return nil
	}
	workflowID := strings.TrimSpace(binding.GetWorkflowId())
	if workflowID == "" {
		workflowID = binding.GetWorkflowPath()
	}
	leaseID := strings.TrimSpace(binding.GetIsolationLeaseId())
	if leaseID == "" {
		leaseID = runID
	}
	return &execution.ValidationContext{
		ContextID: binding.GetContextId(), ScenarioName: binding.GetScenarioName(), ArtifactDigest: binding.GetArtifactDigest(),
		TargetID: binding.GetTargetId(), ProfileID: binding.GetProfileId(), WorkflowID: workflowID, IsolationLeaseID: leaseID,
	}
}

func (h *connectHandler) finish(record workflowrun.Record, event core.Event, terminal *scenariovalidationv1.ValidateScenarioResponse, artifacts []string, runErr error) {
	current, err := h.deps.Ledger.Get(context.Background(), record.Run.ID)
	if err != nil || current.Run.State.Terminal() {
		return
	}
	if current.Run.CancellationRequested {
		event = core.EventCancel
		terminal = nil
		artifacts = nil
		runErr = nil
	}
	next, err := core.Transition(current.Run, event, time.Now().UTC())
	if err != nil {
		return
	}
	next.Version = current.Run.Version + 1
	current.Run = next
	current.Terminal = terminal
	current.Artifacts = append([]string(nil), artifacts...)
	if runErr != nil {
		current.ErrorCode = string(core.ErrorExecutionFailed)
		current.Error = runErr.Error()
	}
	_ = h.deps.Ledger.Update(context.Background(), current, next.Version-1)
	h.signal(record.Run.ID)
	h.signalLifecycleChange()
}

func (h *connectHandler) responseForReport(report providerReport, execMetrics *commonv1.ExecutionMetrics) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	if h.deps.MaturitySpec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	maturity, err := internalvalidation.BuildMaturityAssessment(report.Scenario, report.Findings, *h.deps.MaturitySpec)
	if err != nil {
		return nil, err
	}
	native, err := nativeDetail(report)
	if err != nil {
		return nil, err
	}
	return assessment.BuildValidationResponse(report.Scenario, maturity, native, execMetrics, validationStatusOptions(report)...)
}

func (h *connectHandler) notice(id string) <-chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.notices[id] == nil {
		h.notices[id] = make(chan struct{})
	}
	return h.notices[id]
}

func (h *connectHandler) signal(id string) {
	h.mu.Lock()
	if ch := h.notices[id]; ch != nil {
		close(ch)
	}
	h.notices[id] = make(chan struct{})
	h.mu.Unlock()
}

func (h *connectHandler) cancel(id string) {
	h.mu.Lock()
	if cancel := h.cancels[id]; cancel != nil {
		cancel()
		delete(h.cancels, id)
	}
	h.mu.Unlock()
}

func runError(err error) error {
	if core.IsCode(err, core.ErrorNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func runProto(record workflowrun.Record) *scenariovalidationv1.ValidationRun {
	run := &scenariovalidationv1.ValidationRun{RunId: record.Run.ID, Scenario: record.Run.Target.Scenario, Path: record.Run.Target.Path, IdempotencyKey: record.Run.IdempotencyKey, ParentRunId: record.Run.ParentRunID, State: stateProto(record.Run.State), CreatedAt: timestamppb.New(record.Run.CreatedAt), StartedAt: timestamppb.New(record.Run.StartedAt), CompletedAt: timestamppb.New(record.Run.CompletedAt), EstimatedRemaining: durationpb.New(record.ETA), PreliminaryStaticResult: record.Preliminary, TerminalResult: record.Terminal, CancellationRequested: record.Run.CancellationRequested}
	if record.BrowserCasePaths != nil {
		run.BrowserCaseSelector = &scenariovalidationv1.BrowserCaseSelector{CasePaths: slices.Clone(record.BrowserCasePaths)}
	}
	if record.Error != "" {
		run.Error = &scenariovalidationv1.ValidationRunError{Code: errorCodeProto(record.ErrorCode), Message: record.Error}
	}
	for _, reference := range record.Artifacts {
		value, err := structpb.NewStruct(map[string]any{"reference": reference})
		if err != nil {
			continue
		}
		artifact, err := anypb.New(value)
		if err == nil {
			run.ArtifactReferences = append(run.ArtifactReferences, artifact)
		}
	}
	return run
}

func artifactReferences(report providerReport) []string {
	references := make([]string, 0, len(report.Runs)*3)
	for _, run := range report.Runs {
		for _, reference := range []string{run.Artifact.Workflow, run.Artifact.Latest, run.Artifact.Timeline} {
			if strings.TrimSpace(reference) != "" {
				references = append(references, reference)
			}
		}
	}
	return references
}

func errorCodeProto(code string) scenariovalidationv1.ValidationRunErrorCode {
	switch code {
	case string(core.ErrorRecoveryFailed):
		return scenariovalidationv1.ValidationRunErrorCode_VALIDATION_RUN_ERROR_CODE_RECOVERY_FAILED
	case string(core.ErrorWaitTimeout):
		return scenariovalidationv1.ValidationRunErrorCode_VALIDATION_RUN_ERROR_CODE_WAIT_TIMEOUT
	default:
		return scenariovalidationv1.ValidationRunErrorCode_VALIDATION_RUN_ERROR_CODE_EXECUTION_FAILED
	}
}

func stateProto(state core.State) scenariovalidationv1.ValidationRunState {
	return map[core.State]scenariovalidationv1.ValidationRunState{core.StateQueued: scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_QUEUED, core.StateRunning: scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_RUNNING, core.StateSucceeded: scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_SUCCEEDED, core.StateFailed: scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_FAILED, core.StateCanceled: scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_CANCELED, core.StateRecoveryFailed: scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_RECOVERY_FAILED}[state]
}
