package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/database"
)

// flow_executor.go isolates graph/loop orchestration so complex flow support
// has a single home instead of being buried in the SimpleExecutor file.

const (
	defaultLoopItemVar  = "loop.item"
	defaultLoopIndexVar = "loop.index"
)

func (e *SimpleExecutor) executeGraph(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, state *flowState, reuseMode engine.SessionReuseMode) (engine.EngineSession, error) {
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

		outcome, updatedSession, err := e.executePlanStep(ctx, req, execCtx, eng, spec, session, *current, state, reuseMode)
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

func (e *SimpleExecutor) executePlanStep(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, step contracts.PlanStep, state *flowState, reuseMode engine.SessionReuseMode) (contracts.StepOutcome, engine.EngineSession, error) {
	if strings.EqualFold(strings.TrimSpace(step.Type), "workflowcall") {
		return contracts.StepOutcome{}, session, fmt.Errorf("workflowCall nodes are no longer supported; use subflow instead")
	}
	if isSubflowStep(step.Type) {
		return e.executeSubflow(ctx, req, execCtx, eng, spec, session, step, state, reuseMode)
	}

	if session == nil {
		newSession, err := eng.StartSession(ctx, spec)
		if err != nil {
			return contracts.StepOutcome{}, session, fmt.Errorf("start session: %w", err)
		}
		session = newSession
	}

	step = e.interpolatePlanStep(step, state)

	if strings.EqualFold(step.Type, "loop") && step.Loop != nil {
		return e.executeLoop(ctx, req, execCtx, eng, spec, session, step, state, reuseMode)
	}

	// Built-in variable mutation node used to support while/forEach flows without engine involvement.
	if isSetVariableStep(step.Type) {
		outcome, err := e.applySetVariable(ctx, req, step.Index, step.Type, step.NodeID, step.Params, state)
		return outcome, session, err
	}

	instruction := planStepToInstruction(step)
	stepIndex := instruction.Index

	stepCtx, cancel := context.WithCancel(ctx)
	startedAt := time.Now().UTC()
	attempt := 1

	e.emitEvent(stepCtx, req, contracts.EventKindStepStarted, &stepIndex, &attempt, map[string]any{
		"node_id":   instruction.NodeID,
		"step_type": instruction.Type,
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

	// Store extracted data to flowState if storeResult is specified
	if normalized.Success && normalized.ExtractedData != nil {
		if storeKey := stringValue(instruction.Params, "storeResult"); storeKey != "" {
			// Store the raw ExtractedData directly - it's not wrapped at this point
			// (wrapping only happens when creating database artifacts in db_recorder)
			state.set(storeKey, normalized.ExtractedData)
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

func (e *SimpleExecutor) executeLoop(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, step contracts.PlanStep, state *flowState, reuseMode engine.SessionReuseMode) (contracts.StepOutcome, engine.EngineSession, error) {
	loopType := strings.ToLower(strings.TrimSpace(stringValue(step.Params, "loopType")))
	if loopType == "" {
		return contracts.StepOutcome{}, session, fmt.Errorf("loop node %s missing loopType", step.NodeID)
	}

	maxIterations := intValue(step.Params, "loopMaxIterations")
	if maxIterations <= 0 {
		maxIterations = 100
	}

	var lastOutcome contracts.StepOutcome
	var iterations int
	switch loopType {
	case "repeat":
		iterations = intValue(step.Params, "loopCount")
		if iterations <= 0 {
			return contracts.StepOutcome{}, session, fmt.Errorf("loop node %s repeat requires loopCount > 0", step.NodeID)
		}
		iterations = minInt(iterations, maxIterations)
		if iterations == 0 {
			return contracts.StepOutcome{}, session, fmt.Errorf("loop node %s has zero iterations after clamping", step.NodeID)
		}
		for i := 0; i < iterations; i++ {
			control, session, err := e.executeGraphIteration(ctx, req, execCtx, eng, spec, session, step.Loop, state, reuseMode)
			if err != nil {
				return contracts.StepOutcome{}, session, err
			}
			lastOutcome = control.LastOutcome
			if control.Break {
				break
			}
		}
	case "foreach":
		items := extractLoopItems(step.Params, state)
		if len(items) == 0 {
			break
		}
		maxIterations = minInt(maxIterations, len(items))
		itemVar := stringValue(step.Params, "loopItemVariable")
		if itemVar == "" {
			itemVar = stringValue(step.Params, "itemVariable")
		}
		if itemVar == "" {
			itemVar = defaultLoopItemVar
		}
		indexVar := stringValue(step.Params, "loopIndexVariable")
		if indexVar == "" {
			indexVar = stringValue(step.Params, "indexVariable")
		}
		if indexVar == "" {
			indexVar = defaultLoopIndexVar
		}
		executed := 0
		for i := 0; i < maxIterations; i++ {
			state.set(itemVar, items[i])
			state.set(indexVar, i)
			control, session, err := e.executeGraphIteration(ctx, req, execCtx, eng, spec, session, step.Loop, state, reuseMode)
			if err != nil {
				return contracts.StepOutcome{}, session, err
			}
			executed++
			lastOutcome = control.LastOutcome
			if control.Break {
				break
			}
		}
		iterations = executed
	case "while":
		iterations = 0
		for iterations < maxIterations {
			if !evaluateLoopCondition(step.Params, state) {
				break
			}
			control, session, err := e.executeGraphIteration(ctx, req, execCtx, eng, spec, session, step.Loop, state, reuseMode)
			if err != nil {
				return contracts.StepOutcome{}, session, err
			}
			lastOutcome = control.LastOutcome
			iterations++
			if control.Break {
				break
			}
		}
	default:
		return contracts.StepOutcome{}, session, fmt.Errorf("loop node %s uses unsupported loopType %s", step.NodeID, loopType)
	}

	// Record synthetic outcome for the loop node itself.
	loopOutcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    req.Plan.ExecutionID,
		StepIndex:      step.Index,
		NodeID:         step.NodeID,
		StepType:       step.Type,
		Success:        true,
		StartedAt:      time.Now().UTC(),
		CompletedAt: func() *time.Time {
			t := time.Now().UTC()
			return &t
		}(),
		Notes: map[string]string{
			"iterations": fmt.Sprintf("%d", iterations),
		},
	}
	if lastOutcome.Failure != nil && !lastOutcome.Success {
		loopOutcome.Success = false
		loopOutcome.Failure = lastOutcome.Failure
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

func (e *SimpleExecutor) executeSubflow(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, step contracts.PlanStep, state *flowState, reuseMode engine.SessionReuseMode) (contracts.StepOutcome, engine.EngineSession, error) {
	stepCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedAt := time.Now().UTC()
	attempt := 1
	stepIndex := step.Index

	e.emitEvent(stepCtx, req, contracts.EventKindStepStarted, &stepIndex, &attempt, map[string]any{
		"node_id":   step.NodeID,
		"step_type": step.Type,
	})
	stopHeartbeat := e.startHeartbeat(stepCtx, req, stepIndex, attempt, startedAt)

	outcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    req.Plan.ExecutionID,
		StepIndex:      step.Index,
		Attempt:        attempt,
		NodeID:         step.NodeID,
		StepType:       step.Type,
		Success:        true,
		StartedAt:      startedAt,
	}

	updatedSession, runErr := e.runSubflow(stepCtx, req, execCtx, eng, spec, session, step, state, reuseMode)

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

func (e *SimpleExecutor) runSubflow(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, step contracts.PlanStep, state *flowState, reuseMode engine.SessionReuseMode) (engine.EngineSession, error) {
	if execCtx.maxDepth > 0 && len(execCtx.callStack) >= execCtx.maxDepth {
		return session, fmt.Errorf("subflow depth exceeded (max %d)", execCtx.maxDepth)
	}

	specData, err := parseSubflowSpec(step)
	if err != nil {
		return session, err
	}

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
		return session, err
	}

	childPlan, _, err := BuildContractsPlanWithCompiler(ctx, req.Plan.ExecutionID, childWorkflow, execCtx.compiler)
	if err != nil {
		return session, fmt.Errorf("compile subflow %s: %w", childWorkflowID.String(), err)
	}

	childCount := planStepCount(childPlan)
	baseIndex := state.allocateIndexRange(childCount)
	rebasePlanIndices(&childPlan, baseIndex)

	seedVars := copyMap(state.vars)
	if len(specData.params) > 0 {
		for k, v := range specData.params {
			seedVars[k] = v
		}
	}

	childState := newFlowState(seedVars)
	childState.setNextIndexFromPlan(childPlan)

	childReq := req
	childReq.Plan = childPlan
	childReq.SubflowStack = append(append([]uuid.UUID{}, execCtx.callStack...), childWorkflowID)
	childReq.EngineCaps = &execCtx.caps

	childCtx := execCtx
	childCtx.callStack = childReq.SubflowStack

	updatedSession, err := e.runPlan(ctx, childReq, childCtx, eng, spec, session, childState, reuseMode)
	state.merge(childState.vars)
	return updatedSession, err
}

type loopControl struct {
	Break       bool
	LastOutcome contracts.StepOutcome
}

func (e *SimpleExecutor) executeGraphIteration(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, graph *contracts.PlanGraph, state *flowState, reuseMode engine.SessionReuseMode) (loopControl, engine.EngineSession, error) {
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

		outcome, updatedSession, err := e.executePlanStep(ctx, req, execCtx, eng, spec, session, *current, state, reuseMode)
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

func isSetVariableStep(stepType string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(stepType, "_", ""))
	return normalized == "setvariable"
}

func isSubflowStep(stepType string) bool {
	return strings.EqualFold(strings.TrimSpace(stepType), "subflow")
}

// applySetVariable handles executor-scoped variable mutations without invoking an engine.
func (e *SimpleExecutor) applySetVariable(ctx context.Context, req Request, stepIndex int, stepType, nodeID string, params map[string]any, state *flowState) (contracts.StepOutcome, error) {
	name := stringValue(params, "name")
	if name == "" {
		name = stringValue(params, "variable")
	}
	if name == "" {
		name = stringValue(params, "variableName")
	}
	if name == "" {
		return contracts.StepOutcome{}, fmt.Errorf("set_variable node %s missing name", nodeID)
	}
	value := firstPresent(params, "value", "variableValue")
	valueType := stringValue(params, "valueType")
	if valueType == "" {
		valueType = stringValue(params, "variableType")
	}
	state.set(name, normalizeVariableValue(value, valueType))

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
	workflowID *uuid.UUID
	inlineDef  map[string]any
	params     map[string]any
}

func parseSubflowSpec(step contracts.PlanStep) (subflowSpec, error) {
	spec := subflowSpec{}
	if idStr := stringValue(step.Params, "workflowId"); strings.TrimSpace(idStr) != "" {
		if parsed, err := uuid.Parse(idStr); err == nil {
			spec.workflowID = &parsed
		} else {
			return spec, fmt.Errorf("subflow %s has invalid workflowId: %w", step.NodeID, err)
		}
	}
	if rawDef, ok := step.Params["workflowDefinition"]; ok {
		if def, ok := rawDef.(map[string]any); ok && len(def) > 0 {
			spec.inlineDef = def
		}
	}
	if rawParams, ok := step.Params["parameters"]; ok {
		if params, ok := rawParams.(map[string]any); ok && len(params) > 0 {
			spec.params = params
		}
	}
	if spec.workflowID == nil && spec.inlineDef == nil {
		return spec, fmt.Errorf("subflow %s missing workflowId or workflowDefinition", step.NodeID)
	}
	return spec, nil
}

func resolveSubflowWorkflow(ctx context.Context, req Request, spec subflowSpec) (*database.Workflow, uuid.UUID, error) {
	if spec.workflowID != nil {
		if req.WorkflowResolver == nil {
			return nil, uuid.Nil, fmt.Errorf("subflow requires workflow resolver to fetch %s", spec.workflowID.String())
		}
		wf, err := req.WorkflowResolver.GetWorkflow(ctx, *spec.workflowID)
		if err != nil {
			return nil, uuid.Nil, err
		}
		return wf, wf.ID, nil
	}

	id := uuid.New()
	return &database.Workflow{
		ID:             id,
		FlowDefinition: database.JSONMap(spec.inlineDef),
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
