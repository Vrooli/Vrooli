package executor

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestSimpleExecutorHappyPath(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	factory := &fakeEngineFactory{
		session: &fakeSession{
			outcome: contracts.StepOutcome{
				Success: true,
			},
			delay: 30 * time.Millisecond,
		},
	}

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "node-1", Type: "click"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 5 * time.Millisecond,
	}

	executor := NewSimpleExecutor(nil)
	if err := executor.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if len(rec.outcomes) != 1 {
		t.Fatalf("expected 1 recorded outcome, got %d", len(rec.outcomes))
	}
	outcome := rec.outcomes[0]
	if !outcome.Success {
		t.Fatalf("expected success outcome")
	}
	if outcome.StepIndex != 0 || outcome.NodeID != "node-1" {
		t.Fatalf("unexpected outcome metadata: %+v", outcome)
	}

	events := sink.Events()
	if len(events) < 3 {
		t.Fatalf("expected at least step started/heartbeat/complete events, got %d", len(events))
	}
	for i := 1; i < len(events); i++ {
		if events[i].Sequence <= events[i-1].Sequence {
			t.Fatalf("events not strictly ordered by sequence: %+v", events)
		}
	}

	hasHeartbeat := false
	hasComplete := false
	var heartbeatSeq []uint64
	for _, ev := range events {
		switch ev.Kind {
		case contracts.EventKindStepHeartbeat, contracts.EventKindStepTelemetry:
			hasHeartbeat = true
			if hb, ok := ev.Payload.(contracts.StepTelemetry); ok {
				heartbeatSeq = append(heartbeatSeq, hb.Sequence)
			}
		case contracts.EventKindStepCompleted:
			hasComplete = true
		}
	}
	if !hasHeartbeat {
		t.Fatalf("expected heartbeat telemetry to be emitted")
	}
	if !hasComplete {
		t.Fatalf("expected step completion event")
	}
	if len(heartbeatSeq) == 0 {
		t.Fatalf("expected heartbeat payload to be present")
	}
	if heartbeatSeq[0] != 1 {
		t.Fatalf("expected first heartbeat sequence to start at 1, got %d", heartbeatSeq[0])
	}
	for i := 1; i < len(heartbeatSeq); i++ {
		if heartbeatSeq[i] <= heartbeatSeq[i-1] {
			t.Fatalf("heartbeat sequences not strictly increasing: %+v", heartbeatSeq)
		}
	}
}

func TestSimpleExecutorPropagatesEngineError(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	factory := &fakeEngineFactory{
		session: &fakeSession{
			err: errors.New("engine boom"),
		},
	}

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "node-1", Type: "click"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(context.Background(), req)
	if err == nil || err.Error() != "engine boom" {
		t.Fatalf("expected engine error, got %v", err)
	}

	events := sink.Events()
	foundFailed := false
	for _, ev := range events {
		if ev.Kind == contracts.EventKindStepFailed {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Fatalf("expected step.failed event to be emitted")
	}
}

func TestSimpleExecutorEmitsArtifactsAndTimelineMetadata(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()
	artifactID := uuid.New()
	timelineID := uuid.New()

	rec := &artifactRecorder{
		result: recorder.RecordResult{
			ArtifactIDs:        []uuid.UUID{artifactID},
			TimelineArtifactID: &timelineID,
		},
	}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	factory := &fakeEngineFactory{
		session: &fakeSession{outcome: contracts.StepOutcome{Success: true}},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "node-1", Type: "navigate"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	}

	if err := NewSimpleExecutor(nil).Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	found := false
	for _, ev := range sink.Events() {
		if ev.Kind != contracts.EventKindStepCompleted {
			continue
		}
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			t.Fatalf("expected map payload, got %T", ev.Payload)
		}
		rawArtifacts, ok := payload["artifacts"].([]uuid.UUID)
		if !ok || len(rawArtifacts) != 1 || rawArtifacts[0] != artifactID {
			t.Fatalf("expected artifacts payload with %s, got %+v", artifactID, payload["artifacts"])
		}
		rawTimeline, ok := payload["timeline_artifact_id"].(uuid.UUID)
		if !ok || rawTimeline != timelineID {
			t.Fatalf("expected timeline artifact id %s, got %+v", timelineID, payload["timeline_artifact_id"])
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("expected step completion event with artifact metadata")
	}
}

func TestSimpleExecutorRunsSubflowWithWorkflowID(t *testing.T) {
	execID := uuid.New()
	parentWorkflowID := uuid.New()
	childWorkflowID := uuid.New()

	session := &fakeSession{
		outcomeByNode: map[string]contracts.StepOutcome{
			"child-step": {Success: true, NodeID: "child-step", StepType: "click"},
			"parent-set": {Success: true, NodeID: "parent-set", StepType: "set_variable"},
		},
	}
	factory := &fakeEngineFactory{session: session}

	childPlan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		WorkflowID:     childWorkflowID,
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "child-step", Type: "click", Params: map[string]any{}},
		},
		CreatedAt: time.Now().UTC(),
	}

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     parentWorkflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "subflow-node", Type: "subflow", Params: map[string]any{"workflowId": childWorkflowID.String()}},
				{Index: 1, NodeID: "parent-set", Type: "set_variable", Params: map[string]any{"name": "afterChild", "value": "ok"}},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
		WorkflowResolver:  &stubWorkflowResolver{workflows: map[uuid.UUID]*database.Workflow{childWorkflowID: {ID: childWorkflowID}}},
		PlanCompiler: &subflowPlanCompiler{plans: map[uuid.UUID]contracts.ExecutionPlan{
			childWorkflowID: childPlan,
		}},
		MaxSubflowDepth: 5,
	}

	executor := NewSimpleExecutor(nil)
	if err := executor.Execute(context.Background(), req); err != nil {
		t.Fatalf("expected subflow to succeed, got %v", err)
	}

	if session.runs < 1 {
		t.Fatalf("expected engine to run child steps, got %d", session.runs)
	}
	if len(rec.outcomes) == 0 {
		t.Fatalf("expected recorder to capture subflow outcomes")
	}
	if len(sink.Events()) == 0 {
		t.Fatalf("expected events emitted for subflow execution")
	}
}

func TestSimpleExecutorRunsInlineSubflowAndMergesVars(t *testing.T) {
	execID := uuid.New()
	parentWorkflowID := uuid.New()

	session := &fakeSession{
		outcomeByNode: map[string]contracts.StepOutcome{
			"child-set": {Success: true, NodeID: "child-set", StepType: "set_variable"},
			"parent-assert": {
				Success: true, NodeID: "parent-assert", StepType: "assert",
			},
		},
	}
	factory := &fakeEngineFactory{session: session}

	inlineChildID := uuid.New()
	childPlan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		WorkflowID:     inlineChildID,
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "child-set", Type: "set_variable", Params: map[string]any{"name": "foo", "value": "bar"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     parentWorkflowID,
			Instructions: []contracts.CompiledInstruction{
				{
					Index: 0, NodeID: "subflow-inline", Type: "subflow",
					Params: map[string]any{"workflowDefinition": map[string]any{"nodes": []any{}, "edges": []any{}}},
				},
				{Index: 1, NodeID: "parent-assert", Type: "assert", Params: map[string]any{"selector": "${foo}"}},
			},
			CreatedAt: time.Now().UTC(),
			Metadata: map[string]any{
				"variables": map[string]any{},
			},
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
		WorkflowResolver:  &stubWorkflowResolver{},
		PlanCompiler: &subflowPlanCompiler{
			plans:       map[uuid.UUID]contracts.ExecutionPlan{inlineChildID: childPlan},
			defaultPlan: &childPlan,
		},
	}

	executor := NewSimpleExecutor(nil)
	if err := executor.Execute(context.Background(), req); err != nil {
		t.Fatalf("expected inline subflow to succeed, got %v", err)
	}
	if session.runs < 1 {
		t.Fatalf("expected engine runs for child/parent steps, got %d", session.runs)
	}
	if len(rec.outcomes) == 0 {
		t.Fatalf("expected recorder to capture outcomes")
	}
}

func TestSimpleExecutorDetectsSubflowRecursion(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	session := &fakeSession{}
	factory := &fakeEngineFactory{session: session}

	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "self", Type: "subflow", Params: map[string]any{"workflowId": workflowID.String()}},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
		WorkflowResolver:  &stubWorkflowResolver{workflows: map[uuid.UUID]*database.Workflow{workflowID: {ID: workflowID}}},
		PlanCompiler: &subflowPlanCompiler{plans: map[uuid.UUID]contracts.ExecutionPlan{
			workflowID: {
				SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
				PayloadVersion: contracts.PayloadVersion,
				ExecutionID:    execID,
				WorkflowID:     workflowID,
				Instructions: []contracts.CompiledInstruction{
					{Index: 0, NodeID: "noop", Type: "click"},
				},
				CreatedAt: time.Now().UTC(),
			},
		}},
		MaxSubflowDepth: 3,
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "recursion") {
		t.Fatalf("expected recursion detection, got %v", err)
	}
}

// Ensures capability gaps fail fast before execution.
func TestSimpleExecutorFailsOnCapabilityGap(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	// Engine that does not support uploads/downloads.
	factory := &fakeEngineFactory{
		session: &fakeSession{outcome: contracts.StepOutcome{Success: true}},
		caps: &contracts.EngineCapabilities{
			SchemaVersion:         contracts.CapabilitiesSchemaVersion,
			Engine:                "stub",
			MaxConcurrentSessions: 1,
			AllowsParallelTabs:    false,
			SupportsHAR:           false,
			SupportsVideo:         false,
			SupportsIframes:       false,
			SupportsFileUploads:   false,
			SupportsDownloads:     false,
			SupportsTracing:       false,
			MaxViewportWidth:      1024,
			MaxViewportHeight:     768,
		},
	}

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "upload", Type: "upload", Params: map[string]any{"filePaths": []any{"one.txt"}}},
			},
			CreatedAt: time.Now().UTC(),
			Metadata: map[string]any{
				"executionViewport": map[string]any{"width": float64(1920), "height": float64(1080)},
			},
		},
		EngineName:        "stub",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected capability error")
	}
	if _, ok := err.(*CapabilityError); !ok {
		t.Fatalf("expected CapabilityError, got %T: %v", err, err)
	}
}

func TestSimpleExecutorHonorsInstructionTimeout(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	// The executor adds a 2-second buffer to instruction timeouts for HTTP overhead.
	// To trigger a timeout, the session delay must exceed timeout + buffer.
	// We use a 100ms timeout with a 2.5s delay, ensuring timeout triggers.
	session := &fakeSession{
		outcome: contracts.StepOutcome{Success: true},
		delay:   2500 * time.Millisecond,
	}

	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: execID,
			WorkflowID:  workflowID,
			Instructions: []contracts.CompiledInstruction{
				// 100ms timeout + 2s buffer = 2.1s context deadline
				// 2.5s delay > 2.1s deadline -> timeout triggers
				{Index: 0, NodeID: "slow", Type: "navigate", Params: map[string]any{"timeoutMs": float64(100)}},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineFactory:     &fakeEngineFactory{session: session},
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	}

	executor := NewSimpleExecutor(nil)
	start := time.Now()
	err := executor.Execute(context.Background(), req)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	// Should abort within ~2.1s (timeout + buffer), with some margin
	if time.Since(start) > 3*time.Second {
		t.Fatalf("timeout should abort within buffer window, took %v", time.Since(start))
	}
	if len(rec.outcomes) != 1 {
		t.Fatalf("expected recorder to capture one outcome, got %d", len(rec.outcomes))
	}
	out := rec.outcomes[0]
	if out.Success {
		t.Fatalf("expected timeout outcome to be unsuccessful")
	}
	if out.Failure == nil || out.Failure.Kind != contracts.FailureKindTimeout || out.Failure.Retryable {
		t.Fatalf("expected timeout failure non-retryable, got %+v", out.Failure)
	}
}

func TestSimpleExecutorExecutionTimeoutCancelsRun(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	session := &fakeSession{
		outcome: contracts.StepOutcome{Success: true},
		delay:   50 * time.Millisecond,
	}

	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: execID,
			WorkflowID:  workflowID,
			Metadata: map[string]any{
				"executionTimeoutMs": float64(5),
			},
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "slow", Type: "navigate"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineFactory:     &fakeEngineFactory{session: session},
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(context.Background(), req)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected execution timeout (deadline exceeded), got %v", err)
	}
	if len(rec.outcomes) != 1 {
		t.Fatalf("expected one recorded timeout outcome, got %d", len(rec.outcomes))
	}
	out := rec.outcomes[0]
	if out.Failure == nil || out.Failure.Kind != contracts.FailureKindTimeout {
		t.Fatalf("expected timeout failure, got %+v", out.Failure)
	}
	if out.Failure.Source != contracts.FailureSourceExecutor {
		t.Fatalf("expected failure source executor, got %s", out.Failure.Source)
	}
}

func TestSimpleExecutorCancellationRecordsCancelledOutcome(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	session := &fakeSession{
		outcome: contracts.StepOutcome{Success: true},
	}
	rec := &memoryRecorder{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: execID,
			WorkflowID:  workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "cancel-me", Type: "navigate"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineFactory:     &fakeEngineFactory{session: session},
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(ctx, req)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
	if len(rec.outcomes) != 1 {
		t.Fatalf("expected one recorded cancellation outcome, got %d", len(rec.outcomes))
	}
	out := rec.outcomes[0]
	if out.Failure == nil || out.Failure.Kind != contracts.FailureKindCancelled || out.Failure.Retryable {
		t.Fatalf("expected cancelled failure, got %+v", out.Failure)
	}
}

func TestNormalizeOutcomeDefaultsAndFailureTaxonomy(t *testing.T) {
	exec := NewSimpleExecutor(nil)
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		WorkflowID:     workflowID,
		CreatedAt:      time.Now().UTC(),
	}
	instruction := contracts.CompiledInstruction{
		Index:  2,
		NodeID: "node-x",
		Type:   "navigate",
	}

	start := time.Now().Add(-10 * time.Millisecond).UTC()
	out := exec.normalizeOutcome(plan, instruction, 0, start, contracts.StepOutcome{}, context.DeadlineExceeded)

	if out.SchemaVersion != contracts.StepOutcomeSchemaVersion || out.PayloadVersion != contracts.PayloadVersion {
		t.Fatalf("expected schema/payload defaults, got %s/%s", out.SchemaVersion, out.PayloadVersion)
	}
	if out.ExecutionID != execID || out.StepIndex != 2 || out.NodeID != "node-x" || out.StepType != "navigate" {
		t.Fatalf("unexpected metadata: %+v", out)
	}
	if out.Attempt != 0 {
		t.Fatalf("expected attempt fallback to 0, got %d", out.Attempt)
	}
	if out.CompletedAt == nil || out.DurationMs == 0 {
		t.Fatalf("expected completion timestamp and duration to be set, got %+v", out)
	}
	if out.Success {
		t.Fatalf("expected success=false when runErr present")
	}
	if out.Failure == nil || out.Failure.Kind != contracts.FailureKindTimeout || out.Failure.Retryable {
		t.Fatalf("expected timeout failure and non-retryable, got %+v", out.Failure)
	}
}

func TestNormalizeOutcomeDefaultsFailureSource(t *testing.T) {
	exec := NewSimpleExecutor(nil)
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		CreatedAt:   time.Now().UTC(),
	}
	instruction := contracts.CompiledInstruction{
		Index:  1,
		NodeID: "node-1",
		Type:   "click",
	}

	raw := contracts.StepOutcome{
		Success: false,
		Failure: &contracts.StepFailure{Kind: contracts.FailureKindEngine},
	}

	out := exec.normalizeOutcome(plan, instruction, 1, time.Now().UTC(), raw, nil)
	if out.Failure == nil || out.Failure.Source != contracts.FailureSourceEngine {
		t.Fatalf("expected failure source to default to engine, got %+v", out.Failure)
	}
}

func TestGraphExecutorConditionalBranching(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	factory := &fakeEngineFactory{
		session: &fakeSession{
			outcomeByNode: map[string]contracts.StepOutcome{
				"cond": {
					Success: true,
					Condition: &contracts.ConditionOutcome{
						Outcome: true,
					},
				},
				"true-branch": {Success: true},
				"false-branch": {
					Success: false,
					Failure: &contracts.StepFailure{Kind: contracts.FailureKindEngine, Message: "should not run"},
				},
			},
		},
	}

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	graph := &contracts.PlanGraph{
		Steps: []contracts.PlanStep{
			{
				Index:  0,
				NodeID: "cond",
				Type:   "conditional",
				Outgoing: []contracts.PlanEdge{
					{Target: "true-branch", Condition: "true"},
					{Target: "false-branch", Condition: "false"},
				},
			},
			{Index: 1, NodeID: "true-branch", Type: "click"},
			{Index: 2, NodeID: "false-branch", Type: "click"},
		},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: execID,
			WorkflowID:  workflowID,
			Graph:       graph,
			CreatedAt:   time.Now().UTC(),
		},
		EngineFactory: factory,
		Recorder:      rec,
		EventSink:     sink,
	}

	executor := NewSimpleExecutor(nil)
	if err := executor.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if len(rec.outcomes) != 2 {
		t.Fatalf("expected only conditional and true branch recorded, got %d", len(rec.outcomes))
	}
	if rec.outcomes[1].NodeID != "true-branch" {
		t.Fatalf("expected to follow true branch, got %s", rec.outcomes[1].NodeID)
	}
}

func TestGraphExecutorRepeatLoop(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	factory := &fakeEngineFactory{
		session: &fakeSession{
			outcome: contracts.StepOutcome{Success: true},
		},
	}

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	loopBody := &contracts.PlanGraph{
		Steps: []contracts.PlanStep{
			{
				Index:  0,
				NodeID: "loop-step-a",
				Type:   "click",
				Outgoing: []contracts.PlanEdge{
					{Target: "loop-step-b"},
				},
			},
			{
				Index:    1,
				NodeID:   "loop-step-b",
				Type:     "click",
				Outgoing: []contracts.PlanEdge{},
			},
		},
	}

	graph := &contracts.PlanGraph{
		Steps: []contracts.PlanStep{
			{
				Index:  0,
				NodeID: "loop",
				Type:   "loop",
				Params: map[string]any{
					"loopType":          "repeat",
					"loopCount":         2,
					"loopMaxIterations": 5,
				},
				Loop: loopBody,
			},
		},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: execID,
			WorkflowID:  workflowID,
			Graph:       graph,
			CreatedAt:   time.Now().UTC(),
		},
		EngineFactory: factory,
		Recorder:      rec,
		EventSink:     sink,
	}

	executor := NewSimpleExecutor(nil)
	if err := executor.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if len(rec.outcomes) != 5 {
		t.Fatalf("expected 5 outcomes (2 iterations * 2 steps + loop), got %d", len(rec.outcomes))
	}
}

func TestGraphExecutorFailureBranch(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	factory := &fakeEngineFactory{
		session: &fakeSession{
			outcomeByNode: map[string]contracts.StepOutcome{
				"start": {
					Success: false,
					Failure: &contracts.StepFailure{Kind: contracts.FailureKindEngine, Message: "boom"},
				},
				"failure-path": {Success: true},
			},
		},
	}

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	graph := &contracts.PlanGraph{
		Steps: []contracts.PlanStep{
			{
				Index:  0,
				NodeID: "start",
				Type:   "click",
				Outgoing: []contracts.PlanEdge{
					{Target: "failure-path", Condition: "failure"},
					{Target: "success-path", Condition: "success"},
				},
			},
			{Index: 1, NodeID: "failure-path", Type: "click"},
			{Index: 2, NodeID: "success-path", Type: "click"},
		},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: execID,
			WorkflowID:  workflowID,
			Graph:       graph,
			CreatedAt:   time.Now().UTC(),
		},
		EngineFactory: factory,
		Recorder:      rec,
		EventSink:     sink,
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected executor to return failure")
	}
	if len(rec.outcomes) != 2 {
		t.Fatalf("expected failure branch to execute one additional step, got %d outcomes", len(rec.outcomes))
	}
	if rec.outcomes[1].NodeID != "failure-path" {
		t.Fatalf("expected failure branch, got %s", rec.outcomes[1].NodeID)
	}
}

func TestSimpleExecutorEventPayloadsCarrySchemaVersions(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	factory := &fakeEngineFactory{
		session: &fakeSession{
			outcome: contracts.StepOutcome{
				Success: true,
			},
		},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "node-1", Type: "click"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	}

	executor := NewSimpleExecutor(nil)
	if err := executor.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if len(rec.outcomes) != 1 {
		t.Fatalf("expected recorder to capture one outcome, got %d", len(rec.outcomes))
	}
	outcome := rec.outcomes[0]
	if outcome.SchemaVersion != contracts.StepOutcomeSchemaVersion || outcome.PayloadVersion != contracts.PayloadVersion {
		t.Fatalf("expected outcome schema/payload versions to be set, got %s/%s", outcome.SchemaVersion, outcome.PayloadVersion)
	}

	evs := sink.Events()
	if len(evs) == 0 {
		t.Fatalf("expected events to be emitted")
	}
	for _, ev := range evs {
		if ev.SchemaVersion != contracts.EventEnvelopeSchemaVersion || ev.PayloadVersion != contracts.PayloadVersion {
			t.Fatalf("expected envelope schema/payload versions to be set, got %s/%s", ev.SchemaVersion, ev.PayloadVersion)
		}
	}

	// Ensure completion payload still carries the normalized outcome.
	hasComplete := false
	for _, ev := range evs {
		if ev.Kind == contracts.EventKindStepCompleted {
			hasComplete = true
			payload, ok := ev.Payload.(map[string]any)
			if !ok {
				t.Fatalf("expected step completion payload map, got %T", ev.Payload)
			}
			rawOutcome, ok := payload["outcome"].(contracts.StepOutcome)
			if !ok {
				t.Fatalf("expected outcome in completion payload, got %T", payload["outcome"])
			}
			if rawOutcome.SchemaVersion != contracts.StepOutcomeSchemaVersion || rawOutcome.PayloadVersion != contracts.PayloadVersion {
				t.Fatalf("expected completion outcome schema/payload versions to be set, got %s/%s", rawOutcome.SchemaVersion, rawOutcome.PayloadVersion)
			}
		}
	}
	if !hasComplete {
		t.Fatalf("expected a step.completed event")
	}
}

func TestSimpleExecutorCapabilityError(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	factory := &fakeEngineFactory{
		session: &fakeSession{
			outcome: contracts.StepOutcome{Success: true},
		},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "node-1", Type: "click", Params: map[string]any{"tabSwitchBy": "title"}},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          &memoryRecorder{},
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	}

	if len(req.Plan.Instructions) == 0 {
		t.Fatalf("expected instructions present")
	}
	if _, ok := req.Plan.Instructions[0].Params["tabSwitchBy"]; !ok {
		t.Fatalf("expected tabSwitchBy param present, got %+v", req.Plan.Instructions[0].Params)
	}
	if !requiresParallelTabs(req.Plan.Instructions[0].Params) {
		t.Fatalf("requiresParallelTabs should be true for params %+v", req.Plan.Instructions[0].Params)
	}

	reqs := deriveRequirements(req.Plan)
	if !reqs.NeedsParallelTabs {
		t.Fatalf("expected requirements to mark parallel tabs needed, got %+v", reqs)
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected capability error")
	}
	if _, ok := err.(*CapabilityError); !ok {
		t.Fatalf("expected CapabilityError, got %T", err)
	}
}

func TestSimpleExecutorCapabilityErrorDownloads(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	factory := &fakeEngineFactory{
		session: &fakeSession{
			outcome: contracts.StepOutcome{Success: true},
		},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "download", Type: "download_file"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          &memoryRecorder{},
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected capability error")
	}
	if _, ok := err.(*CapabilityError); !ok {
		t.Fatalf("expected CapabilityError, got %T", err)
	}
}

func TestSimpleExecutorCapabilityErrorReasonsFilteredToMissing(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		WorkflowID:     workflowID,
		Metadata: map[string]any{
			"requiresDownloads": true,
			"requiresVideo":     true,
		},
		Instructions: []contracts.CompiledInstruction{
			{
				Index:  0,
				NodeID: "mock",
				Type:   "network_mock",
				Params: map[string]any{
					"networkMockType":   "abort",
					"networkUrlPattern": "*",
				},
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	caps := &contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                "fake",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    true,
		SupportsHAR:           false, // missing
		SupportsVideo:         true,  // satisfied; should not appear in reasons
		SupportsIframes:       true,
		SupportsFileUploads:   true,
		SupportsDownloads:     false, // missing
		SupportsTracing:       false, // missing
	}

	req := Request{
		Plan:              plan,
		EngineName:        "browserless",
		EngineFactory:     &fakeEngineFactory{session: &fakeSession{outcome: contracts.StepOutcome{Success: true}}, caps: caps},
		Recorder:          &memoryRecorder{},
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	}

	err := NewSimpleExecutor(nil).Execute(context.Background(), req)
	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Fatalf("expected CapabilityError, got %T (%v)", err, err)
	}

	if len(capErr.Missing) == 0 {
		t.Fatalf("expected missing capabilities recorded")
	}

	// Reasons should only include missing keys (har, downloads, tracing) and
	// omit satisfied capabilities such as video.
	for key := range capErr.Reasons {
		switch key {
		case "har", "downloads", "tracing":
		default:
			t.Fatalf("unexpected reason key %q; expected only missing capabilities", key)
		}
	}
	if _, ok := capErr.Reasons["video"]; ok {
		t.Fatalf("video is satisfied and should not appear in reasons map")
	}
	if len(capErr.Reasons["har"]) == 0 || len(capErr.Reasons["downloads"]) == 0 || len(capErr.Reasons["tracing"]) == 0 {
		t.Fatalf("expected reasons to be populated for missing capabilities, got %+v", capErr.Reasons)
	}
}

func TestSimpleExecutorUsesProvidedCapsWithoutFetchingEngine(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		WorkflowID:     workflowID,
		Metadata: map[string]any{
			"requiresDownloads": true,
		},
		Instructions: []contracts.CompiledInstruction{
			{
				Index:  0,
				NodeID: "mock",
				Type:   "network_mock",
				Params: map[string]any{
					"networkMockType": "abort",
				},
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	engineCaps := &contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                "preflight",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    true,
		SupportsHAR:           false,
		SupportsVideo:         true,
		SupportsIframes:       true,
		SupportsFileUploads:   true,
		SupportsDownloads:     false,
		SupportsTracing:       false,
	}

	factory := &fakeEngineFactory{
		session: &fakeSession{outcome: contracts.StepOutcome{Success: true}},
	}

	req := Request{
		Plan:              plan,
		EngineName:        "browserless",
		EngineFactory:     factory,
		Recorder:          &memoryRecorder{},
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
		EngineCaps:        engineCaps,
	}

	err := NewSimpleExecutor(nil).Execute(context.Background(), req)
	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Fatalf("expected CapabilityError, got %T (%v)", err, err)
	}
	if factory.engine == nil {
		t.Fatalf("expected fake engine to be initialized")
	}
	if factory.engine.capCalls != 0 {
		t.Fatalf("expected provided EngineCaps to skip Capabilities fetch, got %d calls", factory.engine.capCalls)
	}
	for _, missing := range []string{"downloads", "har", "tracing"} {
		found := false
		for _, gap := range capErr.Missing {
			if gap == missing {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected missing capability %s in gap %+v", missing, capErr.Missing)
		}
	}
}

func TestSimpleExecutorValidatesEngineCapabilities(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	badCaps := &contracts.EngineCapabilities{
		Engine:                "fake",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    true,
	}

	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "nav", Type: "navigate"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     &fakeEngineFactory{session: &fakeSession{outcome: contracts.StepOutcome{Success: true}}, caps: badCaps},
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	}

	err := NewSimpleExecutor(nil).Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected invalid capability error")
	}
	if !strings.Contains(err.Error(), "engine capabilities invalid") {
		t.Fatalf("expected invalid capabilities error, got %v", err)
	}
	if len(rec.outcomes) != 0 {
		t.Fatalf("expected no outcomes recorded on capability validation failure, got %d", len(rec.outcomes))
	}
}

func TestEntryProbeFailsFast(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	session := &fakeSession{
		outcomeByNode: map[string]contracts.StepOutcome{
			entryProbeNodeID: {
				Success: false,
				Failure: &contracts.StepFailure{Message: "missing selector"},
			},
		},
		outcome: contracts.StepOutcome{Success: true},
	}

	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "node-1", Type: "click", Params: map[string]any{"selector": "[data-testid=ready]"}, Metadata: map[string]string{}},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineFactory:     &fakeEngineFactory{session: session},
		Recorder:          &memoryRecorder{},
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 5 * time.Millisecond,
	}

	executor := NewSimpleExecutor(nil)
	err := executor.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected entry probe failure")
	}
	if session.runs != 1 {
		t.Fatalf("expected only entry probe to run, got %d runs", session.runs)
	}
}

func TestEntryProbePassesAndRunsSteps(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	session := &fakeSession{
		outcomeByNode: map[string]contracts.StepOutcome{
			entryProbeNodeID: {Success: true},
		},
		outcome: contracts.StepOutcome{Success: true},
	}

	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "node-1", Type: "click", Params: map[string]any{"selector": "[data-testid=ready]"}, Metadata: map[string]string{}},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineFactory:     &fakeEngineFactory{session: session},
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 5 * time.Millisecond,
	}

	executor := NewSimpleExecutor(nil)
	if err := executor.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if session.runs != 2 {
		t.Fatalf("expected entry probe + step to run, got %d runs", session.runs)
	}
	if len(rec.outcomes) != 1 || !rec.outcomes[0].Success {
		t.Fatalf("expected recorded step outcome, got %+v", rec.outcomes)
	}
}

func TestSimpleExecutorRejectsPlanWithMismatchedSchemaVersions(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			SchemaVersion:  "unexpected-schema",
			PayloadVersion: contracts.PayloadVersion,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "nav", Type: "navigate"},
			},
			CreatedAt: time.Now().UTC(),
		},
		EngineName:        "browserless",
		EngineFactory:     &fakeEngineFactory{session: &fakeSession{outcome: contracts.StepOutcome{Success: true}}},
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	}

	err := NewSimpleExecutor(nil).Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected schema mismatch error")
	}
	if !strings.Contains(err.Error(), "schema_version mismatch") {
		t.Fatalf("expected schema mismatch error message, got %v", err)
	}
	if len(rec.outcomes) != 0 {
		t.Fatalf("expected no outcomes recorded on schema mismatch, got %d", len(rec.outcomes))
	}
}

func TestSimpleExecutorResetsSessionInCleanMode(t *testing.T) {
	session := &fakeSession{
		outcomeByNode: map[string]contracts.StepOutcome{
			"one": {Success: true},
			"two": {Success: true},
		},
	}

	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "one", Type: "navigate"},
				{Index: 1, NodeID: "two", Type: "click"},
			},
		},
		EngineFactory:     &fakeEngineFactory{session: session},
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
		ReuseMode:         engine.ReuseModeClean,
	}

	exec := NewSimpleExecutor(nil)
	if err := exec.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if session.resets != 2 {
		t.Fatalf("expected session reset after each step, got %d", session.resets)
	}
}

func TestSimpleExecutorFreshCreatesSessionPerStep(t *testing.T) {
	session := &fakeSession{
		outcomeByNode: map[string]contracts.StepOutcome{
			"a": {Success: true},
			"b": {Success: true},
		},
	}

	engineFactory := &fakeEngineFactory{session: session}
	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "a", Type: "navigate"},
				{Index: 1, NodeID: "b", Type: "click"},
			},
		},
		EngineFactory:     engineFactory,
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
		ReuseMode:         engine.ReuseModeFresh,
	}

	exec := NewSimpleExecutor(nil)
	if err := exec.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if engineFactory.engine == nil || engineFactory.engine.startCount != 2 {
		t.Fatalf("expected StartSession to be called per step (2), got %d", engineFactory.engine.startCount)
	}
	if session.resets != 0 {
		t.Fatalf("fresh mode should not reset; got %d resets", session.resets)
	}
}

func TestGraphExecutorCleanResetsBetweenSteps(t *testing.T) {
	session := &fakeSession{
		outcomeByNode: map[string]contracts.StepOutcome{
			"first":  {Success: true},
			"second": {Success: true},
		},
	}

	engineFactory := &fakeEngineFactory{session: session}
	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
			Graph: &contracts.PlanGraph{
				Steps: []contracts.PlanStep{
					{Index: 0, NodeID: "first", Type: "navigate", Outgoing: []contracts.PlanEdge{{Target: "second"}}},
					{Index: 1, NodeID: "second", Type: "click"},
				},
			},
		},
		EngineFactory:     engineFactory,
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
		ReuseMode:         engine.ReuseModeClean,
	}

	exec := NewSimpleExecutor(nil)
	if err := exec.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if session.resets != 2 {
		t.Fatalf("expected reset after each graph step (2), got %d", session.resets)
	}
}

func TestGraphExecutorFreshCreatesSessionPerGraphStep(t *testing.T) {
	session := &fakeSession{
		outcomeByNode: map[string]contracts.StepOutcome{
			"first":  {Success: true},
			"second": {Success: true},
		},
	}

	engineFactory := &fakeEngineFactory{session: session}
	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
			Graph: &contracts.PlanGraph{
				Steps: []contracts.PlanStep{
					{Index: 0, NodeID: "first", Type: "navigate", Outgoing: []contracts.PlanEdge{{Target: "second"}}},
					{Index: 1, NodeID: "second", Type: "click"},
				},
			},
		},
		EngineFactory:     engineFactory,
		Recorder:          rec,
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
		ReuseMode:         engine.ReuseModeFresh,
	}

	exec := NewSimpleExecutor(nil)
	if err := exec.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if engineFactory.engine == nil || engineFactory.engine.startCount != 2 {
		t.Fatalf("expected StartSession per graph step (2), got %d", engineFactory.engine.startCount)
	}
	if session.resets != 0 {
		t.Fatalf("fresh mode should not reset in graph mode; got %d resets", session.resets)
	}
}

// --- test fakes ---

type fakeEngineFactory struct {
	session *fakeSession
	engine  *fakeEngine
	caps    *contracts.EngineCapabilities
}

func (f *fakeEngineFactory) Resolve(ctx context.Context, name string) (engine.AutomationEngine, error) {
	if f.engine == nil {
		f.engine = &fakeEngine{session: f.session, caps: f.caps}
	}
	return f.engine, nil
}

type fakeEngine struct {
	session    *fakeSession
	startCount int
	caps       *contracts.EngineCapabilities
	capCalls   int
}

func (f *fakeEngine) Name() string { return "fake" }

func (f *fakeEngine) Capabilities(ctx context.Context) (contracts.EngineCapabilities, error) {
	f.capCalls++
	if f.caps != nil {
		return *f.caps, nil
	}
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                "fake",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    false,
	}, nil
}

func (f *fakeEngine) StartSession(ctx context.Context, spec engine.SessionSpec) (engine.EngineSession, error) {
	f.startCount++
	return f.session, nil
}

type fakeSession struct {
	outcome       contracts.StepOutcome
	outcomeByNode map[string]contracts.StepOutcome
	err           error
	delay         time.Duration
	resets        int
	runs          int
}

func (f *fakeSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	f.runs++
	if f.delay > 0 {
		select {
		case <-ctx.Done():
			return contracts.StepOutcome{}, ctx.Err()
		case <-time.After(f.delay):
		}
	}
	if f.outcomeByNode != nil {
		if o, ok := f.outcomeByNode[instruction.NodeID]; ok {
			return o, f.err
		}
	}
	return f.outcome, f.err
}

func (f *fakeSession) Reset(ctx context.Context) error {
	f.resets++
	return nil
}
func (f *fakeSession) Close(ctx context.Context) error { return nil }

type memoryRecorder struct {
	mu        sync.Mutex
	outcomes  []contracts.StepOutcome
	telemetry []contracts.StepTelemetry
}

func (m *memoryRecorder) RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (recorder.RecordResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outcomes = append(m.outcomes, outcome)
	return recorder.RecordResult{}, nil
}

func (m *memoryRecorder) RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.telemetry = append(m.telemetry, telemetry)
	return nil
}

func (m *memoryRecorder) MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error {
	return nil
}

func (m *memoryRecorder) UpdateCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, totalSteps int) error {
	return nil
}

type artifactRecorder struct {
	result recorder.RecordResult
}

func (a *artifactRecorder) RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (recorder.RecordResult, error) {
	return a.result, nil
}

func (a *artifactRecorder) RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error {
	return nil
}

func (a *artifactRecorder) MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error {
	return nil
}

func (a *artifactRecorder) UpdateCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, totalSteps int) error {
	return nil
}

type stubWorkflowResolver struct {
	workflows map[uuid.UUID]*database.Workflow
}

func (s *stubWorkflowResolver) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*database.Workflow, error) {
	if wf, ok := s.workflows[workflowID]; ok {
		return wf, nil
	}
	return nil, database.ErrNotFound
}

func (s *stubWorkflowResolver) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.Workflow, error) {
	return s.GetWorkflow(ctx, workflowID)
}

func (s *stubWorkflowResolver) GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string) (*database.Workflow, error) {
	return nil, database.ErrNotFound
}

type subflowPlanCompiler struct {
	plans       map[uuid.UUID]contracts.ExecutionPlan
	defaultPlan *contracts.ExecutionPlan
}

func (s *subflowPlanCompiler) Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	if workflow == nil {
		return contracts.ExecutionPlan{}, nil, errors.New("workflow required")
	}
	if plan, ok := s.plans[workflow.ID]; ok {
		return plan, plan.Instructions, nil
	}
	if s.defaultPlan != nil {
		return *s.defaultPlan, s.defaultPlan.Instructions, nil
	}
	return contracts.ExecutionPlan{}, nil, errors.New("plan not found")
}

// TestValidateSubflowResolver verifies that the executor correctly validates
// WorkflowResolver presence when subflow instructions reference external workflows.
func TestValidateSubflowResolver(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()
	childWorkflowID := uuid.New()

	factory := &fakeEngineFactory{session: &fakeSession{outcome: contracts.StepOutcome{Success: true}}}
	rec := &memoryRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	tests := []struct {
		name          string
		instructions  []contracts.CompiledInstruction
		resolver      WorkflowResolver
		expectError   bool
		errorContains string
	}{
		{
			name: "no subflow - no resolver required",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "nav-1", Type: "navigate", Params: map[string]any{"url": "https://example.com"}},
			},
			resolver:    nil,
			expectError: false,
		},
		{
			name: "inline subflow - no resolver required",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "subflow-inline", Type: "subflow", Params: map[string]any{
					"workflowDefinition": map[string]any{
						"nodes": []any{},
						"edges": []any{},
					},
				}},
			},
			resolver:    nil,
			expectError: false,
		},
		{
			name: "external subflow without resolver - error",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "subflow-ext", Type: "subflow", Params: map[string]any{
					"workflowId": childWorkflowID.String(),
				}},
			},
			resolver:      nil,
			expectError:   true,
			errorContains: "WorkflowResolver is required",
		},
		{
			name: "external subflow with resolver - ok",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "subflow-ext", Type: "subflow", Params: map[string]any{
					"workflowId": childWorkflowID.String(),
				}},
			},
			resolver:    &stubWorkflowResolver{workflows: map[uuid.UUID]*database.Workflow{childWorkflowID: {ID: childWorkflowID}}},
			expectError: false,
		},
		{
			name: "subflow with both workflowId and workflowDefinition (inline takes precedence) - ok",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "subflow-both", Type: "subflow", Params: map[string]any{
					"workflowId": childWorkflowID.String(),
					"workflowDefinition": map[string]any{
						"nodes": []any{},
						"edges": []any{},
					},
				}},
			},
			resolver:    nil,
			expectError: false,
		},
		{
			name: "empty workflowId string - no resolver required",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "subflow-empty", Type: "subflow", Params: map[string]any{
					"workflowId": "",
				}},
			},
			resolver:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := Request{
				Plan: contracts.ExecutionPlan{
					SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
					PayloadVersion: contracts.PayloadVersion,
					ExecutionID:    execID,
					WorkflowID:     workflowID,
					Instructions:   tt.instructions,
					CreatedAt:      time.Now().UTC(),
				},
				EngineName:       "browserless",
				EngineFactory:    factory,
				Recorder:         rec,
				EventSink:        sink,
				WorkflowResolver: tt.resolver,
			}

			executor := NewSimpleExecutor(nil)
			err := executor.Execute(context.Background(), req)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil && strings.Contains(err.Error(), "WorkflowResolver") {
				// If we got a resolver error but didn't expect one, that's a test failure
				t.Fatalf("unexpected WorkflowResolver error: %v", err)
			}
			// Note: other errors (like execution failures) are ok since we're only testing validation
		})
	}
}

// TestComputeDynamicTimeout verifies the dynamic timeout calculation based on step count.
func TestComputeDynamicTimeout(t *testing.T) {
	tests := []struct {
		name         string
		instructions []contracts.CompiledInstruction
		wantMin      time.Duration
		wantMax      time.Duration
	}{
		{
			name:         "empty instructions - returns minimum",
			instructions: []contracts.CompiledInstruction{},
			wantMin:      90 * time.Second,
			wantMax:      90 * time.Second,
		},
		{
			name: "1 step - returns minimum (30s base + 10s = 40s < 90s min)",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "n1", Type: "click"},
			},
			wantMin: 90 * time.Second,
			wantMax: 90 * time.Second,
		},
		{
			name: "6 steps - still returns minimum (30s + 60s = 90s = min)",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "n1", Type: "click"},
				{Index: 1, NodeID: "n2", Type: "click"},
				{Index: 2, NodeID: "n3", Type: "click"},
				{Index: 3, NodeID: "n4", Type: "click"},
				{Index: 4, NodeID: "n5", Type: "click"},
				{Index: 5, NodeID: "n6", Type: "click"},
			},
			wantMin: 90 * time.Second,
			wantMax: 90 * time.Second,
		},
		{
			name: "10 steps simple workflow - 30s base + 100s = 130s",
			instructions: func() []contracts.CompiledInstruction {
				var instrs []contracts.CompiledInstruction
				for i := 0; i < 10; i++ {
					instrs = append(instrs, contracts.CompiledInstruction{Index: i, NodeID: "n", Type: "click"})
				}
				return instrs
			}(),
			wantMin: 130 * time.Second,
			wantMax: 130 * time.Second,
		},
		{
			name: "10 steps with subflow - 30s base + 150s = 180s",
			instructions: func() []contracts.CompiledInstruction {
				var instrs []contracts.CompiledInstruction
				for i := 0; i < 9; i++ {
					instrs = append(instrs, contracts.CompiledInstruction{Index: i, NodeID: "n", Type: "click"})
				}
				instrs = append(instrs, contracts.CompiledInstruction{Index: 9, NodeID: "sf", Type: "subflow"})
				return instrs
			}(),
			wantMin: 180 * time.Second,
			wantMax: 180 * time.Second,
		},
		{
			name: "25 steps - hits maximum (30s + 250s = 280s > 270s max)",
			instructions: func() []contracts.CompiledInstruction {
				var instrs []contracts.CompiledInstruction
				for i := 0; i < 25; i++ {
					instrs = append(instrs, contracts.CompiledInstruction{Index: i, NodeID: "n", Type: "click"})
				}
				return instrs
			}(),
			wantMin: 270 * time.Second,
			wantMax: 270 * time.Second,
		},
		{
			name: "50 steps - capped at maximum",
			instructions: func() []contracts.CompiledInstruction {
				var instrs []contracts.CompiledInstruction
				for i := 0; i < 50; i++ {
					instrs = append(instrs, contracts.CompiledInstruction{Index: i, NodeID: "n", Type: "navigate"})
				}
				return instrs
			}(),
			wantMin: 270 * time.Second,
			wantMax: 270 * time.Second,
		},
		{
			name: "subflow type case insensitive - SUBFLOW",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "n1", Type: "click"},
				{Index: 1, NodeID: "n2", Type: "click"},
				{Index: 2, NodeID: "n3", Type: "click"},
				{Index: 3, NodeID: "n4", Type: "click"},
				{Index: 4, NodeID: "sf", Type: "SUBFLOW"},
				{Index: 5, NodeID: "n5", Type: "click"},
			},
			// 6 steps * 15s + 30s base = 120s > 90s min
			wantMin: 120 * time.Second,
			wantMax: 120 * time.Second,
		},
		{
			name: "subflow type with whitespace - ' subflow '",
			instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "n1", Type: "click"},
				{Index: 1, NodeID: "n2", Type: "click"},
				{Index: 2, NodeID: "n3", Type: "click"},
				{Index: 3, NodeID: "n4", Type: "click"},
				{Index: 4, NodeID: "sf", Type: " subflow "},
				{Index: 5, NodeID: "n5", Type: "click"},
			},
			// 6 steps * 15s + 30s base = 120s > 90s min
			wantMin: 120 * time.Second,
			wantMax: 120 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := contracts.ExecutionPlan{
				Instructions: tt.instructions,
			}
			got := computeDynamicTimeout(plan)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("computeDynamicTimeout() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestExecutionTimeoutWithExplicitValue verifies explicit timeout in metadata takes precedence.
func TestExecutionTimeoutWithExplicitValue(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		want     time.Duration
	}{
		{
			name:     "int value",
			metadata: map[string]any{"executionTimeoutMs": 5000},
			want:     5 * time.Second,
		},
		{
			name:     "int64 value",
			metadata: map[string]any{"executionTimeoutMs": int64(10000)},
			want:     10 * time.Second,
		},
		{
			name:     "float64 value",
			metadata: map[string]any{"executionTimeoutMs": float64(15000)},
			want:     15 * time.Second,
		},
		{
			name:     "zero value falls back to dynamic",
			metadata: map[string]any{"executionTimeoutMs": 0},
			want:     90 * time.Second, // minimum for 0-1 step workflow
		},
		{
			name:     "negative value falls back to dynamic",
			metadata: map[string]any{"executionTimeoutMs": -1000},
			want:     90 * time.Second,
		},
		{
			name:     "nil metadata falls back to dynamic",
			metadata: nil,
			want:     90 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := contracts.ExecutionPlan{
				Metadata: tt.metadata,
				Instructions: []contracts.CompiledInstruction{
					{Index: 0, NodeID: "n1", Type: "click"},
				},
			}
			got := executionTimeout(plan)
			if got != tt.want {
				t.Errorf("executionTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}
