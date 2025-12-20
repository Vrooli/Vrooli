package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/state"
	"github.com/vrooli/browser-automation-studio/config"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// SimpleExecutor provides a minimal sequential executor that delegates step
// execution to an AutomationEngine and persists outcomes via a Recorder. It
// keeps telemetry/event emission contained so we can grow into the new
// architecture without touching handlers yet.
type SimpleExecutor struct {
	sequencer    events.Sequencer
	telemetrySeq map[string]uint64
	telemetryMu  sync.Mutex
}

type executionContext struct {
	caps      contracts.EngineCapabilities
	compiler  PlanCompiler
	maxDepth  int
	callStack []uuid.UUID
}

// NewSimpleExecutor constructs a SimpleExecutor. If no sequencer is supplied,
// a per-execution sequencer is used.
func NewSimpleExecutor(seq events.Sequencer) *SimpleExecutor {
	if seq == nil {
		seq = events.NewPerExecutionSequencer()
	}
	return &SimpleExecutor{
		sequencer:    seq,
		telemetrySeq: make(map[string]uint64),
	}
}

// Execute runs the plan sequentially. Retries/branching will be layered on
// later; this is the minimal happy path required to exercise the abstraction.
func (e *SimpleExecutor) Execute(ctx context.Context, req Request) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	startedAt := time.Now()
	engineName := req.EngineName
	defer func() {
		status := "success"
		if err != nil {
			status = "failed"
		}
		logrus.WithFields(logrus.Fields{
			"execution_id": req.Plan.ExecutionID,
			"workflow_id":  req.Plan.WorkflowID,
			"engine":       engineName,
			"status":       status,
			"duration_ms":  time.Since(startedAt).Milliseconds(),
		}).Info("Execution completed")
	}()

	// Log context deadline for debugging timeout issues
	if deadline, ok := ctx.Deadline(); ok {
		logrus.WithFields(logrus.Fields{
			"execution_id":   req.Plan.ExecutionID,
			"deadline":       deadline,
			"time_remaining": time.Until(deadline).String(),
		}).Debug("Execute called with context deadline")
	} else {
		logrus.WithField("execution_id", req.Plan.ExecutionID).Debug("Execute called without context deadline")
	}

	if err := e.validateRequest(req); err != nil {
		return err
	}

	if timeout := executionTimeout(req.Plan); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		logrus.WithFields(logrus.Fields{
			"execution_id": req.Plan.ExecutionID,
			"timeout":      timeout.String(),
		}).Debug("Set execution timeout")
	} else {
		logrus.WithField("execution_id", req.Plan.ExecutionID).Debug("No execution timeout set")
	}

	if engineName == "" {
		engineName = "playwright"
	}
	logrus.WithFields(logrus.Fields{
		"execution_id": req.Plan.ExecutionID,
		"workflow_id":  req.Plan.WorkflowID,
		"engine":       engineName,
	}).Info("Execution started")

	compiler := req.PlanCompiler
	if compiler == nil {
		compiler = PlanCompilerForEngine(engineName)
	}
	maxDepth := req.MaxSubflowDepth
	if maxDepth <= 0 {
		maxDepth = config.Load().Execution.MaxSubflowDepth
		if maxDepth <= 0 {
			maxDepth = 5
		}
	}

	eng, err := req.EngineFactory.Resolve(ctx, engineName)
	if err != nil {
		return fmt.Errorf("resolve engine %q: %w", engineName, err)
	}

	reuseMode := resolveReuseMode(req)
	requirements, requirementReasons := analyzeRequirements(req.Plan)

	// Capabilities are fetched up front to allow callers to validate or log.
	caps := req.EngineCaps
	if caps == nil {
		fields := logrus.Fields{
			"execution_id": req.Plan.ExecutionID,
			"engine":       engineName,
		}
		if deadline, ok := ctx.Deadline(); ok {
			fields["deadline"] = deadline
			fields["time_remaining"] = time.Until(deadline).String()
		}
		logrus.WithFields(fields).Debug("Fetching engine capabilities")

		descriptor, err := eng.Capabilities(ctx)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"execution_id": req.Plan.ExecutionID,
				"engine":       engineName,
				"error":        err.Error(),
			}).Debug("Engine capabilities fetch failed")
			return fmt.Errorf("engine capabilities: %w", err)
		}
		logrus.WithField("execution_id", req.Plan.ExecutionID).Debug("Engine capabilities fetched successfully")
		caps = &descriptor
	}
	if err := caps.Validate(); err != nil {
		return fmt.Errorf("engine capabilities invalid: %w", err)
	}
	if gap := caps.CheckCompatibility(requirements); !gap.Satisfied() {
		return &CapabilityError{
			Engine:    eng.Name(),
			Missing:   gap.Missing,
			Warnings:  gap.Warnings,
			Reasons:   filterReasons(requirementReasons, gap.Missing),
			Execution: req.Plan.ExecutionID,
			Workflow:  req.Plan.WorkflowID,
		}
	}

	viewportWidth, viewportHeight := extractViewport(req.Plan.Metadata)
	frameStreamingConfig := extractFrameStreamingConfig(req.Plan.Metadata, req.Plan.ExecutionID)
	spec := engine.SessionSpec{
		ExecutionID:    req.Plan.ExecutionID,
		WorkflowID:     req.Plan.WorkflowID,
		ReuseMode:      reuseMode,
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
		Labels:         map[string]string{},
		Capabilities:   requirements,
		FrameStreaming: frameStreamingConfig,
	}

	var session engine.EngineSession
	defer func() {
		if session != nil {
			_ = session.Close(context.Background()) // Best-effort cleanup.
		}
	}()

	// Initialize execution state based on whether namespace-aware fields are provided.
	// Priority for @store/ initialization:
	//   1. req.InitialStore (explicit namespace-aware seed)
	//   2. req.InitialVariables (resume support, backward compat)
	//   3. req.Plan.Metadata["variables"] (legacy plan-embedded variables)
	var execState *state.ExecutionState
	if req.InitialStore != nil || req.InitialParams != nil || req.Env != nil {
		// Namespace-aware execution: build initial store from multiple sources
		initialStore := map[string]any{}
		// Start with plan metadata variables (lowest priority)
		if metaVars, ok := req.Plan.Metadata["variables"].(map[string]any); ok {
			for k, v := range metaVars {
				initialStore[k] = v
			}
		}
		// Merge InitialVariables from resume (middle priority)
		if req.InitialVariables != nil {
			for k, v := range req.InitialVariables {
				initialStore[k] = v
			}
		}
		// Merge InitialStore (highest priority)
		if req.InitialStore != nil {
			for k, v := range req.InitialStore {
				initialStore[k] = v
			}
		}
		execState = state.New(initialStore, req.InitialParams, req.Env)
	} else {
		// Legacy execution: use flat variable map
		seedVars := map[string]any{}
		if metaVars, ok := req.Plan.Metadata["variables"].(map[string]any); ok {
			for k, v := range metaVars {
				seedVars[k] = v
			}
		}
		// Merge initial variables from a resumed execution (takes precedence over plan defaults)
		if req.InitialVariables != nil {
			for k, v := range req.InitialVariables {
				seedVars[k] = v
			}
		}
		execState = state.NewFromStore(seedVars)
	}

	if req.Plan.Graph != nil && len(req.Plan.Graph.Steps) > 0 {
	}
	execState.SetNextIndexFromPlan(req.Plan)

	execCtx := executionContext{
		caps:     *caps,
		compiler: compiler,
		maxDepth: maxDepth,
		callStack: func() []uuid.UUID {
			stack := append([]uuid.UUID{}, req.SubflowStack...)
			if req.Plan.WorkflowID != uuid.Nil {
				stack = append(stack, req.Plan.WorkflowID)
			}
			return stack
		}(),
	}

	session, err = e.runPlan(ctx, req, execCtx, eng, spec, session, execState, reuseMode)
	return err
}

func (e *SimpleExecutor) runPlan(ctx context.Context, req Request, execCtx executionContext, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, execState *state.ExecutionState, reuseMode engine.SessionReuseMode) (engine.EngineSession, error) {
	var err error

	// Log resume context if resuming from a previous execution
	if req.StartFromStepIndex > 0 {
		logrus.WithFields(logrus.Fields{
			"execution_id":          req.Plan.ExecutionID,
			"start_from_step_index": req.StartFromStepIndex,
			"resumed_from_id":       req.ResumedFromID,
			"initial_vars_count":    len(req.InitialVariables),
		}).Info("Resuming execution from checkpoint")
	}

	session, err = e.maybeRunEntrypointProbe(ctx, req.Plan, eng, spec, session, execState)
	if err != nil {
		return session, err
	}

	if req.Plan.Graph != nil && len(req.Plan.Graph.Steps) > 0 {
		return e.executeGraph(ctx, req, execCtx, eng, spec, session, execState, reuseMode)
	}

	for idx := range req.Plan.Instructions {
		instruction := req.Plan.Instructions[idx]

		// Skip steps that were already completed in a resumed execution.
		// StartFromStepIndex represents the last completed step, so we skip all steps <= StartFromStepIndex.
		instrStepType := InstructionStepType(instruction)
		if req.StartFromStepIndex > 0 && instruction.Index <= req.StartFromStepIndex {
			logrus.WithFields(logrus.Fields{
				"execution_id": req.Plan.ExecutionID,
				"step_index":   instruction.Index,
				"step_type":    instrStepType,
				"node_id":      instruction.NodeID,
			}).Debug("Skipping already completed step (resume)")
			continue
		}

		if session == nil {
			s, err := eng.StartSession(ctx, spec)
			if err != nil {
				return session, fmt.Errorf("start session: %w", err)
			}
			session = s
		}
		instruction = e.interpolateInstruction(instruction, execState)
		instrStepType = InstructionStepType(instruction) // Refresh after interpolation

		if strings.EqualFold(strings.TrimSpace(instrStepType), "workflowcall") {
			return session, fmt.Errorf("workflowCall nodes are no longer supported; use subflow instead")
		}

		if isSubflowInstruction(instruction) {
			step := planStepToInstructionStep(instruction)
			outcome, updatedSession, err := e.executeSubflow(ctx, req, execCtx, eng, spec, session, step, execState, reuseMode)
			session = updatedSession
			if err != nil {
				return session, err
			}
			if !outcome.Success {
				return session, fmt.Errorf("subflow %s failed: %s", step.NodeID, e.failureMessage(outcome))
			}
			continue
		}

		if isSetVariableInstruction(instruction) {
			instrParams := InstructionParams(instruction)
			outcome, setErr := e.applySetVariable(ctx, req, instruction.Index, instrStepType, instruction.NodeID, instrParams, execState)
			if setErr != nil {
				return session, setErr
			}
			if !outcome.Success {
				return session, fmt.Errorf("set_variable step %d failed", instruction.Index)
			}
			continue
		}

		if ctx.Err() != nil {
			if _, cancelErr := e.recordTerminatedStep(ctx, req, instruction, ctx.Err()); cancelErr != nil {
				return session, cancelErr
			}
			return session, ctx.Err()
		}

		stepCtx, cancel := context.WithCancel(ctx)
		startedAt := time.Now().UTC()
		attempt := 1

		e.emitEvent(stepCtx, req, contracts.EventKindStepStarted, &instruction.Index, &attempt, map[string]any{
			"node_id":   instruction.NodeID,
			"step_type": instrStepType,
		})

		stopHeartbeat := e.startHeartbeat(stepCtx, req, instruction.Index, attempt, startedAt)

		outcome, runErr := e.runWithRetries(stepCtx, req, session, instruction)
		if stopHeartbeat != nil {
			stopHeartbeat()
		}
		cancel()

		normalized := e.normalizeOutcome(req.Plan, instruction, attempt, startedAt, outcome, runErr)

		// Use WithoutCancel to ensure outcomes are persisted even when context is cancelled.
		// This is critical for debugging and audit trails when executions are cancelled mid-flight.
		persistCtx := context.WithoutCancel(ctx)
		recordResult, recordErr := req.Recorder.RecordStepOutcome(persistCtx, req.Plan, normalized)
		if recordErr != nil {
			return session, fmt.Errorf("record step outcome: %w", recordErr)
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
		e.emitEvent(persistCtx, req, eventKind, &instruction.Index, &attempt, payload)

		// Update checkpoint for progress continuity after successful steps
		if normalized.Success {
			totalSteps := len(req.Plan.Instructions)
			if err := req.Recorder.UpdateCheckpoint(persistCtx, req.Plan.ExecutionID, instruction.Index, totalSteps); err != nil {
				// Log but don't fail the execution - checkpoint is best-effort
				logrus.WithError(err).WithFields(logrus.Fields{
					"execution_id": req.Plan.ExecutionID,
					"step_index":   instruction.Index,
				}).Warn("Failed to update execution checkpoint")
			}
		}

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
			return session, fmt.Errorf("reset session: %w", resetErr)
		}
		session = newSession

		if runErr != nil {
			return session, runErr
		}
		if !normalized.Success {
			return session, fmt.Errorf("step %d failed: %s", normalized.StepIndex, e.failureMessage(normalized))
		}
	}

	return session, nil
}

func (e *SimpleExecutor) validateRequest(req Request) error {
	switch {
	case req.EngineFactory == nil:
		return errors.New("engine factory is required")
	case req.Recorder == nil:
		return errors.New("recorder is required")
	case req.EventSink == nil:
		return errors.New("event sink is required")
	case req.Plan.ExecutionID == uuid.Nil:
		return errors.New("plan execution_id is required")
	case req.Plan.WorkflowID == uuid.Nil:
		return errors.New("plan workflow_id is required")
	case req.Plan.SchemaVersion != "" && req.Plan.SchemaVersion != contracts.ExecutionPlanSchemaVersion:
		return fmt.Errorf("plan schema_version mismatch: got %s want %s", req.Plan.SchemaVersion, contracts.ExecutionPlanSchemaVersion)
	case req.Plan.PayloadVersion != "" && req.Plan.PayloadVersion != contracts.PayloadVersion:
		return fmt.Errorf("plan payload_version mismatch: got %s want %s", req.Plan.PayloadVersion, contracts.PayloadVersion)
	}

	// Validate WorkflowResolver is provided if plan contains external subflow references
	if err := e.validateSubflowResolver(req); err != nil {
		return err
	}

	return nil
}

// validateSubflowResolver checks that WorkflowResolver is provided if any subflow
// instruction references an external workflowId (as opposed to inline workflowDefinition).
// This catches configuration errors early with a clear message.
func (e *SimpleExecutor) validateSubflowResolver(req Request) error {
	if req.WorkflowResolver != nil {
		return nil // Resolver present, no validation needed
	}

	// Check compiled instructions
	for _, instr := range req.Plan.Instructions {
		if !isSubflowInstruction(instr) {
			continue
		}

		// Check if this subflow references an external workflow (workflowId/workflowPath)
		// vs inline definition (workflowDefinition).
		instrParams := InstructionParams(instr)
		if instrParams == nil {
			continue
		}

		hasWorkflowID := false
		hasWorkflowPath := false
		hasInlineDefinition := false

		if wfID, ok := instrParams["workflowId"]; ok && wfID != nil && wfID != "" {
			hasWorkflowID = true
		}
		if wfPath, ok := instrParams["workflowPath"]; ok && wfPath != nil && wfPath != "" {
			hasWorkflowPath = true
		}
		if wfDef, ok := instrParams["workflowDefinition"]; ok && wfDef != nil {
			hasInlineDefinition = true
		}

		// If workflowId/workflowPath is specified without inline definition, resolver is required.
		if (hasWorkflowID || hasWorkflowPath) && !hasInlineDefinition {
			return fmt.Errorf(
				"WorkflowResolver is required: subflow node %q references an external workflow - "+
					"provide a WorkflowResolver in the execution request to resolve external workflow references",
				instr.NodeID,
			)
		}
	}

	return nil
}

func (e *SimpleExecutor) normalizeOutcome(plan contracts.ExecutionPlan, instruction contracts.CompiledInstruction, attempt int, startedAt time.Time, outcome contracts.StepOutcome, runErr error) contracts.StepOutcome {
	now := time.Now().UTC()
	outcome.SchemaVersion = valueOrDefault(outcome.SchemaVersion, contracts.StepOutcomeSchemaVersion)
	outcome.PayloadVersion = valueOrDefault(outcome.PayloadVersion, contracts.PayloadVersion)
	outcome.ExecutionID = uuidOrDefault(outcome.ExecutionID, plan.ExecutionID)
	outcome.StepIndex = intOrDefault(outcome.StepIndex, instruction.Index)
	outcome.NodeID = valueOrDefault(outcome.NodeID, instruction.NodeID)
	outcome.StepType = valueOrDefault(outcome.StepType, InstructionStepType(instruction))
	outcome.Attempt = intOrDefault(outcome.Attempt, attempt)

	if outcome.StartedAt.IsZero() {
		outcome.StartedAt = startedAt
	}
	if outcome.CompletedAt == nil {
		t := now
		outcome.CompletedAt = &t
	}
	if outcome.DurationMs == 0 && !outcome.StartedAt.IsZero() && outcome.CompletedAt != nil {
		outcome.DurationMs = int(outcome.CompletedAt.Sub(outcome.StartedAt).Milliseconds())
	}

	if runErr != nil {
		outcome.Success = false
		if outcome.Failure == nil {
			kind := contracts.FailureKindEngine
			retryable := true
			switch {
			case errors.Is(runErr, context.DeadlineExceeded):
				kind = contracts.FailureKindTimeout
				retryable = false
			case errors.Is(runErr, context.Canceled):
				kind = contracts.FailureKindCancelled
				retryable = false
			}
			outcome.Failure = &contracts.StepFailure{
				Kind:      kind,
				Message:   runErr.Error(),
				Retryable: retryable,
				Source: func() contracts.FailureSource {
					if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(runErr, context.Canceled) {
						return contracts.FailureSourceExecutor
					}
					return contracts.FailureSourceEngine
				}(),
				OccurredAt: func() *time.Time {
					t := now
					return &t
				}(),
			}
		}
	}

	if !outcome.Success && outcome.Failure == nil {
		message := "step failed without failure metadata"
		details := map[string]any{}
		if outcome.Assertion != nil {
			if outcome.Assertion.Message != "" {
				message = outcome.Assertion.Message
			} else if outcome.Assertion.Selector != "" || outcome.Assertion.Mode != "" {
				message = fmt.Sprintf("assertion failed (mode=%s selector=%s)", outcome.Assertion.Mode, outcome.Assertion.Selector)
			}
			details["assertion"] = outcome.Assertion
		} else if instruction.Action != nil && instruction.Action.GetAssert() != nil {
			assertParams := instruction.Action.GetAssert()
			mode := assertParams.GetMode().String()
			selector := assertParams.GetSelector()
			if selector != "" || mode != "" {
				message = fmt.Sprintf("assertion failed (mode=%s selector=%s)", mode, selector)
			}
			details["assertion"] = map[string]any{
				"mode":     mode,
				"selector": selector,
			}
		}
		if outcome.Condition != nil {
			details["condition"] = outcome.Condition
		}
		outcome.Failure = &contracts.StepFailure{
			Kind:      contracts.FailureKindEngine,
			Message:   message,
			Retryable: false,
			Source:    contracts.FailureSourceEngine,
			Details:   details,
			OccurredAt: func() *time.Time {
				t := now
				return &t
			}(),
		}
	}

	if outcome.Failure != nil && outcome.Failure.Source == "" {
		outcome.Failure.Source = contracts.FailureSourceEngine
	}

	return outcome
}

func (e *SimpleExecutor) failureMessage(outcome contracts.StepOutcome) string {
	if outcome.Failure != nil && outcome.Failure.Message != "" {
		return outcome.Failure.Message
	}
	return "unknown failure"
}

const (
	entryProbeNodeID      = "__entry_probe__"
	defaultEntryTimeoutMs = 3000
	minEntryTimeoutMs     = 250
)

func (e *SimpleExecutor) maybeRunEntrypointProbe(ctx context.Context, plan contracts.ExecutionPlan, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, execState *state.ExecutionState) (engine.EngineSession, error) {
	if execState == nil || execState.HasCheckedEntry() {
		return session, nil
	}
	selector, timeoutMs := entrySelectorFromPlan(plan)
	execState.MarkEntryChecked()
	if strings.TrimSpace(selector) == "" {
		return session, nil
	}
	if strings.Contains(selector, "${") {
		// Skip probes on unresolved template placeholders to avoid false negatives
		return session, nil
	}
	if timeoutMs <= 0 {
		timeoutMs = defaultEntryTimeoutMs
	}
	if timeoutMs < minEntryTimeoutMs {
		timeoutMs = minEntryTimeoutMs
	}

	if session == nil {
		var err error
		session, err = eng.StartSession(ctx, spec)
		if err != nil {
			return session, fmt.Errorf("start session for entry probe: %w", err)
		}
	}

	probeCtx := ctx
	if timeoutMs > 0 {
		var cancel context.CancelFunc
		probeCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}

	instr := contracts.CompiledInstruction{
		Index:  -1,
		NodeID: entryProbeNodeID,
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_WAIT,
			Params: &basactions.ActionDefinition_Wait{
				Wait: &basactions.WaitParams{
					WaitFor: &basactions.WaitParams_Selector{Selector: selector},
				},
			},
		},
	}

	outcome, err := session.Run(probeCtx, instr)
	if err != nil {
		return session, fmt.Errorf("entry selector %q not reachable within %dms: %w", selector, timeoutMs, err)
	}
	if !outcome.Success {
		return session, fmt.Errorf("entry selector %q not reachable within %dms: %s", selector, timeoutMs, e.failureMessage(outcome))
	}
	return session, nil
}

func entrySelectorFromPlan(plan contracts.ExecutionPlan) (string, int) {
	if selector, ok := readString(plan.Metadata, "entrySelector"); ok {
		logrus.WithFields(logrus.Fields{
			"execution_id": plan.ExecutionID,
			"selector":     selector,
			"source":       "metadata",
		}).Debug("Using explicit entrySelector from metadata")
		return selector, readInt(plan.Metadata, "entrySelectorTimeoutMs", "entryTimeoutMs")
	}

	// Skip entry probe whenever the first actionable instruction is navigation.
	// Many workflows (especially playbooks) begin with bookkeeping steps (seed state,
	// store variables, etc.) and then navigate. Probing selectors before navigation
	// creates false timeouts because the page context doesn't exist yet.
	if len(plan.Instructions) > 0 {
		sorted := append([]contracts.CompiledInstruction{}, plan.Instructions...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Index < sorted[j].Index
		})
		for _, instr := range sorted {
			instrType := InstructionStepType(instr)
			if instrType == "navigate" {
				logrus.WithField("execution_id", plan.ExecutionID).Debug("Skipping entry probe - workflow navigates before first selector-bearing instruction")
				return "", 0
			}
			if instrType == "subflow" {
				logrus.WithField("execution_id", plan.ExecutionID).Debug("Skipping entry probe - workflow starts with subflow")
				return "", 0
			}
			instrParams := InstructionParams(instr)
			if selector := firstSelectorFromParams(instrParams); selector != "" {
				timeout := readInt(plan.Metadata, "entrySelectorTimeoutMs", "entryTimeoutMs")
				logrus.WithFields(logrus.Fields{
					"execution_id": plan.ExecutionID,
					"selector":     selector,
					"instruction":  instrType,
					"source":       "instructions",
				}).Debug("Extracted entry selector from first selector-bearing instruction")
				return selector, timeout
			}
		}
	}

	selector := firstSelectorFromInstructions(plan.Instructions)
	logrus.WithFields(logrus.Fields{
		"execution_id":      plan.ExecutionID,
		"selector":          selector,
		"instruction_count": len(plan.Instructions),
		"source":            "instructions",
	}).Debug("Extracted entry selector from instructions")
	return selector, readInt(plan.Metadata, "entrySelectorTimeoutMs", "entryTimeoutMs")
}

func firstSelectorFromInstructions(instructions []contracts.CompiledInstruction) string {
	if len(instructions) == 0 {
		return ""
	}
	sorted := append([]contracts.CompiledInstruction{}, instructions...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Index < sorted[j].Index
	})
	for _, instr := range sorted {
		// Skip navigation instructions - they don't have meaningful selectors for entry probe
		// Navigation establishes the page context, so we should wait for the first
		// interactive element instead
		if InstructionStepType(instr) == "navigate" {
			continue
		}
		selector := firstSelectorFromParams(InstructionParams(instr))
		if selector != "" {
			return selector
		}
	}
	return ""
}

var selectorPriority = []string{
	"selector",
	"waitForSelector",
	"preconditionSelector",
	"successSelector",
	"targetSelector",
	"dragTargetSelector",
	"dragSourceSelector",
	"sourceSelector",
	"focusSelector",
	"gestureSelector",
	"scrollTargetSelector",
	"frameSelector",
	"conditionSelector",
}

func firstSelectorFromParams(params map[string]any) string {
	for _, key := range selectorPriority {
		if val, ok := params[key]; ok {
			if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !strings.Contains(strings.ToLower(k), "selector") {
			continue
		}
		if val, ok := params[k]; ok {
			if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func readString(meta map[string]any, key string) (string, bool) {
	if meta == nil {
		return "", false
	}
	raw, ok := meta[key]
	if !ok {
		return "", false
	}
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

func readInt(meta map[string]any, keys ...string) int {
	if meta == nil {
		return 0
	}
	for _, key := range keys {
		if raw, ok := meta[key]; ok {
			switch v := raw.(type) {
			case float64:
				if v > 0 {
					return int(v)
				}
			case int:
				if v > 0 {
					return v
				}
			case int64:
				if v > 0 {
					return int(v)
				}
			}
		}
	}
	return 0
}

func (e *SimpleExecutor) emitEvent(ctx context.Context, req Request, kind contracts.EventKind, stepIndex *int, attempt *int, payload any) {
	if req.EventSink == nil {
		return
	}
	ev := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           kind,
		ExecutionID:    req.Plan.ExecutionID,
		WorkflowID:     req.Plan.WorkflowID,
		StepIndex:      stepIndex,
		Attempt:        attempt,
		Sequence:       e.sequencer.Next(req.Plan.ExecutionID),
		Timestamp:      time.Now().UTC(),
		Payload:        payload,
	}
	_ = req.EventSink.Publish(ctx, ev) // Sink decides drop policy; executor is fire-and-forget.
}

func (e *SimpleExecutor) emitTelemetry(ctx context.Context, req Request, telemetry contracts.StepTelemetry) {
	if req.EventSink == nil {
		return
	}
	if telemetry.Sequence == 0 {
		telemetry.Sequence = e.nextTelemetrySequence(req.Plan.ExecutionID, telemetry.StepIndex, telemetry.Attempt)
	}
	ev := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepTelemetry,
		ExecutionID:    req.Plan.ExecutionID,
		WorkflowID:     req.Plan.WorkflowID,
		StepIndex:      &telemetry.StepIndex,
		Attempt:        &telemetry.Attempt,
		Sequence:       e.sequencer.Next(req.Plan.ExecutionID),
		Timestamp:      telemetry.Timestamp,
		Payload:        telemetry,
	}
	_ = req.EventSink.Publish(ctx, ev)
}

func (e *SimpleExecutor) startHeartbeat(ctx context.Context, req Request, stepIndex int, attempt int, started time.Time) func() {
	if req.HeartbeatInterval <= 0 {
		return nil
	}
	ticker := time.NewTicker(req.HeartbeatInterval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-done:
				ticker.Stop()
				return
			case t := <-ticker.C:
				elapsed := t.Sub(started).Milliseconds()
				telemetry := contracts.StepTelemetry{
					SchemaVersion:  contracts.TelemetrySchemaVersion,
					PayloadVersion: contracts.PayloadVersion,
					ExecutionID:    req.Plan.ExecutionID,
					StepIndex:      stepIndex,
					Attempt:        attempt,
					Kind:           contracts.TelemetryKindHeartbeat,
					Sequence:       0, // Sequence within attempt can be layered later; envelope carries ordering.
					Timestamp:      t.UTC(),
					ElapsedMs:      elapsed,
					Heartbeat: &contracts.HeartbeatTelemetry{
						Progress: 0,
						Message:  "heartbeat",
					},
				}
				e.emitTelemetry(ctx, req, telemetry)
			}
		}
	}()

	return func() { close(done) }
}

func (e *SimpleExecutor) telemetryKey(executionID uuid.UUID, stepIndex, attempt int) string {
	return fmt.Sprintf("%s:%d:%d", executionID.String(), stepIndex, attempt)
}

func (e *SimpleExecutor) nextTelemetrySequence(executionID uuid.UUID, stepIndex, attempt int) uint64 {
	e.telemetryMu.Lock()
	defer e.telemetryMu.Unlock()

	if e.telemetrySeq == nil {
		e.telemetrySeq = make(map[string]uint64)
	}
	key := e.telemetryKey(executionID, stepIndex, attempt)
	e.telemetrySeq[key]++
	return e.telemetrySeq[key]
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func intOrDefault(value, fallback int) int {
	if value != 0 {
		return value
	}
	return fallback
}

// resolveReuseMode selects the session reuse policy from the request, plan metadata, or environment.
func resolveReuseMode(req Request) engine.SessionReuseMode {
	if req.ReuseMode != "" {
		return req.ReuseMode
	}
	if v, ok := req.Plan.Metadata["sessionReuseMode"].(string); ok && strings.TrimSpace(v) != "" {
		return normalizeReuseMode(v)
	}
	if env := strings.TrimSpace(os.Getenv("BAS_SESSION_STRATEGY")); env != "" {
		return normalizeReuseMode(env)
	}
	return engine.ReuseModeReuse
}

func normalizeReuseMode(raw string) engine.SessionReuseMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "fresh":
		return engine.ReuseModeFresh
	case "clean":
		return engine.ReuseModeClean
	default:
		return engine.ReuseModeReuse
	}
}

// maybeResetSession clears or recreates browser state based on reuse policy.
func (e *SimpleExecutor) maybeResetSession(ctx context.Context, eng engine.AutomationEngine, spec engine.SessionSpec, session engine.EngineSession, reuseMode engine.SessionReuseMode) (engine.EngineSession, error) {
	if session == nil {
		return session, nil
	}

	switch reuseMode {
	case engine.ReuseModeClean:
		return session, session.Reset(ctx)
	case engine.ReuseModeFresh:
		_ = session.Close(ctx)
		return nil, nil
	default:
		return session, nil
	}
}

func extractViewport(metadata map[string]any) (int, int) {
	raw, ok := metadata["executionViewport"].(map[string]any)
	if !ok || raw == nil {
		logrus.WithFields(logrus.Fields{
			"metadata":          metadata,
			"executionViewport": metadata["executionViewport"],
			"type_ok":           ok,
		}).Warn("extractViewport: executionViewport not found or wrong type")
		return 0, 0
	}
	width := state.IntValue(raw, "width")
	height := state.IntValue(raw, "height")
	logrus.WithFields(logrus.Fields{
		"raw":    raw,
		"width":  width,
		"height": height,
	}).Info("extractViewport: extracted viewport dimensions")
	return width, height
}

// extractFrameStreamingConfig extracts frame streaming configuration from plan metadata.
// Returns nil if frame streaming is not enabled or not configured.
// Frame streaming enables live preview during workflow execution.
func extractFrameStreamingConfig(metadata map[string]any, executionID uuid.UUID) *engine.FrameStreamingConfig {
	raw, ok := metadata["frameStreaming"].(map[string]any)
	if !ok || raw == nil {
		return nil
	}

	// Check if explicitly enabled
	enabled, _ := raw["enabled"].(bool)
	if !enabled {
		return nil
	}

	// Build callback URL for this execution
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	// Extract quality, fps, scale with defaults
	quality := state.IntValue(raw, "quality")
	if quality <= 0 {
		quality = 55 // Default JPEG quality
	}
	fps := state.IntValue(raw, "fps")
	if fps <= 0 {
		fps = 6 // Default FPS
	}
	scale := "css"
	if s, ok := raw["scale"].(string); ok && s != "" {
		scale = s
	}

	config := &engine.FrameStreamingConfig{
		CallbackURL: fmt.Sprintf("http://%s:%s/api/v1/executions/%s/frames",
			apiHost, apiPort, executionID.String()),
		Quality: quality,
		FPS:     fps,
		Scale:   scale,
	}

	logrus.WithFields(logrus.Fields{
		"execution_id": executionID,
		"callback_url": config.CallbackURL,
		"quality":      quality,
		"fps":          fps,
		"scale":        scale,
	}).Info("extractFrameStreamingConfig: enabled live frame streaming")

	return config
}

func (e *SimpleExecutor) runWithRetries(ctx context.Context, req Request, session engine.EngineSession, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	cfg := retryConfigFromInstruction(instruction)
	var lastOutcome contracts.StepOutcome
	var lastErr error
	delay := cfg.Delay
	timeout := instructionTimeout(req.Plan, instruction)

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		attemptStart := time.Now().UTC()

		attemptCtx := ctx
		var cancel context.CancelFunc
		if timeout > 0 {
			// Add 2 seconds of buffer time to account for HTTP overhead, network latency,
			// and playwright-driver internal polling logic. The driver needs the full timeout
			// duration to poll for element presence/absence, but the HTTP context deadline
			// would otherwise cut it off before polling completes.
			httpBufferTime := 2 * time.Second
			attemptCtx, cancel = context.WithTimeout(ctx, timeout+httpBufferTime)
		}

		outcome, err := session.Run(attemptCtx, instruction)
		if cancel != nil {
			cancel()
		}

		outcome.Attempt = attempt
		outcome = e.normalizeOutcome(req.Plan, instruction, attempt, attemptStart, outcome, err)

		if req.Recorder != nil && outcome.Failure != nil {
			_ = req.Recorder.RecordTelemetry(ctx, req.Plan, contracts.StepTelemetry{
				SchemaVersion:  contracts.TelemetrySchemaVersion,
				PayloadVersion: contracts.PayloadVersion,
				ExecutionID:    req.Plan.ExecutionID,
				StepIndex:      instruction.Index,
				Attempt:        attempt,
				Kind:           contracts.TelemetryKindRetry,
				Sequence:       e.nextTelemetrySequence(req.Plan.ExecutionID, instruction.Index, attempt),
				Timestamp:      time.Now().UTC(),
				Retry: &contracts.RetryTelemetry{
					Attempt:     attempt,
					MaxAttempts: cfg.MaxAttempts,
					Reason:      outcome.Failure,
				},
			})
		}

		lastOutcome, lastErr = outcome, err
		if err == nil && outcome.Success {
			return outcome, nil
		}
		if attempt >= cfg.MaxAttempts || (outcome.Failure != nil && !outcome.Failure.Retryable) {
			return outcome, err
		}

		select {
		case <-ctx.Done():
			return outcome, ctx.Err()
		case <-time.After(delay):
		}
		delay = time.Duration(float64(delay) * cfg.BackoffFactor)
	}

	return lastOutcome, lastErr
}

type retryConfig struct {
	MaxAttempts   int
	Delay         time.Duration
	BackoffFactor float64
}

func retryConfigFromInstruction(instruction contracts.CompiledInstruction) retryConfig {
	cfg := retryConfig{
		MaxAttempts:   1,
		Delay:         750 * time.Millisecond,
		BackoffFactor: 1.5,
	}
	instrParams := InstructionParams(instruction)
	if instrParams == nil {
		return cfg
	}
	if v, ok := instrParams["retryAttempts"].(float64); ok && v > 0 {
		cfg.MaxAttempts = int(v)
	}
	if v, ok := instrParams["retryDelayMs"].(float64); ok && v > 0 {
		cfg.Delay = time.Duration(v) * time.Millisecond
	}
	if v, ok := instrParams["retryBackoffFactor"].(float64); ok && v > 0 {
		cfg.BackoffFactor = v
	}
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.BackoffFactor < 1 {
		cfg.BackoffFactor = 1
	}
	return cfg
}

// instructionTimeout returns a step-level timeout derived from instruction params or plan metadata.
// timeoutMs is expected to be milliseconds; zero disables executor-side deadline enforcement.
func instructionTimeout(plan contracts.ExecutionPlan, instruction contracts.CompiledInstruction) time.Duration {
	ms := 0
	instrParams := InstructionParams(instruction)
	if v, ok := instrParams["timeoutMs"]; ok {
		switch t := v.(type) {
		case int:
			ms = t
		case int64:
			ms = int(t)
		case float64:
			ms = int(t)
		}
	}
	if ms <= 0 {
		if v, ok := plan.Metadata["defaultTimeoutMs"]; ok {
			switch t := v.(type) {
			case int:
				ms = t
			case int64:
				ms = int(t)
			case float64:
				ms = int(t)
			}
		}
	}
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// Timeout configuration defaults for dynamic timeout calculation.
// Values are overridden by config.Execution when present.
const (
	// baseExecutionTimeout is the minimum overhead for session setup, network round-trips, etc.
	baseExecutionTimeout = 30 * time.Second
	// perStepTimeout is the additional time allocated per instruction in a simple workflow.
	perStepTimeout = 10 * time.Second
	// perStepTimeoutWithSubflows increases per-step allocation for workflows with nested execution.
	perStepTimeoutWithSubflows = 15 * time.Second
	// minExecutionTimeout prevents unreasonably short timeouts for small workflows.
	minExecutionTimeout = 90 * time.Second
	// maxExecutionTimeout prevents timeouts from exceeding the HTTP client timeout (5 minutes).
	// The 30-second buffer accounts for network overhead and response processing.
	maxExecutionTimeout = 270 * time.Second // 4.5 minutes
)

// executionTimeout reads the execution-level timeout (ms) from plan metadata.
// If not explicitly set, it calculates a dynamic timeout based on step count.
func executionTimeout(plan contracts.ExecutionPlan) time.Duration {
	// Check for explicit timeout in metadata (takes precedence)
	if plan.Metadata != nil {
		if v, ok := plan.Metadata["executionTimeoutMs"]; ok {
			switch t := v.(type) {
			case int:
				if t > 0 {
					return time.Duration(t) * time.Millisecond
				}
			case int64:
				if t > 0 {
					return time.Duration(t) * time.Millisecond
				}
			case float64:
				if t > 0 {
					return time.Duration(t) * time.Millisecond
				}
			}
		}
	}

	// Calculate dynamic timeout based on step count and complexity
	return computeDynamicTimeout(plan)
}

// computeDynamicTimeout calculates execution timeout based on the number and type of instructions.
// Formula: base + (stepCount * perStepTimeout), clamped to [min, max]
// This ensures complex workflows get proportionally more time while preventing unbounded timeouts.
func computeDynamicTimeout(plan contracts.ExecutionPlan) time.Duration {
	execCfg := config.Load().Execution

	base := execCfg.BaseTimeout
	if base <= 0 {
		base = baseExecutionTimeout
	}
	perStep := execCfg.PerStepTimeout
	if perStep <= 0 {
		perStep = perStepTimeout
	}
	perStepSubflow := execCfg.PerStepSubflowTimeout
	if perStepSubflow <= 0 {
		perStepSubflow = perStepTimeoutWithSubflows
	}
	minTimeout := execCfg.MinTimeout
	if minTimeout <= 0 {
		minTimeout = minExecutionTimeout
	}
	maxTimeout := execCfg.MaxTimeout
	if maxTimeout <= 0 {
		maxTimeout = maxExecutionTimeout
	}
	// Hardening: misconfigured clamp windows (max < min) can silently create
	// extremely short execution timeouts and cause spurious DeadlineExceeded.
	// Prefer the safer interpretation: treat max as at least min.
	if maxTimeout < minTimeout {
		maxTimeout = minTimeout
	}

	stepCount := len(plan.Instructions)
	if stepCount == 0 {
		return minTimeout
	}

	// Determine if workflow has subflows (which need more time per step)
	hasSubflows := false
	for _, instr := range plan.Instructions {
		if isSubflowInstruction(instr) {
			hasSubflows = true
			break
		}
	}

	// Calculate timeout based on step count
	if hasSubflows {
		perStep = perStepSubflow
	}

	// Dynamic timeout: base + (stepCount * perStep)
	timeout := base + time.Duration(stepCount)*perStep

	// Clamp to valid range
	if timeout < minTimeout {
		return minTimeout
	}
	if timeout > maxTimeout {
		return maxTimeout
	}
	return timeout
}

// recordTerminatedStep best-effort persists an outcome when execution is cancelled or times out,
// and emits a failed event with appropriate failure taxonomy.
func (e *SimpleExecutor) recordTerminatedStep(ctx context.Context, req Request, instruction contracts.CompiledInstruction, cause error) (contracts.StepOutcome, error) {
	startedAt := time.Now().UTC()
	if cause == nil {
		cause = context.Canceled
	}
	outcome := e.normalizeOutcome(req.Plan, instruction, 1, startedAt, contracts.StepOutcome{}, cause)

	persistCtx := context.WithoutCancel(ctx)
	recordResult, recordErr := req.Recorder.RecordStepOutcome(persistCtx, req.Plan, outcome)
	if recordErr != nil {
		return outcome, fmt.Errorf("record terminated step outcome: %w", recordErr)
	}

	payload := map[string]any{
		"outcome":   outcome,
		"artifacts": recordResult.ArtifactIDs,
	}
	if recordResult.TimelineArtifactID != nil {
		payload["timeline_artifact_id"] = *recordResult.TimelineArtifactID
	}
	e.emitEvent(persistCtx, req, contracts.EventKindStepFailed, &outcome.StepIndex, &outcome.Attempt, payload)

	return outcome, cause
}
