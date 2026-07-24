package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	core "github.com/vrooli/api-core/validationrun"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"workflow-health/internal/execution"
	internalvalidation "workflow-health/internal/validation"
	workflowrun "workflow-health/internal/validationrun"
)

func (h *connectHandler) StartValidationRun(ctx context.Context, req *connect.Request[scenariovalidationv1.StartValidationRunRequest]) (*connect.Response[scenariovalidationv1.StartValidationRunResponse], error) {
	if h.deps.Ledger.DB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("validation-run ledger is unavailable"))
	}
	key := strings.TrimSpace(req.Msg.GetIdempotencyKey())
	if key == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("idempotency_key is required"))
	}
	target := core.Target{Scenario: req.Msg.GetScenario(), Path: req.Msg.GetPath()}
	if err := target.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if existing, err := h.deps.Ledger.FindByIdempotency(ctx, key); err == nil {
		if !existing.Run.Target.Equal(target) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("idempotency_key belongs to a different target"))
		}
		return connect.NewResponse(&scenariovalidationv1.StartValidationRunResponse{Run: runProto(existing)}), nil
	} else if !core.IsCode(err, core.ErrorNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	static, err := h.staticResponse(ctx, target.Scenario, target.Path)
	if err != nil {
		return nil, err
	}
	record := workflowrun.Record{Run: core.Run{ID: uuid.NewString(), Target: target, IdempotencyKey: key, ParentRunID: req.Msg.GetParentRunId(), State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}, ETA: 2 * time.Minute, Preliminary: static.Msg}
	if err := h.deps.Ledger.Create(ctx, record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("persist validation run: %w", err))
	}
	h.launch(record.Run.ID)
	return connect.NewResponse(&scenariovalidationv1.StartValidationRunResponse{Run: runProto(record)}), nil
}

func (h *connectHandler) GetValidationRun(ctx context.Context, req *connect.Request[scenariovalidationv1.GetValidationRunRequest]) (*connect.Response[scenariovalidationv1.GetValidationRunResponse], error) {
	record, err := h.deps.Ledger.Get(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, runError(err)
	}
	return connect.NewResponse(&scenariovalidationv1.GetValidationRunResponse{Run: runProto(record)}), nil
}

func (h *connectHandler) WaitValidationRun(ctx context.Context, req *connect.Request[scenariovalidationv1.WaitValidationRunRequest]) (*connect.Response[scenariovalidationv1.WaitValidationRunResponse], error) {
	if timeout := req.Msg.GetTimeout(); timeout != nil && timeout.AsDuration() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout.AsDuration())
		defer cancel()
	}
	for {
		record, err := h.deps.Ledger.Get(ctx, req.Msg.GetRunId())
		if err != nil {
			return nil, runError(err)
		}
		if record.Run.State.Terminal() {
			return connect.NewResponse(&scenariovalidationv1.WaitValidationRunResponse{Run: runProto(record)}), nil
		}
		notice := h.notice(record.Run.ID)
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
		report, err := h.run(ctx, record.Run.Target.Scenario, record.Run.Target.Path, execution.Options{IncludeExecution: true, RunID: id, Isolation: execution.NewRoutingIsolation()})
		if err != nil {
			h.finish(record, core.EventFail, nil, nil, err)
			return
		}
		terminal, err := h.responseForReport(report)
		if err != nil {
			h.finish(record, core.EventFail, nil, nil, err)
			return
		}
		h.finish(record, core.EventSucceed, terminal, artifactReferences(report), nil)
	}()
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
}
func (h *connectHandler) responseForReport(report providerReport) (*scenariovalidationv1.ValidateScenarioResponse, error) {
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
	return assessment.BuildValidationResponse(report.Scenario, maturity, native, &commonv1.ExecutionMetrics{}, validationStatusOptions(report)...)
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
