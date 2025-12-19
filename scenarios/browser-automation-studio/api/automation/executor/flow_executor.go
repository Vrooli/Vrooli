package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/state"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// flow_executor.go isolates graph/loop orchestration so complex flow support
// has a single home instead of being buried in the SimpleExecutor file.

const (
	defaultLoopItemVar  = "loop.item"
	defaultLoopIndexVar = "loop.index"
)

func (e *SimpleExecutor) executeGraph(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, execState *state.ExecutionState, reuseMode engine.SessionReuseMode) (engine.EngineSession, error) {
	stepMap := indexGraph(req.Plan.Graph)
	current := firstStep(req.Plan.Graph)
	visited := 0
	maxVisited := len(stepMap) * 10
	var lastFailure error

	for current != nil {
		visited++
		if maxVisited > 0 && visited > maxVisited {
			return session, fmt.Errorf("graph execution exceeded step budget (%d)", maxVisited)
		}
		if ctx.Err() != nil {
			instr := planStepToInstruction(*current)
			if _, err := e.recordTerminatedStep(ctx, req, instr, ctx.Err()); err != nil {
				return session, err
			}
			return session, ctx.Err()
		}

		outcome, updatedSession, err := e.executePlanStep(ctx, req, execCtx, eng, spec, session, *current, execState, reuseMode)
		if err != nil {
			return session, err
		}
		session = updatedSession
		if !outcome.Success {
			lastFailure = fmt.Errorf("step %d failed: %s", outcome.StepIndex, e.failureMessage(outcome))
		}

		nextID := e.nextNodeID(*current, outcome)
		if nextID == "" {
			if lastFailure != nil {
				return session, lastFailure
			}
			return session, nil
		}
		next, ok := stepMap[nextID]
		if !ok {
			return session, fmt.Errorf("graph references missing node %s", nextID)
		}
		current = next
	}

	return session, lastFailure
}

func (e *SimpleExecutor) executePlanStep(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, step contracts.PlanStep, execState *state.ExecutionState, reuseMode engine.SessionReuseMode) (contracts.StepOutcome, engine.EngineSession, error) {
	stepType := PlanStepType(step)
	logrus.WithFields(logrus.Fields{
		"execution_id": req.Plan.ExecutionID,
		"node_id":      step.NodeID,
		"step_type":    stepType,
		"is_subflow":   isSubflowPlanStep(step),
	}).Debug("Executing plan step")

	if strings.EqualFold(strings.TrimSpace(stepType), "workflowcall") {
		return contracts.StepOutcome{}, session, fmt.Errorf("unsupported step type 'workflowCall'; use 'subflow' instead")
	}
	if isSubflowPlanStep(step) {
		logrus.WithFields(logrus.Fields{
			"execution_id": req.Plan.ExecutionID,
			"node_id":      step.NodeID,
		}).Debug("Executing subflow")
		return e.executeSubflow(ctx, req, execCtx, eng, spec, session, step, execState, reuseMode)
	}

	if session == nil {
		fields := logrus.Fields{
			"execution_id": req.Plan.ExecutionID,
			"node_id":      step.NodeID,
		}
		if deadline, ok := ctx.Deadline(); ok {
			fields["deadline"] = deadline
			fields["time_remaining"] = time.Until(deadline).String()
		}
		logrus.WithFields(fields).Debug("Starting new session")

		newSession, err := eng.StartSession(ctx, spec)
		if err != nil {
			return contracts.StepOutcome{}, session, fmt.Errorf("start session: %w", err)
		}
		session = newSession
	}

	step = e.interpolatePlanStep(step, execState)

	if IsPlanStepActionType(step, basactions.ActionType_ACTION_TYPE_LOOP) && step.Loop != nil {
		return e.executeLoop(ctx, req, execCtx, eng, spec, session, step, execState, reuseMode)
	}

	// Built-in variable mutation node used to support while/forEach flows without engine involvement.
	if isSetVariablePlanStep(step) {
		stepParams := PlanStepParams(step)
		outcome, err := e.applySetVariable(ctx, req, step.Index, stepType, step.NodeID, stepParams, execState)
		return outcome, session, err
	}

	instruction := planStepToInstruction(step)
	stepIndex := instruction.Index

	stepCtx, cancel := context.WithCancel(ctx)
	startedAt := time.Now().UTC()
	attempt := 1

	e.emitEvent(stepCtx, req, contracts.EventKindStepStarted, &stepIndex, &attempt, map[string]any{
		"node_id":   instruction.NodeID,
		"step_type": InstructionStepType(instruction),
	})

	stopHeartbeat := e.startHeartbeat(stepCtx, req, instruction.Index, attempt, startedAt)

	outcome, runErr := e.runWithRetries(stepCtx, req, session, instruction)
	if stopHeartbeat != nil {
		stopHeartbeat()
	}
	cancel()

	normalized := e.normalizeOutcome(req.Plan, instruction, attempt, startedAt, outcome, runErr)

	recordResult, recordErr := req.Recorder.RecordStepOutcome(ctx, req.Plan, normalized)
	if recordErr != nil {
		return normalized, session, fmt.Errorf("record step outcome: %w", recordErr)
	}

	payload := map[string]any{
		"outcome":   normalized,
		"artifacts": recordResult.ArtifactIDs,
	}
	if recordResult.TimelineArtifactID != nil {
		payload["timeline_artifact_id"] = *recordResult.TimelineArtifactID
	}

	eventKind := contracts.EventKindStepCompleted
	if !normalized.Success {
		eventKind = contracts.EventKindStepFailed
	}
	e.emitEvent(ctx, req, eventKind, &instruction.Index, &attempt, payload)

	// Store extracted data to execState if storeResult is specified
	if normalized.Success && normalized.ExtractedData != nil {
		instrParams := InstructionParams(instruction)
		if storeKey := state.StringValue(instrParams, "storeResult"); storeKey != "" {
			// Store the raw ExtractedData directly - it's not wrapped at this point
			// (wrapping only happens when creating database artifacts in db_recorder)
			execState.Set(storeKey, normalized.ExtractedData)
		}
	}

	newSession, resetErr := e.maybeResetSession(ctx, eng, spec, session, reuseMode)
	if resetErr != nil {
		return normalized, session, fmt.Errorf("reset session: %w", resetErr)
	}
	session = newSession

	if runErr != nil {
		return normalized, session, runErr
	}

	return normalized, session, nil
}

type loopExecutionResult struct {
	iterations  int
	lastOutcome contracts.StepOutcome
	session     engine.EngineSession
}

func (e *SimpleExecutor) executeLoop(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, step contracts.PlanStep, execState *state.ExecutionState, reuseMode engine.SessionReuseMode) (contracts.StepOutcome, engine.EngineSession, error) {
	stepParams := PlanStepParams(step)
	loopType := strings.ToLower(strings.TrimSpace(state.StringValue(stepParams, "loopType")))
	if loopType == "" {
		return contracts.StepOutcome{}, session, fmt.Errorf("loop node %s missing loopType", step.NodeID)
	}

	maxIterations := state.IntValue(stepParams, "loopMaxIterations")
	if maxIterations <= 0 {
		maxIterations = 100
	}

	// Use the loop handler registry for extensibility.
	// New loop types can be added by registering handlers via RegisterLoopHandler().
	handler := GetLoopHandler(loopType)
	if handler == nil {
		return contracts.StepOutcome{}, session, fmt.Errorf("loop node %s uses unsupported loopType %s (supported: %v)", step.NodeID, loopType, SupportedLoopTypes())
	}

	lctx := LoopContext{
		Ctx:           ctx,
		Request:       req,
		ExecCtx:       execCtx,
		Engine:        eng,
		Spec:          spec,
		Session:       session,
		Step:          step,
		State:         execState,
		ReuseMode:     reuseMode,
		MaxIterations: maxIterations,
	}

	result, err := handler.Execute(e, lctx)
	if err != nil {
		return contracts.StepOutcome{}, session, err
	}

	session = result.session

	// Record synthetic outcome for the loop node itself.
	loopOutcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    req.Plan.ExecutionID,
		StepIndex:      step.Index,
		NodeID:         step.NodeID,
		StepType:       PlanStepType(step),
		Success:        true,
		StartedAt:      time.Now().UTC(),
		CompletedAt: func() *time.Time {
			t := time.Now().UTC()
			return &t
		}(),
		Notes: map[string]string{
			"iterations": fmt.Sprintf("%d", result.iterations),
		},
	}
	if result.lastOutcome.Failure != nil && !result.lastOutcome.Success {
		loopOutcome.Success = false
		loopOutcome.Failure = result.lastOutcome.Failure
	}

	recordResult, recordErr := req.Recorder.RecordStepOutcome(ctx, req.Plan, loopOutcome)
	if recordErr != nil {
		return loopOutcome, session, fmt.Errorf("record loop outcome: %w", recordErr)
	}
	payload := map[string]any{
		"outcome":   loopOutcome,
		"artifacts": recordResult.ArtifactIDs,
	}
	if recordResult.TimelineArtifactID != nil {
		payload["timeline_artifact_id"] = *recordResult.TimelineArtifactID
	}
	e.emitEvent(ctx, req, contracts.EventKindStepCompleted, &step.Index, intPtr(1), payload)

	return loopOutcome, session, nil
}

// NOTE: Loop handlers (runRepeatLoop, runForEachLoop, runWhileLoop) have been
// moved to loop_registry.go for extensibility. New loop types can be added by
// implementing LoopHandler and calling RegisterLoopHandler().

func (e *SimpleExecutor) executeSubflow(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, step contracts.PlanStep, execState *state.ExecutionState, reuseMode engine.SessionReuseMode) (contracts.StepOutcome, engine.EngineSession, error) {
	stepCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now().UTC()
	attempt := 1
	stepIndex := step.Index
	stepTypeStr := PlanStepType(step)

	e.emitEvent(stepCtx, req, contracts.EventKindStepStarted, &stepIndex, &attempt, map[string]any{
		"node_id":   step.NodeID,
		"step_type": stepTypeStr,
	})
	stopHeartbeat := e.startHeartbeat(stepCtx, req, stepIndex, attempt, startedAt)

	outcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    req.Plan.ExecutionID,
		StepIndex:      step.Index,
		Attempt:        attempt,
		NodeID:         step.NodeID,
		StepType:       stepTypeStr,
		Success:        true,
		StartedAt:      startedAt,
	}

	updatedSession, runErr := e.runSubflow(stepCtx, req, execCtx, eng, spec, session, step, execState, reuseMode)

	if stopHeartbeat != nil {
		stopHeartbeat()
	}

	normalized := e.normalizeOutcome(req.Plan, planStepToInstruction(step), attempt, startedAt, outcome, runErr)

	recordResult, recordErr := req.Recorder.RecordStepOutcome(ctx, req.Plan, normalized)
	if recordErr != nil {
		return normalized, updatedSession, fmt.Errorf("record subflow outcome: %w", recordErr)
	}

	payload := map[string]any{
		"outcome":   normalized,
		"artifacts": recordResult.ArtifactIDs,
	}
	if recordResult.TimelineArtifactID != nil {
		payload["timeline_artifact_id"] = *recordResult.TimelineArtifactID
	}

	eventKind := contracts.EventKindStepCompleted
	if !normalized.Success {
		eventKind = contracts.EventKindStepFailed
	}
	e.emitEvent(ctx, req, eventKind, &stepIndex, &attempt, payload)

	return normalized, updatedSession, runErr
}

func (e *SimpleExecutor) runSubflow(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, step contracts.PlanStep, execState *state.ExecutionState, reuseMode engine.SessionReuseMode) (engine.EngineSession, error) {
	fields := logrus.Fields{
		"execution_id":  req.Plan.ExecutionID,
		"node_id":       step.NodeID,
		"current_depth": len(execCtx.callStack),
		"max_depth":     execCtx.maxDepth,
	}
	if deadline, ok := ctx.Deadline(); ok {
		fields["deadline"] = deadline
		fields["time_remaining"] = time.Until(deadline).String()
	}
	logrus.WithFields(fields).Debug("Entering subflow")

	if execCtx.maxDepth > 0 && len(execCtx.callStack) >= execCtx.maxDepth {
		return session, fmt.Errorf("subflow depth exceeded (max %d)", execCtx.maxDepth)
	}

	specData, err := parseSubflowSpec(step)
	if err != nil {
		return session, err
	}
	logrus.WithFields(logrus.Fields{
		"execution_id": req.Plan.ExecutionID,
		"node_id":      step.NodeID,
	}).Debug("Parsed subflow spec")

	// Detect reference loops.
	if specData.workflowID != nil {
		for _, id := range execCtx.callStack {
			if id == *specData.workflowID {
				return session, fmt.Errorf("subflow recursion detected for workflow %s", specData.workflowID.String())
			}
		}
	}

	childWorkflow, childWorkflowID, err := resolveSubflowWorkflow(ctx, req, specData)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"execution_id": req.Plan.ExecutionID,
			"node_id":      step.NodeID,
			"error":        err.Error(),
		}).Debug("Failed to resolve subflow workflow")
		return session, err
	}
	logrus.WithFields(logrus.Fields{
		"execution_id":      req.Plan.ExecutionID,
		"node_id":           step.NodeID,
		"child_workflow_id": childWorkflowID,
	}).Debug("Resolved subflow workflow")

	childPlan, _, err := BuildContractsPlanWithCompiler(ctx, req.Plan.ExecutionID, childWorkflow, execCtx.compiler)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"execution_id":      req.Plan.ExecutionID,
			"child_workflow_id": childWorkflowID,
			"error":             err.Error(),
		}).Debug("Failed to compile subflow")
		return session, fmt.Errorf("compile subflow %s: %w", childWorkflowID.String(), err)
	}
	logrus.WithFields(logrus.Fields{
		"execution_id":      req.Plan.ExecutionID,
		"child_workflow_id": childWorkflowID,
		"step_count":        len(childPlan.Instructions),
	}).Debug("Compiled subflow plan")

	childCount := planStepCount(childPlan)
	baseIndex := execState.AllocateIndexRange(childCount)
	rebasePlanIndices(&childPlan, baseIndex)

	// Build child state with proper namespace semantics:
	// - @store/: Child receives copy of parent's store, merges back on completion
	// - @params/: Default: inherit parent's params. Override: use ONLY specified params
	// - @env/: Always inherited from parent, read-only
	childState := buildSubflowState(execState, specData.params)
	childState.SetNextIndexFromPlan(childPlan)

	childReq := req
	childReq.Plan = childPlan
	childReq.SubflowStack = append(append([]uuid.UUID{}, execCtx.callStack...), childWorkflowID)
	childReq.EngineCaps = &execCtx.caps

	childCtx := execCtx
	childCtx.callStack = childReq.SubflowStack

	logrus.WithFields(logrus.Fields{
		"execution_id":      req.Plan.ExecutionID,
		"child_workflow_id": childWorkflowID,
	}).Debug("Executing subflow plan recursively")

	updatedSession, err := e.runPlan(ctx, childReq, childCtx, eng, spec, session, childState, reuseMode)

	logFields := logrus.Fields{
		"execution_id":      req.Plan.ExecutionID,
		"child_workflow_id": childWorkflowID,
		"success":           err == nil,
	}
	if err != nil {
		logFields["error"] = err.Error()
	}
	logrus.WithFields(logFields).Debug("Subflow execution completed")

	// Merge only @store/ back from child to parent (params and env are not propagated)
	mergeSubflowStore(execState, childState)
	return updatedSession, err
}

// buildSubflowState creates child state with proper namespace semantics for subflow execution.
// - @store/: Copy of parent's store
// - @params/: If params specified, use ONLY those; otherwise inherit parent's params
// - @env/: Always inherited from parent
func buildSubflowState(parentState *state.ExecutionState, subflowParams map[string]any) *state.ExecutionState {
	if parentState == nil {
		return state.NewFromStore(nil)
	}

	// Copy parent's store
	childStore := parentState.CopyStore()

	// Determine child params: override if specified, otherwise inherit
	var childParams map[string]any
	if len(subflowParams) > 0 {
		// Override: use ONLY specified params (no inheritance)
		childParams = copyMap(subflowParams)
	} else {
		// Inherit: copy parent's params
		childParams = parentState.CopyParams()
	}

	// Inherit env unchanged
	childEnv := parentState.CopyEnv()

	return state.New(childStore, childParams, childEnv)
}

// mergeSubflowStore merges child's @store/ back to parent after subflow completion.
// Only store is merged; params and env changes in child are discarded.
func mergeSubflowStore(parentState, childState *state.ExecutionState) {
	if parentState == nil || childState == nil {
		return
	}

	// Merge child's store to parent
	childStore := childState.CopyStore()
	parentState.Merge(childStore)
}

type loopControl struct {
	Break       bool
	LastOutcome contracts.StepOutcome
}

func (e *SimpleExecutor) executeGraphIteration(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, graph *contracts.PlanGraph, execState *state.ExecutionState, reuseMode engine.SessionReuseMode) (loopControl, engine.EngineSession, error) {
	if graph == nil || len(graph.Steps) == 0 {
		return loopControl{}, session, nil
	}

	stepMap := indexGraph(graph)
	current := firstStep(graph)
	visited := 0
	maxVisited := len(stepMap) * 10
	var last contracts.StepOutcome

	for current != nil {
		visited++
		if maxVisited > 0 && visited > maxVisited {
			return loopControl{}, session, fmt.Errorf("loop body exceeded step budget (%d)", maxVisited)
		}

		outcome, updatedSession, err := e.executePlanStep(ctx, req, execCtx, eng, spec, session, *current, execState, reuseMode)
		if err != nil {
			return loopControl{}, session, err
		}
		session = updatedSession
		last = outcome

		nextID := e.nextNodeID(*current, outcome)
		if nextID == "" {
			break
		}
		if nextID == "__loop_break__" {
			return loopControl{Break: true, LastOutcome: outcome}, session, nil
		}
		if nextID == "__loop_continue__" {
			break
		}
		next, ok := stepMap[nextID]
		if !ok {
			return loopControl{}, session, fmt.Errorf("loop body references missing node %s", nextID)
		}
		current = next
	}

	return loopControl{LastOutcome: last}, session, nil
}

// isSetVariableStep checks if a step type represents a set_variable action.
// Accepts either a string type or uses Action-aware matching when available.
func isSetVariableStep(stepType string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(stepType, "_", ""))
	return normalized == "setvariable"
}

// isSetVariableInstruction checks if an instruction is a set_variable action.
// Prefers Action.Type over the deprecated Type field.
func isSetVariableInstruction(instr contracts.CompiledInstruction) bool {
	return IsActionType(instr, basactions.ActionType_ACTION_TYPE_SET_VARIABLE)
}

// isSetVariablePlanStep checks if a plan step is a set_variable action.
// Prefers Action.Type over the deprecated Type field.
func isSetVariablePlanStep(step contracts.PlanStep) bool {
	return IsPlanStepActionType(step, basactions.ActionType_ACTION_TYPE_SET_VARIABLE)
}

// isSubflowStep checks if a step type represents a subflow action.
func isSubflowStep(stepType string) bool {
	return strings.EqualFold(strings.TrimSpace(stepType), "subflow")
}

// isSubflowInstruction checks if an instruction is a subflow action.
// Prefers Action.Type over the deprecated Type field.
func isSubflowInstruction(instr contracts.CompiledInstruction) bool {
	return IsActionType(instr, basactions.ActionType_ACTION_TYPE_SUBFLOW)
}

// isSubflowPlanStep checks if a plan step is a subflow action.
// Prefers Action.Type over the deprecated Type field.
func isSubflowPlanStep(step contracts.PlanStep) bool {
	return IsPlanStepActionType(step, basactions.ActionType_ACTION_TYPE_SUBFLOW)
}

// applySetVariable handles executor-scoped variable mutations without invoking an engine.
func (e *SimpleExecutor) applySetVariable(ctx context.Context, req Request, stepIndex int, stepType, nodeID string, params map[string]any, execState *state.ExecutionState) (contracts.StepOutcome, error) {
	name := state.StringValue(params, "name")
	if name == "" {
		name = state.StringValue(params, "variable")
	}
	if name == "" {
		name = state.StringValue(params, "variableName")
	}
	if name == "" {
		return contracts.StepOutcome{}, fmt.Errorf("set_variable node %s missing name", nodeID)
	}
	value := firstPresent(params, "value", "variableValue")
	valueType := state.StringValue(params, "valueType")
	if valueType == "" {
		valueType = state.StringValue(params, "variableType")
	}
	execState.Set(name, state.NormalizeVariableValue(value, valueType))

	outcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    req.Plan.ExecutionID,
		StepIndex:      stepIndex,
		Attempt:        1,
		NodeID:         nodeID,
		StepType:       stepType,
		Success:        true,
		StartedAt:      time.Now().UTC(),
	}
	end := outcome.StartedAt
	outcome.CompletedAt = &end

	recordResult, recordErr := req.Recorder.RecordStepOutcome(ctx, req.Plan, outcome)
	if recordErr != nil {
		return outcome, recordErr
	}

	payload := map[string]any{
		"outcome":   outcome,
		"artifacts": recordResult.ArtifactIDs,
	}
	e.emitEvent(ctx, req, contracts.EventKindStepCompleted, &stepIndex, intPtr(1), payload)
	return outcome, nil
}

type subflowSpec struct {
	workflowID      *uuid.UUID
	workflowVersion *int
	workflowPath    string
	inlineDef       map[string]any
	params          map[string]any
}

func parseSubflowSpec(step contracts.PlanStep) (subflowSpec, error) {
	spec := subflowSpec{}
	stepParams := PlanStepParams(step)
	if idStr := state.StringValue(stepParams, "workflowId"); strings.TrimSpace(idStr) != "" {
		if parsed, err := uuid.Parse(idStr); err == nil {
			spec.workflowID = &parsed
		} else {
			return spec, fmt.Errorf("subflow %s has invalid workflowId: %w", step.NodeID, err)
		}
	}
	if version := state.IntValue(stepParams, "workflowVersion"); version > 0 {
		spec.workflowVersion = &version
	}
	if pathStr := state.StringValue(stepParams, "workflowPath"); strings.TrimSpace(pathStr) != "" {
		spec.workflowPath = strings.TrimSpace(pathStr)
	}
	if rawDef, ok := stepParams["workflowDefinition"]; ok {
		if def, ok := rawDef.(map[string]any); ok && len(def) > 0 {
			spec.inlineDef = def
		}
	}
	if rawParams, ok := stepParams["parameters"]; ok {
		if params, ok := rawParams.(map[string]any); ok && len(params) > 0 {
			spec.params = params
		}
	}
	if spec.workflowID == nil && spec.workflowPath == "" && spec.inlineDef == nil {
		return spec, fmt.Errorf("subflow %s missing workflowId, workflowPath, or workflowDefinition", step.NodeID)
	}
	return spec, nil
}

func resolveSubflowWorkflow(ctx context.Context, req Request, spec subflowSpec) (*basapi.WorkflowSummary, uuid.UUID, error) {
	if spec.workflowID != nil {
		if req.WorkflowResolver == nil {
			return nil, uuid.Nil, fmt.Errorf("subflow requires workflow resolver to fetch %s", spec.workflowID.String())
		}
		var wf *basapi.WorkflowSummary
		var err error
		if spec.workflowVersion != nil {
			wf, err = req.WorkflowResolver.GetWorkflowVersion(ctx, *spec.workflowID, *spec.workflowVersion)
		} else {
			wf, err = req.WorkflowResolver.GetWorkflow(ctx, *spec.workflowID)
		}
		if err != nil {
			return nil, uuid.Nil, err
		}
		workflowID, _ := uuid.Parse(wf.GetId())
		return wf, workflowID, nil
	}

	if spec.workflowPath != "" {
		if req.WorkflowResolver == nil {
			return nil, uuid.Nil, fmt.Errorf("subflow requires workflow resolver to fetch %q", spec.workflowPath)
		}
		wf, err := req.WorkflowResolver.GetWorkflowByProjectPath(ctx, req.Plan.WorkflowID, spec.workflowPath, req.ProjectRoot)
		if err != nil {
			return nil, uuid.Nil, err
		}
		workflowID, _ := uuid.Parse(wf.GetId())
		return wf, workflowID, nil
	}

	// Inline workflow definition - create a WorkflowSummary with the inline definition
	// NOTE: Inline subflows with workflowDefinition are a complex edge case.
	// The common path (workflow_id or workflow_path reference) is fully V2-compatible.
	// Inline definitions would require parsing nested ActionDefinition protos from a map,
	// which is rarely used. For now, inline subflows return an empty flow.
	id := uuid.New()
	flowDef := &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{},
		Edges: []*basworkflows.WorkflowEdgeV2{},
	}
	return &basapi.WorkflowSummary{
		Id:             id.String(),
		FlowDefinition: flowDef,
	}, id, nil
}

func planStepCount(plan contracts.ExecutionPlan) int {
	if plan.Graph != nil {
		return maxGraphIndex(plan.Graph) + 1
	}
	maxIdx := -1
	for _, instr := range plan.Instructions {
		if instr.Index > maxIdx {
			maxIdx = instr.Index
		}
	}
	return maxIdx + 1
}

func rebasePlanIndices(plan *contracts.ExecutionPlan, base int) {
	if plan == nil || base == 0 {
		return
	}
	for i := range plan.Instructions {
		plan.Instructions[i].Index += base
	}
	if plan.Graph != nil {
		rebaseGraph(plan.Graph, base)
	}
}

func rebaseGraph(graph *contracts.PlanGraph, base int) {
	for i := range graph.Steps {
		graph.Steps[i].Index += base
		if graph.Steps[i].Loop != nil {
			rebaseGraph(graph.Steps[i].Loop, base)
		}
	}
}

func copyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
