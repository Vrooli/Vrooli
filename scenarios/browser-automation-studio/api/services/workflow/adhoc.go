package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// executionNamespaceParams stores namespace-aware parameters for a specific execution.
// These are used by the async executor to initialize the namespace-aware state.
type executionNamespaceParams struct {
	ProjectRoot   string
	InitialParams map[string]any
	InitialStore  map[string]any
	Env           map[string]any
}

// executionParamsStore holds namespace params for executions.
// Access is synchronized since params are written during creation and read during async execution.
var executionParamsStore = struct {
	sync.RWMutex
	params map[uuid.UUID]*executionNamespaceParams
}{params: make(map[uuid.UUID]*executionNamespaceParams)}

// adhocFlowDefinition stores the flow definition and parameters for adhoc workflows.
// Since WorkflowIndex doesn't store FlowDefinition, we keep it in memory for the executor.
type adhocFlowDefinition struct {
	FlowDefinition map[string]any
	Parameters     map[string]any
}

// adhocFlowStore holds flow definitions for adhoc workflows.
// Access is synchronized since definitions are written during creation and read during async execution.
var adhocFlowStore = struct {
	sync.RWMutex
	flows map[uuid.UUID]*adhocFlowDefinition
}{flows: make(map[uuid.UUID]*adhocFlowDefinition)}

// storeExecutionParams stores namespace params for later retrieval by the async runner.
func storeExecutionParams(executionID uuid.UUID, params *executionNamespaceParams) {
	if params == nil {
		return
	}
	executionParamsStore.Lock()
	defer executionParamsStore.Unlock()
	executionParamsStore.params[executionID] = params
}

// getAndClearExecutionParams retrieves and removes namespace params for an execution.
func getAndClearExecutionParams(executionID uuid.UUID) *executionNamespaceParams {
	executionParamsStore.Lock()
	defer executionParamsStore.Unlock()
	params := executionParamsStore.params[executionID]
	delete(executionParamsStore.params, executionID)
	return params
}

// storeAdhocFlowDefinition stores the flow definition for an adhoc workflow.
// This is called by the WorkflowService when creating adhoc executions.
func (s *WorkflowService) storeAdhocFlowDefinition(workflowID uuid.UUID, flowDef map[string]any, params map[string]any) {
	adhocFlowStore.Lock()
	defer adhocFlowStore.Unlock()
	adhocFlowStore.flows[workflowID] = &adhocFlowDefinition{
		FlowDefinition: flowDef,
		Parameters:     params,
	}
}

// getAndClearAdhocFlowDefinition retrieves and removes the flow definition for an adhoc workflow.
func getAndClearAdhocFlowDefinition(workflowID uuid.UUID) *adhocFlowDefinition {
	adhocFlowStore.Lock()
	defer adhocFlowStore.Unlock()
	def := adhocFlowStore.flows[workflowID]
	delete(adhocFlowStore.flows, workflowID)
	return def
}

// getAdhocFlowDefinition retrieves (without removing) the flow definition for an adhoc workflow.
func getAdhocFlowDefinition(workflowID uuid.UUID) *adhocFlowDefinition {
	adhocFlowStore.RLock()
	defer adhocFlowStore.RUnlock()
	return adhocFlowStore.flows[workflowID]
}

// ExecuteAdhocWorkflow executes a workflow definition without persisting it to the database.
// This is useful for testing scenarios where workflows should be ephemeral and not pollute
// the database with test data. The workflow definition is validated and executed directly,
// with execution records still persisted for telemetry and replay purposes.
func (s *WorkflowService) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	// Validate workflow definition structure
	if flowDefinition == nil {
		return nil, errors.New("flow_definition is required")
	}

	nodes, hasNodes := flowDefinition["nodes"]
	if !hasNodes {
		return nil, errors.New("flow_definition must contain 'nodes' array")
	}

	nodesArray, ok := nodes.([]interface{})
	if !ok {
		return nil, errors.New("'nodes' must be an array")
	}

	if len(nodesArray) == 0 {
		return nil, errors.New("workflow must contain at least one node")
	}

	_, hasEdges := flowDefinition["edges"]
	if !hasEdges {
		return nil, errors.New("flow_definition must contain 'edges' array")
	}

	// Create ephemeral workflow (temporarily persisted to satisfy FK constraint)
	// This workflow will be auto-deleted when execution is cleaned up (ON DELETE CASCADE)
	// Add timestamp to name to avoid unique constraint violations on (name, folder_path)
	ephemeralID := uuid.New()
	ephemeralName := fmt.Sprintf("%s [adhoc-%s]", name, ephemeralID.String()[:8])

	// NOTE: WorkflowIndex doesn't store FlowDefinition - it's stored on disk.
	// For adhoc workflows, the flow definition is passed to the executor directly.
	ephemeralWorkflow := &database.Workflow{
		ID:        ephemeralID,
		Name:      ephemeralName,
		Version:   0, // Version 0 indicates adhoc/ephemeral workflow
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		// ProjectID is intentionally nil - adhoc workflows have no project context
		// FlowDefinition stored in memory, passed to executor
	}

	// Store flow definition for the async executor to use
	s.storeAdhocFlowDefinition(ephemeralID, flowDefinition, parameters)

	// Temporarily persist ephemeral workflow to satisfy executions.workflow_id FK constraint
	// This allows executions table to maintain referential integrity while still being ephemeral
	if err := s.repo.CreateWorkflow(ctx, ephemeralWorkflow); err != nil {
		return nil, fmt.Errorf("failed to create ephemeral workflow: %w", err)
	}

	// Create execution record (persists for telemetry/replay)
	// NOTE: ExecutionIndex only stores queryable fields.
	// WorkflowVersion, TriggerType, Parameters stored in result file.
	execution := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: ephemeralWorkflow.ID, // Reference ephemeral workflow ID
		Status:     "pending",
		StartedAt:  time.Now(),
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Execute asynchronously (same pattern as normal workflow execution)
	// The executeWorkflowAsync method works with any workflow struct,
	// whether persisted or ephemeral
	s.startExecutionRunner(execution, ephemeralWorkflow)
	go s.scheduleAdhocWorkflowCleanup(ephemeralWorkflow.ID, execution.ID)

	return execution, nil
}

// scheduleAdhocWorkflowCleanup monitors an adhoc execution until it reaches a terminal state,
// then waits for the retention period before cleaning up both the execution and ephemeral workflow.
func (s *WorkflowService) scheduleAdhocWorkflowCleanup(workflowID, executionID uuid.UUID) {
	if s == nil || s.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), adhocExecutionCleanupTimeout)
	defer cancel()

	ticker := time.NewTicker(adhocExecutionCleanupInterval)
	defer ticker.Stop()

	var terminalObservedAt time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		execution, err := s.repo.GetExecution(ctx, executionID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				terminalObservedAt = time.Now()
				break
			}
			if s.log != nil {
				s.log.WithError(err).WithField("execution_id", executionID).Warn("adhoc cleanup: unable to load execution status")
			}
			continue
		}
		if execution == nil {
			terminalObservedAt = time.Now()
			break
		}
		if IsTerminalExecutionStatus(execution.Status) {
			if terminalObservedAt.IsZero() {
				terminalObservedAt = time.Now()
			}
			if time.Since(terminalObservedAt) >= adhocExecutionRetentionPeriod {
				break
			}
			continue
		}
		terminalObservedAt = time.Time{}
	}

	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()
	if err := s.repo.DeleteExecution(cleanupCtx, executionID); err != nil && s.log != nil {
		s.log.WithError(err).WithField("execution_id", executionID).Warn("adhoc cleanup: failed to delete execution")
	}
	if err := s.repo.DeleteWorkflow(cleanupCtx, workflowID); err != nil && s.log != nil {
		s.log.WithError(err).WithField("workflow_id", workflowID).Warn("adhoc cleanup: failed to delete workflow")
	}
}

// ExecuteAdhocWorkflowWithParams executes an adhoc workflow with namespace-aware parameters.
// This is the preferred method for callers that support the new variable interpolation system
// with @store/, @params/, and @env/ namespaces.
//
// The namespace parameters are:
//   - ProjectRoot: Absolute path to project root for workflowPath resolution
//   - InitialParams: Read-only input parameters (@params/ namespace)
//   - InitialStore: Mutable runtime state (@store/ namespace)
//   - Env: Project/user configuration (@env/ namespace)
func (s *WorkflowService) ExecuteAdhocWorkflowWithParams(ctx context.Context, params AdhocExecutionParams) (*database.Execution, error) {
	// Validate workflow definition structure
	if params.FlowDefinition == nil {
		return nil, errors.New("flow_definition is required")
	}

	nodes, hasNodes := params.FlowDefinition["nodes"]
	if !hasNodes {
		return nil, errors.New("flow_definition must contain 'nodes' array")
	}

	nodesArray, ok := nodes.([]interface{})
	if !ok {
		return nil, errors.New("'nodes' must be an array")
	}

	if len(nodesArray) == 0 {
		return nil, errors.New("workflow must contain at least one node")
	}

	_, hasEdges := params.FlowDefinition["edges"]
	if !hasEdges {
		return nil, errors.New("flow_definition must contain 'edges' array")
	}

	// Create ephemeral workflow (temporarily persisted to satisfy FK constraint)
	ephemeralID := uuid.New()
	name := params.Name
	if name == "" {
		name = "adhoc-workflow"
	}
	ephemeralName := fmt.Sprintf("%s [adhoc-%s]", name, ephemeralID.String()[:8])

	// NOTE: WorkflowIndex doesn't store FlowDefinition - it's stored on disk.
	// For adhoc workflows, the flow definition is passed to the executor directly.
	ephemeralWorkflow := &database.Workflow{
		ID:        ephemeralID,
		Name:      ephemeralName,
		Version:   0, // Version 0 indicates adhoc/ephemeral workflow
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Determine initial store: prefer explicit InitialStore, fall back to LegacyParameters
	initialStore := params.InitialStore
	if len(initialStore) == 0 && len(params.LegacyParameters) > 0 {
		initialStore = params.LegacyParameters
	}

	// Build combined parameters for passing to executor (stored in memory, not DB)
	combinedParams := make(map[string]any)
	for k, v := range params.LegacyParameters {
		combinedParams[k] = v
	}
	if params.ProjectRoot != "" {
		combinedParams["__project_root"] = params.ProjectRoot
	}
	if len(params.InitialParams) > 0 {
		combinedParams["__initial_params"] = params.InitialParams
	}
	if len(initialStore) > 0 {
		combinedParams["__initial_store"] = initialStore
	}
	if len(params.Env) > 0 {
		combinedParams["__env"] = params.Env
	}

	// Store flow definition for the async executor to use
	s.storeAdhocFlowDefinition(ephemeralID, params.FlowDefinition, combinedParams)

	if err := s.repo.CreateWorkflow(ctx, ephemeralWorkflow); err != nil {
		return nil, fmt.Errorf("failed to create ephemeral workflow: %w", err)
	}

	// Create execution record
	// NOTE: ExecutionIndex only stores queryable fields.
	// WorkflowVersion, TriggerType, Parameters stored in result file.
	execution := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: ephemeralWorkflow.ID,
		Status:     "pending",
		StartedAt:  time.Now(),
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Store namespace params for retrieval by the async runner
	storeExecutionParams(execution.ID, &executionNamespaceParams{
		ProjectRoot:   params.ProjectRoot,
		InitialParams: params.InitialParams,
		InitialStore:  initialStore,
		Env:           params.Env,
	})

	// Execute asynchronously
	s.startExecutionRunner(execution, ephemeralWorkflow)
	go s.scheduleAdhocWorkflowCleanup(ephemeralWorkflow.ID, execution.ID)

	return execution, nil
}
