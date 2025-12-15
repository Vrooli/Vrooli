package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExecuteWorkflow starts a workflow execution and returns the DB index record.
// Detailed execution results are written to disk by the artifact recorder.
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	if s == nil {
		return nil, errors.New("workflow service not configured")
	}

	getResp, err := s.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: workflowID.String()})
	if err != nil {
		return nil, err
	}
	if getResp == nil || getResp.Workflow == nil {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	now := time.Now().UTC()
	exec := &database.ExecutionIndex{
		ID:        uuid.New(),
		WorkflowID: workflowID,
		Status:    database.ExecutionStatusPending,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return nil, err
	}

	_ = s.writeExecutionSnapshot(ctx, exec, &basexecution.Execution{
		ExecutionId: exec.ID.String(),
		WorkflowId:  workflowID.String(),
		Status:      typeconv.StringToExecutionStatus(exec.Status),
		TriggerType: basbase.TriggerType_TRIGGER_TYPE_MANUAL,
		StartedAt:   timestamppb.New(now),
		CreatedAt:   timestamppb.New(now),
		UpdatedAt:   timestamppb.New(now),
	})

	s.startExecutionRunner(getResp.Workflow, exec.ID, parameters)
	return exec, nil
}

// ExecuteWorkflowAPI starts a workflow execution using proto request/response types.
func (s *WorkflowService) ExecuteWorkflowAPI(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	workflowID, err := uuid.Parse(strings.TrimSpace(req.WorkflowId))
	if err != nil {
		return nil, fmt.Errorf("invalid workflow_id: %w", err)
	}
	version := int(req.GetWorkflowVersion())

	workflowSummary, err := s.resolveWorkflowForExecution(ctx, workflowID, version)
	if err != nil {
		return nil, err
	}

	initialStore, initialParams, env := executionParametersToMaps(req.Parameters)

	now := time.Now().UTC()
	exec := &database.ExecutionIndex{
		ID:        uuid.New(),
		WorkflowID: workflowID,
		Status:    database.ExecutionStatusPending,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return nil, err
	}

	// Persist an initial proto snapshot immediately so the filesystem is the source of truth
	// for parameters/trigger metadata and other rich execution fields not stored in the DB index.
	snapshot := &basexecution.Execution{
		ExecutionId: exec.ID.String(),
		WorkflowId:  workflowID.String(),
		WorkflowVersion: int32(version),
		Status:      typeconv.StringToExecutionStatus(exec.Status),
		TriggerType: basbase.TriggerType_TRIGGER_TYPE_API,
		StartedAt:   timestamppb.New(now),
		CreatedAt:   timestamppb.New(now),
		UpdatedAt:   timestamppb.New(now),
		Parameters:  req.Parameters,
	}
	_ = s.writeExecutionSnapshot(ctx, exec, snapshot)

	s.startExecutionRunnerWithNamespaces(workflowSummary, exec.ID, initialStore, initialParams, env)

	if req.WaitForCompletion {
		// Poll for completion; execution updates are persisted to the DB index by the runner.
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				latest, err := s.repo.GetExecution(ctx, exec.ID)
				if err != nil {
					return nil, err
				}
				if latest.CompletedAt != nil {
					resp := &basapi.ExecuteWorkflowResponse{
						ExecutionId: latest.ID.String(),
						Status:      typeconv.StringToExecutionStatus(latest.Status),
						CompletedAt: timestamppb.New(*latest.CompletedAt),
					}
					if strings.TrimSpace(latest.ErrorMessage) != "" {
						msg := latest.ErrorMessage
						resp.Error = &msg
					}
					return resp, nil
				}
			}
		}
	}

	return &basapi.ExecuteWorkflowResponse{
		ExecutionId: exec.ID.String(),
		Status:      typeconv.StringToExecutionStatus(exec.Status),
	}, nil
}

func (s *WorkflowService) resolveWorkflowForExecution(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error) {
	if version > 0 {
		return s.GetWorkflowVersion(ctx, workflowID, version)
	}
	getResp, err := s.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: workflowID.String()})
	if err != nil {
		return nil, err
	}
	if getResp == nil || getResp.Workflow == nil {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	return getResp.Workflow, nil
}

func executionParametersToMaps(p *basexecution.ExecutionParameters) (store map[string]any, params map[string]any, env map[string]any) {
	store = map[string]any{}
	params = map[string]any{}
	env = map[string]any{}
	if p == nil {
		return store, params, env
	}

	for k, v := range p.InitialStore {
		store[k] = jsonValueToAny(v)
	}
	for k, v := range p.InitialParams {
		params[k] = jsonValueToAny(v)
	}
	for k, v := range p.Env {
		env[k] = jsonValueToAny(v)
	}
	// Backward compatibility: variables map<string,string> feeds @store/ as strings.
	for k, v := range p.Variables {
		if _, exists := store[k]; !exists {
			store[k] = v
		}
	}
	return store, params, env
}

func jsonValueToAny(v *commonv1.JsonValue) any {
	return typeconv.JsonValueToAny(v)
}

func (s *WorkflowService) startExecutionRunner(workflow *basapi.WorkflowSummary, executionID uuid.UUID, parameters map[string]any) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeExecutionCancel(executionID, cancel)
	go s.executeWorkflowAsync(ctx, workflow, executionID, parameters)
}

func (s *WorkflowService) startExecutionRunnerWithNamespaces(workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeExecutionCancel(executionID, cancel)
	go s.executeWorkflowAsyncWithNamespaces(ctx, workflow, executionID, store, params, env)
}

func (s *WorkflowService) executeWorkflowAsync(ctx context.Context, workflow *basapi.WorkflowSummary, executionID uuid.UUID, parameters map[string]any) {
	defer s.cancelExecutionByID(executionID)

	persistenceCtx := context.Background()
	execIndex, err := s.repo.GetExecution(persistenceCtx, executionID)
	if err != nil {
		return
	}

	execIndex.Status = database.ExecutionStatusRunning
	execIndex.UpdatedAt = time.Now().UTC()
	_ = s.repo.UpdateExecution(persistenceCtx, execIndex)
	_ = s.writeExecutionSnapshot(persistenceCtx, execIndex, &basexecution.Execution{
		ExecutionId: execIndex.ID.String(),
		WorkflowId:  execIndex.WorkflowID.String(),
		Status:      typeconv.StringToExecutionStatus(execIndex.Status),
		StartedAt:   timestamppb.New(execIndex.StartedAt),
		CreatedAt:   timestamppb.New(execIndex.CreatedAt),
		UpdatedAt:   timestamppb.New(execIndex.UpdatedAt),
	})

	engineName := autoengine.FromEnv().Resolve("")
	eventSink := s.newEventSink()
	_ = eventSink

	plan, _, err := autoexecutor.BuildContractsPlan(ctx, executionID, workflow)
	if err != nil {
		execIndex.Status = database.ExecutionStatusFailed
		execIndex.ErrorMessage = err.Error()
		now := time.Now().UTC()
		execIndex.CompletedAt = &now
		execIndex.UpdatedAt = now
		_ = s.repo.UpdateExecution(persistenceCtx, execIndex)
		return
	}

	req := autoexecutor.Request{
		Plan:              plan,
		EngineName:        engineName,
		EngineFactory:     s.engineFactory,
		Recorder:          s.artifactRecorder,
		EventSink:         eventSink,
		HeartbeatInterval: 2 * time.Second,
		ReuseMode:         autoengine.ReuseModeReuse,
		WorkflowResolver:  s,
		PlanCompiler:      s.planCompiler,
		MaxSubflowDepth:   5,
		StartFromStepIndex: -1,
		InitialStore:      parameters,
	}

	executor := s.executor
	if executor == nil {
		executor = autoexecutor.NewSimpleExecutor(nil)
	}
	runErr := executor.Execute(ctx, req)

	status := database.ExecutionStatusCompleted
	errMsg := ""
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) || strings.Contains(strings.ToLower(runErr.Error()), "cancel") {
			status = database.ExecutionStatusFailed
			errMsg = "execution cancelled"
		} else {
			status = database.ExecutionStatusFailed
			errMsg = runErr.Error()
		}
	}

	now := time.Now().UTC()
	execIndex.Status = status
	execIndex.ErrorMessage = errMsg
	execIndex.CompletedAt = &now
	execIndex.UpdatedAt = now
	_ = s.repo.UpdateExecution(persistenceCtx, execIndex)
	_ = s.writeExecutionSnapshot(persistenceCtx, execIndex, &basexecution.Execution{
		ExecutionId: execIndex.ID.String(),
		WorkflowId:  execIndex.WorkflowID.String(),
		Status:      typeconv.StringToExecutionStatus(execIndex.Status),
		StartedAt:   timestamppb.New(execIndex.StartedAt),
		CreatedAt:   timestamppb.New(execIndex.CreatedAt),
		UpdatedAt:   timestamppb.New(execIndex.UpdatedAt),
		CompletedAt: timestamppb.New(now),
		Error: func() *string {
			if strings.TrimSpace(execIndex.ErrorMessage) == "" {
				return nil
			}
			msg := execIndex.ErrorMessage
			return &msg
		}(),
	})
}

func (s *WorkflowService) executeWorkflowAsyncWithNamespaces(ctx context.Context, workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any) {
	defer s.cancelExecutionByID(executionID)

	persistenceCtx := context.Background()
	execIndex, err := s.repo.GetExecution(persistenceCtx, executionID)
	if err != nil {
		return
	}

	execIndex.Status = database.ExecutionStatusRunning
	execIndex.UpdatedAt = time.Now().UTC()
	_ = s.repo.UpdateExecution(persistenceCtx, execIndex)
	_ = s.writeExecutionSnapshot(persistenceCtx, execIndex, &basexecution.Execution{
		ExecutionId: execIndex.ID.String(),
		WorkflowId:  execIndex.WorkflowID.String(),
		Status:      typeconv.StringToExecutionStatus(execIndex.Status),
		StartedAt:   timestamppb.New(execIndex.StartedAt),
		CreatedAt:   timestamppb.New(execIndex.CreatedAt),
		UpdatedAt:   timestamppb.New(execIndex.UpdatedAt),
	})

	engineName := autoengine.FromEnv().Resolve("")
	eventSink := s.newEventSink()

	plan, _, err := autoexecutor.BuildContractsPlan(ctx, executionID, workflow)
	if err != nil {
		execIndex.Status = database.ExecutionStatusFailed
		execIndex.ErrorMessage = err.Error()
		now := time.Now().UTC()
		execIndex.CompletedAt = &now
		execIndex.UpdatedAt = now
		_ = s.repo.UpdateExecution(persistenceCtx, execIndex)
		return
	}

	req := autoexecutor.Request{
		Plan:              plan,
		EngineName:        engineName,
		EngineFactory:     s.engineFactory,
		Recorder:          s.artifactRecorder,
		EventSink:         eventSink,
		HeartbeatInterval: 2 * time.Second,
		ReuseMode:         autoengine.ReuseModeReuse,
		WorkflowResolver:  s,
		PlanCompiler:      s.planCompiler,
		MaxSubflowDepth:   5,
		StartFromStepIndex: -1,
		InitialStore:      store,
		InitialParams:     params,
		Env:              env,
	}

	executor := s.executor
	if executor == nil {
		executor = autoexecutor.NewSimpleExecutor(nil)
	}
	runErr := executor.Execute(ctx, req)

	status := database.ExecutionStatusCompleted
	errMsg := ""
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) || strings.Contains(strings.ToLower(runErr.Error()), "cancel") {
			status = database.ExecutionStatusFailed
			errMsg = "execution cancelled"
		} else {
			status = database.ExecutionStatusFailed
			errMsg = runErr.Error()
		}
	}

	now := time.Now().UTC()
	execIndex.Status = status
	execIndex.ErrorMessage = errMsg
	execIndex.CompletedAt = &now
	execIndex.UpdatedAt = now
	_ = s.repo.UpdateExecution(persistenceCtx, execIndex)
	_ = s.writeExecutionSnapshot(persistenceCtx, execIndex, &basexecution.Execution{
		ExecutionId: execIndex.ID.String(),
		WorkflowId:  execIndex.WorkflowID.String(),
		Status:      typeconv.StringToExecutionStatus(execIndex.Status),
		StartedAt:   timestamppb.New(execIndex.StartedAt),
		CreatedAt:   timestamppb.New(execIndex.CreatedAt),
		UpdatedAt:   timestamppb.New(execIndex.UpdatedAt),
		CompletedAt: timestamppb.New(now),
		Error: func() *string {
			if strings.TrimSpace(execIndex.ErrorMessage) == "" {
				return nil
			}
			msg := execIndex.ErrorMessage
			return &msg
		}(),
	})
}

func (s *WorkflowService) storeExecutionCancel(executionID uuid.UUID, cancel context.CancelFunc) {
	if s == nil {
		return
	}
	s.executionCancels.Store(executionID, cancel)
}

func (s *WorkflowService) cancelExecutionByID(executionID uuid.UUID) {
	if s == nil {
		return
	}
	if value, ok := s.executionCancels.Load(executionID); ok {
		if cancel, ok := value.(context.CancelFunc); ok && cancel != nil {
			cancel()
		}
	}
	s.executionCancels.Delete(executionID)
}

func (s *WorkflowService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	if s == nil {
		return errors.New("workflow service not configured")
	}
	_ = ctx
	s.cancelExecutionByID(executionID)
	return nil
}
