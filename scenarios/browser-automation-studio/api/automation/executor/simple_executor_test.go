package executor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/recorder"
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
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
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
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
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

// Ensures capability gaps fail fast before execution.
func TestSimpleExecutorFailsOnCapabilityGap(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	// Engine that does not support uploads/downloads.
	factory := &fakeEngineFactory{
		session: &fakeSession{outcome: contracts.StepOutcome{Success: true}},
		caps: &contracts.EngineCapabilities{
			SchemaVersion:         contracts.EventEnvelopeSchemaVersion,
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
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
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

	session := &fakeSession{
		outcome: contracts.StepOutcome{Success: true},
		delay:   50 * time.Millisecond,
	}

	rec := &memoryRecorder{}
	req := Request{
		Plan: contracts.ExecutionPlan{
			ExecutionID: execID,
			WorkflowID:  workflowID,
			Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "slow", Type: "navigate", Params: map[string]any{"timeoutMs": float64(5)}},
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
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("timeout should abort quickly, took %v", time.Since(start))
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
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
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
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
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
			SchemaVersion:  contracts.StepOutcomeSchemaVersion,
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
}

func (f *fakeEngine) Name() string { return "fake" }

func (f *fakeEngine) Capabilities(ctx context.Context) (contracts.EngineCapabilities, error) {
	if f.caps != nil {
		return *f.caps, nil
	}
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.EventEnvelopeSchemaVersion,
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
}

func (f *fakeSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
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
