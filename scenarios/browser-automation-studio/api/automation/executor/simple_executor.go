package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
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
func (e *SimpleExecutor) Execute(ctx context.Context, req Request) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := e.validateRequest(req); err != nil {
		return err
	}

	engineName := req.EngineName
	if engineName == "" {
		engineName = "browserless"
	}

	eng, err := req.EngineFactory.Resolve(ctx, engineName)
	if err != nil {
		return fmt.Errorf("resolve engine %q: %w", engineName, err)
	}

	// Capabilities are fetched up front to allow callers to validate or log.
	caps, err := eng.Capabilities(ctx)
	if err != nil {
		return fmt.Errorf("engine capabilities: %w", err)
	}
	if gap := caps.CheckCompatibility(deriveRequirements(req.Plan)); !gap.Satisfied() {
		return &CapabilityError{
			Engine:    eng.Name(),
			Missing:   gap.Missing,
			Warnings:  gap.Warnings,
			Execution: req.Plan.ExecutionID,
			Workflow:  req.Plan.WorkflowID,
		}
	}

	session, err := eng.StartSession(ctx, engine.SessionSpec{
		ExecutionID: req.Plan.ExecutionID,
		WorkflowID:  req.Plan.WorkflowID,
		ReuseMode:   engine.ReuseModeReuse,
		Labels:      map[string]string{},
		Capabilities: contracts.CapabilityRequirement{
			NeedsParallelTabs: false,
		},
	})
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	defer session.Close(context.Background()) // Best-effort cleanup.

	if req.Plan.Graph != nil && len(req.Plan.Graph.Steps) > 0 {
		return e.executeGraph(ctx, req, session)
	}

	for idx := range req.Plan.Instructions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		instruction := req.Plan.Instructions[idx]

		stepCtx, cancel := context.WithCancel(ctx)
		startedAt := time.Now().UTC()
		attempt := 1

		e.emitEvent(stepCtx, req, contracts.EventKindStepStarted, &instruction.Index, &attempt, map[string]any{
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
			return fmt.Errorf("record step outcome: %w", recordErr)
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

		if runErr != nil {
			return runErr
		}
		if !normalized.Success {
			return fmt.Errorf("step %d failed: %s", normalized.StepIndex, e.failureMessage(normalized))
		}
	}

	return nil
}

func (e *SimpleExecutor) executeGraph(ctx context.Context, req Request, session engine.EngineSession) error {
	stepMap := indexGraph(req.Plan.Graph)
	current := firstStep(req.Plan.Graph)
	visited := 0
	maxVisited := len(stepMap) * 10
	var lastFailure error

	for current != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		visited++
		if maxVisited > 0 && visited > maxVisited {
			return fmt.Errorf("graph execution exceeded step budget (%d)", maxVisited)
		}

		outcome, err := e.executePlanStep(ctx, req, session, *current)
		if err != nil {
			return err
		}
		if !outcome.Success {
			lastFailure = fmt.Errorf("step %d failed: %s", outcome.StepIndex, e.failureMessage(outcome))
		}

		nextID := e.nextNodeID(*current, outcome)
		if nextID == "" {
			if lastFailure != nil {
				return lastFailure
			}
			return nil
		}
		next, ok := stepMap[nextID]
		if !ok {
			return fmt.Errorf("graph references missing node %s", nextID)
		}
		current = next
	}

	return lastFailure
}

func (e *SimpleExecutor) executePlanStep(ctx context.Context, req Request, session engine.EngineSession, step contracts.PlanStep) (contracts.StepOutcome, error) {
	if strings.EqualFold(step.Type, "loop") && step.Loop != nil {
		return e.executeLoop(ctx, req, session, step)
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
		return normalized, fmt.Errorf("record step outcome: %w", recordErr)
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

	if runErr != nil {
		return normalized, runErr
	}

	return normalized, nil
}

func (e *SimpleExecutor) executeLoop(ctx context.Context, req Request, session engine.EngineSession, step contracts.PlanStep) (contracts.StepOutcome, error) {
	loopType := strings.ToLower(strings.TrimSpace(stringValue(step.Params, "loopType")))
	if loopType == "" {
		return contracts.StepOutcome{}, fmt.Errorf("loop node %s missing loopType", step.NodeID)
	}

	maxIterations := intValue(step.Params, "loopMaxIterations")
	if maxIterations <= 0 {
		maxIterations = 100
	}

	var iterations int
	switch loopType {
	case "repeat":
		iterations = intValue(step.Params, "loopCount")
		if iterations <= 0 {
			return contracts.StepOutcome{}, fmt.Errorf("loop node %s repeat requires loopCount > 0", step.NodeID)
		}
	default:
		return contracts.StepOutcome{}, fmt.Errorf("loop node %s uses unsupported loopType %s", step.NodeID, loopType)
	}

	iterations = minInt(iterations, maxIterations)
	if iterations == 0 {
		return contracts.StepOutcome{}, fmt.Errorf("loop node %s has zero iterations after clamping", step.NodeID)
	}

	var lastOutcome contracts.StepOutcome
	for i := 0; i < iterations; i++ {
		control, err := e.executeGraphIteration(ctx, req, session, step.Loop)
		if err != nil {
			return contracts.StepOutcome{}, err
		}
		lastOutcome = control.LastOutcome
		if control.Break {
			break
		}
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
		return loopOutcome, fmt.Errorf("record loop outcome: %w", recordErr)
	}
	payload := map[string]any{
		"outcome":   loopOutcome,
		"artifacts": recordResult.ArtifactIDs,
	}
	if recordResult.TimelineArtifactID != nil {
		payload["timeline_artifact_id"] = *recordResult.TimelineArtifactID
	}
	e.emitEvent(ctx, req, contracts.EventKindStepCompleted, &step.Index, intPtr(1), payload)

	return loopOutcome, nil
}

type loopControl struct {
	Break       bool
	LastOutcome contracts.StepOutcome
}

func (e *SimpleExecutor) executeGraphIteration(ctx context.Context, req Request, session engine.EngineSession, graph *contracts.PlanGraph) (loopControl, error) {
	if graph == nil || len(graph.Steps) == 0 {
		return loopControl{}, nil
	}

	stepMap := indexGraph(graph)
	current := firstStep(graph)
	visited := 0
	maxVisited := len(stepMap) * 10
	var last contracts.StepOutcome

	for current != nil {
		visited++
		if maxVisited > 0 && visited > maxVisited {
			return loopControl{}, fmt.Errorf("loop body exceeded step budget (%d)", maxVisited)
		}

		outcome, err := e.executePlanStep(ctx, req, session, *current)
		if err != nil {
			return loopControl{}, err
		}
		last = outcome

		nextID := e.nextNodeID(*current, outcome)
		if nextID == "" {
			break
		}
		if nextID == "__loop_break__" {
			return loopControl{Break: true, LastOutcome: outcome}, nil
		}
		if nextID == "__loop_continue__" {
			break
		}
		next, ok := stepMap[nextID]
		if !ok {
			return loopControl{}, fmt.Errorf("loop body references missing node %s", nextID)
		}
		current = next
	}

	return loopControl{LastOutcome: last}, nil
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
	outcome.StepType = valueOrDefault(outcome.StepType, instruction.Type)
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
				Source:    contracts.FailureSourceEngine,
				OccurredAt: func() *time.Time {
					t := now
					return &t
				}(),
			}
		}
	}

	if !outcome.Success && outcome.Failure == nil {
		outcome.Failure = &contracts.StepFailure{
			Kind:      contracts.FailureKindEngine,
			Message:   "step failed without failure metadata",
			Retryable: false,
			Source:    contracts.FailureSourceEngine,
			OccurredAt: func() *time.Time {
				t := now
				return &t
			}(),
		}
	}

	return outcome
}

func (e *SimpleExecutor) failureMessage(outcome contracts.StepOutcome) string {
	if outcome.Failure != nil && outcome.Failure.Message != "" {
		return outcome.Failure.Message
	}
	return "unknown failure"
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

func uuidOrDefault(value, fallback uuid.UUID) uuid.UUID {
	if value != uuid.Nil {
		return value
	}
	return fallback
}

func indexGraph(graph *contracts.PlanGraph) map[string]*contracts.PlanStep {
	index := make(map[string]*contracts.PlanStep)
	if graph == nil {
		return index
	}
	for i := range graph.Steps {
		step := &graph.Steps[i]
		index[step.NodeID] = step
	}
	return index
}

func firstStep(graph *contracts.PlanGraph) *contracts.PlanStep {
	if graph == nil || len(graph.Steps) == 0 {
		return nil
	}
	first := &graph.Steps[0]
	for i := range graph.Steps {
		if graph.Steps[i].Index < first.Index {
			first = &graph.Steps[i]
		}
	}
	return first
}

func planStepToInstruction(step contracts.PlanStep) contracts.CompiledInstruction {
	return contracts.CompiledInstruction{
		Index:       step.Index,
		NodeID:      step.NodeID,
		Type:        step.Type,
		Params:      step.Params,
		PreloadHTML: step.Preload,
		Context:     step.Context,
		Metadata:    step.Metadata,
	}
}

func (e *SimpleExecutor) nextNodeID(step contracts.PlanStep, outcome contracts.StepOutcome) string {
	if len(step.Outgoing) == 0 {
		return ""
	}

	normalized := func(value string) string {
		return strings.ToLower(strings.TrimSpace(value))
	}

	if strings.EqualFold(step.Type, "conditional") && outcome.Condition != nil {
		targetValue := "false"
		if outcome.Condition.Outcome {
			targetValue = "true"
		}
		for _, edge := range step.Outgoing {
			cond := normalized(edge.Condition)
			switch cond {
			case targetValue:
				return edge.Target
			case "yes", "success", "ok", "pass":
				if targetValue == "true" {
					return edge.Target
				}
			case "no", "fail", "failure":
				if targetValue == "false" {
					return edge.Target
				}
			}
		}
	}

	if outcome.Failure != nil {
		for _, edge := range step.Outgoing {
			cond := normalized(edge.Condition)
			if cond == "failure" || cond == "error" || cond == "fail" {
				return edge.Target
			}
		}
	}

	if outcome.Success {
		for _, edge := range step.Outgoing {
			cond := normalized(edge.Condition)
			if cond == "success" || cond == "ok" || cond == "pass" || cond == "" {
				return edge.Target
			}
		}
	}

	// Default: first edge wins.
	return step.Outgoing[0].Target
}

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intValue(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case int:
			return t
		case int64:
			return int(t)
		case float64:
			return int(t)
		}
	}
	return 0
}

func intPtr(v int) *int {
	return &v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func deriveRequirements(plan contracts.ExecutionPlan) contracts.CapabilityRequirement {
	req := contracts.CapabilityRequirement{}
	if viewportRaw, ok := plan.Metadata["executionViewport"]; ok {
		if viewport, ok := viewportRaw.(map[string]any); ok {
			if w, ok := viewport["width"].(float64); ok && w > 0 {
				req.MinViewportWidth = int(w)
			}
			if h, ok := viewport["height"].(float64); ok && h > 0 {
				req.MinViewportHeight = int(h)
			}
		}
	}
	req = mergeMetadataCapabilities(req, plan.Metadata)

	for _, instr := range plan.Instructions {
		params := instr.Params
		if params == nil {
			continue
		}
		// Multi-tab support required when any tab switch directive is present.
		if requiresParallelTabs(params) {
			req.NeedsParallelTabs = true
		}
		// Iframe support required when frame navigation occurs.
		if requiresIframes(params) {
			req.NeedsIframes = true
		}
		// Network interception/mocking implies HAR/tracing capability.
		if requiresNetworkMock(params) {
			req.NeedsHAR = true
			req.NeedsTracing = true
		}
		// File upload support required when uploads are configured.
		if requiresFileUploads(params) {
			req.NeedsFileUploads = true
		}
		// Instruction-level viewport hints should not shrink global minima.
		if w, ok := numericParam(params, "viewportWidth"); ok && w > req.MinViewportWidth {
			req.MinViewportWidth = w
		}
		if h, ok := numericParam(params, "viewportHeight"); ok && h > req.MinViewportHeight {
			req.MinViewportHeight = h
		}
	}
	return req
}

func mergeMetadataCapabilities(req contracts.CapabilityRequirement, metadata map[string]any) contracts.CapabilityRequirement {
	flag := func(key string) bool {
		raw, ok := metadata[key]
		if !ok {
			return false
		}
		if b, ok := raw.(bool); ok {
			return b
		}
		return false
	}
	if flag("requiresDownloads") {
		req.NeedsDownloads = true
	}
	if flag("requiresFileUploads") {
		req.NeedsFileUploads = true
	}
	if flag("requiresHar") || flag("requiresHAR") {
		req.NeedsHAR = true
	}
	if flag("requiresVideo") {
		req.NeedsVideo = true
	}
	if flag("requiresTracing") {
		req.NeedsTracing = true
	}
	if flag("requiresIframes") {
		req.NeedsIframes = true
	}
	if flag("requiresParallelTabs") {
		req.NeedsParallelTabs = true
	}
	return req
}

func requiresParallelTabs(params map[string]any) bool {
	for _, key := range []string{
		"tabSwitchBy", "tabIndex", "tabTitleMatch", "tabUrlMatch", "tabWaitForNew", "tabCloseOld",
	} {
		if _, ok := params[key]; ok {
			return true
		}
	}
	return false
}

func requiresIframes(params map[string]any) bool {
	for _, key := range []string{
		"frameSwitchBy", "frameIndex", "frameName", "frameSelector", "frameUrlMatch",
	} {
		if _, ok := params[key]; ok {
			return true
		}
	}
	return false
}

func requiresFileUploads(params map[string]any) bool {
	if v, ok := params["filePath"]; ok && v != nil && fmt.Sprint(v) != "" {
		return true
	}
	if v, ok := params["filePaths"]; ok {
		if arr, ok := v.([]any); ok && len(arr) > 0 {
			return true
		}
	}
	return false
}

func requiresNetworkMock(params map[string]any) bool {
	for _, key := range []string{
		"networkMockType", "networkUrlPattern", "networkMethod", "networkStatusCode", "networkHeaders", "networkBody", "networkDelayMs", "networkAbortReason",
	} {
		if _, ok := params[key]; ok {
			return true
		}
	}
	return false
}

func numericParam(params map[string]any, key string) (int, bool) {
	if raw, ok := params[key]; ok {
		switch v := raw.(type) {
		case float64:
			return int(v), true
		case int:
			return v, true
		}
	}
	return 0, false
}

func (e *SimpleExecutor) runWithRetries(ctx context.Context, req Request, session engine.EngineSession, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	cfg := retryConfigFromInstruction(instruction)
	var lastOutcome contracts.StepOutcome
	var lastErr error
	delay := cfg.Delay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		attemptStart := time.Now().UTC()
		outcome, err := session.Run(ctx, instruction)
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
	if instruction.Params == nil {
		return cfg
	}
	if v, ok := instruction.Params["retryAttempts"].(float64); ok && v > 0 {
		cfg.MaxAttempts = int(v)
	}
	if v, ok := instruction.Params["retryDelayMs"].(float64); ok && v > 0 {
		cfg.Delay = time.Duration(v) * time.Millisecond
	}
	if v, ok := instruction.Params["retryBackoffFactor"].(float64); ok && v > 0 {
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
