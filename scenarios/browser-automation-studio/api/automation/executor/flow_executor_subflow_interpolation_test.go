package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	compiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/automation/state"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func TestExecuteGraphReturnsNewSessionWhenItsFirstStepFails(t *testing.T) {
	t.Parallel()

	session := &failingEngineSession{}
	executor := NewSimpleExecutor(nil)
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Graph: &contracts.PlanGraph{Steps: []contracts.PlanStep{{
			Index:  0,
			NodeID: "failing-navigate",
			Action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}},
			},
		}}},
	}
	returned, err := executor.executeGraph(
		context.Background(),
		Request{Plan: plan, Recorder: &stubExecutionWriter{}},
		executionContext{navigation: &navigationState{}},
		&failingAutomationEngine{session: session},
		engine.SessionSpec{},
		nil,
		state.NewFromStore(nil),
		engine.ReuseModeReuse,
	)
	if err == nil {
		t.Fatal("expected first graph step to fail")
	}
	if returned == nil {
		t.Fatal("failed first step discarded the newly created session")
	}
	require.Same(t, session, returned)
}

func TestExecuteGraphCompletesWhenAFailedStepIsConfiguredToContinue(t *testing.T) {
	t.Parallel()

	continueOnError := true
	session := &failingEngineSession{}
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Graph: &contracts.PlanGraph{Steps: []contracts.PlanStep{{
			Index:   0,
			NodeID:  "recoverable-navigate",
			Context: map[string]any{"continueOnError": continueOnError},
			Action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}},
			},
		}}},
	}

	returned, err := NewSimpleExecutor(nil).executeGraph(
		context.Background(),
		Request{Plan: plan, Recorder: &stubExecutionWriter{}},
		executionContext{navigation: &navigationState{}},
		&failingAutomationEngine{session: session},
		engine.SessionSpec{},
		nil,
		state.NewFromStore(nil),
		engine.ReuseModeReuse,
	)
	require.NoError(t, err)
	require.Same(t, session, returned)
}

func TestExecuteGraphHonorsContinueOnErrorCompiledFromWorkflow(t *testing.T) {
	t.Parallel()

	continueOnError := true
	workflow := &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{{
			Id: "recoverable-navigate",
			Action: &basactions.ActionDefinition{
				Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{
					Url: "https://example.com",
				}},
			},
			ExecutionSettings: &basworkflows.NodeExecutionSettings{ContinueOnError: &continueOnError},
		}},
	}
	plan, _, err := compiler.CompileWorkflowToContracts(context.Background(), uuid.New(), &basapi.WorkflowSummary{
		Id:             uuid.NewString(),
		Name:           "recoverable-compiled-flow",
		FlowDefinition: workflow,
	})
	require.NoError(t, err)

	session := &failingEngineSession{}
	returned, err := NewSimpleExecutor(nil).executeGraph(
		context.Background(),
		Request{Plan: plan, Recorder: &stubExecutionWriter{}},
		executionContext{navigation: &navigationState{}},
		&failingAutomationEngine{session: session},
		engine.SessionSpec{},
		nil,
		state.NewFromStore(nil),
		engine.ReuseModeReuse,
	)
	require.NoError(t, err)
	require.Same(t, session, returned)
}

func TestExecuteGraphAcceptsPersistedSnakeCaseContinueOnError(t *testing.T) {
	t.Parallel()

	session := &failingEngineSession{}
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Graph: &contracts.PlanGraph{Steps: []contracts.PlanStep{{
			Index:   0,
			NodeID:  "recoverable-persisted-navigate",
			Context: map[string]any{"continue_on_error": true},
			Action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}},
			},
		}}},
	}

	_, err := NewSimpleExecutor(nil).executeGraph(
		context.Background(),
		Request{Plan: plan, Recorder: &stubExecutionWriter{}},
		executionContext{navigation: &navigationState{}},
		&failingAutomationEngine{session: session},
		engine.SessionSpec{},
		nil,
		state.NewFromStore(nil),
		engine.ReuseModeReuse,
	)
	require.NoError(t, err)
}

func TestExecutePlanStep_SubflowInterpolatesArgsBeforeExecution(t *testing.T) {
	t.Parallel()

	executionID := uuid.New()
	parentWorkflowID := uuid.New()
	childWorkflowID := uuid.New()

	parentStep := contracts.PlanStep{
		Index:  0,
		NodeID: "parent-subflow",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
			Params: &basactions.ActionDefinition_Subflow{
				Subflow: &basactions.SubflowParams{
					Target: &basactions.SubflowParams_WorkflowId{WorkflowId: childWorkflowID.String()},
					Args: typeconv.ToJsonValueMap(map[string]any{
						"command": "${@params/command}",
					}),
				},
			},
		},
	}

	childWorkflow := &basapi.WorkflowSummary{
		Id:   childWorkflowID.String(),
		Name: "child-input",
		FlowDefinition: &basworkflows.WorkflowDefinitionV2{
			Nodes: []*basworkflows.WorkflowNodeV2{
				{
					Id: "type-command",
					Action: &basactions.ActionDefinition{
						Type: basactions.ActionType_ACTION_TYPE_INPUT,
						Params: &basactions.ActionDefinition_Input{
							Input: &basactions.InputParams{
								Selector: "#terminal-command",
								Value:    "${@params/command}",
							},
						},
					},
				},
			},
		},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: executionID,
			WorkflowID:  parentWorkflowID,
			Graph: &contracts.PlanGraph{
				Steps: []contracts.PlanStep{parentStep},
			},
		},
		Recorder:         &stubExecutionWriter{},
		WorkflowResolver: &stubWorkflowResolver{workflows: map[uuid.UUID]*basapi.WorkflowSummary{childWorkflowID: childWorkflow}},
		StartURL:         "https://example.com",
	}

	execCtx := executionContext{
		compiler:   DefaultPlanCompiler,
		maxDepth:   5,
		callStack:  []uuid.UUID{parentWorkflowID},
		navigation: &navigationState{},
	}

	expectedCommand := "printf 'BAS_E2E_TERMINAL_OK\\n'"
	execState := state.New(nil, map[string]any{
		"command": expectedCommand,
	}, nil)

	stubEngine := &stubAutomationEngine{session: &stubEngineSession{}}
	executor := NewSimpleExecutor(nil)
	outcome, _, err := executor.executePlanStep(
		context.Background(),
		req,
		execCtx,
		stubEngine,
		engine.SessionSpec{},
		nil,
		parentStep,
		execState,
		engine.ReuseModeReuse,
	)
	require.NoError(t, err)
	assert.True(t, outcome.Success)
	assert.Equal(t, expectedCommand, stubEngine.session.lastInputValue)
}

type stubWorkflowResolver struct {
	workflows map[uuid.UUID]*basapi.WorkflowSummary
}

func (s *stubWorkflowResolver) GetWorkflow(_ context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error) {
	if wf, ok := s.workflows[workflowID]; ok {
		return wf, nil
	}
	return nil, assert.AnError
}

func (s *stubWorkflowResolver) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, _ int) (*basapi.WorkflowSummary, error) {
	return s.GetWorkflow(ctx, workflowID)
}

func (s *stubWorkflowResolver) GetWorkflowByProjectPath(_ context.Context, _ uuid.UUID, _ string, _ string) (*basapi.WorkflowSummary, error) {
	return nil, assert.AnError
}

type stubAutomationEngine struct {
	session *stubEngineSession
}

type failingAutomationEngine struct {
	session *failingEngineSession
}

func (e *failingAutomationEngine) Name() string { return "failing" }

func (e *failingAutomationEngine) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{}, nil
}

func (e *failingAutomationEngine) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	return e.session, nil
}

func (s *stubAutomationEngine) Name() string { return "stub" }

func (s *stubAutomationEngine) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{}, nil
}

func (s *stubAutomationEngine) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	if s.session == nil {
		s.session = &stubEngineSession{}
	}
	return s.session, nil
}

type stubEngineSession struct {
	lastInputValue string
}

type failingEngineSession struct{}

func (*failingEngineSession) Run(context.Context, contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	return contracts.StepOutcome{}, assert.AnError
}

func (*failingEngineSession) Reset(context.Context) error { return nil }

func (*failingEngineSession) Close(context.Context) error { return nil }

func (*failingEngineSession) GetStorageState(context.Context) (json.RawMessage, error) {
	return nil, nil
}

func (s *stubEngineSession) Run(_ context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	if input := instruction.Action.GetInput(); input != nil {
		s.lastInputValue = input.GetValue()
	}
	return contracts.StepOutcome{Success: true}, nil
}

func (s *stubEngineSession) Reset(context.Context) error { return nil }

func (s *stubEngineSession) Close(context.Context) error { return nil }

func (s *stubEngineSession) GetStorageState(context.Context) (json.RawMessage, error) {
	return nil, nil
}

type stubExecutionWriter struct{}

func (s *stubExecutionWriter) RecordStepOutcome(context.Context, contracts.ExecutionPlan, contracts.StepOutcome) (executionwriter.RecordResult, error) {
	return executionwriter.RecordResult{}, nil
}

func (s *stubExecutionWriter) RecordTelemetry(context.Context, contracts.ExecutionPlan, contracts.StepTelemetry) error {
	return nil
}

func (s *stubExecutionWriter) MarkCrash(context.Context, uuid.UUID, contracts.StepFailure) error {
	return nil
}

func (s *stubExecutionWriter) UpdateCheckpoint(context.Context, uuid.UUID, int, int) error {
	return nil
}

func (s *stubExecutionWriter) RecordExecutionArtifacts(context.Context, contracts.ExecutionPlan, []executionwriter.ExternalArtifact) error {
	return nil
}

func (s *stubExecutionWriter) SetArtifactConfig(*config.ArtifactCollectionSettings) {}

func (s *stubExecutionWriter) GetArtifactConfig() config.ArtifactCollectionSettings {
	return config.ArtifactCollectionSettings{}
}

func (s *stubExecutionWriter) SetArtifactConfigForExecution(uuid.UUID, *config.ArtifactCollectionSettings) {
}

func (s *stubExecutionWriter) ForgetExecution(uuid.UUID) {}

var (
	_ executionwriter.ExecutionWriter = (*stubExecutionWriter)(nil)
	_ WorkflowResolver                = (*stubWorkflowResolver)(nil)
	_ engine.AutomationEngine         = (*stubAutomationEngine)(nil)
	_ engine.EngineSession            = (*stubEngineSession)(nil)
)
